# TODO List

**Updated:** 2026-06-17
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).

## Legend

- `[ ]` = Open
- `[x]` = Done
- `[v3]` = Breaking change, deferred to next major (v3)
- `[v4]` = Breaking change, deferred to v4

---

## Open Items

### High impact

- [ ] **Schema registry** — JSON Schema validation middleware for events (ADR-0017). Uses `catalog/schema/` infrastructure.
- [ ] **Distributed checkpointing** — multi-instance projection coordination (ADR-0018). Large scope.
- [ ] **Prometheus metrics exporter** — OTel→Prometheus bridge to replace custom `MetricsRecorder` in `middleware/`.
- [ ] **Schema registry** placeholder — moved here from High impact (schema registry is now High impact above).

### Medium impact

- [ ] **Pebble coverage 85%+** — currently ~84%; target error branches in `helpers.go`, `serialization.go`.
- [ ] **Pebble CompactionFilter** — time-based TTL for automatic event expiry in the LSM tree.
- [ ] **Streaming event reads** — `EventIterator` interface for `event.Store` to avoid materializing full slices.
- [ ] **cqrs-gen v2** — struct tag scanning code generator improvements.

### Experimental / long-term

- [ ] **gRPC transport adapter** — new module for command/query dispatch over gRPC.
- [ ] **NATS/Redis Stream adapter** — message broker integration.
- [ ] **jsonv2 codec experiment** — behind build tag, pending Go stdlib stabilization.
- [ ] **Arena allocation experiment** — behind build tag and Go experiment flag.
- [ ] **WASM compilation target** — decider module for browser/edge.
- [ ] **Documentation site** — Docusaurus/MkDocs/Hugo.

---

## Recently Completed

- [x] **Projection replay→live dedup** — Reactive pipeline refactor: `SubscriberToObservable` adapts callback-based `Subscriber` to `ro.Observable[Event]`; `DistinctByEventIDWith(seen)` seeds dedup with replay IDs. Runner's `subscribeLive` now builds `live → DistinctByEventIDWith(replayIDs) → handler` pipeline. Zero duplicate processing at the replay→live boundary.
- [x] **OTel baggage → event enricher** — `middleware.OTelCorrelationEnricher` bridges OTel baggage correlation IDs into event metadata via `event.WithCustom`. Composes with `CommandCausalityEnricher` via `CompositeEnricher`. Placed in `middleware/` (Layer 5) which imports both `event/` and `otel/`.
- [x] **Ghost API cleanup** — Removed `EventSlice[T]` and `SeedFromEnv()` from `testutil/` (trivial wrappers with zero consumers).
- [x] **NewCQRSViews bug fix** — OTel `NewCQRSViews()` instrument name filter was `"cqrs."` (exact match, matched nothing). Fixed to `"cqrs.*"` (wildcard). MeterProvider integration test added.
- [x] **Layer violation fix** — All 3 pre-existing budget violations (`id`, `codec`, `pebble`) resolved by excluding test-only deps (gomega, ginkgo, rapid) from the production dep count.
- [x] **Watermill integration test** — Router integration test for `CorrelationIDMiddleware()` + `NewRetryMiddleware()` verifying retry behavior and correlation ID propagation.
- [x] **PostgreSQL CI service container** — wired into GitHub Actions.
- [x] **Pebble KV Store adapter** (`pebble/`) — `NewKVStore()` wraps `*pebble.DB` as `kv.Store`, first consumer of the kv/ abstraction (ADR-0023).
- [x] **Reactive CommandBus and QueryBus** (`command/`, `query/`) — reactive extensions mirroring the event API.
- [x] **Built-in pprof endpoints** (`middleware/`) — `ProfilingHandler()` and `RegisterProfiling()` for runtime profiling.
- [x] **Pebble benchmarks** (`pebble/`) — Save100, SaveLoad100, Save1, LoadEmpty for regression tracking.
- [x] **KV contract tests** (`pebble/`) — 10-test suite proving PebbleAdapter and MemStore semantic equivalence.
- [x] **PostgreSQL CI** — service container wired to storage integration tests.
- [x] **Codec fuzz fix** — CBOR duplicate map key type ambiguity handled gracefully.
- [x] **Module READMEs** — `kv/README.md` and `pebble/README.md`.

---

## Deferred Breaking Changes

### v3 (Next Major)

- [v3] **Remove `io.Closer` from core interfaces** — ADR-0010 accepted. Affects `event.Store`, `snapshot.SnapshotStore`, `command.Store`.
- [v3] **Add global `TransactionID` branded type** — Cross-aggregate consistency tracking.
- [v3] **Make event Core truly immutable** — Currently opts pointer is shallow-copied on Clone (payload/metadata are deep-copied).
- [v3] **Move HTTP code out of middleware** — SSE, healthcheck, metrics_http → transport/ module.
- [v3] **Fix `query.Handler` returns `any`** — Generic `TypedHandler[T]` returning `(T, error)` instead of `(any, error)`.

### v4

- [v4] **Split `catalog.Message` into Message + MessageMeta** — 17 fields → structured embedding. Changes exported struct literal construction.
- [v4] **Split `catalog.Service` into Service + ServiceMeta** — 16 fields → structured embedding.

---

_15 open actionable items + 7 deferred breaking changes. See [ROADMAP.md](ROADMAP.md) for long-term vision._
