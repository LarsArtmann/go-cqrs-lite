# TODO List

**Updated:** 2026-06-20
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).

## Legend

- `[ ]` = Open
- `[x]` = Done
- `[v3]` = Breaking change, deferred to next major (v3)
- `[v4]` = Breaking change, deferred to v4

---

## Open Items

### Experimental / long-term

- [ ] **gRPC transport adapter** — ADR-0025 accepted. Separate `transport/grpc/` module with protobuf dispatch.
- [ ] **NATS/Redis Stream adapter** — ADR-0025 accepted. Separate `transport/nats/` and `transport/redis/` modules.
- [ ] **jsonv2 codec experiment** — `codec/jsonv2_experiment.go` exists behind `goexperiment.jsonv2` build tag (ADR-0026). Pending Go stdlib stabilization.
- [ ] **Arena allocation experiment** — `event/arena_experiment.go` exists behind `goexperiment.arenas` build tag (ADR-0026). Pending Go arena API stabilization.

### v3 Breaking Changes (remaining)

- [v3] **Delete ghost bus code** — `event/reactive*.go` (343 LOC). Replacement: `watermill.EventBus` + `bus.SubscribeAll`. Zero production consumers since projection/ deletion.
- [v3] **Break command/query Metadata = event.Metadata alias** — `storage/sql.MarshalMetadata` takes `event.Metadata`. Cascades through SQL stores. (ADR-0031)
- [v3] **Remove io.Closer from core interfaces** — ADR-0010 accepted. Affects `event.Store`, `snapshot.SnapshotStore`, `command.Store`.
- [v3] **Move HTTP code out of middleware** — SSE, healthcheck, metrics_http → transport/ module.
- [v3] **Make event Core truly immutable** — Currently opts pointer is shallow-copied on Clone.

### Completed v3 Breaking Changes

- [x] ~~Move memory/ stores → storage/memory/~~ — DONE
- [x] ~~Version → uint64~~ — DONE
- [x] ~~Delete readmodel/ module~~ — DONE (merged into kv/, ADR-0032)
- [x] ~~Delete projection/ module~~ — DONE (replaced by bus.SubscribeAll + stack.Materialize + CatchUpSubscriber, ADR-0030)
- [x] ~~Fix query.Handler returns any~~ — TypedHandler shipped

---

## Recently Completed

- [x] **watermill.EventBus adapter** — Full `event.Bus` implementation using Watermill GoChannel. All 4 stack presets migrated from `memory.MemoryBus`.
- [ ] **TransactionID branded type** — Cross-aggregate consistency tracking phantom type (deleted in v2.8 — was ghost code with zero consumers; needs proper wiring before re-adding).
- [x] **Version drift CI check** — `scripts/check-version-drift.sh` detects sibling module version mismatches.
- [x] **CI file-size gate fix** — Subshell bug fixed, gate now actually works.
- [x] **File-size compliance** — All production files under 350 lines (7 files split).
- [x] **Error taxonomy** — All `fmt.Errorf` calls classified into 5-family taxonomy.
- [x] **Dead build tags removed** — goroutineleakprofile, runtimesecret, simd removed; goexperiment.jsonv2 added.
- [x] **Security theater removed** — gosec `-no-fail`, vulncheck/secrets-scan `|| true` removed.
- [x] **Deprecated notices** — Added to `memory.MemoryBus`, `memory.MemoryCommandBus`, `query.Handler`.
- [x] **Streaming event reads** — `StreamingSource`/`StreamingJournal` on SQL, Pebble, Memory stores.
- [x] **Distributed checkpointing** — `DistributedRunner` with `LeaderElection` gating.
- [x] **cqrs-gen v3** — Event handler generation via `-type=event`.
- [x] **Postgres LISTEN/NOTIFY event bus** — `storage.PostgresBus` with PgxListener.
- [x] **WASM compilation** — 7/7 core modules compile to WASM.
- [x] **Bundle composition layer** — `Bundle` with ISP-honest fields + 4 presets (memory, sqlite, pebble, postgres).
- [x] **Typed read-model store + cache** — Merged into `kv.TypedStore[T,K]` + `kv.Cache[T,K]` (ADR-0032). readmodel/ deleted.
- [x] **Typed stores** — `TypedSnapshot[State]`, `TypedCommandStore[P]`, `TypedQueryStore[P]`.
- [x] **Schema registry validator** — `Validator` with `RegisterType[T]()`.
- [x] **Prometheus metrics exporter** — `prometheus/` module with OTel bridge.
- [x] **KV store abstraction** — `kv.Store` (Reader+Writer+Closer), Pebble `KVAdapter`.
- [x] **CBOR compact codec** — ~35% smaller payloads via positional array encoding.
- [x] **Event signing** — HMAC-SHA256, Ed25519, multisig.
- [x] **Event encryption** — XChaCha20-Poly1305, AES-256-GCM.

---

_5 v3 breaking changes remaining + completed items. See [ROADMAP.md](ROADMAP.md) for long-term vision._
