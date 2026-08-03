# Changelog

All notable changes to cqrs-lint are documented here.
Tags use the full module path: `cmd/cqrs-lint/vX.Y.Z`.

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
