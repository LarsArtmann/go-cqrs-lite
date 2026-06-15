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

## HIGH — Remaining CQRS Audit Trail Work

The CQRS audit trail feature (command/query persistence with journal support) is now **fully implemented** across memory, SQL, **and Pebble** backends. Pebble now has complete backend parity: EventStore + Journal + SnapshotStore + CheckpointStore, all sharing a single `*pebble.DB` via disjoint key prefixes.

**Completed in this session:**

- [x] Pebble SnapshotStore (`NewSnapshotStore`) — CBOR envelope, ignores older versions
- [x] Pebble CheckpointStore (`NewCheckpointStore`) — CBOR envelope, zero-value on miss
- [x] Pebble Journal + SeekableJournal (was already implemented)
- [x] SQL CommandJournal `ReadAll` + `ReadFrom` (was already implemented)
- [x] MemoryCommandBus tests — 14 tests covering pub/sub, middleware, closed-state, concurrency
- [x] Event causality tests — `WithCommandCausality` + `CommandCausalityEnricher` round-trip
- [x] SQLBackend goroutine-safe lazy-init (race condition fix)

Remaining items:

- [ ] **Extract shared SQL helpers** — `withTx` (~25 lines) and `isDuplicateKeyError` (~10 lines) are duplicated across `SQLCommandStore` and `SQLQueryStore`. Move to `storage/sql/` package.

---

## MEDIUM

- [ ] **`go-snaps` across remaining modules** — signing, middleware, storage, listing, watermill, pebble, turso, codec, otel, schema, snapshot, memory (some already have golden tests)
- [ ] **OTel tracing for pebble stores** — `pebble.SnapshotStore` and `pebble.CheckpointStore` have no spans. SQL stores trace every operation via `cqrsotel.StartSpan`. Add the same pattern for parity (requires adding `otel/v2` dependency to `pebble/go.mod`).
- [ ] **Docker build CI step** — linux/amd64 + linux/arm64 multi-arch build in GitHub Actions
- [ ] **Add `replace` directive CI check** — Script that verifies all modules pass `GOWORK=off go test` to catch the silent regression class

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

_5 open items + 8 deferred breaking changes. See [ROADMAP.md](ROADMAP.md) for long-term vision and sprint history._
