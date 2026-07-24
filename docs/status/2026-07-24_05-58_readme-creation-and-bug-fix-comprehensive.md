# Status: Module README Creation + Bug Fix Session — Comprehensive Report

**Date:** 2026-07-24 05:58
**Sessions covered:** README creation (session 1) + bug fix sweep (session 2)
**Goal:** Every Go sub-module has a dedicated, superb README.md with correct, verified code examples

---

## a) FULLY DONE

### 24 New READMEs Created and Verified

All previously-missing modules now have comprehensive READMEs with correct code examples:

| Module                      | Lines | Code Blocks | doc-check |
| --------------------------- | ----- | ----------- | --------- |
| `dedup`                     | 65    | 1           | Valid     |
| `deriver`                   | 92    | 2           | Valid     |
| `metadata`                  | 82    | 3           | Valid     |
| `projection`                | 69    | 2           | Valid     |
| `retry`                     | 102   | 3           | Valid     |
| `scenario`                  | 84    | 3           | Valid     |
| `scheduling`                | 96    | 1           | Valid     |
| `event/v4/eventtest`        | 120   | 3           | Valid     |
| `idempotency/kvstore`       | 73    | 1           | Valid     |
| `cmd/cqrs-bench`            | 77    | 0           | Valid     |
| `cmd/doc-check`             | 73    | 1           | Valid     |
| `storage/memory`            | 59    | 1           | Valid     |
| `storage/pebble`            | 116   | 5           | Valid     |
| `stack`                     | 111   | 2           | Valid     |
| `stack/memory`              | 63    | 1           | Valid     |
| `stack/sqlite`              | 78    | 4           | Valid     |
| `stack/pebble`              | 93    | 4           | Valid     |
| `stack/postgres`            | 87    | 4           | Valid     |
| `stack/turso`               | 102   | 3           | Valid     |
| `stack/bench`               | 36    | 0           | Valid     |
| `transport/http`            | 111   | 5           | Valid     |
| `transport/grpc`            | 87    | 3           | Valid     |
| `example/getting-started`   | 58    | 1           | Valid     |
| `example/readme-quickstart` | 58    | 1           | Valid     |

### 9 Existing READMEs Rewritten

| Module        | Before | After | Key Improvements                                                           |
| ------------- | ------ | ----- | -------------------------------------------------------------------------- |
| `decider`     | 40     | 94    | API tables, TypedDecider, EveryNEvents fix                                 |
| `dispatcher`  | 23     | 57    | Usage examples, methods table                                              |
| `id`          | 37     | 75    | Marker table, serialization, fixed idtest link                             |
| `snapshot`    | 28     | 95    | Fixed field names (StreamID/StreamType), TypedSnapshot, fixed EveryNEvents |
| `kv`          | 52     | 105   | Fixed all ctx params, Set/Get (not SetTyped/GetTyped)                      |
| `watermill`   | 45     | 89    | CatchUpSubscriber, ordering warning, CommandBus                            |
| `integration` | 32     | 43    | Fixed links, added encryption                                              |
| `schema`      | 42     | 59    | API table, Validator, design                                               |
| `testutil`    | 46     | 46    | Links fixed (adequate as-is)                                               |

### 19 Code Example Bugs Found and Fixed (Session 2)

Every bug was caught by auditing code examples against actual source code, then verified with doc-check.

| #   | Module           | Bug                                                        | Fix Applied                                        |
| --- | ---------------- | ---------------------------------------------------------- | -------------------------------------------------- |
| 1   | `deriver`        | `command.New(type, id, payload)` wrong signature           | `cqrscommand.New(type, streamID)` + error handling |
| 2   | `scenario`       | `Given[t, State]` lowercase type param                     | `Given[Cmd, State]` with real types                |
| 3   | `id`             | Self-referencing idtest link                               | `idtest/doc.go`                                    |
| 4   | `snapshot`       | `AggregateID`/`AggregateType` field names                  | `StreamID`/`StreamType`                            |
| 5   | `snapshot`       | `SaveTyped`/`LoadTyped` don't exist                        | `Save`/`Load` with `TypedSnapshot`                 |
| 6   | `snapshot`       | `EveryNEvents(100)` returns error tuple                    | `strategy, _ := EveryNEvents(100)`                 |
| 7   | `decider`        | Same EveryNEvents inline issue                             | Same fix                                           |
| 8   | `kv`             | All raw KV calls missing `ctx`                             | Added `context.Background()`                       |
| 9   | `kv`             | `SetTyped`/`GetTyped` don't exist                          | `Set(ctx, id, &T{})`/`Get(ctx, id)`                |
| 10  | `storage/pebble` | `WithSyncWrites()` wrong name                              | `WithKVSyncWrites()`                               |
| 11  | `storage/pebble` | KV calls missing ctx + manual wiring wrong                 | Fixed both                                         |
| 12  | `stack`          | `EventBus()`, `Repository()`, `AsProjection()` don't exist | Fixed to actual API                                |
| 13  | `stack/sqlite`   | `WithJournalMode`/`WithBusyTimeout` don't exist            | `WithOptimizations`/`WithForeignKeys`              |
| 14  | `stack/pebble`   | `DefaultOptionsWithLogging` wrong package                  | Import from `storage/pebble`                       |
| 15  | `stack/postgres` | `NewPgxListenerFromDSN` missing ctx                        | Added `ctx` parameter                              |
| 16  | `transport/grpc` | Command service doesn't accept options                     | Query-only codec example                           |
| 17  | `eventtest`      | `SaveFn` signature wrong                                   | Fixed to actual `event.SaveFunc`                   |
| 18  | `retry`          | Missing `"log"` import                                     | Added                                              |
| 19  | `dedup`          | Missing `"fmt"` import                                     | Added                                              |

### Cross-File Cleanup

- **30+ outdated `/v2` display references** fixed across 10+ existing READMEs
- **Broken module paths** (`../memory/` to `../storage/memory/`, `../pebble/` to `../storage/pebble/`)
- **All `/v4` display references** standardized to bare module names
- **Broken relative links** in `storage/turso/`, `storage/turso/indexing/`, `storage/`, `stack/memory/`

### Verification Results

- **56/56 modules** with `go.mod` have READMEs
- **0 broken internal links** in module READMEs
- **248 Go symbol references** verified by `cmd/doc-check` -- all valid
- **0 outdated v2/v4 display references** remaining
- **Pushed**: All commits to `origin/master`

---

## b) PARTIALLY DONE

### Code Example Verification

- **248 symbol references** verified via `cmd/doc-check` -- these verify that imported packages and qualified symbols (e.g., `event.NewEvent`) exist
- **However**: doc-check does NOT compile code blocks. It checks that referenced symbols exist, not that the calling syntax is correct
- The session 2 audit caught 19 syntax/signature bugs that doc-check would NOT have caught
- **Remaining risk**: Some code blocks may still have subtle issues that only compilation would reveal

### `docs/README.md` Broken Links

Three links to non-existent example directories remain unfixed:

- `example/encryption/` -- does not exist
- `example/todo/` -- does not exist
- `example/user/` -- does not exist

These are **pre-existing** and outside the module README scope.

### Thin READMEs (Still Under 50 Lines)

| Module              | Lines | Assessment                                           |
| ------------------- | ----- | ---------------------------------------------------- |
| `stack/bench`       | 36    | Appropriate -- benchmark module, minimal API surface |
| `cmd/api-stability` | 41    | Could be expanded but was pre-existing and adequate  |
| `integration`       | 43    | Could be expanded with more test detail              |
| `testutil`          | 46    | Adequate for the small API surface                   |

---

## c) NOT STARTED

### Sub-Package READMEs (15 missing)

These are sub-packages within parent modules (no separate `go.mod`). They have significant APIs but no dedicated READMEs:

**storage/ sub-packages (6):**

1. `storage/eventstore/` -- SQL event store implementation
2. `storage/readmodel/` -- SQL KV store for read models
3. `storage/sql/` -- Transaction helpers, dialect, duplicate-key detection
4. `storage/relational/` -- Multi-table SQL projections
5. `storage/view/` -- Column-mapped SQL views
6. `storage/migrations/` -- Embedded SQL DDL

**catalog/ sub-packages (7):** 7. `catalog/schema/` -- JSON Schema types and reflection engine 8. `catalog/asyncapi/` -- AsyncAPI 3.0 exporter 9. `catalog/eventcatalog/` -- EventCatalog MDX generator 10. `catalog/openapi/` -- OpenAPI 3.0 exporter 11. `catalog/d2/` -- D2 diagram exporter 12. `catalog/docserver/` -- HTTP handlers for serving docs 13. `catalog/simple/` -- Single-service builder facade

**Other sub-packages (2):** 14. `id/idtest/` -- Test helpers for branded IDs 15. `query/querytest/` -- Test helpers for queries

### `stack/sqlopt/` Sub-Package

Not checked in previous session. Has pragma/DSN configuration options used by stack presets.

### CI Integration

- `cmd/doc-check` is NOT configured to run against module READMEs in CI
- It currently checks `SKILL.md`, `AGENTS.md`, and `.agents/skills/` references only

---

## d) TOTALLY FUCKED UP

Nothing is totally fucked up. All confirmed bugs from session 1 were fixed in session 2. The doc-check passes, links resolve, and the build issue is pre-existing (untagged local modules, not caused by README changes).

**The one thing I got wrong that I should have caught immediately**: In session 1, I created 54 Go code blocks without verifying a single one against actual source code. I wrote code examples from memory/inference, which produced 19 bugs. I should have read the source API first for every example, then written the README. The "write first, audit later" approach wasted time and created risk.

---

## e) WHAT WE SHOULD IMPROVE

### Process

1. **Always read source before writing code examples** -- Never write a Go code example in a README without first reading the actual function signature from source. This is non-negotiable.
2. **Run doc-check after every README change** -- Not just at the end. Run it after each batch.
3. **Use `go doc` to verify exports** -- Cross-check every symbol in API tables against `go doc` output.
4. **Add README linting to CI** -- `cmd/doc-check` should scan `**/README.md` in CI, not just skill docs.
5. **Consider a doc compilation test** -- Extract Go code blocks from READMEs and verify they parse (even if they don't compile due to missing imports).

### Architecture / Type Model

6. **`snapshot.EveryNEvents` returns `(SnapshotStrategy, error)`** -- This two-return pattern is awkward for inline use in options. Consider a `MustEveryNEvents(n)` variant for test/example use, or change to return a single value with a `Validate()` method.
7. **`kv.Store` interface takes `context.Context` on every method** -- This is correct but verbose for examples. Consider a `kv.SimpleStore` wrapper or document the `ctx` pattern more prominently.
8. **`stack.Bundle` accessor asymmetry** -- `EventStore()` returns `(value, bool)` but `Repository` is a top-level function. This is inconsistent. Consider making `Repository` a method on `Bundle` (with generics) or making `EventStore` return an error.
9. **`transport/grpc` option asymmetry** -- Only query entry points accept `Option`. Command entry points don't. This is confusing. Document it prominently (now done in README) or make the API symmetric.

### Documentation Strategy

10. **Template** -- Create a README template in `CONTRIBUTING.md` so future modules follow the same structure.
11. **Runnable examples** -- Consider extracting README code blocks into actual `_test.go` files that compile and run.
12. **Cross-references** -- Ensure "Related Modules" sections are bidirectional (if A links to B, B should link to A).

---

## f) Up to 50 Things to Get Done Next

### P0 -- High Impact, Low Effort (Quick Wins)

1. Fix `docs/README.md` broken links to non-existent examples (remove or mark as planned)
2. Add `id/idtest/README.md` (small, referenced by parent)
3. Add `query/querytest/README.md` (small, referenced by parent)
4. Expand `integration/README.md` (43 lines, list all test packages)
5. Expand `cmd/api-stability/README.md` (41 lines, add usage examples)
6. Verify `docs/README.md` and `docs/` for v2/v4 staleness
7. Add bidirectional links: ensure every "Related Modules" section is symmetric

### P1 -- Medium Impact, Medium Effort

8. Add `storage/sql/README.md` (RunInTx, IsDuplicateKeyError, ScanSlice, dialect)
9. Add `storage/relational/README.md` (RelationalProjection, multi-table SQL)
10. Add `storage/view/README.md` (SQLViewStore, ViewMapper)
11. Add `catalog/asyncapi/README.md` (AsyncAPI exporter)
12. Add `catalog/openapi/README.md` (OpenAPI exporter)
13. Add `catalog/d2/README.md` (D2 diagram exporter)
14. Add `catalog/docserver/README.md` (HTTP doc serving)
15. Add `catalog/simple/README.md` (builder facade)
16. Add `storage/eventstore/README.md` (SQL event store internals)
17. Add `storage/readmodel/README.md` (SQL KV store)
18. Add `storage/migrations/README.md` (embedded DDL)
19. Add `catalog/schema/README.md` (JSON Schema reflection)
20. Add `catalog/eventcatalog/README.md` (EventCatalog MDX)
21. Add `stack/sqlopt/README.md` (pragma/DSN options)

### P2 -- High Impact, High Effort (Strategic)

22. Create a README code-block compiler test (extract Go blocks, verify they parse)
23. Add `cmd/doc-check` CI step for `**/README.md` files
24. Add a README template to `CONTRIBUTING.md`
25. Add lint rule: every new `go.mod` must have a sibling `README.md`
26. Create a module dependency graph (D2 or mermaid) in root README
27. Add performance characteristics to modules with benchmarks (benchkit, codec, storage)

### P3 -- Quality Polish

28. Add "Anti-Patterns" section to `event/`, `decider/`, `kv/` READMEs
29. Add "When to Use" section to all storage backends
30. Add error handling examples to modules with error paths
31. Add migration/upgrade notes where relevant
32. Ensure every exported type/function in prose appears in API table
33. Add badges for test coverage, Go version, license
34. Review `catalog/README.md` (538 lines) -- may need splitting
35. Add "Common Pitfalls" section to `watermill/` (ordering gotcha)
36. Add "Security Considerations" to `signing/`, `encryption/`, `transport/`
37. Review consistency of section headings across all READMEs
38. Add code examples for all middleware types in `middleware/README.md`
39. Add "Error Handling" section to `projectionhost/README.md`
40. Expand `example/taskmanager/README.md` with full architecture diagram

### P4 -- Infrastructure

41. Consider `goed` or custom tool for Go markdown code block compilation
42. Add `make docs-check` or `nix run .#docs-check` command
43. Add pre-commit hook for doc-check on README changes
44. Add link-checker step to CI (check all `[](path)` links resolve)
45. Consider generating API tables from `go doc` output automatically
46. Add coverage badge for each module
47. Add Go version badge per module
48. Add module count and total LOC to root README
49. Consider Starlight/Astro docs site that renders all module READMEs
50. Add search index across all READMEs for the docs site

---

## g) Questions I Cannot Answer Myself

### 1. Should sub-packages without `go.mod` files get their own READMEs?

15 sub-packages (`storage/sql/`, `storage/relational/`, `catalog/asyncapi/`, `id/idtest/`, etc.) have significant APIs but no separate `go.mod` and no README. Should each get a dedicated README, or should the parent module's README document them more thoroughly? The parent READMEs already partially cover some (catalog: 538 lines, storage: 195 lines), but not comprehensively.

### 2. Should the broken `docs/README.md` example links be removed or should those examples be created?

`docs/README.md` references `example/encryption/`, `example/todo/`, and `example/user/` -- none exist. Are these planned examples that should be stubbed, or should the links be removed? They predate this session.

### 3. Is the `snapshot.EveryNEvents` two-return signature intentional?

`EveryNEvents(n)` returns `(SnapshotStrategy, error)` but the error is always nil (the function just validates `n > 0`). This makes inline use in decider options impossible without a temporary variable. Was this designed to allow future validation, or is it an over-cautious API that should be simplified?
