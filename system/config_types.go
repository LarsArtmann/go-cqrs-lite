package system

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
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
	//
	// This is a PayloadDecoder — it does NOT have access to the event's
	// StreamID. For Map ADT queries keyed by entity ID, use
	// ProjectionTypeDecoder or ProjectionEventDecoder instead.
	ProjectionDecoder func(eventType string, payload []byte) (any, error)

	// ProjectionTypeDecoder is the recommended way to decode events for
	// projection folds. It wraps each event's payload in EventWithID,
	// giving fold handlers access to the stream ID (entity key) needed
	// for Map ADT queries. Build it with projectionadapter.NewTypeDecoder(
	// projectionadapter.Register(...), ...).
	//
	// If set, takes precedence over ProjectionEventDecoder and ProjectionDecoder.
	ProjectionTypeDecoder *projectionadapter.TypeDecoder

	// ProjectionEventDecoder provides full event context (StreamID, metadata,
	// version) to fold handlers. Use this when you need a custom decoder that
	// doesn't fit the TypeDecoder registration pattern.
	//
	// If set, takes precedence over ProjectionDecoder. Ignored if
	// ProjectionTypeDecoder is also set.
	ProjectionEventDecoder projectionadapter.EventDecoder

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
// Intentional duplicate: see stack/durability.go. Values MUST match.
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
