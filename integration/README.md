# integration — Cross-Module Tests

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/integration/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/integration/v3)

End-to-end tests that verify multiple go-cqrs-lite modules work together correctly.

## What's Tested

| Package                | Modules Covered        | What It Tests                                      |
| ---------------------- | ---------------------- | -------------------------------------------------- |
| `integration/command/` | command, event, memory | Command dispatch → event store → bus roundtrip     |
| `integration/event/`   | event, memory          | Event creation, store, load, bus publish/subscribe |
| `integration/query/`   | query, event, memory   | Query dispatch → typed result extraction           |
| `integration/signing/` | signing, event, memory | HMAC/Ed25519 sign → verify → middleware chain      |

## Running

```bash
# All integration tests
go test ./integration/... -count=1

# Specific package
go test ./integration/signing/... -v -count=1
```

## Related Modules

- [**command/v2**](../command/README.md) — Command dispatch roundtrip tests
- [**event/v2**](../event/README.md) — Event store/load/bus integration
- [**query/v2**](../query/README.md) — Typed result extraction tests
- [**memory/v2**](../memory/README.md) — In-memory store/bus used as test fixtures
- [**signing/v2**](../signing/README.md) — Sign → verify → middleware chain tests
