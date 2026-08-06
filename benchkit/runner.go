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
	config           Config
	factory          Factory
	gen              *Generator
	codec            codec.Codec
	codecName        string
	bundle           *stack.Bundle
	aggIDs           []id.StreamID
	refs             []id.StreamRef
	concurrency      int
	result           Result
	sampler          *resourceSampler
	startCPU         uint64
	baselineMemStats runtime.MemStats
	progress         *progressReporter
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

	r.progress = newProgressReporter(
		r.config.ProgressWriter,
		r.config.ProgressInterval,
		r.config.Backend,
		r.countActivePhases(),
	)
	r.progress.start()
	defer r.progress.stop()

	r.startCPU = cpuTime()
	runtime.ReadMemStats(&r.baselineMemStats)

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
	steps := r.phaseSteps()

	phaseNum := 0

	for _, s := range steps {
		if s.skip {
			continue
		}

		phaseNum++

		r.progress.beginPhase(phaseNum, s.msg)

		phaseStart := time.Now()
		err := s.phase(runCtx)
		r.progress.endPhase(s.msg, time.Since(phaseStart))

		if err != nil {
			return errorfamily.WrapTransient(err, s.code, s.msg)
		}
	}

	r.durabilityPhase()

	if r.config.Recovery {
		if err := r.recoveryPhase(parentCtx); err != nil {
			return errorfamily.WrapTransient(err, "benchkit.recovery_phase", "recovery phase")
		}
	}

	return nil
}

type phaseStep struct {
	skip  bool
	code  string
	msg   string
	phase func(context.Context) error
}

// phaseSteps returns the ordered list of benchmark phases with their skip
// flags resolved from the current config. Extracted from runPhases so
// countActivePhases can share the same definitions.
func (r *runner) phaseSteps() []phaseStep {
	return []phaseStep{
		{r.config.ReplayOnly, "benchkit.write_phase", "write phase", r.writePhase},
		{r.config.SkipReads, "benchkit.read_phase", "read phase", r.readPhase},
		{
			r.config.SkipReadModels,
			"benchkit.read_model_phase",
			"read model phase",
			r.readModelPhase,
		},
		{
			r.config.SkipProjections,
			"benchkit.projection_phase",
			"projection phase",
			r.projectionPhase,
		},
		{
			r.config.SkipMixed || r.config.ReplayOnly, "benchkit.mixed_workload",
			"mixed workload phase", r.mixedWorkloadPhase,
		},
		{
			r.config.ReplayOnly || r.config.SkipJourney,
			"benchkit.journey_phase",
			"journey phase",
			r.journeyPhase,
		},
		{r.config.SkipQuery, "benchkit.query_phase", "query phase", r.queryPhase},
		{
			r.config.ReplayOnly || r.config.SkipSnapshot,
			"benchkit.snapshot_phase",
			"snapshot phase",
			r.snapshotPhase,
		},
		{
			r.config.SkipMetaEngine,
			"benchkit.metaengine_phase",
			"metaengine phase",
			r.metaEnginePhase,
		},
		{
			r.config.ReplayOnly || r.config.SkipRawSink,
			"benchkit.raw_sink_phase",
			"raw sink phase",
			r.rawSinkPhase,
		},
	}
}

func (r *runner) countActivePhases() int {
	count := 0

	for _, s := range r.phaseSteps() {
		if !s.skip {
			count++
		}
	}

	return count
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
		GoVersion:     runtime.Version(),
		NumCPU:        runtime.NumCPU(),
		GOMAXPROCS:    runtime.GOMAXPROCS(0),
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		CPUModel:      detectCPUModel(),
		TotalRAMBytes: detectTotalRAM(),
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
		//cqrs-lint:ignore(C015,C023) best-effort cleanup in teardown
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
