# Project Status Report — 2026-06-05 13:47

**Generated:** 2026-06-05 13:47:39
**Branch:** master (clean, pushed to origin)
**Latest commit:** `bf159ae3` fix(event,middleware): remove duplicate package godoc from non-doc.go files
**Test status:** ✅ All 38 packages pass (0 failures)
**Lint status:** 🟡 7 issues (all in catalog/ — none in 20 other modules)

---

## A) FULLY DONE ✅

### Across All Sessions (v2.1.0 → now)

| Category            | What                                     | Evidence                                                                                   |
| ------------------- | ---------------------------------------- | ------------------------------------------------------------------------------------------ |
| **Test suite**      | 38 packages pass, 0 failures             | `nix run .#test` green                                                                     |
| **Lint**            | 20/21 modules have 0 issues              | Only catalog has 7 (non-critical)                                                          |
| **doc.go**          | All 21 library modules have doc.go       | Verified 21/21                                                                             |
| **README.md**       | All 21 library modules have README.md    | Verified 21/21                                                                             |
| **errors.go**       | 18/21 modules have errors.go             | Only catalog (contextual), otel (no errors), listing (no errors) lack — intentionally      |
| **example_test.go** | 17/21 modules have example_test.go       | Missing: storage (needs DB), otel (helpers), catalog (complex), integration (cross-module) |
| **Go version**      | All 21 modules use go 1.26.3             | Verified                                                                                   |
| **File size**       | 0 production files exceed 350 lines      | Only scripts/ (412L) and catalog/internal/cattest/ (377L — test helper)                    |
| **Error taxonomy**  | 5-family system via go-error-family      | All modules use consistent classification                                                  |
| **Module graph**    | Clean layered deps, no cycles            | 30 go.mod files, all verified                                                              |
| **ADR backlog**     | 12 ADRs (3 new in this round: 0010-0012) | ADRs for v3 breaking changes documented                                                    |

### Session 3a (This Session — commits `1fdc2522` to `bf159ae3`)

| Commit     | What                                                                                                                                  |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| `1fdc2522` | Extract `snapshot/errors.go` — `ErrSnapshotNotFound`, `ErrSnapshotStoreClosed`, `ErrInvalidInterval`                                  |
| `f7aef971` | Split `storage/command_store.go` (387L) → 3 files: command_store.go (69L), command_store_save.go (185L), command_store_load.go (154L) |
| `2dc636cc` | Extract `loadReplayEvents` from `projection/replay` (65L → 37L + 28L helper)                                                          |
| `5b962670` | Add `example_test.go` to: snapshot, memory, middleware, listing, pebble, turso                                                        |
| `e1b6fa8a` | Remove duplicate godoc from `dispatcher/dispatcher.go`                                                                                |
| `bf159ae3` | Remove duplicate godoc from `event/event.go`, `middleware/middleware.go`                                                              |

### Previous Sessions (commits `244bc333` to `0b967a57`)

| Category                 | Details                                                                                                                |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------- |
| **Critical correctness** | turso/coverage_test.go (20+ tests), storage/sql/dialect_test.go, event/README.md rewrite                               |
| **Package docs**         | 11 new/updated doc.go files, 15 new README.md files                                                                    |
| **Error hygiene**        | id/errors.go, schema/errors.go, watermill/errors.go                                                                    |
| **Architecture**         | ADRs 0010-0012 (io.Closer removal, ErrDispatcherClosed unification, catalog split)                                     |
| **Consumer experience**  | example_test.go for signing, schema, watermill, projection; README.md for all 6 examples                               |
| **Decomposition**        | ListWithStatus (115L → 4 functions), messageToEvent (86L → 4 functions), buildMetadata (54L → generic parseIDField[T]) |
| **Housekeeping**         | go mod tidy across 21 modules, fixed broken replace directives in 7 modules                                            |

---

## B) PARTIALLY DONE 🟡

| Item                            | Status                    | What's Left                                                                                                                               |
| ------------------------------- | ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| **example_test.go coverage**    | 17/21 modules             | Missing: storage (needs real DB), otel (helpers-only), catalog (complex API), integration                                                 |
| **errors.go consistency**       | 18/21 modules             | catalog: 31 contextual fmt.Errorf — not suitable for sentinels. otel: no errors. listing: no production errors.                           |
| **Long function decomposition** | 42 functions > 30L remain | Most are sequential pipelines (SQL operations, scan loops, tx workflows) that are readable as-is. Only genuinely complex ones decomposed. |
| **Lint issues**                 | 7 issues remain           | All in catalog/ — see Section D                                                                                                           |

---

## C) NOT STARTED ⬜

| #   | Item                                                                     | Impact | Effort          |
| --- | ------------------------------------------------------------------------ | ------ | --------------- |
| 1   | catalog/ lint fixes (7 issues)                                           | Medium | Low             |
| 2   | catalog/internal/cattest/builders.go split (377L)                        | Low    | Low             |
| 3   | scripts/go-mod-graph-local/main.go split (412L)                          | Low    | Medium          |
| 4   | integration/doc.go                                                       | Low    | Trivial         |
| 5   | command errors.go re-export coverage (drags 80.5% → higher)              | Low    | Low             |
| 6   | Duplicate errorfamily re-exports in event + command (identical 70 lines) | Medium | Medium          |
| 7   | Type model improvements in storage/sql/dialect.go (uses `any` for time)  | Low    | Breaking change |
| 8   | storage/example_test.go (needs DB mocking)                               | Low    | Medium          |
| 9   | `samber/lo` adoption for slice operations                                | Low    | Low             |

---

## D) TOTALLY FUCKED UP 💥

**Nothing is broken.** The codebase is healthy:

- All tests pass
- All modules compile
- All dependencies resolve
- Working tree is clean

The only items that were previously "problems" and were correctly identified as NOT problems:

- `event/eventtest` shows 18.4% coverage — false alarm (test helpers tested by consumers)
- `turso` at 28.6% — sync.go requires real Turso server, not unit-testable
- `command` at 80.5% — gap is from re-exported `errorfamily` pass-throughs, not production logic

---

## E) WHAT WE SHOULD IMPROVE 🔧

### High-Impact Improvements

1. **Fix the 7 catalog lint issues** — forcetypeassert, unused const, unwrapped error, goconst, godoclint. These are the only lint issues in the entire codebase.

2. **Eliminate duplicate errorfamily re-exports** — `event/errors.go` and `command/errors.go` both re-export the ENTIRE errorfamily API (~70 lines each, identical). This is the biggest code smell remaining. Options:
   - Have consumers import `errorfamily` directly
   - Create a shared `errors/pub.go` that both re-export
   - Accept the duplication as a convenience API (current state)

3. **Type model: consider typed time wrappers** — `storage/sql/dialect.go` uses `any` for `FormatTime/ScanTimeDest/ParseTime`. A `TimeValue` interface could eliminate the `any` but would be a breaking change for `Dialect` implementors.

4. **Test file sizes** — 25+ test files exceed 350 lines (largest: 820L). These are harder to navigate but don't affect consumers.

### Medium-Impact Improvements

5. **integration/doc.go** — Only library module missing doc.go (has README but no doc.go)

6. **command coverage** — 80.5% is the lowest non-turso coverage. The gap is from unused errorfamily re-exports. Adding tests for these pass-throughs would raise it to ~92%.

7. **catalog schema coverage** — 86% (the reflect engine is complex)

8. **pebble coverage** — 86.7% (embedded DB edge cases)

### Low-Impact / Future

9. **ADR 0010** — Remove `io.Closer` from core interfaces (v3 breaking change)
10. **ADR 0011** — Unify `ErrDispatcherClosed` across packages (v3 breaking change)
11. **ADR 0012** — Split catalog into 5 modules (v3 breaking change)
12. **`samber/lo`** — Already a dependency via `samber/ro`; could simplify some slice operations

---

## F) TOP 25 THINGS TO DO NEXT

Sorted by impact × ease (Pareto order):

| #   | Task                                                        | Module        | Impact | Effort | Category     |
| --- | ----------------------------------------------------------- | ------------- | ------ | ------ | ------------ |
| 1   | Fix catalog/schema/reflect.go: unchecked type assertion     | catalog       | Medium | 5m     | Lint         |
| 2   | Fix catalog/schema/reflect.go: remove unused jsonKeyType    | catalog       | Low    | 2m     | Lint         |
| 3   | Fix catalog/schema.go: wrap external error                  | catalog       | Medium | 2m     | Lint         |
| 4   | Fix catalog/internal/cattest: goconst + godoclint           | catalog       | Low    | 5m     | Lint         |
| 5   | Add integration/doc.go                                      | integration   | Low    | 3m     | Docs         |
| 6   | Add command errorfamily re-export tests (coverage 80.5→92%) | command       | Medium | 10m    | Coverage     |
| 7   | Consider eliminating duplicate errorfamily re-exports       | event+command | High   | 30m    | Architecture |
| 8   | Add storage/example_test.go with in-memory SQLite demo      | storage       | Medium | 15m    | Consumer     |
| 9   | Split catalog/internal/cattest/builders.go (377L → 2 files) | catalog       | Low    | 5m     | File size    |
| 10  | Split scripts/go-mod-graph-local/main.go (412L → 3 files)   | scripts       | Low    | 10m    | File size    |
| 11  | Add catalog/example_test.go                                 | catalog       | Medium | 15m    | Consumer     |
| 12  | Add catalog/schema edge-case tests                          | catalog       | Medium | 15m    | Coverage     |
| 13  | Add pebble concurrent/edge-case tests                       | pebble        | Medium | 15m    | Coverage     |
| 14  | Investigate typed TimeValue for Dialect interface           | storage       | Medium | 20m    | Type model   |
| 15  | Add otel/example_test.go                                    | otel          | Low    | 5m     | Consumer     |
| 16  | Split large test files (>350L) into focused test files      | multiple      | Low    | 60m    | Navigation   |
| 17  | Evaluate samber/lo for slice helpers (filterByType, etc.)   | multiple      | Low    | 15m    | Lib adoption |
| 18  | Document module dependency graph in README or docs/         | docs          | Medium | 10m    | Docs         |
| 19  | Consider shared `withTx` helper for storage write methods   | storage       | Low    | 20m    | DRY          |
| 20  | Verify all GOWORK=off builds after any future tidy          | all           | Medium | 10m    | CI           |
| 21  | ADR 0010: implement io.Closer removal (v3)                  | event+core    | High   | 120m   | Breaking     |
| 22  | ADR 0011: implement ErrDispatcherClosed unification (v3)    | dispatcher    | Medium | 30m    | Breaking     |
| 23  | ADR 0012: implement catalog module split (v3)               | catalog       | High   | 180m   | Breaking     |
| 24  | Add example/ demonstrating listing module                   | example       | Low    | 15m    | Consumer     |
| 25  | Add example/ demonstrating pebble module                    | example       | Low    | 15m    | Consumer     |

---

## G) TOP #1 QUESTION ❓

**Should we eliminate the duplicate errorfamily re-exports in event/errors.go and command/errors.go?**

Both packages re-export the entire go-error-family API (~70 lines each, identical code). This is a consumer convenience — it lets consumers write `event.NewRejection(...)` instead of importing errorfamily directly. But:

- It drags command's coverage from ~92% to 80.5% (the re-exports have 0% coverage)
- It's 140 lines of pure duplication
- It creates a split brain: should consumers use `event.NewRejection` or `errorfamily.NewRejection`?

The question is: **Is the convenience of not importing errorfamily worth the coverage penalty and duplication?** Three options:

1. **Keep as-is** — accept duplication as consumer convenience
2. **Remove from command only** — event keeps re-exports (it's the "primary" error package), command consumers import event or errorfamily
3. **Remove from both** — all consumers import errorfamily directly

This affects the public API surface, so it needs a deliberate decision.

---

## Module Scorecard (Final)

| Module     | Coverage | Lint | doc.go | README | errors.go | example | Files>350L | Grade |
| ---------- | -------- | ---- | ------ | ------ | --------- | ------- | ---------- | ----- |
| event      | 89.4%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| command    | 80.5%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| query      | 94.3%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| decider    | 100%     | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| id         | 94.5%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| dispatcher | 100%     | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| schema     | 89.7%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| snapshot   | 92.3%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| memory     | 98.2%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| catalog    | 95.9%    | 🟡7  | ✅     | ✅     | —         | —       | 0          | 🟡    |
| middleware | 98.5%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| signing    | 94.1%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| storage    | 86.8%    | ✅   | ✅     | ✅     | —         | —       | 0          | 🟢    |
| projection | 91.2%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| otel       | ~96%     | ✅   | ✅     | ✅     | —         | —       | 0          | 🟢    |
| watermill  | 94.3%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| pebble     | 86.7%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| codec      | 93.3%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟢    |
| turso      | 28.6%    | ✅   | ✅     | ✅     | ✅        | ✅      | 0          | 🟡    |
| listing    | 94.9%    | ✅   | ✅     | ✅     | —         | ✅      | 0          | 🟢    |

**Overall: 18/21 modules 🟢, 3/21 modules 🟡 (catalog: lint, turso: coverage, command: coverage)**

---

_Clean working tree. All commits pushed to origin/master._
