# TODO List

**Updated:** 2026-06-21
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

_All v3 breaking changes are complete._ The event Core immutability item was
investigated and verified safe: `eventOptions` holds only immutable types
(`Clock` func, `codec.Codec` interface, `time.Time` value), so the shallow
struct copy in `Clone()` is semantically a deep copy. Regression test
`TestClone_IndependentOpts` (`event/event_type_clone_test.go:229`) locks this in.

### Completed v3 Breaking Changes

- [x] ~~Move memory/ stores → storage/memory/~~ — DONE
- [x] ~~Version → uint64~~ — DONE
- [x] ~~Delete readmodel/ module~~ — DONE (merged into kv/, ADR-0032)
- [x] ~~Delete projection/ module~~ — DONE (replaced by bus.SubscribeAll + stack.Materialize + CatchUpSubscriber, ADR-0030)
- [x] ~~Fix query.Handler returns any~~ — TypedHandler shipped
- [x] ~~Delete ghost bus code~~ — DONE (`event/reactive*.go` removed; watermill.EventBus + bus.SubscribeAll is the replacement)
- [x] ~~Remove io.Closer from core interfaces~~ — DONE (ADR-0010; callers type-assert to io.Closer)
- [x] ~~Break command/query Metadata = event.Metadata alias~~ — DONE (ADR-0031; each module owns its Metadata embedding event.Tracing)
- [x] ~~Rename Decider.Fold → Apply~~ — DONE (naming honesty)
- [x] ~~Make event.Event a concrete type~~ — DONE (`type Event = *ImmutableEvent`; interface removed, 7 type assertions deleted)
- [x] ~~Move SSE to transport/http/~~ — DONE (ADR-0025; SSE moved, healthcheck/metrics_http/pprof deleted — generic utilities with zero CQRS deps and zero consumers)

---

## Recently Completed

- [x] **watermill.EventBus adapter** — Full `event.Bus` implementation using Watermill GoChannel. All 4 stack presets migrated from `memory.MemoryBus`.
- [x] **TransactionID branded type** — Deleted in v2.8 (ghost code, zero consumers). Re-adding requires a real consumer need + proper wiring through `event.Metadata` + `WithTransactionID()` Option. Tracked in ROADMAP as long-term.
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

_All v3 breaking changes complete. Open items are transport adapters (ADR-0025, waiting for consumer signal) and Go-stdlib-blocked experiments (jsonv2, arenas). See [ROADMAP.md](ROADMAP.md) for long-term vision._
