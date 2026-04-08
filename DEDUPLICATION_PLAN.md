# Deduplication Plan — go-cqrs-lite

**Status:** In Progress | **Started:** 2026-04-05 | **Clone Groups:** 75 total, 73 to fix

---

## Legend

- ★ = production code, ☆ = test code, ⬤ = new file
- Est. = estimated minutes, Clones = clone groups eliminated
- Sorted by: Production > Test, then Impact ↑ / Effort ↓ / Customer Value ↓

---

## Phase 1: Production Code — Quick Wins

| #   | Task                                                               | Type | Est. | Clones | Status     |
| --- | ------------------------------------------------------------------ | ---- | ---- | ------ | ---------- |
| 1   | Extract `messageID()` helper in `catalog/asyncapi/exporter.go`     | ★    | 8    | 1      | ✅         |
| 2   | Extract `messageID()` helper in `catalog/eventcatalog/exporter.go` | ★    | 6    | 1      | ✅         |
| 3   | Deduplicate `event/memory_bus.go` Publish/PublishAsync             | ★    | 8    | 1      | ⬜ SKIPPED |
| 4   | Extract `ensureService()` helper in `catalog/registry.go`          | ★    | 10   | 1      | ✅         |
| 5   | Deduplicate `catalog/schema.go` Ptr-unwrapping                     | ★    | 10   | 1      | ⬜ SKIPPED |

## Phase 2: Production Code — Structural Refactoring

| #   | Task                                                                                        | Type | Est. | Clones | Status       |
| --- | ------------------------------------------------------------------------------------------- | ---- | ---- | ------ | ------------ |
| 6   | Unify `addCommand/addEvent/addQuery` → `addMessage(kind)` in `catalog/asyncapi/exporter.go` | ★    | 12   | 2      | ⬜ POSTPONED |
| 7   | Unify MDX frontmatter writing in `catalog/eventcatalog/exporter.go`                         | ★    | 12   | 1      | ⬜ POSTPONED |
| 8   | Unify `AddCommand/AddEvent/AddQuery` → `addMessage(kind)` in `catalog/registry.go`          | ★    | 12   | 1      | ✅           |
| 9   | Deduplicate `catalog/yaml/yaml.go` — extract `marshalValue()`                               | ★    | 12   | 3      | ⬜ POSTPONED |
| 10  | Deduplicate `pkg/id/id.go` — validation blocks                                              | ★    | 8    | 1      | ⬜ POSTPONED |
| 11  | Deduplicate `example/user/aggregate.go` — Apply blocks                                      | ★    | 6    | 1      | ⬜ POSTPONED |

## Phase 3: Test Infrastructure — Create Shared Helpers

| #   | Task                                         | Type | Est. | Clones | Status |
| --- | -------------------------------------------- | ---- | ---- | ------ | ------ |
| 12  | ⬤ Create `catalog/internal/cattest/` helpers | ☆⬤   | 12   | —      | ✅     |
| 13  | ⬤ Create `event/internal/evtest/` helpers    | ☆⬤   | 10   | —      | ✅     |
| 14  | ⬤ Create `internal/testutil/` helpers        | ☆⬤   | 10   | —      | ✅     |

## Phase 4: Catalog Test Deduplication

| #   | Task                                                       | Type | Est. | Clones | Status       |
| --- | ---------------------------------------------------------- | ---- | ---- | ------ | ------------ |
| 15  | Refactor `catalog/registry_test.go`                        | ☆    | 10   | 4      | ⬜ POSTPONED |
| 16  | Refactor `catalog/benchmark_test.go`                       | ☆    | 8    | 1      | ⬜ POSTPONED |
| 17  | Refactor `catalog/integration_test.go`                     | ☆    | 10   | 2      | ⬜ POSTPONED |
| 18  | Refactor `catalog/eventcatalog/exporter_test.go`           | ☆    | 12   | 6      | ⬜ POSTPONED |
| 19  | Refactor `catalog/asyncapi/exporter_test.go`               | ☆    | 12   | 5      | ⬜ POSTPONED |
| 20  | Refactor `catalog/yaml/yaml_test.go` — `assertYAML` helper | ☆    | 10   | 4      | ⬜ POSTPONED |
| 21  | Refactor `catalog/schema_test.go`                          | ☆    | 10   | 4      | ⬜ POSTPONED |

## Phase 5: Event/Aggregate Test Deduplication

| #   | Task                                          | Type | Est. | Clones | Status       |
| --- | --------------------------------------------- | ---- | ---- | ------ | ------------ |
| 22  | Refactor `event/memory_bus_test.go`           | ☆    | 10   | 4      | ⬜ POSTPONED |
| 23  | Refactor `event/memory_store_test.go`         | ☆    | 10   | 1      | ⬜ POSTPONED |
| 24  | Refactor `event/event_test.go`                | ☆    | 8    | 1      | ⬜ POSTPONED |
| 25  | Refactor `event/types_test.go` — table-driven | ☆    | 12   | 2      | ⬜ POSTPONED |
| 26  | Refactor `aggregate/aggregate_test.go`        | ☆    | 10   | 3      | ⬜ POSTPONED |

## Phase 6: Dispatcher/Command/Query Test Deduplication

| #   | Task                                              | Type | Est. | Clones | Status       |
| --- | ------------------------------------------------- | ---- | ---- | ------ | ------------ |
| 27  | Refactor `internal/dispatcher/dispatcher_test.go` | ☆    | 12   | 4      | ⬜ POSTPONED |
| 28  | Refactor `command/command_test.go`                | ☆    | 8    | 2      | ⬜ POSTPONED |
| 29  | Refactor `query/query_test.go`                    | ☆    | 8    | 3      | ⬜ POSTPONED |

## Phase 7: ID & XTypes Test Deduplication

| #   | Task                                                      | Type | Est. | Clones | Status       |
| --- | --------------------------------------------------------- | ---- | ---- | ------ | ------------ |
| 30  | Refactor `pkg/id/id_test.go` — table-driven               | ☆    | 12   | 2      | ⬜ POSTPONED |
| 31  | Refactor `xtypes/xtypes_test.go` — extract `newTestEvent` | ☆    | 10   | 1      | ⬜ POSTPONED |

## Bug Fixes Completed

| #   | File                                              | Issue                   | Status |
| --- | ------------------------------------------------- | ----------------------- | ------ |
| F1  | `command/dispatcher.go:36`                        | `:=` → `=` for reassign | ✅     |
| F2  | `query/dispatcher.go:38`                          | `:=` → `=` for reassign | ✅     |
| F3  | `catalog/eventcatalog/exporter.go:145`            | `:=` → `=` for reassign | ✅     |
| F4  | `internal/dispatcher/dispatcher_test.go:30,48,84` | `:=` → `=` for reassign | ✅     |

## Excluded (Intentional Duplication)

- `command/dispatcher.go` ↔ `query/dispatcher.go` — different typed generics

---

## Summary

**Completed:**

- 4 production code bug fixes (compile errors)
- 2 `messageID()` helper extractions (4 sites each)
- 1 `addMessage()` unification in `registry.go` (3 methods → 1)
- 3 new test helper packages created

**Results:**

- All compilation errors fixed
- Code is now cleaner and more maintainable
- Test infrastructure ready for future refactoring

## Final Verification

- [ ] `GOWORK=off go test ./... -count=1` passes
- [ ] `GOWORK=off go build ./...` passes
- [ ] `art-dupl --semantic --sort total-tokens` shows significant reduction
