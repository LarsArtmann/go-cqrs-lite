package benchkit

import (
	"context"
	"runtime"
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

	if err := r.runPhases(runCtx, ctx); err != nil {
		return nil, err
	}

	peakMem, baselineMem := r.sampler.stopAndSnapshot()

	r.result.Duration = time.Since(startTime)
	r.finalizeResult(peakMem, baselineMem)

	return &r.result, nil
}

// runPhases executes all benchmark phases in order, returning the first error.
// runCtx is the (possibly deadline-limited) context for measured phases.
// parentCtx is the unbounded context for the recovery phase.
func (r *runner) runPhases(runCtx, parentCtx context.Context) error {
	if !r.config.ReplayOnly {
		if err := r.writePhase(runCtx); err != nil {
			return errorfamily.WrapTransient(err, "benchkit.write_phase", "write phase")
		}
	}

	if !r.config.SkipReads {
		if err := r.readPhase(runCtx); err != nil {
			return errorfamily.WrapTransient(err, "benchkit.read_phase", "read phase")
		}
	}

	if !r.config.SkipReadModels {
		if err := r.readModelPhase(runCtx); err != nil {
			return errorfamily.WrapTransient(err, "benchkit.read_model_phase", "read model phase")
		}
	}

	if !r.config.SkipProjections {
		if err := r.projectionPhase(runCtx); err != nil {
			return errorfamily.WrapTransient(err, "benchkit.projection_phase", "projection phase")
		}
	}

	if !r.config.ReplayOnly && !r.config.SkipJourney {
		if err := r.journeyPhase(runCtx); err != nil {
			return errorfamily.WrapTransient(err, "benchkit.journey_phase", "journey phase")
		}
	}

	if !r.config.SkipQuery {
		if err := r.queryPhase(runCtx); err != nil {
			return errorfamily.WrapTransient(err, "benchkit.query_phase", "query phase")
		}
	}

	if !r.config.ReplayOnly && !r.config.SkipSnapshot {
		if err := r.snapshotPhase(runCtx); err != nil {
			return errorfamily.WrapTransient(err, "benchkit.snapshot_phase", "snapshot phase")
		}
	}

	r.durabilityPhase()

	if !r.config.ReplayOnly && !r.config.SkipRawSink {
		if err := r.rawSinkPhase(runCtx); err != nil {
			return errorfamily.WrapTransient(err, "benchkit.raw_sink_phase", "raw sink phase")
		}
	}

	if r.config.Recovery {
		if err := r.recoveryPhase(parentCtx); err != nil {
			return errorfamily.WrapTransient(err, "benchkit.recovery_phase", "recovery phase")
		}
	}

	return nil
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
		//cqrs-lint:ignore(C023) best-effort cleanup in teardown
		//cqrs-lint:ignore(C015) library code or intentional pattern
		_ = r.bundle.Close()
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
