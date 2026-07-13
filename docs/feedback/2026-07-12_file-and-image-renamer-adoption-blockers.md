# file-and-image-renamer Adoption Feedback

**Date:** 2026-07-12
**Consumer:** `github.com/LarsArtmann/file-and-image-renamer` (CLI tool, Go 1.26.4)
**Author:** Crush (AI analysis), verified by reading actual source code
**Verdict:** 1 module cleanly adoptable today, 2 modules blocked by specific design issues (this doc explains what to fix)

---

## What I Wanted to Import

| Module           | Why                                                | Outcome                                                                                                    |
| ---------------- | -------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `id/v4`          | Branded IDs (EntryID, CorrelationID)               | **Can import but chose go-branded-id directly** (already in go.mod, avoids ulid dep for string-backed IDs) |
| `idempotency/v4` | Operation-level dedup (prevents double-processing) | **BLOCKED** — kv_store.go in same package pulls kv→codec→cbor+otter                                        |
| `middleware/v4`  | RetryConfig + CircuitBreaker patterns              | **BLOCKED** — OTel hardcoded in retry loop, CQRS message coupling, 6 module deps                           |

---

## Blocker 1: idempotency/ — KVStore in Same Package

### The Problem

`idempotency/store.go` defines `Store`, `MemoryStore`, and `ErrDuplicate` — zero dependencies beyond `go-error-family`.

`idempotency/kv_store.go` defines `KVStore`, `KVBackend` — depends on `kv/v4`.

**Both files are in `package idempotency`.** Go compiles all files in a package together. Importing `idempotency/v4` for just `MemoryStore` pulls `kv/v4` → `codec/v4` → `fxamacker/cbor/v2` + `maypok86/otter/v2`.

### The Impact

A consumer who wants a 50-line in-memory TTL store gets:

- `fxamacker/cbor/v2` (CBOR serialization — not used)
- `maypok86/otter/v2` (cache library — not used)
- `codec/v4` (go-cqrs-lite codec — not used)
- `kv/v4` (go-cqrs-lite KV abstraction — not used)

### The Fix

Move `kv_store.go` to a subpackage: `idempotency/kvstore/` (or `idempotency/durable/`).

```go
// idempotency/kvstore/kvstore.go
package kvstore

import "github.com/larsartmann/go-cqrs-lite/idempotency/v4"
import "github.com/larsartmann/go-cqrs-lite/kv/v4"

type KVBackend interface { ... }
type KVStore struct { ... } // implements idempotency.Store
```

Now `import "github.com/larsartmann/go-cqrs-lite/idempotency/v4"` pulls ONLY `go-error-family`. Consumers who need the durable KV-backed store import `idempotency/kvstore/v4` explicitly.

### Effort

~30 minutes: move file, change package name, update imports, update go.mod (remove kv/ dep from idempotency/, add it to kvstore/).

---

## Blocker 2: middleware/ — OTel Hardcoded in Retry Loop

### The Problem

`middleware/retry.go:75`:

```go
for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
    attemptCtx, attemptSpan := cqrsotel.StartSpan(
        ctx, retryTracer(), fmt.Sprintf("retry.attempt.%d", attempt),
        cqrsotel.SpanKindInternal,
        cqrsotel.WithAttributes(
            cqrsotel.AttrInt("cqrs.retry.attempt", attempt),
            cqrsotel.AttrInt("cqrs.retry.max_attempts", config.MaxAttempts),
        ),
    )
```

Every retry attempt unconditionally creates an OpenTelemetry span via `cqrsotel.StartSpan()`. This is not behind an interface, not optional, not feature-flagged.

### The Impact

Importing `middleware/v4` for retry logic forces the consumer to accept:

- `go.opentelemetry.io/otel` SDK
- `go.opentelemetry.io/otel/trace`
- `go.opentelemetry.io/otel/sdk/metric`
- `go.opentelemetry.io/otel/sdk`
- `cqrsotel` (go-cqrs-lite otel/ module)
- `command/v4`, `event/v4`, `query/v4` (CQRS message types)
- `idempotency/v4` (→ kv → codec → cbor → otter)

For a CLI tool that uses `log/slog` for logging and has no distributed tracing, this is 10+ transitive dependencies for what should be a retry loop with backoff.

### The Fix (Option A: Interface-based tracing)

```go
type SpanHandler interface {
    StartSpan(ctx context.Context, name string, attrs ...Attr) (context.Context, Span)
}

type Span interface {
    End()
    RecordError(err error)
}

type NoopSpanHandler struct{}
func (NoopSpanHandler) StartSpan(ctx context.Context, _ string, _ ...Attr) (context.Context, Span) {
    return ctx, NoopSpan{}
}

// In RetryConfig:
type RetryConfig struct {
    // ... existing fields ...
    SpanHandler SpanHandler // defaults to NoopSpanHandler if nil
}
```

Consumers who want OTel pass `cqrsotel.SpanHandler{}`. Consumers who don't pass nothing (noop default).

### The Fix (Option B: Separate retry/ module)

Split `middleware/` into:

- `retry/v4` — `RetryConfig`, `NewRetry[M]`, `backoff()`. Zero OTel. Zero CQRS deps.
- `circuitbreaker/v4` — `CircuitBreakerConfig`, `NewCircuitBreaker[M]`. Zero OTel.
- `middleware/v4` — composition layer that adds OTel spans, CQRS adapters, dead-letter integration

This lets consumers import `retry/v4` without the CQRS ceremony.

### Effort

Option A: ~2 hours (interface + noop + update retry.go + tests).
Option B: ~4 hours (new module(s), move code, update go.work, tag releases).

---

## Blocker 3: middleware/ — CQRS Message Coupling in Generic Retry

### The Problem

`NewRetry[M any]` requires `MessageAdapter[M]`:

```go
type MessageAdapter[M any] struct {
    Kind        string
    ExtractType func(M) string
    ExtractID   func(M) id.AggregateID  // <-- requires id.AggregateID
}
```

A consumer wanting retry for a simple operation must:

1. Import `id/v4` (for `AggregateID`)
2. Define a message type `M`
3. Implement `ExtractID` returning `id.AggregateID`

For a CLI tool renaming files, there is no "aggregate." The operation is `func(ctx) (filename, usage, error)`. Forcing it through `MessageAdapter[M]` + `Handler[M]` adds ceremony without value.

### The Fix

Add a `retry.Func(fn func(context.Context) error, config RetryConfig) error` entry point that doesn't require message types:

```go
// retry/v4/retry.go
func Do(ctx context.Context, config Config, fn func(context.Context) error) error {
    // same backoff + retry logic, no MessageAdapter needed
}
```

The generic `NewRetry[M]` stays for CQRS consumers. `Do()` serves everyone else.

---

## What Works Well

- **`id/v4`** is clean: zero go-cqrs-lite internal deps, well-designed branded types, `DeriveAggregateID` for deterministic IDs. The `AggregateID` (string-backed) vs `Of[T]` (ULID-backed) split is thoughtful.
- **`RetryConfig` struct** is well-designed: `MaxAttempts`, `InitialDelay`, `MaxDelay`, `Multiplier`, `IsRetryable`, `OnDeadLetter`. Clean validation. Good defaults.
- **`CircuitBreakerConfig`** is clean and the state machine (closed→open→half-open→closed) is correct.
- **`MemoryStore`** is textbook Go: `sync.RWMutex`, lazy expiry, background sweeper, idempotent `Close()`.
- **Error taxonomy** (`go-error-family` integration) is consistently applied: `ErrDuplicate` as `Conflict`, retry exhaustion as `Infrastructure`.

---

## Adopted (Without Importing)

Due to the blockers above, `file-and-image-renamer` currently has **copied patterns** from go-cqrs-lite in:

- `pkg/renamer/idempotency.go` — copied from `idempotency/store.go` (156 lines)
- `pkg/provider/retry.go` — RetryConfig pattern from `middleware/middleware.go` (137 lines)
- `pkg/provider/circuit_breaker.go` — CircuitBreakerConfig from `middleware/circuit_breaker.go` (184 lines)

**This is the outcome you built the library to prevent.** The consumer is maintaining duplicate code because the library's dependency graph makes importing more expensive than copying. Fixing Blockers 1-3 would eliminate this duplication.

---

## Priority Order for Fixes

| Priority | Fix                                               | Unblocks                                  | Effort    |
| -------- | ------------------------------------------------- | ----------------------------------------- | --------- |
| 1        | Move `kv_store.go` to subpackage                  | idempotency/ adoption                     | 30 min    |
| 2        | Add `retry.Do()` entry point (no MessageAdapter)  | retry/ adoption for non-CQRS consumers    | 1 hour    |
| 3        | Make OTel optional (interface or separate module) | middleware/ adoption without tracing deps | 2-4 hours |

After fix #1, the consumer can `import idempotency/v4` and delete the copied `idempotency.go`.
After fixes #2+#3, the consumer can import retry logic and delete the copied `retry.go` + `circuit_breaker.go`.

---

## Appendix A: Critical Review (2026-07-12)

**Reviewer:** Crush (critical analysis with source verification)
**Verdict:** All three blockers are **factually verified and directionally correct**. Two proposed fixes need refinement. One fix (Blocker 1) was implemented immediately.

### Verification Summary

Every technical claim was verified against source code:

| Claim                                                   | Verified | Evidence                                                                                      |
| ------------------------------------------------------- | -------- | --------------------------------------------------------------------------------------------- |
| `kv_store.go` in `package idempotency`                  | YES      | `idempotency/kv_store.go:1` (now moved) — `package idempotency`                               |
| `store.go` has zero deps beyond go-error-family         | YES      | `idempotency/store.go:1-9` — imports only `context`, `sync`, `time`, `go-error-family`        |
| Importing idempotency pulls kv -> codec -> cbor + otter | YES      | `idempotency/go.mod` (pre-fix) — `kv/v4` was a direct dependency                              |
| `retry.go` hardcodes `cqrsotel.StartSpan()` per attempt | YES      | `middleware/retry.go:75` — unconditional call, no nil guard, no feature flag                  |
| `MessageAdapter[M].ExtractID` forces `id/v4` import     | YES      | `middleware/generic.go:23` — `ExtractID func(M) id.AggregateID` in struct type signature      |
| middleware go.mod has 6+ direct module deps             | YES      | `middleware/go.mod:6-19` — command, event, eventtest, id, otel, query, idempotency + OTel SDK |

**All claims are accurate.** The feedback author did their homework.

### PRO: What the Feedback Gets Right

1. **Blocker 1 diagnosis is textbook-correct.** `store.go` defines `Store`, `MemoryStore`, and `ErrDuplicate` — a clean, self-contained abstraction with exactly one dependency (`go-error-family`). `kv_store.go` in the same package drags in `kv/v4` -> `codec/v4` -> `fxamacker/cbor/v2` + `maypok86/otter/v2`. This is a package boundary violation, not a design preference.

2. **The core thesis is valid and important.** A library whose quality gate is "Would a consumer trust this enough to import it?" must not force 10+ transitive dependencies for a retry loop. The consumer was forced to copy 477 lines of code rather than import the library.

3. **The "What Works Well" section is fair and accurate.** The praise for `RetryConfig` struct design, `CircuitBreakerConfig` state machine, `MemoryStore` concurrency, and error taxonomy integration is all verified and deserved.

4. **Priority ordering is correct.** Fix #1 (idempotency, 30 min, highest impact) before #2-3 (retry, more effort). Right Pareto ordering.

5. **Blocker 3's core observation is sound.** Forcing non-CQRS consumers through `MessageAdapter[M]` + `Handler[M]` to get retry logic is unnecessary ceremony. A `retry.Do()` entry point is the right abstraction.

### CONTRA: What the Feedback Gets Wrong or Oversimplifies

1. **Option A (SpanHandler interface) is the wrong abstraction.** The feedback proposes a custom `SpanHandler` interface to make OTel optional, but misses that `cqrsotel.NewTracer()` already returns a no-op tracer at runtime when no OTel provider is configured (`otel/tracer.go:16`: `otel.GetTracerProvider().Tracer(...)`). The runtime overhead is already zero. The problem is purely compile-time/module-graph dependency. A custom SpanHandler duplicates what `go.opentelemetry.io/otel/trace.Tracer` already provides and creates a maintenance burden. **The correct fix is Option B (separate module)**, which the feedback also proposes but ranks as the alternative.

2. **The feedback doesn't mention the runtime is already no-op.** It frames the problem as if OTel spans cause real overhead, leading the reader toward Option A (interface) when Option B (module split) is the correct remedy for the actual problem (module graph bloat).

3. **"Must implement ExtractID" is partially inaccurate.** `ExtractID` is optional at runtime — `retry.go:36` checks `if adapter.ExtractID != nil`, and `QueryAdapter` (`generic.go:49-52`) already omits it. A consumer CAN set `ExtractID: nil`. The real issue is that the `MessageAdapter[M]` struct type signature contains `func(M) id.AggregateID`, which forces `id/v4` into the compile graph even if you never populate the field. This is a compile-time coupling, not a runtime requirement. The fix (`retry.Do()`) is still correct, but the reasoning should be precise about why.

4. **Hidden coupling in DeadLetterEntry.** `DeadLetterEntry` (`deadletter.go:20-46`) contains `AggregateID id.AggregateID`. A clean `retry/` module cannot use `DeadLetterEntry`. It needs a simpler callback like `OnExhausted func(attempts int, err error)`. The middleware layer translates this to `DeadLetterEntry` for CQRS consumers. The feedback's `retry.Do()` example doesn't address dead-letter at all.

5. **Effort estimates are optimistic for a 48-module workspace.** Each new module triggers go.work update, api_surface.txt regeneration, cmd/api-stability module list update, AGENTS.md update, doc-check verification, and full test suite.

6. **The id/v4 analysis undersells.** `id/v4` IS `go-branded-id` wrapped. The consumer's choice to use the upstream directly is the expected outcome for a consumer that doesn't need ULID generation, not a near-miss.

7. **Missing: circuit breaker should get the same treatment.** `CircuitBreakerConfig` was copied (184 lines) but no extraction is proposed. It has the same coupling as retry.

### Implementation Status (all shipped, all tests passing)

**Fix 1: `idempotency/kvstore/` subpackage — DONE**

Moved `KVStore`/`KVBackend` to `idempotency/kvstore/`. Now `import idempotency/v4` pulls ONLY `go-error-family`. The `idempotency/go.mod` went from 8 deps to 1. Consumers who need the KV-backed store import `idempotency/kvstore/v4` explicitly.

**Fix 2+3: `retry/` module — DONE**

New zero-dep module: `retry.Do(ctx, config, fn)`, `retry.Config`, `retry.Backoff()`. Only depends on `go-error-family`. 14 tests, all passing. Used Option B (module split), NOT Option A (interface) per the review's recommendation.

**Decision: middleware/ kept standalone.** Translating retry.Do()'s errors through middleware would risk breaking the error chain for existing consumers. The ~15 lines of shared backoff algorithm is acceptable duplication vs. a v5 breaking change.

### What NOT to do

- **Do NOT create a SpanHandler interface** — it duplicates OTel's existing API and solves the wrong problem
- **Do NOT break middleware/ API** — `RetryConfig`, `NewRetry[M]`, `CommandRetry` etc. must stay backward-compatible
- **Do NOT rush the semver implications** — adding new modules is additive (non-breaking). Moving kv_store.go to a subpackage IS a breaking change for consumers of `idempotency.NewKVStore` (it becomes `kvstore.New`). This needs a migration note.
