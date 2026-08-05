# Project: go-cqrs-lite

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import modules (`event`, `command`, `decider`, `storage`, `memory`, `catalog`, etc.) into THEIR projects.
> There is no "main app." Every module is independently importable.
>
> | If you catch yourself thinking…              | STOP — this is a LIBRARY, not an app                                                                                                       |
> | -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
> | "Nothing in this repo uses it, so delete it" | **DELETING EXTERNAL-FACING API IS BREAKING THE PRODUCT.** Consumers live outside this repo. Zero internal consumers is the EXPECTED state. |
> | "Module needs a service that uses it"        | Module needs tests + stable API, not an internal consumer                                                                                  |
> | "example/ should drive real traffic"         | example/ is a usage demo, not a deployment                                                                                                 |
> | "Unused exports are waste"                   | Public API surface IS the product                                                                                                          |
>
> **The quality gate for every module: "Would a consumer trust this enough to import it?"**

A lightweight CQRS **library/SDK** for Go with Event Sourcing support, branded IDs, and auto-documentation generation.

Consumers import what they need and compose their own stack. Not a framework — no opinionated transport, message broker, or SQL driver.

> **AI consumer guide:** [`SKILL.md`](SKILL.md) is the canonical reference for AI agents using this library — module decision matrix, composition recipes, conventions, and anti-patterns. This file (AGENTS.md) is for contributors working **inside** the repo.

## Quick Reference

| Item       | Value                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Language   | Go 1.26.4                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Modules    | `event`, `command`, `query`, `decider`, `deriver`, `id`, `id/idtest`, `idempotency`, `idempotency/kvstore`, `idempotency/sqlstore`, `metadata`, `query/querytest`, `dispatcher`, `schema`, `snapshot`, `catalog`, `middleware`, `integration`, `storage`, `storage/eventstore`, `storage/readmodel`, `storage/memory`, `storage/pebble`, `storage/turso`, `signing`, `encryption`, `otel`, `prometheus`, `watermill`, `transport/http`, `transport/grpc`, `codec`, `kv`, `kv/viewstoretest`, `listing`, `graph`, `projection`, `projectionhost`, `scenario`, `scheduling`, `scheduling/sqlstore`, `testutil`, `dedup`, `retry`, `flightrecorder`, `benchkit`, `cqrs-gen`, `cqrs-lint`, `cqrs-bench`, `api-stability`, `doc-check`, `stack`, `stack/memory`, `stack/sqlite`, `stack/duckdb`, `stack/turso`, `stack/pebble`, `stack/postgres`, `stack/mysql`, `stack/bench`, `example/taskmanager`, `example/getting-started`, `example/readme-quickstart`, `metaengine`, `metaengine/adttest`, `metaengine/pebbleengine`, `metaengine/duckdbengine`, `metaengine/pgengine`, `metaengine/irohengine`, `metaengine/irohengine/loopback`, `metaengine/irohengine/quic`, `metaengine/projectionadapter`, `system`                                                   |
| Build      | `nix run .#build`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| Test       | `nix run .#test` or `go test ./event/... ./command/... ./idempotency/... ./idempotency/kvstore/... ./query/... ./decider/... ./deriver/... ./id/... ./metadata/... ./dispatcher/... ./schema/... ./snapshot/... ./storage/memory/... ./catalog/... ./middleware/... ./integration/... ./signing/... ./encryption/... ./storage/... ./storage/pebble/... ./storage/turso/... ./watermill/... ./transport/http/... ./transport/grpc/... ./codec/... ./kv/... ./listing/... ./projection/... ./projectionhost/... ./scenario/... ./scheduling/... ./scheduling/sqlstore/... ./testutil/... ./dedup/... ./retry/... ./flightrecorder/... ./benchkit/... ./cmd/cqrs-gen/... ./cmd/cqrs-lint/... ./cmd/cqrs-bench/... ./cmd/doc-check/... ./prometheus/... ./otel/... ./stack/... ./stack/memory/... ./stack/sqlite/... ./stack/duckdb/... ./stack/turso/... ./stack/pebble/... ./stack/postgres/... ./stack/mysql/... ./stack/bench/... ./example/taskmanager/... ./example/getting-started/... ./example/readme-quickstart/... ./graph/... ./metaengine/... ./metaengine/pebbleengine/... ./metaengine/duckdbengine/... ./metaengine/pgengine/... ./metaengine/irohengine/... ./metaengine/projectionadapter/... ./system/... ./idempotency/sqlstore/... -count=1` |
| Lint       | `nix run .#lint`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| Format     | `nix fmt`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Dev shell  | `nix develop`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| CI         | GitHub Actions: ci.yml (Nix-based, build/vet/test/lint/race/coverage + GOWORK=off per-module)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Int. PG    | `nix run .#integration-pg` (ephemeral, no Docker) or `nix run .#integration-pg-vm` (QEMU VM)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| Int. MySQL | `nix run .#integration-mysql-nspawn` (nspawn, ~15s, needs root + uid-range) or `nix run .#integration-mysql-vm` (QEMU VM, ~131s, always works)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| Int. All   | `nix run .#integration-all` (all PG+MySQL tests, nspawn preferred with QEMU fallback) or `nix run .#verify-integration` (integration gate only)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |

## Monorepo Structure

Multi-module Go workspace (`go.work`) with 69 `go.mod` files — all wired into `go.work`. Breakdown: 49 library + 9 stack presets + 3 examples + 5 cmd + 1 root workspace + 1 eventtest nested + 1 external. Verify: `find . -name go.mod -not -path './vendor/*' | wc -l`:

```
go-cqrs-lite/
├── event/               # EventSink, EventSource, Store, Journal, SeekableJournal, Bus, ImmutableEvent, NewEvent, Clone
│   └── v4/eventtest/    # FakeStore, FakeBus, FakeSnapshotStore, event factories, test assertions
├── command/             # Dispatcher, Handler, Middleware, Command, BasicCommand, PersistedCommand
│                        # Store: CommandSink, CommandSource, Store, CommandJournal, SeekableCommandJournal
│                        # Bus: Publisher, Subscriber, Bus, PublishMiddleware (command pub/sub)
├── query/               # Dispatcher, Handler, Pagination, PaginatedResult[T], RegisterTyped[Q,R], TypedHandler[Q,R]
│                        # Store: PersistedQuery, QuerySink, QuerySource, QueryStore, QueryJournal, SeekableQueryJournal
│   └── querytest/       # New(tb, queryType) test helper — tb.Fatalf on error, no panics
├── decider/             # Decider[State], Repository[State], Execute, Load (pure-function style)
├── deriver/             # Event→command derivation: Deriver, Then, Filter, Idempotent, AsHandler (ADR-0040)
├── id/                  # Branded IDs: id.Of[T] = cbid.ID[T, ulid.ULID], StreamID, EventID, etc.
│   └── idtest/          # Parse*(tb, s) test helpers — tb.Fatalf on error, no panics
├── metadata/            # Tracing, CustomData[K] (extracted from event/ — shared metadata types for command/query/event)
├── metaengine/          # Cost-based storage planner — THE STRATEGIC FUTURE of this project (possibly a future dedicated project)
│                        # Core: Engine, Store, Plan, 7 ADTs, cost model, SQLite engine. Zero production deps in core.
│                        # ⚠️ CANONICAL DESIGN DOCS (read before working on metaengine):
│                        #   [project-definition](docs/planning/meta-engine-project-definition.md),
│                        #   [design/vision](docs/planning/meta-engine-design.md),
│                        #   [assumptions & query-planning](docs/planning/meta-engine-assumptions-and-query-planning.md)
│                        # SQLite engine: tx-atomic MapUpdate, restart-safe multimap seq, ExecuteTyped reifies map[string]any→struct.
│                        # Caller owns the *sql.DB (engine Close is a no-op).
│                        # ADRs: [0061](docs/adr/0061-metaengine-sqlite-engine.md), [0062](docs/adr/0062-metaengine-dependency-boundary.md), [0063](docs/adr/0063-metaengine-pushdown.md), [0073](docs/adr/0073-metaengine-layout-planning.md), [0093](docs/adr/0093-metaengine-replication-model.md), [0094](docs/adr/0094-metaengine-universal-adt-support.md), [0096](docs/adr/0096-iroh-distributed-engine-bridge-evaluation.md)
│                        # **Query pushdown** (Phase 1): FilterOnField/SortOnField → SQLite json_extract() WHERE/ORDER BY pushdown
│                        # **Layout planning** (Phase 3): LayoutPlan generates DDL from declared query patterns (indexed columns, 10x faster)
│                        # **Streaming reads** (Phase 5): StreamScan(ctx) iter.Seq2 for OOM-safe lazy iteration
│                        # **SSE reconnection**: Watcher.WithReplay(cap) → SSEReplay[V] ring buffer; ServeSSE writes `id: <seq>`,
│                        #   replays via `Last-Event-ID` header with dedup.Ring (WithSSEReplayLimit caps backlog). WatchWithSeq → SeqValue[V].
│                        # **Cursor-encoded prefetch**: WithCursorString(s) parses Cursor.Encode() (base64+JSON); both WithCursor(raw)
│                        #   and WithCursorString(encoded) produce matching PrefetchCache keys (thread-safe via RWMutex).
│                        # TypedReader[V], RawValueReader/RawScanReader. Sentinels: ErrNotFound, ErrAmbiguousKey, ErrUnsupportedADT, ErrLayoutConflict, ErrKeyTypeMismatch (input struct missing declared key type field).
│                        # **Typed watcher convenience**: WatchTyped[V](store, ctx, coll, key) and WatchTypedWithSeq[V] combine NewWatcher + Watch in one call (free functions, Go can't have generic methods).
│                        # **Planner rule pipeline** (ADR-pending): PlanRule interface, RulePipeline, composable rules. 4 extracted: schemaRule, layoutRule, writeAmpRule. Planner.go dissolved from 279→226 lines.
│                        # **Materialize-vs-replay** (ES-specific killer feature): WithWorkloadStats, ReplayCost/MaterializeCost/ShouldMaterialize, materializeRule. Advisory INFO/WARN diagnostics.
│                        # **StorageLayout + cost matrix**: Layout{Row,Columnar,LSM,KV}, (ADT × Layout)→Complexity, EngineProfile.Layouts, RuleTrace, SerializablePlan (JSON serialize/diff/pin).
│                        # **Columnar-native storage** ([ADR-0092](docs/adr/0092-duckdb-columnar-native-storage.md)): WithColumnarLayout() extracts ALL exported fields of R into native SQL columns. LayoutPlanApplier interface (DuckDB) receives reflection-derived types (float64→DOUBLE, int→INTEGER). Enables vectorized GROUP BY/SUM/AVG on DuckDB.
│                        # **Replication model** (DDIA Ch5): EngineProfile declares Replication (none/single-leader/multi-leader/leaderless), ReplicationLag (staleness, diagnostic-only), NetworkRTT (additive latency). All current engines are ReplicationNone (zero value). replicationRule emits INFO diagnostic for replicated engines with non-zero lag. mapUpdateReplicationRule emits WARN when Map ADT with update folds is routed to a replicated engine. CollectionInfo exposes Replication/ReplicationLagMs/NetworkRTTMs via store.Collections(). store.ReplicationMode(queryName) returns the topology for a single query. WithReplication/WithNetworkRTT plan options override engine profiles for cost estimation ("what-if" analysis). SerializablePlan includes Replication/ReplicationLagMs/NetworkRTTMs per query. ExplainPlan() shows replication suffix on engine lines; Doctor() has a --- Replication --- section. Foundation for future distributed engines (Iroh, CockroachDB).
│                        # **Persistence** ([ADR-0098](docs/adr/0098-metaengine-persistence-enum.md)): EngineProfile declares Persistence (volatile="" / persistent="persistent"). Zero value is PersistenceVolatile — safe default, planner WARNs. Three engines set it dynamically: SQLite/Pebble/DuckDB are volatile for in-memory constructors (":memory:", vfs.NewMem, dir=""), persistent for file/DB constructors. Memory engine is always volatile; Postgres is always persistent. durabilityRule emits WARN (no persistent alternative) / INFO (persistent alternative exists, shows +Xms/query cost delta) / silent (persistent engine). CollectionInfo exposes Persistence via store.Collections(). store.Persistence(queryName) returns the classification for a single query. SerializableQuery includes Persistence. ExplainPlan() shows "volatile" suffix on engine lines; Doctor() has a --- Persistence --- section listing volatile collections.
│   └── pebbleengine/   # Pebble-backed metaengine Engine (LSM point reads, 7x faster than SQLite on MapGet). MapBackend, ScanBackend, SetBackend, CounterBackend, GraphBackend, MultimapBackend, LogBackend. **RawValueReader + RawScanReader** (eliminates JSON decode tax on point lookups and filtered scans). Separate module (cockroachdb/pebble dep)
│   └── duckdbengine/   # DuckDB-backed metaengine Engine (columnar OLAP, CGo). MapBackend, CounterBackend, ScanBackend, **PushdownScan** (json_extract filter/sort pushdown), **LayoutPlanner** (planned tables with extracted columns + ART indexes), **LayoutPlanApplier** (columnar-native: WithColumnarLayout extracts all fields as typed columns, float64→DOUBLE). Cross-engine parity via adttest.RunMatrix. Separate module (duckdb-go dep, CGo required)
│   └── pgengine/       # Postgres-backed metaengine Engine (JSONB + B-tree). MapBackend, CounterBackend, ScanBackend, **PushdownScan** (JSONB operator filter/sort pushdown), **LayoutPlanner** (expression indexes on JSONB paths). Cross-engine parity via adttest.RunMatrix. Pure Go (pgx driver, no CGo)
│   └── adttest/       # Exported ADT test harness: RunMatrix (10 ADTs), Scenarios, canonicalize helpers — imported by engine modules for cross-engine parity
│   └── projectionadapter/ # Projection adapter: wraps metaengine Store as projection.Projection ([ADR-0062](docs/adr/0062-metaengine-dependency-boundary.md))
├── irohengine/           # Iroh Level 2 replication wrapper: Replicated(localEngine, ...) adds CRDT convergence via pluggable Transport. THREE transports available: InProcessNetwork (goroutine calls, fastest, no CGo), loopback.LoopbackTransport (real TCP, no CGo), quic.QuicTransport (real QUIC via iroh-go, requires CGo + pre-compiled static lib). CRDT-safe: MapSet (LWW), SetAdd (OR-Set), CounterIncrement (PN-Counter), MultiAdd, LogAppend. Non-CRDT ops (MapUpdate) stay local.
│   └── loopback/         # LoopbackTransport: real TCP connections with length-prefix framing (NO CGo). Middle tier of transport testing pyramid: catches serialization/framing bugs that InProcessNetwork cannot, runs in CI without Rust toolchain. 9 convergence tests pass with -race.
│   └── quic/             # QuicTransport: real Iroh QUIC streams via iroh-go CGo bindings (pre-compiled static lib, needs gcc only). Real NAT traversal, real RTT from conn.Rtt() ACK timing, ticket-based bootstrap, relay mode for star topology. 9 convergence tests + 2 reconnect tests pass with -race. Separate module (CGo isolation).
├── idempotency/         # Re-export aliases for github.com/larsartmann/go-idempotency: Store, MemoryStore, ErrDuplicate (core extracted; kvstore/sqlstore remain local — [ADR-0065](docs/adr/0065-extract-idempotency-module.md))
│   └── kvstore/        # KVStore, KVBackend (KV-backed idempotency — optional subpackage, pulls kv/). Record uses SetIfAbsent: no-op on an existing key, TTL NOT extended, matching MemoryStore + sqlstore + the documented Store contract
│   └── sqlstore/       # SQLStore: NewSQLiteStore/NewPostgresStore (INSERT ON CONFLICT DO NOTHING, TTL sweep)
├── dispatcher/          # Generic Dispatcher[H, M] with LifecycleMixin
├── schema/              # Upcaster, VersionedStore, VersionedSeekableJournal, upcasterRegistry (schema evolution); Validator with RegisterType[T]() (ADR-0017)
├── snapshot/            # Snapshot, SnapshotSink/Source/Store, SnapshotStrategy, EveryNEvents
├── storage/memory/       # MemoryStore, MemorySnapshotStore, MemoryCheckpointStore, MemoryCommandStore, MemoryQueryStore (in-memory test impls)
├── catalog/             # Registry, SchemaFromType[T](), AsyncAPI/D2/EventCatalog/OpenAPI exporters
│   └── schema/          # JSON Schema types, reflection engine, YAML serialization
├── middleware/           # Logging, Retry, Recovery, Validation, Idempotency, Metrics, OTel Tracing+Metrics (command+event+query)
├── signing/             # Event signing/verification: HMAC-SHA256, Ed25519, multisig, middleware
├── encryption/          # Event payload encryption: XChaCha20-Poly1305, AES-256-GCM, codec wrapper, middleware
├── storage/             # SQLBackend facade, SQLCommandStore, SQLQueryStore (dispatch log), SQLite/PG helpers. Re-exports eventstore/ + readmodel/ types for backward compat
│   ├── eventstore/      # SQLEventStore, SQLSnapshotStore, SQLCheckpointStore (+ migrations)
│   ├── readmodel/       # SQLKVStore (kv.Store backed by SQL)
│   ├── sql/             # Dialect, DBHandle, OwnedDBHandle, QueryEngine, RunInTx, IsDuplicateKeyError (typed codes + string fallback), ScanSlice, CommitTx, MarshalMetadata
│   ├── relational/      # RelationalSchema, RelationalProjection, RelationalStore, ProjectionSink (multi-table SQL projections, rollup counters via Increment, Resettable, ADR-0033)
│   ├── view/            # SQLViewStore[V,K] (column-mapped views), ViewMapper, ViewColumn, IndexSpec, AutoMapper
│   ├── migrations/      # Embedded .sql DDL files (postgres.sql, sqlite.sql, duckdb.sql) via //go:embed
│   ├── pebble/          # Embedded KV store (PebbleDB): EventStore, SnapshotStore, CheckpointStore, KVAdapter (kv.Store). CBOR envelope, shared DB
│   └── turso/           # Turso database connector (embedded Turso Database sync), indexing advisor
├── otel/                # Shared OpenTelemetry helpers: Tracer, Meter, Spans, Attributes
├── prometheus/         # OTel→Prometheus metrics bridge: Setup() MeterProvider + /metrics HTTP handler, WithRegistry(), WithViews()
├── listing/             # StreamListing, StreamStatus, tombstone detection, StatusMiddleware, InMemoryStreamReader
├── watermill/           # Watermill adapter: event AND command bridges — EventBus, CommandBus, PublisherAdapter, SubscriberAdapter, EventPublisher, CommandPublisher, CatchUpSubscriber (replay+live+checkpoint), MessageToEvent/MessageToCommand
├── transport/http/       # SSE event delivery: SSEBroker, SSEHandler (bridges event.Bus to HTTP clients, ADR-0025)
├── transport/grpc/       # gRPC transport: RegisterCommandService, RegisterQueryService, CommandClient, QueryClient (ADR-0025)
├── codec/               # Payload encoding: JSON, CBOR (deterministic), Raw passthrough
├── graph/               # Graph projection tier: NodeRef, EdgeRef, GraphSink, GraphDriver, GraphProjection, MemoryDriver, Schema (closed-world validation), ReadableDriver (Query/Traverse/Neighbors/ShortestPath) (ADR-0033, ADR-0039)
├── projection/          # Projection interface (consumer-side): Projection, NewProjection — extracted from event/
├── projectionhost/      # Managed projection host: Host, Worker, DeadLetterStore, Reset/Resettable — composes journal + projection + checkpoint + DLQ into a crash-restart lifecycle with OTel tracing, OnFailed callbacks, bounded dedup ring, WorkerDraining status, configurable shutdown timeout (framework gap A1, ADR-0030)
├── scheduling/          # Durable deadline timers: TimerStore, MemoryTimerStore, Scheduler (idempotent schedule/due/fired; "cancel order after 30 min") (framework gap A6)
│   └── sqlstore/       # SQLTimerStore[P]: SQLite/Postgres/MySQL-backed TimerStore — timers survive process restarts (M44)
├── kv/                  # Layer-0 KV store abstraction: Store, MemStore, Iterator, Batch. PLUS TypedStore[T,K], Cache[T,K], ViewStore[V,K] interface, ViewQuery, ViewQuerier, TombstoneQuerier
├── testutil/            # Shared test helpers: NewCmd(tb, ...) (cross-module test utilities)
├── dedup/               # Bounded dedup ring buffer: Ring, DefaultCapacity (O(1) fixed-capacity ID dedup for stream boundaries)
├── retry/               # Re-export aliases for github.com/larsartmann/go-retry: Do, Config, Backoff, ErrExhausted, ErrCanceled (extracted — [ADR-0064](docs/adr/0064-extract-retry-module.md)) — zero CQRS/OTel deps
├── flightrecorder/     # Go 1.25 runtime/trace FlightRecorder wrapper: Recorder, TriggerFunc, OnLatency, OnError, OnErrorOrLatency (zero-dep core; CQRS middleware in middleware/)
├── scenario/            # Fluent BDD test DSL: Given/When/Then + ThenError/ThenState for deciders, GivenProjection/ThenNoError for projections (framework gap A5)
├── cmd/cqrs-gen/        # Code generator: typed handler registration from Go structs
├── cmd/cqrs-lint/       # Domain-aware linter: 186 rules across 10 categories (correctness, API misuse, boilerplate, consistency, architecture, security, performance, version, testing, adoption). Built on go-finding + cmdguard. CLI: struct-tag flags, config file (.cqrs-lint.json), --min-confidence, --health-score, --adoption, --scorecard, --strict-load, --verbose, --group-by (none/module/aggregate), --color, SARIF/JSON/markdown output. Subcommands: version (--verbose), rules, doctor, scorecard, explain, init, changelog. Feature profile system: auto-detects which go-cqrs-lite modules a consumer uses (store, command-flow, server, server-local, soft-delete, tracing, snapshot, transport) and adapts context-dependent rules. `cqrs-lint doctor` prints the detected profile. `cqrs-lint scorecard` (or `--scorecard`) prints a bilateral module-adoption scorecard: "Adoption: 8/15 relevant modules (53%)" with Used/Missing/Irrelevant tables + top-3 recommendations. Uses a hand-curated ModuleCatalog (28 scored + 6 core modules in `pkg/analyzer/module_catalog.go`) with profile-relative denominators (`TestCatalogEveryGoWorkModuleCovered` prevents drift). Config presets (local-cli, production, library, read-only) as unified feature-flags + rule-disable defaults — single source of truth (`PresetDefinitions` map), both `init` and runtime read from it. Warns on unknown preset names and unknown disabled rule IDs. Library self-lint mode (`IsLibrarySelfLint()`) auto-skips consumer-coaching rules when linting the go-cqrs-lite source itself. C008 config overrides: c008-ignore-fields (case-insensitive) and c008-ignore-structs (skip entire structs). Aggregate grouping (`--group-by aggregate`) stamps `Finding.Metadata["aggregate"]` from event type prefixes + decider/fold state types via `enrichWithAggregate`, then groups output by aggregate (most issues first). C038/C039/C040 event-type mismatch + dead-fold-case detection. Per-module feature detection via `ProfileForFile` (C017 migrated; 26 detectors still on primary profile). JSONC config loader (comment support in `.cqrs-lint.json`). `explain` subcommand for interactive config/rules/presets documentation. Version constant must match latest `cmd/cqrs-lint/v*` tag (TestVersionMatchesLatestTag CI gate)
├── cmd/api-stability/   # API surface checker: compares exported symbols against golden file
├── cmd/cqrs-bench/      # CLI: benchmark any backend with named workload profiles (built on benchkit)
├── cmd/doc-check/       # Doc checker: verifies Go import paths + qualified symbols in markdown files
├── benchkit/            # Factory-driven benchmarking suite: Run/Compare, latency percentiles, throughput, memory (mirrors contracttest pattern)
├── integration/         # Cross-module tests (command, event, query, signing, encryption)
├── system/              # Deployer-driven composition root: DomainConfig (consumer) + DeploymentConfig (operator). Driver registry (database/sql model), SQLite + Memory engines, auto-wired projections, scream store safety checks, MultiBus fan-out, introspection. Replaces stack.Bundle with deployer-picks-infrastructure model (D6, D9, D11)
├── example/taskmanager/    # Flagship full HTTP service: event sourcing, CQRS, projections, middleware, OTel, signing
├── example/getting-started/ # Minimal 80-line example showing the core pipeline
├── example/readme-quickstart/ # README-driven quickstart example
└── docs/                # Status reports, ADRs, architecture patterns, storage guide
```

## AI Skill (Crush)

The repo ships a [Crush](https://github.com/crush) skill at `.agents/skills/go-cqrs-lite/` — the **single source of truth for AI consumers** of this library. It replaces the need to read module READMEs.

- **`SKILL.md`** (root, a symlink to `.agents/skills/go-cqrs-lite/SKILL.md`) — a ~1000-char index: one-line mental model + a routing table to the reference guides. Loaded on every trigger.
- **`references/`** — loaded on demand: `core.md` (mental model, quickstart, decision matrix, conventions, anti-patterns, cheat sheet), `recipes.md` (copy-paste composition), `readmodels.md` (projections/SQL views/tier selection), `modules.md` (per-module table), `advanced.md` (14 advanced patterns), `faq.md` (common pitfalls).

**Global availability:** the skill is symlinked into `~/.config/crush/skills/go-cqrs-lite` (reproducibly, via the `flake.nix` devShell `shellHook`) so it triggers from any consumer project, not just inside this repo.

**Contributing:** edit the `.md` files under `.agents/skills/go-cqrs-lite/`, then run `cmd/doc-check` to verify every Go import path + qualified symbol is still valid:

```bash
cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md
```

## Design Principles

1. **Library, not framework** — Consumers import what they need. No opinionated transport, broker, or SQL driver.
2. **Trustworthy modules** — Quality gate: "Would a consumer trust this enough to import it?"
3. **Minimal dependencies** — event depends on `oklog/ulid`, `go-branded-id`, `go-error-family`.
4. **Composition over inheritance** — Per Go best practices.
5. **Interface-first design** — Core store/bus types are interfaces (Store = EventSink + EventSource, ISP split). `event.Event` is a concrete type alias (`= *ImmutableEvent`) — the single implementation, so the interface was removed to eliminate type assertions on every internal hot path.
6. **Interface Segregation** — Journal (ReadAll), SeekableJournal (ReadFrom), BackwardsSource.
7. **Context-aware** — All handlers accept `context.Context`.
8. **Errors as values** — No panics, explicit error returns, sentinel errors + `%w` wrapping.
9. **Strong types** — No `any` as a value type in domain/business logic. Legitimate exceptions: JSON schema serialization (`catalog/`: `map[string]any`, `Payload any`), `recover()` return value (`middleware/recovery.go`), and `database/sql` interop (`storage/sql/dialect.go`). Generic type constraints (`[T any]`) are standard Go and always allowed. Max 350 lines/file (CI-enforced), 30 lines/function.
10. **Multi-module isolation** — Each module has its own `go.mod` with only needed deps.
11. **Tombstone over delete** — Soft-delete via metadata (TombstoneStatus: Active/Tombstoned/Undetermined). No Delete on Store.
12. **Dependency budgets** — Per-module direct PRODUCTION dep limits enforced by `nix run .#check-layers`. Test-only packages (gomega, ginkgo, rapid) are excluded from the count. Adding production deps requires explicit budget review.
13. **OTel through otel/** — Modules import `otel/` re-exports instead of `go.opentelemetry.io` directly. OTel SDK is indirect in decider, storage, middleware go.mod files. The `otel/` module re-exports `Int64Counter`, `AddOption`, `AddSpanEvent()`, `ServiceResourceAttributes()`, `CQRSHistogramBoundaries`, `NewCQRSViews()`, `CounterAddWithAttributes()`, `Setup()`, `WithStdoutExporter()`, `TextMapPropagator()`, and `Version()` for provider setup, stdout exporters, W3C propagation, and instrumentation scope versioning. Span names follow the `{component}.{action}` convention — see `docs/SPAN_NAMING.md`.
14. **Zero-copy internal reads** — `PayloadReadOnly(evt)` bypasses `Payload()` clone for read-only paths by accessing the `*ImmutableEvent` field directly (Event is now a concrete type alias, so no assertion is needed). Used by signing (SHA-256 hashing, CloneEvent), pebble (json.Marshal), storage/sql (ExecContext), transport/http/sse (string conversion). Internal-only `payloadForDecode()` and `encodingForCopy()` for same-package paths.
15. **Defensive clone on all public accessors** — `Payload()` returns `slices.Clone`, `Metadata()` returns `.Clone()`, `EventTypes()` returns `slices.Clone`, `MultiSignature.Get()` returns a copy, `WithCommandMetadata` clones on intake. The `Event` interface documents this contract for third-party implementors.
16. **Hot-path zero-allocation discipline** — Public API clones stay, but internal hot paths eliminate allocs via: lazy map init (`NewMetadata()` returns zero-value), pre-computed middleware chains (EventBus rebuilds on `Use()`/`UsePublish()` only), cached SQL templates (built once at construction), pre-sized result slices (`make([]T, 0, hint)`), batch SQL inserts (multi-VALUES with SQLite 999-param chunking). **Lesson learned**: type assertions for fast paths are dead code if users create types via different constructors. Cache at the integration boundary instead.
17. **Circuit breaker uses failsafe-go** — `middleware/circuit_breaker.go` wraps `failsafe-go/circuitbreaker`. Note: half-open semantics differ from the original hand-rolled version — failsafe-go limits trial executions to `SuccessThreshold` count (not unlimited). The `CircuitBreakerConfig` API is preserved; only the internal implementation changed. Similarly, `decider/cache.go` uses `maypok86/otter/v2` TinyLFU (not hand-rolled LRU).
18. **Load coalescing via singleflight** — `decider.Repository[State]` uses `singleflight.Group` to coalesce concurrent `Load` calls for the same stream into one `store.Load` query. Events are immutable (`*ImmutableEvent`), so sharing the loaded slice across callers is safe. Only load is coalesced — Save/Publish still execute independently per caller. Disable via `WithLoadCoalescing[State](false)`.
19. **Go experimental build tags** — Builds use `-tags "goexperiment.jsonv2"` enabling JSON v2 encoding (`encoding/json/v2`). This is a Go experiment flag, NOT a standard build tag — it requires `GOEXPERIMENT` support in the toolchain. CI and `nix run .#build` apply it automatically. JSON v2 is fully adopted (~25 production files); the tag remains only until Go graduates it from experimental (expected Go 1.27+). Arena allocation was removed — the 36-line stub had zero consumers and provided no real GC benefit.

## Error Handling

- **Sentinel errors**: `errors.New` in `errors.go` files
- **Contextual errors**: `fmt.Errorf("failed to process %s: %w", name, err)`
- **Classified errors**: `errorfamily.NewRejection(...)`, `errorfamily.WrapConflict(...)` via [go-error-family](https://github.com/larsartmann/go-error-family) — imported directly, no facade
- **6-family taxonomy**: Rejection / Conflict / Transient / Infrastructure / Corruption / Orchestration
- **Direct import**: All modules import `errorfamily "github.com/larsartmann/go-error-family"` directly. The `event/` package retains type aliases (`event.Family`, `event.Error`) and family constants for backward compatibility, but error construction/classification/wrapping functions were removed. Use `go-error-family` directly.
- **Error-wrapping helpers**: When `if err != nil { return WrapX(err, code, msg) }; return nil` appears 3+ times in a module, extract an unexported `wrapXOrOK(err, code, msg) error` helper (returns nil when err is nil). Keep helpers per-module — see [ADR-0069](docs/adr/0069-error-wrapping-helpers.md). When modules share a dependency (e.g., encryption + signing → codec), push the helper into the shared module (e.g., `codec.MarshalBase64JSONWithModule`).

## Key Patterns

```go
// Event creation (typed payload, auto-marshaled)
evt, err := event.NewEvent("user.created", userID, "User", event.Version(1),
    UserCreated{Name: "Alice"}, event.WithCorrelationID(correlationID))

// Decider (pure-function)
decider := decider.Decider[State]{Initial: State{}, Fold: foldFunc}
result, err := decider.Repository[State].Execute(ctx, repo, streamID, decider, command)

// Branded IDs (markers are exported: UserMarker, CorrelationMarker, RequestMarker, StreamMarker)
type UserID = id.Of[id.UserMarker]
uid := id.New[UserID]()

// Query dispatch (type-safe)
result, err := query.DispatchTyped[*GetUserResult](ctx, dispatcher, q)

// Sink/Source split (ISP)
var sink event.EventSink = store   // write side: Save, AppendBatch
var source event.EventSource = store // read side: Load, LoadFromVersion, LoadToVersion, LoadToTimestamp
var journal event.Journal = store   // ReadAll (cross-stream)
var seekable event.SeekableJournal = store // ReadFrom (position-based)

// Tombstone soft-delete (no Delete on Store)
status := event.DetectTombstone(events) // Active, Tombstoned, or Undetermined
marked, _ := event.MarkTombstone(evt)   // sets tombstone metadata

// Processing mode (replay vs live context)
replayCtx := event.WithProcessingMode(ctx, event.ModeReplay)
mode := event.ProcessingModeFrom(ctx)    // ModeLive or ModeReplay

// Command & query persistence (audit trail)
//   cmd, _ := command.NewPersistedCommand("user.create", ref, payload)
//   cmdStore.Save(ctx, ref, cmd)         // CommandSink
//   cmds, _ := cmdStore.Load(ctx, ref)   // CommandSource
//   var journal command.CommandJournal = cmdStore        // ReadAll (global)
//   var seekable command.SeekableCommandJournal = cmdStore // ReadFrom(afterCmdID, limit)
//
//   pq, _ := query.NewPersistedQuery("user.get", payload)
//   qStore.SaveQuery(ctx, pq)            // QuerySink
//   queries, _ := qStore.LoadQueries(ctx, after) // QuerySource
//   var qJournal query.QueryJournal = qStore             // ReadAllQueries
//   var qSeekable query.SeekableQueryJournal = qStore     // ReadQueriesFrom(afterReqID, limit)

// Command bus (pub/sub) — typed subscription dispatch
//   bus := command.NewMemoryBus()
//   bus.Subscribe("user.create", handlerFunc)  // typed subscription
//   bus.SubscribeAll(auditHandler)             // catch-all (audit log)
//   bus.Use(middleware.CommandTracing(tracer)) // middleware chain
//   bus.Publish(ctx, cmd1, cmd2)               // variadic publish

// Event causality (command → event traceability)
//   ctx = event.WithCommandCausality(ctx, "user.create", cmdID)
//   // decider.Repository applies CommandCausalityEnricher(ctx) automatically;
//   // resulting events carry metadata.command.type and metadata.command.id
//   cmdType, cmdID, ok := event.CommandCausalityFromContext(ctx)

// Pebble single-DB full stack — PebbleBackend facade (preferred)
//   backend, _ := pebble.Open(dir, &pebble.Options{}, logger)
//   defer backend.Close() // closes DB AND all stores
//   eventStore := backend.EventStore()
//   snapStore  := backend.SnapshotStore()
//   cpStore    := backend.CheckpointStore()
//   // All three share db via disjoint key prefixes (cqrs_event:, cqrs_snapshot:, cqrs_checkpoint:)
//
// Pebble single-DB manual wiring (advanced)
//   db, _ := pebble.Open(dir, &pebble.Options{})
//   eventStore, _ := pebble.NewStore(db, logger)
//   snapStore, _  := pebble.NewSnapshotStore(db, logger)
//   cpStore, _    := pebble.NewCheckpointStore(db, logger)

// Pebble as kv.Store (generic KV interface, ADR-0023)
//   kvStore, _ := pebble.NewKVStore(db, pebble.WithSyncWrites())
//   defer kvStore.Close()
//   kvStore.Set([]byte("k"), []byte("v"))    // → nil
//   val, _ := kvStore.Get([]byte("k"))        // → "v"
//   batch, _ := kvStore.Batch()               // atomic writes
//   // WithBorrowedDB() = adapter doesn't close the DB (shared via Backend)

// Event upcasting (schema migration on load)
//   upcaster := schema.NewUpcaster("UserCreated", 1, func(evt event.Event) (*event.ImmutableEvent, error) {
//       return event.NewEvent(evt.Type(), evt.StreamID(), evt.StreamType(), evt.Version(),
//           newPayload, event.WithSchemaVersion(2))
//   })
//   versioned := schema.NewVersionedStore(store, upcaster)
//
//   // VersionedSeekableJournal — upcasters for projectionhost (ADR-feedback-gap1)
//   // VersionedStore implements EventSource, but projectionhost.New() takes
//   // SeekableJournal. VersionedSeekableJournal bridges this gap.
//   vjournal, _ := schema.NewVersionedSeekableJournal(store, upcaster)
//   host, _ := projectionhost.New(vjournal, cpStore)
//   events, _ := versioned.Load(ctx, id.NewStreamRef("User", streamID))

// Event signing (tamper-proof streams)
//   signer, _ := signing.NewHMAC(secret)
//   bus.UsePublish(signing.SignMiddleware(signer))
//   bus.Use(signing.VerifyMiddleware(signer))

// Event encryption (confidential payloads)
//   enc, _ := encryption.NewXChaCha20Poly1305(key)
//   bus.UsePublish(encryption.EncryptMiddleware(enc, encryption.WithMiddlewareKeyID("key-v1")))
//   bus.Use(encryption.DecryptMiddleware(enc))
//
//   // Composable codec wrapper
//   codec := encryption.NewCodec(codec.JSONCodec{}, enc)
//   alg := encryption.ExtractAlgorithm(evt)  // "xchacha20-poly1305"
//   keyID := encryption.ExtractKeyID(evt)    // "key-v1"

// OTel one-call setup + bundle (preferred for new code)
//   provider, _ := cqrsotel.Setup(
//       cqrsotel.WithService("my-app", "1.0.0", "instance-1"),
//       cqrsotel.WithStdoutExporter(os.Stdout), // dev; swap for OTLP in prod
//   )
//   defer provider.Shutdown(ctx)
//   bundle, _ := middleware.NewOTelBundle(
//       cqrsotel.NewTracer("my-app"), cqrsotel.NewMeter("my-app"),
//   )
//   cmdDisp.Use(bundle.Command()...)
//   bus.Use(bundle.Event()...)
//   bus.UsePublish(bundle.Publish()...)
//   qryDisp.Use(bundle.Query()...)
//   // Tracing-only: pass nil meter + middleware.WithMetricsDisabled()
//
//   // Isolated providers (no global registration — tests, multi-service):
//   provider, _ := cqrsotel.Setup(
//       cqrsotel.WithService("test", "1.0", "test"),
//       cqrsotel.WithoutGlobalRegistration(), // skip otel.Set*Provider calls
//   )
//
//   // otel.Setup + prometheus.Setup composition (tracing + Prometheus metrics):
//   // Call otel.Setup for tracing, then override the meter provider with
//   // prometheus.Setup, passing CQRS views so histogram boundaries are applied.
//   tracingProvider, _ := cqrsotel.Setup(cqrsotel.WithService("app", "1.0", "i1"))
//   defer tracingProvider.Shutdown(ctx)
//   metricsProvider, _ := cqrsprometheus.Setup(
//       cqrsprometheus.WithViews(cqrsotel.NewCQRSViews()...),
//   )
//   defer metricsProvider.Shutdown(ctx)
//   otel.SetMeterProvider(metricsProvider.AsMeterProvider())

// OpenTelemetry tracing (opt-in, no-op when no provider configured)
//   tracer := otel.GetTracerProvider().Tracer("my-app")
//   bus.Use(middleware.EventTracing(tracer))
//   bus.UsePublish(middleware.EventPublishTracing(tracer))
//   cmdDispatcher.Use(middleware.CommandTracing(tracer))

// OpenTelemetry metrics (opt-in, typed metrics via TypedMetricsRecorder)
//   meter := otel.GetMeterProvider().Meter("my-app")
//   recorder, _ := middleware.NewOTelMetricsRecorder(meter)
//   cmdDispatcher.Use(middleware.CommandTypedMetrics(recorder))
//
//   // Rate metrics (Int64Counter + Float64Histogram)
//   counter, _ := meter.Int64Counter("cqrs.operation.count")
//   cmdDispatcher.Use(middleware.CommandOTelMetricsWithCounter(histogram, counter))
//
//   // Span events for projection retry observability
//   cqrsotel.AddSpanEvent(span, "retry_attempt", cqrsotel.AttrInt("attempt", 2))
//
//   // Service identification in traces
//   attrs := cqrsotel.ServiceResourceAttributes("my-app", "1.0.0", "instance-1")
//
//   // Custom histogram boundaries for CQRS latency ranges
//   _ = cqrsotel.CQRSHistogramBoundaries // [0.05, 0.1, ..., 10000] ms

// CBOR compact codec (opt-in, ~35% smaller payloads via toarray)
//   codec := codec.CBORCompactCodec{}  // NOT compatible with CBORCodec data
//   data, _ := codec.Encode(event)     // struct fields encoded as positional array
//
//   // Human-readable CBOR for debugging
//   diag, _ := codec.Diagnose(cborData)
//   log.Printf("CBOR: %s", diag)
//
//   // toarray struct tag — positional CBOR arrays (30-40% payload reduction)
//   type UserCreated struct {
//       _     struct{} `cbor:",toarray"`
//       Name  string
//       Email string
//   }
//
//   // Streaming CBOR encoder/decoder (large batches without materializing)
//   enc := codec.NewCBOREncoder(w)
//   _ = enc.Encode(event)
//   dec := codec.NewCBORDecoder(r)
//   _ = dec.Decode(&event)
//
//   // Zero-allocation encoding via BufferEncoder interface
//   buf := &bytes.Buffer{}
//   if be, ok := codec.(codec.BufferEncoder); ok { be.EncodeToBuffer(payload, buf) }

// Stack-level CBOR default (one-call adoption for all read models)
//   bundle, _ := sqlite.New(dsn, stack.WithDefaultCodec(codec.CBORCodec{}))
//   store, _ := stack.ReadModel[Todo, TodoID](bundle, nil) // nil → uses CBOR
//   mat, _ := stack.NewMaterialize[Todo, TodoID](bundle, nil, keyFunc)   // nil → uses CBOR
//
// ⚠️ CODEC DEFAULTS (read this if you're debugging encoding issues):
//   The default codec differs by layer:
//
//   LAYER                     | DEFAULT CODEC     | HOW TO OVERRIDE
//   --------------------------|-------------------|----------------------------------
//   stack.ReadModel/Materialize| CBORCodec         | stack.WithDefaultCodec(json)
//   event.New()               | CBORCodec         | event.DefaultCodec = codec.JSONCodec{}
//                             |                   |   or event.WithCodec(c) per-event
//   kv.NewTypedStore()        | CBORCodec         | kv.WithTypedCodec(c)
//   snapshot.NewTypedStore()  | CBORCodec         | positional arg: NewTypedStore(store, c)
//   command typed store       | CBORCodec         | positional arg: NewTypedCommandStore(store, c)
//   query typed store         | CBORCodec         | positional arg: NewTypedQueryStore(store, c)
//
//   Events are SELF-DESCRIBING: evt.Encoding() is stamped on every event,
//   so mixed JSON+CBOR event streams decode correctly via DecodePayloadAuto.
//   Blind stores (kv/snapshot/command/query) are now self-describing too via
//   the ADR-0044 envelope: WrapEncode/UnwrapDecode stamp the codec on write and
//   auto-detect it on read. The UnwrapDecode fallback uses JSONCodec for
//   backward compat with pre-envelope data.
//
//   One-call CBOR for both events AND read models:
//   bundle, _ := sqlite.New(dsn, stack.WithEventCodec(codec.CBORCodec{}))
//   // Then in decide functions: event.WithCodec(bundle.EventCodec())

// Mixed-stream decode — JSON and CBOR events in the same store (ADR-CODEC)
//   // DecodePayloadAuto dispatches based on each event's encoding stamp.
//   // Use this in fold/apply functions when migrating from JSON to CBOR:
//   p, err := event.DecodePayloadAuto[UserCreated](evt)
//   // Internally: codec.ForEncoding(evt.Encoding()) picks JSON or CBOR.
//   // For unknown encodings ("raw", "encrypted"), returns an error.

// Pebble recommended defaults (bloom filter, concurrent compactions, logging)
//   backend, _ := pebble.Open(dir, pebble.DefaultOptions(), logger)
//   // Or with operational logging:
//   backend, _ := pebble.Open(dir, pebble.DefaultOptionsWithLogging(logger), logger)
//
//   // LSM metrics for health checks
//   metrics := backend.Metrics()
//   hitRate := float64(metrics.BlockCacheHits) /
//       float64(metrics.BlockCacheHits + metrics.BlockCacheMisses)

// SQLite busy_timeout (eliminates "database is locked" errors)
//   _ = storage.SQLiteEnableWAL(ctx, db)  // now includes busy_timeout=5000

// SQL backend facade — all stores share one *sql.DB connection
//   backend, _ := storage.NewSQLiteBackend(db)  // or NewSQLBackend for Postgres
//   eventStore  := backend.EventStore()                   // *SQLEventStore (eager)
//   cmdStore, _ := backend.CommandStore()                 // *SQLCommandStore (lazy, goroutine-safe)
//   qStore, _   := backend.QueryStore()                   // *SQLQueryStore (lazy, goroutine-safe)
//   snapStore,_ := backend.SnapshotStore()                // *SQLSnapshotStore (lazy, goroutine-safe)
//   cpStore, _  := backend.CheckpointStore()              // *SQLCheckpointStore (lazy, goroutine-safe)
//   defer backend.Close()                                 // closes all stores (NOT the *sql.DB)
//   // Each store embeds *sqlpkg.OwnedDBHandle for Close/checkClosed lifecycle

// SQLite foreign keys (opt-in referential integrity)
//   _ = storage.SQLiteEnableForeignKeys(ctx, db)  // PRAGMA foreign_keys=ON

// HKDF key derivation (multi-tenant encryption)
//   key, _ := encryption.DeriveKey(masterKey, "tenant:acme", 32)  // HKDF-SHA256
//   enc, _ := encryption.NewXChaCha20Poly1305(key)

// Decider singleflight (concurrent load coalescing)
//   // Repository[State] uses singleflight.Group internally — concurrent Load
//   // calls for the same stream coalesce into one store.Load query.
//   // No API change needed; it's transparent.

// Decider hot-state cache (incremental loads — 7.4x faster for hot streams)
//   cache := decider.NewStateCache[MyState](256) // LRU-bounded, process-local
//   repo, _ := decider.NewRepository(store, bus, d,
//       decider.WithStateCache[MyState](cache))
//   // On cache hit: LoadFromVersion(cachedVer) + fold delta → O(new events)
//   // On miss: full Load → O(total events), then cache is populated
//   // Execute updates the cache after every successful write
//   // Fold/store errors invalidate the entry (next Load repopulates from store)

// Read-pressure snapshot strategy (snapshot hot-read, cold-write streams)
//   rp, _ := snapshot.NewReadPressure(50) // snapshot after 50 loads + next write
//   repo, _ := decider.NewRepository(store, bus, d,
//       decider.WithSnapshotStore[MyState](snapStore),
//       decider.WithCodec[MyState](codec.JSONCodec{}),
//       decider.WithSnapshotStrategy[MyState](rp))
//   // Combine with EveryNEvents (either triggers):
//   rp, _ := snapshot.NewReadPressure(50,
//       snapshot.WithInnerStrategy(everyN100))
//   // AggregateAwareStrategy + ReadTracker are optional interfaces;
//   // EveryNEvents (no ReadTracker) still works via ShouldSnapshot fallback.

// OTel distributed correlation (baggage propagation)
//   ctx = cqrsotel.WithCorrelationID(ctx, "abc-123") // store in baggage
//   corrID := cqrsotel.CorrelationIDFromContext(ctx)  // retrieve from baggage
//   otel.SetTextMapPropagator(cqrsotel.NewTextMapPropagator()) // W3C trace + baggage

// Watermill middleware wrappers
//   router.AddMiddleware(watermill.CorrelationIDMiddleware())
//   router.AddMiddleware(watermill.NewRetryMiddleware(watermill.DefaultRetryConfig()))
//   router.AddMiddleware(watermill.ProcessingModeMiddleware()) // reconstruct replay/live flag

// Pebble backup and consistent reads
//   backend.Checkpoint("backups/2026-06-17")         // point-in-time DB snapshot
//   snap := backend.NewSnapshot(); defer snap.Close() // consistent read view

// Codec zero-allocation encoding (BufferEncoder)
//   buf := &bytes.Buffer{}
//   if be, ok := codec.(codec.BufferEncoder); ok { be.EncodeToBuffer(payload, buf) }

// Typed Metadata fields (ADR-0031)
//   // Tracing is embedded in event.Metadata — field promotion keeps JSON shape
//   md := evt.Metadata()
//   fmt.Println(md.CorrelationID)  // promoted from Tracing
//   if md.Tombstone != nil { ... } // typed tombstone mark
//   if md.Causation != nil { ... } // typed command causation

// TypedDecider[State, Cmd] — command type bound at compile time (ADR-0001)
//   d := decider.TypedDecider[CounterState, IncrementCmd]{
//       Initial: CounterState{},
//       Decide:  decideIncrement,
//       Fold:    foldCounter,
//   }
//   repo, _ := decider.NewTypedRepository(store, bus, d)
//   err := repo.ExecuteCommand(ctx, aggID, "Counter", IncrementCmd{Amount: 5})

// kv.TypedStore and kv.Cache (ADR-0032 — moved from readmodel)
//   store := kv.NewTypedStore[UserView, UserID](kvBackend)
//   cache, _ := kv.NewCache[UserView, UserID](store, kv.WithCacheCapacity(500))

// SQL-backed views with queryable columns (storage.SQLViewStore)
//   mapper := storage.ViewMapper[TodoView]{
//       Table: "todos_view",
//       Columns: []storage.ViewColumn[TodoView]{
//           {Name: "title", Type: "TEXT", Extract: func(v *TodoView) any { return v.Title }},
//           {Name: "completed", Type: "INTEGER", Extract: func(v *TodoView) any { return v.Completed }},
//       },
//       ScanRow: func(scan func(dest ...any) error) (*TodoView, error) { ... },
//       TombstoneColumn: "tombstoned", // optional: server-side tombstone filtering
//   }
//   store, _ := storage.NewSQLiteViewStore[TodoView, id.StreamID](db, mapper)
//   mat := stack.Materialize[TodoView, id.StreamID]{Store: store, ...}
//   // Query with SQL power: WHERE, ORDER BY, LIMIT/OFFSET
//   results, _ := store.Query(ctx, kv.ViewQuery{
//       Conditions: []kv.Condition{{Column: "completed", Op: kv.OpEq, Value: false}},
//   })
//
//   // Advanced capabilities (optional interfaces — checked at runtime):
//   // Count:        store.Count(ctx, kv.ViewQuery{Conditions: []kv.Condition{{Column: "completed", Op: kv.OpEq, Value: false}}})
//   // BatchSet:     store.BatchSet(ctx, items) // chunked upsert (SQLite 999-param aware)
//   // DeleteAll:    store.DeleteAll(ctx)       // DELETE FROM table (projection reset)
//   // Query:        store.Query(ctx, kv.ViewQuery{Conditions: []kv.Condition{
//   //                   {Column: "completed", Op: kv.OpEq, Value: false}}, OrderBy: "title", Limit: 10})
//   // AutoMapper:   storage.AutoMapperWithTombstone[TodoView]("todos", "tombstoned") // from struct tags
//   // Indexes:      ViewMapper.Indexes = []storage.IndexSpec{{Name: "idx_title", Columns: []string{"title"}}}
//
//   // From a Bundle preset (one-call path):
//   //   store, _ := sqlite.SQLViewModel[TodoView, TodoID](bundle, mapper)

// Relational projections — multi-table, SQL-dialect-independent (storage.RelationalProjection)
//   // NOTE: This tier is SQL-ONLY (SQLite/Postgres/MySQL), portable at deployment
//   // via the dialect — NOT portable to KV or Graph. Row/column/table/set-predicate
//   // semantics are relational by design. For KV/document backends use
//   // stack.Materialize + kv.ViewStore[V,K] (one document per key). A graph tier
//   // would need a distinct sink (MergeNode/MergeEdge) — see RelationalSink docs.
//   // SQLViewStore/Materialize write ONE record to ONE table per event. When an
//   // event must update several related tables atomically (message + guild +
//   // channel + user + attachments[], a member_roles junction, an append-only
//   // message_edits history table), use RelationalProjection instead.
//   schema := storage.RelationalSchema{Tables: []storage.RelationalTable{
//       {Name: "messages", PrimaryKey: []string{"id"}, Columns: []storage.RelationalColumn{
//           {Name: "id", Type: "TEXT"}, {Name: "channel_id", Type: "TEXT"},
//           {Name: "content", Type: "TEXT"}, {Name: "created_at", Type: "TEXT"},
//       }},
//       {Name: "attachments", PrimaryKey: []string{"id"}, Columns: []storage.RelationalColumn{
//           {Name: "id", Type: "TEXT"}, {Name: "message_id", Type: "TEXT"}, {Name: "filename", Type: "TEXT"},
//       }},
//       // Junction table: composite primary key
//       {Name: "member_roles", PrimaryKey: []string{"guild_id", "user_id", "role_id"}, Columns: ...},
//       // Append-only history: autoincrement PK declared in Type, no PrimaryKey
//       {Name: "message_edits", Columns: []storage.RelationalColumn{
//           {Name: "id", Type: "INTEGER PRIMARY KEY AUTOINCREMENT", Nullable: true},
//           {Name: "message_id", Type: "TEXT"}, {Name: "before_content", Type: "TEXT"},
//       }},
//   }}
//
//   // Handler is dialect-agnostic — never touches *sql.DB. Backend chosen at
//   // deployment via the dialect passed to NewRelationalProjection.
//   proj, _ := storage.NewRelationalProjection("discord-messages", schema, db, sqlpkg.SQLiteDialect{},
//       func(ctx context.Context, evt event.Event, sink storage.ProjectionSink) error {
//           var p MessageCreated
//           _ = json.Unmarshal(evt.Payload(), &p)
//           sink.Ensure(ctx, "channels", storage.Row{"id": p.ChannelID, "name": "", "created_at": p.CreatedAt})
//           sink.Upsert(ctx, "messages", storage.Row{  // conflict on PK "id"
//               "id": p.ID, "channel_id": p.ChannelID, "content": p.Content, "created_at": p.CreatedAt,
//           })
//           for _, a := range p.Attachments {
//               sink.Ensure(ctx, "attachments", storage.Row{"id": a.ID, "message_id": p.ID, "filename": a.Name})
//           }
//           return nil  // all writes commit atomically; error → full rollback
//       }, []event.Type{"MESSAGE_CREATED"})
//   // proj implements projection.Projection → register with any projection runner.
//
//   // Read side: dialect-agnostic queries (replaces hand-written SQL).
//   reader, _ := storage.NewRelationalStore(schema, db, sqlpkg.SQLiteDialect{})
//   counts, _ := reader.CountMany(ctx, []string{"messages", "channels", "users"}) // stats endpoint
//   _ = reader.Query(ctx, "messages", []string{"id", "content"}, kv.ViewQuery{
//       Conditions: []kv.Condition{{Column: "channel_id", Op: kv.OpEq, Value: chID},
//                                   {Column: "created_at", Op: kv.OpLt, Value: cursor}},
//       OrderBy: "created_at", Desc: true, Limit: 50,
//   }, func(scan func(...any) error) error { var r Row; return scan(&r.ID, &r.Content) })

// Incremental rollup counters — pre-materialized O(1) aggregations (storage.ProjectionSink.Increment)
//   // The relational tier's counter primitive. Where Upsert replaces column
//   // values, Increment atomically adds a delta to a numeric column via
//   // INSERT ... ON CONFLICT DO UPDATE SET col = COALESCE(col, 0) + excluded.col.
//   // Used for pre-computed rollup tables (messages per channel per day, total
//   // attachment counts, etc.) so dashboard reads are O(1) instead of O(scan).
//   //
//   // The key Row must include the table's primary key columns and must not
//   // contain the counter column. Multi-counter tables should declare counter
//   // columns as Nullable — COALESCE handles the NULL + N = NULL case when
//   // a different counter creates the row first.
//   sink.Increment(ctx, "channel_activity_by_day", storage.Row{
//       "guild_id": p.GuildID, "channel_id": p.ChannelID,
//       "day": p.CreatedAt.Format("2006-01-02"),
//   }, "message_count", +1) // +1 on create, -1 on delete
//   // Multiple counters on the same row in one handler:
//   sink.Increment(ctx, "attachment_stats", key, "total", 1)
//   sink.Increment(ctx, "attachment_stats", key, "downloaded", 1)

// Relational projection reset — wipe tables for replay from zero
//   // RelationalProjection implements projectionhost.Resettable.
//   // Host.Reset(ctx, "my-projection") calls Reset, which does
//   // DELETE FROM <table> for each table in the schema, then drops
//   // the checkpoint and replays from the beginning.
//   host.Reset(ctx, "discord-messages")

// Graph projections — nodes + edges for traversal-heavy read models (graph.GraphProjection)
//   // The third projection tier. Where Materialize writes ONE document per key
//   // and RelationalProjection writes across SQL tables, GraphProjection merges
//   // events into nodes and edges — the right shape for variable-depth traversal,
//   // adjacency, path-finding, causation DAGs, reply chains, role memberships,
//   // reaction networks. Use when N-hop queries (recursive CTEs in SQL) dominate.
//   //
//   // Writes ARE portable across backends (openCypher MERGE semantics shared by
//   // Neo4j, Memgraph, Apache Age, RedisGraph). Reads are NOT abstracted — run
//   // native Cypher/Gremlin via the driver. This asymmetry is documented.
//   driver := graph.NewMemoryDriver() // or graph/neo4j.NewDriver(...) in sibling module
//   proj, _ := graph.NewGraphProjection("discord-graph", driver,
//       func(ctx context.Context, evt event.Event, sink graph.GraphSink) error {
//           var p MessageCreated
//           _ = json.Unmarshal(evt.Payload(), &p)
//           msgRef := graph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: p.ID}
//           sink.MergeNode(msgRef, map[string]any{"created_at": p.CreatedAt})
//           // Auto-creates endpoint nodes — handlers need not pre-merge.
//           sink.MergeEdge(graph.EdgeRef{Type: "AUTHORED_BY", From: msgRef,
//               To: graph.NodeRef{Label: "User", KeyProp: "id", KeyValue: p.AuthorID}}, nil)
//           // The recursive edge — relational tier needs WITH RECURSIVE CTE.
//           if p.ReplyToMessageID != "" {
//               sink.MergeEdge(graph.EdgeRef{Type: "REPLY_TO", From: msgRef,
//                   To: graph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: p.ReplyToMessageID}},
//                   map[string]any{"at": p.CreatedAt})
//           }
//           return nil // atomic: all merges commit or all roll back
//       }, []event.Type{"MESSAGE_CREATED"})
//   // proj implements projection.Projection → register with any projection runner.
//   // Reads: driver.Query/Traverse/Neighbors/ShortestPath (memory) or native Cypher (Neo4j).
//
//   // Graph Schema validation (opt-in, ADR-0039) — catch typos at the sink boundary
//   schema := &graph.Schema{
//       Nodes: []graph.NodeType{
//           {Label: "User", KeyProp: "id", Properties: []graph.PropertyType{{Name: "name"}}},
//           {Label: "Message", KeyProp: "id", Properties: []graph.PropertyType{{Name: "content"}}},
//       },
//       Edges: []graph.EdgeType{
//           {Type: "AUTHORED_BY", FromLabel: "Message", ToLabel: "User"},
//       },
//   }
//   proj, _ := graph.NewGraphProjection("graph", driver, handler, types, graph.WithSchema(schema))
//   // → MergeNode with Label "Bogus" now returns errSinkUnknownNodeLabel
//
//   // Graph Read API (MemoryDriver only — Go-native predicates, NOT a query language)
//   nodes := driver.Query(graph.Pattern{Label: "User", Where: func(p map[string]any) bool { return p["active"] == true }})
//   ancestors := driver.Traverse(msgRef, "REPLY_TO", -1) // BFS unlimited depth
//   neighbors, edges := driver.Neighbors(centerRef)      // 1-hop adjacency
//   path, _ := driver.ShortestPath(userA, userB)          // BFS shortest path

// gRPC transport (remote command/query dispatch, ADR-0025)
//   srv := grpc.NewServer()
//   cqrsgrpc.RegisterCommandService(srv, cmdDispatcher)
//   cqrsgrpc.RegisterQueryService(srv, qDispatcher)
//   // Client:
//   conn, _ := grpc.NewClient("host:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
//   cmdClient := cqrsgrpc.NewCommandClient(conn)
//   err := cmdClient.Dispatch(ctx, cmd) // transparent remote dispatch

// Flight recorder (Go 1.25 runtime/trace — capture execution trace on slow/error)
//   recorder, _ := flightrecorder.New(
//       flightrecorder.WithMinAge(10*time.Second),
//       flightrecorder.WithFile("slow.trace"),
//   )
//   recorder.Start()
//   defer recorder.Close() // Close stops recording AND closes file writers
//   // Capture when any command exceeds 100ms OR errors:
//   cmdDisp.Use(middleware.CommandFlightRecorder(recorder,
//       flightrecorder.OnErrorOrLatency(100*time.Millisecond)))
//   // Same for events and queries:
//   bus.Use(middleware.EventFlightRecorder(recorder, flightrecorder.OnError()))
//   qryDisp.Use(middleware.QueryFlightRecorder(recorder, flightrecorder.OnLatency(500*time.Millisecond)))
//   // Decider integration (captures on slow/error Execute calls):
//   repo, _ := decider.NewRepository(store, bus, d,
//       decider.WithFlightRecorder[MyState](recorder, flightrecorder.OnErrorOrLatency(200*time.Millisecond)))
//   // Projection host integration (captures on terminal worker failure):
//   host, _ := projectionhost.New(journal, cpStore,
//       projectionhost.WithFlightRecorder(recorder, flightrecorder.OnAlways()))
//   // Stack bundle integration (lifecycle management + discovery):
//   bundle, _ := sqlite.New(dsn, stack.WithFlightRecorder(recorder))
//   // Analyze: go tool trace slow.trace
//   // Once-semantics: first trigger captures, rest are no-ops (call Reset() for multiple)
//   // Only 1 active recorder per process (ErrAlreadyEnabled on double Start)

// Idempotency middleware (dedup for at-least-once delivery, all 3 CQRS message types)
//   store := idempotency.NewMemoryStore(5 * time.Minute)
//   defer store.Close()
//   cmds.Use(middleware.CommandIdempotency(store, 10*time.Minute, nil))
//   bus.Use(middleware.EventIdempotency(store, 10*time.Minute, nil))
//   qry.Use(middleware.QueryIdempotency(store, 10*time.Minute, keyFn))
//   // nil keyExtractor defaults to cmd.ID().String() / evt.ID().String()
//   // For queries, a keyExtractor must be provided (no default identity)

// In-memory command bus (typed pub/sub, first command.Bus impl)
//   bus := command.NewMemoryBus()
//   bus.Subscribe("user.create", handlerFunc)
//   bus.Publish(ctx, cmd1, cmd2)

// Watermill CatchUpSubscriber (replay from journal + live handoff, ADR-0030)
//   catchUp, _ := watermill.NewCatchUpSubscriber(journal, liveSub, cpStore, logger)
//   defer catchUp.Close()
//   msgs, _ := catchUp.Subscribe(ctx, "user.created")
//   // Phase 1: replay historical events with ProcessingMode=ModeReplay
//   // Phase 2: live handoff with EventID-based deduplication
//   // Checkpoint saved after every forwarded event
//
//   // The synchronous replay path is ALWAYS ordered. The LIVE phase uses
//   // BlockPublishUntilSubscriberAck=true for ordered delivery.

// ⚠️ ORDERING — Watermill Router processes messages in parallel (one goroutine
//   per message, message/router.go:30). Do NOT route ordered projections
//   through the Router. Instead, consume the CatchUpSubscriber's output channel
//   from a single goroutine (FIFO guarantees ordering). The EventBus default
//   GoChannel uses BlockPublishUntilSubscriberAck=true + Persistent=false:
//   the former ensures ordered live delivery, the latter avoids GoChannel's
//   unordered Persistent-mode replay (CatchUpSubscriber handles replay from
//   the journal instead). See example/taskmanager for the correct pattern.

// stack.Materialize[V,K] — tombstone-aware projection builder (ADR-0030)
//   mat := stack.Materialize[UserView, UserID]{
//       Store:       kvStore,
//       KeyFromEvent: func(evt event.Event) (UserID, error) { ... },
//       OnCreate:    func(ctx, evt) (*UserView, error) { ... },
//       OnUpdate:    func(ctx, evt, existing *UserView) (*UserView, error) { ... },
//       OnTombstone: func(ctx, evt, existing *UserView) (*UserView, error) { ... },
//   }
//   router.AddNoPublisherHandler("users", topic, catchUpSub, mat.HandlerFunc())

// Watermill EventPublisher — cqrs events → Watermill topic (ADR-0028)
//   pub := watermill.NewEventPublisher(wmPublisher, "events")
//   repo, _ := decider.NewRepository(store, pub, decider)
//
//   // W3C trace context is injected into message metadata on publish.
//   // Extract on consume to link producer → consumer spans:
//   //   ctx := watermill.ExtractContext(msg.Context(), msg)
//   // Or add as router middleware: router.AddMiddleware(watermill.TraceContextMiddleware())

// Watermill CommandBus — command distribution over any broker (ADR-0025)
//   bus := watermill.NewCommandBus()  // GoChannel (single-process)
//   defer bus.Close()
//   bus.Subscribe("user.create", handlerFunc)
//   bus.Publish(ctx, cmd)
//
//   // Multi-process: inject a NATS/Redis/Kafka backend
//   bus := watermill.NewCommandBus(
//       watermill.WithCommandBackend(natsPub, natsSub, closer))
//
//   // Or wrap an existing message.Publisher as command.Publisher
//   pub := watermill.NewCommandPublisher(wmPublisher, "commands")

// Multi-DB SQLite preset (deployer chooses database isolation)
//   bundle, _ := sqlite.New(":memory:",
//       sqlite.WithDSN(
//           sqlopt.WithEventDB("events.db"),   // events+snapshots+checkpoints
//           sqlopt.WithQueryDB("queries.db"),  // command+query audit
//           sqlopt.WithViewDB("views.db"),     // read-model KV store
//       ),
//   )
//
// Durability tiers (unified vocabulary across all SQL + Pebble presets)
//   bundle, _ := sqlite.New(dsn, sqlite.WithDurability(stack.DurabilityStrict))
//   bundle, _ := pebble.New(dir, pebble.WithDurability(stack.DurabilityRelaxed))
//   bundle, _ := postgres.New(dsn, postgres.WithDurability(stack.DurabilityNormal))
//   // Strict:  fsync per commit (SQLite synchronous=FULL, Postgres sync_commit=on)
//   // Normal:  safe against app crash (SQLite synchronous=NORMAL, Postgres sync_commit=off)
//   // Relaxed: data loss possible on crash (SQLite synchronous=OFF, Pebble DisableWAL=true)
//   // Default: DurabilityNormal for every preset
//   tier := bundle.Durability()  // introspect via Bundle accessor
//
// Backend capabilities (machine-checkable tradeoff matrix)
//   caps := bundle.Capabilities()
//   caps.Persistent  // data survives restart
//   caps.Embedded    // in-process (no server)
//   caps.Distributed // cross-process pub/sub
//   caps.OLAP        // columnar/analytical optimized
//   caps.CGoRequired // needs C compiler
//   caps.SyncEnabled // remote sync support (Turso)
//
// SQLite granular options (beyond WithPragmas)
//   bundle, _ := sqlite.New(dsn,
//       sqlite.WithCacheSize(128*1024*1024),  // 128 MB page cache
//       sqlite.WithBusyTimeout(10*time.Second),
//   )
//
// Postgres pool + timeout tuning
//   bundle, _ := postgres.New(dsn,
//       postgres.WithPoolSize(20, 5),               // maxOpen, maxIdle
//       postgres.WithStatementTimeout(30*time.Second),
//   )
//
// Multi-DB Postgres preset (same API, separate databases on same server)
//   bundle, _ := postgres.New(primaryDSN,
//       postgres.WithDSN(
//           sqlopt.WithEventDB("postgres://host/events_db"),
//           sqlopt.WithQueryDB("postgres://host/queries_db"),
//           sqlopt.WithViewDB("postgres://host/views_db"),
//       ))

// Pure event-sourcing mode (no publisher needed)
//   repo, _ := decider.NewRepository(store, nil, decider)
//   // Events are persisted but NOT published — for pure ES without a bus

// Metaengine integration with stack.Bundle (ADR-0061+)
//   // 1. Build the metaengine Store (typed generics — consumer calls Plan)
//   store, _ := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()},
//       metaengine.Query[CountInput, map[string]int64]("counts",
//           metaengine.On(ItemCreated{}, func(e ItemCreated) metaengine.Delta {
//               return metaengine.Delta{e.Status: +1}
//           }),
//       ),
//   )
//   // 2. Register with the Bundle for lifecycle management
//   bundle, _ := sqlite.New(dsn, sqlite.WithStack(stack.WithMetaEngine(store)))
//   // 3. Access via the Bundle
//   counts, _ := metaengine.ExecuteTyped[CountInput, map[string]int64](
//       ctx, bundle.MetaEngine(), CountInput{})
//   // 4. For projection lifecycle, wrap in projectionadapter (separate module)
//   adapter := projectionadapter.New("counts", store, payloadDecoder)
//   host.Register(adapter)
//   // Bundle.Close() now closes the metaengine Store automatically

// Metaengine EventDecoder + eventWithID (recommended for Map ADT queries)
//   // The EventDecoder gives fold handlers full event context (StreamID, Version).
//   // Wrap the typed payload with the stream ID so Map folds can key by entity:
//   type eventWithID[P any] struct { ID string; Payload P }
//   func myDecoder(evt event.Event) (any, error) {
//       id := evt.StreamID().String()
//       switch evt.Type() {
//       case "item.created":
//           var p ItemCreated
//           json.Unmarshal(evt.Payload(), &p)
//           return eventWithID[ItemCreated]{ID: id, Payload: p}, nil
//       ...
//       }
//   }
//   adapter := projectionadapter.New("items", store, nil,
//       projectionadapter.WithEventDecoder(myDecoder))

// Metaengine FilterOnField + SortOnField (SQLite json_extract pushdown)
//   // Declare at query time — planner pushes filter/sort to SQLite WHERE/ORDER BY.
//   // 50x faster than Memory engine O(N) scan at 10K rows.
//   q := metaengine.Query[ListInput, ItemView]("items",
//       metaengine.On(CreatedEvent{}, func(e CreatedEvent) (string, ItemView) { ... }),
//       metaengine.FilterOnField[ItemView]("status", metaengine.FilterEq),
//       metaengine.SortOnField[ItemView]("priority", true), // DESC
//   )

// Metaengine TypedReader (typed Scan/Get without ExecuteTyped)
//   reader := metaengine.NewReader[ItemView](store, "items")
//   items, _ := reader.Scan(ctx,
//       metaengine.WithFilter("status", metaengine.FilterEq, "active"),
//       metaengine.WithSort("priority", true),
//       metaengine.WithLimit(50))
//   item, found, _ := reader.Get(ctx, "item-123")

// Metaengine QueryBuilder (fluent API on top of TypedReader)
//   qb := metaengine.NewQueryBuilder[ItemView](reader)
//   results, _ := qb.Where("status", metaengine.FilterEq, "active").
//       OrderBy("priority", true).Limit(50).Execute(ctx)

// DuckDB preset — embedded analytical (OLAP) engine (CGo required)
//   b, _ := duckdb.New("analytics.db")     // persistent file
//   defer b.Close()
//   // Or in-memory: duckdb.New("")
//   // Tuning: duckdb.New("", duckdb.WithThreads(4), duckdb.WithMemoryLimit("1GB"))
//   // DuckDB excels at analytical queries, GROUP BY aggregations, columnar scans.
//   // CGo required: statically links C++ engine (~30-50MB binary).
//   // Isolated in stack/duckdb module — consumers who don't import it never need CGo.

// Metaengine planner pipeline (composable PlanRule chain — ADR-0083)
//   // The planner runs a sequence of PlanRule implementations. Each rule can
//   // inspect the query declaration, engine profiles, and statistics, then
//   // modify the PlanResult (assign engine, emit diagnostics, apply layout).
//   //
//   // Default rules (in order):
//   //   1. EnforceSchemaCompatibility — fold valueType must match result type
//   //   2. AutoLayout — detect LayoutPlanner engines, generate DDL
//   //   3. DetectWriteAmplification — warn if fold produces >1 write per event
//   //   4. CheckScaleThreshold — warn if estimated N exceeds engine capacity
//   //   5. VersionedReadCheck — warn if temporal query needs VersionedStorage
//   //
//   // Custom rules can be added via Plan() options (future API).
//   report := store.Explain(ctx)
//   // report shows: assigned engine per query, rule diagnostics, cost estimate

// Metaengine materialize-vs-replay (THE ES-specific killer feature)
//   // The planner can recommend whether to materialize a projection (store
//   // the result) or replay events on each read. The cost formula:
//   //
//   //   replay_cost(q)      = read_rate × avg_stream_length × fold_cost_per_event
//   //   materialize_cost(q) = write_rate × fold_cost_per_event + read_rate × query_cost
//   //
//   // Materialize when: materialize_cost < replay_cost
//   //
//   // This is advisory (INFO/WARN diagnostic in PlanResult), not a hard override.
//   // Future: a WithStats option will feed write/read rates for automatic recommendations.

// Metaengine new ADTs: Vector, Search, Spatial (ADR-0085)
//   // Three new ADTs extend the planner beyond CRUD:
//   //
//   //   ADTVector  → VectorBackend  (k-NN similarity search, cosine/euclidean/dot)
//   //   ADTSearch  → SearchBackend  (full-text search, TF-IDF inverted index)
//   //   ADTSpatial → SpatialBackend (geo range queries, haversine distance)
//   //
//   // Classification priority: Vector → Search → Spatial → Graph → Counter → ...
//   // Each has a typed execute helper: VectorExecuteTyped, SearchExecuteTyped, SpatialExecuteTyped
//   //
//   // Currently only Memory engine implements these backends (brute-force).
//   // Future: DuckDB VSS extension (vector), Postgres tsvector (search), PostGIS (spatial).

// Metaengine temporal queries (VersionedStorage — Memory engine only)
//   // The Memory engine tracks version chains for point-in-time reads:
//   val, err := store.ExecuteAsOf(ctx, "users", "u1", timestamp)
//   // Returns the value as it existed at that time, or metaengine.ErrNotFound

// Metaengine replication model — distributed engine foundation (DDIA Ch5)
//   // EngineProfile declares HOW data propagates (not queries — queries declare
//   // WHAT to compute). All current engines are ReplicationNone (zero value).
//   //
//   //   Replication            = topology: none | single-leader | multi-leader | leaderless
//   //   ReplicationLag         = staleness (diagnostics only, NOT latency)
//   //   NetworkRTT             = additive latency (DDIA Ch1, does NOT scale with volume)
//   //
//   // Cost formula: estimated_latency = (ops × nsPerRead / 1e6) + NetworkRTT
//   //
//   // Declaring a replicated engine (future Iroh/CockroachDB):
//   profile := metaengine.EngineProfile{
//       Name: "iroh-sync",
//       Supports: map[metaengine.ADT]metaengine.Complexity{
//           metaengine.ADTMap: metaengine.ComplexityO1,
//       },
//       Replication:    metaengine.ReplicationLeaderless,  // CRDT convergence
//       ReplicationLag: 200 * time.Millisecond,
//       NetworkRTT:     5 * time.Millisecond,
//   }
//   // The planner emits an INFO diagnostic when routing to a replicated engine
//   // with non-zero lag: "routed to leaderless engine ... reads may be stale by 200ms".
//   // String() output: "iroh-sync: map@O(1) (replication=leaderless, lag=200ms, rtt=5ms)"

// Metaengine persistence — survivability classification (DDIA Ch1, ADR-0098)
//   // EngineProfile declares WHETHER data survives process exit.
//   // Zero value is PersistenceVolatile — safe default, planner WARNs.
//   //
//   //   Persistence = volatile ("" / zero value) | persistent ("persistent")
//   //
//   // Three engines set it dynamically:
//   //   pebbleengine.NewPebbleEngine("")    → volatile (vfs.NewMem)
//   //   pebbleengine.NewPebbleEngine("/db") → persistent (disk LSM)
//   //   duckdbengine.New("")                → volatile (:memory:)
//   //   duckdbengine.New("file.db")         → persistent (disk)
//   //   Memory engine                       → always volatile
//   //   Postgres                            → always persistent
//   //
//   // The durabilityRule planner rule emits:
//   //   WARN  — volatile engine, no persistent alternative exists
//   //   INFO  — volatile engine, persistent alternative exists (+Xms/query cost delta)
//   //   silent — persistent engine
//   //
//   // Inspect:
//   eng.Profile().IsVolatile()                // bool
//   eng.Profile().IsPersistent()              // bool
//   store.Persistence("find_user")            // PersistenceVolatile or PersistencePersistent
//   report := store.Doctor(ctx)               // --- Persistence --- section lists volatile collections
//   // SerializableQuery includes Persistence for plan serialization/diff/pin

// Pebble backup + graceful shutdown (production operations)
//   b, _ := pebble.New("/var/lib/myapp/pebble")
//   defer b.GracefulClose(ctx) // Bundle.GracefulClose bounds Close with a timeout
//   _ = b.Checkpoint("/backups/2026-06-21") // point-in-time physical snapshot
//   m := b.Metrics()                         // LSM health (block cache hit rate, etc.)

// Iroh CRDT replication — three-tier transport testing pyramid
//   // Tier 0 (fastest, no CGo): InProcessNetwork — goroutine function calls
//   // Catches: CRDT merge logic, subscriber dispatch, ordering correctness
//   network := irohengine.NewInProcessNetwork()
//   tA := network.Join("node-a")
//   engine := irohengine.Replicated(memEngine,
//       irohengine.WithTransport(tA),
//       irohengine.WithAuthor("node-a"))
//
//   // Tier 1 (middle, no CGo): loopback.LoopbackTransport — real TCP connections
//   // Catches: serialization bugs, length-prefix framing, connection lifecycle, partial reads
//   tA, _ := loopback.New()
//   tB, _ := loopback.New()
//   _ = tB.Connect(tA.Addr()) // real TCP dial
//
//   // Tier 2 (full, CGo): quic.QuicTransport — real QUIC via iroh-go
//   // Catches: NAT traversal, QUIC ACK timing, connection migration
//   // Requires: CGO_ENABLED=1, gcc (pre-compiled static lib, no Rust toolchain needed)
//   tA, _ := quic.New(quic.WithLocalOnly(), quic.WithRelay())
//   ticket, _ := tA.Ticket() // base32 connection string for peer bootstrap
//   _ = tB.Connect(ticket)   // real QUIC stream

// SSE with Last-Event-ID reconnection (resilient event delivery)
//   broker, _ := http.NewSSEBroker(bus,
//       http.WithReconnectJournal(journalStore, 1000)) // replay cap
//   // Clients sending "Last-Event-ID" header get missed events replayed
//   // from the journal before live streaming begins. Dedup prevents
//   // duplicate delivery (same strategy as CatchUpSubscriber).
//
//   // Unlimited replay + browser timeout safety:
//   broker, _ := http.NewSSEBroker(bus,
//       http.WithReconnectJournal(journalStore, 0), // 0 = unlimited streaming
//       http.WithReplayTimeout(30*time.Second))     // stop replay after 30s
//   // replayLimit <= 0 streams ALL events in batches of 500 (memory-bounded).
//   // WithReplayTimeout caps the duration — on timeout, an
//   // SSEReplayIncompleteEvent advisory is sent before live starts.
//   // DefaultSSEReplayLimit (1000) is the suggested bounded cap.
//
//   // Byte-budgeted replay (prevent memory exhaustion from large payloads):
//   broker, _ := http.NewSSEBroker(bus,
//       http.WithReconnectJournal(journalStore, 0),
//       http.WithReplayByteBudget(8*1024*1024))           // 8MB default
//   // WithReplayByteBudget(0) also defaults to 8MB (safety).
//   // http.SSEReplayBudgetDisabled = -1 to explicitly disable budgeting.
//
//   // Payload transform for non-JSON codecs (CBOR→JSON for browsers):
//   broker, _ := http.NewSSEBroker(bus, http.WithPayloadTransform(http.CBORToJSONTransform))
//   // CBORToJSONTransform wraps codec.TranscodeToJSON with graceful fallback
//   // (raw payload on decode failure). Without a transform, CBOR-encoded events
//   // go out as raw CBOR bytes that browsers cannot parse.
//   // Applied uniformly across live, replay, AND backfill paths.
//   // For schema-aware JSON (field names from toarray structs), use a custom
//   // transform with event.DecodePayloadAuto[T].
//
//   // BackfillHandler — REST backfill using the broker's journal + transform:
//   // GET /events/backfill?after=<event-id>&limit=500 → JSON array of events.
//   // The broker's WithPayloadTransform (if set) is applied automatically,
//   // so SSE and REST backfill share the same codec configuration.
//   mux.Handle("/events/backfill", http.BackfillHandler(broker))
//   // The broker must have WithReconnectJournal configured; otherwise 503.

// Managed projection host — the "last loop every consumer rewrites" (projectionhost)
//   host, _ := projectionhost.New(journal, checkpointStore,
//       projectionhost.WithBatchSize(100),
//       projectionhost.WithDeadLetterStore(dlqStore, 3), // poison after 3 retries
//       projectionhost.WithOnFailed(func(name, err string) { // terminal failure alert
//           alerting.Notify(ctx, fmt.Sprintf("projection %q failed: %s", name, err))
//       }),
//       projectionhost.WithShutdownTimeout(60*time.Second), // default 30s
//   )
//   _ = host.Register(&UserProjection{})  // Register returns error (name must be unique)
//   _ = host.Register(&OrderProjection{})
//   go host.Start(ctx)          // one goroutine per projection, crash auto-restart + backoff
//   defer host.Stop()           // graceful drain (WorkerDraining → WorkerStopped)
//   for _, s := range host.Status() {  // health: idle/running/live/backoff/draining/stopped/failed
//       fmt.Printf("%s: %s processed=%d errors=%d lag=%s\n", s.Name, s.Status, s.Processed, s.Errors, s.Lag)
//   }
//   // Rebuild a projection from scratch after fixing a handler bug:
//   host.Stop()
//   host.Reset(ctx, "users") // drops checkpoint + calls Resettable.Reset if implemented
//   // host.Reset(ctx, "users", projectionhost.WithPurgeDeadLetters()) // also purge DLQ entries
//   host.Start(ctx)           // replays from zero
//   // Per-projection lag for dashboards:
//   for name, lag := range host.LagPerProjection() {
//       gauge.WithLabelValues(name).Set(float64(lag.Milliseconds()))
//   }
//   // Total lag (max across all workers):
//   gauge.Set(float64(host.LagDuration().Milliseconds()))
//   // OTel tracing: automatic spans (projectionhost.handle_event) when provider configured
//   // Reads directly from event.SeekableJournal — no Watermill dependency.
//   // For live push delivery, pair with watermill/CatchUpSubscriber.
//
//   // SQLite dead-letter store (persists poison events across restarts):
//   dlqStore, _ := projectionhost.NewSQLiteDeadLetterStore(ctx, db)
//   host, _ := projectionhost.New(journal, cpStore,
//       projectionhost.WithDeadLetterStore(dlqStore, 3))
//
//   // Dead-letter store admin (production management — DeadLetterStoreAdmin interface):
//   if admin, ok := dlqStore.(projectionhost.DeadLetterStoreAdmin); ok {
//       count, _ := admin.Count(ctx)                                     // total entries
//       page, _ := admin.ListPaged(ctx, "users", 0, 100)                // paginated list
//       deleted, _ := admin.PurgeBefore(ctx, time.Now().Add(-7*24*time.Hour)) // cleanup old entries
//   }

// Scenario-testing DSL — fluent Given/When/Then for deciders + projections (scenario)
//   scenario.Given[incrementCmd, counterState](t, foldCounter, counterState{},
//       mustEvent(evtIncremented)).           // pre-existing events folded into state
//       When(incrementCmd{}, decideIncrement). // pure decide function
//       Then(evtIncremented)                   // asserts emitted event TYPES
//   // Also: .ThenError(target) / .ThenState(fold, initial, expected)
//   scenario.GivenProjection(t, proj, evt1, evt2).ThenNoError()  // projection handles all OK

// Scheduled commands / durable deadlines — "cancel order after 30 min unpaid" (scheduling)
//   store := scheduling.NewMemoryTimerStore()
//   sched := scheduling.New(store, dispatchFunc,
//       scheduling.WithPollInterval(500*time.Millisecond),
//       scheduling.WithMaxRetries(5),
//   )
//   _ = store.Schedule(ctx, scheduling.Timer{
//       ID: "order-123-timeout", FireAt: time.Now().Add(30 * time.Minute), Payload: cancelCmd,
//   })  // idempotent: re-scheduling the same ID is a no-op
//   _ = store.Cancel(ctx, "order-123-timeout") // order paid → cancel timeout
//   go sched.Start(ctx) // polls Due(), dispatches, MarkFired(); retries failed dispatches

// Health checks — Kubernetes liveness/readiness probes (stack.Bundle)
//   bundle, _ := sqlite.New(dsn)
//   if err := bundle.HealthCheck(ctx); err != nil {
//       // database unreachable or a registered resource is unhealthy
//   }
//   // SQLEventStore implements stack.HealthChecker via db.PingContext.
//   // Bundle.HealthCheck pings the DB + calls HealthCheck on every registered
//   // closer that implements the interface.

// Shutdown ordering — declare close-time dependencies (stack.Bundle)
//   bundle, _ := sqlite.New(dsn,
//       stack.WithShutdownDependency("eventstore", "projectionhost"),
//   )
//   defer bundle.Close() // topological sort: eventstore closes AFTER projectionhost
//   // Cycles fall back to registration order. Use for stores that projections
//   // read from — projections must drain before the event store closes.
```

## Testing

- Table-driven tests preferred; BDD via Ginkgo v2 + Gomega for event/decider/query; fluent Given/When/Then via `scenario/v4` (framework gap A5)
- `t.Parallel()` for independent tests; core packages >80% coverage (most >90%)
- Per-module isolation: `cd event && GOWORK=off go test ./... -count=1`
- Golden tests use `go-snaps` (`snaps.MatchSnapshot`) — powered by `eventtest.AssertGolden` from `event/v4/eventtest`. Update snapshots with `UPDATE_SNAPS=true go test ./...` (or `-update` flag, honored backward-compat). Colored diff on mismatch. Clean obsolete: `UPDATE_SNAPS=clean go test ./...`. Each module using golden tests has a `snaps_clean_test.go` with `TestMain` calling `snaps.Clean(m)` for obsolete-snapshot cleanup
- Modules without event dependency (otel, codec) use `go-snaps` directly via local `matchGolden(t, name, got)` helpers
- **eventtest nested module**: `event/v4/eventtest` lives at `event/v4/eventtest/` (directory MUST match module path per Go spec — a path without a trailing `/vN` suffix requires `go.mod` at the exact subdirectory). `go mod tidy` in `event/` (or any consumer) emits warnings about the nested `go.mod`; run `go mod tidy -e` to suppress (warnings-only, not a build failure). Tagged as `event/v4/eventtest/v0.1.0` (v0 because the path's last element is `eventtest`, not `/vN`). See `docs/adr/0045-eventtest-module-path-fix.md` for the full rationale.
- **Postgres integration tests** use `testcontainers-go` (postgres:16-alpine). Each test gets its own fresh database within a shared container for isolation (critical: `contracttest.RunSuite` runs subtests in parallel). Tests run automatically when Docker is available; skip gracefully when not. `POSTGRES_TEST_DSN`/`DATABASE_URL` env var overrides for CI service containers. Pattern: TestMain starts shared container, `postgresDSN(t)` creates per-test DB.
- **Nix-based integration tests (no Docker needed)** — three approaches, all using nixpkgs services pinned by flake.lock:
  - **Ephemeral PG** (`nix run .#integration-pg`): starts a `pg_ctl` process from nixpkgs in a temp dir, runs all PG integration tests, cleans up. No VM, no Docker. Fast (~3s startup). Works on macOS too.
  - **NixOS VM tests** (`nix build .#checks.x86_64-linux.postgres-vm`): boot a QEMU VM with `services.postgresql`, verify service health, JSON operations, and LISTEN/NOTIFY. Cached by Nix. Runs in CI without Docker.
  - **MySQL VM test** (`nix build .#checks.x86_64-linux.mysql-vm`): same pattern for MariaDB. Required because MariaDB's `install-db` is broken on NixOS host (read-only Nix store plugin dir permissions).
  - **MySQL nspawn test** (`nix build .#checks.x86_64-linux.mysql-nspawn`): systemd-nspawn container variant of the MySQL VM test. ~10x faster (~15s vs ~131s) because nspawn shares the host kernel — no full QEMU boot. Uses `containers.machine` instead of `nodes.machine` in `runNixOSTest`. Requires `uid-range` system feature + `auto-allocate-uids` on the host. One-shot setup: `sudo bash scripts/enable-nspawn-support.sh`. For interactive integration tests: `sudo nix run .#integration-mysql-nspawn` (builds the driver without uid-range, then runs it with root). Falls back to QEMU automatically when nspawn is unavailable.
  - VM tests live in `nix/vm/postgres.nix` + `nix/vm/mysql.nix` and are wired as `checks` in flake.nix. The `nixos-vm-tests` CI job runs them.
- **Race-aware test thresholds**: the `-race` detector inflates allocations and CPU 5-10x, so hardcoded timing/heap thresholds in tests flake under `-race`. Use the `testutil.RaceEnabled` build-tag constant (`testutil/race_on.go` + `race_off.go`) to pick a relaxed bound: `if testutil.RaceEnabled { hang = 30*time.Second }`. Modules with a lean dependency budget that cannot import testutil (e.g. `benchkit`, `transport/grpc`) copy the two-file idiom locally (`benchkit/race_on.go`/`race_off.go`, `transport/grpc/race_on_test.go`/`race_off_test.go`) — the latter uses `_test.go` suffix since the constant is test-package only. See the file headers for the rationale. Always run the affected test 3x with `-count=3 -race` after touching a threshold.
- **Soak test env vars**: `SOAK_SKIP_10M=1` skips the 10M-event memory-bounded soak test (`TestSoak_MemoryBounded_10M`, ~5s/25s-race). Use in CI or when the full verify gate is already running heavy parallel tests. The 50K-event `TestSoak_MemoryBounded` always runs as the smoke variant. Both tests report `TotalAlloc` delta (allocs/event) alongside heap growth for allocation-rate diagnostics.

### Lint Conventions

- **Always `nix fmt` BEFORE placing `//nolint` directives** — golines (max-len: 120) reformats long lines and moves nolint comments to wrong positions
- For `gosec` G115 (integer overflow) conversions, extract a helper function that isolates the `uint64()`/`uint32()` call on a short single line
- **Scoped formatting**: `nix fmt` runs treefmt on the whole repo (can be slow). For a single module, use `gofumpt -w <path>` + `goimports -w <path>` directly
- Keep `//nolint` comments under ~40 chars to survive formatting
- When adding new dependencies, add them to `.golangci.yml` depguard allow list at the same time
- **SQL store helpers live in `storage/sql/`** — `RunInTx`, `IsDuplicateKeyError`, `CommitTx`, `ScanSlice`, `MarshalMetadata`. Don't duplicate transaction/duplicate-key logic in domain-specific store files. The `sql` package already imports `otel` for span recording.
- **`scanCommand` and `scanQuery` must unmarshal metadata** — both scan a `metadataJSON []byte` column. Use `json.Unmarshal` into `command.Metadata` / `query.Metadata` (standalone structs, NOT aliases for `event.Metadata` — each module owns its own shape), then pass via `WithCommandMetadata` / `WithQueryMetadata`. Forgetting this causes silent metadata loss on SQL load.
- **Process safety: NEVER commit code that doesn't compile** — Commit `b3931503` shipped `slices.Contains()` with zero arguments, breaking the entire `cmd/cqrs-lint/pkg/rules/api` package. The BuildFlow pre-commit hook catches this. If you bypass it, run `go build ./...` manually before committing. A meta-test in `cmd/cqrs-lint/pkg/rules/meta_test.go` now instantiates all 65 detectors to guard against broken constructors.
- **Verification gate: `nix run .#verify`** — One command: build + vet + test + race + lint + doc-check + doc-assertions. Run before tagging releases.
- **Release process** — See `CONTRIBUTING.md` → Release Process. Per-module annotated tags via `scripts/tag-release.sh`. Never use lightweight tags.
- **NEVER use `git checkout <commit> -- .`** — This destructively overwrites the working tree. To inspect or test code at a specific commit, use `git worktree add /tmp/work <commit>` instead. Worktrees are isolated, non-destructive, and cleanable with `git worktree remove`.
- **Private Go module auth (non-interactive fetch)** — the devShell sets `GOWORK=off`, so `govalid` / `go mod download` / `vulncheck` fetch every internal module from VCS as a consumer would. `GOPRIVATE` makes Go bypass the public proxy and clone over **HTTPS**, which fails without a display/keyring: `could not read Username for 'https://github.com': terminal prompts disabled` (exit 128). Tagged modules only work because they're already in `~/go/pkg/mod`; any **untagged pseudo-version not yet cached** (e.g. `stack/duckdb/v4`, never tagged) triggers a fresh fetch → auth fail → cascades as `markers: failed prerequisites` / `could not import ... (invalid package name: "")` and kills `govalid-generate`. **Fix:** the flake devShell `shellHook` exports `GIT_CONFIG_*` to redirect `https://github.com/larsartmann/` → `git@github.com:LarsArtmann/` (SSH, uses your key, no credential helper). To apply outside the devShell, add the same `url.<base>.insteadOf` to `~/.gitconfig` (the nix-managed `~/.config/git/config` is read-only). Symptom signature: `git ls-remote -q origin ... exit status 128` inside `~/go/pkg/mod/cache/vcs/`.
- **Verify module version exists before requiring it** — Before adding a `require github.com/larsartmann/go-cqrs-lite/<module>/v4 v4.x.y` line, ALWAYS check the tag exists: `git tag -l '<module>/v4.x.y'`. Commit `169b5d42` shipped a broken `integration/go.mod` because `idempotency/sqlstore/v4@v4.1.0` was assumed by analogy with siblings but never tagged. One-second check prevents the entire broken-go.mod-commit class of failure.
- **API-surface changes require golden regen in the same edit** — Whenever you add/rename/remove an exported symbol (function, type, method, constant, variable), immediately regenerate the api-stability golden: `cd cmd/api-stability && GOWORK=off go run main.go -update`. Do NOT rely on the `#verify` gate to catch this — it catches it, but at the cost of a full 3-4 min verify cycle wasted on a stale golden. The gate is a backstop, not a planner.
- **Every directory with a `go.mod` must be in the api-stability modules list** — The `TestEveryGoModDirIsInModulesList` meta-test enforces this automatically. If you create a new module, add it to `cmd/api-stability/main.go` `modules` slice in the same change.
- **gopls shows phantom errors after file splits / large moves** — gopls keeps an in-memory snapshot of package positions. After a file split (e.g. `sink.go` → `sink.go` + `sink_advanced.go`) it can report `DuplicateMethod`/`already declared` at line numbers that **no longer exist** in the cited file (file is shorter than the cited line). Root cause = stale snapshot, NOT a real error. **Fix: restart the gopls LSP** (`lsp_restart gopls`); do NOT trust gopls immediately after a refactor — `go build -tags "goexperiment.jsonv2" ./...` and `go vet` are authoritative. Also: gopls runs WITHOUT the `goexperiment.jsonv2` build tag, so its analysis of `encoding/json/v2` code is unreliable; right after a restart it briefly floods "`X` is not in your go.mod file" across the whole workspace until it finishes re-loading the 58-module graph — ignore those until reindex completes.
- **Dedup helper patterns** (introduced 2026-07-27, clone groups 34→19): `storage/memory` uses `withWriteLock(code, msg, fn)` methods (write-side) + `withReadLock[T](s, code, msg, fn)` top-level generic functions (read-side, Go has no generic methods) + `wrapClosed(err, code, msg)` for the `CheckClosed → WrapInfrastructure` nil-short-circuit pattern — each store defines its own pair because `CheckClosed` sentinels differ per store. Test helpers: `parallelTimeoutCtx(t, timeout)` in benchkit, `parallelViewStore(t)` in storage/view, variadic `NewTestRegistry(svc...)` in catalog. The `.art-dupl-baseline.json` golden + `nix run .#check-duplication` gate enforce no-new-clones; run `art-dupl baseline . --threshold 3 --semantic` to update after an accepted consolidation. Coverage drift is checked by `nix run .#check-coverage` (`scripts/check-coverage.sh`).
- **Auto-commit daemon can break the build** — commit `85ac81f1` bumped `go-output` root to v0.33.0 but `go-output/table` maxes at v0.32.0 (no v0.33.0 release), silently breaking `cmd/cqrs-lint` for 3+ sessions while the verify-gate "GREEN" claim stayed stale. The daemon ships real features (DSN busy_timeout, multi-DB support) but also ships breaking bumps. Always run `go build -tags "goexperiment.jsonv2" ./...` after a daemon commit, not just `nix run .#build` (which uses `allPaths` — verify the cmd/* modules actually compile).
- **"Stale GREEN" anti-pattern** — claiming `nix run .#verify` is GREEN based on a prior session's run, without re-running it in the current session. This occurred across 4+ sessions (07-25_14-19, 07-25_17-32, 07-25_19-00, 07-26_06-39). The verify gate takes 3-4 minutes; the temptation to skip it is strong. RULE: every session that changes code, go.mod, or docs must run `nix run .#verify` (or at minimum `nix run .#verify-fast`) before claiming GREEN. The verify gate is the ONLY source of truth for build/lint/test status. A stale GREEN claim is worse than no claim — it lulls the next session into false confidence.
- **Version-sequence breaks in published tags** — tags must be monotonically increasing in BOTH semver AND commit ancestry. A tag created later chronologically but with a LOWER semver (e.g., `storage/v4.2.0` tagged after `storage/v4.3.1` but v4.2.0 < v4.3.1) causes consumers resolving "latest" to get the OLDER version missing new functions. The `storage/v4.3.1` tag lacked `EnsureSQLiteDSNBusyTimeout` even though `storage/v4.2.0` (chronologically newer) had it. Fix: always tag with the NEXT semver above all existing tags, and verify with `git tag -l '<module>/v4*' | sort -V | tail -1`. The `nix run .#vulncheck` gate (with `-tags "goexperiment.jsonv2"`) catches these by building each module standalone (GOWORK=off).
- **WithoutGlobalRegistration for isolated OTel providers** — `otel.Setup(cqrsotel.WithoutGlobalRegistration())` skips the `otel.SetTracerProvider()` / `otel.SetMeterProvider()` global calls. Use in tests and multi-service setups where global state would conflict. Without this flag, calling `Setup()` twice in the same process panics or silently overwrites the first provider.
- **DuckDB/CGo isolation** — The `stack/duckdb` module is the ONLY module requiring CGo (`//go:build cgo` on `drivers.go`). The DuckDB driver statically links a C++ engine (~30-50MB binary). It is isolated in its own Go module so consumers who don't import it never need CGo. The devShell includes `pkgs.gcc` for this purpose. DuckDB's `metadata` column is `BLOB` (not VARCHAR) to avoid byte-slice escaping issues on roundtrip. DuckDB dialect uses `$1` placeholders (Postgres-compatible) and returns `time.Time` natively.
- **`slices.Backward` yields copies (Go footgun)** — `for _, v := range slices.Backward(s)` binds `v` to a COPY of each element; `v++` mutates the copy and leaves `s` unchanged. This silently broke `nextKey()` (the exclusive-upper-bound helper behind EVERY Pebble prefix scan): the copy increment was discarded so the upper bound equalled the lower bound and all scans returned empty. When mutating elements in place, use direct index access (`for i := len(s) - 1; i >= 0; i-- { s[i]++ }`) or take an address explicitly (`for i := range slices.Backward(s) { s[i]++ }`). The `nextkey_test.go` regression test pins this. The auto-commit daemon reverted this fix TWICE (commit ancestry showed the broken `slices.Backward` version reappearing) — always diff the committed `nextKey` against the indexed form.

## Dependencies

| Category   | Packages                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Production | oklog/ulid/v2, go-branded-id, go-error-family, go-retry (retry/), go-idempotency (idempotency/), go-faster/yaml (catalog); go.opentelemetry.io/otel (otel, event, storage, middleware, prometheus); prometheus/client_golang (prometheus); golang.org/x/crypto (encryption); fxamacker/cbor/v2 (codec); maypok86/otter/v2 (kv, decider); failsafe-go/failsafe-go (middleware); modernc.org/sqlite (storage, stack/sqlite — pure-Go); github.com/jackc/pgx (storage, stack/postgres); github.com/go-sql-driver/mysql (stack/mysql — pure-Go); github.com/duckdb/duckdb-go (stack/duckdb — CGo, C++ engine); cockroachdb/pebble (storage/pebble) |
| Test-only  | onsi/ginkgo/v2, onsi/gomega, pgregory.net/rapid (event, encryption); testcontainers-go/modules/postgres (stack/postgres, storage, benchkit); testcontainers-go/modules/mysql (stack/mysql); go-snaps (eventtest, catalog, otel, codec — golden/snapshot testing)                                                                                                                                                                                                                                                                                                                                                                               |

**Coverage** (verified 2026-08-02 via `go test -tags "goexperiment.jsonv2" -cover`, workspace mode): core modules (decider 96.1%, storage/memory 96.9%, schema 89.9%, command 88.3%, event 88.2%, id 86.4%); mid-tier (snapshot 91.9%, metaengine 76.3%, dispatcher 81.5%, query 83.0%); newer modules (kv 71.9%, codec 70.2%). Bundle layer (stack presets, cache) 0–87% — presets emphasise the shared contract suite + happy paths. stack/postgres tests now run locally via testcontainers (was 0% when skipping without POSTGRES_TEST_DSN). Coverage drift is checked by `scripts/check-coverage.sh` (run via `nix run .#check-coverage`). See `docs/status/` for latest.

**Module Graph** (seven-tier model, see [ADR-0046](docs/adr/0046-seven-tier-model.md) and [FOUR-TIER-MODEL.md](docs/architecture-understanding/FOUR-TIER-MODEL.md)):

```
Tier 0 — Primitives: id/, dispatcher/, codec/, kv/, dedup/, retry/, flightrecorder/, metaengine/
Tier 1 — Core Domain: event/, command/, query/, scheduling/, metadata/
Tier 2 — Domain Utilities: schema/, snapshot/, projection/, idempotency/, deriver/
Tier 3 — Aggregation: decider/, graph/, scenario/, projectionhost/, listing/
Tier 4 — Infrastructure: storage/memory/, storage/, middleware/, signing/, encryption/, otel/, watermill/, transport/http/, transport/grpc/, storage/pebble/, storage/turso/, prometheus/, metaengine/projectionadapter/
Tier 5 — Composition: stack/, stack/memory/, stack/sqlite/, stack/duckdb/, stack/pebble/, stack/postgres/, stack/mysql/, stack/turso/
Tier 6 — Tooling & Examples: catalog/, integration/, stack/bench/, examples/, cmd/*
```

> Note: the old 7-layer system (pre-ADR-0046) was inaccurate — kv/ depends on codec/, command/
> depends on event/, and 40 of 58 modules depend on codec/. The four-tier model reflects reality.
>
> **metaengine/ is THE STRATEGIC FUTURE of this project** (possibly a future dedicated project).
> It is Tier 0 (Primitive), not Tier 3 (Aggregation) — intentional but surprising:
> the core planner has ZERO internal deps (stdlib + `database/sql` only), so by ADR-0046's
> dependency rule it is a leaf primitive. Conceptually it aggregates events into query
> projections, but tiering is dependency-based, not conceptual. The bridge to the rest of
> the system lives in `metaengine/projectionadapter/` (Tier 4), which depends on
> event/projection/projectionhost. The SQLite engine's tx-atomic MapUpdate, restart-safe
> multimap seq-seed, and cross-engine reify are documented in ADR-0066/0067/0068.
> ⚠️ **CANONICAL DESIGN DOCS** — read these before working on metaengine:
> [project-definition](docs/planning/meta-engine-project-definition.md),
> [design/vision](docs/planning/meta-engine-design.md),
> [assumptions & query-planning](docs/planning/meta-engine-assumptions-and-query-planning.md).

> **Saga pattern**: No dedicated saga module. Multi-step orchestration emerges from bus.SubscribeAll + command dispatch. See `example/taskmanager/` for a real architecture.

> **Historical details**: Session milestones, catalog architecture, and known issues in
> [`docs/sessions/SESSION_MILESTONES.md`](docs/sessions/SESSION_MILESTONES.md)
> and [`docs/planning/CATALOG_ARCHITECTURE.md`](docs/planning/CATALOG_ARCHITECTURE.md).

> **Schema evolution**: Upcaster and VersionedStore moved to `schema/` module. See `schema/` package.
> **Snapshot persistence**: Snapshot types moved to `snapshot/` module. See `snapshot/` package.
