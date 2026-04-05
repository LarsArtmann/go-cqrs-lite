# TODO List

**Updated:** 2026-04-05

Actionable items for the next 2–4 weeks.

## 🔴 HIGH Priority

- [ ] Add aggregate Repository interface
- [ ] Add integration test: full CQRS roundtrip (command → handler → event → store → bus → aggregate rebuild)
- [ ] Create example/user/ with full CQRS flow (aggregate.go, commands.go, events.go, handlers.go, main.go)

## 🟡 MEDIUM Priority

- [ ] Add middleware/logging.go
- [ ] Add middleware/recovery.go
- [ ] Add middleware/retry.go
- [ ] Add middleware/validation.go
- [ ] Add middleware/metrics.go
- [ ] Add benchmarks for ID operations and dispatcher throughput
- [ ] Add fuzzing for Parse functions
- [ ] Update README.md with xtypes usage
- [ ] Refactor With\* methods in event/event.go
- [ ] Add AppendBatch to Store
- [ ] Add snapshot store interface
- [ ] Add query/pagination.go
