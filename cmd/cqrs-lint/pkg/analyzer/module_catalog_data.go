package analyzer

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
			Description: "Cost-based planner: SQL pushdown (FilterOnField/SortOnField), 10 ADTs, layout planning, multi-engine (Memory/SQLite/DuckDB/Pebble/Postgres)",
			Suggestion:  "Cost-based storage planner with FilterOnField/SortOnField SQL pushdown, layout planning, and streaming reads",
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

		// ── Infrastructure ─────────────────────────────────────────────
		{
			Key: "middleware", DisplayName: "Middleware", Category: CategoryObservability,
			ImportHints: []string{"go-cqrs-lite/middleware"},
			Description: "Cross-cutting middleware (tracing, metrics, retry, recovery, validation)",
			Suggestion:  "Middleware for tracing, metrics, retry, and recovery",
		},
		{
			Key: "storage", DisplayName: "SQL Storage", Category: CategoryPersistence,
			ImportHints: []string{"go-cqrs-lite/storage"},
			Description: "SQL backend facade, event/command/query stores, relational projections",
			Suggestion:  "SQL backend for custom store wiring without a stack preset",
		},
		{
			Key: "stack/memory", DisplayName: "Memory Stack", Category: CategoryPersistence,
			ImportHints: []string{"go-cqrs-lite/stack/memory"},
			Description: "In-memory stack preset for testing and development",
			Suggestion:  "In-memory stack preset for fast tests and local development",
		},
		{
			Key: "scenario", DisplayName: "Scenario Testing", Category: CategoryReliability,
			ImportHints: []string{"go-cqrs-lite/scenario"},
			Description: "Fluent BDD test DSL (Given/When/Then) for deciders and projections",
			Suggestion:  "BDD scenario tests for decider and projection behavior",
		},
	}
}
