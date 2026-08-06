package system

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// New constructs a System from a DomainConfig (consumer) and a DeploymentConfig
// (operator). The two-config split (D11) enforces that the consumer never
// touches infrastructure and the operator never writes domain code.
//
// After construction, the consumer registers commands, queries, and deciders
// via the generic top-level functions (RegisterDecider, RegisterCommand,
// RegisterQuery). Then call Start to begin projection processing.
func New(ctx context.Context, domain DomainConfig, deployment DeploymentConfig) (*System, error) {
	// Safety check: refuse to start if SCREAM-tier violations exist.
	if report, err := CheckSafety(ctx, deployment); err != nil {
		return nil, fmt.Errorf("system: safety check: %w", err)
	} else if report.HasErrors() {
		return nil, fmt.Errorf("%w: %s", ErrUnsafeChange, report.Diagnostics[0].Detail)
	}

	sys := &System{
		deployment: deployment,
		repos:      make(map[string]any),
		deciders:   make(map[string]any),
		cmdDisp:    command.NewDispatcher(),
		qryDisp:    query.NewDispatcher(),
		bus:        newSimpleBus(),
	}

	// Create engines from the deployment config via the driver registry.
	engineCache := make(map[string]metaengine.Engine)

	for name, cfg := range deployment.Engines {
		eng, err := createEngineFromDriver(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("system: create engine %q: %w", name, err)
		}

		engineCache[name] = eng
		sys.engines = append(sys.engines, eng)
	}

	// Find the source-of-truth instance and wire the adapters.
	for _, inst := range deployment.Instances {
		if isSourceOfTruth(inst.Role) {
			engineName := inst.Engine
			if engineName == "" && len(inst.Engines) > 0 {
				engineName = inst.Engines[0]
			}

			eng, ok := engineCache[engineName]
			if !ok {
				return nil, fmt.Errorf(
					"%w: instance %q references engine %q",
					ErrUnknownEngine, inst.Role, engineName,
				)
			}

			backend, ok := eng.(metaengine.StreamLogBackend)
			if !ok {
				return nil, fmt.Errorf(
					"%w: engine %q",
					ErrNotStreamLogBackend, engineName,
				)
			}

			// Auto-detect serialization: Memory stores pointers directly; all
			// other engines need JSON envelope serialization.
			var adapterOpts []EventAdapterOption
			if engCfg, hasCfg := deployment.Engines[engineName]; hasCfg &&
				engCfg.Driver != "memory" {
				adapterOpts = append(adapterOpts, WithSerialization())
			}

			sys.eventStore = NewEventAdapter(backend, "events", adapterOpts...)

			// Wire snapshot store if the engine implements SnapshotBackend (D12).
			if snapBackend, ok := eng.(metaengine.SnapshotBackend); ok {
				sys.snapStore = NewSnapshotAdapter(snapBackend, "snapshots")
			}

			// Wire command and query audit stores from the same backend.
			if inst.Role == RoleSourceOfTruth || inst.Role == RoleCommands {
				sys.cmdStore = NewCommandAdapter(backend, "commands")
			}

			if inst.Role == RoleSourceOfTruth || inst.Role == RoleQueries {
				sys.queryStore = NewQueryAdapter(backend, "queries")
			}

			// Wire cache tier if configured.
			if inst.Cache != nil && inst.Cache.Capacity > 0 {
				cached, err := NewCachedEventStore(sys.eventStore, inst.Cache.Capacity)
				if err != nil {
					return nil, fmt.Errorf("system: create cache: %w", err)
				}

				sys.eventStore = cached
			}
		}

		if inst.Role == RoleProjections && sys.projStore == nil {
			engineNames := inst.Engines
			if len(engineNames) == 0 && inst.Engine != "" {
				engineNames = []string{inst.Engine}
			}

			var projEngines []metaengine.Engine

			for _, name := range engineNames {
				if eng, ok := engineCache[name]; ok {
					projEngines = append(projEngines, eng)
				}
			}

			if len(projEngines) > 0 && len(domain.Projections) > 0 {
				args := make([]any, len(domain.Projections))
				copy(args, domain.Projections)

				store, err := metaengine.Plan(projEngines, args...)
				if err != nil {
					return nil, fmt.Errorf("system: plan projections: %w", err)
				}

				sys.projStore = store
			}
		}
	}

	// If no source-of-truth instance, create a default Memory engine.
	if sys.eventStore == nil {
		eng := metaengine.NewMemoryEngine()
		sys.engines = append(sys.engines, eng)
		sys.eventStore = NewEventAdapter(eng.(metaengine.StreamLogBackend), "events")
	}

	// If no projection store, create one from Memory if projections are declared.
	if sys.projStore == nil && len(domain.Projections) > 0 {
		eng := metaengine.NewMemoryEngine()
		sys.engines = append(sys.engines, eng)

		args := make([]any, len(domain.Projections))
		copy(args, domain.Projections)

		store, err := metaengine.Plan([]metaengine.Engine{eng}, args...)
		if err != nil {
			return nil, fmt.Errorf("system: plan default projections: %w", err)
		}

		sys.projStore = store
	}

	// Wire projection host if we have projections and an event journal.
	if sys.projStore != nil {
		journal, ok := sys.eventStore.(event.SeekableJournal)
		if !ok {
			return nil, ErrSeekableJournalMissing
		}

		host, err := projectionhost.New(journal, &memoryCheckpointStore{})
		if err != nil {
			return nil, fmt.Errorf("system: create projection host: %w", err)
		}

		// Register a projection adapter that feeds events into the metaengine Store.
		// Decoder priority: TypeDecoder > EventDecoder > PayloadDecoder > generic JSON.
		var adapter *projectionadapter.Adapter

		switch {
		case domain.ProjectionTypeDecoder != nil:
			adapter = projectionadapter.NewWithDecoder(
				"projections", sys.projStore, domain.ProjectionTypeDecoder,
			)
		case domain.ProjectionEventDecoder != nil:
			adapter = projectionadapter.New("projections", sys.projStore, nil,
				projectionadapter.WithEventDecoder(domain.ProjectionEventDecoder),
			)
		default:
			var decoder projectionadapter.PayloadDecoder

			if domain.ProjectionDecoder != nil {
				decoder = projectionadapter.PayloadDecoder(domain.ProjectionDecoder)
			}

			adapter = projectionadapter.New("projections", sys.projStore, decoder)
		}

		if err := host.Register(adapter); err != nil {
			return nil, fmt.Errorf("system: register projection adapter: %w", err)
		}

		sys.projHost = host
	}

	// Wire MultiBus if the source-of-truth instance has multiple Publish targets (D9).
	sys.bus = buildEventBus(deployment)
	sys.pubBus = buildPublisher(deployment, sys.bus)

	// Register domain middleware.
	sys.UseCommandMiddleware(domain.Middleware...)

	// Let the consumer register commands, queries, and deciders.
	if domain.Commands != nil {
		domain.Commands(sys)
	}

	if domain.Queries != nil {
		domain.Queries(sys)
	}

	return sys, nil
}

// Start begins projection processing (if configured).
func (s *System) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return ErrAlreadyStarted
	}

	s.started = true

	if s.projHost != nil {
		if err := s.projHost.Start(ctx); err != nil {
			return fmt.Errorf("system: start projection host: %w", err)
		}
	}

	return nil
}

// isSourceOfTruth returns true for instances that hold event/command/query logs.
func isSourceOfTruth(role InstanceRole) bool {
	return role == RoleSourceOfTruth || role == RoleEvents
}
