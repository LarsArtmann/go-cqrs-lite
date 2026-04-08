# TODO List

**Updated:** 2026-04-05

Actionable items for the next 2–4 weeks.

## 🔴 HIGH Priority

- [x] Add aggregate Repository interface
- [x] Add integration test: full CQRS roundtrip (command → handler → event → store → bus → aggregate rebuild)
- [x] Create example/user/ with full CQRS flow (aggregate.go, commands.go, events.go, handlers.go, main.go)

## 🟡 MEDIUM Priority

- [x] Add middleware/logging.go
- [x] Add middleware/recovery.go
- [x] Add middleware/retry.go
- [x] Add middleware/validation.go
- [x] Add middleware/metrics.go
- [x] Add benchmarks for ID operations and dispatcher throughput
- [x] Add fuzzing for Parse functions
- [x] Update README.md with xtypes usage
- [x] Refactor With\* methods in event/event.go
- [x] Add AppendBatch to Store
- [x] Add snapshot store interface
- [x] Add query/pagination.go

All TODO items complete. See [ROADMAP.md](ROADMAP.md) for future work.
