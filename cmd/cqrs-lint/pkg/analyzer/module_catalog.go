package analyzer

import "sort"

// ModuleCategory groups adoptable modules for display grouping and
// recommendation priority ordering.
type ModuleCategory string

const (
	CategoryCore          ModuleCategory = "Core"
	CategoryPersistence   ModuleCategory = "Persistence"
	CategoryObservability ModuleCategory = "Observability"
	CategorySecurity      ModuleCategory = "Security"
	CategoryReliability   ModuleCategory = "Reliability"
	CategorySchema        ModuleCategory = "Schema"
	CategoryProjections   ModuleCategory = "Projections"
	CategoryMessaging     ModuleCategory = "Messaging"
	CategoryWorkflow      ModuleCategory = "Workflow"
	CategoryDocumentation ModuleCategory = "Documentation"
	CategoryOptimization  ModuleCategory = "Optimization"
)

// categoryPriority orders categories for recommendation sorting. Lower number
// = higher priority. Missing categories default to 99 (lowest priority).
var categoryPriority = map[ModuleCategory]int{
	CategorySecurity:      1,
	CategoryReliability:   2,
	CategoryObservability: 3,
	CategoryPersistence:   4,
	CategorySchema:        5,
	CategoryProjections:   6,
	CategoryWorkflow:      7,
	CategoryMessaging:     8,
	CategoryOptimization:  9,
	CategoryDocumentation: 10,
	CategoryCore:          99,
}

// ModuleKey uniquely identifies a catalog entry (e.g. "scheduling", "stack/sqlite").
type ModuleKey string

// ModuleEntry describes one adoptable go-cqrs-lite module in the catalog.
type ModuleEntry struct {
	Key         ModuleKey
	DisplayName string
	Category    ModuleCategory
	// ImportHints are substrings matched against consumer import paths.
	// A package import path containing any hint marks this module as used.
	ImportHints []string
	// Description is a one-line summary of what the module provides.
	Description string
	// Suggestion is the "Consider:" text shown when the module is missing
	// and relevant for the detected profile.
	Suggestion string
	// Profiles restricts relevance to specific presets. Empty = all profiles.
	// Non-empty = only relevant when the project's effective preset matches.
	Profiles []ConfigPreset
	// Core marks foundational modules (event, command, etc.) that are
	// always used. Core modules are excluded from the scorecard denominator
	// and numerator — they are infrastructure, not adoption decisions.
	Core bool
}

// Catalog is the canonical universe of adoptable go-cqrs-lite modules.
type Catalog struct {
	entries []ModuleEntry
}

// All returns all catalog entries in canonical order.
func (c Catalog) All() []ModuleEntry {
	return c.entries
}

// Scored returns only non-core entries (the ones counted in the scorecard).
func (c Catalog) Scored() []ModuleEntry {
	result := make([]ModuleEntry, 0, len(c.entries))
	for _, e := range c.entries {
		if !e.Core {
			result = append(result, e)
		}
	}
	return result
}

// Core returns only core entries.
func (c Catalog) Core() []ModuleEntry {
	result := make([]ModuleEntry, 0, 6)
	for _, e := range c.entries {
		if e.Core {
			result = append(result, e)
		}
	}
	return result
}

// Get returns the entry for the given key, or false.
func (c Catalog) Get(key ModuleKey) (ModuleEntry, bool) {
	for _, e := range c.entries {
		if e.Key == key {
			return e, true
		}
	}
	return ModuleEntry{}, false
}

// Keys returns all entry keys in canonical order.
func (c Catalog) Keys() []ModuleKey {
	keys := make([]ModuleKey, len(c.entries))
	for i, e := range c.entries {
		keys[i] = e.Key
	}
	return keys
}

// RelevantFor returns entries relevant to the given feature profile and
// explicit preset. Core entries are excluded. Profile-restricted entries
// (transport, server infra) are excluded when the profile indicates they
// are irrelevant (e.g. a local CLI without a server).
func (c Catalog) RelevantFor(fp FeatureProfile, preset ConfigPreset) []ModuleEntry {
	result := make([]ModuleEntry, 0, len(c.entries))
	for _, e := range c.entries {
		if e.RelevantForProfile(fp, preset) {
			result = append(result, e)
		}
	}
	return result
}

// ByCategory groups entries by category, returning categories in priority order.
func (c Catalog) ByCategory() map[ModuleCategory][]ModuleEntry {
	result := make(map[ModuleCategory][]ModuleEntry)
	for _, e := range c.entries {
		result[e.Category] = append(result[e.Category], e)
	}
	return result
}

// RelevantForProfile reports whether this module should appear in the
// scorecard denominator for the given feature profile and explicit preset.
//
// Core modules are never scored (they are infrastructure, not adoption decisions).
// Modules with no Profiles restriction are always relevant. Modules with a
// Profiles restriction are relevant only when the project's effective preset
// matches — a local-CLI project is not penalized for missing server-only modules.
func (e ModuleEntry) RelevantForProfile(fp FeatureProfile, preset ConfigPreset) bool {
	if e.Core {
		return false
	}
	if len(e.Profiles) == 0 {
		return true
	}
	// Check explicit preset match.
	for _, p := range e.Profiles {
		if p == preset {
			return true
		}
	}
	// Derive effective preset from FeatureProfile signals.
	// A production server (HasServer && !ServerLocal) implies the production preset.
	if fp.HasServer && !fp.ServerLocal {
		for _, p := range e.Profiles {
			if p == PresetProduction {
				return true
			}
		}
	}
	return false
}

// CategoryPriority returns the recommendation priority for this entry's category.
// Lower number = higher priority (Security missing > Documentation missing).
func (e ModuleEntry) CategoryPriority() int {
	if p, ok := categoryPriority[e.Category]; ok {
		return p
	}
	return 99
}

// DefaultCatalog is the canonical universe of adoptable go-cqrs-lite modules.
// It contains 6 core (infrastructure) entries and 28 scored (adoptable) entries.
//
//nolint:gochecknoglobals // read-only catalog data
var DefaultCatalog = Catalog{entries: buildDefaultCatalog()}

func buildDefaultCatalog() []ModuleEntry {
	return []ModuleEntry{
		// ── Core (always used, not scored) ──────────────────────────────
		{
			Key: "event", DisplayName: "Event", Category: CategoryCore,
			ImportHints: []string{"go-cqrs-lite/event"},
			Description: "Event sourcing primitives (Store, Bus, Event, ImmutableEvent)",
			Core:        true,
		},
		{
			Key: "command", DisplayName: "Command", Category: CategoryCore,
			ImportHints: []string{"go-cqrs-lite/command"},
			Description: "Command dispatch, handlers, middleware, bus",
			Core:        true,
		},
		{
			Key: "query", DisplayName: "Query", Category: CategoryCore,
			ImportHints: []string{"go-cqrs-lite/query"},
			Description: "Query dispatch, handlers, pagination",
			Core:        true,
		},
		{
			Key: "decider", DisplayName: "Decider", Category: CategoryCore,
			ImportHints: []string{"go-cqrs-lite/decider"},
			Description: "Pure-function decider pattern (Decide, Fold, Repository)",
			Core:        true,
		},
		{
			Key: "id", DisplayName: "Branded IDs", Category: CategoryCore,
			ImportHints: []string{"go-cqrs-lite/id"},
			Description: "Type-safe branded IDs (id.Of[T], StreamID, EventID)",
			Core:        true,
		},
		{
			Key: "metadata", DisplayName: "Metadata", Category: CategoryCore,
			ImportHints: []string{"go-cqrs-lite/metadata"},
			Description: "Shared metadata types (Tracing, CustomData)",
			Core:        true,
		},

		// ── Persistence ────────────────────────────────────────────────
		{
			Key: "stack/sqlite", DisplayName: "SQLite Stack", Category: CategoryPersistence,
			ImportHints: []string{"go-cqrs-lite/stack/sqlite"},
			Description: "SQLite stack preset (event store + read models + snapshots)",
			Suggestion:  "SQLite backend for local/dev or single-node production",
		},
		{
			Key: "stack/postgres", DisplayName: "Postgres Stack", Category: CategoryPersistence,
			ImportHints: []string{"go-cqrs-lite/stack/postgres"},
			Description: "Postgres stack preset for production event sourcing",
			Suggestion:  "Postgres backend for production multi-instance deployments",
			Profiles:    []ConfigPreset{PresetProduction},
		},
		{
			Key: "stack/mysql", DisplayName: "MySQL Stack", Category: CategoryPersistence,
			ImportHints: []string{"go-cqrs-lite/stack/mysql"},
			Description: "MySQL stack preset",
			Suggestion:  "MySQL backend for MySQL-centric production deployments",
			Profiles:    []ConfigPreset{PresetProduction},
		},
		{
			Key: "stack/pebble", DisplayName: "Pebble Stack", Category: CategoryPersistence,
			ImportHints: []string{"go-cqrs-lite/stack/pebble"},
			Description: "Pebble LSM stack preset for high-throughput workloads",
			Suggestion:  "Pebble LSM backend for high-throughput write workloads",
		},
		{
			Key: "stack/turso", DisplayName: "Turso Stack", Category: CategoryPersistence,
			ImportHints: []string{"go-cqrs-lite/stack/turso"},
			Description: "Turso distributed SQLite stack preset",
			Suggestion:  "Turso for distributed SQLite with edge replication",
			Profiles:    []ConfigPreset{PresetProduction},
		},
		{
			Key: "stack/duckdb", DisplayName: "DuckDB Stack", Category: CategoryPersistence,
			ImportHints: []string{"go-cqrs-lite/stack/duckdb"},
			Description: "DuckDB columnar OLAP stack preset",
			Suggestion:  "DuckDB for columnar analytics and OLAP workloads (CGo required)",
		},

		// ── Observability ──────────────────────────────────────────────
		{
			Key: "otel", DisplayName: "OpenTelemetry", Category: CategoryObservability,
			ImportHints: []string{"go-cqrs-lite/otel"},
			Description: "OpenTelemetry helpers (Tracer, Meter, Spans, Attributes)",
			Suggestion:  "Distributed tracing for production observability",
		},
		{
			Key: "prometheus", DisplayName: "Prometheus", Category: CategoryObservability,
			ImportHints: []string{"go-cqrs-lite/prometheus"},
			Description: "OTel-to-Prometheus metrics bridge with /metrics endpoint",
			Suggestion:  "Prometheus metrics endpoint for production monitoring",
			Profiles:    []ConfigPreset{PresetProduction},
		},
		{
			Key: "flightrecorder", DisplayName: "Flight Recorder", Category: CategoryObservability,
			ImportHints: []string{"go-cqrs-lite/flightrecorder"},
			Description: "Go 1.25 runtime/trace flight recorder for slow/error capture",
			Suggestion:  "Capture execution traces on slow commands or errors",
		},

		// ── Security ───────────────────────────────────────────────────
		{
			Key: "signing", DisplayName: "Event Signing", Category: CategorySecurity,
			ImportHints: []string{"go-cqrs-lite/signing"},
			Description: "Event signing/verification (HMAC-SHA256, Ed25519, multisig)",
			Suggestion:  "Tamper-proof event streams with HMAC or Ed25519 signing",
		},
		{
			Key: "encryption", DisplayName: "Event Encryption", Category: CategorySecurity,
			ImportHints: []string{"go-cqrs-lite/encryption"},
			Description: "Event payload encryption (XChaCha20-Poly1305, AES-256-GCM)",
			Suggestion:  "Encrypt sensitive event payloads at rest",
		},

		// ── Reliability ────────────────────────────────────────────────
		{
			Key: "idempotency", DisplayName: "Idempotency", Category: CategoryReliability,
			ImportHints: []string{"go-cqrs-lite/idempotency"},
			Description: "At-least-once delivery dedup (MemoryStore, SQLStore)",
			Suggestion:  "Dedup at-least-once delivery for command/event handlers",
		},
		{
			Key: "dedup", DisplayName: "Dedup Ring", Category: CategoryReliability,
			ImportHints: []string{"go-cqrs-lite/dedup"},
			Description: "Bounded O(1) ring buffer for stream-boundary dedup",
			Suggestion:  "Bounded ring dedup at stream boundaries",
		},
		{
			Key: "retry", DisplayName: "Retry", Category: CategoryReliability,
			ImportHints: []string{"go-cqrs-lite/retry"},
			Description: "Exponential backoff retry with configurable strategies",
			Suggestion:  "Exponential backoff for transient error recovery",
		},

		// ── Schema ─────────────────────────────────────────────────────
		{
			Key: "schema", DisplayName: "Schema Evolution", Category: CategorySchema,
			ImportHints: []string{"go-cqrs-lite/schema"},
			Description: "Upcasters, VersionedStore, schema evolution support",
			Suggestion:  "Schema evolution with upcasters for long-lived event streams",
		},

		// ── Projections ────────────────────────────────────────────────
		{
			Key: "kv", DisplayName: "KV Store", Category: CategoryProjections,
			ImportHints: []string{"go-cqrs-lite/kv"},
			Description: "Typed KV store, Cache, ViewStore for read models",
			Suggestion:  "Typed KV store for read-model projections",
		},
		{
			Key: "projectionhost", DisplayName: "Projection Host", Category: CategoryProjections,
			ImportHints: []string{"go-cqrs-lite/projectionhost"},
			Description: "Managed projection lifecycle (crash-restart, DLQ, backoff)",
			Suggestion:  "Managed projection lifecycle with crash-restart and dead-letter queue",
		},
		{
			Key: "graph", DisplayName: "Graph Projections", Category: CategoryProjections,
			ImportHints: []string{"go-cqrs-lite/graph"},
			Description: "Graph projections for traversal-heavy read models",
			Suggestion:  "Graph projections for variable-depth traversal queries",
		},
		{
			Key: "listing", DisplayName: "Stream Listing", Category: CategoryProjections,
			ImportHints: []string{"go-cqrs-lite/listing"},
			Description: "Stream listing, tombstone detection, status middleware",
			Suggestion:  "Stream listing and tombstone detection",
		},
		{
			Key: "metaengine", DisplayName: "Metaengine", Category: CategoryProjections,
			ImportHints: []string{"go-cqrs-lite/metaengine"},
			Description: "Cost-based storage planner with universal ADT support",
			Suggestion:  "Cost-based storage planner for query-optimized projections",
		},

		// ── Messaging ──────────────────────────────────────────────────
		{
			Key: "watermill", DisplayName: "Watermill", Category: CategoryMessaging,
			ImportHints: []string{"go-cqrs-lite/watermill"},
			Description: "Distributed event/command bus (NATS, Redis, Kafka backends)",
			Suggestion:  "Distributed event/command bus for multi-process deployments",
			Profiles:    []ConfigPreset{PresetProduction},
		},
		{
			Key: "transport/http", DisplayName: "HTTP Transport", Category: CategoryMessaging,
			ImportHints: []string{"go-cqrs-lite/transport/http"},
			Description: "SSE event delivery over HTTP",
			Suggestion:  "SSE event delivery for real-time HTTP clients",
			Profiles:    []ConfigPreset{PresetProduction},
		},
		{
			Key: "transport/grpc", DisplayName: "gRPC Transport", Category: CategoryMessaging,
			ImportHints: []string{"go-cqrs-lite/transport/grpc"},
			Description: "gRPC transport for remote command/query dispatch",
			Suggestion:  "Remote command/query dispatch over gRPC",
			Profiles:    []ConfigPreset{PresetProduction},
		},
		{
			Key: "deriver", DisplayName: "Deriver", Category: CategoryMessaging,
			ImportHints: []string{"go-cqrs-lite/deriver"},
			Description: "Event-to-command derivation (saga pattern)",
			Suggestion:  "Event-to-command derivation for saga orchestration",
		},

		// ── Workflow ───────────────────────────────────────────────────
		{
			Key: "scheduling", DisplayName: "Scheduling", Category: CategoryWorkflow,
			ImportHints: []string{"go-cqrs-lite/scheduling"},
			Description: "Durable deadline timers (cancel order after 30 min)",
			Suggestion:  "Durable deadline timers for time-based business rules",
		},
		{
			Key: "snapshot", DisplayName: "Snapshot", Category: CategoryWorkflow,
			ImportHints: []string{"go-cqrs-lite/snapshot"},
			Description: "Snapshot strategy for hot streams (EveryNEvents, ReadPressure)",
			Suggestion:  "Snapshot strategy to speed up hot-stream loads",
		},

		// ── Documentation ──────────────────────────────────────────────
		{
			Key: "catalog", DisplayName: "Catalog", Category: CategoryDocumentation,
			ImportHints: []string{"go-cqrs-lite/catalog"},
			Description: "Auto-generate AsyncAPI/D2/OpenAPI event documentation",
			Suggestion:  "Auto-generate event/command API documentation",
		},

		// ── Optimization ───────────────────────────────────────────────
		{
			Key: "codec", DisplayName: "Codec", Category: CategoryOptimization,
			ImportHints: []string{"go-cqrs-lite/codec"},
			Description: "Payload encoding (JSON, CBOR deterministic, Raw)",
			Suggestion:  "CBOR codec for ~35% smaller event payloads",
		},
	}
}

// sortedKeys returns keys sorted alphabetically for deterministic test output.
func (c Catalog) sortedKeys() []ModuleKey {
	keys := c.Keys()
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}
