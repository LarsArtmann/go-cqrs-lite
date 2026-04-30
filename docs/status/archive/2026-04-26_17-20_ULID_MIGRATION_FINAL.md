# ULID Migration — Final Status

**Date:** 2026-04-26
**Status:** ✅ Complete

## Summary

Migrated `core/pkg/id` from `cbid.ID[T, string]` (type alias) to a wrapper struct around `cbid.ID[T, ulid.ULID]`, making IDs binary-sortable with proper ULID backing. All serialization methods reimplemented locally since `go-composable-business-types` doesn't support `ulid.ULID` as a value type.

## Changes This Session

### Bug Fixes

| Bug                                                 | Fix                                                          | File                                                                |
| --------------------------------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------------- |
| `MarshalJSON` returned bare text (no JSON quotes)   | `[]byte('"' + id.String() + '"')`                            | `core/pkg/id/id.go`                                                 |
| `UnmarshalBinary` checked `ulid.EncodedSize` (26)   | Changed to `ulidBinarySize = 16`                             | `core/pkg/id/id.go`                                                 |
| `Value()` returned `[]byte` (breaks SQL drivers)    | Returns `string` (ULID text)                                 | `core/pkg/id/id.go`                                                 |
| `NewWithPrefix` silently discarded prefix parameter | Deleted function, type, test, benchmark                      | `core/pkg/id/id.go`                                                 |
| Benchmark used UUID-format string for `Parse`       | Changed to valid ULID                                        | `core/pkg/id/benchmark_test.go`                                     |
| Stale comment "google/uuid" in event.go             | Changed to "oklog/ulid"                                      | `core/event/event.go:17`                                            |
| `example/catalog/go.mod` missing replace directive  | Added `go-composable-business-types` replace + `go mod tidy` | `example/catalog/go.mod`                                            |
| xtypes test error message said "expected cmd-123"   | Changed to proper ULID                                       | `xtypes/xtypes_test.go`                                             |
| `nilnil` lint on `Value()` zero return              | Added `//nolint:nilnil` (SQL convention)                     | `core/pkg/id/id.go`                                                 |
| Magic number 16 in `UnmarshalBinary`                | Extracted `ulidBinarySize` constant                          | `core/pkg/id/id.go`                                                 |
| `wsl_v5` lint issues in test file                   | Added blank lines between statements                         | `core/pkg/id/id_test.go`                                            |
| `golines` lint on long ULID-containing lines        | Extracted IDs to variables, split multi-line                 | `core/event/event_test.go`, `core/event/event_sourcing_bdd_test.go` |

### Deleted

- `NewWithPrefix[T]()` — prefix incompatible with ULID format, no production callers
- `PrefixString` type — only used by `NewWithPrefix`
- `TestNewWithPrefix` — tested the deleted function
- `BenchmarkNewWithPrefix` — benchmarked the deleted function

### Test Fixture Migration (~120 replacements)

All non-ULID string literals replaced with valid 26-char Crockford base32 ULIDs across 10 test files.

## Architecture

```
id.Of[T] struct {
    wrapped cbid.ID[T, ulid.ULID]   // wraps remote library's ID[T, ULID]
}

// All serialization reimplemented locally:
MarshalJSON()     → "01HK1549P84T9XF8R94E960633" (JSON string with quotes)
UnmarshalJSON()   ← parses ULID string from JSON
Scan()            ← accepts string/[]byte, parses as ULID
Value()           → string (ULID text) for SQL compatibility
MarshalBinary()   → 16 bytes (big-endian ULID)
UnmarshalBinary() ← expects 16 bytes
MarshalText()     → ULID string bytes
UnmarshalText()   ← parses ULID string
Compare()         → uses ulid.ULID.Compare() (not cmp.Compare)
```

## Verification

| Check                                                                          | Result      |
| ------------------------------------------------------------------------------ | ----------- |
| `go test ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/...`  | ✅ All pass |
| `go vet ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/...`   | ✅ Clean    |
| `golangci-lint run ./core/pkg/id/... ./core/event/... ./xtypes/...`            | ✅ 0 issues |
| `go build ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/...` | ✅ Clean    |
| `go build .` (example/user, GOWORK=off)                                        | ✅ Clean    |
| `go build .` (example/catalog, GOWORK=off)                                     | ✅ Clean    |

## Remaining Known Issues (pre-existing, not introduced this session)

| Issue                                                    | Severity | Detail                                              |
| -------------------------------------------------------- | -------- | --------------------------------------------------- |
| `MemoryBus.Publish` holds RLock during handler execution | LOW      | Subscribers block publishers                        |
| `xtypes.TypedCommand.Command()` allocates on every call  | LOW      | Creates new `command.Core` each time                |
| `toDotAddress` number handling                           | LOW      | "Get3DView" → "get.3.d.view"                        |
| No `EventRetry` tests                                    | LOW      | `EventValidation` tested, `EventRetry` not          |
| `go.work` version mismatch                               | LOW      | go.work says `go 1.26`, modules require `go 1.26.0` |
