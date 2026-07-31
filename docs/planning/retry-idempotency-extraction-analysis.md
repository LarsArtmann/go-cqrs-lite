# Module Extraction Analysis: retry/ and idempotency/

> **Question:** Should `retry/` and `idempotency/` be replaced with existing libraries?
>
> **Answer:** `retry/` should be deleted. `idempotency/` should stay.

---

## Executive Summary

| Module | Lines | Internal Consumers | Verdict | Replacement |
|---|---|---|---|---|
| `retry/` | 210 | **None** (middleware ignores `retry.Do`, uses only `ComputeDelay`) | **DELETE** | `cenkalti/backoff/v4` |
| `idempotency/` (interface + middleware) | ~160 | middleware (3 middleware funcs) | **KEEP** | No equivalent exists |
| `idempotency/sqlstore/` | ~260 | standalone module | **KEEP** (but stop over-celebrating) | No equivalent exists |
| `idempotency/kvstore/` | ~100 | standalone module | **KEEP** | Genuine composition with `kv/` |

---

## retry/ — The Case for Deletion

### What it is

210 lines across 4 files. Zero CQRS deps — only `go-error-family`. Provides:

- `Do(ctx, config, fn)` — retry loop with exponential backoff + jitter
- `Config` — MaxAttempts, InitialDelay, MaxDelay, Multiplier, IsRetryable, OnRetry, OnExhausted
- `Backoff(config, attempt)` / `ComputeDelay(initial, max, multiplier, attempt)`
- `ErrExhausted`, `ErrCanceled` — error-family classified sentinels

### The brutal truth

**`middleware/retry.go` does not call `retry.Do`.** It imports exactly one function:

```go
// middleware/retry.go:141
func backoff(config RetryConfig, attempt int) time.Duration {
    return retrypkg.ComputeDelay(config.InitialDelay, config.MaxDelay, config.Multiplier, attempt)
}
```

Then it runs its **own loop** because it needs per-attempt OTel spans and `DeadLetterEntry` construction — features `retry.Do` doesn't support. The standalone `retry.Do` has **zero internal consumers**.

### Why it exists

The `doc.go` says: "consumers who only need retry logic (CLI tools, batch processors, simple services) can import it without pulling in CQRS message types or OpenTelemetry SDK."

This is YAGNI. No consumer has asked for this. It's a hypothetical convenience module that costs real maintenance.

### The replacement: `cenkalti/backoff/v4`

The de facto Go retry library. 12k+ stars. Handles:
- Exponential backoff with decorrelated jitter (better than our fixed 50% jitter)
- Context cancellation
- Permanent errors (non-retryable)
- Retry-notify callbacks
- Randomization factor

Integration is trivial:

```go
import "github.com/cenkalti/backoff/v4"

b := backoff.NewExponentialBackOff()
b.MaxElapsedTime = maxElapsed
b.InitialInterval = initialDelay
b.Multiplier = multiplier
b.MaxInterval = maxDelay

// Non-retryable errors:
if !errorfamily.IsRetryable(err) {
    return backoff.Permanent(err)
}

return backoff.RetryNotifyWithContext(fn, backoff.WithMaxRetries(b, maxAttempts), notify)
```

### What we lose by deleting

- `ErrExhausted` / `ErrCanceled` classified sentinels — middleware already defines its own (`ErrRetryExhausted`, `ErrRetryCanceled`)
- `errorfamily.IsRetryable` as default predicate — 1 line to pass explicitly
- Zero-dep story — replaced by one well-maintained dep

### What we gain

- 210 fewer lines to maintain
- Better backoff (decorrelated jitter is provably better than additive jitter)
- Features we'd never build ourselves: RandomizationFactor, MaxElapsedTime, Retry-After header support
- One less module in the workspace, API surface, CI matrix, dep-budget checks, vulncheck

### Migration plan

1. Add `cenkalti/backoff/v4` to `middleware/go.mod`
2. Rewrite `middleware/retry.go` to use `backoff.RetryNotifyWithContext` — OTel spans go in the `notify` callback
3. Delete `retry/` module entirely
4. Remove `retry/v4` from `go.work`, api-stability modules list, CI matrix
5. Update AGENTS.md module list

---

## idempotency/ — The Case for Keeping

### What it is

~520 lines across 3 sub-packages. The core value:

**The `Store` interface** (`store.go:30-48`):

```go
type Store interface {
    Seen(ctx context.Context, key string) (bool, error)
    Record(ctx context.Context, key string, ttl time.Duration) error
    CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error
}
```

The critical insight is `CheckAndRecord` — atomic check-and-record that prevents the TOCTOU race. This is the CQRS-specific contract:

> Implementations MUST make this atomic (single lock or single round-trip) to prevent the TOCTOU race that a separate Seen + Record pair would create.

**The middleware wiring** (`middleware/idempotency.go`):

```go
CommandIdempotency(store, ttl, nil)  // command dispatch dedup
EventIdempotency(store, ttl, nil)    // event processing dedup
QueryIdempotency(store, ttl, keyFn)  // query result dedup
```

These are real CQRS value — they wire the store into the dispatch pipeline with the right error handling (`ErrDuplicate` → skip handler, don't propagate).

**`ErrDuplicate`** — classified as `Conflict` by error-family. This classification is deliberate: a retried command with the same key *conflicts* with a prior recording.

### Why no off-the-shelf replacement exists

There is **no well-known Go idempotency-key library**. This is usually either:
- Hand-rolled inline SQL (`INSERT ... ON CONFLICT`)
- A 20-line Redis `SET NX` wrapper
- Part of a framework (Rails, Django) but never extracted as a standalone library

The reason: it's conceptually simple. But the devil is in the details:
- TTL semantics (not extended on re-record — documented and tested)
- Expired key reclamation (lazy on read + background sweep)
- Atomicity across 3 SQL dialects with different syntax

### What's genuinely valuable vs commodity

| Component | Value | Justification |
|---|---|---|
| `Store` interface | **High** | The `CheckAndRecord` atomicity contract is the core insight |
| `MemoryStore` | Medium | ~80 lines, but well-tested with sweep + lazy deletion |
| `ErrDuplicate` + middleware | **High** | CQRS-specific wiring into dispatch pipelines |
| `sqlstore` (3 dialects) | Medium | `ON CONFLICT` is boilerplate; MySQL `IF()` is a Stack Overflow answer. But no library to delegate to. |
| `kvstore` | Medium | Genuine composition with `kv/` module — shows the Store abstraction works across backends |

### What we should NOT do

- **Don't celebrate the SQL implementations.** They're correct but not novel.
- **Don't add Redis/Memcached stores speculatively.** YAGNI — add when a consumer needs it.
- **Don't extract into a separate repo.** The value is in the CQRS integration (middleware, error classification), not the storage primitive alone.

### What we SHOULD do

- Keep the `Store` interface stable — it's the contract
- Consider adding a `Result` return to `CheckAndRecord` so consumers can cache the first execution's result (the current API forces re-execution after dedup confirmation)
- Keep `sqlstore` but don't over-invest — the 3 dialects are sufficient

---

## The Principle (Corrected)

The "dep budget" principle exists to prevent heavy/wrong dependencies from polluting focused modules. It does NOT exist to justify reimplementing solved problems to avoid a small, well-maintained library.

**Test for whether to keep a module:**

1. Does it have internal consumers? → `retry/` fails this test
2. Does it encode domain-specific insight? → `idempotency/` passes (atomicity contract, CQRS middleware)
3. Is there a mature, well-maintained replacement? → `cenkalti/backoff/v4` for retry; nothing for idempotency
4. Would a consumer benefit from importing it standalone? → Debatable for retry (they'd use backoff directly); yes for idempotency (the interface + middleware)

**When "zero deps" is an excuse:** when the module duplicates a solved problem, has no internal consumers, and the only justification is philosophical purity.

**When "zero deps" is correct:** when the module encodes domain insight that no library captures, and adding a dep would require adapters that negate the simplification.
