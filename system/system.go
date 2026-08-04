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
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// ─── Config types (D11: DomainConfig + DeploymentConfig separation) ───

// DomainConfig carries consumer-facing concerns: deciders, commands, queries,
// projections, and domain middleware. This is Go code — closures and typed
// handlers. It never touches DSNs, engine names, or bus drivers.
type DomainConfig struct {
	// Commands is a function that registers typed command handlers on the
	// System via sys.Command(...) and sys.RegisterDecider(...).
	Commands func(*System)

	// Queries is a function that registers typed query handlers on the
	// System via sys.Query(...).
	Queries func(*System)

	// Projections are metaengine query declarations that the System will
	// auto-wire into projection instances. Pass them as []any because Go
	// generics are invariant — QueryDecl[ConcreteInput, ConcreteResult] is
	// not assignable to QueryDecl[any, any].
	Projections []any

	// ProjectionDecoder decodes event payloads for the projection fold handlers.
	// If nil, events are decoded as generic JSON (map[string]any).
	ProjectionDecoder func(eventType string, payload []byte) (any, error)

	// Middleware is command-level domain middleware (validation, authz, etc.).
	Middleware []command.Middleware
}

// DeploymentConfig carries operator-facing concerns: engines, buses,
// instances, durability tiers, and cache configuration. This is data —
// loadable from YAML or env via koanf.
type DeploymentConfig struct {
	// Engines maps named engine declarations. Each engine has a driver name
	// (e.g., "sqlite", "memory") and a DSN. Instances reference engines by name.
	Engines map[string]EngineConfig

	// Buses maps named bus declarations. Each bus has a driver name
	// (e.g., "gochannel", "nats") and configuration.
	Buses map[string]BusConfig

	// Instances describes how collections are grouped into metaengine.Store
	// instances. Each instance has a role, an engine reference, and optional
	// bus/cache configuration.
	Instances []InstanceConfig

	// AcknowledgeWarnings lists scream-store warnings the operator has ACKed.
	// Format: "rule:target" (e.g., "durability-downgrade:events").
	AcknowledgeWarnings []string
}

// EngineConfig declares a named storage engine.
type EngineConfig struct {
	Driver  string   // "sqlite", "memory", "pebble", "duckdb", "postgres"
	DSN     string   // connection string (empty for memory)
	Pragmas []string // SQLite pragmas (e.g., "wal", "foreign_keys")
}

// BusConfig declares a named message bus.
type BusConfig struct {
	Driver string // "gochannel", "nats", "redis"
	URL    string // broker URL (empty for gochannel)
	Mode   string // "sync" (block on publish) or "async" (fire-and-forget)
}

// CacheConfig declares an optional read-through cache tier for an instance.
type CacheConfig struct {
	Engine   string // named engine to use as cache (e.g., "hot-cache")
	Capacity int    // max entries (otter W-TinyLFU handles eviction)
}

// InstanceConfig describes one metaengine.Store instance.
type InstanceConfig struct {
	// Role classifies this instance: RoleEvents, RoleCommands, RoleQueries,
	// RoleSnapshots, RoleProjections, or RoleSourceOfTruth (combined).
	Role InstanceRole

	// Collections lists the collection names this instance serves.
	// For RoleEvents, this is typically ["events"]. For RoleSourceOfTruth,
	// it may be ["events", "commands", "queries", "snapshots", "checkpoints"].
	Collections []string

	// Engine is the named engine for a single-engine instance.
	// Mutually exclusive with Engines.
	Engine string

	// Engines lists named engines for a mixed-pool instance (projection layer).
	// The metaengine planner routes freely within the pool.
	// Mutually exclusive with Engine.
	Engines []string

	// Durability is the persistence tier for this instance.
	Durability DurabilityTier

	// Publish lists bus names that events from this instance are published to.
	// Events fan-out to all listed buses (D9: multi-bus support).
	Publish []string

	// Subscribe lists bus names that projections on this instance consume from.
	// The projectionhost uses CatchUpSubscriber for each subscribed bus.
	Subscribe []string

	// Cache configures an optional read-through cache tier.
	Cache *CacheConfig
}

// InstanceRole classifies a metaengine.Store instance by its function.
type InstanceRole string

const (
	// RoleSourceOfTruth is a combined instance holding events, commands,
	// queries, snapshots, and checkpoints. The default for minimal deployments.
	RoleSourceOfTruth InstanceRole = "source-of-truth"

	// RoleEvents is a dedicated event log instance.
	RoleEvents InstanceRole = "events"

	// RoleCommands is a dedicated command audit log instance.
	RoleCommands InstanceRole = "commands"

	// RoleQueries is a dedicated query audit log instance.
	RoleQueries InstanceRole = "queries"

	// RoleSnapshots is a dedicated snapshot store instance.
	RoleSnapshots InstanceRole = "snapshots"

	// RoleProjections is a projection-layer instance with a mixed engine pool.
	RoleProjections InstanceRole = "projections"
)

// DurabilityTier controls the persistence guarantees of an instance.
type DurabilityTier string

const (
	// DurabilityStrict fsyncs every commit. Safe against power loss.
	DurabilityStrict DurabilityTier = "strict"

	// DurabilityNormal is safe against app/OS crash. WAL checkpoint window
	// may be lost on power loss. Default for all instances.
	DurabilityNormal DurabilityTier = "normal"

	// DurabilityRelaxed may lose data on crash. Use only for rebuildable
	// projections or cache tiers.
	DurabilityRelaxed DurabilityTier = "relaxed"
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

	// Adapters wrap StreamLogBackend as standard CQRS interfaces.
	eventStore event.Store
	cmdStore   command.Store
	queryStore query.QueryStore

	// Infrastructure owned by the System (D6).
	repos    map[string]any // streamType -> *decider.Repository[State]
	deciders map[string]any // streamType -> decider.Decider[State]

	cmdDisp  *command.Dispatcher
	qryDisp  *query.Dispatcher
	projHost *projectionhost.Host
	bus      event.Bus
	pubBus   event.Publisher // publisher for decider repository (may be MultiBus)

	// Projection-layer metaengine store.
	projStore *metaengine.Store

	// Engines and closers for lifecycle management.
	engines []metaengine.Engine
	closers []func() error

	// handlerCount tracks registered command handlers for introspection.
	cmdHandlerCount int

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

// ProjectionHost returns the projection host, or nil if not configured.
func (s *System) ProjectionHost() *projectionhost.Host {
	return s.projHost
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

// ProjectionPlan returns the serializable plan for the projection layer, or
// nil if no projection store is configured. Useful for pinning plans and
// detecting drift across restarts.
func (s *System) ProjectionPlan() *metaengine.SerializablePlan {
	if s.projStore == nil {
		return nil
	}

	result := s.projStore.Plan()
	if result == nil {
		return nil
	}

	return metaengine.Serialize(result, s.engines)
}

// VerifyProjections checks the consistency of the projection layer plan
// against the configured engines. Returns nil if no projection store exists.
func (s *System) VerifyProjections(ctx context.Context) error {
	if s.projStore == nil {
		return nil
	}

	return s.projStore.Verify(ctx, s.engines)
}

// ProjectionExplain returns a human-readable plan explanation for the
// projection layer, or an empty string if no projection store is configured.
func (s *System) ProjectionExplain() string {
	if s.projStore == nil {
		return ""
	}

	return s.projStore.ExplainPlan()
}

// Close shuts down all owned infrastructure: projection host, engines, stores.
func (s *System) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return nil
	}

	s.stopped = true

	var firstErr error

	if s.projHost != nil {
		if err := s.projHost.Stop(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("system: stop projection host: %w", err)
		}
	}

	for _, closer := range s.closers {
		if err := closer(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	for _, eng := range s.engines {
		_ = eng.Close()
	}

	return firstErr
}
