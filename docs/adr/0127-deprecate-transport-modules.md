# ADR-0127: Deprecate transport/* Modules — External Delivery via watermill/ + go-sse

**Date:** 2026-08-14
**Status:** Accepted
**Supersedes:** ADR-0025 (transport adapter strategy) in full
**Builds on:** ADR-0123 (v5 unification horizon)

---

## Context

ADR-0025 established a `transport/` boundary for process-boundary delivery.
Two modules shipped under it:

- `transport/http` — a from-scratch SSE stack (~25 files): SSEBroker,
  backfill handler, replay budget, CBOR→JSON transform, graceful close.
- `transport/grpc` — remote command/query dispatch over gRPC with generated
  protobuf.

Four facts argue against both:

1. **The README's own doctrine.** "Not a framework — no opinionated
   transport, message broker, or SQL driver." `transport/` contradicts it.
2. **Better dedicated libraries exist.** `github.com/larsartmann/go-sse`
   already provides generic SSE fanout — and `metaengine/sse.go` (ServeSSE)
   already builds on it. The sanctioned broker story has been the
   `watermill/` bridge since 2026-06-28 (ADR-0025's own NATS/Redis
   correction): `NewEventPublisher(message.Publisher, topic)` and
   `WithBackend()` accept any Watermill-compatible backend (NATS, Redis,
   Kafka).
3. **Zero internal adoption of gRPC.** `transport/grpc` has no example, no
   integration suite, no consumer outside cqrs-lint fixtures — pure
   maintenance tax (proto codegen, own go.mod, race-threshold exceptions).
4. **The linter coached users onto a parallel path.** cqrs-lint's E009/F013
   suggested `transport/http`/`transport/grpc`, blind to the go-sse and
   watermill paths the library itself uses.

## Decision

1. **Deprecate `transport/http` and `transport/grpc`.** Deprecation notices
   in `doc.go` + README. Removal at v5 (per ADR-0123's breaking-cut horizon).
2. **The sanctioned external-delivery paths are:**

   | Need                                | Path                                                                                                           |
   | ----------------------------------- | -------------------------------------------------------------------------------------------------------------- |
   | SSE delivery                        | `github.com/larsartmann/go-sse` (already used by `metaengine.ServeSSE`)                                        |
   | Broker transport (NATS/Redis/Kafka) | `watermill/` bridge: `NewEventPublisher` / `NewCommandPublisher` / `WithBackend()` + official Watermill plugin |
   | HTTP UI delivery                    | cqrs-htmx                                                                                                      |

3. **cqrs-lint coaches the sanctioned paths.** `HasTransport` now detects
   `watermill/`, `go-sse`, `cqrs-htmx`, and (for legacy projects) the
   deprecated `transport/*` imports. E009/F013 messages point at watermill +
   go-sse. `transport/http`/`transport/grpc` are removed from the module
   catalog — deprecated modules are not adoption targets.
4. **No replacement module is planned.** Broker transports are the
   watermill plugin ecosystem, not go-cqrs-lite surface area. The
   `docs/design/transport-{nats,redis}.md` spikes are marked superseded.

## Consequences

- Consumers on `transport/http` SSE keep compiling until v5; migration is
  go-sse (`metaengine.ServeSSE` for read-model push) or watermill for broker
  fanout.
- Consumers on `transport/grpc` bridge their dispatcher over grpc-go
  directly; the adapter is thin by design.
- cqrs-lint no longer counts transport/* as scored modules (catalog shrank
  34 → 32 scored entries).
- The v5 cut (ADR-0123 Phase 8) deletes both modules; the v5 migration
  guide covers the mapping table above.

## Migration

| Deprecated                    | Replacement                                                           |
| ----------------------------- | --------------------------------------------------------------------- |
| `http.NewSSEBroker(bus)`      | go-sse broker (`metaengine.ServeSSE` for materialized views)          |
| `http.BackfillHandler`        | watermill `CatchUpSubscriber` (checkpointing + DLQ)                   |
| `http.CBORToJSONTransform`    | go-codec transcode (CBOR decode → JSON encode)                        |
| `grpc.RegisterCommandService` | grpc-go server bridging dispatcher directly, or watermill command bus |
| `grpc.NewCommandClient`       | watermill `NewCommandPublisher` over a broker                         |

---

**Related:** [ADR-0025](0025-transport-adapter-strategy.md) (superseded),
[ADR-0123](0123-v5-unification-single-composition-root.md) (v5 cut),
[`watermill/`](../../watermill/) module,
[`docs/planning/nats-transport-design.md`](../planning/nats-transport-design.md)
(JetStream recipe).
