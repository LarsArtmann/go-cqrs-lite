# go-cqrs-lite — Consumer Feedback: go-retry Extraction

**Consumer:** [bank-sync](https://github.com/LarsArtmann/bank-sync) — bank transaction sync CLI (Wise API + Qonto CSV → SQLite)
**SDK version:** go-retry v0.3.1 (standalone module)
**Date:** 2026-08-21
**Severity:** n/a — positive adoption report with one API ask
**Status:** Verified against bank-sync source (`internal/bank/wise/retry.go`).

---

## TL;DR

The extraction of go-retry out of the cqrs internals was the right call: bank-sync
adopts it **outside any cqrs context** — directly in its Wise HTTP adapter — where
none of the cqrs-lite machinery is present. The `IsRetryable` / `OnRetry` /
`OnExhausted` hook design is what won it over alternatives. One API gap forced a
local generic wrapper that belongs upstream: **`retry.DoWithValue[T]`**.

---

## 1. Where bank-sync uses it (verified against source)

The Wise adapter retries every API call with a policy built entirely from
public `retry.Config` fields:

```go
// internal/bank/wise/retry.go
retry.Config{
    MaxAttempts:  3,
    InitialDelay: 500 * time.Millisecond,
    MaxDelay:     5 * time.Second,
    Multiplier:   2.0,
    IsRetryable:  isRetryable, // net.Error timeouts, HTTP 429, 5xx — never ctx.Canceled, never 4xx
    OnRetry:      func(attempt int, delay time.Duration, err error) { log.Warn(...) },
    OnExhausted:  func(attempts int, err error) { log.Error(...) },
}
```

The hooks were the deciding factor over `hashicorp/go-retryablehttp`: each retry
attempt becomes a structured log line with attempt number, computed backoff and
the underlying error — which is how the 2026-08-21 Wise outage was diagnosed from
logs alone (attempt counts visible per endpoint).

The config is user-tunable (`WiseConfig.Retry` in bank-sync's koanf config) —
`newRetryFromConfig` overrides `MaxAttempts`/`InitialDelay`/`MaxDelay` with
caller values, falling back to defaults on zero values. Zero-value-fallback
semantics work well for this; no ask there.

## 2. The ask: `retry.DoWithValue[T]`

`retry.Do` returns only `error`. Every value-returning call site (all HTTP GETs
in the Wise adapter) needs the same local wrapper:

```go
// internal/bank/wise/retry.go
func retryWithValue[T any](
    ctx context.Context,
    config retry.Config,
    fn func(context.Context) (T, error),
) (T, error) {
    var zero T
    // pre-flight context check (retry.Do does not do one before attempt 1),
    // then: if err := retry.Do(ctx, config, func(ctx) error { v, err = fn(ctx); return err }); err != nil { return zero, err }
    ...
}
```

Two details worth absorbing if this moves upstream:

1. **Pre-flight context check.** `retry.Do` does not check `ctx.Err()` before the
   first attempt, so a pre-cancelled context still invokes `fn` once. The wrapper
   preserves the guarantee callers actually expect: cancelled context → zero
   calls. That check (or an equivalent guarantee inside `Do`) would improve the
   core API for everyone, value-returning or not.
2. **Typed result on success, `zero` + wrapped error on failure** — no heap, no
   `interface{}` gymnastics.

With generics long stable, a first-class `DoWithValue[T]` removes the last piece
of copy-paste retry plumbing from consumers. bank-sync's local wrapper
(`//nolint`-free, tested via the adapter suite) can be the reference
implementation.

## 3. Non-asks (deliberate)

- **Retry budgets / circuit breaking:** bank-sync is a single-user CLI/daemon at
  15-minute sync intervals; per-call attempt caps are sufficient. Adding
  circuit-breaker machinery would grow the API surface for a need this consumer
  cannot validate.
- **Metrics hooks beyond OnRetry/OnExhausted:** the log hooks carry everything a
  structured-logging consumer needs; Prometheus wiring belongs to the caller.
