# idempotency — Command Idempotency Store

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/idempotency/v3.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/idempotency/v3)

A deduplication store for command idempotency keys (and any other opaque
at-most-once-processing keys).

```bash
go get github.com/larsartmann/go-cqrs-lite/idempotency/v3
```

## Why

Delivery in a CQRS system is **at-least-once**. A client may submit a command,
lose the acknowledgement, and retry. Without deduplication the retried command
executes twice, producing duplicate events and duplicate side effects.

The client attaches a stable key to each logical command; the server records the
key before processing and rejects retries whose key has already been recorded.

## Quick Start

```go
store := idempotency.NewMemoryStore(5 * time.Minute)
defer store.Close()

err := store.CheckAndRecord(ctx, clientCmdKey, 10*time.Minute)
if errors.Is(err, idempotency.ErrDuplicate) {
    return err // already processed — drop the retry
}
```

`CheckAndRecord` is atomic: the check and the record happen in a single step, so
concurrent callers with the same key produce exactly one winner.

## Key Types

| Type           | Purpose                                                   |
| -------------- | --------------------------------------------------------- |
| `Store`        | Interface: `Seen`, `Record`, `CheckAndRecord`             |
| `MemoryStore`  | In-memory `Store` with TTL expiration + background sweep  |
| `ErrDuplicate` | Conflict sentinel returned when a key is already recorded |

## Design

- **Opaque string keys** — matches the industry-standard idempotency-key
  pattern (e.g. HTTP `Idempotency-Key` / `X-Command-Id` headers). Keys are
  client-defined; the store does not interpret them.
- **TTL-based expiration** — keys expire after a configurable duration so the
  store can bound its memory. Expired keys are removed both by a background
  sweeper and lazily on read.
- **Atomic check-and-record** — `CheckAndRecord` prevents the TOCTOU race that a
  separate `Seen` + `Record` pair would create.
- **No dispatch coupling** — the store owns deduplication only. Wiring it into a
  command dispatch pipeline (via [middleware], a transport hook, or a manual
  check) is the consumer's choice.

[middleware]: ../middleware

## Future Backends

`MemoryStore` is for single-process use. The `Store` interface is designed for
distributed backends:

- **Redis**: `SET NX EX` — a single round-trip, atomic.
- **SQL**: `INSERT ... ON CONFLICT (key) DO NOTHING` with a TTL column.

## Related Modules

- [command/v3](../command) — `Command.ID()` / `WithCommandID` provide the stable
  command identity that feeds this store.
- [middleware/v3](../middleware) — command dispatch middleware chain where an
  idempotency check composes.
- [event/v3](../event) — `ErrDuplicate` is classified as a `Conflict` via the
  shared error taxonomy.
