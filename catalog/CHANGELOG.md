# Changelog — catalog/v4

All notable changes to the `catalog/v4` module are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added — REST / OpenAPI operation support

- **`catalog.MsgOperation(method, path, statusCodes...)`** — message option that
  attaches an explicit HTTP operation (method + path) to any command, query, or
  event. Exporters use this to emit the real REST path instead of the derived
  default.
- **`catalog.Operation` struct** — holds `Method`, `Path`, and optional
  `StatusCodes`. Set via `MsgOperation` or directly on `Message.Operation`.
- **`catalog.Response[T](statusCode, description)`** — typed response spec that
  derives a JSON schema from the generic type `T`. Exporters render these as
  typed response bodies instead of fabricated defaults.
- **`catalog.ResponseSpec` struct** — `StatusCode`, `Description`, `Schema`,
  `Examples`. Carried on `Message.Responses`.
- **`catalog.SecurityScheme` + `MsgSecurity(schemeIDs...)`** — declare API-key
  or bearer-token security schemes at the catalog level and attach them to
  individual messages.
- **`catalog.Parameter` + `Schema.Parameters`** — explicit path/query/header
  parameter extraction with `In`, `Required`, `Description`, and `Schema` fields.
- **OpenAPI exporter** — maps operations to `paths`, responses to typed content,
  security schemes to `components.securitySchemes`, entity schemas to
  `components.schemas`.
- **AsyncAPI exporter** — operations with `http:GET` / `http:POST` tags appear
  as HTTP-channel messages alongside traditional event channels.
- **D2 exporter** — operation labels render as `[POST /api/users]` on edges.
- **`httptyped` package** — generic `RequestSchema[T]`, `ResponseSchema[T]`,
  `ToResponseSpec`, and convenience helpers `OKResponse[T]`, `CreatedResponse[T]`,
  `ErrorResponse[T]` for consumers who want type-safe HTTP contracts.
- **`huma` package** — adapter that converts Huma-style operation metadata
  (`HumaOperation`) into catalog `MessageConfig` entries via `ToMessages`.
- **`catalog.Validate()`** — checks for duplicate `(method, path)` operations,
  method-without-path errors, and 2xx responses without body schemas.
- **`cmd/go-cqrs-lite-catalog` CLI** — generates OpenAPI, AsyncAPI, D2, and
  llms.txt from a catalog definition.

### Changed

- **`openapi/exporter.go`** split into `exporter.go` (core, 262 lines) and
  `exporter_rest.go` (REST helpers, 260 lines) — both under the 350-line limit.
- **`validateOperation`** — response schema checks now run regardless of whether
  `Operation` is set (previously early-returned, silently skipping response
  validation for messages without an explicit operation).

### Fixed

- Stale golden fixtures regenerated for json/v2 compact array formatting.
- `writeLLMsTxt` output ordering corrected — per-service command/query lists now
  appear under their respective service headers.

## [v4.0.0] — 2026-06-28

Initial public release of the catalog module.
