# Status Report: cqrs-lint A020-A029 Rules + Cleanup

**Date:** 2026-07-30 14:24
**Session:** Single session, ~90 minutes
**Verify Gate:** GREEN (build + vet + test + race + lint + API stability + doc check)

---

## a) FULLY DONE

### Loose Ends from Prior Session

1. **Extracted `fileImportsCQRS` to `lintutil.FileImportsCQRS`** — Removed duplicated function from `correctness/c025.go` and `consistency/d006.go`. Both now call `lintutil.FileImportsCQRS(gf.AST)`. Removed unused `strings` imports from both files. The `lintutil` package now exports: `AppendBuild`, `IsNonCQRSRegisterPackage`, `CollectPkgLevelVarCalls`, `IsFmtErrorf`, `HasWrapVerb`, `FileImportsCQRS`.

2. **Added C025/D006 overlap integration tests** — 3 tests in `correctness/overlap_test.go`:
   - `TestC025_D006_Overlap_NoDoubleReport` — C025 fires on fmt.Errorf in CQRS file, D006 does NOT
   - `TestC025_D006_Overlap_ErrorsNewStillReportedByD006` — D006 still reports errors.New in CQRS files
   - `TestC025_D006_Overlap_NonCQRSFile_D006Reports` — D006 reports fmt.Errorf in non-CQRS files, C025 skips

3. **Regenerated API stability golden** — `cmd/api-stability` updated to 2749 exports.

4. **Fixed pre-existing `forcetypeassert` lint issue** in `correctness/c023.go:116` — Replaced unchecked chained type assertions (`assign.Rhs[0].(*ast.CallExpr).Fun.(*ast.SelectorExpr)`) with checked form using `ok` returns.

5. **`isInReturnStmt` dead code** — Confirmed already removed by prior session (grep finds zero matches).

### New Rules Implemented (8 detectors)

| Rule | File                         | LOC | Severity | Confidence | Detection Logic                                                                                                       |
| ---- | ---------------------------- | --- | -------- | ---------- | --------------------------------------------------------------------------------------------------------------------- |
| A020 | `api/a020_a021_a022_a023.go` | 60  | warning  | medium     | Struct with 4+ of 6 event.Bus methods (Subscribe, SubscribeAll, Use, UsePublish, Publish, Close) including UsePublish |
| A021 | same                         | 42  | warning  | medium     | Struct with Save+Load+LoadFromVersion (event.Store interface)                                                         |
| A022 | same                         | 55  | info     | high       | Direct `otel.Tracer()`/`otel.Meter()` call in files importing go-cqrs-lite modules                                    |
| A023 | same                         | 42  | warning  | medium     | Struct with "Snapshot" in name + Save+Load methods (SnapshotSink/SnapshotSource)                                      |
| A024 | `api/a024_a025_a026.go`      | 35  | info     | high       | Imports event/ + decider/ but no `event.New`/`event.NewEvent` and no `decider.NewRepository`/`NewTypedRepository`     |
| A025 | same                         | 30  | info     | low        | Imports command/ + query/ but no event/ or decider/ (CQRS without ES)                                                 |
| A026 | same                         | 30  | info     | low        | Imports event/ + watermill/ but no command/ or decider/ or query/ (bare event bus)                                    |
| A029 | `api/a029.go`                | 50  | warning  | high       | `UsePublish` method with body exactly `return nil` (stubbed middleware chain)                                         |

**Skipped rules:**

- **A028** (cqrs-htmx HTTP middleware only) — Too project-specific. cqrs-htmx is a separate framework, not a go-cqrs-lite module. Would require hardcoding an external import path.
- **A030** (in-memory checkpoint store + persistent event store) — Already covered by C017, which detects `NewMemoryCheckpointStore` from the `memory` package paired with a persistent event store.
- **A031** (in-memory DLQ + persistent event store) — Already covered by C017, which detects `NewMemoryDeadLetterStore` from the `projectionhost` package.

### Tests

- **25 new test cases** across 2 files:
  - `api/a020_a029_test.go` — 22 tests (positive + negative for each rule)
  - `correctness/overlap_test.go` — 3 tests (C025/D006 overlap integration)
- All tests use `analyzer.BuildContextFromSource` pattern with inline Go source.
- Each rule has at least one positive test (fires) and one negative test (no false positive).

### Registration & Catalog

- **`register.go`** — 8 new detectors added to `RegisterAll` (A020-A026, A029). Total: 100 detectors (was 84 at session start, 92 after our additions, 100 after daemon added B016-B026).
- **`catalog.go`** — 8 new `RuleInfo` entries added. Total: 55 entries in catalog.go + 45 in catalog_extra.go = 100.
- **`meta_test.go`** — Updated `expected 100 detectors` (daemon updated from 92 to 100).
- **`TestCatalogCountMatchesRegister`** — Passes: catalog and register agree bidirectionally.

### IMPROVEMENT_IDEAS.md

- 8 A-series items (A020-A026, A029) annotated with `~done at c9c2fe86~` format.
- A028 annotated as skipped with rationale.
- A030/A031 annotated as covered by C017.
- Summary table updated: API Misuse existing 20→28, Boilerplate existing 18→26, Total 84→100.
- Header rule count updated: "84 rules" → "100 rules".

### Verification

- `nix run .#verify` — FULLY GREEN (2 consecutive runs)
  - Build: pass
  - Vet: pass
  - Test: all 90+ packages pass
  - Race: all packages pass with -race
  - Lint: 0 issues across all 50+ modules (including the c023.go fix)
  - API Stability: pass (2749 exports)
  - Doc Check: 953 references valid across 39 packages
- `nix fmt` — clean (0 files changed after final formatting)

---

## b) PARTIALLY DONE

### `projectImportsCQRS` / `projectImportsModule` helpers

- `lintutil.FileImportsCQRS` was extracted for file-level checks (used by C025, D006, A022, A029).
- `projectImportsCQRS` (project-level — any file in the project imports CQRS) was created in `a020_a021_a022_a023.go` but NOT extracted to `lintutil`. It's used by A020, A021, A023, A029.
- `projectImportsModule` (project-level — any file imports a specific module suffix) was created in `a024_a025_a026.go` but NOT extracted to `lintutil`. It's used by A024, A025, A026.
- These are currently fine (only used within the `api` package), but if future rules in other packages need them, they should be extracted.

### IMPROVEMENT_IDEAS.md summary table

- The "existing rules" column was updated for API (20→28) and Boilerplate (18→26), but the "Ideas" column still says "31 (A001-A019 existing + A020-A031 new)" which is correct but could clarify that A028/A030/A031 are skipped/covered. The strikethrough annotations handle this, so it's a minor presentation issue.

---

## c) NOT STARTED

1. **A028** (cqrs-htmx HTTP-only middleware) — Skipped by design. Not a go-cqrs-lite module.
2. **A030** (in-memory checkpoint + persistent store) — Covered by C017. No new detector needed.
3. **A031** (in-memory DLQ + persistent store) — Covered by C017. No new detector needed.
4. **Remaining IMPROVEMENT_IDEAS.md items** — B-series (B016-B026 were added by the daemon), E-series (E008-E015), D-series (D007-D012), S-series (S004-S007), P-series (P002-P013), V-series (V002-V006), T-series (T001-T008), F-series (F001-F017). ~70 ideas remain unimplemented.
5. **AGENTS.md update** — The cqrs-lint section in AGENTS.md still says "65 rules across 6 categories" — needs updating to "100 rules".
6. **Status report for this session** — Being written now (this file).

---

## d) TOTALLY FUCKED UP

### Nothing is totally fucked up.

But there are things I should have caught:

1. **`receiverTypeName` duplicates `analyzer.recvTypeName`** — I wrote a `receiverTypeName(recv *ast.FieldList)` function in `a020_a021_a022_a023.go` that duplicates the existing `recvTypeName(fn *ast.FuncDecl)` in the analyzer package. They have different signatures (mine takes `*ast.FieldList`, the existing one takes `*ast.FuncDecl`), so it's not a compile error, but the existing `BaseTypeName` helper could have been used instead. This is a DRY violation I introduced.

2. **`projectImportsCQRS` in `a020_a021_a022_a023.go` reimplements `lintutil.FileImportsCQRS`** — `projectImportsCQRS` iterates all GoFiles and calls `lintutil.FileImportsCQRS(gf.AST)` on each. This is correct but the function could be in `lintutil` as `lintutil.ProjectImportsCQRS(ctx)` since it's a natural project-level counterpart to the file-level helper. Three rules (A020, A021, A023) + A029 use it.

3. **A020/A021/A023 call `collectMethodsByType` on every detector run** — This scans ALL non-test Go files in the project. If all three detectors run in the same lint session, the scan happens 3 times. The result should be cached or shared via the context. Not a correctness issue, but a performance one.

4. **A024 `projectCallsPkgFunction` is too narrow** — It only checks `event.New` and `event.NewEvent`, but a consumer might create events via a helper function (e.g., `makeEvent()` which internally calls `event.New`). This would cause a false positive on A024. The heuristic is documented as "imports event/ and decider/ but never creates events", but "creates events" should include indirect creation.

5. **A025/A026 are informational with low confidence** — They flag "CQRS without ES" and "bare event bus" patterns. These may be intentional architectural choices. The low confidence is correct, but the finding message could be more nuanced — it currently says "may miss audit trail" which is vague. Could suggest specific features the consumer is missing (replay, temporal queries, etc.).

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. **Extract `receiverTypeName` to use `analyzer.BaseTypeName`** — Replace the custom `receiverTypeName(recv *ast.FieldList)` with a call to `analyzer.BaseTypeName(recv.List[0].Type)`. The existing helper handles `*ast.StarExpr` and `*ast.IndexExpr` unwrapping.

2. **Extract `projectImportsCQRS` and `projectImportsModule` to `lintutil`** — Both are project-level counterparts to `FileImportsCQRS`. If future rules in other packages (correctness, consistency, architecture) need project-level CQRS import checks, they'd need to reimplement these.

3. **Cache `collectMethodsByType` results** — The method-type map is computed independently by A020, A021, and A023. It should be computed once and shared (either via the `AnalysisContext` or a sync.OnceValue).

4. **A024: detect indirect event creation** — Extend the heuristic to detect functions that call `event.New` or `event.NewEvent` internally, not just direct calls.

5. **A020: reduce false positive risk** — The 4-of-6 method threshold with UsePublish required is good, but could be strengthened by checking that the struct is NOT from a known library package (watermill, storage/memory). Currently relies on the struct being defined in the consumer's code, which is implicit from the analysis context.

6. **A022: also detect `otel.GetTracerProvider().Tracer()` pattern** — Some consumers call `otel.GetTracerProvider().Tracer("name")` instead of `otel.Tracer("name")`. The current detector only catches the direct `otel.Tracer()` form.

7. **A029: detect more stub patterns** — Currently only matches `return nil` as the sole statement. Should also match `return nil, nil` (for methods returning two values) and empty body with just a comment.

### Architecture

8. **`collectMethodsByType` should be a context method** — The analyzer package should provide a `CollectMethodsByType(ctx)` that caches the result in `AnalysisContext`. Multiple rules needing method-set information should share one scan.

9. **`projectImportsModule` should use `lintutil.FileImportsCQRS`** — The current implementation reimplements the import-scanning logic with `strings.Contains(path, "go-cqrs-lite/"+moduleSuffix)`. It should use `analyzer.IsCQRSModulePath` for consistency.

10. **AGENTS.md stale** — Still says "65 rules across 6 categories". Needs updating to "100 rules across 7 categories" (correctness, API, boilerplate, consistency, architecture, security, performance + version).

### Testing

11. **No tests for A020/A021/A023 false-positive edge cases** — Missing tests for:
    - Struct that implements the full interface but is from watermill/storage/memory (should not fire)
    - Struct with methods from embedded interface (should not fire on the embedder)
    - Generic struct types with method sets

12. **No integration test running all 8 new detectors together** — Each test runs one detector in isolation. A test that runs all 100 detectors on a fixture project would catch registration/ordering issues.

13. **A024 test doesn't verify the suggestion message** — The test checks finding count but not the message content. Should verify the suggestion mentions "event store, bus, and decider.Repository".

14. **No test for A022 with `otel.Meter()`** — Wait, actually `TestA022_DetectsRawOtelMeter` exists. But no test for `otel.GetTracerProvider().Tracer()` pattern (see improvement #6).

15. **Race detector parallel test flake** — `TestB025_NoFindingWithStateCache` flaked under `-race` when running all packages in parallel. Passes in isolation. Pre-existing (daemon's B025 code, not ours), but the test suite should be robust to parallel execution.

---

## f) Up to 50 Things We Should Get Done Next

### Immediate (fix what I broke or missed)

1. Replace `receiverTypeName` with `analyzer.BaseTypeName` in `a020_a021_a022_a023.go`
2. Extract `projectImportsCQRS` to `lintutil.ProjectImportsCQRS(ctx)`
3. Extract `projectImportsModule` to `lintutil.ProjectImportsModule(ctx, suffix)`
4. Cache `collectMethodsByType` in `AnalysisContext` or via `sync.OnceValue`
5. Fix A024 to detect indirect event creation (functions that call `event.New` internally)
6. Add test for A020 with watermill bus (should not fire)
7. Add test for A021 with storage/memory.MemoryStore (should not fire)
8. Add test for A023 with storage/memory.NewMemorySnapshotStore (should not fire)
9. Add test for A022 with `otel.GetTracerProvider().Tracer()` pattern
10. Add test for A029 with `return nil, nil` stub pattern
11. Update AGENTS.md cqrs-lint section: "65 rules" → "100 rules", update category count
12. Investigate B025 parallel test flake (pre-existing, not ours, but should be fixed)

### A-series remaining

13. Implement A028 alternative: detect projects importing go-cqrs-lite but using zero CQRS types (more generic than cqrs-htmx-specific)
14. Consider whether A030/A031 need their own findings or if C017's message should be more specific about which store type is in-memory
15. A032+: review remaining IMPROVEMENT_IDEAS.md for any A-series items not yet covered

### B-series (daemon added B016-B026, verify quality)

16. Review daemon's B016-B026 detectors for correctness
17. Review daemon's B016-B026 tests for coverage gaps
18. Run `nix run .#verify` one more time after daemon's commits settle
19. Check if daemon's B022-B026 catalog entries match register entries
20. Verify daemon's boilerplate detectors don't overlap with existing A-series rules

### E-series (E008-E015)

21. Implement E008: Circular module dependency in go.work
22. Implement E009: Missing go.work file (multi-module project without workspace)
23. Implement E010: Module version mismatch (v3 vs v4)
24. Implement E011: Fat domain package (>20 types in one package)
25. Implement E012: Projection package importing command types
26. Implement E013: Handler package importing storage types
27. Implement E014: Domain package importing transport types
28. Implement E015: Test package importing production internals

### D-series (D007-D012)

29. Implement D007: Inconsistent event type naming (dot vs Pascal)
30. Implement D008: Inconsistent ID type usage (string vs id.Of[T])
31. Implement D009: Inconsistent error wrapping style
32. Implement D010: Inconsistent snapshot frequency
33. Implement D011 already exists — verify it's complete
34. Implement D012: Inconsistent codec usage (JSON + CBOR mixed)

### S-series (S004-S007)

35. Implement S004: Hardcoded secrets in event payloads
36. Implement S005: Unencrypted sensitive event payloads
37. Implement S006: Missing event signing in tamper-sensitive domains
38. Implement S007: Unverified event signatures on consume

### P-series (P002-P013)

39. Implement P002: N+1 query pattern in projection handler
40. Implement P003: Unbounded event stream load (no pagination)
41. Implement P004: Missing index on frequently queried column
42. Implement P005: Synchronous publish in hot path
43. Implement P006: Missing batch write in projection handler
44. Implement P008-P013: Various performance anti-patterns from IMPROVEMENT_IDEAS.md

### V-series (V002-V006)

45. Implement V002: Stale go-cqrs-lite version (major behind)
46. Implement V003: Pinned version preventing updates
47. Implement V004: Mixed v3 and v4 in same module
48. Implement V005: Replace directive masking version
49. Implement V006: Missing go.sum entry

### Meta/T-series/F-series

50. Implement T001-T008 (testing rules) and F001-F017 (feature adoption rules) — these are newer categories with zero existing rules

---

## g) Questions I Cannot Answer Myself

1. **Should A020/A021/A023 exclude types from known library packages (watermill, storage/memory) by checking the import path of the file where the struct is defined?** The current approach relies on the struct being in consumer code (implicit from the analysis context), but a consumer could import and alias a library type. I don't know if the analyzer has the type information to distinguish.

2. **Should the `collectMethodsByType` cache live in `AnalysisContext` or in a separate caching layer?** The `AnalysisContext` is already carrying `Registry`, `FeatureProfile`, etc. Adding a method-map cache would couple it to a specific use case. But `sync.OnceValue` per-detector means the scan still runs 3 times if 3 detectors need it. The right answer depends on how the linter orchestrates detector execution (parallel? sequential?).

3. **Should A025/A026 (CQRS-without-ES and bare-bus patterns) be suppressed by a feature profile flag?** These are informational findings that may be intentional. A016 (idempotency) uses `ctx.FeatureProfile.CommandFlow` to suppress for read-only systems. Should A025/A026 have a similar gate, or is the low confidence + info severity sufficient? I don't know the project's philosophy on when to suppress vs always-report informational findings.
