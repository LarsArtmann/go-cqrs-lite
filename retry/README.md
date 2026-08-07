# retry — DEPRECATED

> **This module is deprecated.** Import [`github.com/larsartmann/go-retry`](https://github.com/LarsArtmann/go-retry) directly instead.
>
> This package re-exports `go-retry` for backward compatibility (ADR-0064).
> The sole internal consumer (`middleware/`) has been migrated to import
> `go-retry` directly. This module will not receive updates.

## Migration

Change your import:

```diff
- "github.com/larsartmann/go-cqrs-lite/retry/v4"
+ "github.com/larsartmann/go-retry"
```

All types, functions, and sentinels are identical (type aliases). No code changes needed beyond the import path.

## Quick Start (deprecated — use go-retry directly)

```bash
go get github.com/larsartmann/go-retry
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/larsartmann/go-retry"
)

func main() {
    config := retry.DefaultConfig()

    err := retry.Do(context.Background(), config, func(ctx context.Context, attempt int) error {
        fmt.Printf("attempt %d\n", attempt)
        return doSomethingFragile()
    })

    if err != nil {
        log.Fatal(err)
    }
}
```

### Preview Backoff Delays

```go
config := retry.DefaultConfig()
for attempt := 1; attempt <= 5; attempt++ {
    delay, err := retry.Backoff(config, attempt)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("attempt %d delay: %v\n", attempt, delay)
}
```

## Related Modules

- [**go-retry**](https://github.com/LarsArtmann/go-retry) — The canonical standalone implementation
- [**middleware**](../middleware/README.md) — CQRS-wrapped retry: `MessageAdapter`, OTel span, dead-letter entries with `StreamID`
