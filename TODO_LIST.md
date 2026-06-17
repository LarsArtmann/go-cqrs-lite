# TODO List

**Updated:** 2026-06-16
**Version:** v2.3.0
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).

## Legend

- `[x]` = Done
- `[v3]` = Breaking change, deferred to next major (v3)
- `[v4]` = Breaking change, deferred to v4

---

## All Items Resolved

Every actionable item from the previous TODO list has been completed or found to be already done:

- [x] **Extract shared SQL helpers** — `sql.RunInTx` and `sql.IsDuplicateKeyError` in `storage/sql/`
- [x] **OTel tracing for pebble stores** — `pebble.SnapshotStore` and `pebble.CheckpointStore` have spans
- [x] **Docker build CI step** — `docker-build` job in ci.yml builds linux/amd64 + linux/arm64
- [x] **Replace directive CI check** — `scripts/check-replace-directives.sh` wired into GitHub Actions
- [x] **go-snaps golden tests for codec + otel** — Both modules already had golden tests using the project's own `testdata/golden/` pattern (the project doesn't use go-snaps; it uses `os.WriteFile`/`os.ReadFile` + `-update` flag)
- [x] **Playwright E2E tests** — Not applicable. `example/user` is a CLI demo with no HTTP server. `example/todo` has HTTP but already has comprehensive Go integration tests covering all endpoints (`integration_test.go`). Playwright would add Node.js infrastructure for zero new coverage.
- [x] **Reactive CommandBus and QueryBus test suites** — `command/reactive_test.go` and `query/reactive_test.go`
- [x] **Pebble integration tests** — `integration/pebble_test.go`: projection Runner replay+live, decider Repository + SnapshotStore
- [x] **Module layer budgets** — updated to actual direct dependency counts
- [x] **command.Compose / query.Compose** — re-exported `go-error-family.Compose`

---

## Open Items

### High impact

- [ ] **Schema registry** — JSON Schema validation middleware for events (ADR-0017)
- [ ] **Distributed checkpointing** — multi-instance projection coordination (ADR-0018)
- [ ] **Prometheus metrics exporter** — replace custom `MetricsRecorder` in `middleware/`
- [ ] **Structured logging middleware** — configurable `slog` levels for command/event/query processing
- [ ] **Distributed tracing propagation** — span context across module boundaries
- [ ] **PostgreSQL CI service container** — wire `storage/pg_integration_test.go` into GitHub Actions

### Recently completed

- [x] **Pebble KV Store adapter** (`pebble/`) — `NewKVStore()` wraps `*pebble.DB` as `kv.Store`, first consumer of the kv/ abstraction (ADR-0023)
- [x] **Reactive CommandBus and QueryBus** (`command/`, `query/`) — reactive extensions mirroring the event API

### Medium impact

- [ ] **Pebble coverage 85%+** — currently ~84%; target error branches in `helpers.go`, `serialization.go`
- [ ] **Pebble golden test** — deterministic CBOR envelope bytes for regression safety
- [ ] **MemorySnapshotStore golden test** — baseline for pebble snapshot comparison
- [ ] **Reactive bus documentation** — add usage examples to `command/doc.go`, `query/doc.go`, and `AGENTS.md`
- [ ] **Benchmark pebble vs SQL store** — `Save 100 events` comparison
- [ ] **cqrs-gen v2** — struct tag scanning code generator improvements
- [ ] **Built-in pprof endpoints** — profiling HTTP handler in `middleware/`

### Experimental / long-term

- [ ] **gRPC transport adapter** — new module for command/query dispatch over gRPC
- [ ] **NATS/Redis Stream adapter** — message broker integration
- [ ] **Streaming event reads** — `StreamLoader` without materializing full slice
- [ ] **jsonv2 codec experiment** — behind build tag
- [ ] **Arena allocation experiment** — behind build tag and Go experiment flag
- [ ] **WASM compilation target** — decider module for browser/edge
- [ ] **Documentation site** — Docusaurus/MkDocs/Hugo

---

## Deferred Breaking Changes

### v3 (Next Major — currently v2.3+)

- [v3] **Remove `io.Closer` from core interfaces** — ADR-0010 accepted. Affects `event.Store`, `snapshot.SnapshotStore`, `command.Store`.
- ~~Split `event.Store` into Writer/Reader/Deleter~~ — **ALREADY SATISFIED**: Sink/Source split exists, tombstones handle soft-delete
- [v3] **Add global `TransactionID` branded type** — Cross-aggregate consistency tracking.
- [v3] **Make event Core truly immutable** — Currently opts pointer is shallow-copied on Clone (payload/metadata are deep-copied).
- [v3] **Move HTTP code out of middleware** — SSE, healthcheck, metrics_http → transport/ module.
- [v3] **Fix `query.Handler` returns `any`** — Generic `TypedHandler[T]` returning `(T, error)` instead of `(any, error)`.

### v4

- [v4] **Split `catalog.Message` into Message + MessageMeta** — 17 fields → structured embedding. Changes exported struct literal construction.
- [v4] **Split `catalog.Service` into Service + ServiceMeta** — 16 fields → structured embedding.

---

_0 open actionable items + 8 deferred breaking changes. See [ROADMAP.md](ROADMAP.md) for long-term vision and sprint history._
