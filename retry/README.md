# retry — Zero-Dependency Retry with Exponential Backoff

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/retry/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/retry/v4)

A dependency-free retry loop with exponential backoff and jitter. Import it without pulling in CQRS message types or OpenTelemetry SDK.

```bash
go get github.com/larsartmann/go-cqrs-lite/retry/v4
```

## Why?

CLI tools, batch processors, and simple services need retry logic without the overhead of the full CQRS stack. This package provides the zero-CQRS, zero-OTel core. For the CQRS-wrapped version (with message adapters, OTel spans, and dead-letter entries), use [middleware](../middleware/README.md).

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/larsartmann/go-cqrs-lite/retry/v4"
)

func main() {
    config := retry.DefaultConfig() // 3 attempts, 100ms initial, 5s max, 2x multiplier

    err := retry.Do(context.Background(), config, func(ctx context.Context, attempt int) error {
        fmt.Printf("attempt %d\n", attempt)
        return doSomethingFragile()
    })

    if err != nil {
        log.Fatal(err)
    }
}
```

### Custom Configuration

```go
config := retry.Config{
    MaxAttempts:  5,
    InitialDelay: 200 * time.Millisecond,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0,
    IsRetryable:  errorfamily.IsRetryable, // default — skips Rejection/Conflict errors
    OnRetry: func(attempt int, delay time.Duration, err error) {
        log.Printf("retry %d after %v: %v", attempt, delay, err)
    },
    OnExhausted: func(attempts int, err error) {
        log.Printf("giving up after %d attempts: %v", attempts, err)
    },
}
```

### Preview Backoff Delays

```go
config := retry.DefaultConfig()
for attempt := 1; attempt <= 5; attempt++ {
    fmt.Printf("attempt %d delay: %v\n", attempt, retry.Backoff(config, attempt))
}
```

## API

| Symbol                     | Kind   | Description                                                           |
| -------------------------- | ------ | --------------------------------------------------------------------- |
| `Config`                   | Struct | Retry configuration: attempts, delays, multiplier, callbacks.         |
| `DefaultConfig()`          | Func   | Sensible defaults: 3 attempts, 100ms, 5s max, 2x.                     |
| `Config.Validate()`        | Method | Checks configuration validity. Returns error on invalid values.       |
| `Do(ctx, config, fn)`      | Func   | Executes `fn` with retry logic. Stops on success or non-retryable.    |
| `Backoff(config, attempt)` | Func   | Calculates the delay before attempt N (exported for preview/logging). |
| `ComputeDelay(...)`        | Func   | Raw delay calculation without a Config struct.                        |
| `AttemptFunc`              | Type   | `func(ctx context.Context, attempt int) error`                        |
| `ErrExhausted`             | Var    | Returned when all attempts fail (classified as Infrastructure).       |
| `ErrCanceled`              | Var    | Returned when context is canceled during backoff.                     |

## Backoff Formula

The delay for attempt N is:

```
InitialDelay * Multiplier^(N-1) + random jitter (up to 50% of the delay)
```

Capped at `MaxDelay`. Jitter prevents thundering-herd retries across distributed clients.

## Error Classification

- **Retryable**: Uses `errorfamily.IsRetryable` by default. Transient and Infrastructure errors retry; Rejection and Conflict errors return immediately.
- **ErrExhausted**: Classified as Infrastructure. Wraps the last error as its cause.
- **ErrCanceled**: Classified as Infrastructure. Wraps `context.Canceled` as its cause.

## Related Modules

- [**middleware**](../middleware/README.md) — CQRS-wrapped retry: `MessageAdapter`, OTel spans, dead-letter entries with `StreamID`
- [**projectionhost**](../projectionhost/README.md) — Uses retry internally for projection error handling
