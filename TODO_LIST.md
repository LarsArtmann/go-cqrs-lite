# TODO List

**Updated:** 2026-06-19
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).

## Legend

- `[ ]` = Open
- `[x]` = Done
- `[v3]` = Breaking change, deferred to next major (v3)
- `[v4]` = Breaking change, deferred to v4

---

## Open Items

### High impact

- [ ] **Distributed checkpointing — implementation** — ADR-0018 accepted, `LeaderElection` interface in `projection/`. Consumers implement coordination (Redis, etcd, k8s). Library provides `AlwaysLeader` default. Future: `DistributedRunner` wrapper with pluggable lock.

### Medium impact

- [ ] **Postgres LISTEN/NOTIFY event bus** — Deferred to v2.8.0 (ADR-0027). All v2.7.0 presets use an in-memory bus (single-process). A distributed bus must solve the 8KB NOTIFY limit (notify-with-reference + listener re-fetch), listener lifecycle, and real-Postgres testing. Consumers can already supply any `event.Bus` via `stack.WithBus`.

- [ ] **Streaming event reads — store implementation** — `EventIterator`, `StreamingSource`, `StreamingJournal` interfaces added to `event/`. `SliceIterator` adapter exists. SQL and Pebble stores should implement streaming variants for large aggregates.
- [ ] **cqrs-gen v3 — event handler generation** — Struct tag scanning added (`cqrs:"command:CreateUser"`). Event handler generation (`-type=event`) still needed.

### Experimental / long-term

- [ ] **gRPC transport adapter** — ADR-0025 accepted. Separate `transport/grpc/` module with protobuf dispatch.
- [ ] **NATS/Redis Stream adapter** — ADR-0025 accepted. Separate `transport/nats/` and `transport/redis/` modules.
- [ ] **jsonv2 codec experiment** — `codec/jsonv2_experiment.go` exists behind `goexperiment.jsonv2` build tag (ADR-0026). Pending Go stdlib stabilization.
- [ ] **Arena allocation experiment** — `event/arena_experiment.go` exists behind `goexperiment.arenas` build tag (ADR-0026). Pending Go arena API stabilization.
- [ ] **WASM compilation** — 6 of 7 core modules compile to `GOOS=js GOARCH=wasm` (id, codec, dispatcher, event, command, query). `decider/` blocked by OTel SDK `os/user` dependency.
- [ ] **Documentation site** — MkDocs Material scaffold created (`mkdocs.yml`). Content population and GitHub Pages deployment pending.

---

## Recently Completed

- [x] **Bundle composition layer** (`stack/v2`) — v2.7.0. `Bundle` with ISP-honest fields + pointer-dedup Close; repository/read-model helpers as top-level generic functions.
- [x] **Bundle presets** (`stack/{memory,sqlite,pebble,postgres}/v2`) — one-call wiring of every store + bus + read-model backend.
- [x] **Typed read-model store + cache** (`readmodel/v2`, `readmodel/cache/v2`) — `Store[T,K]` over `kv.Store`; Otter-backed `CachedStore[T,K]`.
- [x] **Typed stores** (`snapshot`, `command`, `query`) — `TypedSnapshot[State]`, `TypedCommandStore[P]`, `TypedQueryStore[P]`.
- [x] **Persistent read models for SQL presets** (`storage/v2`) — `SQLKVStore` over `cqrs_kv`; SQLite + Postgres presets now keep read models across restarts.
- [x] **Postgres preset tests in CI** — env-var mismatch fixed (`POSTGRES_TEST_DSN` now set in the postgres-integration job).
- [x] **Zero lint violations** — 0 across all 34 modules; `nix run .#lint` now resilient (reports all failures).
- [x] **Pebble DeleteEventsBefore removed** — Events are immutable truth. Removed the retention function, its test, and all doc references. Automatic event deletion is a footgun in an event sourcing library.
- [x] **Pebble coverage 85%+** — From 84.6% to 86.6% via targeted error-branch tests.
- [x] **Schema registry validator** (`schema/`) — `Validator` with `RegisterType[T]()`, `RegisterTypeWithValidator[T]()`, strict/lenient modes, custom codec support. ADR-0017 accepted.
- [x] **Prometheus metrics exporter** (`prometheus/`) — New module wrapping OTel Prometheus exporter. `Setup()`, `WithRegistry()`, `MustSetup()`. Full test suite.
- [x] **cqrs-gen v2 struct tag scanning** (`cmd/cqrs-gen/`) — Supports `cqrs:"command:CreateUser"` struct tags in addition to comment markers. Comment markers take precedence.
- [x] **LeaderElection interface** (`projection/`) — `LeaderElection` interface + `AlwaysLeader` default implementation for ADR-0018.
- [x] **Streaming event interfaces** (`event/`) — `EventIterator`, `StreamingSource`, `StreamingJournal`, `SliceIterator` adapter.
- [x] **Bounded dedup** (`event/`, `projection/`) — `DistinctByEventIDBounded(cap)` with FIFO ring eviction. `WithDedupCapacity(n)` Runner option for bounded memory in 24/7 projections.
- [x] **gopls false positives suppressed** — `.vscode/settings.json` disables `mod_tidy` analyzer. ADR-0007 updated with root cause.
- [x] **Transport adapter strategy** (ADR-0025) — Each transport (gRPC, NATS, Redis) gets a separate module. Core stays dependency-free.
- [x] **Experimental features policy** (ADR-0026) — Build-tag-gated experiments documented. jsonv2 and arena stubs created.
- [x] **WASM verification binary** (`wasm/main.go`) — Confirms 6/7 core modules compile to WASM.
- [x] **Documentation site scaffold** (`mkdocs.yml`, `docs/site/`) — MkDocs Material theme configured.
- [x] **Projection replay→live dedup** — Reactive pipeline refactor: `SubscriberToObservable` adapts callback-based `Subscriber` to `ro.Observable[Event]`; `DistinctByEventIDWith(seen)` seeds dedup with replay IDs.
- [x] **OTel baggage → event enricher** — `middleware.OTelCorrelationEnricher` bridges OTel baggage correlation IDs into event metadata via `event.WithCustom`.
- [x] **Ghost API cleanup** — Removed `EventSlice[T]` and `SeedFromEnv()` from `testutil/`.
- [x] **NewCQRSViews bug fix** — OTel instrument name filter fixed from `"cqrs."` to `"cqrs.*"`.
- [x] **Layer violation fix** — All 3 pre-existing budget violations resolved.
- [x] **Watermill integration test** — Router integration test for `CorrelationIDMiddleware()` + `NewRetryMiddleware()`.
- [x] **PostgreSQL CI service container** — wired into GitHub Actions.
- [x] **Pebble KV Store adapter** (`pebble/`) — `NewKVStore()` wraps `*pebble.DB` as `kv.Store`.
- [x] **Reactive CommandBus and QueryBus** (`command/`, `query/`).
- [x] **Built-in pprof endpoints** (`middleware/`).
- [x] **Pebble benchmarks** (`pebble/`).
- [x] **KV contract tests** (`pebble/`).
- [x] **Codec fuzz fix** — CBOR duplicate map key type ambiguity handled.
- [x] **Module READMEs** — `kv/README.md` and `pebble/README.md`.

---

## Deferred Breaking Changes

### v3 (Next Major)

- [v3] **Remove `io.Closer` from core interfaces** — ADR-0010 accepted. Affects `event.Store`, `snapshot.SnapshotStore`, `command.Store`.
- [v3] **Add global `TransactionID` branded type** — Cross-aggregate consistency tracking.
- [v3] **Make event Core truly immutable** — Currently opts pointer is shallow-copied on Clone.
- [v3] **Move HTTP code out of middleware** — SSE, healthcheck, metrics_http → transport/ module.
- [v3] **Fix `query.Handler` returns `any`** — Generic `TypedHandler[T]` returning `(T, error)`.

### v4

- [v4] **Split `catalog.Message` into Message + MessageMeta** — 17 fields → structured embedding.
- [v4] **Split `catalog.Service` into Service + ServiceMeta** — 16 fields → structured embedding.

---

_8 open actionable items + 7 deferred breaking changes. See [ROADMAP.md](ROADMAP.md) for long-term vision._
