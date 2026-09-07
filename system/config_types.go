package system

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
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

	// Projections are sealed projection/query declarations that the System
	// will auto-wire into projection instances. Pass values created by
	// [Lookup], [QuerySet], [Count], or [RawQuery].
	//
	// The sealed [ProjectionDeclaration] interface replaces the previous
	// []any slice: stray strings, nils, or typos are now compile-time errors.
	Projections []ProjectionDeclaration

	// Evolutions declare how result types materialize from events (the fold).
	// Projections without their own samples inherit folds from the matching
	// Evolution by result type.
	Evolutions []EvolutionSpec

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

	// ProjectionHostOptions configures the projection host (batch size, DLQ,
	// logger, restart policy, etc.). These are applied when the System creates
	// the projection host. If nil, defaults are used.
	ProjectionHostOptions []projectionhost.HostOption

	// CheckpointStore provides persistent checkpoint storage for the projection
	// host. If nil, an in-memory store is used (checkpoints are lost on
	// restart, forcing full projection replays). Set this to a persistent
	// store (e.g., SQLCheckpointStore) for production deployments where
	// projections must resume from their last position after a restart.
	CheckpointStore event.CheckpointStore

	// ShutdownDependencies declares ordering constraints for System.Close().
	// Each edge says "Before must close before After". Resource names are
	// engine names from DeploymentConfig.Engines. Resources not in any edge
	// close in creation order (projection host first, then engines). Cycles
	// fall back to creation order.
	//
	// Example: ensure the event store outlives the projection engine:
	//   ShutdownDependencies: []system.ShutdownDependency{
	//       {Before: "projections", After: "primary"},
	//   }
	ShutdownDependencies []ShutdownDependency
}

// ShutdownDependency declares that Before must close before After during
// System.Close(). Resource names are engine names from DeploymentConfig.Engines.
// The projection host always closes first and cannot participate in edges.
type ShutdownDependency struct {
	Before string // close this resource first
	After  string // close this resource after
}

// DeploymentConfig carries operator-facing concerns: engines, buses,
// instances, durability tiers, and cache configuration. This is data —
// loadable from YAML or env via koanf.
type DeploymentConfig struct {
	// Engines maps named engine declarations. Each engine has a driver name
	// (e.g., "sqlite", "memory") and a DSN. Instances reference engines by name.
	Engines map[string]EngineConfig `koanf:"engines"`

	// Buses maps named bus declarations. Each bus has a driver name
	// (e.g., "gochannel", "nats") and configuration.
	Buses map[string]BusConfig `koanf:"buses"`

	// Instances describes how collections are grouped into metaengine.Store
	// instances. Each instance has a role, an engine reference, and optional
	// bus/cache configuration.
	Instances []InstanceConfig `koanf:"instances"`

	// AcknowledgeWarnings lists scream-store warnings the operator has ACKed.
	// Format: "rule:target" (e.g., "durability-downgrade:events").
	AcknowledgeWarnings []string `koanf:"acknowledge_warnings"`

	// ManifestPath is the file path for the pinned projection-plan manifest.
	// When set, the scream store loads the previous SerializablePlan from this
	// file on startup, diffs it against the current plan, and blocks (SCREAM)
	// or warns on unsafe changes. After a successful startup, the current plan
	// is saved to this path for the next restart.
	// Leave empty to disable plan-drift detection.
	ManifestPath string `koanf:"manifest_path"`

	// Priority is the operator's layout-planning objective for the whole
	// deployment (ADR-0124): one of "WriteSpeed", "ReadSpeed",
	// "StorageSpace", or "Balanced" (default). Resolution order:
	// query-level (perQuery) → engine-level (perEngine/engine priority) →
	// global (here) → Balanced. Leave empty for Balanced everywhere.
	Priority *PriorityConfig `koanf:"priority"`
}

// PriorityConfig is the operator-facing YAML shape for layout-planning
// priorities (ADR-0124). It maps to metaengine.PriorityConfig with PerEngine
// keyed by engine name and PerQuery keyed by query name.
type PriorityConfig struct {
	// Global applies to every query unless a more specific level is set.
	Global metaengine.Priority `koanf:"global"`

	// PerEngine overrides the global priority for queries routed to a named
	// engine (key = engine name from Engines).
	PerEngine map[string]metaengine.Priority `koanf:"perEngine"`

	// PerQuery overrides all other levels for a named query (key = query name).
	PerQuery map[string]metaengine.Priority `koanf:"perQuery"`
}

// toMeta returns the metaengine equivalent of this config.
func (pc PriorityConfig) toMeta() *metaengine.PriorityConfig {
	return &metaengine.PriorityConfig{
		Global:    pc.Global,
		PerEngine: pc.PerEngine,
		PerQuery:  pc.PerQuery,
	}
}

// EngineConfig declares a named storage engine.
type EngineConfig struct {
	Driver  string   `koanf:"driver"`  // "sqlite", "memory", "pebble", "duckdb", "postgres"
	DSN     string   `koanf:"dsn"`     // connection string (empty for memory)
	Pragmas []string `koanf:"pragmas"` // SQLite pragmas (e.g., "wal", "foreign_keys")

	// Priority is the operator's layout-planning objective for this engine
	// (ADR-0124): one of "WriteSpeed", "ReadSpeed", "StorageSpace", or
	// "Balanced" (default). It flows into DriverConfig.Priority and weights
	// the cost model's scoring for queries routed to this engine. Leave
	// empty for the global default.
	Priority metaengine.Priority `koanf:"priority"`

	// MaterializedViews declares operator-owned aggregate accelerations
	// (Turso/libSQL incremental view maintenance). Each entry names a
	// collection and an aggregate shape (fn, column, optional groupBy); the
	// engine derives and creates the view at construction and serves matching
	// unfiltered aggregates from it. This is a deployment-time concern —
	// developers never declare views. Unsupported engines fail construction
	// loudly.
	MaterializedViews []MaterializedViewConfig `koanf:"materialized_views"`
}

// MaterializedViewConfig is the operator-facing YAML shape for one
// materialized view acceleration (see metaengine.MaterializedViewSpec).
type MaterializedViewConfig struct {
	// Collection is the collection whose aggregate this view accelerates
	// (e.g. "orders").
	Collection string `koanf:"collection"`

	// Fn is the aggregate function: "COUNT", "SUM", "MIN", "MAX", or "AVG"
	// (case-insensitive).
	Fn string `koanf:"fn"`

	// Column is the JSON field name aggregated over (e.g. "amount"). Empty
	// (and required to be empty) for COUNT.
	Column string `koanf:"column"`

	// GroupBy is the optional JSON field name to group by (e.g. "customer").
	// Empty creates a single-row scalar view (fastest reads).
	GroupBy string `koanf:"group_by"`
}

// materializedViewSpecs maps the operator config to validated
// metaengine.MaterializedViewSpec values.
func (c EngineConfig) materializedViewSpecs() ([]metaengine.MaterializedViewSpec, error) {
	if len(c.MaterializedViews) == 0 {
		return nil, nil
	}

	specs := make([]metaengine.MaterializedViewSpec, 0, len(c.MaterializedViews))

	for i, mv := range c.MaterializedViews {
		fn, err := parseMatViewFn(mv.Fn)
		if err != nil {
			return nil, fmt.Errorf(
				"system: engine %q materialized_views[%d]: %w",
				labelOr(c.Driver, i+1),
				i,
				err,
			)
		}

		spec := metaengine.MaterializedViewSpec{
			Collection: mv.Collection,
			Fn:         fn,
			Column:     mv.Column,
			GroupBy:    mv.GroupBy,
		}

		if err := spec.Validate(); err != nil {
			return nil, fmt.Errorf(
				"system: engine %q materialized_views[%d]: %w",
				labelOr(c.Driver, i+1),
				i,
				err,
			)
		}

		specs = append(specs, spec)
	}

	return specs, nil
}

// parseMatViewFn resolves the operator-supplied aggregate function name
// (case-insensitive) to a metaengine.AggregateFn.
func parseMatViewFn(s string) (metaengine.AggregateFn, error) {
	switch metaengine.AggregateFn(strings.ToUpper(strings.TrimSpace(s))) {
	case metaengine.MatViewCount:
		return metaengine.MatViewCount, nil
	case metaengine.MatViewSum:
		return metaengine.MatViewSum, nil
	case metaengine.MatViewMin:
		return metaengine.MatViewMin, nil
	case metaengine.MatViewMax:
		return metaengine.MatViewMax, nil
	case metaengine.MatViewAvg:
		return metaengine.MatViewAvg, nil
	case "":
		return "", fmt.Errorf("fn is required (COUNT, SUM, MIN, MAX, or AVG)")
	default:
		return "", fmt.Errorf("unknown fn %q (want COUNT, SUM, MIN, MAX, or AVG)", s)
	}
}

// labelOr returns label when non-empty, else a positional fallback for error
// messages.
func labelOr(label string, pos int) string {
	if label != "" {
		return label
	}

	return fmt.Sprintf("#%d", pos)
}

// BusConfig declares a named message bus.
type BusConfig struct {
	Driver string `koanf:"driver"` // "gochannel", "nats", "redis"
	URL    string `koanf:"url"`    // broker URL (empty for gochannel)

	// Mode is introspection-only: parsed and surfaced in Introspection(),
	// but publish is always synchronous on the gochannel bus regardless of
	// this value. DECISION (2026-08-18): the field will be REMOVED at v5
	// rather than gaining sync/async publish semantics — setting it changes
	// nothing and never will.
	Mode string `koanf:"mode"`
}

// CacheConfig declares an optional read-through cache tier for an instance.
type CacheConfig struct {
	// Engine is reserved and NOT read: the cache tier wraps the instance's
	// event store directly, so a separate cache engine is never opened.
	// The field is parsed for backward compatibility; removal at v5.
	Engine string `koanf:"engine"`

	// Capacity is the max entry count (otter W-TinyLFU handles eviction).
	Capacity int `koanf:"capacity"`
}

// InstanceConfig describes one metaengine.Store instance.
type InstanceConfig struct {
	// Role classifies this instance: RoleEvents, RoleCommands, RoleQueries,
	// RoleSnapshots, RoleProjections, or RoleSourceOfTruth (combined).
	Role InstanceRole `koanf:"role"`

	// Collections lists the collection names this instance serves. It is
	// introspection-only: surfaced in Introspection() and topology output,
	// but it does not gate routing — the role determines which collections
	// are wired. For RoleEvents, this is typically ["events"]. For
	// RoleSourceOfTruth, it may be ["events", "commands", "queries",
	// "snapshots", "checkpoints"].
	Collections []string `koanf:"collections"`

	// Engine is the named engine for a single-engine instance.
	// Mutually exclusive with Engines.
	Engine string `koanf:"engine"`

	// Engines lists named engines for a mixed-pool instance (projection layer).
	// The metaengine planner routes freely within the pool.
	// Mutually exclusive with Engine.
	Engines []string `koanf:"engines"`

	// Durability is the persistence tier for this instance: strict, normal,
	// or relaxed. Empty means unspecified — engine defaults. All instances
	// sharing an engine must agree on the tier; the resolved tier is applied
	// at engine construction (e.g. SQLite PRAGMA synchronous). Engines
	// without per-tier behavior fail construction on an explicit tier.
	Durability DurabilityTier `koanf:"durability"`

	// Publish lists bus names that events from this instance are published to.
	// Events fan-out to all listed buses (D9: multi-bus support).
	Publish []string `koanf:"publish"`

	// Subscribe is reserved and NOT read: the projection host consumes from
	// the deployment's bus topology directly, so per-instance subscriptions
	// are not (yet) wired. Parsed for backward compatibility; per-instance
	// subscription may arrive at v5.
	Subscribe []string `koanf:"subscribe"`

	// Cache configures an optional read-through cache tier.
	Cache *CacheConfig `koanf:"cache"`
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
// art-dupl:accept intentional cross-module duplicate — separate go.mod, values MUST match
type DurabilityTier string

const (
	// DurabilityStrict fsyncs every commit. Safe against power loss.
	DurabilityStrict DurabilityTier = "strict"

	// DurabilityNormal is safe against app/OS crash. WAL checkpoint window
	// may be lost on power loss.
	DurabilityNormal DurabilityTier = "normal"

	// DurabilityRelaxed may lose data on crash. Use only for rebuildable
	// projections or cache tiers.
	DurabilityRelaxed DurabilityTier = "relaxed"
)
