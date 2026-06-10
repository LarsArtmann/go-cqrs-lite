# V2.0.0 Release — Session 4 Status Report

**Date:** 2026-06-01
**Branch:** master
**Commits:** 3 new (ea1b48f8, 88e33452, bd97386d)

---

## Summary

Session 4 resumed from session 3 interruption. Fixed broken test file, added schema coverage tests, completed the highest-impact API cleanup (`Event.Context()` removal), and resolved all lint issues. Full build+test+lint now passes clean.

---

## Completed This Session

### 1. Schema Coverage Tests (`ea1b48f8`)

- Fixed broken `schema/versioned_source_test.go` (unused `time` import from session 3)
- Added 4 new test functions covering `LoadToVersion` and `LoadToTimestamp` with upcasters:
  - `TestVersionedStore_LoadToVersion_Upcast` — verifies upcasting applies when loading to a version cap
  - `TestVersionedStore_LoadToTimestamp_Upcast` — verifies upcasting applies when loading to a timestamp cap
  - `TestVersionedStore_LoadToVersion_UpcastError` — error path via `failingUpcaster`
  - `TestVersionedStore_LoadToTimestamp_UpcastError` — error path via `failingUpcaster`

### 2. Event.Context() Removal (`88e33452`) — BREAKING CHANGE

**The single biggest API cleanup for v2.**

Removed `Context() context.Context` from the `Event` interface:

- `deadlineCtx` type and `event/context.go` — **deleted entirely**
- `event/context_test.go` — **deleted entirely** (tested only the removed type)
- `ImmutableEvent.Context()` method — **removed**
- `Event` interface — `Context()` replaced with `Deadline() (time.Time, bool)` (promoted from concrete to interface)

**Kept (still valuable):**

- `deadline time.Time` field on `ImmutableEvent`
- `WithDeadline(t time.Time) Option` — sets deadline on event
- `FromContext(ctx context.Context) Option` — extracts deadline from context
- `ImmutableEvent.Deadline() (time.Time, bool)` — pure data accessor, no allocation

**Rationale:** Storing `context.Context` in a data object is a Go anti-pattern. Zero production callers existed. `Deadline()` is a pure function that returns the same information without the anti-pattern.

**Updated callers:**

- `decider/decider_coverage_test.go` — `nonImmutableEvent` stub updated to implement `Deadline()` instead of `Context()`
- `event/event_context_test.go` — removed `TestEventContext_ContextMethod` (3 sub-tests), kept `Deadline`, `FromContext`, `Clone` tests

### 3. Lint Fixes (`bd97386d`)

- `listing/in_memory.go` — added `TombstoneInclude` case to exhaustive switch
- `pebble/save.go` — replaced inline `if err :=` with separate assignment (noinlineerr)
- `decider/decider_coverage_test.go` — gci formatting fix
- `schema/versioned_source_test.go` — replaced `fmt.Errorf` with `errors.New` (perfsprint)

---

## Verification

| Check             | Status                             |
| ----------------- | ---------------------------------- |
| `nix run .#build` | PASS                               |
| `nix run .#test`  | PASS (33/33 modules)               |
| `nix run .#lint`  | PASS (0 issues across all modules) |

---

## API Surface Impact

### Removed

- `event.Event.Context() context.Context` — use `Event.Deadline()` instead
- `event.deadlineCtx` type (unexported, internal)
- `event/context.go` file

### Promoted to Interface

- `Deadline() (time.Time, bool)` — was already on `ImmutableEvent`, now part of the `Event` interface contract

### Unchanged

- `event.WithDeadline(t time.Time) Option`
- `event.FromContext(ctx context.Context) Option`
- `event.ImmutableEvent.Deadline()` implementation

---

## Remaining Work

From TODO_LIST.md — items still open:

- Storage coverage (72.7% — options, aggregate_reader, stream, projection uncovered)
- Turso tests (0% coverage)
- `ReconstructEventFromFields`, `UnmarshalMetadataJSON`, `MarshalMetadataJSON` (0% coverage in event)
- `EventStream.StreamKey`, `Bus.Publish` (0% coverage)
