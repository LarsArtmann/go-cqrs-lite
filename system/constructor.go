package system

import (
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

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
	// WARN+OVERRIDE / ADVISORY findings are kept on the System for
	// post-construction introspection via ScreamReport.
	safetyReport := &ScreamReport{}

	if report, err := CheckSafety(ctx, deployment); err != nil {
		return nil, fmt.Errorf("system: safety check: %w", err)
	} else if report.HasErrors() {
		return nil, fmt.Errorf("%w: %s", ErrUnsafeChange, report.Diagnostics[0].Detail)
	} else {
		safetyReport.Diagnostics = append(
			[]ScreamDiagnostic(nil), report.Diagnostics...,
		)
	}

	sys := &System{
		deployment:   deployment,
		safetyReport: safetyReport,
		repos:        make(map[string]any),
		deciders:     make(map[string]any),
		cmdDisp:      command.NewDispatcher(),
		qryDisp:      query.NewDispatcher(),
	}

	// Process ProjectionDeclaration values (auto-projection) before planning.
	autoEventDecoder := eventDecoderFn(nil)
	processedProjections := []any(nil)

	if len(domain.Projections) > 0 {
		var buildErr error

		processedProjections, autoEventDecoder, buildErr = buildProjections(
			domain.Evolutions, domain.Projections,
		)
		if buildErr != nil {
			return nil, fmt.Errorf("system: build projections: %w", buildErr)
		}
	}

	// Create engines from the deployment config via the driver registry.
	// Iterate in sorted name order so engine creation (and error selection
	// when two engines are both invalid) is deterministic across boots.
	// Durability tiers are resolved per engine first: instances sharing an
	// engine must agree (see resolveEngineDurability).
	engineDurability, err := resolveEngineDurability(deployment)
	if err != nil {
		return nil, err
	}

	engineCache := make(map[string]metaengine.Engine)

	for _, name := range slices.Sorted(maps.Keys(deployment.Engines)) {
		cfg := deployment.Engines[name]
		eng, err := createEngineFromDriver(ctx, cfg, engineDurability[name])
		if err != nil {
			return nil, fmt.Errorf("system: create engine %q: %w", name, err)
		}

		engineCache[name] = eng
		sys.engines = append(sys.engines, namedEngine{engine: eng, name: name})
	}

	// Dedicated role instances (commands/queries/snapshots) take precedence
	// over the source-of-truth instance for their store.
	dedicated, err := resolveDedicatedRoles(deployment)
	if err != nil {
		return nil, err
	}

	// Wire source-of-truth and projection instances.
	for _, inst := range deployment.Instances {
		if isSourceOfTruth(inst.Role) {
			if err := wireSourceOfTruth(sys, deployment, inst, engineCache, dedicated); err != nil {
				return nil, err
			}
		}

		if inst.Role == RoleProjections && sys.projStore == nil {
			engineNames := inst.Engines
			if len(engineNames) == 0 && inst.Engine != "" {
				engineNames = []string{inst.Engine}
			}

			var projEngines []metaengine.Engine
			var unresolved []string

			for _, name := range engineNames {
				if eng, ok := engineCache[name]; ok {
					projEngines = append(projEngines, eng)
				} else {
					unresolved = append(unresolved, name)
				}
			}

			if len(unresolved) > 0 {
				return nil, fmt.Errorf(
					"%w: projections instance references undefined engine(s): %s",
					ErrUnknownEngine, strings.Join(unresolved, ", "),
				)
			}

			if len(projEngines) > 0 && len(processedProjections) > 0 {
				var planOpts []any

				if deployment.Priority != nil {
					planOpts = append(
						planOpts,
						metaengine.WithPriorityConfig(deployment.Priority.toMeta()),
					)
				}

				store, err := metaengine.Plan(
					projEngines,
					append(planOpts, processedProjections...)...)
				if err != nil {
					return nil, fmt.Errorf("system: plan projections: %w", err)
				}

				sys.projStore = store
			}
		}
	}

	// Bind dedicated command/query/snapshot instances (after the loop so they
	// take precedence over any source-of-truth wiring above).
	if err := wireDedicatedRoles(sys, deployment, dedicated, engineCache); err != nil {
		return nil, err
	}

	// If no source-of-truth instance, create a default Memory engine.
	if sys.eventStore == nil {
		eng := metaengine.NewMemoryEngine()
		sys.engines = append(sys.engines, namedEngine{engine: eng, name: "default"})
		sys.eventStore = NewEventAdapter(eng.(metaengine.StreamLogBackend), "events")
	}

	// If no projection store, create one from Memory if projections are declared.
	if sys.projStore == nil && len(processedProjections) > 0 {
		eng := metaengine.NewMemoryEngine()
		sys.engines = append(sys.engines, namedEngine{engine: eng, name: "projections"})

		store, err := metaengine.Plan([]metaengine.Engine{eng}, processedProjections...)
		if err != nil {
			return nil, fmt.Errorf("system: plan default projections: %w", err)
		}

		sys.projStore = store
	}

	// Wire the event bus and publisher BEFORE the projection host so the
	// host can use the bus as a live subscriber.
	bus, err := createEventBus(deployment)
	if err != nil {
		return nil, err
	}

	sys.bus = bus
	pub, fanouts := buildPublisher(deployment, sys.bus)
	sys.pubBus = pub

	// Register the bus for lifecycle management (watermill.EventBus implements io.Closer).
	if closer, ok := bus.(io.Closer); ok {
		sys.closers = append(sys.closers, namedCloser{closer: closer, name: "event-bus"})
	}

	// Register each fan-out bus by its Publish target name so Close() does not
	// leak them and diagnostics show the operator-facing name.
	for _, fb := range fanouts {
		sys.closers = append(
			sys.closers,
			namedCloser{closer: fb.closer, name: "fanout-bus-" + fb.name},
		)
	}

	// Wire projection host if we have projections and an event journal.
	if sys.projStore != nil {
		journal, ok := sys.eventStore.(event.SeekableJournal)
		if !ok {
			return nil, ErrSeekableJournalMissing
		}

		// Auto-wire the system bus as the subscriber for live event delivery
		// after the initial journal drain. Append to consumer-provided options.
		hostOpts := append(
			make([]projectionhost.HostOption, 0, len(domain.ProjectionHostOptions)+1),
			domain.ProjectionHostOptions...,
		)
		if bus, ok := sys.bus.(event.Subscriber); ok {
			hostOpts = append(hostOpts, projectionhost.WithSubscriber(bus))
		}

		// Use the consumer-provided checkpoint store, or fall back to in-memory.
		cpStore := domain.CheckpointStore
		if cpStore == nil {
			cpStore = &memoryCheckpointStore{}
		}

		host, err := projectionhost.New(journal, cpStore, hostOpts...)
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
		case autoEventDecoder != nil:
			adapter = projectionadapter.New("projections", sys.projStore, nil,
				projectionadapter.WithEventDecoder(
					projectionadapter.EventDecoder(autoEventDecoder),
				),
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

	// Plan-drift detection: if a manifest path is configured, compare the
	// current projection plan against the pinned manifest from the previous
	// startup. SCREAM-tier violations block startup; WARN-tier diagnostics
	// are surfaced to the caller.
	if deployment.ManifestPath != "" {
		currentPlan := sys.ProjectionPlan()

		planReport, err := CheckPlanSafety(ctx, currentPlan, deployment.ManifestPath)
		if err != nil {
			return nil, fmt.Errorf("system: plan safety check: %w", err)
		}

		// Surface plan-drift WARN/ADVISORY findings alongside config findings.
		safetyReport.Diagnostics = append(
			safetyReport.Diagnostics, planReport.Diagnostics...,
		)

		if planReport.HasErrors() {
			detail := "unknown"
			if len(planReport.Diagnostics) > 0 {
				detail = planReport.Diagnostics[0].Detail
			}

			return nil, fmt.Errorf("%w: %s", ErrUnsafeChange, detail)
		}
	}

	// Wire shutdown dependencies from domain config.
	for _, dep := range domain.ShutdownDependencies {
		sys.shutdownDeps = append(sys.shutdownDeps, shutdownEdge{
			before: dep.Before,
			after:  dep.After,
		})
	}

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
