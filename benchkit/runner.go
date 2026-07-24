package benchkit

import (
	"context"
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
	if config.PayloadSize <= 0 {
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

	return &runner{
		config:      config,
		factory:     factory,
		gen:         NewGenerator(config.Seed, config.PayloadSize, c),
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

	if err := r.writePhase(runCtx); err != nil {
		return nil, errorfamily.WrapTransient(err, "benchkit.write_phase",
			"write phase")
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

	if bundle.EventSink == nil || bundle.EventSource == nil {
		return ErrIncompleteBundle
	}

	r.bundle = bundle

	profile := r.config.Profile
	r.aggIDs = make([]id.StreamID, profile.Streams)
	r.refs = make([]id.StreamRef, profile.Streams)

	for i := range profile.Streams {
		aggID := id.NewStreamID()
		r.aggIDs[i] = aggID
		r.refs[i] = id.NewStreamRef(benchStreamType, aggID)
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
	r.result.Streams = profile.Streams
	r.result.EventsPerStream = profile.EventsPerStream
	r.result.PayloadBytes = r.config.PayloadSize
	r.result.Codec = r.codecName

	return nil
}

func (r *runner) teardown() {
	if r.bundle != nil {
		_ = r.bundle.Close()
	}
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

	r.result.Disk.EventBytes = int64(r.result.TotalEvents) * int64(r.config.PayloadSize)

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

// warmup runs a few write+load cycles on a throwaway aggregate to warm
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
