# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **TODO/ROADMAP/CHANGELOG split**: Extracted roadmap into separate files for better organization

## [0.2.0] - 2026-04-05

### Added

- **Event catalog system** (`catalog/`): Three-layer architecture with reflection-based schema generation, custom YAML marshaler, AsyncAPI and EventCatalog exporters. Packages: `catalog/`, `catalog/asyncapi/`, `catalog/eventcatalog/`, `catalog/yaml/`
- **ID type methods**: `Equal`, `Compare`, `Or`, `Reset`, `GoString`, `Format` (fmt.Formatter), `MarshalBinary`/`UnmarshalBinary`, `MarshalText`/`UnmarshalText`
- **Comprehensive test suites**: `internal/dispatcher/` (0%→100%), `pkg/id/` (48%→88%), `aggregate/` (64%→100%), `xtypes/` (53%→95.6%), `event/` (75%→92.8%), `catalog/` (91.9%), `catalog/asyncapi/` (92.6%)
- **EventCatalog frontmatter**: `schemaPath` in message frontmatter when schema exists; `sends`/`receives` arrays in service frontmatter
- **AsyncAPI functional options**: `WithServer()` and `WithDescription()` for configurable server name, host, and protocol
- **Integration test**: full E2E flow (Registry → Build → AsyncAPI + EventCatalog export)
- **Benchmarks**: `Registry.Build`, `SchemaFromType`, AsyncAPI `Export`/`MarshalYAML`, EventCatalog `Export`

### Changed

- **YAML marshaler**: struct fields now preserve definition order instead of alphabetical sorting (maps still sorted for determinism)
- **AsyncAPI**: `toSnakeCase` renamed to `toDotAddress` for truthful naming (produces `dot.separated`, not `snake_case`)
- JSON serialization: zero-value IDs now marshal to `null` instead of empty string; `UnmarshalJSON` supports both `null` and string values
- Dispatcher closed-check now returns `ErrHandlerNotFound` with descriptive wrapping instead of silently returning `nil`

### Fixed

- EventCatalog: `schemaPath` now correctly references `schemas/schema.json` in message frontmatter
- EventCatalog: service frontmatter now includes `sends`/`receives` message lists (required for EventCatalog rendering)
- YAML marshaler: struct field ordering preserved (was sorting alphabetically)
- CI workflows: updated Go version matrix from 1.21/1.22/1.23 to 1.26
- Dispatcher lifecycle: `Register()` and `Dispatch()` on closed dispatcher now correctly return errors (was no-op due to `CheckClosed(nil)`)
- Lint: replaced `[]byte(fmt.Sprintf(...))` with `fmt.Appendf`, used tagged switch on `msg.Kind`

### Infrastructure

- **GitHub Actions CI**: Created `.github/workflows/test.yml` and `.github/workflows/lint.yml` with Go 1.26 version matrix
- **Parallel tests**: Added `t.Parallel()` to all test functions across the codebase

## [0.1.0] - 2026-01-01

### Added

- Initial release
