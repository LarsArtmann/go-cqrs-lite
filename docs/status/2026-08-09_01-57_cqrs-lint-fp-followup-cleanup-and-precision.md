# Status: cqrs-lint FP Follow-up — Regression Tests, Precision Fixes, Dead Code Cleanup

**Date:** 2026-08-09 01:57
**Session start:** ~01:30
**Branch:** master (15 commits ahead of origin)

---

## Context

This session resumed from a prior session (`2026-08-09_01-26`) that had completed 7
cqrs-lint false-positive follow-up tasks but left several gaps: dead code, missing
tests, no `nix fmt`, no `nix run .#verify`. The resuming prompt listed 8 exact
next steps. This session executed 5 of them.

---

## a) FULLY DONE

### 1. Removed dead `PackagesWithRegistration` code (3 sites)

The `PackagesWithRegistration` field in `CQRSRegistry` was populated by the
scanner but never read by any rule after the prior session removed the E007
package-wide suppression heuristic. Cleaned up all 3 sites:

- **`registry.go:57-61`** — Removed field declaration + doc comment
- **`registry.go:75`** — Removed `make(map[string]bool)` init in `NewCQRSRegistry`
- **`scanner_calls.go:29-32`** — Removed the `if funcName == "RegisterTyped" || ...`
  block that populated the map

Verified with `rg "PackagesWithRegistration"` — zero remaining non-test references
(the E007 regression test comment references it historically as documentation,
which is correct).

**Commit:** `664279d8b` (auto-committed by daemon)

### 2. E009 custom-HTTP suppression regression test (7 frameworks)

The `fileImportsCustomHTTP` helper in `architecture/helpers.go` detects 7
HTTP frameworks (net/http, gin, echo, chi, gorilla/mux, fiber, httprouter)
but had ZERO test coverage. Added `TestE009_CustomHTTPSuppressesFinding` —
a table-driven test with 7 subtests, one per framework. Each subtest:

- Creates source with command + query + one framework import
- Sets `HasTransport: false` (forces the import-based detection path)
- Asserts 0 findings (framework import satisfies transport requirement)

All 7 subtests pass. Covers the entire `fileImportsCustomHTTP` detection matrix.

**File:** `cmd/cqrs-lint/pkg/rules/architecture/e009_permodule_test.go`
**Commit:** `664279d8b`

### 3. Upgraded taskmanager integration test to golden-file finding profile

The prior session's `TestIntegration_TaskmanagerExpectedFindings` was a smoke
test — it logged findings but asserted nothing about counts. Upgraded to a
proper golden-profile regression guard:

- **`taskmanagerGoldenProfile` var** — pins the exact rule→count map
  (16 rules, 31 total findings: A009×1, A032×3, B004×1, B005×1, B028×1,
  C004×1, C009×2, C013×1, C023×3, C026×2, D013×1, E003×1, E005×10,
  E017×1, S010×1, V006×1)
- **NEW finding detection** — any rule not in the golden profile triggers
  a test failure with "not in golden profile" message
- **LOST finding detection** — any rule in the golden that didn't fire
  triggers "a rule may have regressed"
- **Count drift detection** — count mismatch triggers "got N, golden expects M"
- **`CQRS_LINT_UPDATE_GOLDEN=1`** env var — prints the paste-ready profile
  for easy updates
- **No critical-severity findings** — still enforced

**File:** `cmd/cqrs-lint/pkg/rules/integration_test.go`
**Commit:** `664279d8b`

### 4. D018 alias-aware event import resolution

The prior session removed the broad `isNewEventAlt` fallback (any `NewEvent`
selector) and kept only `pkg.Name == "event"` — which broke when consumers
alias the event package (`import ev "go-cqrs-lite/event/v4"` → `ev.NewEvent`).

Added `fileEventQualifiers(gf)` helper that scans a file's import declarations
for any path containing `go-cqrs-lite/event` and records both the default
`event` qualifier and any explicit import alias. Updated `collectEventNewTypes`
to check `eventQualifiers[pkg.Name]` instead of `pkg.Name == "event"`.

Added `TestD018_AliasedEventImport` regression test verifying 0 findings when
`ev.NewEvent("user.created", ...)` matches `catalog.NewBuilder("user.created", ...)`.

**Files:** `d018_d019.go` (helper + updated collector), `d018_d019_test.go` (test)
**Commit:** `f88214a93`

### 5. Ran `nix fmt` on all changed files

Ran `nix fmt` (treefmt). 5 files reformatted:
- `d005_internal_test.go` — formatting alignment
- `d018_d019.go` — formatting alignment
- `integration_test.go` — import grouping
- `convergence_suite.go` — const grouping (`const x; const y` → `const ( x; y )`)
- `transport_test.go` — trailing whitespace removal

All changes verified as formatting-only (no semantic changes).

### 6. Ran `nix run .#verify` — identified and fixed API stability failure

First verify run revealed the api-stability golden was stale: the prior
session's irohengine transport hardening added 5 new exports
(`FrameHeaderSize`, `RunConvergenceSuite`, `StreamsOpenedForPeer`,
`ClusterFactory`, `ErrFrameTooLarge`) that weren't captured in the golden.

Fixed by regenerating: `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update`
→ Updated `docs/api_surface.txt` (3824 exports, up from 3819).

The golden update was auto-committed by the daemon in `341baa999`.

**The full verify gate was started a second time but was killed before
completion** (the user requested this status update). The api-stability test
now passes in isolation. The lint phase passed (0 issues across all modules).
All cqrs-lint tests pass (18 packages green). The race/test/doc-check phases
were not re-verified after the golden fix.

---

## b) PARTIALLY DONE

### Verify gate

- **Build + Vet + Test + Lint**: PASSED (first run, before golden fix)
- **API stability**: FAILED on first run → FIXED (golden regenerated) →
  PASSES in isolation
- **Race + doc-check**: NOT re-verified after the golden fix (second verify
  run was killed to write this report)

---

## c) NOT STARTED

From the prior session's 50-item next-steps list:

1. **Re-validate E007 against 8 consumer repos** — removing package-wide
   suppression may surface new FPs in real consumer code. Not started.
2. **Cut v4.3.1 or v4.4.0 release** — not started. C041 confidence was
   already `ConfidenceMedium` (task 6 from the original list was a no-op).
3. **Apply alias-aware resolution to other rules** — D018 now handles
   `event.NewEvent` aliases, but D019's `collectCatalogTypes` and other
   rules with `pkg.Name == "catalog"` checks have the same alias blind spot.
4. **E009 per-module test for `gorilla/mux` specifically** — covered in
   the table-driven test, but no dedicated test for the `strings.Contains`
   matching path vs exact match path.

---

## d) TOTALLY FUCKED UP

### Nothing this session.

The prior session left dead code (`PackagesWithRegistration`) and stale
golden (`api_surface.txt`), but those were identified and fixed cleanly.

**However:** The auto-commit daemon committed my work across 2 separate
commits with confusing, mixed messages. Commit `664279d8b` bundles 3
unrelated changes (dead code removal, E009 test, irohengine convergence
suite consolidation) into one commit. The daemon also committed dep bumps,
domain-language docs, and status reports from prior sessions into the
same commit chain, making it hard to isolate this session's work.

---

## e) WHAT WE SHOULD IMPROVE

1. **The verify gate takes 3-4 minutes** — the second run was killed
   because of the status update request. We should have a `verify-fast`
   that skips the slow integration tests (stack/postgres 58s, stack/mysql
   64s, benchkit 85s, duckdbengine 74s) and runs only build+vet+lint+unit
   tests. The full `verify` should be reserved for pre-release.

2. **API-surface golden drift is a recurring pattern** — this is the 3rd+
   time the golden has been stale after a prior session's export changes.
   The AGENTS.md already documents "API-surface changes require golden
   regen in the same edit" but sessions keep missing it. The
   `TestEveryGoModDirIsInModulesList` meta-test works well — we need a
   similar pre-commit check that fails if exported symbols exist that
   aren't in the golden.

3. **The daemon's commit messages are misleading** — `664279d8b` says
   "remove unused tracking and consolidate convergence test suites" but
   also adds the E009 test and golden-profile upgrade. The commit body
   is a wall of text that doesn't match the title. We can't control the
   daemon, but we should be aware that `git log --oneline` doesn't tell
   the real story.

4. **D018 alias resolution is asymmetric** — I fixed `collectEventNewTypes`
   but `collectCatalogTypes` still uses `pkg.Name == "catalog"` without
   alias resolution. If a consumer aliases `import cat "catalog"`, D019
   will miss their catalog entries. The same pattern applies to every
   rule that checks `pkg.Name == "<expected>"`.

5. **The golden-profile integration test is fragile by design** — any
   change to taskmanager code or any rule will drift the golden. This is
   intentional (catches regressions), but `CQRS_LINT_UPDATE_GOLDEN=1` is
   not documented anywhere outside the test comment. It should be in
   AGENTS.md under Testing conventions.

---

## f) Up to 50 Things to Get Done Next

### High Priority (verify-gate blocking)

1. **Finish the verify gate** — re-run `nix run .#verify` to completion
   after the golden fix. The race and doc-check phases were not verified.
2. **Run `nix run .#verify-fast`** — if it exists, use it for faster
   iteration. If not, create it (build+vet+lint+unit only, skip PG/MySQL/benchkit).

### cqrs-lint Precision

3. **Apply alias-aware resolution to `collectCatalogTypes`** — same pattern
   as D018: scan imports for `go-cqrs-lite/catalog` and record aliases.
4. **Apply alias-aware resolution to ALL rules using `pkg.Name == "X"`** —
   audit every rule for hardcoded package qualifier checks. Build a shared
   `fileQualifiersFor(gf, importSubstr)` helper.
5. **Re-validate E007 against consumer repos** — removing
   `PackagesWithRegistration` may surface new FPs. Run cqrs-lint against
   browser-history, SEC, and other known consumers.
6. **Test E007 with aliased imports** — `import q "go-cqrs-lite/query/v4"`
   then `q.RegisterTyped[MyQuery, any](...)` — does the per-type tracing
   handle aliases?
7. **C041 confidence audit** — verify C041 is actually `ConfidenceMedium`
   in all code paths (the task said "raise to Medium" but it was already
   there — double-check no code path overrides it).
8. **B029-B031 bus method call test** — the `hasBusMethodCall` helper was
   added but only one test (`TestB029_NoFindingForNonCQRSBus`) covers it.
   Add positive test: a real bus with `Use`/`Publish` calls SHOULD trigger.

### Test Coverage Gaps

9. **D019 alias-aware test** — same as D018 alias test but for
   `collectCatalogTypes`. Will fail until task 3 is done.
10. **E010 type-aware receiver test** — `projectCallsMethodOnType` uses
    type-aware matching but the test uses AST-only `BuildContextFromSource`.
11. **E012 adapter layer test** — `countTypesWithSuffix("Adapter")` with
    threshold 3 — no test for the boundary (exactly 3 vs exactly 2).
12. **E016 health endpoint test** — `fileImportsCustomHTTP` path for
    health/livez endpoints — partially tested but not exhaustive.
13. **C042 zero-version test** — `TestC042_SaveWithZeroVersion` exists but
    doesn't test the `len(call.Args) < 4` guard.
14. **Golden profile for other examples** — `getting-started`,
    `readme-quickstart`, `metaengine-quickstart` have no golden profiles.
15. **Test the `CQRS_LINT_UPDATE_GOLDEN=1` path** — verify it actually
    prints the correct format for pasting.

### Architecture & Design

16. **Extract shared alias-resolution helper** — `fileEventQualifiers` should
    be generalized to `fileQualifiersFor(gf, "go-cqrs-lite/event")` and
    live in `analyzer/` or `lintutil/`.
17. **Document `CQRS_LINT_UPDATE_GOLDEN=1`** in AGENTS.md under Testing.
18. **Add `verify-fast` to flake.nix** — build+vet+lint+unit only.
19. **Pre-commit hook for api-stability golden** — `go test -run
    TestAPISurfaceUpdateIdempotent` before every commit.
20. **Consider snapshot testing (go-snaps) for finding profiles** — instead
    of a hardcoded `map[string]int`, use `snaps.MatchSnapshot` for richer
    diff output.

### cqrs-lint Rules (New)

21. **Rule: detect missing `defer Close()` on `pebble.Open`** — resource leak.
22. **Rule: detect `slices.Backward` mutation bug** — the footgun documented
    in AGENTS.md. Static detection: `for _, v := range slices.Backward(s)`
    followed by `v.X = ...` (mutation on copy).
23. **Rule: detect `IsEventBusType("")` degraded mode** — warn when a
    type-aware rule runs without type info (degraded mode returns true).
24. **Rule: detect missing `WithCodec` on events in CBOR-default stacks** —
    events are self-describing but blind stores need the envelope.
25. **Rule: detect `command.Type` constant mismatch** — C038/C039 cover
    events, but command type constant mismatches are not checked.

### Release & CI

26. **Cut v4.3.1 or v4.4.0** — the precision improvements (D018 alias,
    B029-B031 bus heuristic, E007 per-type) are release-worthy.
27. **Tag all modules consistently** — check `git tag -l '<module>/v4*'`
    for version-sequence breaks (documented AGENTS.md gotcha).
28. **Update CHANGELOG** — the daemon added an irohengine entry but no
    cqrs-lint precision improvement entry.
29. **CI: add cqrs-lint self-lint step** — run `cqrs-lint` on the
    cqrs-lint source itself.
30. **CI: add golden-profile check** — run
    `TestIntegration_TaskmanagerExpectedFindings` in CI.

### Code Quality

31. **Remove unused `allFindings` slice** — the old integration test
    collected `allFindings` but never used it; the new version doesn't
    either (it uses `byRule` map). Verify no leftover.
32. **Consolidate `singleFinding` helpers** — each rule package has its
    own `singleFinding`/`singleInfoFinding` with slightly different
    defaults. Extract to `lintutil`.
33. **Rename `fileEventQualifiers` → `eventImportQualifiers`** — clearer
    intent (it's about import aliases, not file events).
34. **Add doc comment to `taskmanagerGoldenProfile`** — explain why these
    specific counts are expected and when to update.
35. **Add `//nolint:gochecknoglobals` to `taskmanagerGoldenProfile`** —
    it's a test golden, not a production global.

### Metaengine & Irohengine (noticed during verify)

36. **irohengine convergence_suite.go const grouping** — `nix fmt` changed
    `const x; const y` to `const (x; y)`. Verify this is committed.
37. **loopback/transport_test.go trailing whitespace** — `nix fmt` removed
    trailing blank lines. Verify committed.
38. **Dgraph DeleteJson behavior** — the AGENTS.md gotcha about explicit
    null predicates should have a regression test in dgraphengine.
39. **QUIC transport CBOR int normalization** — the `normalizeAny()` helper
    in `quic/latency.go` should have a dedicated test for `uint64`→`int`.

### Documentation

40. **Update AGENTS.md module list** — verify all 79 `go.mod` files are
    listed in the Modules row.
41. **Document the golden-profile pattern** — `taskmanagerGoldenProfile`
    should be referenced in AGENTS.md Testing section.
42. **Add `CQRS_LINT_UPDATE_GOLDEN=1` to AGENTS.md** — Testing conventions.
43. **Status report for this session** — (this file).
44. **Verify the prior session's status report** —
    `2026-08-09_01-26_cqrs-lint-fp-followup-regression-and-precision.md`
    should be cross-referenced.

### Remaining from Prior Session's 50-item List

45. **E005/E007 per-type tracing for generic register helpers** —
    `register[Q]()` wrapper functions still can't be traced.
46. **Type-aware resolution for D018/D019** — use `types.Info` instead of
    AST-only `SelectorPackage` for more precise matching.
47. **Profile-guided rule prioritization** — use `--scorecard` data to
    prioritize which rules to improve next.
48. **Consumer repo CI integration** — run cqrs-lint in consumer repos'
    CI pipelines.
49. **Rule documentation generation** — auto-generate rule docs from
    detector metadata.
50. **Interactive rule explorer** — web UI for browsing cqrs-lint rules.

---

## g) Questions I Cannot Answer Myself

### 1. Should the verify gate be re-run to completion right now?

The second `nix run .#verify` was killed to write this report. The
api-stability test passes in isolation, lint passes, all cqrs-lint tests
pass. But the race detector and doc-check phases were not re-verified
after the golden fix. Should I re-run the full gate (3-4 min) before
this session is considered done, or is the isolated verification sufficient?

### 2. Should I create `verify-fast` in flake.nix?

The full verify gate is expensive (PG 58s, MySQL 64s, benchkit 85s,
duckdbengine 74s). A `verify-fast` (build+vet+lint+unit only) would
speed up iteration significantly. But adding a new nix output requires
understanding the flake structure and may have unintended consequences
(e.g., CI using the wrong target). Should I add it, or is the full gate
the only acceptable verification?

### 3. Should the D018 alias-aware fix be backported to other rules before release?

The alias-aware resolution pattern (`fileEventQualifiers`) fixes D018 but
the same blind spot exists in D019 (`catalog`), and potentially in every
rule that checks `pkg.Name == "<expected>"`. Should I audit and fix ALL
such rules before cutting a release, or ship the D018 fix alone and
address the rest incrementally?
