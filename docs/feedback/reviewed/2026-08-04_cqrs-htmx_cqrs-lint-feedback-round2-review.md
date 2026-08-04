# Review: cqrs-htmx cqrs-lint Feedback (Round 2)

**Source feedback:** [`new/2026-08-04_cqrs-htmx_cqrs-lint-feedback-round2.md`](../new/2026-08-04_cqrs-htmx_cqrs-lint-feedback-round2.md)
**Date reviewed:** 2026-08-04
**Outcome:** All 4 actionable issues addressed (1 bug + 3 design/detector improvements). Issue 4 is a consumer-side config adoption step (no code change needed beyond the preset improvement in issue 3).

---

## Issue 1: End-of-line suppressions silently ignored — FIXED

**Severity:** HIGH (DX-breaking, silent failure)
**Status:** ✅ Fixed

**Root cause confirmed:** `ParseSuppressions` used `strings.HasPrefix(line, commentPrefix)`, so a trailing comment after code (`x := sdk.X //cqrs-lint:ignore(A008)`) never matched.

**Fix:** `pkg/suppression/parser.go` — replaced the `HasPrefix` guard with a comment-aware scan: locate the line's Go comment (first `//` outside a string literal) and require the directive at the START of that comment's text. This recognizes end-of-line comments (`code //cqrs-lint:ignore(A008)`) while rejecting two classes of false match that an anywhere-in-line search would catch: (1) doc/example strings (`fmt.Println("//cqrs-lint:ignore(RULE)")` — the `//` is inside a string) and (2) godoc comments that merely mention the syntax (`// see the //cqrs-lint:ignore docs` — the directive is not at the comment's start). Both `//cqrs-lint:` and `// cqrs-lint:` (space) variants work, as do comma-separated rule lists.

**Propagate:** the fix flows through stale-suppression detection (inline + block + unknown-rule paths, all unified on the same comment-aware scan) automatically.

**Tests added:** `TestParseSuppressions_EndOfLine`, `TestParseSuppressions_EndOfLine_SpaceAfterSlashes`, `TestParseSuppressions_EndOfLine_CommaSeparated`, `TestNewSuppressionFilter_EndOfLineCommentInFile`.

**Help text:** the `--help` line "Place inline suppressions on the line above the code or at end of line" is now accurate (it was previously lying).

---

## Issue 2: Feature profile is workspace-wide, not per-module — FIXED

**Severity:** MEDIUM (significant for multi-module workspaces)
**Status:** ✅ Fixed

**Root cause confirmed:** `DetectFeatures` scanned ALL packages/files across every `go.mod` into one merged `FeatureProfile`. An `examples/` app's `ListenAndServe` set `server=true` for the library module.

**Fix:** per-module feature detection in `pkg/analyzer/`.

- `BuildContext` now tags each `GoFile` with its `ModuleDir` and records packages per module.
- `DetectFeaturesPerModule(ctx, packagesByModule)` partitions files/packages by module and runs detection once per module → `ctx.FeatureProfiles map[string]FeatureProfile`.
- The **primary** (root) module's profile is exposed via `ctx.FeatureProfile` for global detectors and `doctor`. It is NO LONGER a cross-module merge, so example modules no longer pollute the library profile.
- `ctx.ProfileForFile(path)` resolves the correct profile for a given file's module (longest-prefix match), used by per-file detectors.
- `run.go` resolves each module profile with the same config/preset overrides.
- `doctor` prints a per-module breakdown when more than one module is present.
- `C017` (the one per-file detector that read `FeatureProfile.Store` inside its file loop) now evaluates the store backend per-module via `ProfileForFile`.

**Single-module projects are unchanged** (no `FeatureProfiles` → `ProfileForFile` falls back to the primary profile).

**Tests added:** `TestDetectFeaturesPerModule_SeparatesLibraryAndExample` (verifies an example's server flag does NOT leak into the library profile), `TestProfileForFile_ResolvesByLongestPrefix`, `TestBuildContext_SingleModuleUnchangedByPerModule`.

**Note:** most detectors read the profile GLOBALLY (top-level of the detector function), so they now use the primary module's profile. This is the correct behavior for a library workspace: the library module's profile drives global detectors, and example/demo modules get their own per-file evaluation. Full per-module evaluation of every global detector is a larger future refactor; the primary-profile approach addresses the core complaint (cross-module leakage) without destabilizing 15+ detectors.

---

## Issue 3: The `library` preset doesn't go far enough — FIXED

**Severity:** LOW (design improvement)
**Status:** ✅ Fixed

**Fix:** `pkg/analyzer/feature_profile.go` — the `library` preset's `Rules.Disable` list now also includes:

| Rule | Why it's a library false-positive                                                 |
| ---- | --------------------------------------------------------------------------------- |
| F002 | Library defines event types but doesn't own the catalog (consumer registers them) |
| F006 | Library defines PII payloads but the consumer configures encryption               |
| F010 | Library offers hierarchical queries; graph traversal is the consumer's choice     |
| F011 | Library does multi-table reads; relational projection is a deployment choice      |
| S002 | Library cannot force encryption on consumers (same as F006)                       |
| S003 | Library creates events without signing; consumer wires signing middleware         |

`F009` and `S007` were NOT added because they already self-skip under the preset's `server=false` feature flag (verified via their guards) — adding them would be redundant.

**Test added:** `TestPresetLibrary_DisablesAdoptionAndSecurityFalsePositives`.

**Impact:** a library consumer using `{"preset": "library"}` no longer needs to suppress these 6 rules one-by-one (≈25 suppressions saved per library workspace).

---

## Issue 4: Missing `.cqrs-lint.json` — consumer-side (no code change)

**Status:** N/A — this is the consumer adopting the config system. The preset improvement in issue 3 makes `{"preset": "library"}` do the right thing with maximal rule coverage. No cqrs-lint code change needed.

---

## Issue 5: B025 helper-function tracing — FIXED

**Severity:** detector improvement (LOW impact, ~4 findings in the reporting project)
**Status:** ✅ Fixed

**Root cause confirmed:** B025 only inspected the direct arguments to `decider.NewRepository`. When options were built by a helper (`repositoryOptions[State](cfg)...`), the detector couldn't see the `WithStateCache` inside the helper.

**Fix:** `pkg/rules/boilerplate/b022_b025.go` — the detector now:

1. Builds a function-name → `*ast.FuncDecl` index of all top-level functions in the analyzed package.
2. When `NewRepository` receives a variadic spread from a function call, extracts the helper name (handling generic instantiation `foo[T]()` and selector `pkg.foo()`).
3. Inspects the helper's body for a `WithStateCache` call; if found, suppresses the finding.

This is conservative: it only suppresses when the helper _visibly_ constructs the option. Helpers that DON'T wire the cache still fire (verified by test). Cross-package helpers fall through to the current behavior (fire at low confidence).

**Tests added:** `TestB025_NoFindingWithStateCacheViaHelper`, `TestB025_NoFindingWithStateCacheViaGenericHelper` (the exact cqrs-htmx generic pattern), `TestB025_FiresWhenHelperLacksStateCache`.

---

## What was NOT changed (and why)

- **Issue 5 suggestion "lower confidence for opaque spreads":** the current B025 already fires at `ConfidenceLow` (0.25), not "full confidence", so the lowering ask is already met. The real fix (helper tracing) is implemented instead.
- **Per-module evaluation of every global detector:** would require restructuring ~15 detectors from "fire once" to "fire per module". The primary-profile approach resolves the reported leakage without that risk. Tracked as a future refinement if a consumer hits a remaining cross-module false positive.
- **`examples/` exclusion / `demo` preset:** not implemented — the `library` preset + per-module profiles already eliminate the bulk of example-driven noise, and an exclusion would hide real issues in example code that doubles as integration documentation.

---

## Verification

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./... -race` — all packages green (suppression, analyzer, all rule categories)
- New tests: 14 added across suppression parser (6), stale detection (1), feature-profile (1), per-module detection (3), B025 helper tracing (3)
- Self-lint of `cmd/cqrs-lint` on its own source: no stale/unknown-rule false warnings (previously the help-text strings mentioning the suppression syntax were falsely flagged)
