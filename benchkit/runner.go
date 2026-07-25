package benchkit

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

const (
	benchStreamType id.StreamType = "Bench"
	benchEventType  event.Type    = "bench.event"
)

// runner executes a benchmark in phases against a single backend.
type runner struct {
	config      Config
	factory     Factory
	gen         *Generator
	codec       codec.Codec
	codecName   string
	bundle      *stack.Bundle
	aggIDs      []id.StreamID
	refs        []id.StreamRef
	concurrency int
	result      Result
	sampler     *resourceSampler
	startCPU    uint64
}

func newRunner(config Config, factory Factory) *runner {
	if config.PayloadSize <= 0 && len(config.PayloadSizes) == 0 {
		config.PayloadSize = 256
	}

	if config.Seed == 0 {
		config.Seed = 1
	}

	if config.Backend == "" {
		config.Backend = "unknown"
	}

	c := config.Codec
	if c == nil {
		c = codec.JSONCodec{}
	}

	var gen *Generator
	if len(config.PayloadSizes) > 0 {
		gen = NewMixedGenerator(config.Seed, config.PayloadSizes, c)
	} else {
		gen = NewGenerator(config.Seed, config.PayloadSize, c)
	}

	return &runner{
		config:      config,
		factory:     factory,
		gen:         gen,
		codec:       c,
		codecName:   codecName(c),
		concurrency: config.Concurrency,
	}
}

func codecName(c codec.Codec) string {
	switch c.(type) {
	case codec.JSONCodec:
		return "json"
	case codec.CBORCodec:
		return "cbor"
	case codec.CBORCompactCodec:
		return "cbor-compact"
	default:
		return "custom"
	}
}

func (r *runner) run(ctx context.Context) (*Result, error) {
	if err := r.setup(ctx); err != nil {
		return nil, err
	}

	defer r.teardown()

	r.startCPU = cpuTime()

	if r.config.Warmup > 0 {
		warmupEvents, err := r.warmup(ctx)
		if err != nil {
			return nil, errorfamily.WrapTransient(err, "benchkit.warmup",
				ErrWarmupFailed.Error())
		}

		r.result.WarmupEvents = warmupEvents
	}

	r.sampler = newResourceSampler()
	r.sampler.start()

	startTime := time.Now()

	runCtx := ctx

	if r.config.Duration > 0 {
		var cancel context.CancelFunc

		runCtx, cancel = context.WithTimeout(ctx, r.config.Duration)
		defer cancel()
	}

	if !r.config.ReplayOnly {
		if err := r.writePhase(runCtx); err != nil {
			return nil, errorfamily.WrapTransient(err, "benchkit.write_phase",
				"write phase")
		}
	}

	if !r.config.SkipReads {
		if err := r.readPhase(runCtx); err != nil {
			return nil, errorfamily.WrapTransient(err, "benchkit.read_phase",
				"read phase")
		}
	}

	if !r.config.SkipReadModels {
		if err := r.readModelPhase(runCtx); err != nil {
			return nil, errorfamily.WrapTransient(err, "benchkit.read_model_phase",
				"read model phase")
		}
	}

	if !r.config.SkipProjections {
		if err := r.projectionPhase(runCtx); err != nil {
			return nil, errorfamily.WrapTransient(err, "benchkit.projection_phase",
				"projection phase")
		}
	}

	r.durabilityPhase()

	if !r.config.ReplayOnly && !r.config.SkipRawSink {
		if err := r.rawSinkPhase(runCtx); err != nil {
			return nil, errorfamily.WrapTransient(err, "benchkit.raw_sink_phase",
				"raw sink phase")
		}
	}

	if r.config.Recovery {
		if err := r.recoveryPhase(ctx); err != nil {
			return nil, errorfamily.WrapTransient(err, "benchkit.recovery_phase",
				"recovery phase")
		}
	}

	peakMem, baselineMem := r.sampler.stopAndSnapshot()

	r.result.Duration = time.Since(startTime)
	r.finalizeResult(peakMem, baselineMem)

	return &r.result, nil
}

func (r *runner) setup(ctx context.Context) error {
	bundle, err := r.factory()
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "benchkit.factory",
			ErrFactoryFailed.Error())
	}

	if bundle == nil {
		return ErrNilBundle
	}

	if r.config.ReplayOnly {
		if bundle.EventSource == nil {
			return ErrIncompleteBundle
		}

		if bundle.Journal == nil && bundle.SeekableJournal == nil {
			return ErrIncompleteBundle
		}
	} else if bundle.EventSink == nil || bundle.EventSource == nil {
		return ErrIncompleteBundle
	}

	r.bundle = bundle

	profile := r.config.Profile

	if r.config.ReplayOnly {
		if err := r.discoverStreams(ctx, profile.Streams); err != nil {
			return errorfamily.WrapInfrastructure(err, "benchkit.replay_discovery",
				"replay stream discovery")
		}
	} else {
		r.aggIDs = make([]id.StreamID, profile.Streams)
		r.refs = make([]id.StreamRef, profile.Streams)

		for i := range profile.Streams {
			aggID := id.NewStreamID()
			r.aggIDs[i] = aggID
			r.refs[i] = id.NewStreamRef(benchStreamType, aggID)
		}
	}

	if r.concurrency <= 0 {
		r.concurrency = profile.Concurrency
	}

	if r.concurrency <= 0 {
		r.concurrency = 1
	}

	r.result.Backend = r.config.Backend
	r.result.Profile = profile.Name
	r.result.Timestamp = time.Now()
	r.result.SchemaVersion = SchemaVersion
	r.result.Environment = Environment{
		GoVersion:  runtime.Version(),
		NumCPU:     runtime.NumCPU(),
		GOMAXPROCS: runtime.GOMAXPROCS(0),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}
	r.result.Workers = r.concurrency
	r.result.Streams = profile.Streams
	r.result.EventsPerStream = profile.EventsPerStream
	r.result.PayloadBytes = r.gen.MeanSize()

	if dist := r.gen.SizeDistribution(); len(dist) > 1 {
		r.result.PayloadSizes = dist
	}

	r.result.Codec = r.codecName

	return nil
}

func (r *runner) teardown() {
	if r.bundle != nil {
		_ = r.bundle.Close()
	}
}

// discoverStreams reads the journal to find existing streams and populates
// r.aggIDs and r.refs. It caps the number of discovered streams at maxStreams.
// Sets r.result.TotalEvents to the total events found in the journal.
//
// When SeekableJournal is available, streams are discovered via batched reads
// (1000 events per batch) to avoid loading the entire journal into memory.
// When only Journal is available, ReadAll is used (loads all events — OOM risk
// for very large stores).
func (r *runner) discoverStreams(ctx context.Context, maxStreams int) error {
	const discoveryBatchSize = 1000

	seen := make(map[string]struct{})
	r.aggIDs = make([]id.StreamID, 0, maxStreams)
	r.refs = make([]id.StreamRef, 0, maxStreams)

	totalEvents := 0

	processBatch := func(events []event.Event) {
		totalEvents += len(events)

		for _, evt := range events {
			streamID := evt.StreamID()
			key := streamID.String()

			if _, ok := seen[key]; ok {
				continue
			}

			seen[key] = struct{}{}

			if len(r.aggIDs) < maxStreams {
				r.aggIDs = append(r.aggIDs, streamID)
				r.refs = append(r.refs, id.NewStreamRef(evt.StreamType(), streamID))
			}
		}
	}

	if r.bundle.SeekableJournal != nil {
		var afterID id.EventID

		for {
			if ctx.Err() != nil {
				break
			}

			batch, err := r.bundle.SeekableJournal.ReadFrom(ctx, afterID, discoveryBatchSize)
			if err != nil {
				return fmt.Errorf("batched journal read for stream discovery: %w", err)
			}

			if len(batch) == 0 {
				break
			}

			processBatch(batch)
			afterID = batch[len(batch)-1].ID()

			if len(batch) < discoveryBatchSize {
				break // last page
			}
		}
	} else if r.bundle.Journal != nil {
		events, err := r.bundle.Journal.ReadAll(ctx)
		if err != nil {
			return fmt.Errorf("read journal for stream discovery: %w", err)
		}

		processBatch(events)
	} else {
		return ErrNilBundle
	}

	r.result.TotalEvents = totalEvents
	r.result.Streams = len(r.aggIDs)

	return nil
}

func (r *runner) finalizeResult(peakMem uint64, baseline memSnapshot) {
	r.result.Memory = ResourceStats{
		Before: baseline.heapAlloc,
		After:  peakMem,
	}

	if peakMem > baseline.heapAlloc {
		r.result.Memory.Delta = peakMem - baseline.heapAlloc
	}

	endCPU := cpuTime()

	r.result.CPU = ResourceStats{
		Before: r.startCPU,
		After:  endCPU,
	}

	if endCPU > r.startCPU {
		r.result.CPU.Delta = endCPU - r.startCPU
	}

	r.result.Disk.EventBytes = int64(r.result.TotalEvents) * int64(r.gen.MeanSize())

	if r.result.Disk.DatabaseBytes > 0 {
		r.result.Disk.OverheadBytes = r.result.Disk.DatabaseBytes - r.result.Disk.EventBytes
		r.result.Disk.OverheadPct = float64(r.result.Disk.OverheadBytes) /
			float64(r.result.Disk.DatabaseBytes) * 100
	}
}

// runConcurrent runs op for each index in [0, total) using at most
// concurrency goroutines. Returns the first error encountered.
func runConcurrent(
	ctx context.Context,
	total, concurrency int,
	op func(ctx context.Context, idx int) error,
) error {
	if concurrency <= 0 {
		concurrency = 1
	}

	if concurrency > total {
		concurrency = total
	}

	work := make(chan int)
	errCh := make(chan error, 1)

	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg      sync.WaitGroup
		errOnce sync.Once
	)

	for range concurrency {
		wg.Go(func() {
			for idx := range work {
				if err := op(cancelCtx, idx); err != nil {
					errOnce.Do(func() { errCh <- err })
					cancel()

					return
				}
			}
		})
	}

	go func() {
		defer close(work)

		for i := range total {
			select {
			case work <- i:
			case <-cancelCtx.Done():
				return
			}
		}
	}()

	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

// warmup runs a few write+load cycles on a throwaway stream to warm
// caches, JIT compilation, and connection pools. It uses a separate Bundle
// so warmup events never pollute the measurement store's journal.
// Returns the number of events written during warmup.
func (r *runner) warmup(ctx context.Context) (int, error) {
	warmupBundle, err := r.factory()
	if err != nil {
		return 0, errorfamily.WrapInfrastructure(err, "benchkit.warmup_factory",
			"warmup factory")
	}

	defer func() { _ = warmupBundle.Close() }()

	if warmupBundle == nil || warmupBundle.EventSink == nil || warmupBundle.EventSource == nil {
		return 0, ErrIncompleteBundle
	}

	aggID := id.NewStreamID()
	ref := id.NewStreamRef(benchStreamType, aggID)

	var version event.Version

	for i := range r.config.Warmup {
		evt, err := event.New(
			benchEventType, aggID, benchStreamType,
			version.Add(uint(i+1)), r.gen.Payload(),
			event.WithCodec(r.codec),
		)
		if err != nil {
			return i, err
		}

		if err := warmupBundle.EventSink.Save(ctx, ref, []event.Event{evt}, version); err != nil {
			return i, err
		}

		version = version.Add(1)
	}

	_, err = warmupBundle.EventSource.Load(ctx, ref)

	return r.config.Warmup, err
}
