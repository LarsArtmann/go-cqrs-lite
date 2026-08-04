package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
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
					"system: instance %q references unknown engine %q",
					inst.Role,
					engineName,
				)
			}

			backend, ok := eng.(metaengine.StreamLogBackend)
			if !ok {
				return nil, fmt.Errorf(
					"system: engine %q does not implement StreamLogBackend",
					engineName,
				)
			}

			// Auto-detect serialization: Memory stores pointers directly; all
			// other engines need JSON envelope serialization.
			var adapterOpts []EventAdapterOption
			if engCfg, hasCfg := deployment.Engines[engineName]; hasCfg && engCfg.Driver != "memory" {
				adapterOpts = append(adapterOpts, WithSerialization())
			}

			sys.eventStore = NewEventAdapter(backend, "events", adapterOpts...)

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
			return nil, errors.New("system: event store does not implement SeekableJournal")
		}

		host, err := projectionhost.New(journal, &memoryCheckpointStore{})
		if err != nil {
			return nil, fmt.Errorf("system: create projection host: %w", err)
		}

		// Register a projection adapter that feeds events into the metaengine Store.
		var decoder projectionadapter.PayloadDecoder
		if domain.ProjectionDecoder != nil {
			decoder = projectionadapter.PayloadDecoder(domain.ProjectionDecoder)
		}

		adapter := projectionadapter.New("projections", sys.projStore, decoder)
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
		return errors.New("system: already started")
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

// ─── D10: Generic registration functions ───

// RegisterDecider registers a decider for a stream type. The System creates
// a [decider.Repository] for it, backed by the EventAdapter.
//
// Multiple commands targeting the same stream type share the same repository
// automatically — just call RegisterCommand with the same streamType in the
// system.Execute call.
func RegisterDecider[State any](sys *System, streamType string, d decider.Decider[State]) error {
	if sys.eventStore == nil {
		return errors.New("system: cannot register decider: no event store")
	}

	repo, err := decider.NewRepository[State](sys.eventStore, sys.pubBus, d)
	if err != nil {
		return fmt.Errorf("system: create repository for %q: %w", streamType, err)
	}

	sys.mu.Lock()
	sys.repos[streamType] = repo
	sys.deciders[streamType] = d
	sys.mu.Unlock()

	return nil
}

// RegisterCommand registers a typed command handler that returns an [Op].
// The System executes the Op via the decider repository registered for the
// Op's stream type (D10: declarative routing).
//
// The command type Cmd must implement [command.Command] (Type, StreamID, ID).
// The State type must match the decider registered for the stream type
// referenced in the handler's [Execute] call.
func RegisterCommand[Cmd command.Command, State any](
	sys *System,
	name command.Type,
	handler func(ctx context.Context, cmd Cmd) Op[State],
) error {
	sys.mu.Lock()
	sys.cmdHandlerCount++
	sys.mu.Unlock()

	return sys.cmdDisp.Register(name, func(ctx context.Context, cmd command.Command) error {
		typed, ok := any(cmd).(Cmd)
		if !ok {
			return fmt.Errorf("system: command type mismatch for %q: got %T", name, cmd)
		}

		op := handler(ctx, typed)

		sys.mu.RLock()
		repoAny, exists := sys.repos[string(op.streamType)]
		sys.mu.RUnlock()

		if !exists {
			return fmt.Errorf("system: no decider registered for stream type %q", op.streamType)
		}

		repo, ok := repoAny.(*decider.Repository[State])
		if !ok {
			return fmt.Errorf("system: decider type mismatch for stream type %q", op.streamType)
		}

		return repo.Execute(ctx, op.streamID, op.streamType, op.decide)
	})
}

// RegisterQuery registers a typed query handler.
func RegisterQuery[Q any, R any](
	sys *System,
	name string,
	handler func(ctx context.Context, q Q) (R, error),
) error {
	return sys.qryDisp.Register(
		query.Type(name),
		func(ctx context.Context, q query.Query) (any, error) {
			typed, ok := q.(Q)
			if !ok {
				return nil, fmt.Errorf("system: query type mismatch for %q: got %T", name, q)
			}

			return handler(ctx, typed)
		},
	)
}

// DispatchQuery dispatches a typed query and returns the result.
func DispatchQuery[Q query.Query, R any](ctx context.Context, sys *System, q Q) (R, error) {
	result, err := sys.qryDisp.Dispatch(ctx, q)
	if err != nil {
		var zero R

		return zero, err
	}

	typed, ok := result.(R)
	if !ok {
		var zero R

		return zero, fmt.Errorf("system: query result type mismatch: got %T", result)
	}

	return typed, nil
}
