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

## Where to Find Things

[`SKILL.md`](SKILL.md) (symlink to `.agents/skills/go-cqrs-lite/SKILL.md`) is the canonical API reference for **all** agents — consumers AND contributors. Its `references/` contain verified, copy-paste recipes and module docs. This AGENTS.md covers internal contracts, procedures, and gotchas that only matter when working **inside** the repo.

| Topic                                                                                                                         | Reference                                                                          |
| ----------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Mental model, quickstart, decision matrix, conventions, cheat sheet                                                           | [`references/core.md`](.agents/skills/go-cqrs-lite/references/core.md)             |
| Composition recipes (ES setup, persistence, snapshots, signing, encryption, OTel, catalog, CBOR, metaengine, flight recorder) | [`references/recipes.md`](.agents/skills/go-cqrs-lite/references/recipes.md)       |
| Read models (projections, SQL views, CatchUpSubscriber, tier selection)                                                       | [`references/readmodels.md`](.agents/skills/go-cqrs-lite/references/readmodels.md) |
| Advanced patterns (tombstone, watermill, gRPC, projection host, scheduling, graph, SSE, flight recorder, scenario DSL)        | [`references/advanced.md`](.agents/skills/go-cqrs-lite/references/advanced.md)     |
| Per-module quick lookup                                                                                                       | [`references/modules.md`](.agents/skills/go-cqrs-lite/references/modules.md)       |
| Common pitfalls, error messages, debugging                                                                                    | [`references/faq.md`](.agents/skills/go-cqrs-lite/references/faq.md)               |

**Contributing to the skill:** edit the `.md` files under `.agents/skills/go-cqrs-lite/`, then verify:

```bash
cd cmd/doc-check && GOWORK=off go run -tags "goexperiment.jsonv2" . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md
```

## Quick Reference

| Item        | Value                                                                                                                                           |
| ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Language    | Go 1.26.4                                                                                                                                       |
| Build       | `nix run .#build`                                                                                                                               |
| Test        | `nix run .#test`                                                                                                                                |
| Lint        | `nix run .#lint`                                                                                                                                |
| Format      | `nix fmt`                                                                                                                                       |
| Dev shell   | `nix develop`                                                                                                                                   |
| Verify all  | `nix run .#verify` (build + vet + test + race + lint + doc-check)                                                                               |
| Int. PG     | `nix run .#integration-pg` (ephemeral, no Docker) or `nix run .#integration-pg-vm` (QEMU VM)                                                    |
| Int. MySQL  | `nix run .#integration-mysql-nspawn` (nspawn, ~15s, needs root + uid-range) or `nix run .#integration-mysql-vm` (QEMU VM, ~131s, always works)  |
| Int. All    | `nix run .#test-integration` or `nix run .#test-all-backends` (SQLite+Pebble+bbolt+DuckDB+PG+MySQL+Dgraph)                                      |
| Int. Dgraph | `nix run .#integration-dgraph` (ephemeral nixpkgs Dgraph, full dgraphengine suite; also a CI job)                                               |
| Int. Redis  | `nix run .#integration-redis` (ephemeral nixpkgs Redis; watermill broker suite: roundtrip, Nack redelivery, group exactly-once, 2 MiB payloads) |
| Load sweep  | `nix run .#load-sweep` (timing tests `-run 'Latency\|Timer\|Deadline'` under CPU soakers — run before `#verify` after touching timing paths)    |
| Verify CI   | `nix run .#verify-ci` (GOWORK=off per-module build+test — mirrors the CI matrix job)                                                            |
| Lint config | `nix run .#check-lint-config` (golangci config verify + depguard allow-list)                                                                    |
| Bench       | `nix run .#bench` (full sweep) · `./scripts/benchmark-regression.sh` (gate: median ns/op, 25% threshold — CI fails on breach)                   |
| CI          | GitHub Actions: ci.yml (Nix-based, build/vet/test/lint/race/coverage + GOWORK=off per-module)                                                   |

Multi-module Go workspace (`go.work`) with 82 `go.mod` files (incl. root). Verify: `find . -name go.mod -not -path './vendor/*' | wc -l`

Per-module isolation: `cd event && GOWORK=off go test ./... -count=1`

## Module Map

Compact reference — see [`references/modules.md`](.agents/skills/go-cqrs-lite/references/modules.md) for the full consumer-facing lookup.

| Module                                                         | Role                                                                                                   | Notes                                             |
| -------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------- |
| `event/`                                                       | EventSink/EventSource/Store/Bus/Journal, ImmutableEvent, NewEvent                                      | Core. `v4/eventtest/` nested module (see Gotchas) |
| `command/`                                                     | Dispatcher, Handler, Middleware, Bus, PersistedCommand, Store                                          |                                                   |
| `query/`                                                       | Dispatcher, Handler, PaginatedResult[T], PersistedQuery, Store                                         |                                                   |
| `decider/`                                                     | Decider[State], Repository[State], TypedDecider[State,Cmd]                                             | Pure-function style, singleflight, state cache    |
| `deriver/`                                                     | Event→command derivation (ADR-0040)                                                                    |                                                   |
| `commandlifecycle/`                                            | Command lifecycle as event streams: Recorder, Middleware (ADR-0117)                                    | Tier 2. Events + middleware                       |
| `commandlifecycle/projections/`                                | Pre-built metaengine projections: DLQ, retry count, failure log (ADR-0117)                             | Tier 3. Depends on metaengine                     |
| `id/`                                                          | Branded IDs: `id.Of[T]` = `cbid.ID[T, ulid.ULID]`                                                      |                                                   |
| `metadata/`                                                    | Tracing, CustomData[K]                                                                                 | Extracted from event/ — shared metadata types     |
| `record/`                                                      | Shared Record + CommonMetadata types (ADR-0111)                                                        | Zero deps. Structural base for Events + Commands  |
| `dispatcher/`                                                  | Generic Dispatcher[H, M] with LifecycleMixin                                                           |                                                   |
| `schema/`                                                      | Upcaster, VersionedStore, Validator with RegisterType[T]()                                             | Schema evolution                                  |
| `snapshot/`                                                    | Snapshot, SnapshotStore, SnapshotStrategy, EveryNEvents                                                |                                                   |
| `projection/`                                                  | Projection interface (consumer-side)                                                                   | Extracted from event/                             |
| `projectionhost/`                                              | Managed projection host: crash-restart, DLQ, checkpoint                                                | Reads SeekableJournal directly                    |
| `kv/`                                                          | Layer-0 KV: Store, TypedStore[T,K], Cache[T,K], ViewStore[V,K]                                         |                                                   |
| `graph/`                                                       | Graph projection tier: NodeRef, EdgeRef, MemoryDriver, GraphProjection (deprecated, removed in v5 — graphadapter keeps GraphDriver/GraphSink) | ADR-0033, ADR-0039                                |
| `dedup/`                                                       | Bounded dedup ring buffer (O(1) fixed-capacity)                                                        |                                                   |
| `listing/`                                                     | StreamListing, StreamStatus (`event.TombstoneStatus`), TombstonePolicy, StatusMiddleware               |                                                   |
| `scenario/`                                                    | Fluent BDD DSL: Given/When/Then for deciders + projections                                             |                                                   |
| `scheduling/`                                                  | Durable deadline timers (TimerStore, Scheduler)                                                        |                                                   |
| `idempotency/kvstore/` `idempotency/sqlstore/`                 | KV/SQL idempotency stores (over external `go-idempotency`)                                             | ADR-0065, ADR-0128                                |
| `signing/`                                                     | Event signing: HMAC-SHA256, Ed25519, multisig, middleware                                              |                                                   |
| `encryption/`                                                  | Payload encryption: XChaCha20-Poly1305, AES-256-GCM, codec wrapper                                     |                                                   |
| `middleware/`                                                  | Logging, Retry, Recovery, Validation, Idempotency, Metrics, OTel, Circuit Breaker, Flight Recorder     |                                                   |
| `otel/`                                                        | Shared OTel helpers — re-export instead of go.opentelemetry.io directly                                |                                                   |
| `prometheus/`                                                  | OTel→Prometheus metrics bridge                                                                         |                                                   |
| `transport/http/`                                              | **DEPRECATED** (ADR-0127, removal at v5): SSE delivery. Use go-sse or watermill/                       |                                                   |
| `transport/grpc/`                                              | **DEPRECATED** (ADR-0127, removal at v5): gRPC dispatch. Use watermill/ brokers                        |                                                   |
| `watermill/`                                                   | Watermill adapter: EventBus, CommandBus, CatchUpSubscriber                                             |                                                   |
| `storage/memory/`                                              | In-memory test impls (MemoryStore, etc. over generic `LogStore[T,ID]`)                                 |                                                   |
| `storage/`                                                     | SQLBackend facade, SQL stores, relational projections, views (view + relational tiers deprecated: removed in v5, ADR-0123) |                                                   |
| `storage/eventstore/`                                          | SQLEventStore, SQLSnapshotStore, SQLCheckpointStore                                                    |                                                   |
| `storage/readmodel/`                                           | SQLKVStore (kv.Store backed by SQL)                                                                    |                                                   |
| `storage/sql/`                                                 | Dialect, DBHandle, QueryEngine, RunInTx, IsDuplicateKeyError, ScanSlice, JournalReader[T], Inserter[T] | Shared SQL helpers                                |
| `storage/relational/`                                          | RelationalSchema, RelationalProjection, RelationalStore, ProjectionSink                                | Multi-table SQL, rollup counters                  |
| `storage/view/`                                                | SQLViewStore[V,K], ViewMapper, AutoMapper                                                              | Queryable columns                                 |
| `storage/migrations/`                                          | Embedded .sql DDL (postgres/sqlite/duckdb) via //go:embed                                              |                                                   |
| `storage/pebble/`                                              | PebbleDB: EventStore, SnapshotStore, KVAdapter                                                         | **CGo-free** (cockroachdb/pebble)                 |
| `storage/bbolt/`                                               | bbolt: EventStore, KV, Backend facade                                                                  | Pure Go, single-writer                            |
| `storage/backuptest/`                                          | Shared backup lifecycle test suite (Backend, Factory, RunFullLifecycle)                                | Test-only; imported by bbolt + pebble             |
| `storage/turso/`                                               | Turso connector, indexing advisor                                                                      |                                                   |
| `testutil/`                                                    | Shared test helpers (NewCmd, RaceEnabled)                                                              |                                                   |
| `testutil/pgtestcontainer/`                                    | Shared Postgres testcontainer helpers                                                                  |                                                   |
| `catalog/`                                                     | Registry, SchemaFromType[T](), AsyncAPI/D2/OpenAPI exporters                                           |                                                   |
| `integration/`                                                 | Cross-module tests                                                                                     |                                                   |
| `benchkit/`                                                    | Factory-driven benchmarking suite                                                                      |                                                   |
| `system/`                                                      | Deployer-driven composition root: DomainConfig + DeploymentConfig, AdapterCore[T]                      | The strategic composition layer (D6, D9, D11)     |
| `stack/`                                                       | Stack types + durability tiers (Strict/Normal/Relaxed). Bundle/Materialize/RunProjections deprecated (removed in v5, ADR-0123) |                                                   |
| `stack/memory/` `stack/sqlite/` `stack/pebble/` `stack/bbolt/` | Bundle presets (one-call infrastructure). Deprecated (removed in v5, ADR-0123)                         |                                                   |
| `stack/postgres/` `stack/mysql/` `stack/turso/`                | External-server bundle presets. Deprecated (removed in v5, ADR-0123)                                    |                                                   |
| `stack/duckdb/`                                                | DuckDB bundle preset. Deprecated (removed in v5, ADR-0123)                                             | **CGo required** (C++ engine)                     |
| `stack/bench/`                                                 | Cross-preset benchmark suite                                                                           |                                                   |
| `metaengine/`                                                  | Cost-based storage planner — **THE STRATEGIC FUTURE**                                                  | See Metaengine section below                      |
| `metaengine/*engine`                                           | Engine backends: sqlite, pebble, bbolt, duckdb, pg, mysql, badger, dgraph, turso, iroh                 | Each is a separate module (dep isolation)         |
| `metaengine/adttest/`                                          | Exported ADT test harness (RunMatrix, RunCapabilityConformance)                                        | Package INSIDE metaengine module, not a module    |
| `metaengine/enginetest/`                                       | Exported engine test harness                                                                           | Package INSIDE metaengine module, not a module    |
| `metaengine/bench/`                                            | Cross-engine benchmark module                                                                          | **CGo** (imports all engines)                     |
| `metaengine/projectionadapter/`                                | Wraps Store as projection.Projection                                                                   |                                                   |
| `metaengine/keycodec/`                                         | Key encoding for LSM-style backends                                                                    | Package INSIDE metaengine module, not a module    |
| `example/*`                                                    | 4 examples: taskmanager, getting-started, readme-quickstart, metaengine-quickstart                     | Usage demos, not deployments                      |
| `cmd/cqrs-gen/`                                                | Code generator: typed handler registration                                                             |                                                   |
| `cmd/cqrs-lint/`                                               | Domain-aware linter: 203 rules, 10 categories                                                          |                                                   |
| `cmd/cqrs-bench/`                                              | CLI benchmark tool                                                                                     |                                                   |
| `cmd/api-stability/`                                           | API surface checker (golden file)                                                                      |                                                   |
| `cmd/doc-check/`                                               | Doc checker: verifies Go import paths in markdown                                                      |                                                   |

## Internal Contracts

Non-obvious conventions that apply when editing code inside this repo. Consumer-facing conventions are in [`references/core.md`](.agents/skills/go-cqrs-lite/references/core.md) §3.

1. **Max 350 lines/file (CI-enforced), 30 lines/function.**
2. **Multi-module isolation** — Each module has its own `go.mod` with only needed deps.
3. **Dependency budgets** — Per-module direct PRODUCTION dep limits enforced by `nix run .#check-arch`. Test-only packages (gomega, ginkgo, rapid) are excluded. Adding production deps requires explicit budget review.
4. **OTel through otel/** — Modules import `otel/` re-exports instead of `go.opentelemetry.io` directly. OTel SDK is indirect in decider, storage, middleware go.mod files. The `otel/` module re-exports: `Int64Counter`, `AddOption`, `AddSpanEvent()`, `ServiceResourceAttributes()`, `CQRSHistogramBoundaries`, `NewCQRSViews()`, `CounterAddWithAttributes()`, `Setup()`, `WithStdoutExporter()`, `TextMapPropagator()`, `Version()`. Span names follow `{component}.{action}` — see `docs/SPAN_NAMING.md`.
5. **Zero-copy internal reads** — `PayloadReadOnly(evt)` bypasses `Payload()` clone for read-only paths (Event is a concrete type alias `= *ImmutableEvent`, no assertion needed). Used by signing, pebble, storage/sql, transport/http/sse. Internal-only `payloadForDecode()` and `encodingForCopy()` for same-package paths.
6. **Defensive clone on all public accessors** — `Payload()` returns `slices.Clone`, `Metadata()` returns `.Clone()`, `EventTypes()` returns `slices.Clone`, `MultiSignature.Get()` returns a copy, `WithCommandMetadata` clones on intake.
7. **Hot-path zero-allocation discipline** — Public API clones stay, but internal hot paths eliminate allocs via: lazy map init, pre-computed middleware chains (rebuild on `Use()`/`UsePublish()` only), cached SQL templates, pre-sized result slices, batch SQL inserts (multi-VALUES with SQLite 999-param chunking).
8. **Circuit breaker uses failsafe-go** — `middleware/circuit_breaker.go` wraps `failsafe-go/circuitbreaker`. Half-open semantics differ (limits trial executions to `SuccessThreshold` count). `decider/cache.go` uses `maypok86/otter/v2` TinyLFU.
9. **Load coalescing via singleflight** — `decider.Repository[State]` uses `singleflight.Group` to coalesce concurrent `Load` calls. Events are immutable, sharing is safe. Disable via `WithLoadCoalescing[State](false)`.
10. **Go experimental build tags** — Builds use `-tags "goexperiment.jsonv2"` enabling `encoding/json/v2`. CI and `nix run .#build` apply it automatically. Tag remains until Go graduates it (expected 1.27+).
11. **Deletion as domain events (ADR-0114, direction; partial implementation)** — Deletion SHOULD be expressed as a domain event type (e.g. `user.deleted`), not mutable metadata. Today: metaengine is fully type-based (`metaengine.Remove`); `stack.Materialize.OnTombstone/OnRebirth` are still metadata-triggered (`event.TombstoneMark` — branch on `evt.Type()` in `OnUpdate` for pure domain-event style); `listing.StatusMiddleware(deleteTypes, rebirthTypes)` bridges event types → status. `event.DetectTombstone`/`MarkTombstone` are Deprecated (removal v5). No `Delete` on Store. See ADR-0114 implementation-status addendum.
12. **Strong types** — No `any` as a value type in domain/business logic. Legitimate exceptions: JSON schema serialization (`catalog/`), `recover()` return value (`middleware/recovery.go`), `database/sql` interop. Generic type constraints (`[T any]`) are standard Go and always allowed.
13. **Error-wrapping helpers** — When `if err != nil { return WrapX(err, code, msg) }; return nil` appears 3+ times in a module, extract an unexported `wrapXOrOK(err, code, msg) error` (returns nil when err is nil). Keep per-module — see [ADR-0069](docs/adr/0069-error-wrapping-helpers.md). When modules share a dependency (e.g., encryption + signing → codec), push the helper into the shared module.
14. **Dedup helper patterns** — `storage/memory` uses `withWriteLock(code, msg, fn)` + `withReadLock[T](s, code, msg, fn)` + `wrapClosed(err, code, msg)`. `metaengine.DeferClose(c Closer)` replaces `defer func() { _ = x.Close() }()` across all engine modules (47 production + 17 test sites). The `.art-dupl-baseline.json` golden + `nix run .#check-duplication` gate enforce no-new-clones; run `art-dupl baseline . --threshold 3 --semantic` to update after a consolidation. A `//art-dupl:accept <reason>` comment on/above a clone region suppresses that group LIVE — annotate intentional clones instead of re-pinning the baseline; reserve baseline regen for structural shifts. Annotation is ITERATIVE: art-dupl reports one group per region pair, so suppressing the visible groups unmasks others behind them — re-run until "0 new clone groups" (2026-08-16: 12 groups took 4 rounds). The directive must sit directly on/above the region's FIRST line; placing it above the following function's doc comment does not suppress. The `#check-duplication` app refuses to run while `.art-dupl-baseline.json` has uncommitted changes (dirty-tree guard: re-pins must happen on a committed baseline).
15. **bbolt secondary index** — `storage/bbolt` uses a `cqrs_journal_idx` bucket (eventID → journalKey) as a secondary index for O(log N) Seek-based reads in `ReadStreamFrom`. Old databases without the index fall back to linear scan transparently. The `cqrs_journal` bucket holds the event journal; `cqrs_journal_idx` is the index. Both are created at DB init in `base.go`.
16. **Store wrapping goes through `event.DecorateStore`; journal wrapping through `event.DecorateJournal`** ([ADR-0126](docs/adr/0126-metadata-generic-store-transforms-wal-unification.md)) — Never hand-write Store/Journal wrapper structs: they drop optional capabilities (the old `encryptedStore` silently lost MultiSink; the old `VersionedSeekableJournal` lost StreamingJournal). Compose `SinkTransform`/`SourceTransform` instead (`encryption.EncryptSinkTransform`; `schema.UpcastSourceTransform` + `event.DecorateJournal` for journals). Deprecated shells (`schema.VersionedStore`, `schema.VersionedSeekableJournal`, `signing.Rejecting*`, `encryption.ErrInnerStoreNot*`, `metadata.CustomData`) exist for external consumers only — internal code uses the canonical forms; removal at v5.
17. **WAL cores are generic, policies injected** (ADR-0126) — `storage/memory.LogStore[T, ID]` (via `LogStoreConfig`), `storage/sql.Inserter[T]` (write-side counterpart of `JournalReader[T]`), and `system.AdapterCore[T]` own the shared mechanics. Divergent semantics (duplicate/not-found policy, missing-position replay, per-entity conflict sentinels) live in config funcs, not forked code. New stores embed the core instead of copying it.
18. **Import grouping is owned by treefmt, not gci** — `nix fmt` (treefmt goimports `-local github.com/larsartmann/go-cqrs-lite`) produces the 3-group layout; `gci` was REMOVED from `.golangci.yml` formatters (2026-08-16) because two tools fighting over the same import blocks re-broke 95+ files once. CI's `nix fmt --fail-on-change` gate enforces grouping mechanically.
19. **Engine `register.go` files are intentional clones** — each dep-isolated `metaengine/*engine` module needs its own `init()` calling `metaengine.RegisterDriver` (the database/sql pattern; Go cannot centrally register). Each carries a `//art-dupl:accept` directive; do NOT try to deduplicate across modules.
20. **Root CHANGELOG only; per-module CHANGELOGs forbidden** — nothing reads module-local changelogs and they drifted into describing shipped work as Unreleased (consolidated 2026-08-16; policy in CONTRIBUTING.md). `scripts/check-changelog-symbols.sh` (CI + `#verify`) gates every `pkg.Symbol` cited in the root `[Unreleased]` Added/Changed sections against the api-stability golden + repo source — kills the reverted-work fiction class. `cmd/doc-check` fails on ANY warning (zero-warning policy since 2026-08-15), including zero total references; `cmd/api-stability` fails loudly on unparseable modules instead of skipping them (a silently-shrinking golden is the corruption tell).

## Error Handling

- **Sentinel errors**: `errors.New` in `errors.go` files
- **Contextual errors**: `fmt.Errorf("failed to process %s: %w", name, err)`
- **Classified errors**: `errorfamily.NewRejection(...)`, `errorfamily.WrapConflict(...)` via [go-error-family](https://github.com/larsartmann/go-error-family) — imported directly, no facade
- **6-family taxonomy**: Rejection / Conflict / Transient / Infrastructure / Corruption / Orchestration
- **Direct import**: All modules import `errorfamily "github.com/larsartmann/go-error-family"` directly. The `event/` package retains type aliases (`event.Family`, `event.Error`) and family constants for backward compat, but error construction/classification/wrapping functions were removed. Use `go-error-family` directly.

## Codec Defaults (debugging encoding issues)

The default codec differs by layer. Events are self-describing (`evt.Encoding()` stamped on every event), so mixed JSON+CBOR event streams decode correctly via `DecodePayloadAuto`.

| Layer                           | Default codec | How to override                                                            |
| ------------------------------- | ------------- | -------------------------------------------------------------------------- |
| `stack.ReadModel`/`Materialize` | CBORCodec     | `stack.WithDefaultCodec(json)`                                             |
| `event.New()`                   | CBORCodec     | `event.DefaultCodec = codec.JSONCodec{}` or `event.WithCodec(c)` per-event |
| `kv.NewTypedStore()`            | CBORCodec     | `kv.WithTypedCodec(c)`                                                     |
| `snapshot.NewTypedStore()`      | CBORCodec     | positional arg: `NewTypedStore(store, c)`                                  |
| command typed store             | CBORCodec     | positional arg: `NewTypedCommandStore(store, c)`                           |
| query typed store               | CBORCodec     | positional arg: `NewTypedQueryStore(store, c)`                             |

Blind stores (kv/snapshot/command/query) are self-describing too via the ADR-0044 envelope: `WrapEncode`/`UnwrapDecode` stamp the codec on write and auto-detect it on read. Non-envelope data decodes via the store's configured codec with a JSON↔CBOR cross-retry, so pre-envelope rows written with either standard codec stay readable (ADR-0050 addendum; `decodeEnvelopeOrLegacy` per blind-store module).

One-call CBOR for both events AND read models: `bundle, _ := sqlite.New(dsn, stack.WithEventCodec(codec.CBORCodec{}))`

## Testing

- Table-driven tests preferred; BDD via Ginkgo v2 + Gomega for event/decider/query; fluent Given/When/Then via `scenario/`
- `t.Parallel()` for independent tests; core packages >80% coverage (most >90%)
- Golden tests use `go-snaps` (`snaps.MatchSnapshot`) — powered by `eventtest.AssertGolden`. Update: `UPDATE_SNAPS=true go test ./...`. Clean obsolete: `UPDATE_SNAPS=clean go test ./...`. Each module using golden tests has a `snaps_clean_test.go` with `TestMain` calling `snaps.Clean(m)`
- Modules without event dependency (otel, codec) use `go-snaps` directly via local `matchGolden(t, name, got)` helpers
- **Postgres integration tests** use `testcontainers-go` (postgres:16-alpine). Each test gets its own fresh database. `POSTGRES_TEST_DSN`/`DATABASE_URL` env var overrides.
- **Nix-based integration tests (no Docker)**: ephemeral PG/Redis/NATS/Dgraph via nixpkgs services. See Quick Reference for commands.
- **Relaunching the userspace MariaDB** (datadir persists across sessions at `/tmp/mariadb-cqrs`, port 33061): `nix build nixpkgs#mariadb -o /tmp/mariadb-cqrs-bin && /tmp/mariadb-cqrs-bin/bin/mariadbd --datadir=/tmp/mariadb-cqrs/data --socket=/tmp/mariadb-cqrs/mysql.sock --port=33061 --bind-address=127.0.0.1 --tmpdir=/tmp/mariadb-cqrs/tmpdir --silent-startup &`. Root logs in via the socket (unix_socket auth); reset the TCP user with `ALTER USER 'cqrs'@'%' IDENTIFIED BY 'cqrs';` then `MYSQL_TEST_DSN="cqrs:cqrs@tcp(127.0.0.1:33061)/cqrs_test"`. Probe with the server's own client (`mariadb -e "SELECT 1"`), not /dev/tcp.
- **Race-aware test thresholds**: `-race` inflates allocations/CPU 5-10x. Use `testutil.RaceEnabled` (or `enginetest.RaceEnabled` for metaengine modules) to pick relaxed bounds. Three lean-budget modules (`benchkit`, `transport/grpc`, `idempotency/kvstore`) keep local copies because adding testutil/enginetest would exceed dep budget. Always run affected test 3x with `-count=3 -race` after touching a threshold.
- **Soak test env vars**: `SOAK_SKIP_10M=1` skips 10M-event soak (~5s/25s-race). `SOAK_SKIP_DUCKDB=1` skips DuckDB AutoCRUD soak (~80s/100s-race). `SOAK_SKIP_BOLT=1` skips bbolt AutoCRUD soak (509-1145s under load; exported by the full `nix run .#verify` app, whose 8m per-package timeout it exceeds). The 50K-event `TestSoak_MemoryBounded` always runs as smoke.
- Coverage drift checked by `nix run .#check-coverage` (`scripts/check-coverage.sh`).

## Gotchas & Non-Obvious Behaviors

### Tooling & Build

- **Always `nix fmt` BEFORE placing `//nolint` directives** — golines (max-len: 120) reformats long lines and moves nolint comments to wrong positions. Keep nolint comments under ~40 chars.
- **Scoped formatting**: `nix fmt` runs treefmt on the whole repo. For a single module, use `gofumpt -w <path>` + `goimports -w <path>` directly.
- **gosec G115** (integer overflow): extract a helper that isolates the `uint64()`/`uint32()` call on a short single line.
- **When adding new dependencies**, add them to `.golangci.yml` depguard allow list at the same time.
- **SQL store helpers live in `storage/sql/`** — `RunInTx`, `IsDuplicateKeyError`, `CommitTx`, `ScanSlice`, `MarshalMetadata`. Don't duplicate transaction/duplicate-key logic in domain-specific store files.
- **`scanCommand` and `scanQuery` must unmarshal metadata** — both scan a `metadataJSON []byte` column. Use `json.Unmarshal` into `command.Metadata` / `query.Metadata` (standalone structs, NOT aliases for `event.Metadata`), then pass via `WithCommandMetadata` / `WithQueryMetadata`. Forgetting this causes silent metadata loss on SQL load.
- **NEVER commit code that doesn't compile** — Commit `b3931503` shipped `slices.Contains()` with zero arguments. If you bypass the pre-commit hook, run `go build ./...` manually.
- **Verification gate: `nix run .#verify`** — build + vet + test + race + lint + doc-check + doc-assertions. Run before tagging releases.
- **"Stale GREEN" anti-pattern** — every session that changes code, go.mod, or docs must run `nix run .#verify` (or at minimum `nix run .#verify-fast`) before claiming GREEN. A stale GREEN claim is worse than no claim.
- **Exit codes after pipes lie** — `cmd | tail -N; echo "EXIT=$?"` prints tail's exit code, not cmd's (and `PIPESTATUS[0]` can come back empty in this shell). A golangci run with 11 issues printed `EXIT=0` this way. Always capture gates as `cmd > /tmp/x.log 2>&1; echo $?` and grep the full log — never trust a `| tail` view or a post-pipe `$?`.
- **Never run integration suites concurrently with `#verify`** — benchkit's timing tests (Duration=10ms abort bound) and the per-package `-timeout=5m` are load-sensitive; a concurrent Dgraph soak produced 3 benchkit + 1 duckdb false failures (~25 min wasted). Run the full gate exclusively, nothing else heavy running.
- **Host toolchain may be older than go.work requires** — `go: go.work requires go >= 1.26.6 (running go 1.26.5)` means prefix the command with `GOTOOLCHAIN=auto` (the newer toolchain is already cached). The same error in gopls/golangci LSP diagnostics is noise; `go build -tags "goexperiment.jsonv2"` is authoritative.
- **Workspace-mode gates can fail on event alloc assertions while `../go-codec` has uncommitted changes** — alloc-count tests (`TestAllocs_NewEvent_*`) see the sibling go-codec working tree through `go.work`, so its untagged perf work shifts allocation counts (e.g. 3→2) and `#verify-fast` goes RED even though `GOWORK=off` (published go-codec v0.1.0) is GREEN. Diagnose by running the failing test both ways before suspecting your diff; fix is tagging go-codec + updating the expectations (tracked in TODO_LIST F46).
- **/tmp tmpfs fills up (48G, shared) — link jobs die with "no space left on device"** — set `GOTMPDIR=/mnt/buildcache/tmp` (plus `TMPDIR` for the linker) on workspace-wide `go test ./...` runs. Never delete /tmp/bigtest (unknown ownership); `trash-empty` reclaims ~6.6G when needed.
- **/mnt/buildcache went CORRUPTED 2026-08-16 (I/O errors on mkdir, 99% full)** — until repaired, redirect ALL caches to /tmp (28G free): `GOCACHE=/tmp/gocache-verify GOMODCACHE=/tmp/gomod-verify GOPATH=/tmp/gopath-verify`. NOTE: golangci-lint derives its cache from GOCACHE's PARENT dir — set `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache` too or every module fails with "failed to initialize build cache". `nix run .#build`/`#verify*` inherit the env fix; bare `go` also needs `GOTOOLCHAIN=auto`.
- **NEVER use `git checkout <commit> -- .`** — destructively overwrites the working tree. Use `git worktree add /tmp/work <commit>` instead.
- **Tool-shell false negatives on live servers** — `/dev/tcp/127.0.0.1/<port>` redirections silently fail in the tool shell (mvdan/sh has no /dev/tcp), reporting a healthy server as DOWN; probe with the server's own client instead (`mysqladmin ping` for MariaDB/MySQL). `kill` is not a builtin there — use `/run/current-system/sw/bin/kill <pid>`. A mysqld whose datadir was trashed keeps serving from unlinked inodes with all its old state; before diagnosing "corruption" or "mystery stale data", check `pgrep -a mysqld` + process start time vs datadir mtime, kill, and restart on a fresh datadir.

### system/v4 (composition root) — 2026-08-17 review outcomes

- **Full review done** — every file reviewed; 5×P1 + actionable P2/P3 fixed
  (commits a211ebcb2, 449e0e5a7, 42dfab5b0, each with regression tests);
  remaining design-level items routed in TODO_LIST "system/v4 Full-Code-Review
  Follow-Ups" + `docs/adr/2026-08-17_system-v4-review-proposals.md`.
- **Fixes are NOT in published system/v4.4.0** — system/go.mod carries 6 local
  replaces (metaengine + 4 engine adapters + watermill); local metaengine is
  ≥12 commits past published v4.11.0. Shipping the fixes requires a
  metaengine release, then system/v4.5.0 via the go-release flow.
- **Count projections collide by construction** — metaengine dispatches by
  input type; only one `Count()` projection per system until named dispatch
  lands (routed). Same input type across two `Get`s is fine (dispatch by name).
- **Fan-out buses are positional** — `MultiBus.Publishers()[0]` is always the
  local bus; fan-out buses are closed by `Close()` since 2026-08-17 but still
  have no name binding.
- **ACK keys are `rule:target`** — e.g.
  `volatile-source-of-truth:source-of-truth`, `durability-downgrade:<role>`.
  New scream rules must follow this convention and guard emission with
  `isAcknowledged`.
- **CachedEventStore invalidates on write** — `Save`/`AppendBatch` evict the
  stream key; wrap any new write path in the adapter the same way.
- **cqrs-lint C025 in system/ is a false-positive batch** — the flagged
  fmt.Errorf calls have no error operand to wrap (`WorkerState.LastError` is a
  string). Don't "fix" them into noise.

### Module & Dependency Management

- **`testModules` ↔ `lintModules` coupling** — `testModules` in `flake.nix` feeds BOTH `nix run .#test` AND `nix run .#lint`. Adding a new module requires adding its path to `testModules` — otherwise it's silently never built, tested, or linted. Meta-test: `TestEveryGoModDirIsInTestModules`.
- **API-surface changes require golden regen in the same edit** — Whenever you add/rename/remove an exported symbol, immediately regenerate: `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update`. Don't rely on the `#verify` gate — it catches it but wastes a 3-4 min cycle.
- **BuildFlow owns `.git/hooks/pre-commit`; two hook installers fight** — `buildflow precommit install` REGENERATES the hook file and wipes any manual edits; `nix run .#install-hooks` (copies `scripts/pre-commit.sh`) overwrites it the other way. Only one system can own the hook. TWO gates are wired BOTH in canonical `scripts/pre-commit.sh` AND appended post-BuildFlow in the installed hook — the api-stability golden check and the staged-`.go` syntax gate (`scripts/check-staged-go.sh`, blocks concurrent-session mid-write corruption). The appended blocks are wiped by the next `buildflow precommit install`; `scripts/install-hooks.sh` is the canonical restorer of BuildFlow-hook + both appended gates — re-run it after reinstalling.
- **Every directory with a `go.mod` must be in the api-stability modules list** — Meta-test `TestEveryGoModDirIsInModulesList` enforces this. Add new modules to `cmd/api-stability/main.go` `modules` slice in the same change.
- **Verify module version exists before requiring it** — Before adding `require .../module/v4 v4.x.y`, ALWAYS check the tag exists: `git tag -l '<module>/v4.x.y'`. Commit `169b5d42` shipped a broken go.mod because a tag was assumed but never created.
- **Private Go module auth (non-interactive fetch)** — devShell sets `GOWORK=off`, so `go mod download` fetches internal modules from VCS. `GOPRIVATE` uses HTTPS which fails without credentials. The flake `shellHook` exports `GIT_CONFIG_*` to redirect HTTPS → SSH. Symptom: `git ls-remote -q origin ... exit status 128` inside `~/go/pkg/mod/cache/vcs/`.
- **Auto-commit daemon can break the build** — Always run `go build -tags "goexperiment.jsonv2" ./...` after a daemon commit, not just `nix run .#build` (which uses `allPaths` — verify cmd/* modules actually compile).
- **Version-sequence breaks in published tags** — tags must be monotonically increasing in BOTH semver AND commit ancestry. Always tag with NEXT semver above all existing: `git tag -l '<module>/v4*' | sort -V | tail -1`.
- **Bash maps keyed by module use plain paths, never `/` spaces — and slashed keys MUST be quoted** — `check-module-layers.sh` LAYER/DEP_BUDGET keys MUST be literal module dirs (`LAYER["storage/memory"]`, `LAYER["cmd/cqrs-gen"]`). ROOT CAUSE of the recurring mangling (4 occurrences through 2026-08-15): the buildflow pre-commit hook runs `shfmt` on staged `.sh` files, and shfmt formats unquoted slashed subscripts (`LAYER[storage/memory]`) as arithmetic (`LAYER[storage / memory]`), silently disabling the budget/layer checks. Fix: quote every slashed map key (`LAYER["storage/memory"]=4`) — shfmt leaves quoted subscripts untouched and bash semantics are identical. The api-stability meta-tests (`normalizeLayerKey`) accept both forms. EXCEPTIONS deps also use plain `/` (`storage/memory`). Test-infra modules (`event/v4/eventtest`, `testutil`, `testutil/pgtestcontainer`) are exempt from layer ordering via `TEST_INFRA_MODULES`.
- **WithoutGlobalRegistration for isolated OTel providers** — `otel.Setup(cqrsotel.WithoutGlobalRegistration())` skips global calls. Use in tests and multi-service setups where global state would conflict.
- **Codec-critical methods require a consumer-pin sweep in the same wave** — 2026-08-17: `id.ActorID.MarshalBinary` (what keeps CBOR from silently zeroing the actor) shipped in `id/v4.5.0` while all 59 consumer modules still pinned `v4.4.0`; workspace greens (go.work resolves local source) hid that every `GOWORK=off`/published build silently lost the actor. Whenever a method that codecs/stores depend on lands in a tagged module, bump every consumer pin in the same wave (`grep -rl 'module/v4 vX' --include=go.mod .` + `go mod edit -require` + `go mod tidy`), and gate with a round-trip test run under `GOWORK=off`.
- **Unpublished-symbol sibling replaces (middleware/encryption/signing, 2026-08-17)** — these three modules reference event/metadata symbols that exist only on disk (`metadata.BrandedString`/`ActorString`, `event.ErrInnerStoreNot*`, `event.Rejecting*`); without `replace ... => ../event` + the cascading `=> ../metadata` (replaces do NOT cascade), standalone `GOWORK=off` builds fail with `undefined:`. Drop these replaces in the existing replace-strip sweep once metadata/event tags carrying the symbols are cut.

### Language & Library Footguns

- **Heap-measuring tests must NEVER be `t.Parallel()`** — `runtime.ReadMemStats()` is process-global: concurrent tests' live allocations land in the same snapshot and get misattributed as "leaks" (a 63MB phantom leak cost two sessions). Callers of `enginetest` soak runners must not parallelize; the contract is documented in `enginetest/soak.go`. Enforced mechanically by `scripts/check-heap-parallel.sh` (CI + same-file grep tripwire).
- **GOWORK env is positional** — running `go test` from INSIDE a package dir with the root `go.work` active (or absent) produces "directory prefix . does not contain modules listed in go.work" false failures. Per-module commands run with `GOWORK=off` from the module root; workspace commands run from the repo root. Diagnose the env before suspecting regressions. **Protocol benchmark baselines (evidence docs under `docs/benchmarks/`) run from the WORKSPACE ROOT** — per-module `GOWORK=off` runs can fail on sibling go directives (`requires go >= X` even with `GOTOOLCHAIN=auto`) AND their numbers are not comparable with workspace-mode numbers; mixing the two invalidates a baseline. Check `uptime`/load before contention micro-benchmarks and record it next to the numbers (a ±56% bench cell was traced to ambient load with 40+ user sessions).
- **Exact-equality alloc pins break across dependency graphs** — `go.work` resolves siblings (`../go-codec` etc.) to local checkouts that may be AHEAD of their published tags; an unpublished dependency-side improvement legitimately changes a count (2026-08-16: go-codec envelope fast-path made workspace `NewEvent` 2 allocs vs 3 published — exact `!= 3` pins went red ONLY in `#verify*` workspace runs, green in CI/`GOWORK=off`). Alloc guards on paths that touch dependency code must be upper-bound budgets (`if allocs > N`), not exact equality; first diagnose with both `GOWORK=off` (per-module) and workspace-root runs before assuming a regression.
- **gopls shows phantom errors after file splits** — stale snapshot reports `DuplicateMethod`/`already declared` at line numbers that no longer exist. **Fix: restart gopls** (`lsp_restart gopls`). gopls also runs WITHOUT `goexperiment.jsonv2`, so `encoding/json/v2` analysis is unreliable. Ignore "`X` is not in your go.mod file" floods until reindex completes. `go build -tags "goexperiment.jsonv2" ./...` and `go vet` are authoritative.
- **go-output color detection is env-aware — never reimplement it** — `output.ColorMode.ShouldColor()` honors `NO_COLOR`, `CI`/`GITHUB_ACTIONS`/`GITLAB_CI`/etc., and `FORCE_COLOR`/`GO_OUTPUT_FORCE_COLOR`. A hand-rolled check on `os.ModeCharDevice` (or `term.IsTerminal`) diverges from `table.Render` (which calls `cm.ShouldColor()` internally), producing colored findings text but colorless tables (or vice versa) within one run. Always delegate: `cm.ShouldColor()` for the decision, `output.ParseColorMode(strings.ToLower(s))` for `--color` flag parsing (the parser is case-sensitive). See `cmd/cqrs-lint/output.go`.
- **`slices.Backward` yields copies (Go footgun)** — `for _, v := range slices.Backward(s)` binds `v` to a COPY; `v++` leaves `s` unchanged. This silently broke `nextKey()` (behind EVERY Pebble prefix scan). When mutating in place, use direct index access (`for i := len(s) - 1; i >= 0; i-- { s[i]++ }`). The auto-commit daemon reverted this fix TWICE — always diff committed `nextKey` against the indexed form.
- **CBOR int→uint64 type drift (QUIC transport)** — CBOR encodes Go `int` as unsigned; on decode into `any`, fxamacker/cbor produces `uint64`, not `int`. The QUIC transport's `decodeOp` runs `normalizeAny()` to coerce. Tests must use `gomega.Equal` (not `BeEquivalentTo`). Any future CBOR transport encoding `any`-typed fields must apply the same normalization.
- **Dgraph 25.x `DeleteJson` requires explicit null predicates** — bare `{"uid": "0x1"}` does NOT delete predicates in 25.x. You MUST list each predicate as `null`: `{"uid": "0x1", "cqrs.map_collection": null, ...}`. `dgraphengine.MapDelete` handles this via upsert pattern.
- **MariaDB JSON SQL dialect (mysqlengine)** — MariaDB (which the nix integration envs actually run, via `pkgs.mariadb`) rejects `->` and `CAST(? AS JSON)` with Error 1064, and `JSON_EXTRACT` returns LONGTEXT so a bare ORDER BY text-sorts numbers ("10" < "2"). Universal filter form: `JSON_UNQUOTE(JSON_EXTRACT(value,'$.x')) = ?` with natively-bound scalars. Numeric-safe sort: dual key `CAST(JSON_EXTRACT(...) AS DECIMAL(65,10)), JSON_UNQUOTE(JSON_EXTRACT(...))` with cursor predicates matched to cursor Go type. Empirically verified against MySQL 8.4 + MariaDB 11.8 — see `metaengine/mysqlengine/dialect.go`.
- **modernc SQLite `file::memory:` is per-connection** — every pooled connection gets its own private, empty database. Any query overlapping another connection's lifetime (e.g. a timer scheduler polling while a tx holds another connection) lands on a schema-less DB → `no such table` flakes under parallel test load. `storage.OpenSQLiteInMemory` uses a unique named shared-cache DSN (`file:<random>?mode=memory&cache=shared`) so all pooled connections share one in-memory schema without pinning the pool to a single connection — allowing read concurrency. Regression test: `TestOpenSQLiteInMemory_SharedCacheDatabase`.
- **DuckDB/CGo isolation** — `stack/duckdb` is the ONLY module requiring CGo (`//go:build cgo` on `drivers.go`). DuckDB's `metadata` column is `BLOB` (not VARCHAR) to avoid byte-slice escaping. DuckDB dialect uses `$1` placeholders and returns `time.Time` natively.
- **eventtest nested module** — `event/v4/eventtest/` directory MUST match module path. `go mod tidy` in consumers emits warnings; run `go mod tidy -e` to suppress. See [ADR-0045](docs/adr/0045-eventtest-module-path-fix.md).
- **Replace directives do NOT cascade** — a dependency module's own `replace` lines are ignored when it is built as a dependency (only the MAIN module's replaces apply). Consequence: any module that replaces an engine (e.g. `metaengine/pebbleengine/v4 => ../metaengine/pebbleengine`) must ALSO replace `metaengine/v4 => ../metaengine` whenever the local engine uses unpublished metaengine symbols — otherwise GOWORK=off standalone builds resolve published metaengine and fail with `undefined:`. Same class bit `cmd/cqrs-bench` (via local `command` → unpublished `metadata.Metadata[K]`).
- **Local-path `replace` hygiene: relative sibling paths for UNPUBLISHED symbols only** — a `replace` pointing at a path outside the checkout (e.g. `/home/lars/projects/go-finding`) fails every CI Release build with "directory … does not exist" (commit `ceb88738b`). Published deps are NEVER replaced (just `require` the tag); unpublished sibling symbols get relative replaces (`=> ../event`), which `scripts/tag-release.sh` strips at cut time. Pre-tag sweep: `grep -rn "=> \.\./\|=> /" --include=go.mod .` and drop every replace whose target has a published tag.
- **Cross-engine dialect SQL similarity is baselined, not deduplicated** — engine modules are dep-isolated by design; pg/mysql `graph.go` (and `encodeNodeKey`) intentionally repeat structure with dialect-specific SQL. Resolution is `art-dupl baseline . --threshold 3 --semantic` (documented intentional similarity), NOT exporting shared SQL fragments. Real logic duplication (e.g. `decodeVector`/`topKNearest` → `metaengine.DecodeVectorJSON`/`TopKNearest`) IS deduplicated.

### Release

- **Release process** — See `CONTRIBUTING.md` → Release Process. Per-module annotated tags via `scripts/tag-release.sh`. Never use lightweight tags.

## Procedures

### Add a New Module

1. Create the directory with a `go.mod` (module path: `github.com/larsartmann/go-cqrs-lite/<name>/v4`)
2. Add the module path to `go.work`
3. Add the module path to `testModules` in `flake.nix` (feeds both `#test` and `#lint`)
4. Add the module path to `cmd/api-stability/main.go` `modules` slice
5. Run `go build -tags "goexperiment.jsonv2" ./...` to verify compilation
6. Run `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update` to generate golden
7. Run the meta-tests: `cd cmd/api-stability && GOWORK=off go test -tags "goexperiment.jsonv2" -run TestEvery .`

### Change an Exported Symbol

1. Make the code change
2. Immediately: `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update` (regenerate golden)
3. Update any affected skill references (`.agents/skills/go-cqrs-lite/references/*.md`)
4. Run `cd cmd/doc-check && GOWORK=off go run -tags "goexperiment.jsonv2" . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md`
5. Run `nix run .#verify` (or at minimum `nix run .#verify-fast`)

### Verify Before Release

```bash
nix run .#verify          # build + vet + test + race + lint + doc-check + doc-assertions
nix run .#vulncheck       # per-module standalone build (catches version-sequence breaks)
nix run .#check-arch      # dependency budget enforcement
nix run .#check-coverage  # coverage drift
nix run .#check-duplication  # no-new-clones gate
```

## Module Tiers

Seven-tier model — see [ADR-0046](docs/adr/0046-seven-tier-model.md) and [SEVEN-TIER-MODEL.md](docs/architecture-understanding/SEVEN-TIER-MODEL.md) for full mapping (78 modules across 7 tiers).

```
Tier 0 — Primitives: id/, dispatcher/, kv/, dedup/, record/ (codec, retry, flightrecorder extracted → external repos, ADR-0128)
Tier 1 — Core Domain: event/, command/, query/, scheduling/, metadata/
Tier 2 — Domain Utilities: schema/, snapshot/, projection/, idempotency/, deriver/, commandlifecycle/, idempotency/kvstore/, idempotency/sqlstore/
Tier 3 — Aggregation: decider/, graph/, scenario/, projectionhost/, listing/, metaengine/, commandlifecycle/projections/
Tier 4 — Infrastructure: storage/*, signing/, encryption/, otel/, prometheus/, middleware/, transport/*, watermill/,
                     testutil/, metaengine/*engine/, metaengine/projectionadapter/, metaengine/keycodec/, scheduling/sqlstore/
Tier 5 — Composition: stack/, stack/*presets/, system/
Tier 6 — Tooling & Examples: catalog/, integration/, benchkit/, cmd/*, example/*, event/v4/eventtest/
```

## Dependencies

Rules only — see each module's `go.mod` for the actual package list.

- **Production deps per module**: enforced by `nix run .#check-arch` (Layer 1 cross-module rules). Adding production deps requires budget review.
- **Test-only packages** (gomega, ginkgo, rapid, go-snaps, testcontainers) are excluded from dep budget counts.
- **CGo isolation**: Only `stack/duckdb` and `metaengine/duckdbengine` require CGo. Each is in its own module so consumers who don't import them never need a C compiler.
- **External extracted modules**: `go-codec`, `go-retry`, `go-idempotency`, `go-flightrecorder`. The in-repo re-export shims were deleted (ADR-0128); import the external paths directly. Workspace `use` block points at sibling checkouts.

## Metaengine

> **metaengine/ is THE STRATEGIC FUTURE of this project** (possibly a future dedicated project).

It is Tier 3 (Aggregation) — conceptually aggregates Records into query-optimized projections. The core planner depends only on `dedup/` and `record/` (both Tier 0). The bridge to the CQRS event-sourcing world lives in `metaengine/projectionadapter/` (Tier 4).

### Canonical Design Docs (read before working on metaengine)

### v2 Architecture (ADRs 0111-0117)

ES-native planner depends on the `Record` type ([ADR-0111](docs/adr/0111-record-type-extraction.md)) and understands typed records, not opaque `any` blobs. Tombstones are domain events ([ADR-0114](docs/adr/0114-tombstone-as-domain-event.md)), not mutable metadata. GraphBackend is deleted; `graph.GraphDriver` implements `metaengine.Engine` ([ADR-0113](docs/adr/0113-delete-graphbackend.md)). SQLite engine moves to `metaengine/sqliteengine/` ([ADR-0115](docs/adr/0115-sqlite-engine-extraction.md)). Auto-projection is layered ([ADR-0116](docs/adr/0116-layered-auto-projection.md)): 80% auto-generated from type inspection, 100% auto-routed. Command lifecycle (DLQ, retries) is event streams ([ADR-0117](docs/adr/0117-command-lifecycle-as-events.md)).

### Live Cost Measurement (dynamic NetworkRTT / per-op latency)

Remote engines (PG, MySQL, Dgraph, Turso) declare compile-time RTT priors. The live measurement system replaces those priors with runtime observations so the cost-based planner routes on honest data.

| Component                                       | Role                                                                                                                                                                                                                                                    |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Prober` / `TransactMeasurer`                   | Optional engine capability interfaces (Probe = RTT, MeasureTransact = per-read latency). PG: `SELECT 1` + `meta_map` point lookup. MySQL: same. Dgraph: healthcheck query + predicate index seek. Turso: `db.PingContext` via `sqliteengine.SetProber`. |
| `ProbeEngine(eng, opts...) *ProbeHandle`        | Starts background probe loop, installs live trackers via `Calibration`. Returns `ProbeHandle` with `Stop()` + `Failures()`. No-op for local engines (IsRemote guard).                                                                                   |
| `LatencyTracker`                                | Ring buffer + incremental EWMA + P50/P95/P99. Configurable window, alpha, stale-after. `Fresh()` is RTT-specific (read-only tracker doesn't set it).                                                                                                    |
| `Calibration.ApplyCalibration`                  | Layers live tracker EWMA into `Profile()` when fresh. Precedence: compile-time defaults → calibration priors → live measurement (highest).                                                                                                              |
| `Store.Replan(ctx)`                             | In-place re-plan picking up live latency shifts. Three-phase locking (assign → rules → swap). Increments plan version.                                                                                                                                  |
| `Store.CheckRouting(ctx)`                       | Execution-time re-scoring with hysteresis deadband. Differential: caches result until any engine's RTT changes. Emits `REPLAN-SUGGESTED` beyond threshold.                                                                                              |
| `Store.StartAutoReplan(ctx, interval)`          | Background loop calling CheckRouting + Replan when drift detected. Parent context controls lifecycle.                                                                                                                                                   |
| `WithRoutingHysteresis` / `WithRoutingMinDelta` | Plan options to tune the re-routing deadband (fractional + absolute floor).                                                                                                                                                                             |
| `GetEngineStats` / `Doctor` / `EXPLAIN`         | Surface live RTT, stale labels, and routing drift in diagnostics. Doctor includes `--- Routing ---` section with plan version, replan count, and drift summary.                                                                                         |
| `NsForRead` RTT amortization                    | Scan-pattern fallback costs subtract RTT to avoid overestimating batch reads (a 10K-row scan pays RTT once, not 10K times).                                                                                                                             |

Design doc: [`METAENGINE-LIVE-LATENCY-MODEL.md`](docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md). Recipe: `recipes.md` §2.11.

### User's vision statement (the north star)

This is the guiding intent for every metaengine decision. When design choices conflict, defer to this.

```text
"Developers declare ONLY Commands + Events + Queries and their relationships. We should be able to build
superb projections (materialized views) and developers never need to worry about anything else, while where
data lives is up to operators at DEPLOYMENT time."

"If I only give you SQLite, metaengine should deal with all query projections via SQLite. If there are graph
queries, it should warn about them being slow. At the same time I should be able to ONLY provide a GraphDB,
even for event logs."

"metaengine should be SMART enough to handle EVERYTHING so developers REALLY NEVER need to think about the
storage layer!"
```

**Decoded into three invariants every change must serve:**

1. **Developer declares, operator deploys** — The developer's surface area is Commands + Events + Queries + relationships ONLY. Storage selection is a deployment-time concern, never a code change.
2. **Graceful degradation, never failure** — Given one engine, metaengine serves every query on it. Unsupported/unsuited query shapes (e.g. graph traversal on SQLite) emit advisory diagnostics (WARN: slow), not errors.
3. **Zero storage-layer thinking** — A consumer who never reads a storage doc must still succeed. If a user must reach for a manual store, hand-rolled SQL, or a bespoke projection, that is a metaengine gap to close, not an acceptable workaround.

> **Saga pattern**: No dedicated saga module. Multi-step orchestration emerges from bus.SubscribeAll + command dispatch. See `example/taskmanager/`.

> **Historical details**: Early session notes are retired under [`docs/sessions/archive/`](docs/sessions/archive/) (superseded by `CHANGELOG.md`, `docs/status/`, and git log). Catalog architecture in [`docs/planning/CATALOG_ARCHITECTURE.md`](docs/planning/CATALOG_ARCHITECTURE.md).
