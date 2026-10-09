package system

import (
	"context"
	"errors"
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
func New(
	ctx context.Context,
	domain DomainConfig,
	deployment DeploymentConfig,
	opts ...Option,
) (*System, error) {
	options := systemOptions{}
	for _, opt := range opts {
		opt(&options)
	}

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
		clock:        options.clock,
		repos:        make(map[string]any),
		deciders:     make(map[string]any),
		cmdDisp:      command.NewDispatcher(),
		qryDisp:      query.NewDispatcher(),
	}

	// Process ProjectionDeclaration values (auto-projection) before planning.
	autoEventDecoder := eventDecoderFn(nil)
	processedProjections := []any(nil)
	var consumedEventTypes map[event.Type][]string

	if len(domain.Projections) > 0 {
		var buildErr error

		processedProjections, autoEventDecoder, consumedEventTypes, buildErr = buildProjections(
			domain.Evolutions, domain.Projections,
		)
		if buildErr != nil {
			return nil, fmt.Errorf("system: build projections: %w", buildErr)
		}
	}

	// Coeffect validation gate (ADR-0136 follow-up): fail composition when a
	// declared event universe exists and something consumes outside it.
	// DomainConfig.Schema types join the universe — a payload-contract
	// declaration is also a journal-universe declaration (2026-10-09 T1).
	eventUniverse := declaredEventTypes(domain)

	if len(eventUniverse) > 0 && !domain.DisableCoeffectValidation {
		if err := validateCoeffectGraph(
			eventUniverse, domain.Evolutions, consumedEventTypes, safetyReport,
		); err != nil {
			return nil, err
		}
	}

	// Create engines from the deployment config via the driver registry,
	// in sorted name order so error selection is deterministic across
	// boots. Durability tiers resolve per engine (see resolveEngineDurability).
	engineDurability, err := resolveEngineDurability(deployment)
	if err != nil {
		return nil, err
	}

	engineCache := make(map[string]metaengine.Engine)

	for _, name := range slices.Sorted(maps.Keys(deployment.Engines)) {
		cfg := deployment.Engines[name]
		eng, err := createEngineFromDriver(ctx, cfg, engineDurability[name])
		if err != nil {
			return nil, sys.fail(fmt.Errorf("system: create engine %q: %w", name, err))
		}

		engineCache[name] = eng
		sys.engines = append(sys.engines, namedEngine{engine: eng, name: name})
	}

	// Dedicated role instances (commands/queries/snapshots) take precedence
	// over the source-of-truth instance for their store.
	dedicated, err := resolveDedicatedRoles(deployment)
	if err != nil {
		return nil, sys.fail(err)
	}

	// Wire source-of-truth and projection instances.
	for _, inst := range deployment.Instances {
		if isSourceOfTruth(inst.Role) {
			if err := wireSourceOfTruth(sys, deployment, inst, engineCache, dedicated); err != nil {
				return nil, sys.fail(err)
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
				return nil, sys.fail(fmt.Errorf(
					"%w: projections instance references undefined engine(s): %s "+
						"(configured engines: %s; registered drivers: %s)",
					ErrUnknownEngine, strings.Join(unresolved, ", "),
					strings.Join(slices.Sorted(maps.Keys(deployment.Engines)), ", "),
					strings.Join(metaengine.RegisteredDrivers(), ", "),
				))
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
					return nil, sys.fail(fmt.Errorf("system: plan projections: %w", err))
				}

				sys.projStore = store
			}
		}
	}

	// Bind dedicated command/query/snapshot instances (after the loop so they
	// take precedence over any source-of-truth wiring above).
	if err := wireDedicatedRoles(sys, deployment, dedicated, engineCache); err != nil {
		return nil, sys.fail(err)
	}

	// Default source of truth when nothing was wired: a Memory engine.
	// ADVISORY, not SCREAM: the fallback is legitimate for tests and demos,
	// but a production deployment that reached it by config typo silently
	// loses every event on restart — surface it on the ScreamReport.
	if sys.eventStore == nil {
		safetyReport.Diagnostics = append(safetyReport.Diagnostics, ScreamDiagnostic{
			Tier: TierAdvisory,
			Rule: "sot.implicit_memory",
			Detail: "no source-of-truth/events instance was configured, so the event store " +
				"falls back to an in-memory engine — all events are lost on restart; " +
				"declare a source-of-truth (or events) instance on a durable engine for production",
		})

		eng := metaengine.NewMemoryEngine()
		sys.engines = append(sys.engines, namedEngine{engine: eng, name: "default"})
		sys.eventStore = NewEventAdapter(eng.(metaengine.StreamLogBackend), "events")
	}

	// Apply the declared schema evolution AFTER every event-store assignment
	// path (instances, cache wrapper, memory fallback) so one decoration
	// covers decider loads and the projection host journal alike.
	if err := applySchemaDeclaration(sys, domain); err != nil {
		return nil, sys.fail(err)
	}

	// Default projection store from Memory when projections are declared.
	if sys.projStore == nil && len(processedProjections) > 0 {
		eng := metaengine.NewMemoryEngine()
		sys.engines = append(sys.engines, namedEngine{engine: eng, name: "projections"})

		store, err := metaengine.Plan([]metaengine.Engine{eng}, processedProjections...)
		if err != nil {
			return nil, sys.fail(fmt.Errorf("system: plan default projections: %w", err))
		}

		sys.projStore = store
	}

	// Wire the event bus and publisher BEFORE the projection host so the
	// host can use the bus as a live subscriber.
	bus, err := createEventBus(deployment)
	if err != nil {
		return nil, sys.fail(err)
	}

	sys.bus = bus

	// Register the bus for lifecycle management (watermill.EventBus implements
	// io.Closer) BEFORE the publisher build so a buildPublisher failure cannot
	// strand the bus's goroutines.
	if closer, ok := bus.(io.Closer); ok {
		sys.closers = append(sys.closers, namedCloser{closer: closer, name: "event-bus"})
	}

	pub, fanouts, err := buildPublisher(deployment, sys.bus)
	if err != nil {
		return nil, sys.fail(err)
	}

	sys.pubBus = pub

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
			return nil, sys.fail(ErrSeekableJournalMissing)
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

		cpStore := resolveCheckpointStore(domain, sys)

		host, err := projectionhost.New(journal, cpStore, hostOpts...)
		if err != nil {
			return nil, sys.fail(fmt.Errorf("system: create projection host: %w", err))
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
			return nil, sys.fail(fmt.Errorf("system: register projection adapter: %w", err))
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
			return nil, sys.fail(fmt.Errorf("system: plan safety check: %w", err))
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

			return nil, sys.fail(fmt.Errorf("%w: %s", ErrUnsafeChange, detail))
		}
	}

	// Wire shutdown dependencies from domain config. Validate edge names
	// against the POPULATED engine set (configured + synthesized "default"/
	// "projections"): unknown names are otherwise silently dropped by the
	// shutdown topological sort (E10), while config-only validation would
	// reject the documented synthetic names (the E10 follow-up gap).
	if err := validateShutdownDependencies(domain.ShutdownDependencies, sys.engines); err != nil {
		return nil, sys.fail(err)
	}

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

	if domain.Timers != nil {
		domain.Timers(sys)
	}

	return sys, nil
}

// fail tears down everything New created so far and returns err with any
// teardown failures joined after it. It is the single exit for New's error
// paths once the first engine exists: returning the bare error there would
// drop the System on the floor and leak engine file handles, locks, and
// background goroutines. Ordering mirrors [System.Close] (projection host,
// then engines, then registered closers); teardown errors never mask the
// construction error.
func (sys *System) fail(err error) error {
	var errs []error
	if err != nil {
		errs = append(errs, err)
	}

	if sys.projHost != nil {
		if stopErr := sys.projHost.Stop(); stopErr != nil {
			errs = append(errs, fmt.Errorf("system: fail-cleanup projection host: %w", stopErr))
		}
	}

	for _, eng := range sys.orderedEngines() {
		if closeErr := eng.Close(); closeErr != nil {
			errs = append(errs, fmt.Errorf("system: fail-cleanup engine: %w", closeErr))
		}
	}

	for _, nc := range sys.closers {
		if closeErr := nc.closer.Close(); closeErr != nil {
			errs = append(errs, fmt.Errorf("system: fail-cleanup %s: %w", nc.name, closeErr))
		}
	}

	return errors.Join(errs...)
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

	s.startTimersLocked(ctx)

	return nil
}

// isSourceOfTruth returns true for instances that hold event/command/query logs.
func isSourceOfTruth(role InstanceRole) bool {
	return role == RoleSourceOfTruth || role == RoleEvents
}
