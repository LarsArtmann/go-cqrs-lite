# Status Report: D-Series Consistency Rules (D007-D013) Implementation

**Date:** 2026-07-30 18:04
**Session Scope:** Implementing D007-D010 + D013 from IMPROVEMENT_IDEAS.md §5 (items 48-53)
**Prior Session:** F-series adoption rules (17 rules, all done)

---

## a) FULLY DONE

### 5 New D-Series Detectors Implemented

| Rule | File                | Lines | Detection Type                                                       | Severity    |
| ---- | ------------------- | ----- | -------------------------------------------------------------------- | ----------- |
| D007 | `d007_d008_d013.go` | —     | Project-level (both `event.New` + `event.NewEvent`)                  | info/medium |
| D008 | `d007_d008_d013.go` | —     | Project-level (both `DecodePayload` + `DecodePayloadAuto`)           | info/medium |
| D009 | `d009_d010.go`      | —     | Project-level (`io.Closer` + anonymous `interface{ Close() error }`) | info/low    |
| D010 | `d009_d010.go`      | —     | Per-occurrence (`"internal"` literal in `errorfamily.Wrap*`)         | info/medium |
| D013 | `d007_d008_d013.go` | —     | Project-level (event creation without `WithSchemaVersion`)           | info/low    |

### Wiring

- **Catalog entries:** All 5 added to `consistencyRules()` in `catalog_extra.go`
- **Registration:** All 5 registered in `register.go` in the Consistency block
- **Meta-test count:** Updated from 145 → 150 in `meta_test.go`
- **Bidirectional parity:** Verified by `TestCatalogCountMatchesRegister` (150 = 150)

### Tests

- **13 new tests** in `d007_d013_test.go` (separate file because `new_rules_test.go` was already 241 lines, would exceed 350-line CI limit)
- All tests pass with `-race`
- Each rule has: positive test (finding fires), negative test (no false positive), edge cases

### Verification

- `GOWORK=off go build` — clean
- `GOWORK=off go test ./... -count=1` — all 16 packages pass
- `GOWORK=off go test -race ./pkg/rules/consistency/...` — all pass
- `cmd/cqrs-lint` self-lint: D007 and D009 fire legitimate findings on the repo itself
- `nix fmt` applied to all files
- `nix run .#verify` — build + vet + test + race ALL PASS; lint failures are pre-existing in unrelated modules

### Documentation

- `IMPROVEMENT_IDEAS.md` §5: header marked "DONE (5/5 new rules)", all items 48-53 struck through with `done` + file references
- `AGENTS.md`: rule count updated 145 → 150, added D-series description

### Design Decision: D013 not "D012 schema-version"

Item 53 in IMPROVEMENT_IDEAS.md specified "D012: Schema version not stamped on events". However, D012 was already implemented as "raw-print-in-handler" (done in a prior session, cataloged and registered). Rather than repurposing D012 and breaking existing consumers, assigned **D013** (next free number) to the schema-version rule. Documented this decision in the strikethrough note.

---

## b) PARTIALLY DONE

### Nothing

All 5 rules are fully implemented with tests, wiring, and documentation.

---

## c) NOT STARTED

### From This Session's Scope — Nothing

The spec (items 48-53) is fully covered. D011 was already done (prior session).

### From Prior Session Backlog (F-series, still open)

These are from the F-series session status report, not this session's scope:

1. `nix fmt` on F-series files (done in this session as a side effect)
2. api-stability golden regeneration for F-series `adoption` package
3. F011 false-positive risk (broad `.Exec` matching)
4. F009 incomplete timer detection
5. F013 incomplete HTTP handler detection
6. 4 rule overlaps (F002/B026, F003/B014, F006/S002, F007/A016)
7. Threshold boundary tests for F-series
8. Remove unnecessary indirection in adoption package (`strconv.go`, `itoa()`, `projectHasCall`)
9. 4 self-lint findings from F-series (suppress or adopt)

---

## d) TOTALLY FUCKED UP

### D009 `isSingleCloseInterface` — Overly Broad Matching

The `isSingleCloseInterface` function only checks that the interface has exactly 1 method named `Close`. It does NOT verify the method signature is `Close() error`. An interface declaring `Close() string` or `Close(ctx context.Context)` would also match. This is a false-positive risk.

**Impact:** Low (most anonymous `Close` interfaces do return error), but imprecise.

### D010 — Only Checks `errorfamily` Package Qualifier

D010 matches `errorfamily.WrapTransient(...)` but NOT aliased imports like `ef.WrapTransient(...)`. If a consumer aliases the import, the rule won't fire. This is acceptable (most projects use the full name), but a gap.

### D013 — Fires for Projects with a Single Event

D013 fires the moment any `event.New`/`event.NewEvent` call exists without `WithSchemaVersion`. For a project with just 1-2 events, this is noisy coaching. A threshold (e.g., minimum 3 events before firing) would reduce false positives on small/example projects.

### No Threshold Boundary Tests

None of the new rules have threshold boundary tests (e.g., D010 with exactly 1 `errorfamily.Wrap*` call, D013 with exactly 1 event). The existing tests cover positive/negative but not boundary conditions.

### D009/D010 Shared Helper Placement

The `selectorPkgAndName` helper is defined in `d007_d008_d013.go` but used by `d009_d010.go` in the same package. This works in Go (same package), but the helper logically belongs in a shared helpers file, not embedded in one rule file. The adoption package has a dedicated `helpers.go` — consistency package does not.

### `anchorPos` Helper — Only Used by D007/D008/D013

The `anchorPos` helper handles the case where no call site was captured (falls back to `go.mod`). But in practice, D007/D008/D013 always capture a call site (the condition requires finding calls). The fallback path is dead code for these rules.

---

## e) WHAT WE SHOULD IMPROVE

### Quality of Detection

1. **D009 method signature checking** — Verify the `Close` method returns `error` and takes no parameters, not just that it's named `Close`
2. **D013 threshold** — Only fire when there are 3+ event creation calls (skip tiny/example projects)
3. **D010 import alias awareness** — Resolve the actual package path from the import map, not just the qualifier string. This would catch `ef.WrapTransient(err, "internal", ...)` where `ef` aliases `errorfamily`
4. **D008 cross-file detection precision** — Currently scans for `event.DecodePayload` and `event.DecodePayloadAuto` selector calls. The `event.` qualifier is assumed but not resolved to the actual import path. Same alias problem as D010

### Test Coverage

5. **Boundary tests** — Add tests at exact thresholds (if thresholds are added)
6. **Multi-file test for D010** — D010 fires per-occurrence; a multi-file test would verify cross-file accumulation
7. **D013 test with multiple events** — Current test only has 1 event; test with 3+ to verify the count in the message

### Architecture

8. **Shared helpers file for consistency package** — Extract `selectorPkgAndName`, `anchorPos`, `isSingleCloseInterface`, `isErrorFamilyWrapper`, `hasInternalLiteral` into `helpers.go` (like adoption package)
9. **Remove `anchorPos` dead-code fallback** — Or make it explicit that it's a safety net

### Self-Lint Findings

10. **D007 fires on the repo itself** — `benchkit/phases.go:203` uses both `event.New` and `event.NewEvent`. This is a legitimate finding. Either standardize the repo on `event.New` or suppress
11. **D009 fires on the repo itself** — `command/dispatcher.go:30` uses both patterns. Either fix or suppress

### Documentation

12. **No per-rule README or docs** — The catalog entry descriptions are good, but there's no expanded documentation explaining the rationale and fix steps
13. **D013 naming note not propagated to README.md** — The README table still lists D012 as "raw-print-in-handler" with no mention of D013. This is correct (D013 is new), but the rule count/mention should be verified

---

## f) Up to 50 Things to Get Done Next

### Immediate (This Session's Rules — Hardening)

1. Fix D009: Verify `Close()` method signature (returns `error`, no params), not just method name
2. Add D013 threshold: skip projects with fewer than 3 event creation calls
3. Add boundary tests for D010 (exactly 1 occurrence, exactly 2)
4. Add D013 multi-event test (3+ events, verify message contains count)
5. Add D009 false-positive test: `interface{ Close() string }` should NOT match
6. Extract shared helpers from rule files into `helpers.go` in consistency package
7. Resolve D007 self-lint finding: standardize `benchkit/phases.go` on `event.New`
8. Resolve D009 self-lint finding: fix or suppress `command/dispatcher.go`
9. Remove dead-code fallback in `anchorPos` (or document it as safety net)

### Import Alias Resolution (Cross-Cutting)

10. Build a shared `resolvePackagePath` helper that maps qualifier → import path using the file's import declarations
11. Apply to D007, D008, D010 (all rely on hardcoded `event.` / `errorfamily.` qualifiers)
12. Apply to D013 (same `event.` qualifier assumption)
13. Add test: aliased import (`ef "errorfamily"`) correctly detected by D010

### Prior Session Backlog (F-Series)

14. Fix F011 false-positive risk (broad `.Exec` matching in relational projection detection)
15. Fix F009 incomplete timer detection (add `time.Tick`, `time.After`, `time.NewTicker`)
16. Fix F013 incomplete HTTP handler detection (add chi/gin/echo/fiber)
17. Fix F005 imprecision (parse version argument, only fire when n > 1)
18. Resolve F002/B026 overlap (catalog registration)
19. Resolve F003/B014 overlap (OTel middleware)
20. Resolve F006/S002 overlap (PII encryption)
21. Resolve F007/A016 overlap (idempotency middleware)
22. Add F-series threshold boundary tests
23. Remove `strconv.go` and `itoa()` indirection in adoption package
24. Remove `projectHasCall` wrapper in adoption package
25. Clean up `var _ =` patterns in adoption `rules_test.go`
26. Handle 4 F-series self-lint findings (suppress or adopt features)
27. Extract PII field list to `lintutil.PIIFieldNames` and share between F006 and S002

### API Stability

28. Regenerate api-stability golden for `adoption` package (new exports from F-series)
29. Regenerate api-stability golden for `consistency` package (5 new detector constructors)
30. Verify `TestEveryGoModDirIsInModulesList` still passes

### Catalog & README

31. Update `cmd/cqrs-lint/README.md` rule table with D007-D013 entries
32. Verify catalog severity/confidence values match test expectations
33. Add D013 to the "schema evolution" documentation cross-reference

### Test Infrastructure

34. Add a shared `runDetectorAtCount` helper that asserts BOTH the rule ID and exact finding count, reducing boilerplate in test files
35. Add a `noFindingsForPackage` test helper that verifies a rule produces zero findings on clean code
36. Create a golden-file test for the full self-lint output to catch unexpected finding changes

### Pre-Existing Lint Issues (Not Mine)

37. Fix `benchkit/generator.go` golines + tagalign issues (daemon-introduced)
38. Fix `command/store.go:28` godoclint issue (pre-existing)
39. Fix `query/store.go:28` godoclint issue (pre-existing)
40. Fix `catalog/types_phantom.go:9` godoclint issue (pre-existing)
41. Fix `storage/memory/snapshot.go:34` godoclint issue (pre-existing)
42. Fix `storage/eventstore/snapshot.go:52` godoclint issue (pre-existing)

### Broader Rule Improvements

43. Consider whether D007 should also detect `event.NewEvent` usage patterns (e.g., in test files — currently skipped)
44. Consider whether D008 should distinguish between fold vs projection contexts
45. Consider whether D010 should also flag `"error"`, `"unknown"`, `"default"` as generic codes
46. Consider whether D013 should check for `schema.NewUpcaster` usage (like F005 does) to avoid overlap
47. Add D009 support for `io.ReadCloser`, `io.WriteCloser`, `io.Closer` variants
48. Add a meta-rule that detects when two rules of the same category have overlapping detection conditions
49. Document the rule-numbering policy (when to skip, when to reuse) to prevent future D012-style collisions
50. Consider a `--explain <ruleID>` CLI flag that prints the full rationale, fix steps, and examples for a rule

---

## g) Questions

### Q1: Should the schema-version rule fire for example/ projects?

D013 fires on any project that creates events without `WithSchemaVersion`. The `example/` directory projects (getting-started, readme-quickstart, taskmanager) are intentionally minimal and may not need schema versioning. Should I add a `//cqrs-lint:ignore(D013)` to example files, add a path-based exclusion for `example/`, or leave the findings as legitimate coaching?

### Q2: Should D010 flag other generic codes beyond "internal"?

D010 currently only matches the literal string `"internal"`. Should it also flag other generic/non-descriptive error codes like `"error"`, `"unknown"`, `"default"`, `"misc"`, `"other"`, `"miscellaneous"`? If so, should the list be hardcoded or configurable via `.cqrs-lint.json`?

### Q3: Should D007/D008 also scan test files?

D007 and D008 currently skip test files (`gf.IsTest`). This means inconsistency between production code and test code won't be detected. Is this the right behavior? Test files using `event.NewEvent` while production uses `event.New` is arguably still inconsistent, but test code may intentionally exercise both APIs.
