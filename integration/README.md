# integration — Cross-Module Tests

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/integration/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/integration/v2)

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
