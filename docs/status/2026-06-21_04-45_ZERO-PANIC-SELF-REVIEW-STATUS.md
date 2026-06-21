# Zero-Panic Self-Review — Final Status

**Date:** 2026-06-21 04:45
**Scope:** Audit of zero-panic elimination work (commits `00ba5da0` → `493dbffa`)
**Verdict:** All 7 issues found in self-review have been fixed and verified.

---

## A) FULLY DONE

| Item                                         | Status | Verification                                                                                                 |
| -------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------ |
| 26 production panics → error returns         | DONE   | `rg "panic\(" --glob "*.go" \| grep -v _test.go` = 0 results (excluding eventtest/handlers.go panic utility) |
| Error-family taxonomy on all 4 new sentinels | DONE   | All use `event.NewRejection` with proper error codes                                                         |
| Codec `sync.OnceValue` refactor              | DONE   | 7 impossible-error branches eliminated, `CBOREncMode()/CBORDecMode()` return bare values                     |
| Example function `log.Fatal` regression      | FIXED  | Reverted to `panic(err)` (correct Go idiom for Example functions)                                            |
| listing dead switch case                     | FIXED  | Replaced with clean `if/else if` — no dead TombstoneInclude branch                                           |
| Error wrapping consistency                   | FIXED  | `Decrement()` and `Sub()` both wrap with `fmt.Errorf("%w: values")`                                          |
| `cattest.MustReadFile` rename                | DONE   | Renamed to `ReadFile`                                                                                        |
| CHANGELOG entry for breaking changes         | DONE   | Comprehensive entry with all signature changes + error codes                                                 |
| FEATURES.md accuracy                         | DONE   | Updated "Errors as values" line to reflect zero panics + classified sentinels                                |
| API surface golden                           | DONE   | Regenerated (1597 exports)                                                                                   |
| All 37 modules pass build/vet/test/lint/fmt  | DONE   | Full sweep completed                                                                                         |

---

## B) PARTIALLY DONE

| Item                    | What's Done                         | What Remains                                                                                          |
| ----------------------- | ----------------------------------- | ----------------------------------------------------------------------------------------------------- |
| Migration guide         | CHANGELOG entry added               | No dedicated V3 migration appendix for the zero-panic changes specifically                            |
| AGENTS.md code examples | Pebble constructor examples updated | `Version.Decrement` / `SchemaVersion.Decrement` examples in docs not updated (comments only, no code) |

---

## C) NOT STARTED

| Item                                     | Why                                                                        | Impact                |
| ---------------------------------------- | -------------------------------------------------------------------------- | --------------------- |
| V3 migration doc appendix for zero-panic | All callers are test code (zero external production callers) — low urgency | LOW                   |
| API stability versioning (v2 → v3 bump?) | The signature changes ARE breaking, but go.mod still uses v2               | MED (decision needed) |

---

## D) ISSUES FIXED FROM SELF-REVIEW

### Issue 1: Sentinels bypassed error-family taxonomy (CRITICAL)

**Was:** All 4 sentinels used `errors.New()` → classified as `Transient` (retryable) by `errorfamily.Classify()`
**Fixed:** All 4 now use `event.NewRejection` with proper error codes:

- `event.version_underflow`, `event.schema_version_underflow`
- `pebble.nil_database`, `signing.nil_signer`

### Issue 2: Example function `log.Fatal` regression (REGRESSION)

**Was:** `signing/multisig/example_test.go` changed `panic(err)` → `log.Fatal(err)` which calls `os.Exit(1)`
**Fixed:** Reverted to `panic(err)` — the correct Go idiom for Example functions (no `*testing.T` available)

### Issue 3: Codec impossible-error pollution (DESIGN SMELL)

**Was:** `CBOREncMode()/CBORDecMode()` returned `(mode, error)` forcing 7 call sites to handle an error that cannot occur
**Fixed:** Refactored to `sync.OnceValue` — returns bare values. Panic inside `OnceValue` closure is correct (runs at most once, only fires if library breaks its own constants)

### Issue 4: listing/in_memory.go dead switch case (CLEANUP)

**Was:** `TombstoneInclude` switch case was unreachable (early return above) but had code that misled readers
**Fixed:** Replaced switch with two clean `if/else if` branches — no dead code

### Issue 5: Error wrapping inconsistency (POLISH)

**Was:** `Decrement()` returned raw sentinel, `Sub()` wrapped with values
**Fixed:** Both now use `fmt.Errorf("%w: values")` for consistency

### Issue 6: `MustReadFile` still had "Must" in name (COSMETIC)

**Fixed:** Renamed to `ReadFile`

### Issue 7: No CHANGELOG entry (CRITICAL)

**Fixed:** Comprehensive CHANGELOG entry documenting all breaking signature changes with error codes and taxonomy

---

## E) WHAT WE SHOULD IMPROVE

1. **Versioning strategy** — The zero-panic changes are breaking API changes shipped under v2.x. Decide: bump to v3 or document as "v2.8 breaking"?
2. **Lowercase `mustXxx` test helpers** — ~10 still exist across test files. Not exported, use `tb.Fatalf`, but carry the "Must" name convention. Consider renaming for consistency.
3. **Panic in `sync.OnceValue` closures** — Technically still a panic, but it's an invariant guard (like `init()` panics), not control flow. Document this explicitly in a panic policy section of AGENTS.md.
4. **Test coverage for error classification** — No test verifies `errorfamily.Classify(ErrNilDatabase) == Rejection`. Add classification tests.
5. **Doc examples for new error signatures** — The AGENTS.md/SKILL.md examples show the new `_, err :=` pattern but don't show how to handle the errors in context.

---

## F) TOP 25 NEXT STEPS (sorted by impact/effort)

| #   | Task                                                                                                 | Impact | Effort          |
| --- | ---------------------------------------------------------------------------------------------------- | ------ | --------------- |
| 1   | Decide v2 vs v3 versioning for breaking changes                                                      | HIGH   | 0 (decision)    |
| 2   | Add error-family classification tests (Classify(ErrNilDatabase) == Rejection)                        | HIGH   | 15min           |
| 3   | Write panic policy section in AGENTS.md (when panics ARE acceptable)                                 | HIGH   | 20min           |
| 4   | V3 migration appendix for zero-panic signature changes                                               | MED    | 30min           |
| 5   | Rename remaining lowercase `mustXxx` test helpers                                                    | LOW    | 45min           |
| 6   | Add `errors.Is` examples to SKILL.md for each new sentinel                                           | MED    | 15min           |
| 7   | Consider `Version.SubOrZero` / `Version.DecrementOrZero` convenience methods                         | LOW    | 20min           |
| 8   | Audit all error codes for naming consistency (pebble.nil_database vs event.nil_database)             | MED    | 15min           |
| 9   | Add godoc examples for `Version.Decrement` error handling                                            | LOW    | 10min           |
| 10  | Consider adding `ErrNilLogger` for pebble constructors (currently nil logger is allowed)             | LOW    | 10min           |
| 11  | Update docs/index.md quick start to use new error-returning signatures                               | MED    | 10min           |
| 12  | Add integration test that exercises the full error chain (classify → retry decision)                 | MED    | 30min           |
| 13  | Consider `sync.OnceValues` (plural) for future multi-value lazy init needs                           | LOW    | 0 (awareness)   |
| 14  | Audit `command/errors.go` and `query/errors.go` for any remaining plain `errors.New`                 | MED    | 10min           |
| 15  | Add `go-error-family` to the dependency budget docs                                                  | LOW    | 5min            |
| 16  | Document the codec `sync.OnceValue` pattern as a reusable recipe                                     | LOW    | 10min           |
| 17  | Consider structured error fields instead of `fmt.Errorf` string interpolation                        | LOW    | 30min           |
| 18  | Add benchmark for error allocation in hot paths (Version.Sub in projection)                          | LOW    | 20min           |
| 19  | Consider `errors.Join` for multi-error scenarios (Go 1.20+)                                          | LOW    | 0 (awareness)   |
| 20  | Audit example/todo for any remaining non-error-handling patterns                                     | LOW    | 15min           |
| 21  | Add CI check that `errorfamily.Classify` returns non-Transient for all sentinels                     | MED    | 40min           |
| 22  | Consider `Result[T]` type for fallible operations (Go 2 proposal style)                              | LOW    | 0 (exploration) |
| 23  | Document error wrapping convention in AGENTS.md (when to `%w`, when to raw)                          | MED    | 15min           |
| 24  | Add `go generate`-able error code registry from sentinel definitions                                 | LOW    | 60min           |
| 25  | Consider automatic retry-policy based on error family (e.g., middleware that retries Transient only) | LOW    | 90min           |

---

## G) TOP QUESTION

**Should we bump to v3 for these breaking changes, or ship under v2.x?**

All 43 API call sites for the breaking changes are in `_test.go` files — zero external production callers within this repo. But consumers outside the repo WILL be affected:

- Anyone calling `pebble.NewStore(db, logger)` gets a compile error (was `*EventStore`, now `(*EventStore, error)`)
- Anyone calling `Version.Decrement()` gets a compile error

The `go.mod` currently uses `v2` for all modules. Go module versioning says: major version bump (v3) is required for breaking changes. But the convention in this repo has been shipping breaking changes under v2.x pseudo-versions (see CHANGELOG [2.7.0] and [Unreleased] sections — both contain breaking changes without a v3 bump).

**I cannot figure out:** Is the deliberate choice to ship breaking changes under v2.x (accepting the Go module semantic versioning violation), or should this be the trigger to properly bump to v3?

---

## Commit History (this session)

| Commit     | Description                                                           |
| ---------- | --------------------------------------------------------------------- |
| `ce67590e` | Pareto execution plan (HTML + D2 graph)                               |
| `00ba5da0` | Phase 1: 8 non-breaking panics eliminated                             |
| `d7e6ee52` | Phase 2: 7 pebble + signing constructor panics eliminated             |
| `93941de3` | Phase 3: 5 event arithmetic panics eliminated                         |
| `a40795cd` | Phase 4+5: 6 test setup panics + docs                                 |
| `afd0424c` | Self-review fixes: error taxonomy, sync.OnceValue, example regression |
| `493dbffa` | CHANGELOG + error wrapping consistency + MustReadFile rename          |
