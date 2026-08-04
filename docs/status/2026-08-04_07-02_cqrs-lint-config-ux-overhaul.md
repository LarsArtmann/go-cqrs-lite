# Status Report: cqrs-lint Config UX Overhaul

**Date:** 2026-08-04 07:02
**Session goal:** Make `.cqrs-lint.json` a first-class, well-documented, commentable config format. Fix critical bugs. Add an `explain` subcommand and overhaul `doctor`.

---

## a) FULLY DONE

### 1. JSONC Support (comments in config)
- **`config_loader.go`** (133 lines): `JSONCLoader` implements `cmdguard.ConfigFileLoader`. Strips `//` line comments and `/* block */` comments before JSON parsing, respecting string literals.
- Wired into `main.go` via `cmdguard.WithConfigFileLoader(JSONCLoader{}, ".cqrs-lint.json")` — replaces the koanf-based loader.
- All direct config readers in `diagnostics.go` (`loadRawRulesJSON`, `loadParentRulesConfig`) updated to strip comments via `loadConfigFileBytes()`.
- **6 tests** in `config_loader_test.go` covering: line comments, block comments, URLs in strings, `//` inside string literals, no-comments baseline, escaped quotes.
- **Verified end-to-end**: a `.cqrs-lint.json` with `//` and `/* */` comments parses correctly and all values load into the config struct.

### 2. `cqrs-lint explain` Subcommand (new)
- **`explain.go`** (468 lines): comprehensive interactive reference generated from type definitions.
- Sections: CONFIG FILE (location, format, JSONC example), TOP-LEVEL KEYS (table with key/type/default/description), PRESETS (all 4 with descriptions, pinned features, disabled rules, severity floor, and minimal JSON snippet), FEATURES (all 10 flags with valid values), RULES (all 4 config keys with examples), HEALTH (info-cap), CONFIG RESOLUTION ORDER (5-step priority chain, severity floor semantics, monorepo inheritance), SUPPRESSION SYNTAX (inline, block, project-wide).
- Registered in `main.go` and added to root help text.
- **Verified**: `cqrs-lint explain` produces well-formatted output.

### 3. `cqrs-lint doctor` Overhauled
- **`doctor.go`** (441 lines, rewritten): structured into `renderDoctor*` functions.
- **Sections now shown**:
  1. Raw config file content (path, byte count, full content with indentation)
  2. Parent config inheritance chain (monorepo support)
  3. Active preset with full description, pinned features, disabled rules, severity floor
  4. **EFFECTIVE SETTINGS** — resolved min-severity with source annotation (`default` / `config` / `preset floor`), min-confidence, format, color, preset, total/active/disabled rule counts with per-source breakdown (from preset vs from config), rules overrides, health config
  5. Feature profile with annotation about pinned vs auto-detected
  6. Per-module profiles (multi-module workspaces)
  7. Suggested `.cqrs-lint.json` (copy-pasteable features section)
  8. Inline suppression counts (sorted by frequency, with coaching note)
- **Verified end-to-end** against this repo (`{"preset": "library"}`) and a synthetic `/tmp/jsonc-test` project.

### 4. G1 Bug Fixed (preset MinSeverity dead code)
- **`run.go`**: `resolveMinSeverity(presetMinSeverity, userMinSeverity)` applies the preset floor as a lower bound. Users can raise it (stricter) but cannot go below the preset floor.
- **6 tests** in `run_severity_test.go`: preset floor wins, user stricter wins, equal stays, no preset floor, both empty, critical vs warning.
- **Verified end-to-end**: `{"preset":"local-cli","min-severity":"info"}` correctly resolves to `warning (preset floor)` in `doctor` output.

### 5. `cqrs-lint init` Now Generates Commented Configs
- **`init.go`** (120 lines, rewritten): `defaultConfigTemplate()` and `presetConfigTemplate(preset)` produce hand-formatted JSONC with `//` comments explaining every setting.
- Preset configs include the description from `presetDescriptions` and a comment explaining the severity floor semantics.
- **Tests updated** (`init_test.go`): tests now strip comments before unmarshaling, verifying the JSONC output is valid.

### 6. Documentation Updated
- **README.md**: quickstart updated with `explain`/`doctor` commands and JSONC note; Configuration Reference section mentions JSONC support; Config File section rewritten with JSONC example and pointers to `explain`/`doctor`.
- **CHANGELOG.md**: `[Unreleased]` section updated with all session 2 changes (JSONC, explain, doctor overhaul, severity floor fix, init comments).
- **`.cqrs-lint.json`** (library root): updated to showcase JSONC comments.
- **Example configs** (`example/taskmanager/`, `example/getting-started/`): updated with comments.
- **API stability golden** (`docs/api_surface.txt`): regenerated to include `JSONCLoader` and `Load` method.

### Build & Test Status
- `go build -tags "goexperiment.jsonv2" ./...` — **PASS**
- `go test -tags "goexperiment.jsonv2" -count=1 . ./pkg/analyzer/...` — **PASS** (all 15+ test functions)
- `go vet -tags "goexperiment.jsonv2" ./...` — **PASS**
- `cmd/api-stability` — **PASS** (3186 exports verified)
- Pre-existing C009 test failure in `pkg/rules/correctness` — **NOT OUR CHANGE** (was failing before session start)

---

## b) PARTIALLY DONE

### 1. Doctor test coverage
- The doctor command was completely rewritten but has **zero unit tests** for the new output. The old doctor had no tests either, but the new version is significantly more complex (8 render functions, config file reading, preset resolution, rule source breakdown, parent config discovery).
- The `renderDoctor*` functions all write to stdout via `fmt.Print*`, making them hard to test without refactoring to accept an `io.Writer`. The explain functions have the same problem.

### 2. Explain test coverage
- `explain.go` (468 lines) has **zero tests**. The `renderExplain()` function is pure (returns a string), so it could be tested for: non-empty output, presence of all preset names, presence of all top-level keys, presence of "JSONC" mention, etc.

### 3. JSONC loader edge cases
- The `stripJSONComments` function handles the common cases but has edge cases that could theoretically break:
  - Trailing comma after a comment-stripped line (we don't fix trailing commas — JSONC spec allows them but our parser doesn't)
  - Unicode in comments (should work but untested)
  - Very deeply nested `/* /* nested */ */` block comments (the parser is non-greedy and would stop at the first `*/`)

---

## c) NOT STARTED

1. **Doctor `--json` output mode** — for programmatic consumption (CI tooling, dashboards). Not requested but would be a natural extension.
2. **`cqrs-lint explain <key>` drill-down** — e.g. `cqrs-lint explain preset` shows only the preset section. Currently `explain` dumps everything.
3. **Config validation on init** — `cqrs-lint init` could validate the generated config by round-tripping it through the loader.
4. **Config file schema documentation as JSON Schema** — for editor autocomplete in `.cqrs-lint.json`.
5. **Trailing comma support in JSONC** — the JSONC spec allows trailing commas; our parser does not.
6. **Version bump** — the `version` constant in `main.go` is still `"4.3.0"` and no new tag has been created for these changes.

---

## d) TOTALLY FUCKED UP

### 1. Auto-commit daemon interleaved unrelated changes
- The auto-commit daemon committed **concurrently** with our work, producing commits with empty messages (`f25f7400`, `b537de55`, `e16f5499`, `15e9ead6`) that mix our cqrs-lint changes with unrelated benchmark test changes (memory-storage, decider, integration, metaengine calibration benches). These empty-message commits make the git history noisy and hard to follow. The `5702c5af` commit titled "feat" also mixes our README/CHANGELOG/init changes with metaengine redesign docs.
- **Impact**: The cqrs-lint config UX changes are spread across 5+ interleaved commits with no clean boundary. A consumer reading `git log -- cmd/cqrs-lint/` sees a mix of unrelated benchmark and metaengine changes.

### 2. No integration test for the full config → doctor round-trip
- We verified `doctor` and `explain` work by running the binary manually, but there is no automated test that: creates a temp `.cqrs-lint.json` → runs `doctor` → asserts the output contains expected sections. This means a future refactor could break the doctor output without any test catching it.

### 3. `explain.go` has 468 lines of string concatenation
- The entire explain output is built from `b.WriteString("...")` and `fmt.Fprintf(&b, ...)` calls. This is fragile and hard to maintain. If a preset changes, someone has to manually update both `PresetDefinitions` and the explain output (though explain does read from `PresetDefinitions`, the formatting is hardcoded). A table-driven approach or template would be more maintainable.

---

## e) WHAT WE SHOULD IMPROVE

1. **Test the doctor output** — refactor `renderDoctor*` functions to accept `io.Writer` instead of writing to stdout directly. Then write tests that capture the output and assert key sections are present.
2. **Test the explain output** — `renderExplain()` returns a string; trivial to test for non-empty, presence of all presets, presence of "JSONC", etc.
3. **Trailing comma support** — JSONC (as used by VS Code, TypeScript, etc.) allows trailing commas. Our parser doesn't. Adding it would make hand-editing less error-prone.
4. **Explain drill-down** — `cqrs-lint explain presets` or `cqrs-lint explain rules` would be more useful than dumping everything.
5. **Doctor `--json` mode** — machine-readable doctor output for CI integration.
6. **Config file watch mode** — `cqrs-lint doctor --watch` re-runs detection on file changes.
7. **Version bump and tag** — these changes are significant enough to warrant a `v4.4.0` tag.
8. **Clean up empty-message commits** — consider a `git rebase -i` to squash the interleaved commits (requires user approval — destructive git operation).

---

## f) Up to 50 Things to Get Done Next

### Testing (HIGH PRIORITY)
1. Refactor `renderDoctor*` functions to accept `io.Writer` parameter for testability
2. Write doctor output test: assert CONFIG FILE section, ACTIVE PRESET section, EFFECTIVE SETTINGS section, FEATURE PROFILE section, suppression counts
3. Write doctor test with preset: verify severity floor annotation `(preset floor)` appears
4. Write doctor test with parent config: verify parent config inheritance chain is shown
5. Write doctor test with no config file: verify "(not found)" message
6. Write explain output test: assert non-empty, contains "JSONC", contains all 4 preset names, contains all top-level key names
7. Write integration test: create temp `.cqrs-lint.json` with comments → verify `doctor` reads it
8. Write test: `cqrs-lint init` output passes through `stripJSONComments` and loads into `AppConfig`
9. Add test for `JSONCLoader.Load` directly (not just `stripJSONComments`): pass bytes with comments, verify struct population + setFields return
10. Add test for trailing block comment at EOF (no closing `*/`)

### JSONC Enhancements
11. Add trailing comma support to `stripJSONComments` (strip trailing commas before `}` and `]`)
12. Add test for trailing comma after comment-stripped line
13. Consider switching to a proper JSONC library (e.g. `github.com/titanous/jsonc`) if edge cases proliferate

### Explain Improvements
14. Add `cqrs-lint explain <section>` drill-down (presets, features, rules, health, resolution-order, suppressions)
15. Add `--json` output to explain for programmatic consumption
16. Add color to explain output (respect `--color` flag)
17. Add config example with ALL keys populated (for reference)
18. Add "Common Patterns" section to explain (e.g. "Migrating from bare JSON", "Monorepo setup")

### Doctor Improvements
19. Add `doctor --json` output mode for CI dashboards
20. Add rule-category breakdown to effective settings (how many active per category)
21. Show config file modification time in doctor output
22. Add validation warnings to doctor (e.g. "preset local-cli disables 5 rules — verify this is intended")
23. Show auto-detected features that DIFFER from preset-pinned features (highlight conflicts)
24. Add `doctor --diff` to compare current config vs detected profile

### Init Improvements
25. Add `cqrs-lint init --interactive` wizard (prompts for preset, severity, rules)
26. Add `cqrs-lint init --force` to overwrite existing config
27. Show a preview of the config before writing
28. Add `--features` flag to init for pinning specific features

### Documentation
29. Update `CONTRIBUTING.md` to document the JSONC loader and explain command
30. Add a "Config Cookbook" to README (5-10 common config patterns with explanations)
31. Generate a JSON Schema for `.cqrs-lint.json` for editor autocomplete
32. Add VS Code extension recommendation in README (for JSONC syntax highlighting)
33. Document the `JSONCLoader` as a public type in the module's API docs

### Config System
34. Add `cqrs-lint config validate` subcommand (validate without running lint)
35. Add config migration command for breaking changes (`cqrs-lint config migrate`)
36. Add `--config-path` flag to specify a non-default config file location
37. Support environment variable expansion in config values (`$HOME`, etc.)
38. Support config includes (`{"include": "base.json"}`) for shared monorepo configs

### Release
39. Bump version constant from `4.3.0` to `4.4.0`
40. Create `cmd/cqrs-lint/v4.4.0` annotated tag
41. Update `TestVersionMatchesLatestTag` CI gate
42. Regenerate api-stability golden after version bump (if needed)
43. Run full `nix run .#verify` gate before tagging

### Cleanup
44. Review whether the interleaved benchmark test changes (from the auto-commit daemon) are correct
45. Verify the `module_catalog.go` untracked file is not our responsibility
46. Check if `go.mod`/`go.sum` changes in `cmd/cqrs-lint/` are from our work or the daemon
47. Consider whether `explain.go` should be split into `explain.go` + `explain_data.go` (data tables vs rendering)
48. Run `nix fmt` on all changed files
49. Run `nix run .#lint` on the cqrs-lint module
50. Run `nix run .#verify` as the final gate

---

## g) Questions (3 max — things I CANNOT figure out myself)

### Q1: Should doctor and explain write to an `io.Writer` instead of stdout?
Currently every `renderDoctor*` function calls `fmt.Print*` directly, making them untestable without capturing stdout. Refactoring to accept `io.Writer` would make them testable but changes the function signatures significantly. This is an architectural choice that affects every future test.

**Options:**
- A) Refactor all render functions to accept `io.Writer` (clean, testable, more work)
- B) Leave as-is, test via running the binary in a subprocess (fragile, slow)
- C) Extract pure string-returning functions and have thin stdout wrappers (compromise)

### Q2: Should we add trailing comma support to the JSONC parser?
The JSONC spec (as used by VS Code, TypeScript, esprima) allows trailing commas. Our parser doesn't strip them, so `{"preset": "library",}` fails. This is a common edit mistake. But adding it increases parser complexity and there's a risk of over-engineering for a config format.

### Q3: Should the interleaved auto-commit-daemon commits be cleaned up via rebase?
The git history has empty-message commits and commits mixing cqrs-lint changes with unrelated benchmark/metaengine changes. A `git rebase -i` could squash these into clean topic commits. But this is a destructive git operation (rewrites history) and per the AGENTS.md rules, requires explicit user approval.
