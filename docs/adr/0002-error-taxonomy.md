# ADR-0002: Error Taxonomy with Five Families

**Status:** Accepted  
**Date:** 2026-05-03

## Context

Go's error handling is ad-hoc — any `error` value can be returned from any function. Without classification, consumers cannot make informed decisions about retries, user feedback, or alerting. The library had ~20 sentinel errors across packages but no systematic way to categorize them.

Retry middleware was using a blanket "always retry on error" approach, which is wrong for business rule violations (rejection) or data corruption (should alert, not retry).

## Decision

Introduce `event.Family` enum and `event.Classify(err) Family` in `core/event/errors.go`:

| Family          | Meaning                      | Retryable | Example                          |
|-----------------|------------------------------|-----------|----------------------------------|
| `Rejection`     | Business rule violation      | No        | "email already exists"           |
| `Conflict`      | Optimistic concurrency       | No        | `ErrVersionConflict`             |
| `Transient`     | Temporary infrastructure     | Yes       | connection timeout               |
| `Corruption`    | Data integrity violation     | No        | "failed to decode snapshot"      |
| `Infrastructure`| Non-transient system error   | No        | "permission denied"              |

Key API:
- `event.Classify(err) Family` — maps sentinels to families, defaults to `Transient`
- `event.IsRetryable(err) bool` — returns true only for `Transient`
- `event.Error` struct with `Code`, `Message`, `Family`, cause — extractable via `errors.As`
- Constructors: `NewRejection`, `NewConflict`, `NewTransient`, `NewCorruption`, `NewInfrastructure`
- `middleware.DefaultRetryConfig().IsRetryable` defaults to `event.IsRetryable`

## Consequences

**Positive:**
- Retry middleware automatically skips non-retryable errors
- Consumers can switch on `Family` for user-facing error messages
- Structured `event.Error` carries machine-readable `Code` for API responses
- `errors.Is` support via `Code+Family` matching

**Negative:**
- Cross-package sentinels (aggregate, projection, storage) cannot be classified without circular imports — documented limitation
- `Classify(nil)` returns `Rejection` (not `Transient`) — intentional: nil means "no error occurred", which is a rejection of the retry premise

**Neutral:**
- Only 5 families — extensible via new `Family` constants but requires careful design to avoid combinatorial explosion
