# Status Report: .cqrs-lint.json Design Review & Polish

> **Date:** 2026-08-04 05:42
> **Scope:** Sessions 1+2 — full `.cqrs-lint.json` config system review
> **Verdict:** Architecture significantly improved. One runtime gap found. Tests incomplete.

---

## a) FULLY DONE (working, tested, verified)

### Session 1 — Architecture (all green, tests pass)

| # | What | Status |
|---|------|--------|
| 1 | Unified `PresetDefinitions` map (single source of truth for features + rules + severity) | ✅ Done, tested |
| 2 | Rewrote `init.go` to generate configs programmatically from struct definitions | ✅ Done, tested |
| 3 | Preset rule-disable defaults applied at runtime (union with explicit config) | ✅ Done, tested |
| 4 | Preset-name validation (warns on typos like `"server"`, `"prod"`) | ✅ Done, tested |
| 5 | Disabled-rule-ID validation (warns on unknown IDs like `"C999"`) | ✅ Done, tested |
| 6 | Fixed stale known-keys warning (was missing `c008-ignore-structs`) | ✅ Done |
| 7 | Created `.cqrs-lint.json` for the library itself (`{"preset":"library"}`) | ✅ Done |
| 8 | `ResolvePresetDefinition`, `IsKnownPreset`, `ValidPresetNames` exported + tested | ✅ Done, tested |
| 9 | Updated README preset docs with full table (features + rules + severity floor) | ✅ Done |
| 10 | Updated CHANGELOG with session 1 changes | ✅ Done |

### Session 2 — Polish (mostly green, see gaps below)

| # | What | Status |
|---|------|--------|
| 11 | CONTRIBUTING.md: 6→10 categories, correct rule ID ranges, `AllRules()` ref | ✅ Done |
| 12 | CONTRIBUTING.md: added preset/FeatureProfile guidance to contributor flow | ✅ Done |
| 13 | Doctor shows active preset name + resolved rule disables + severity floor | ✅ Compiles, untested |
| 14 | README config example includes `domain: "financial"` with explanation | ✅ Done |
| 15 | `HasAsyncBus` wired into `String()`, `ConfigFeatures`, `ToConfigFeatures()` | ✅ Compiles, untested |
| 16 | Full "Configuration Reference" section in README (all keys with type/default) | ✅ Done |
| 17 | VALIDATION_REPORT.md marked as HISTORICAL SNAPSHOT | ✅ Done |
| 18 | IMPROVEMENT_IDEAS.md marked as HISTORICAL IDEAS BACKLOG | ✅ Done |
| 19 | Example `.cqrs-lint.json` in taskmanager (production) + getting-started (local-cli) | ✅ Done |
| 20 | Planning doc with mermaid graph at `docs/planning/` | ✅ Done |
| 21 | All commits pushed to origin/master | ✅ Done |

---

## b) PARTIALLY DONE (started but incomplete)

| # | What's partial | What's missing |
|---|----------------|----------------|
| P1 | Doctor preset display (code written, compiles) | **No test** — plan said "B3: Add test for doctor preset output", skipped |
| P2 | HasAsyncBus config wiring (code written, compiles) | **No round-trip test** — Transport and ServerLocal have override tests, AsyncBus doesn't |
| P3 | Preset severity floor (`MinSeverity` in `PresetDefinition`) | Written into init config and doctor output, but **NOT applied at runtime** — see gap G1 below |

---

## c) NOT STARTED (planned but never began)

| # | What |
|---|------|
| N1 | Running `cqrs-lint doctor` against the actual library to verify the output looks right |
| N2 | Running `cqrs-lint` against the example projects to verify their `.cqrs-lint.json` works |
| N3 | Adding `.cqrs-lint.json` to `example/readme-quickstart/` |
| N4 | CHANGELOG entry for session 2 changes (doctor preset display, async-bus wiring, config reference, etc.) |
| N5 | Updating AGENTS.md cqrs-lint module description with the new `async-bus` config key |

---

## d) TOTALLY FUCKED UP (bugs, regressions, mistakes)

### G1: 🔴 CRITICAL — Preset MinSeverity is dead code at runtime

**The bug:** `PresetDefinition.MinSeverity` (e.g. `"warning"` for `local-cli`) is:
- ✅ Written into the init config by `generateInitConfig` → becomes `"min-severity": "warning"` in `.cqrs-lint.json`
- ✅ Shown in `doctor` output
- ❌ **NOT applied at runtime** when someone writes just `{"preset": "local-cli"}` without the `min-severity` key

**Root cause:** In `applyConfigOverrides` (run.go), I merge `presetDef.Rules.Disable` but I never apply `presetDef.MinSeverity` to `cfg.MinSeverity`. So the preset's severity floor is a recommendation baked into the init template only — if you set the preset without generating via init, it silently doesn't apply.

**Impact:** A user who writes `{"preset": "local-cli"}` manually gets `min-severity: "info"` (the default), not `"warning"` as the preset promises. The preset table in the README claims `local-cli` has severity floor `warning`, but it doesn't actually work at runtime.

**Fix needed:** In `applyConfigOverrides`, after resolving the preset definition, apply the severity floor: if `cfg.MinSeverity` is empty/default AND `presetDef.MinSeverity` is set, use the preset's value. But only as a floor — if the user explicitly sets a higher severity, that wins.

### G2: 🟡 MODERATE — Never executed the binary

I never ran `cqrs-lint doctor` or `cqrs-lint` against any real project this session. All verification was `go build` + `go test`. The doctor preset display code could have a formatting bug, wrong output order, or nil-pointer issue that only shows at runtime.

### G3: 🟡 MODERATE — Pre-existing build break not flagged

The build was broken at session start (`escape/v0.36.0` doesn't exist). I silently downgraded to v0.35.0 to unblock myself, but this is a symptom of the auto-commit daemon bumping deps to non-existent versions (documented in AGENTS.md). I should have flagged this more prominently.

### G4: 🟢 MINOR — Unexplained HTML file modification

`docs/architecture-understanding/2026-08-04_metaengine-goal-gap.html` appeared as modified in git status. I didn't touch it — likely modified by the pre-commit hook or auto-commit daemon. I ignored it, which is correct per the "respect existing changes" rule, but I should have investigated what changed.

---

## e) WHAT WE SHOULD IMPROVE (honest self-critique)

1. **I claim "done" without running the binary** — I wrote doctor.go changes and verified they compile + existing tests pass, but never ran the actual command. This is the "stale GREEN" anti-pattern documented in AGENTS.md.

2. **I skip tests from my own plan** — The plan explicitly listed "B3: Add test for doctor preset output" and I didn't write it. I need to follow my own plans.

3. **I introduced dead code** — `PresetDefinition.MinSeverity` is a field that doesn't do anything at runtime. This is exactly the kind of "looks done but isn't" that erodes trust.

4. **No integration test for the full config chain** — There's no test that loads a `.cqrs-lint.json` with `{"preset":"library"}`, runs the linter, and verifies the preset's feature flags + rule disables + severity floor are all applied. Every piece is unit-tested in isolation, but the chain isn't verified end-to-end.

5. **CHANGELOG is stale for session 2** — I updated it in session 1 but forgot in session 2, so the doctor display, async-bus wiring, config reference, and example configs are undocumented in the changelog.

6. **I wrote example configs without verifying them** — `example/taskmanager/.cqrs-lint.json` says `{"preset":"production"}` but I never ran cqrs-lint against taskmanager to verify the preset resolves correctly.

---

## f) Up to 50 things we should get done next

### Critical (fix the bugs from this session)

| # | Task | Est |
|---|------|-----|
| 1 | **Fix G1:** Apply `presetDef.MinSeverity` as a floor in `applyConfigOverrides` when user hasn't explicitly set it | 15min |
| 2 | **Fix P1:** Write test for doctor preset output (verify preset name, disabled rules, severity floor appear) | 15min |
| 3 | **Fix P2:** Write round-trip test for `AsyncBus` config override (ConfigFeatures → ResolveFeatureProfile → ToConfigFeatures) | 10min |
| 4 | Run `cqrs-lint doctor` against the library itself and verify output | 5min |
| 5 | Run `cqrs-lint` against `example/taskmanager/` and verify the preset works | 10min |
| 6 | Update CHANGELOG with session 2 changes | 10min |

### High value (completeness)

| # | Task | Est |
|---|------|-----|
| 7 | Add integration test: load `.cqrs-lint.json` with preset → verify full resolution chain (features + rules + severity) | 30min |
| 8 | Add `.cqrs-lint.json` to `example/readme-quickstart/` | 5min |
| 9 | Update AGENTS.md cqrs-lint description to mention `async-bus` config key | 5min |
| 10 | Verify `HasAsyncBus` is actually consulted by F015/F016/F017 (the only rule that uses it) — confirm the wiring is meaningful | 10min |
| 11 | Add `domain` to the doctor output's suggested config section (verify it appears when domain is detected) | 5min |

### Medium value (polish)

| # | Task | Est |
|---|------|-----|
| 12 | Add `LookupRule` to CONTRIBUTING.md alongside `AllRules()` | 2min |
| 13 | Consider whether `HasAsyncBus` should be in any preset definition (currently no preset sets it) | 5min |
| 14 | Document the config inheritance feature (parent `.cqrs-lint.json` merging) in the README Configuration Reference | 10min |
| 15 | Add a `--dry-run` note to the init command (what would it write?) | 5min |
| 16 | Consider adding `cqrs-lint init --list-presets` to show available presets with descriptions | 15min |
| 17 | Add JSON schema for `.cqrs-lint.json` (for IDE autocompletion) — may be YAGNI | 20min |
| 18 | Verify the `info-cap` health config is documented in the Configuration Reference (it is, but verify accuracy) | 5min |
| 19 | Check if `exclude-rules` CLI flag should be documented in the Configuration Reference (currently only in CLI flags) | 5min |
| 20 | Consider whether `rules.disable` should validate against `consumerOnlyRules` (library self-lint) to warn when disabling a rule that's already auto-suppressed | 10min |

### Lower value but worth considering

| # | Task | Est |
|---|------|-----|
| 21 | Add a `cqrs-lint config` subcommand that prints the resolved config (preset + features + rules + health) as JSON | 30min |
| 22 | Add `--validate-config` flag that ONLY validates `.cqrs-lint.json` without linting (fast feedback) | 20min |
| 23 | Consider config hot-reload for watch mode (if `--watch` is ever added) | — |
| 24 | Add migration guide if config keys are ever renamed (versioned config schema) | — |
| 25 | Consider whether `.cqrs-lint.json` should support environment variable interpolation (e.g. `"min-severity": "${LINT_SEVERITY}"`) | — |
| 26 | Add `cqrs-lint init --force` to overwrite an existing config | 5min |
| 27 | Consider `cqrs-lint init --print` to write to stdout instead of a file (for piping) | 5min |
| 28 | Add tests for `loadParentRulesConfig` (config inheritance) — currently only tested indirectly | 15min |
| 29 | Document the `encoding/json/v2` + `MatchCaseInsensitiveNames` loading behavior (config keys are case-insensitive) | 5min |
| 30 | Add a test that verifies every preset in `PresetDefinitions` produces valid JSON via `generateInitConfig` (round-trip) | 10min |

### Rule/data quality

| # | Task | Est |
|---|------|-----|
| 31 | Audit all 185 rules for accurate severity (some may be miscategorized) | 2h |
| 32 | Verify the rule count in README (says 185) matches `AllRules()` actual count | 5min |
| 33 | Check if any rules reference `HasAsyncBus` but should also check `HasTransport` or `HasServer` | 15min |
| 34 | Audit `consumerOnlyRules` list — is it complete? Are there rules that should be auto-suppressed for libraries? | 30min |

### Documentation

| # | Task | Est |
|---|------|-----|
| 35 | Write a "Getting Started with cqrs-lint" guide for consumers (separate from README) | 1h |
| 36 | Add "Common Configurations" examples to README (e.g. CI, pre-commit, monorepo) | 30min |
| 37 | Document the suppression syntax more prominently (currently buried in root command help) | 15min |
| 38 | Add a "Rule Severity Explained" section (what info/warning/error/critical mean for exit codes) | 15min |
| 39 | Document the health-score formula in the README (currently only in code) | 20min |
| 40 | Add inline links from rule tables to detailed rule docs (if they exist) | 30min |

### Architecture

| # | Task | Est |
|---|------|-----|
| 41 | Consider extracting config types into a separate `config` package (currently spread across main.go + analyzer) | 1h |
| 42 | Evaluate whether `PresetDefinition` should be in the analyzer package or a dedicated `presets` package | 15min |
| 43 | Consider making `FeatureProfile` fields consistent (some are `Kind` enums, some are `bool`) | 30min |
| 44 | Evaluate whether the doctor command should be testable without loading real packages (mock context) | 30min |

### Testing

| # | Task | Est |
|---|------|-----|
| 45 | Add snapshot/golden test for doctor output format | 20min |
| 46 | Add test that verifies `validatePresetName` and `validateDisabledRuleIDs` output goes to stderr (not stdout) | 10min |
| 47 | Add test for `generateInitConfig` output format (indentation, trailing newline, key ordering) | 10min |
| 48 | Add fuzz test for `RulesConfig.Validate` (arbitrary JSON input should never panic) | 15min |
| 49 | Add test that config inheritance (`loadParentRulesConfig`) handles circular symlinks gracefully | 10min |
| 50 | Add benchmark for `applyConfigOverrides` (should be <1ms for any config) | 10min |

---

## g) Questions I CANNOT answer myself

### Q1: Should preset `MinSeverity` override or floor the user's explicit setting?

If a user writes `{"preset": "local-cli", "min-severity": "error"}`, should the result be:
- **(a)** `"warning"` (preset wins — presets are opinionated defaults) 
- **(b)** `"error"` (user wins — explicit config always overrides presets)
- **(c)** `"error"` (max of the two — preset is a floor, user can go higher but not lower)

I assumed (c) in my plan but you may want (b) to match how feature flags work (explicit always wins).

### Q2: Is the `HasAsyncBus` config override actually useful to consumers?

`HasAsyncBus` is only consulted by F015/F016/F017 (adoption coaching rules). No consumer has ever asked to override it. Adding it to `ConfigFeatures` makes the config surface larger for a flag that may never be used. Should I keep it (completeness) or revert it (YAGNI)?

### Q3: Should `cqrs-lint init` write different configs for the SAME preset based on what it detects?

Currently `init --preset local-cli` always writes the same `{"preset":"local-cli"}`. An alternative: `init` could detect the project's features (like `doctor` does) and write a fully-resolved config with all features pinned. This would be more explicit but less DRY. Which philosophy do you prefer?
