// Package system is the deployer-driven composition root for go-cqrs-lite.
//
// System replaces [stack.Bundle] with a model where the consumer provides
// only domain types (DomainConfig) and the operator provides only
// infrastructure decisions (DeploymentConfig). The two are separate types,
// compiler-enforced per decision D11.
//
// The System owns ALL infrastructure wiring: storage instances, bus(es),
// projectionhost, and dispatchers (D6). Internally it uses N
// [metaengine.Store] instances across two layers (source-of-truth +
// projections), connected via the [metaengine.StreamLogBackend] interface.
package system

import (
	"context"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// ─── Op[State] and Execute (D10: declarative command routing) ───

// Op[State] carries the information the System needs to execute a command
// against a decider. It is returned by command handlers registered via
// [RegisterCommand].
type Op[State any] struct {
	streamID   id.StreamID
	streamType id.StreamType
	decide     decider.DecideFunc[State]
}

// Execute creates an [Op] that the System will execute against the decider
// registered for streamType.
func Execute[State any](_ context.Context, streamID id.StreamID, streamType id.StreamType,
	decide decider.DecideFunc[State],
) Op[State] {
	return Op[State]{
		streamID:   streamID,
		streamType: streamType,
		decide:     decide,
	}
}

// StreamID returns the target stream for this Op.
func (o Op[State]) StreamID() id.StreamID { return o.streamID }

// StreamType returns the stream type (entity type) for this Op.
func (o Op[State]) StreamType() id.StreamType { return o.streamType }

// ─── System type ───

// namedEngine pairs a metaengine Engine with its deployment-config name.
// Storing them together eliminates the fragile parallel-slice pattern where
// appending to one without the other causes silent misalignment.
type namedEngine struct {
	engine metaengine.Engine
	name   string
}

// namedCloser pairs an external io.Closer with a name for diagnostics.
type namedCloser struct {
	closer io.Closer
	name   string
}

// engineSlice returns a flat slice of just the engines (without names).
// Used by HealthCheck, Explain, Serialize, and Verify which need []Engine.
func (s *System) engineSlice() []metaengine.Engine {
	result := make([]metaengine.Engine, len(s.engines))
	for i, ne := range s.engines {
		result[i] = ne.engine
	}

	return result
}

// projectionHostLifecycle is the subset of projectionhost.Host methods that
// System calls internally. Defining it as an interface enables test injection
// of mock projection hosts (e.g., to simulate Stop failures). *projectionhost.Host
// satisfies this interface at construction time.
type projectionHostLifecycle interface {
	Start(ctx context.Context) error
	Stop() error
	Status() []projectionhost.WorkerState
	LagPerProjection() map[string]time.Duration
	LagDuration() time.Duration
	Reset(ctx context.Context, name string, opts ...projectionhost.ResetOption) error
}

// System is the composition root that owns ALL infrastructure wiring (D6).
// It is constructed via [New] with a [DomainConfig] and a [DeploymentConfig].
//
// The consumer interacts with the System via:
//   - [System.Command] — register typed command handlers
//   - [System.RegisterDecider] — register a decider for a stream type
//   - [System.Query] — register typed query handlers
//   - [System.UseCommandMiddleware] — inject domain middleware
//   - [System.MetaEngine] — access the projection-layer store
//   - [System.Start] / [System.Stop] — lifecycle
type System struct {
	mu sync.RWMutex

	deployment DeploymentConfig

	// clock is the injected time source (nil = RealClock); see clock.go.
	clock Clock

	// Adapters wrap StreamLogBackend as standard CQRS interfaces.
	eventStore event.Store
	cmdStore   command.Store
	queryStore query.QueryStore

	// snapStore holds the snapshot store (nil if engine lacks SnapshotBackend).
	snapStore snapshot.SnapshotStore

	// Infrastructure owned by the System (D6).
	repos    map[string]any // streamType -> *decider.Repository[State]
	deciders map[string]any // streamType -> decider.Decider[State]

	cmdDisp  *command.Dispatcher
	qryDisp  *query.Dispatcher
	projHost projectionHostLifecycle
	bus      event.Bus
	pubBus   event.Publisher // publisher for decider repository (may be MultiBus)

	// Projection-layer metaengine store.
	projStore *metaengine.Store

	// Engines for lifecycle management. Each entry pairs the engine with its
	// deployment-config name so shutdown ordering can reference engines by name.
	engines []namedEngine

	// ADR-0142 timers: schedulers registered via ManageTimers, started with
	// the System and stopped (cancelled + waited) on Close/GracefulClose
	// (timers.go). timersWG tracks the running Start goroutines so teardown
	// can wait for dispatch to fully finish.
	timers      []TimerScheduler
	timerCancel context.CancelFunc
	timersWG    sync.WaitGroup

	// shutdownDeps declares ordering constraints for Close(). Each edge says
	// "before must close before after". Resources not in any edge keep their
	// creation order. Ported from [stack.Bundle].
	shutdownDeps []shutdownEdge

	// drainers are called by GracefulClose before Close to drain in-flight work.
	drainers []Drainer

	// closers are external resources registered via RegisterCloser. They are
	// closed after engines during Close(), in registration order.
	closers []namedCloser

	// handlerCount tracks registered command handlers for introspection.
	cmdHandlerCount int

	// cmdTypes/qryTypes record registered command/query payload types for
	// introspection ([CommandRegistrations]/[QueryRegistrations]) — the
	// runtime-inventory surface documentation bridges derive from.
	cmdTypes map[command.Type]reflect.Type
	qryTypes map[query.Type]reflect.Type

	// safetyReport holds the construction-time safety diagnostics (SCREAM
	// blocked construction; WARN+OVERRIDE and ADVISORY findings are kept here
	// so operators can inspect them via ScreamReport without re-running the
	// checks against a possibly-mutated deployment config).
	safetyReport *ScreamReport

	started bool
	stopped bool
}

// MetaEngine returns the projection-layer metaengine store, or nil if no
// projection instance is configured. Consumers use this for typed queries
// via [metaengine.ExecuteTyped] or [metaengine.NewReader].
func (s *System) MetaEngine() *metaengine.Store {
	return s.projStore
}

// CommandDispatcher returns the command dispatcher. Consumers use this to
// dispatch commands programmatically (in addition to HTTP handlers).
func (s *System) CommandDispatcher() *command.Dispatcher {
	return s.cmdDisp
}

// QueryDispatcher returns the query dispatcher.
func (s *System) QueryDispatcher() *query.Dispatcher {
	return s.qryDisp
}

// Publisher returns the event publisher used by decider repositories.
// This may be a MultiBus if the deployment configures multiple Publish targets.
func (s *System) Publisher() event.Publisher {
	return s.pubBus
}

// PublisherFor returns the fan-out publisher bound to the given Publish
// target name from the deployment config — the name the operator wrote in
// YAML (e.g. publish: [local, nats] → PublisherFor("nats")).
//
// The bool is false when no fan-out publisher is bound to that name (no
// multi-bus configured, or an unknown target). The local bus is always entry
// 0 of MultiBus.Publishers() and has no name binding.
func (s *System) PublisherFor(target string) (event.Publisher, bool) {
	multi, ok := s.pubBus.(*MultiBus)
	if !ok {
		return nil, false
	}

	return multi.PublisherByName(target)
}

// ProjectionHost returns the projection host, or nil if not configured.
func (s *System) ProjectionHost() *projectionhost.Host {
	if h, ok := s.projHost.(*projectionhost.Host); ok {
		return h
	}

	return nil
}

// Bus returns the event bus for pub/sub. Consumers use this to subscribe to
// events for custom side-effects (notifications, webhooks, etc.).
func (s *System) Bus() event.Bus {
	return s.bus
}

// UseCommandMiddleware adds domain middleware to the command dispatcher.
// This is the consumer's extension point for validation, authorization, etc.
func (s *System) UseCommandMiddleware(mw ...command.Middleware) {
	if s.cmdDisp == nil {
		return
	}

	for _, m := range mw {
		s.cmdDisp.Use(m)
	}
}

// EventStore returns the event store backed by the source-of-truth instance.
func (s *System) EventStore() event.Store {
	return s.eventStore
}

// SnapshotStore returns the snapshot store if the engine implements
// SnapshotBackend, or nil otherwise. Consumers can use this to manage snapshots
// directly (e.g., manual SaveSnapshot after batch imports).
func (s *System) SnapshotStore() snapshot.SnapshotStore {
	return s.snapStore
}

// CommandStore returns the command audit store, or nil when no instance
// provides one (no source-of-truth, no dedicated commands instance).
func (s *System) CommandStore() command.Store {
	return s.cmdStore
}

// QueryStore returns the query audit store, or nil when no instance provides
// one (no source-of-truth, no dedicated queries instance).
func (s *System) QueryStore() query.QueryStore {
	return s.queryStore
}
