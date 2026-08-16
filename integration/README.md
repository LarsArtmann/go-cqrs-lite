# integration — Cross-Module Integration Tests

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/integration/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/integration/v4)

End-to-end tests that verify multiple go-cqrs-lite modules work together correctly. These are NOT unit tests — they exercise the full pipeline with real in-memory implementations.

## What's Tested

### Sub-package suites

| Package                   | Modules Covered           | What It Tests                                             |
| ------------------------- | ------------------------- | --------------------------------------------------------- |
| `integration/command/`    | command, event, memory    | Command dispatch, event store roundtrip, bus publish      |
| `integration/event/`      | event, memory             | Event creation, store, load, bus publish/subscribe        |
| `integration/query/`      | query, event, memory      | Query dispatch, typed result extraction                   |
| `integration/signing/`    | signing, event, memory    | HMAC/Ed25519 sign then verify, middleware chain order     |
| `integration/encryption/` | encryption, event, memory | Encrypt then decrypt, mixed encrypted/unencrypted streams |

### Root-package suites

| Suite (`*_test.go`)             | Modules Covered                   | What It Tests                                                       |
| ------------------------------- | --------------------------------- | ------------------------------------------------------------------- |
| `full_flow_test.go`             | command, event, projection, query | The whole CQRS loop: dispatch → events → projection → query         |
| `actor_propagation_test.go`     | id, command, event                | `id.ActorID` propagates end-to-end into event metadata              |
| `chaos_test.go`                 | command, middleware               | Handler errors/panics with and without recovery middleware          |
| `error_classification_test.go`  | errorfamily, command, event       | 6-family classification across wrapped chains                       |
| `graph_projection_test.go`      | stack, graph                      | `Bundle.RunProjections` driving a `GraphProjection`                  |
| `idempotency_test.go`           | middleware, command               | Idempotency dedup across retried dispatches                          |
| `metaengine_test.go`            | metaengine, projectionadapter     | Counter + Map pipelines through the planner                          |
| `otel_integration_test.go`      | otel, middleware, command, event  | Spans emitted from command dispatch and event bus                    |
| `otel_span_tree_test.go`        | otel, middleware                  | Parent/child span tree shape end-to-end                              |
| `pebble_test.go`                | storage/pebble, decider, snapshot | Real Pebble event store + projection runner; snapshot + Repository  |
| `snapshot_test.go`              | snapshot, event                   | Snapshot serialization roundtrip                                     |

Plus `simulation/` — a deterministic event generator package (not an
assertion suite) used to drive load/simulation scenarios.

For backend-specific integration suites (Postgres, MySQL, engines), see the
flake apps: `nix run .#integration-pg`, `.#integration-mysql-nspawn` /
`.#integration-mysql-vm`, `.#test-integration`, `.#test-all-backends`.

## Running

```bash
# All integration tests
go test ./integration/... -count=1

# Specific package
go test ./integration/signing/... -v -count=1

# With race detector
go test ./integration/... -count=1 -race
```

## Design

- **Real implementations, not mocks**: Uses `storage/memory` stores and `watermill.EventBus` for realistic behavior.
- **Cross-module coverage**: Each test exercises at least 2-3 modules together, catching integration bugs that unit tests miss.
- **Deterministic**: No external dependencies, no network, no flaky timing.

## Related Modules

- [**command**](../command/README.md) — Command dispatch roundtrip tests
- [**event**](../event/README.md) — Event store/load/bus integration
- [**query**](../query/README.md) — Typed result extraction tests
- [**signing**](../signing/README.md) — Sign then verify, middleware chain tests
- [**encryption**](../encryption/README.md) — Encrypt then decrypt integration
- [**storage/memory**](../storage/memory/README.md) — In-memory store/bus used as test fixtures
