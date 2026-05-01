# TODO List

**Audited:** 2026-05-01 · **Session 24**
**All items verified against codebase — completed items removed.**

## 🟡 MEDIUM Priority

- [ ] Add projection parallel processing (goroutine pool for handler dispatch)

## 🟢 LOW Priority

- [ ] Consider `Value()` returning text (26-char) instead of binary (16-byte) for SQL friendliness
- [ ] Watermill module — pub/sub adapter (Kafka, NATS, etc.)

## 📐 PLANNED (No Code Exists)

- [ ] Outbox background publisher — goroutine that polls outbox and publishes to bus
- [ ] Saga / Process Manager — long-running process orchestration (design doc exists at `docs/planning/SAGA_DESIGN.md`)
- [ ] Tag `v0.1.0-alpha` releases for core, memory, catalog, middleware modules
