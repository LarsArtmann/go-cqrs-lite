# TODO List

**Updated:** 2026-07-05
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).

## Legend

- `[ ]` = Open
- `[x]` = Done
- `[v3]` = Breaking change, deferred to next major (v3)
- `[v4]` = Breaking change, deferred to v4

---

## Open Items

### Operability

- [ ] **Surface Pebble Checkpoint from stack presets** — `pebble.Backend.Checkpoint(dir)` exists but isn't exposed through `stack/pebble.Bundle`. Consumers using presets can't access backup without dropping to raw modules.
- [ ] **Surface graceful shutdown from stack presets** — `pebble.Backend.Close()` exists but no context-bounded `GracefulClose(ctx)` variant. Documented in AGENTS.md but not yet implemented.

### Architecture & Quality

- [ ] **Resolve genproto conflict for transport/grpc** [BLOCKED UPSTREAM] — `transport/grpc` builds only with `GOWORK=off` because `cockroachdb/pebble` → `cockroachdb/errors@v1.14.0` pulls the monolithic `google.golang.org/genproto`, which conflicts with grpc v1.81.1's split `genproto/googleapis/rpc`. Blocked until cockroachdb/errors drops the monolithic genproto or grpc drops the split version.

### Experimental / long-term

- [ ] **NATS/ValKey Stream adapter** — ADR-0025 accepted. Separate `transport/nats/` and `transport/redis/` modules. _(Author is not a fan of Redis; [ValKey](https://valkey.io) is the recommended alternative for consumers who want Redis-compatible infrastructure.)_
- [ ] **jsonv2 codec experiment** — `codec/jsonv2_experiment.go` exists behind `goexperiment.jsonv2` build tag (ADR-0026). Pending Go stdlib stabilization.
- [ ] **Arena allocation experiment** — `event/arena_experiment.go` exists behind `goexperiment.arenas` build tag (ADR-0026). Pending Go arena API stabilization.
- [ ] **Hot-State cache (decider)** — Optional `RepositoryOption[State]` that caches folded aggregate state keyed by `(aggID, version)` to eliminate snapshot+delta replay on sustained-hot aggregates. Profile before building — snapshot + page-cache-resident events already make sequential loads cheap; this only pays off for aggregates commanded 100+ times/sec.
- [ ] **Read-pressure snapshot strategy** — `EveryNEvents` snapshots based on writes, but reads are the expensive path in ES. Add a `ReadPressureStrategy` that snapshots based on load frequency. Consider after hot-state cache.

### v4 Breaking Changes (deferred)

- [v4] **Flip codec defaults** — Events default to JSON, blind stores (KV/snapshot/command/query) default to JSON. v4 flips both to CBOR. Migration guide needed.
- [v4] **Remove deprecated APIs** — `query.Handler` (replaced by `TypedHandler`), `memory.MemoryBus` (replaced by `watermill.EventBus`).

---

## Recently Completed

- [x] **DOMAIN_LANGUAGE.md rebuild** — Complete rewrite with all 47 modules verified against source code. Embedded verification code block for doc-check (89 refs).
- [x] **doc-check false-negative fix** — Tool now warns on 0 references instead of reporting false success. Fixed repoRoot resolution and added DOMAIN_LANGUAGE.md to default path.
- [x] **SKILL.md doc-check refs fixed** — `query.QueryRecovery/Logging/Metrics` → `middleware.*` (3 broken references).
- [x] **Dead code removed** — `errRelational` (storage/relational), `errUnmarshalResult` (transport/grpc), `mustOpenDB` (storage/view).
- [x] **Error taxonomy sweep** — All `fmt.Errorf` calls classified into 5-family taxonomy.
- [x] **Deriver module** — Event→command derivation: `Deriver`, `Then`, `Filter`, `Idempotent`, `AsHandler` (ADR-0040).
- [x] **Scenario-testing DSL** — Fluent Given/When/Then for deciders + projections (`scenario/`).
- [x] **Scheduling module** — Durable deadline timers: `TimerStore`, `Scheduler` with retry.
- [x] **Managed projection host** — `projectionhost.Host` with crash-restart, DLQ, batch processing.
- [x] **Idempotency module** — Command dedup for at-least-once delivery.
- [x] **Three projection tiers** — Materialize/KV, RelationalProjection/SQL, GraphProjection/graph.
- [x] **Watermill EventBus + CommandBus** — Event AND command pub/sub over any broker.
- [x] **gRPC transport** — Remote command/query/event dispatch.
- [x] **SSE transport** — Server-Sent Events with Last-Event-ID reconnection.
- [x] **CBOR compact codec** — ~35% smaller payloads via positional array encoding.
- [x] **Event signing** — HMAC-SHA256, Ed25519, multisig.
- [x] **Event encryption** — XChaCha20-Poly1305, AES-256-GCM.
- [x] **Bundle composition layer** — `Bundle` with ISP-honest fields + 5 presets (memory, sqlite, pebble, postgres, turso).
- [x] **KV store abstraction** — `kv.Store` (Reader+Writer+Closer), Pebble `KVAdapter`.
- [x] **Typed read-model store + cache** — `kv.TypedStore[T,K]` + `kv.Cache[T,K]` (ADR-0032).
- [x] **Prometheus metrics exporter** — OTel→Prometheus bridge.
- [x] **Schema registry validator** — `Validator` with `RegisterType[T]()`.

---

_v3.6.0 tagged (2026-07-05). All framework gaps (A1–A6) shipped: projectionhost, scenario, scheduling, deriver, idempotency. Open items: operability surfacing, Go-stdlib-blocked experiments (jsonv2, arenas), performance features (hot-state cache). See [ROADMAP.md](ROADMAP.md) for long-term vision._
