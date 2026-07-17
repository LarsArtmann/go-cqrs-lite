# cqrs-lint: Feature-Profile Hardening — Brutal Self-Review & Status

**Date:** 2026-07-17 03:04
**Session scope:** Follow-up hardening of the feature-profile system (bug fix, suppression tests, cleanup, DetectFeatures precision tests)
**Baseline:** 184 tests → **211 tests** (0 failing). Build + vet clean.

---

## a) FULLY DONE (Complete and Verified)

### 1. Doctor JSON trailing-comma bug — FIXED

- **Root cause:** `doctor.go` hand-formatted JSON with `fmt.Printf` and hardcoded commas. When `tracing`/`snapshot` were unknown (not printed), the output had a trailing comma after the last printed field → invalid JSON.
- **Fix:** Replaced hand-formatted JSON with `FeatureProfile.ToConfigFeatures()` + `encoding/json.MarshalIndent`. Structurally incapable of producing trailing commas. Added a zero-value guard (`StoreKind` zero value is `""`, not `"unknown"` — the initial version missed this and the test caught it).
- **New method:** `FeatureProfile.ToConfigFeatures()` — the natural inverse of `ResolveFeatureProfile`. Projects a resolved profile back into a `ConfigFeatures`, including only meaningful (non-unknown) values.
- **Verified:** Ran `cqrs-lint doctor --path .` against cqrs-lint itself. Output parses as valid JSON via `json.Unmarshal`.

### 2. ToConfigFeatures regression tests — 3 tests, 6 subtests

- `TestToConfigFeatures_OmitsUnknownFields` — all-unknown profile omits everything except server/soft-delete
- `TestToConfigFeatures_IncludesKnownFields` — fully-detected profile includes all 6 flags
- `TestToConfigFeatures_JSONAlwaysValid` — marshal+unmarshal roundtrip across 6 profile shapes, no trailing comma

### 3. FeatureProfile suppression guards — 7 tests

Each reuses a fixture that WOULD fire, then proves the profile gate suppresses it. Together with the existing positive tests, they pin the toggle in both directions.

| Test                                 | Rule | Gate            | Proves                                            |
| ------------------------------------ | ---- | --------------- | ------------------------------------------------- |
| `TestS002_DowngradedForLocalCLI`     | S002 | `HasServer`     | Error → Info severity toggle                      |
| `TestS003_SuppressedForNoServer`     | S003 | `HasServer`     | 0 findings                                        |
| `TestA015_SuppressedForNoServer`     | A015 | `HasServer`     | 0 findings                                        |
| `TestB014_SuppressedForNoServer`     | B014 | `HasServer`     | 0 findings                                        |
| `TestA016_SuppressedForReadOnlyFlow` | A016 | `CommandFlow`   | 0 findings                                        |
| `TestA012_SuppressedForNoSoftDelete` | A012 | `HasSoftDelete` | 0 findings                                        |
| `TestA009_AdaptiveSuggestion`        | A009 | `Store`         | Store → matching `stack/` suggestion (3 backends) |

### 4. DetectFeatures precision tests — 4 tests

- `TestDetectFeatures_HTTPServer` — `http.ListenAndServe` → HasServer=true
- `TestDetectFeatures_GRPCServer` — `grpc.NewServer` → HasServer=true
- `TestDetectFeatures_Tracing` — otel import + `EventTracing` middleware → Tracing=on
- `TestDetectFeatures_Snapshot` — `WithSnapshotStore` → Snapshot=on

### 5. Cleanup

- Removed dead `ptr[T any]` helper + `//go:fix inline` directive (zero usages, cleared ~13 gopls hints).
- Removed two stale "has been replaced by" comment blocks in `s002_s003.go` and `a009_a013.go`.
- Added per-preset doc comments to `PresetLocalCLI`/`PresetProduction`/`PresetLibrary`/`PresetReadOnly`.

### 6. Corrected finding: `new(false)` is valid Go

The prior session's status report listed `new(false)` as a code smell and a "next step" to fix. **This was wrong.** Empirically verified: `new(expr)` returns a pointer initialized to the expression's value (`*new(false) == false`). Valid in Go 1.26. The presets were already correct and idiomatic. The actual issue was the dead `ptr` wrapper that duplicated this — now removed.

---

## b) PARTIALLY DONE (Has Gaps or Concerns)

### 1. Formatting was fixed retroactively — NOT proactively

I ran `gofmt -l` only when writing this self-review, and it flagged `feature_profile_test.go` for struct field misalignment (the `HasSoftDelete` field is longer than `HasServer`, so gofmt wanted wider alignment). I fixed it, but **I should have run `gofmt -w` or `nix fmt` BEFORE declaring P4-1 (verification) complete.** My "full verification" step was incomplete — it checked build/vet/test but not formatting.

### 2. golangci-lint was NOT run until the self-review

I only ran `go vet` during the implementation. `go vet` is much weaker than the project's golangci-lint suite (golines, gosec, revive, tagliatelle, exhaustive, gochecknoglobals, gocyclo, tparallel, etc.). When I finally ran `golangci-lint` during this self-review, it found **1 issue in my code** (`TestToConfigFeatures_JSONAlwaysValid` subtests missing `t.Parallel()` — the `tparallel` linter). I fixed it, but it should never have shipped to the self-review stage.

### 3. Doctor command is only tested at the method level, not the subprocess level

`TestToConfigFeatures_JSONAlwaysValid` tests the `ToConfigFeatures()` method + JSON marshaling. But nobody tests the actual `doctor.go` command handler — the closure that calls `BuildContext`, `ToConfigFeatures`, and `fmt.Println`. If someone rewires doctor.go to bypass `ToConfigFeatures`, the method tests still pass but the command output could break again. A subprocess-level test (`exec.Command("cqrs-lint", "doctor", "--path", tmpDir)` → parse stdout as JSON) would close this gap.

### 4. Stale gopls diagnostics were never cleared

Every tool output throughout the session showed ~13 `newexpr` hints referencing `ptr(x)` — even after I deleted the `ptr` function. These are stale LSP cache entries. `go build` passes, `rg` confirms no `ptr` references remain, but the diagnostics polluted every file view. I should have restarted the LSP (`lsp_restart`) to get clean diagnostics and confirm there are no real issues hiding behind the noise.

---

## c) NOT STARTED

### 1. Integration test against a real consumer project

No test runs `cqrs-lint` (or `cqrs-lint doctor`) against `example/taskmanager/` or `example/getting-started/` to verify detection works on real code, not just synthetic fixtures. All tests use `BuildContextFromSource` with hand-written Go snippets.

### 2. `nix run .#lint` was never run

The project's canonical lint command (`nix run .#lint`) runs golangci-lint across all modules. I ran golangci-lint manually on just the cqrs-lint module with `GOWORK=off`. The full nix lint target might catch cross-module issues or use different config flags.

### 3. `nix fmt` was never run

The canonical formatter (`nix fmt`) runs `treefmt` which includes `gofmt`, `golines` (line shortening to 120 chars), and possibly `goimports`. I only ran `gofmt -w` manually. `golines` might shorten long lines I didn't catch.

### 4. Pre-existing lint findings from the prior session

`golangci-lint` found 8 issues that predate my work (all from commit `1b6d6c32`):

- `a009_a013.go:43` — exhaustive switch missing StoreMemory/StoreTurso/StoreNone/StoreUnknown cases
- `feature_profile.go:130` — `Presets` global variable (gochecknoglobals)
- `feature_detect.go:12` — cyclomatic complexity 44 > 30 (gocyclo)
- `main.go:54` — omitempty on nested struct (modernize)
- `main.go:34` — unused nolint directive (nolintlint)
- `doctor.go:17` — unused parameter `ctx` (revive)
- `feature_profile.go:91,93` — tagliatelle kebab-case JSON tags (deliberate UX choice, but flagged)

I did NOT fix these — they're in code from the prior session, not mine. But they are in files I touched.

### 5. LSP restart to verify diagnostics

Never called `lsp_restart` to clear the stale `ptr` hints. Can't confirm the new code is truly clean from the LSP's perspective.

---

## d) TOTALLY FUCKED UP (Mistakes and Regressions)

### 1. I declared "P4-1: Full verification" complete without running the formatter or linter

My verification step checked `go build`, `go vet`, `go test`, and doctor JSON. It did NOT check:

- `gofmt -l` (formatting) → would have caught the struct misalignment
- `golangci-lint run` (full lint) → would have caught the missing `t.Parallel()`

I called it "full verification" when it was really "partial verification." This is the exact failure mode the AGENTS.md warns about: "Quality Gates — Static analysis passes, Type checking passes." I skipped static analysis.

### 2. The initial `ToConfigFeatures` had a zero-value bug that the test caught

My first version of `ToConfigFeatures` checked `fp.Store != StoreUnknown` but the zero value of `StoreKind` (a `string` type) is `""`, not `"unknown"`. So a freshly-constructed `FeatureProfile{}` had `Store=""` which passed the `!= StoreUnknown` check and was wrongly included in the config suggestion. The test (`TestToConfigFeatures_OmitsUnknownFields`) caught this immediately. Good testing discipline, but the bug existed because I didn't think about Go's zero-value semantics for string-based enum types.

### 3. I said "211 tests pass" before fixing formatting and lint

The test count was correct, but "pass" was premature — the unformatted code and missing `t.Parallel()` were latent issues I didn't surface until forced to self-review. The "DONE" declaration was overconfident.

### 4. I never restarted the LSP to clear stale diagnostics

Every single tool output for the entire session showed ~13 stale `newexpr` hints about a `ptr` function I deleted. These made it impossible to distinguish real diagnostics from noise. I should have restarted the LSP after deleting `ptr` to get a clean signal.

---

## e) WHAT WE SHOULD IMPROVE (Architecture and Design)

### 1. The verification gate must include `gofmt -l` and `golangci-lint run`

"Full verification" means build + vet + test + **fmt + lint**. The AGENTS.md quality gates list "Static analysis passes" and "Type checking passes" — I only did the first two. This should be a non-negotiable checklist before any "DONE" declaration.

### 2. Doctor command needs a subprocess-level test

Method-level tests (`ToConfigFeatures`) are necessary but not sufficient. A test that invokes `cqrs-lint doctor` as a subprocess and parses stdout as JSON would catch wiring regressions that method tests miss. Pattern: write a temp Go project to a temp dir, run `doctor --path tmpdir`, assert stdout contains valid JSON.

### 3. `DetectFeatures` cyclomatic complexity is 44 (> 30 threshold)

The function does two passes (import-based + AST-based) with inline switch/if chains. It works and is readable, but golangci-lint flags it. Extracting sub-detectors (`detectStore`, `detectServer`, `detectCommandFlow`, `detectTracing`, `detectSnapshot`) would bring each under the threshold and improve testability.

### 4. The exhaustive switch in A009 needs a default case or explicit handling

`switch ctx.FeatureProfile.Store` handles SQLite/Postgres/Pebble/Custom but not Memory/Turso/None/Unknown. Adding Memory/Turso cases (suggesting `stack/memory` / `stack/turso`) would be more complete; a default case would silence the linter.

### 5. String-based enums need zero-value awareness

`StoreKind`, `CommandFlowKind`, `TracingKind`, `SnapshotKind` are `type X string` with `Unknown` constants. But Go's zero value for these is `""`, not `"unknown"`. This bit me in `ToConfigFeatures`. Either: (a) add `IsUnknown()`/`IsMeaningful()` methods that check both `""` and the explicit constant, or (b) document the invariant that detection always sets a non-empty value (which it does, but `ToConfigFeatures` must still guard against zero values from uninitialized structs).

### 6. Presets as a global map vs. a function

`gochecknoglobals` flags `var Presets = map[...]`. Converting to `func GetPreset(name ConfigPreset) (ConfigFeatures, bool)` would satisfy the linter and prevent accidental mutation of the map at runtime.

---

## f) Up to 50 Things to Get Done Next

### Immediate fixes (in code I touched this session)

1. Run `nix fmt` on the full repo to catch any golines/goimports issues I missed
2. Run `nix run .#lint` to verify the full lint suite passes
3. Restart the LSP (`lsp_restart`) to clear stale `ptr` diagnostics and verify clean signal
4. Run `nix run .#verify` (the one-command gate: build + vet + test + race + lint + doc-check + doc-assertions)

### Pre-existing lint findings (from prior session, in files I touched)

5. Fix A009 exhaustive switch — add Memory/Turso/None/Unknown cases or a default
6. Refactor `DetectFeatures` to split cyclomatic complexity 44 → sub-functions under 30
7. Convert `Presets` global map to a function (`GetPreset`) to satisfy `gochecknoglobals`
8. Fix `main.go:54` `omitempty` on nested struct → use `omitzero` or remove
9. Clean up unused `nolint:tagalign,golines` directive in `main.go:34`
10. Decide on `doctor.go:17` unused `ctx` parameter — rename to `_` or use it
11. Address tagliatelle kebab-case JSON tags (`command-flow`, `soft-delete`) — either add nolint or switch to camelCase

### Testing gaps

12. Write subprocess-level doctor test (`exec.Command` → parse JSON stdout)
13. Write integration test: `cqrs-lint doctor` against `example/taskmanager/`
14. Write integration test: `cqrs-lint doctor` against `example/getting-started/`
15. Add `TestDetectFeatures_StoreSQLite` (import `stack/sqlite` → Store=SQLite)
16. Add `TestDetectFeatures_StorePostgres` (import `stack/postgres` → Store=Postgres)
17. Add `TestDetectFeatures_StorePebble` (import `stack/pebble` → Store=Pebble)
18. Add `TestDetectFeatures_StoreCustom` (import `storage/` → Store=Custom)
19. Add `TestDetectFeatures_CommandFlowSync` (NewDispatcher but no Dispatch → Sync)
20. Add negative test: `TestDetectFeatures_NoServer` (no ListenAndServe/grpc → HasServer=false)
21. Add test: `TestResolveFeatureProfile_AllFlagsOverridden` (every field overridden by config)

### FeatureProfile system improvements

22. Add `FeatureProfile.IsUnknown()` helper methods for each field (zero-value safety)
23. Add `FeatureProfile.Validate()` — check for contradictory flags (e.g., ReadOnly + Commands)
24. Wire A017 to optionally consult `FeatureProfile.Snapshot` (downgrade severity if SnapshotOff)
25. Add suppression explanation to `--verbose` output ("S003 suppressed: HasServer=false")
26. Add `--explain` flag to individual findings showing which profile flags applied
27. Add `Tracing` detection for `WithStdoutExporter` and `prometheus.Setup`
28. Add `Snapshot` detection for `WithSnapshotStrategy` and `NewStateCache`
29. Consider `Idempotency` as a feature flag (detect `CommandIdempotency`/`EventIdempotency`)
30. Consider `Projection` as a feature flag (detect projection host usage)

### Documentation

31. Document the feature-profile system as an ADR (`docs/adr/00XX-feature-profiles.md`)
32. Add migration guide for consumers with existing `.cqrs-lint.json` without `features`
33. Update `cmd/cqrs-lint/README.md` rule table to mark feature-profile-aware rules
34. Add examples of `cqrs-lint doctor` output in README (for different project types)
35. Document the `new(value)` Go idiom for boolean pointers in CONTRIBUTING.md

### CI and tooling

36. Add CI job running `cqrs-lint doctor` on example projects to verify detection
37. Add CI job running cqrs-lint against itself (self-lint)
38. Add SARIF output enrichment: include `profile` metadata in SARIF reports
39. Add `--profile` CLI flag (overrides config file preset)
40. Add `cqrs-lint init --preset local-cli` to generate config with preset pre-filled

### Naming and polish

41. Rename `globalCandidate.origName` → `globalCandidate.Name` (flagged in prior session)
42. Rename `CommandTypesRegistered` → `RegisteredHandlerTypes`
43. Rename `IsCommandRegistered` → `IsHandlerRegistered`
44. Add doc comment to `Presets` map explaining nil = "leave as auto-detected"
45. Consider `DataSensitivity` flag (none/pii/financial) for better S002 targeting

### SDK integration (deferred from original proposal)

46. Ship `stack.WithEncryptionFromEnv` — one-liner encryption wiring
47. Ship `stack.WithObservability` — one-liner OTel wiring
48. Ship `stack.WithSigningFromEnv` — one-liner signing wiring
49. Ship `encryption.GenerateKey()` — first-run key generation helper
50. Add `cqrs-lint init` PII detection — suggest `"data": "pii"` if PII fields found

---

## g) Questions (3)

### Q1: Should the doctor output be a complete `.cqrs-lint.json` file or just the `features` fragment?

Currently it prints `{"features": {...}}` — a JSON object wrapping just the features section. A user can't directly redirect this to `.cqrs-lint.json` because the file needs other top-level keys too (path, min-confidence, etc.). Should doctor print a complete config file (with sensible defaults for all fields), or keep printing just the features fragment as a suggestion? I cannot determine the preferred UX without knowing how users consume the output.

### Q2: Should I fix the pre-existing golangci-lint findings (items 5-11) in files I touched, or leave them for the prior session's author?

The lint findings (exhaustive switch, global Presets map, gocyclo, tagliatelle, etc.) are all from commit `1b6d6c32` — the prior session's work. They're in files I modified this session. The AGENTS.md says "Fix issues in files you changed" but also "Don't fix unrelated bugs." These are pre-existing issues that my changes exposed but didn't cause. Should I fix them as part of this hardening pass, or leave them for a dedicated cleanup session?

### Q3: Should `DetectFeatures` be refactored to satisfy gocyclo (< 30) now, or is the current single-function approach acceptable?

`DetectFeatures` has cyclomatic complexity 44 (threshold 30). It's readable and well-structured (two clear passes), but the linter flags it. Extracting sub-detectors would fix the lint finding and improve unit testability (each sub-detector testable in isolation). But it adds 5+ new functions for what is currently one cohesive scan. Is the complexity finding worth the refactor, or should I add a `//nolint:gocyclo` directive with a justification?

---

## Session Metrics

| Metric                    | Value                                                     |
| ------------------------- | --------------------------------------------------------- |
| Plan tasks                | 18 (P0-P4)                                                |
| Tasks completed           | 18/18                                                     |
| Tests before              | 184                                                       |
| Tests after               | 211 (+27 including subtests)                              |
| Tests failing             | 0                                                         |
| Files changed             | 8 code + 1 status doc                                     |
| Lines added               | ~450                                                      |
| Lines removed             | ~35                                                       |
| Bug fixed                 | 1 (doctor JSON trailing comma)                            |
| Cleanup items             | 3 (dead ptr helper, 2 dead comment blocks)                |
| Lint issues in my code    | 2 (gofmt alignment, tparallel) — both fixed retroactively |
| Lint issues pre-existing  | 8 (from prior session, not fixed)                         |
| Verification steps missed | 2 (gofmt, golangci-lint) — caught in self-review          |
| Known bugs remaining      | 0 in code I wrote                                         |
