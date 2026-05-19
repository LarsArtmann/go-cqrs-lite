# Code Quality Scan — Session 74

**Date:** 2026-05-19 | **Modules:** 11 | **Go Files:** 277 | **LOC:** 43,136

## Build Status

| Check | Result |
|-------|--------|
| `nix run .#build` | PASS |
| `nix run .#test` | PASS (23/23 packages) |
| `nix run .#lint` | 1 issue (golines in core/event/event_test.go:396) |

## Critical Issues

### 1. testhelpers v1.1.0 Incompatible with Current Core (CRITICAL)

The published `testhelpers v1.1.0` tag uses `int` for version parameters, but `core` now requires `event.Version` (branded type). Modules building in isolation (GOWORK=off) fail:

- `core/event` — build fails (testhelpers returns `int`, core expects `event.Version`)
- `core/aggregate` — build fails (same)
- `core/decider` — build fails (same)
- `example/todo` — build fails (storage API drift + same testhelpers issue)

**Root cause:** `event.Version` type change (Session 65) was not accompanied by a `testhelpers` version bump.

**Fix:** Bump `testhelpers` to v1.2.0 with updated signatures. Then bump all consumers' go.mod.

### 2. example/todo — Build Failures (HIGH)

Multiple compilation errors due to storage/core API drift:
- `storage` outbox_helpers.go: `event.Version` → `int` mismatch
- `storage` transactional_store.go: `SaveWithOutbox` signature changed (added `Outbox` param)
- `storage` helpers.go: `event.SchemaVersion` undefined
- `storage` pebble_serialization.go: `.Int()` method missing on `int` type

**Fix:** Update `example/todo` and `storage` to use current core API. The storage module itself passes tests through the workspace — the issue is that `example/todo` references a stale storage.

## File Size Violations (>250 lines)

### Production Code

| Lines | File | Status |
|-------|------|--------|
| 330 | `example/todo/cmd/api/main.go` | VIOLATION |
| 284 | `core/event/event.go` | VIOLATION |

### Test Code (>250 lines, 47 files)

Top 10 by size:

| Lines | File |
|-------|------|
| 1146 | `core/decider/decider_test.go` |
| 1057 | `projection/runner_test.go` |
| 993 | `core/pkg/id/id_test.go` |
| 884 | `storage/event_store_test.go` |
| 874 | `core/aggregate/repository_test.go` |
| 756 | `catalog/eventcatalog/exporter_test.go` |
| 648 | `core/event/event_test.go` |
| 617 | `core/event/outbox_publisher_test.go` |
| 615 | `storage/sqlite_integration_test.go` |
| 604 | `catalog/schema_test.go` |

## Test Coverage

| Module | Coverage |
|--------|----------|
| core/command | 100.0% |
| core/query | 100.0% |
| core/pkg/dispatcher | 100.0% |
| middleware | 100.0% |
| memory | 99.5% |
| projection | 98.3% |
| core/pkg/id | 97.8% |
| catalog/adapters | 97.1% |
| catalog/d2 | 97.6% |
| catalog/openapi | 96.6% |
| catalog/eventcatalog | 95.7% |
| catalog | 95.3% |
| storage | 88.1% |
| integration | [no statements] |
| sync | unmeasured |

## `any` Usage in Production Code

Known acceptable uses (generic constraints, codec interfaces):
- Generic type constraints: `T any` in Decider, Repository, Dispatcher, PaginatedResult, etc.
- Codec interfaces: `Encode(v any)`, `Decode(data []byte, v any)`
- AsyncAPI/OpenAPI types: `Payload any`, `Schema any` (dynamic JSON schema)

Known violation (documented):
- `query.Handler = func(context.Context, Query) (any, error)` — returns untyped result
- Workaround: `DispatchTyped[T]` for type safety

## Error Handling

- All errors use `fmt.Errorf` with `%w` for wrapping or `errors.New` for sentinels
- Zero TODO/FIXME/HACK/XXX comments in codebase
- 38 sentinel errors across 7 modules, all classified via `RegisterClassification`

## Summary

| Category | Count |
|----------|-------|
| Critical issues | 1 (testhelpers version mismatch) |
| High issues | 1 (example/todo build broken) |
| File size violations (prod) | 2 |
| Lint issues | 1 (golines) |
| TODO/FIXME comments | 0 |
