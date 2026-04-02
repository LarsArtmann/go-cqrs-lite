# TODO List

**Updated:** 2026-04-02

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
- [ ] Refactor With* methods in event/event.go
- [ ] Add AppendBatch to Store
- [ ] Add snapshot store interface
- [ ] Add query/pagination.go

## 🟢 LOW Priority

- [ ] Re-run buildflow --semantic --fix
- [ ] Update .golangci.yml
- [ ] Document testing approach in AGENTS.md
- [ ] Create architecture docs
- [ ] Create CONTRIBUTING.md
- [ ] Create CODE_OF_CONDUCT.md
- [ ] Add GoDoc package examples
- [ ] Add coverage tracking
- [ ] Add error assertion tests

## ✅ Completed (2026-04-02)

- [x] Create .github/workflows/test.yml
- [x] Create .github/workflows/lint.yml
- [x] Fix CI Go version matrix (→ 1.26)
- [x] Write tests for internal/dispatcher/ (0%→100%)
- [x] Improve aggregate/ tests (64%→100%)
- [x] Improve pkg/id/ tests (48%→88%) + add Equal/Compare/Or/Reset/BinaryMarshaler/TextMarshaler/GoString/Format
- [x] Improve xtypes/ tests (53%→95.6%)
- [x] Improve event/ tests (75%→92.8%)
- [x] Fix dispatcher closed-check bug (CheckClosed(nil) → CheckClosed(ErrHandlerNotFound))
- [x] Fix JSON null handling for zero-value IDs
- [x] Add catalog system (catalog/, catalog/asyncapi/, catalog/eventcatalog/, catalog/yaml/)
- [x] Extract Close() pattern to shared utility (internal/dispatcher LifecycleMixin)
- [x] Extract Use() middleware pattern (internal/dispatcher MiddlewareChain)
- [x] Update CHANGELOG.md
- [x] Add t.Parallel() to all test functions
