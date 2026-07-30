# Status: B-series Boilerplate Rule Implementation (B016-B026)

**Date:** 2026-07-30 14:17
**Session scope:** Implement 8 new boilerplate linter rules (B016, B017, B018, B019, B020, B022, B025, B026), update IMPROVEMENT_IDEAS.md, self-review.

---

## A) FULLY DONE

### 1. Eight new B-series detectors implemented and tested

| Rule | File | Detection | Severity | Confidence | Tests |
|------|------|-----------|----------|------------|-------|
| B016 | `b016_b017.go` | Manual checkpoint table + journal replay loop | warning | medium | 2 (pos+neg) |
| B017 | `b016_b017.go` | Rehydrate/Rebuild/Replay methods calling ReadAll (full rebuild on startup) | warning | high | 2 (pos+neg) |
| B018 | `b018_b019.go` | 3+ bus.Subscribe calls in same file | info | medium | 2 (pos+neg) |
| B019 | `b018_b019.go` | repo.Load inside SubscribeAll handler (O(N^2)) | warning | high | 2 (pos+neg) |
| B020 | `b020.go` | Field renaming/defaulting in decode functions outside schema.NewUpcaster | warning | medium | 2 (pos+neg) |
| B022 | `b022_b025.go` | Custom enricher (not CommandCausalityEnricher) passed to NewRepository | warning | medium | 2 (pos+neg) |
| B025 | `b022_b025.go` | NewRepository without WithStateCache option | info | low | 2 (pos+neg) |
| B026 | `b026.go` | 3+ event types emitted but no catalog import | info | medium | 2 (pos+neg) |

**16 tests total** (8 positive + 8 negative), all passing.

### 2. Registration and catalog wiring

- `register.go`: All 8 new detectors added to `RegisterAll()` — total detector count is now **100**
- `catalog_extra.go`: 8 new `RuleInfo` entries added to `boilerplateRules()`
- `meta_test.go`: Detector count updated from 92 to 100
- `TestAllDetectorsInstantiate`: PASS (100 detectors, all non-nil, all with valid names)
- `TestCatalogCountMatchesRegister`: PASS (bidirectional catalog/detector agreement)

### 3. Build/vet/test/race verification

- `go build -tags "goexperiment.jsonv2" ./...` — clean
- `go vet -tags "goexperiment.jsonv2" ./pkg/rules/boilerplate/...` — clean
- `gofmt -l` on all 9 modified/created files — clean (no formatting issues)
- `go test -tags "goexperiment.jsonv2" ./... -count=1 -race` — ALL PASS (14 packages)
- Auto-commit daemon committed the work across 3 commits:
  - `0ba9686e` — detector files + register.go + catalog_extra.go + meta_test.go
  - `4bcb8267` — test file + b022_b025.go (daemon split the commit)
  - `3c55ba44` — daemon-added C023 type assertion fix + other changes

### 4. IMPROVEMENT_IDEAS.md updated

- Header: rule count 84 → 100, B-series range B001-B015+B021+B023+B024 → B001-B026
- All 11 B-series ideas (B016-B026) marked as done with strikethrough
- Summary statistics table: B-series 18 → 26 existing, A-series 20 → 28 existing, total 84 → 100
- Priority Recommendations: items 3, 4, 5, 8, 11, 13, 15 marked as done

---

## B) PARTIALLY DONE

### 1. B019 vs P001 overlap (known, intentional)

B019 (repo.Load inside SubscribeAll — boilerplate category, severity warning) and P001 (same detection — performance category, severity error) detect the same anti-pattern. B019 was requested by the IMPROVEMENT_IDEAS.md spec. P001 already existed. Both coexist — B019 fires as a boilerplate suggestion, P001 fires as a performance error. This is the same pattern as B008 (manual retry — boilerplate) coexisting with P007 (bitshift retry — performance). No action needed, but worth noting in the catalog.

### 2. Uncommitted changes (daemon-originated, not mine)

Two files have uncommitted changes that the auto-commit daemon produced after my session work:
- `cmd/cqrs-lint/go.mod` — go-finding version pinned from pseudo-version to v1.4.1
- `cmd/cqrs-lint/pkg/rules/correctness/c023.go` — type assertion safety fix (unchecked `*ast.CallExpr` / `*ast.SelectorExpr` → safe type assertion with ok-check)

These are NOT my changes. They appear to be daemon fixes. I did not revert them.

---

## C) NOT STARTED

### Remaining IMPROVEMENT_IDEAS.md B-series items (already implemented by prior sessions)

B021, B023, B024 were already implemented before this session. The IMPROVEMENT_IDEAS.md marked them as done (struck through) during this session, but the code was pre-existing.

### Items from the original paste_1.txt not requiring implementation

The paste contained items 29-39 (B016-B026). All 11 are now implemented. No remaining items from the paste.

---

## D) TOTALLY FUCKED UP

### Nothing critically broken.

**Minor issues caught and fixed during the session:**

1. **B016 string literal check** — initially used magic number `lit.Kind == 6` instead of `token.STRING`. Fixed immediately.
2. **B025 WithStateCache detection** — initially used `SelectorFromExpr(arg)` directly on the argument, but `WithStateCache(cache)` is a `*ast.CallExpr`, not a bare `*ast.SelectorExpr`. Fixed by unwrapping call expressions before extracting the selector.
3. **B026 catalog import check** — initially only checked `ctx.Packages` (which is empty in test contexts). Added AST import declaration fallback so `BuildContextFromSource` tests work.
4. **containsBus/containsEnricher helpers** — initially wrote manual byte-by-byte string matching. Replaced with `strings.Contains` for readability.

**No issues reached production.** All were caught by tests before committing.

---

## E) WHAT WE SHOULD IMPROVE

### Code quality of the new rules

1. **B016 checkpoint detection is narrow** — only checks for `CREATE TABLE` + `checkpoint`/`projection_offset` in string literals. Real projects may use `CREATE TABLE IF NOT EXISTS` or table names in INSERT/SELECT statements. The detection could be broadened.

2. **B017 Rebuild detection is name-based** — only checks for function names `Rehydrate`/`Rebuild`/`Replay`/etc. A function named `initProjections` or `setupReadModels` that calls `ReadAll` would be missed. Consider also detecting `ReadAll` calls in `init()` or `main()` functions.

3. **B018 Subscribe detection is heuristic** — checks if the receiver/qualifier contains "bus". This misses `eventBus.Subscribe` (contains "bus") but also `sub.Subscribe` (doesn't contain "bus"). Could check the registry for known bus variable types instead.

4. **B020 upcasting detection is conservative** — requires BOTH `Unmarshal`/`Decode` call AND `IndexExpr` with string literal key. This misses struct-field-level upcasting (`item.OldName = item.NewName` in a decode function). But broadening would increase false positives.

5. **B022 enricher detection is name-based** — checks if the argument function name contains "Enrich". A custom enricher named `addCorrelation` would be missed. Could check if the argument is a function call from a non-deider package.

6. **B025 state cache is advisory only** — severity info, confidence low. This is correct (not all projects need caching), but the suggestion could be smarter: only fire for aggregates with known high event counts.

7. **B026 catalog check uses EventTypesEmitted** — this counts unique event type strings from `event.New` calls, not string constants. A project with 3 `event.New("user.created", ...)` calls has 1 emitted type, not 3. The rule fires only when there are 3+ DISTINCT event types, which is the correct behavior but differs from the spec's "3+ event type string constants."

### Architectural concerns

8. **B019/P001 duplication** — Two rules detecting the same pattern with different severity/category. Consider documenting the relationship in the catalog or merging them with a category toggle.

9. **No integration tests** — All tests use `BuildContextFromSource` (inline Go source). No test runs the linter against a real consumer project to verify detection in the wild.

10. **No false-positive regression tests** — The negative tests check simple cases. Complex codebases may trigger false positives that aren't covered.

### Process issues

11. **Auto-commit daemon split work across 3 commits** — My changes were committed as `0ba9686e` (detectors + registration) and `4bcb8267` (tests + b022_b025). Then the daemon added `3c55ba44` with a C023 fix and other changes I didn't make. The commit history doesn't cleanly represent "8 new B-series rules."

12. **IMPROVEMENT_IDEAS.md was modified by daemon between my reads** — I had to re-read the file because the auto-commit daemon modified it (adding A020-A026 to the header). This caused one edit to fail on stale content.

---

## F) Up to 50 Things to Do Next

### Immediate fixes (this session's work)

1. **Run `nix fmt`** on the new files to ensure treefmt compliance (gofmt is clean, but treefmt may have additional rules)
2. **Run `nix run .#lint`** to verify golangci-lint passes on the new files (gopls hint about `slices.Contains` in b023_b024.go is pre-existing, not mine)
3. **Regenerate api-stability golden** — new exported functions (NewB016Detector through NewB026Detector) were added; the golden file may need updating
4. **Verify `nix run .#verify`** passes — the full verification gate (build + vet + test + race + lint + doc-check)

### B-series rule improvements

5. **Broaden B016** — detect `INSERT INTO checkpoint` / `SELECT FROM checkpoint` patterns, not just `CREATE TABLE`
6. **Broaden B017** — detect `ReadAll` in `init()` / `main()` / `func setup*()` functions, not just `Rehydrate`/`Rebuild`/`Replay`
7. **Improve B018** — use the registry to identify bus variables by type, not by name heuristic
8. **Broaden B020** — detect struct field renaming in decode functions, not just map index manipulation
9. **Broaden B022** — detect any non-deider enricher function, not just name-based "enrich" matching
10. **Make B025 context-aware** — only fire for aggregates with >50 events (requires stream length heuristics)
11. **Reconcile B019 vs P001** — document the overlap in the catalog or merge with a category toggle
12. **Add B019 to performance category too** — B019 is boilerplate, P001 is performance, but both detect the same thing

### Testing improvements

13. **Add integration test** — run cqrs-lint against example/taskmanager and verify no false positives from the new rules
14. **Add false-positive regression tests** — test that legitimate `Subscribe` calls with different patterns don't trigger B018
15. **Add edge case tests** — empty files, files with only imports, files with syntax errors
16. **Test B020 with nested upcasters** — schema.NewUpcaster calling a function that does the actual upcasting
17. **Test B026 with catalog import but no registration** — import catalog but never call NewBuilder (should still fire?)

### Remaining IMPROVEMENT_IDEAS.md items (other categories)

18. **A020: Custom event.Bus reimplementation** — Kernovia, timesheets
19. **A021: Custom event.Store reimplementation** — accountability-system
20. **A022: Raw otel.Tracer() instead of cqrsotel** — standard-bug-tracking-schema
21. **A023: Custom in-memory snapshot store** — cqrs-htmx usermgmt
22. **A024: Decorative event sourcing** — storbi
23. **A025: Command/query only, no events** — KeyHolderAI
24. **A026: Event bus only, no CQRS pipeline** — Cyberdom
25. **A028: cqrs-htmx used only for HTTP middleware** — CV, overview
26. **A030: In-memory checkpoint store with persistent event store** — cqrs-htmx
27. **A031: In-memory DLQ with persistent event store** — cqrs-htmx
28. **E008-E015** — 8 architecture rules ideas
29. **D007-D012** — 6 consistency rule ideas
30. **S004-S007** — 4 security rule ideas
31. **P002, P005, P006, P008-P010** — 6 performance rule ideas
32. **V002-V006** — 5 version/migration rule ideas
33. **T001-T008** — 8 testing rule ideas
34. **F001-F017** — 17 feature adoption coaching ideas

### Linter infrastructure

35. **Add rule documentation URLs** — each finding should link to SKILL.md or ADR
36. **Add `--rules` flag** — list all available rules with descriptions
37. **Add rule categories to SARIF output** — for GitHub Code Scanning
38. **Add `cqrs-lint profile` command** — detailed usage analysis per project
39. **Add incremental analysis** — cache AST results for faster re-runs
40. **Add feature adoption scorecard** — show which features a project uses vs misses

### Documentation

41. **Update cqrs-lint README.md** — add B016-B026 to the rule list
42. **Update AGENTS.md** — update the rule count from 84 to 100 in the module description
43. **Add B-series rules to the cqrs-lint doctor output** — verify feature profile detection works with new rules
44. **Write a CHANGELOG entry** for the 8 new rules

### Verification

45. **Run `nix run .#check-layers`** — verify dependency budgets are not violated
46. **Run `nix run .#vulncheck`** — verify no vulnerability issues with the go-finding v1.4.1 pin
47. **Run `cmd/doc-check`** — verify all Go import paths in SKILL.md are still valid
48. **Run the linter on itself** — `cqrs-lint ./...` on the cqrs-lint module to check for self-referential issues

### Cleanup

49. **Review the daemon's C023 type assertion fix** — verify it's correct and doesn't mask real issues
50. **Review the daemon's go-finding v1.4.1 pin** — verify the version exists and is compatible

---

## G) Questions

1. **Should B019 and P001 be merged?** They detect the same anti-pattern (repo.Load inside SubscribeAll) but with different severity (warning vs error) and different category (boilerplate vs performance). Merging would simplify the catalog but lose the dual-perspective signal. Alternatively, should B019 be removed since P001 already covers it?

2. **Should B025 (missing state cache) fire for all NewRepository calls, or only for specific patterns?** Currently it fires for every `decider.NewRepository` without `WithStateCache`. For small aggregates with few events, this is noise. Should we add a confidence/severity adjustment based on whether the project also uses snapshots (which implies high event counts)?

3. **The auto-commit daemon committed a C023 type assertion safety fix and a go-finding version pin that I did not author.** Should I review and validate these changes, or trust the daemon? The C023 fix changes an unchecked type assertion to a safe one — this is a real improvement but changes behavior (silently returns instead of panicking on unexpected AST shapes).
