# TODO List

**Updated:** 2026-06-16
**Version:** v2.3.0
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).

## Legend

- `[ ]` = Open, actionable
- `[x]` = Done
- `[v2]` = Breaking change, deferred to next major
- `[v3]` = Breaking change, deferred to v3

---

## All Open Items

### Done This Session

- [x] **Extract shared SQL helpers** — `sql.RunInTx` and `sql.IsDuplicateKeyError` moved to `storage/sql/` package
- [x] **OTel tracing for pebble stores** — `pebble.SnapshotStore` and `pebble.CheckpointStore` now have spans
- [x] **Docker build CI step** — `docker-build` job in ci.yml builds for linux/amd64 + linux/arm64
- [x] **Replace directive CI check** — `per-module-test` job runs `GOWORK=off go test` for every module
- [x] **SQL helpers dedup** — `withTx` and `isDuplicateKeyError` extracted to `sql` package

### Remaining

- [ ] **`go-snaps` golden tests for `codec` and `otel`** — Last 2 modules without snapshot tests. All others have at least 1 golden test file.
- [ ] **Playwright E2E tests for `example/user/`** — Health endpoint + command→event→query flow. Requires Node.js + browser testing infrastructure.

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

_2 open items + 8 deferred breaking changes. See [ROADMAP.md](ROADMAP.md) for long-term vision and sprint history._
