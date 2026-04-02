# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **Event catalog system** (`catalog/`): Three-layer architecture with reflection-based schema generation, custom YAML marshaler, AsyncAPI and EventCatalog exporters. Packages: `catalog/`, `catalog/asyncapi/`, `catalog/eventcatalog/`, `catalog/yaml/`
- **ID type methods**: `Equal`, `Compare`, `Or`, `Reset`, `GoString`, `Format` (fmt.Formatter), `MarshalBinary`/`UnmarshalBinary`, `MarshalText`/`UnmarshalText`
- **Comprehensive test suites**: `internal/dispatcher/` (0%→100%), `pkg/id/` (48%→88%), `aggregate/` (64%→100%), `xtypes/` (53%→95.6%), `event/` (75%→92.8%)

### Changed

- JSON serialization: zero-value IDs now marshal to `null` instead of empty string; `UnmarshalJSON` supports both `null` and string values
- Dispatcher closed-check now returns `ErrHandlerNotFound` with descriptive wrapping instead of silently returning `nil`

### Fixed

- CI workflows: updated Go version matrix from 1.21/1.22/1.23 to 1.26
- Dispatcher lifecycle: `Register()` and `Dispatch()` on closed dispatcher now correctly return errors (was no-op due to `CheckClosed(nil)`)

### Security

## [0.1.0] - 2026-01-01

### Added

- Initial release
