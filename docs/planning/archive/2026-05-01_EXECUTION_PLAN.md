# Execution Plan: Error Taxonomy, Offline-First Primitives & Bug Fixes

> **Date:** 2026-05-01
> **Status:** Ready to Execute
> **Scope:** go-cqrs-lite only (library scope — go-localfirst is a separate project)
> **Parent:** `docs/planning/2026-05-01_ARCHITECTURE_ROADMAP.md`

---

## Key Design Decisions (Revised After Research)

### 1. Use `cockroachdb/errors` Custom Error Types (Already a Dependency)

We already depend on `cockroachdb/errors` v1.12.0 but only use `New`, `Wrap`, `Wrapf`, `Is`.
It supports **custom error types with `errors.As`/`errors.HasType`** — exactly what we need.
No new dependency required. Use the `errors.As` pattern from their docs.

### 2. Error Families as a Wrapper Type (Not an Enum on a Giant Struct)

The brainstorm proposed a monolithic `domain.Error` struct with 20+ fields.
For a **library**, this is wrong. Instead:

- **`Error` struct** with just `Code`, `Message`, `Family`, `Cause` — 4 fields
- **Family classification via `errors.As`** — check if any error in the chain carries a family
- **Consumer-specific fields** (AggregateID, Field, etc.) are added via `WithCustom` metadata or wrapper types in the consumer's code
- **This keeps the library lean** while giving consumers a classification mechanism

### 3. No New Package — Use `core/event/errors.go` Extension

Creating `core/pkg/errors/` adds import path churn. Instead:

- Extend the existing `core/event/errors.go` with the `Error` type and `Family` enum
- Add classification functions there
- All modules already import `core/event` — zero new dependency edges

### 4. `IdempotencyKey` on Command — Breaking Change Strategy

The Command interface has only 2 methods. Adding `IdempotencyKey()` is breaking.
Strategy: **default implementation via embedding `command.Core`** (which already exists and
is used by all test code). Document the migration. Most consumers embed `command.Core`.

### 5. Client Metadata — Already Supported

`event.WithCustom(key MetadataKey, value string)` already exists.
Offline-first metadata is just convention over `MetadataKey`:

- `event.WithCustom("client.id", clientID)` — no code change needed
- Just document the convention

### 6. New Branded ID for ClientID

`id.Of[T]` already supports creating new branded types in one file.
Add `client_id.go` with a `ClientID` type — consistent with existing pattern.

### 7. Fix the Silent Error Discard in Projection Runner

`projection/runner.go:140` has `_ = r.handleAndCheckpoint(...)` — errors silently dropped.
Fix this to log and support the `WithRetry` option that already exists but is unused.

---

## Task List (Sorted by Impact × Effort)

Each task is designed to be completable in **≤12 minutes**.

| #   | Task                                                                                                 | Impact | Effort | Module      | Type        |
| --- | ---------------------------------------------------------------------------------------------------- | ------ | ------ | ----------- | ----------- |
| 1   | Create `Error` struct + `Family` enum in `core/event/errors.go`                                      | HIGH   | LOW    | core        | Feature     |
| 2   | Add `Classify(err) Family` function mapping sentinels to families                                    | HIGH   | LOW    | core        | Feature     |
| 3   | Add constructor helpers: `Reject()`, `Conflict()`, `Transient()`, `Corruption()`, `Infrastructure()` | HIGH   | LOW    | core        | Feature     |
| 4   | Add `IsRetryable(err) bool` helper                                                                   | HIGH   | LOW    | core        | Feature     |
| 5   | Add `WithCause(err) *Error` fluent setter                                                            | MED    | LOW    | core        | Feature     |
| 6   | Write tests for error taxonomy (100% coverage)                                                       | HIGH   | MED    | core        | Test        |
| 7   | Fix silent error discard in `projection/runner.go` dispatchToProjections                             | HIGH   | LOW    | projection  | Bug         |
| 8   | Wire `WithRetry` option in projection runner (already defined, unused)                               | MED    | MED    | projection  | Feature     |
| 9   | Add `ClientID` branded type in `core/pkg/id/client_id.go`                                            | MED    | LOW    | core        | Feature     |
| 10  | Add `event.WithClientID(id.ClientID) Option` convenience wrapper                                     | MED    | LOW    | core        | Feature     |
| 11  | Document offline-first metadata convention (`docs/OFFLINE_FIRST_METADATA.md`)                        | MED    | LOW    | docs        | Docs        |
| 12  | Update `middleware/retry.go` to use `IsRetryable()` as default `IsRetryable` func                    | MED    | LOW    | middleware  | Enhancement |
| 13  | Add `IdempotencyKey() string` to `command.Command` interface                                         | HIGH   | MED    | core        | Breaking    |
| 14  | Implement `IdempotencyKey()` on `command.Core` (returns `""`)                                        | HIGH   | LOW    | core        | Feature     |
| 15  | Update all test code implementing `command.Command` to add `IdempotencyKey()`                        | MED    | MED    | tests       | Migration   |
| 16  | Update `example/user/` to add `IdempotencyKey()` to command types                                    | MED    | LOW    | example     | Migration   |
| 17  | Update `integration/` tests for new `IdempotencyKey()` method                                        | MED    | MED    | integration | Migration   |
| 18  | Update `testhelpers/` command helpers for `IdempotencyKey()`                                         | MED    | LOW    | testhelpers | Migration   |
| 19  | Update `middleware/` validation/recovery/retry tests for `IdempotencyKey()`                          | MED    | MED    | middleware  | Migration   |
| 20  | Update `projection/` tests for `IdempotencyKey()` (if they use commands)                             | LOW    | LOW    | projection  | Migration   |
| 21  | Run full test suite + lint check                                                                     | HIGH   | LOW    | all         | Verify      |
| 22  | Update AGENTS.md with new types, interfaces, and patterns                                            | MED    | LOW    | docs        | Docs        |
| 23  | Update ARCHITECTURE_ROADMAP.md with what was actually done                                           | LOW    | LOW    | docs        | Docs        |

---

## Detailed Task Specifications

### Task 1: Create Error Struct + Family Enum

**File:** `core/event/errors.go` (append to existing file)

```go
type Family int

const (
    Rejection      Family = iota // Bad input, unauthorized, not found
    Conflict                     // Version mismatch, already exists, state machine violation
    Transient                    // Temporary infrastructure failure, retryable
    Corruption                   // Poison event, damaged data, unparseable payload
    Infrastructure               // System cannot serve: closed, nil deps, startup failure
)

func (f Family) String() string { /* switch */ }

type Error struct {
    Code    string // Machine-readable: "event.version_conflict"
    Message string // Human-readable
    Family  Family
    cause   error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.cause }
```

**Size:** ~40 lines. Uses `cockroachdb/errors` patterns (Unwrap, As).

### Task 2: Classify Function

**File:** `core/event/errors.go` (append)

```go
func Classify(err error) Family {
    var e *Error
    if errors.As(err, &e) {
        return e.Family
    }
    // Map known sentinels
    switch {
    case errors.Is(err, ErrVersionConflict):
        return Conflict
    case errors.Is(err, ErrAggregateNotFound),
         errors.Is(err, ErrSnapshotNotFound):
        return Rejection
    case errors.Is(err, ErrStoreClosed),
         errors.Is(err, ErrBusClosed),
         errors.Is(err, ErrSnapshotStoreClosed),
         errors.Is(err, ErrNilProjection),
         errors.Is(err, ErrNilCheckpointStore):
        return Infrastructure
    default:
        return Transient // unknown errors default to retryable
    }
}
```

**Size:** ~25 lines. Covers all `core/event` sentinels.

### Task 3: Constructor Helpers

```go
func NewRejection(code, msg string) *Error      { return &Error{Code: code, Message: msg, Family: Rejection} }
func NewConflict(code, msg string) *Error        { return &Error{Code: code, Message: msg, Family: Conflict} }
func NewTransient(code, msg string) *Error       { return &Error{Code: code, Message: msg, Family: Transient} }
func NewCorruption(code, msg string) *Error      { return &Error{Code: code, Message: msg, Family: Corruption} }
func NewInfrastructure(code, msg string) *Error  { return &Error{Code: code, Message: msg, Family: Infrastructure} }
```

**Note:** Named `New*` (not bare `Reject`, `Conflict`) to avoid collision with types.
`Reject`/`Conflict` etc. are **Family** values, not functions.

### Task 4: IsRetryable Helper

```go
func IsRetryable(err error) bool {
    return Classify(err) == Transient
}
```

### Task 5: WithCause Fluent Setter

```go
func (e *Error) WithCause(cause error) *Error { e.cause = cause; return e }
```

### Task 6: Error Taxonomy Tests

Create `core/event/errors_taxonomy_test.go`:

- Test each constructor produces correct family
- Test Classify maps all sentinels correctly
- Test Classify falls through to Transient for unknown errors
- Test errors.As extracts \*Error from wrapped chain
- Test IsRetryable true/false
- Test WithCause sets cause correctly

### Task 7: Fix Projection Runner Silent Error Discard

**File:** `projection/runner.go` line 140

Change `_ = r.handleAndCheckpoint(ctx, p, evt)` to:

```go
if err := r.handleAndCheckpoint(ctx, p, evt); err != nil {
    if r.opts.retryCount > 0 {
        if retryErr := r.retryHandler(ctx, p, evt, err); retryErr != nil {
            // Log the final error, continue processing other projections
        }
    }
}
```

Add `retryHandler` method that uses `opts.retryCount` and `opts.retryDelay`.

### Task 8: Wire WithRetry Option

The `WithRetry` option already exists in `options.go` but `runner.go` never reads `opts.retryCount`.
Wire it up in `dispatchToProjections` and `replay`.

### Task 9: ClientID Branded Type

**File:** `core/pkg/id/client_id.go` (new file)

```go
package id

type clientMarker struct{}

type ClientID = Of[clientMarker]

func NewClientID() ClientID        { return New[ClientID]() }
func ParseClientID(s string) (ClientID, error) { return Parse[ClientID](s) }
func MustParseClientID(s string) ClientID      { return MustParse[ClientID](s) }
```

**Size:** ~15 lines. Follows exact pattern of `causation_id.go`.

### Task 10: WithClientID Convenience Wrapper

**File:** `core/event/options.go` (append)

```go
func WithClientID(v id.ClientID) Option {
    return WithCustom("client.id", v.String())
}
```

Wait — `WithCustom` takes `MetadataKey` which is `string`. And we have `apply` for typed metadata.
But `ClientID` doesn't have a dedicated Metadata field. Two options:

**Option A:** Add `ClientID` field to `Metadata` struct (breaking for serialization).
**Option B:** Use `WithCustom("client.id", v.String())` — non-breaking, convention-based.

**Choose Option B** for now. Add the convenience wrapper that uses `WithCustom`.

Also add: `WithClientOccurredAt(t time.Time)` → `WithCustom("client.occurred_at", t.Format(time.RFC3339Nano))`

### Task 11: Offline-First Metadata Convention Docs

Create `docs/OFFLINE_FIRST_METADATA.md` documenting the metadata key convention.

### Task 12: Default IsRetryable in Middleware

Update `middleware/middleware.go` `DefaultRetryConfig()`:

```go
IsRetryable: event.IsRetryable, // was: func(error) bool { return false }
```

This makes retry middleware actually useful out of the box.

### Tasks 13-20: IdempotencyKey Migration

**Task 13:** Add `IdempotencyKey() string` to `command.Command` interface.
**Task 14:** Implement on `command.Core` returning `""`.
**Tasks 15-20:** Update all downstream code implementing Command interface.

The main consumers are:

- `testhelpers/helpers.go` — all command handler test types
- `integration/` — BDD test command types (`submitExpenseCmd` etc.)
- `middleware/` — test command types
- `example/user/` — domain command types
- `projection/` — if it has command types in tests

---

## What I Got Wrong in the First Plan

| Original Idea                                     | Problem                                                                                  | Revised Approach                                                  |
| ------------------------------------------------- | ---------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| New `core/pkg/errors/` package                    | Adds import path churn, all modules already import `core/event`                          | Extend existing `core/event/errors.go`                            |
| Monolithic `Error` struct with 20+ fields         | Too heavy for a library, consumer-specific concerns leaked in                            | 4-field `Error` + `Classify()` + consumer adds their own wrappers |
| `Reject()`, `Conflict()` as bare function names   | Collides with `Family` const names                                                       | `NewRejection()`, `NewConflict()` etc.                            |
| `IdempotencyKey()` with `BaseCommand` helper      | `command.Core` already IS the base — no new type needed                                  | Just add method to `command.Core`                                 |
| New `WithClientID` option with `apply[T]` pattern | ClientID not in Metadata struct — would need breaking change                             | Use `WithCustom("client.id", ...)` convention                     |
| 9 error families in library                       | Only 5 are library-relevant (Rejection, Conflict, Transient, Corruption, Infrastructure) | 5 families in library, 4 in go-localfirst                         |

---

## Execution Order (Dependency-Aware)

```
Phase 1: Error Taxonomy (no breaking changes)     [Tasks 1-6]
  1 → 2 → 3 → 4 → 5 → 6 (tests last, verify all pass)

Phase 2: Bug Fixes + Enhancements                  [Tasks 7-8, 12]
  7 (fix silent discard) → 8 (wire retry) → 12 (default retryable)

Phase 3: Offline-First Primitives (no breaking changes)  [Tasks 9-11]
  9 (ClientID type) → 10 (WithClientID wrapper) → 11 (docs)

Phase 4: IdempotencyKey Breaking Change            [Tasks 13-20]
  13 (interface) → 14 (Core impl) → 15-20 (migrate all consumers)

Phase 5: Verification + Documentation              [Tasks 21-23]
  21 (full test suite) → 22 (AGENTS.md) → 23 (roadmap update)
```

Each phase ends with `go test` + `nix run .#lint` verification.
