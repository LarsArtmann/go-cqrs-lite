# TODO List

**Audited:** 2026-05-01
**Previous entries audited against actual codebase state — completed items removed.**

## 🔴 HIGH Priority

- [ ] Fix `storage/event_store.go` metadata silently discarded — INSERT passes `nil` for metadata column, losing all correlation IDs, user IDs, custom metadata
- [ ] Fix `storage/event_store.go` codec field unused — `WithStoreCodec` option exists but codec is never called, payloads stored/returned as raw bytes
- [ ] Fix `storage/event_store.go` `nowFunc` field unused — dead code (field set but never called)
- [ ] Add storage module tests — currently 0% coverage, no unit/integration/benchmark tests
- [ ] Fix `catalog/asyncapi/exporter.go` component message key collision — command/event sharing same MessageID overwrites each other in `Components.Messages` and `Components.Schemas`

## 🟡 MEDIUM Priority

- [ ] Fix `UpcasterRegistry.Upcast()` — uses `>=` comparison instead of `==` for version matching, may re-upcast already-current events
- [ ] Fix `toDotAddress` number handling — "Get3DView" → "get.3.d.view" instead of "get.3d.view"
- [ ] Refactor `event.NewEvent` (66 lines → 2-3 functions) for function size compliance
- [ ] Refactor `asyncapi.addMessage` (55 lines → 2 functions) for readability
- [ ] Add SQL SnapshotStore implementation (PostgreSQL-backed)
- [ ] Add SQL CheckpointStore implementation (PostgreSQL-backed)
- [ ] Add outbox background publisher — goroutine that polls outbox and publishes to bus
- [ ] Add lifecycle/Close to `storage/event_store.go` — no graceful shutdown or connection release
- [ ] Add security test for SQL injection in SQLEventStore

## 🟢 LOW Priority

- [ ] Document `MemoryBus.Publish` RLock behavior in godoc
- [ ] Consider `Value()` returning text (26-char) instead of binary (16-byte) for SQL friendliness
- [ ] Add saga/process manager design doc
- [ ] Write getting-started guide / step-by-step tutorial
- [ ] Add fuzz tests for event creation, ID parsing, schema reflection, `DecodePayload`, upcaster chain
- [ ] Add Go doc `Example*` test functions for command, event, query, aggregate, id packages
- [ ] Add E2E throughput benchmarks (commands/sec, events/sec)
- [ ] Add projection parallel processing (goroutine pool)
- [ ] Add `UpcasterRegistry` cycle detection
- [ ] Tag `v0.1.0-alpha` releases for core, memory, catalog, middleware modules

## 📐 PLANNED (No Code Exists)

- [ ] Watermill module — pub/sub adapter (Kafka, NATS, etc.)
- [ ] Saga / Process Manager — long-running process orchestration
- [ ] Tagged releases — semantic versioning and Go module publishing (all modules at v0.0.0)
