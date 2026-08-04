# Changelog

All notable changes to cqrs-lint are documented here.
Tags use the full module path: `cmd/cqrs-lint/vX.Y.Z`.

## [Unreleased]

### Fixed

- **Preset severity floor now applied at runtime** — `PresetDefinition.MinSeverity` (e.g. `"warning"` for `local-cli`) was defined but never applied. A user writing `{"preset":"local-cli"}` now correctly gets `min-severity: "warning"` instead of the default `"info"`. The floor is a lower bound: users can raise it (e.g. `"error"`) but cannot go below the preset floor.
- **End-of-line suppressions now work** — `//cqrs-lint:ignore(RULE)` comments placed at the end of a code line (e.g. `EventType = sdk.EventType //cqrs-lint:ignore(A008) re-export`) are now recognized. Previously the parser required the line to START with the comment prefix, so trailing suppressions were silently ignored despite the help text advertising "at end of line" support. Both `//cqrs-lint:` and `// cqrs-lint:` variants work end-of-line, including comma-separated rule lists. (cqrs-htmx feedback round 2, issue 1)
- **Suppression parser ignores doc strings and godoc mentions** — the parser now locates the line's Go comment (first `//` outside a string literal) and requires the directive at the START of that comment's text. This means documentation strings (`fmt.Println("//cqrs-lint:ignore(RULE)")`) and godoc comments that merely mention the syntax (`// see the //cqrs-lint:ignore docs`) are no longer mistaken for real suppressions or flagged as stale. Applies to inline, block, and unknown-rule detection.
- **Preset split-brain eliminated** — the `init` command and runtime no longer maintain separate preset definitions. Previously `init` had `server`+`full-stack` presets (silently ignored at runtime) while the runtime had `production`+`read-only` (not generatable by init). Now both read from a single `PresetDefinitions` map
- **Stale known-keys warning** — the unknown-rules-key warning now lists all 4 valid keys (was missing `c008-ignore-structs`)

### Added

- **JSONC support for `.cqrs-lint.json`** — config files now support `//` line comments and `/* block comments */`. Comments are stripped before JSON parsing, so you can document every setting inline. The config loader uses a custom `JSONCLoader` (via `cmdguard.WithConfigFileLoader`) that replaces the previous koanf-based loader.
- **`cqrs-lint explain` subcommand** — a comprehensive interactive reference that documents every config key, type, default, valid values, all presets with their full definitions, all feature flags, all rules config keys, health config, config resolution order, and suppression syntax. No Go project required — it's pure documentation.
- **`cqrs-lint init` generates commented configs** — generated configs now include JSONC comments explaining each setting, so users immediately see that comments are supported and understand what each key does
- **Per-module feature profiles** — in a multi-module workspace (multiple `go.mod` files), cqrs-lint now detects the feature profile once PER MODULE instead of merging every module into one. An `examples/` app's `ListenAndServe` no longer flips `server=true` for the library module, and an example's SQLite import no longer triggers SQLite-specific rules on library packages. The primary (root) module's profile drives global detectors and `doctor`; per-file detectors resolve each finding's module via the new `AnalysisContext.ProfileForFile(path)`. `doctor` prints a per-module breakdown when more than one module is present. Single-module projects are unchanged. (cqrs-htmx feedback round 2, issue 2)
- **`library` preset disables adoption-coaching + security-middleware false positives** — the `library` preset now also disables `F002`, `F006`, `F010`, `F011` (adoption rules that coach the CONSUMER to adopt catalog/encryption/graph/relational features the library cannot force) and `S002`, `S003` (encryption/signing middleware the consumer wires at the bus boundary). A library defines types and infrastructure; it cannot mandate how its consumers deploy them. (cqrs-htmx feedback round 2, issue 3)
- **B025 traces option-builder helpers** — `decider.NewRepository(s, b, d, helper(cfg)...)` no longer reports a missing state cache when the helper function (defined in the same package, including generic instantiations like `repositoryOptions[State](cfg)...`) constructs `WithStateCache`. The detector builds a function-name index of the analyzed package and inspects helper bodies for the option call. Genuine missing-cache cases (helpers without `WithStateCache`) still fire. (cqrs-htmx feedback round 2, issue 5)
- **Preset validation** — unknown preset names (typos like `"prod"` or stale names like `"server"`) now produce a warning listing valid presets, instead of silently doing nothing
- **Disabled-rule-ID validation** — rule IDs in `rules.disable` that don't match any known rule now produce a warning, catching typos like `"C99"` and references to removed rules
- **`.cqrs-lint.json` for the library itself** — the repo root now carries `{"preset": "library"}` so self-linting is explicit and reproducible
- `PresetDefinition` type, `PresetDefinitions` map, `ResolvePresetDefinition`, `IsKnownPreset`, `ValidPresetNames` in the analyzer package
- `DetectFeaturesPerModule`, `AnalysisContext.FeatureProfiles`, `AnalysisContext.ProfileForFile`, `GoFile.ModuleDir` in the analyzer package

### Improved

- **`doctor` completely overhauled** — now shows: (1) the raw config file content with path and byte count, (2) active preset with full description and features/rules/severity details, (3) effective settings after all resolution with source annotations (default vs config vs preset floor), (4) total/active/disabled rule counts with per-source breakdown (from preset vs from config), (5) parent config inheritance chain, (6) per-module profiles, (7) suggested config, (8) inline suppression counts sorted by frequency
- **Severity floor semantics documented** — the preset's `min-severity` is a lower bound; users can raise it (stricter) but not lower it below the preset floor. This is now visible in `doctor` output (source annotation) and `explain` output (resolution order section)
- **init generates configs programmatically** — no more hardcoded JSON string templates; `init` now generates from struct definitions and preset descriptions, eliminating trailing-comma corruption risk and format drift
- **init writes DRY configs** — named presets write just `{"preset": "name"}` (the runtime resolves features + rule defaults); the default skeleton writes only 3 core knobs instead of 7 no-op zero-value keys
- **Preset rules applied at runtime** — presets now control both feature flags AND rule-disable defaults; explicit `rules.disable` entries are added on top (union)
- **init help text** — `--preset` help now lists the correct presets: `local-cli, production, library, read-only` (was `local-cli, library, server, full-stack`)
- **doctor per-module view** — `doctor` now prints each module's feature profile separately when the workspace has more than one module

### Added (post-v4.3.0)

- **Scorecard subcommand** — `cqrs-lint scorecard` (or `--scorecard`) prints a bilateral module-adoption scorecard: "Adoption: X/Y relevant modules (Z%)" with Used/Missing/Irrelevant tables and top-3 recommendations. Uses a hand-curated ModuleCatalog (28 scored + 6 core modules) with profile-relative denominators.
- **Group-by aggregate** — `--group-by aggregate` stamps `Finding.Metadata["aggregate"]` from event type prefixes and decider/fold state types, then groups output by aggregate (most issues first). Also supports `--group-by module` and `--group-by none` (default).
- **C038 rewritten** — now detects near-miss event type strings (fold-case typos) using normalization + edit distance. Catches `"user.created"` emitted vs `"UserCreated"` in fold — silent event-drop bug.
- **C039** — goroutine-leak-in-handler: unmanaged goroutine inside event/command handler.
- **C040** — dead-fold-case: fold switch case handles an event type that is never emitted — dead code or a typo.
- **Per-module feature detection** — `ProfileForFile` resolves each finding's module in multi-module workspaces. C017 migrated to per-module evaluation.

## [4.3.0] - 2026-08-03

### Fixed

- **Version constant corrected** from `"0.2.2"` to `"4.3.0"` — aligned with the v4 release track for the entire v4.x series
- **TLS detection** — `NewListener`/`Listen` now correctly gated on `tls` package import; `net.Listen` no longer falsely triggers TLS detection
- **`ListenAndServeTLS` now sets `HasServer=true`** — previously only `ListenAndServe` and `Serve` triggered server detection
- **ConfigFeatures gap** — `Transport` and `ServerLocal` fields added to `ConfigFeatures`, completing the config override round-trip for all `FeatureProfile` fields
- **`c008-ignore-fields` case-insensitivity** — config entries are now lowercased before comparison, matching the documented behavior
- **`/livez` added to E016 health-endpoint recognition** — was recognized by feature detection but missing from E016's suppression list

### Added

- **`version --verbose` subcommand** — shows Go version, OS/arch, and module path
- **`changelog` subcommand** — prints `git log` since the last release tag
- **`c008-ignore-structs` config** — skip entire structs from C008 (float64-for-money) detection
- **`--adoption` flag** — shows F-series adoption coaching but excludes them from the health score
- **`--strict-load` flag** — exit non-zero if any packages failed to load during analysis

### Improved

- **E016 health-endpoint scan narrowed** — only matches string literals that are arguments to routing function calls (`HandleFunc`, `Handle`, `Mount`, `Get`, `Post`, etc.), preventing false positives from health-related strings in comments or variable assignments
- **F015 store gating** — metaengine suggestion now suppressed for `StoreMemory` and `StorePebble` in addition to `StoreSQLite`
- **Server detection** — `ListenAndServeTLS`, `tls.Listen`, and `net.Listen` now correctly set `HasServer=true`
- **Release process** — documented in `CONTRIBUTING.md` with a checklist and `scripts/bump-cqrs-lint.sh` helper
- **Version-tag CI gate** — `TestVersionMatchesLatestTag` verifies the version constant matches the latest `cmd/cqrs-lint/v*` git tag

### Tests

- ConfigFeatures Transport/ServerLocal override + round-trip tests
- TLS detection precision tests (`tls.Listen`, `net.Listen`, `ListenAndServeTLS`)
- C016 shutdown-proximity boundary tests (5 lines suppressed, 6 lines fires)
- C008 case-insensitive ignore-fields and new ignore-structs tests
- E016 `/livez` endpoint suppression and narrowed scan test
- F013 transport/grpc import suppression test
- F015 StoreMemory and StorePebble gating tests
- Health-score adoption mode integration test
- Version format tests (local, with commit, with both, verbose)
