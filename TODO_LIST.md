# TODO List

**Updated:** 2026-06-15
**Version:** v2.3.0
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).

## Legend

- `[ ]` = Open, actionable
- `[v2]` = Breaking change, deferred to next major
- `[v3]` = Breaking change, deferred to v3
- `[BLOCKED]` = Requires external action

---

## HIGH — CQRS Audit Trail (Active Sprint)

Symmetric persistence for commands and queries, matching the existing event journal pattern. Interfaces and memory implementations are done; tests and SQL backends remain.

- [ ] **Tests for `MemoryCommandStore` journal methods** — `ReadAll` ordering, `ReadFrom` position-based pagination, closed-store behavior, empty-store edge cases (`memory/command_store.go`)
- [ ] **Tests for `query/store.go`** — `NewPersistedQuery` validation (empty type rejection), `Payload()` defensive copy, `Metadata()` clone (`query/store_test.go`)
- [ ] **Tests for `MemoryQueryStore`** — `SaveQuery`, `LoadQueries` (after-time filter), `ReadAllQueries`, `ReadQueriesFrom` (position-based) (`memory/query_store_test.go`)
- [ ] **Add query module store sentinel errors** — `ErrQueryStoreClosed`, `ErrQueryNotFound`, `ErrDuplicateQuery` (command module has these; query parity gap)
- [ ] **Update `query/doc.go`** — Document `PersistedQuery`, `QuerySink/Source/Store`, `QueryJournal`, `SeekableQueryJournal` with usage examples
- [ ] **Add `SQLCommandStore` journal support** — `ReadAll`, `ReadFrom` methods for SQL-backed command persistence
- [ ] **Add `SQLQueryStore`** — SQL backend for query persistence (parity with `SQLCommandStore`)
- [ ] **Add `SQLBackend.QueryStore()`** — Facade method on SQL backend

---

## MEDIUM

- [ ] **`query.BasicQuery` has no metadata** — Unlike `BasicCommand`, queries carry no correlation/tracing context. Makes distributed tracing through the query path inconsistent with command/event.
- [ ] **`go-snaps` across remaining modules** — signing, middleware, storage, listing, watermill, pebble, turso, codec, otel, schema, snapshot, memory (some already have golden tests)
- [ ] **Docker build CI step** — linux/amd64 + linux/arm64 multi-arch build in GitHub Actions

---

## LOW

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

_15 open items + 8 deferred breaking changes. See [ROADMAP.md](ROADMAP.md) for long-term vision and sprint history._
