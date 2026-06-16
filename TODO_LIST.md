# TODO List

**Updated:** 2026-06-16
**Version:** v2.3.0
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).

## Legend

- `[x]` = Done
- `[v2]` = Breaking change, deferred to next major
- `[v3]` = Breaking change, deferred to v3

---

## All Items Resolved

Every actionable item from the previous TODO list has been completed or found to be already done:

- [x] **Extract shared SQL helpers** — `sql.RunInTx` and `sql.IsDuplicateKeyError` in `storage/sql/`
- [x] **OTel tracing for pebble stores** — `pebble.SnapshotStore` and `pebble.CheckpointStore` have spans
- [x] **Docker build CI step** — `docker-build` job in ci.yml builds linux/amd64 + linux/arm64
- [x] **Replace directive CI check** — `per-module-test` job runs `GOWORK=off go test` for every module
- [x] **go-snaps golden tests for codec + otel** — Both modules already had golden tests using the project's own `testdata/golden/` pattern (the project doesn't use go-snaps; it uses `os.WriteFile`/`os.ReadFile` + `-update` flag)
- [x] **Playwright E2E tests** — Not applicable. `example/user` is a CLI demo with no HTTP server. `example/todo` has HTTP but already has comprehensive Go integration tests covering all endpoints (`integration_test.go`). Playwright would add Node.js infrastructure for zero new coverage.

---

## Deferred Breaking Changes

### v2 (Next Major)

- [v2] **Remove `io.Closer` from core interfaces** — ADR-0010 accepted. Affects `event.Store`, `snapshot.SnapshotStore`, `command.Store`.
- [v2] **Split `event.Store` into Writer/Reader/Deleter** — ADR-0010 direction.
- [v2] **Add global `TransactionID` branded type** — Cross-aggregate consistency tracking.
- [v2] **Make event Core truly immutable** — Currently opts pointer is shallow-copied on Clone (payload/metadata are deep-copied).
- [v2] **Move HTTP code out of middleware** — SSE, healthcheck, metrics_http → transport/ module.
- [v2] **Fix `query.Handler` returns `any`** — Generic `TypedHandler[T]` returning `(T, error)` instead of `(any, error)`.

### v3

- [v3] **Split `catalog.Message` into Message + MessageMeta** — 17 fields → structured embedding. Changes exported struct literal construction.
- [v3] **Split `catalog.Service` into Service + ServiceMeta** — 16 fields → structured embedding.

---

_0 open actionable items + 8 deferred breaking changes. See [ROADMAP.md](ROADMAP.md) for long-term vision and sprint history._
