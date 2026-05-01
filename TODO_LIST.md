# TODO List

**Audited:** 2026-05-01 · **Session 17**
**Previous entries audited against actual codebase state — completed + stale items removed.**

## 🔴 HIGH Priority

- [ ] Fix `toDotAddress` number handling — "Get3DView" → "get.3.d.view" instead of "get.3d.view"
- [ ] Refactor `event.NewEvent` (66 lines → 2-3 functions) for function size compliance
- [ ] Fix `storage.Close()` closing caller-owned `*sql.DB` — ownership surprise

## 🟡 MEDIUM Priority

- [ ] Refactor `asyncapi.addMessage` (55 lines → 2 functions) for readability
- [ ] Add storage error path tests — BeginTx failure, version query error, insert error, commit error
- [ ] Add `scanEvents` error tests — parse ID failure, row scan failure, rows.Err()
- [ ] Add `Schema()` DDL test
- [ ] Add `Delete()` error path test
- [ ] Add SQL injection security test for SQLEventStore
- [ ] Add duplicate projection detection in `InMemoryRunner.Register`
- [ ] Add `UpcasterRegistry` cycle detection
- [ ] Document `InMemoryRunner` fail-fast behavior in godoc
- [ ] Document `MemoryBus.Publish` RLock behavior in godoc

## 🟢 LOW Priority

- [ ] Consider `Value()` returning text (26-char) instead of binary (16-byte) for SQL friendliness
- [ ] Add projection parallel processing (goroutine pool)
- [ ] Update `FEATURES.md` storage + upcaster status
- [ ] Update `AGENTS.md` storage coverage + stale issues
- [ ] Add Go doc `Example*` test functions for command, event, query, aggregate, id packages
- [ ] Add E2E throughput benchmarks (commands/sec, events/sec)
- [ ] Add fuzz tests for event creation, ID parsing, schema reflection, `DecodePayload`, upcaster chain
- [ ] Write getting-started guide / step-by-step tutorial

## 📐 PLANNED (No Code Exists)

- [ ] SQL SnapshotStore implementation (PostgreSQL-backed)
- [ ] SQL CheckpointStore implementation (PostgreSQL-backed)
- [ ] Outbox background publisher — goroutine that polls outbox and publishes to bus
- [ ] Watermill module — pub/sub adapter (Kafka, NATS, etc.)
- [ ] Saga / Process Manager — long-running process orchestration
- [ ] Tag `v0.1.0-alpha` releases for core, memory, catalog, middleware modules
