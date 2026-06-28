# Zero-Panic Audit — 2026-06-21

> Brutal self-review of the zero-panic elimination work (commits `00ba5da0` → `a40795cd`).
> Identifies 7 concrete issues to fix before this work is production-ready.

---

## What Was Done

Over 4 commits, 26 production `panic()` calls were converted to error returns:

- 8 non-breaking fixes (codec `sync.Once`, listing, cattest, example/todo)
- 7 breaking constructor API changes (6 pebble constructors + multisig VerifierMap)
- 5 event arithmetic invariants (Version/SchemaVersion Decrement/Add/Sub)
- 6 test setup `panic(err)` → `tb.Fatalf` conversions

---

## Issues Found

### 1. Sentinels bypass the error-family taxonomy (CRITICAL)

All 4 new sentinel errors use plain `errors.New()`, which classifies as `Transient` (retryable).
Every other sentinel in the project uses the `go-error-family` taxonomy.

| Sentinel                    | File                            | Current      | Should Be      | Sibling Pattern                               |
| --------------------------- | ------------------------------- | ------------ | -------------- | --------------------------------------------- |
| `ErrNilDatabase`            | `storage/pebble/errors.go:10`   | `errors.New` | `NewRejection` | `ErrAggregateTypeMismatch` uses `NewConflict` |
| `ErrNilSigner`              | `signing/multisig/errors.go:10` | `errors.New` | `NewRejection` | `ErrNoVerifier` uses `NewRejection`           |
| `ErrVersionUnderflow`       | `event/types.go:115`            | `errors.New` | `NewRejection` | `ErrVersionConflict` uses `NewConflict`       |
| `ErrSchemaVersionUnderflow` | `event/types.go:188`            | `errors.New` | `NewRejection` | `ParseSchemaVersion` uses `NewRejection`      |

**Impact**: Consumers using `errorfamily.Classify(err)` will mis-classify these as retryable infrastructure
failures and retry an operation that can never succeed (e.g., retrying `NewStore(nil, ...)`).

### 2. `example_test.go` uses `log.Fatal` instead of `panic` (REGRESSION)

`signing/multisig/example_test.go` — `panic(err)` was changed to `log.Fatal(err)`.
This is wrong: `log.Fatal` calls `os.Exit(1)`, killing the process before `go test` can
capture Output. The Go stdlib convention for Example functions is `panic(err)` since
there's no `*testing.T`. This was a correct idiom that shouldn't have been changed.

### 3. No CHANGELOG or migration entry for breaking API changes (CRITICAL)

Breaking signature changes were shipped without documenting them:

- `NewStore/NewSnapshotStore/NewCheckpointStore/NewKVStore/NewQueryStore/NewCommandStore`
  now return `(T, error)` instead of `T`
- `NewBackend` now returns `(*Backend, error)`
- `VerifierMap` now returns `(map, error)`
- `Version.Decrement/Sub` now return `(Version, error)`
- `SchemaVersion.Decrement/Add/Sub` now return `(SchemaVersion, error)`
- `CBOREncMode/CBORDecMode` now return `(mode, error)`
- `NewCBOREncoder/NewCBORDecoder` now return `(*T, error)`

No entry in `CHANGELOG.md`, no migration guide appendix.

### 4. Codec error returns are impossible-error pollution (DESIGN SMELL)

The CBOR mode creation (`CanonicalEncOptions().EncMode()`) **cannot fail** — the options
are hardcoded valid constants. The fxamacker/cbor library itself discards this error:
`var defaultEncMode, _ = EncOptions{}.encMode()`.

The current `(mode, error)` signature forces 7 call sites to handle an impossible error,
adding dead `if err != nil` branches that obscure real errors.

**Better**: `sync.OnceValue` (Go 1.21+) returns the bare value. The panic-inside-once
is correct: it's a programming-error invariant that fires at most once if the library
breaks its own constants.

### 5. `listing/in_memory.go` dead switch case (CLEANUP)

The `TombstoneInclude` case in the switch was changed from `panic("unreachable")` to
`filtered = append(filtered, r)`. But this case IS unreachable (early return at line 165).
The new code is misleading — it implies the case can be reached.

### 6. Error wrapping inconsistency in `event/types.go` (POLISH)

`Decrement()` returns the raw sentinel, `Sub()` wraps with `fmt.Errorf("%w: values")`.
Sibling methods on the same type should wrap consistently.

### 7. `cattest.MustReadFile` still has "Must" in name (COSMETIC)

`catalog/internal/cattest/assertions.go:9` — still named `MustReadFile` despite using
`tb.Fatalf` (not panic). This is the last exported Must-named function.

---

## Next Steps

See execution plan below.
