# TODO List

**Updated:** 2026-06-28
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).

## Legend

- `[ ]` = Open
- `[x]` = Done
- `[v3]` = Breaking change, deferred to next major (v3)
- `[v4]` = Breaking change, deferred to v4

---

## Open Items

### Module Isolation (2026-06-28)

- [x] ~~**Fix signing/go.mod: add eventtest dep**~~ — DONE (`179f5ee3`). The signing module failed GOWORK=off build because eventtest was missing from go.mod.
- [x] ~~**Fix projection/v3 deps across 11 modules**~~ — DONE (`f9cc123c`). A full isolation audit found 11 of 44 modules failing GOWORK=off build because the projection/v3 require+replace directives were missing. Root cause: TODO_LIST falsely claimed projection/ was 'deleted' — it was alive and load-bearing.
- [x] ~~**Add CI isolation gate**~~ — DONE (`c45b47c8`). The per-module-test CI job had a hardcoded 34-module matrix, missing 10+ modules. Replaced with dynamic discovery + a check-module-isolation.sh gate.
- [x] ~~**Add doc-stub CI gate**~~ — DONE (`c45b47c8`). Greps for placeholder `// Package X provides ...` in doc.go files.

### Architecture & Quality (2026-06-24 multi-skill review)

Surfaced by the code-quality, architecture, data-model, naming, and modularization reviews (see `docs/reviews/`, `docs/architecture-understanding/`, `docs/modularization/`).

- [x] ~~**Kill the test-dependency leak**~~ — **NOT A LEAK.** The `storage/memory` dependency from `event`/`command`/`decider`/`watermill` is a **documented exception** in `check-module-layers.sh:46-53`. Core modules use the real in-memory store for integration tests rather than duplicating fakes. The layer check PASSES. Corrected after brutal self-review (2026-06-24).
- [ ] **Resolve genproto conflict, then wire transport/grpc** [BLOCKED UPSTREAM] — `transport/grpc` builds only with `GOWORK=off` because `cockroachdb/pebble` → `cockroachdb/errors@v1.14.0` pulls the monolithic `google.golang.org/genproto`, which conflicts with grpc v1.81.1's split `genproto/googleapis/rpc`. This is a known ecosystem issue ([cockroachdb/errors#79](https://github.com/cockroachdb/errors/issues/79)). The monolithic genproto was removed from `integration/go.mod` via tidy, but `storage/pebble` still transitively requires it via `cockroachdb/pebble`. **Blocked until cockroachdb/errors drops the monolithic genproto or grpc drops the split version.** Until then, test with `cd transport/grpc && GOWORK=off go test ./...`.
- [x] ~~**Tune `.golangci.yml`**~~ — **DONE**. Disabled `noinlineerr`, set `makezero.always=false`, added test-file exclusions for `errcheck`, `nilnil`, `varnamelen`, `ginkgolinter`. All modules now pass clean (0 findings).
- [x] ~~**Add explicit `TombstoneUndetermined` case**~~ — **DONE** (`cbfdd68e`). Switch in `stack/materialize.go` now handles all 3 tombstone statuses explicitly.
- [x] ~~**Handle unsupported `reflect.Kind` loudly**~~ — **DONE** (`cbfdd68e`). Added scoped `//nolint:exhaustive` with justification on `goTypeToSQL` — unknown kinds intentionally default to TEXT for JSON-serialized types.
- [x] ~~**Exclude generated code from file-size gate**~~ — **DONE**. `flake.nix` and CI exclude `*.pb.go` and `*.gen.go`.
- [x] ~~**Migrate deprecated `query.Handler` usage**~~ — **ACCEPTED WITH JUSTIFICATION**. `AsQuery` in `middleware/generic.go` intentionally bridges to the deprecated API (nolint with comment). Test usage in `retry_query_test.go` also nolint'd. Both are backward-compatibility bridges that will be removed in v4 when `query.Handler` is deleted.
- [x] ~~**Remove `context.Context` from a struct**~~ — **ACCEPTED WITH JUSTIFICATION**. `watermill/event_bus.go` stores a lifecycle context (`subCtx`) for the background subscriber goroutine. This is created from `context.Background()` and cancelled on `Close()` — not the anti-pattern the linter targets (request-scoped contexts in structs). Refactoring would complicate the subscriber lifecycle without safety benefit.
- [x] ~~**Document `AggregateID` string backing**~~ — **ALREADY DOCUMENTED** (`id/aggregate_id.go:19-26`). The string backing is intentional: AggregateID supports both ULID-generated IDs and domain-specific natural keys (e.g. `"lock_user1_user2"`). No change needed.

### Experimental / long-term

- [x] **gRPC transport adapter** — ADR-0025 accepted. `transport/grpc/` module with protobuf command + query dispatch (commit `81d29455`).
- [ ] **NATS/Redis Stream adapter** — ADR-0025 accepted. Separate `transport/nats/` and `transport/redis/` modules.
- [ ] **jsonv2 codec experiment** — `codec/jsonv2_experiment.go` exists behind `goexperiment.jsonv2` build tag (ADR-0026). Pending Go stdlib stabilization.
- [ ] **Arena allocation experiment** — `event/arena_experiment.go` exists behind `goexperiment.arenas` build tag (ADR-0026). Pending Go arena API stabilization.
- [ ] **Hot-State cache (decider)** — Optional `RepositoryOption[State]` that caches folded aggregate state keyed by `(aggID, version)` to eliminate snapshot+delta replay on sustained-hot aggregates. Safe under single-master because optimistic concurrency (`expectedVersion` in `Save`) already gates writes. Write-through: update or invalidate on every successful `Execute`. Lives above singleflight (concurrent-burst coalescing) and above `loadFromSnapshot`/`loadFromStore` (sequential-burst coalescing). Cold start pays the replay cost once; not a correctness concern. Profile before building — snapshot + page-cache-resident events already make sequential loads cheap; this only pays off for aggregates commanded 100+ times/sec.
- [ ] **Read-pressure snapshot strategy** — `EveryNEvents` snapshots based on writes, but reads are the expensive path in ES (writes are already sequential append). Add a `ReadPressureStrategy` that snapshots based on load frequency since last snapshot (with a minimum-delta guard to avoid snapshotting unchanged aggregates). Smaller payoff than the State cache because the cache subsumes most of its benefit; consider after the State cache lands.

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
- [x] ~~Refactor projection/ module~~ — **RE-INTRODUCED** (ADR-0030). The OLD projection/ runner was replaced by bus.SubscribeAll + stack.Materialize + CatchUpSubscriber. But the Projection *interface* was re-homed into projection/ as a shared contract — it is now implemented by storage.RelationalProjection, graph.GraphProjection, and stack.Materialize. It is alive and load-bearing for the relational + graph tiers (ADR-0033). The original "delete" claim was stale; corrected 2026-06-28.
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

_v3.1.0 is tagged (2026-06-25). Open items are transport adapters (ADR-0025, waiting for consumer signal), performance features (hot-state cache, read-pressure snapshots), and Go-stdlib-blocked experiments (jsonv2, arenas). See [ROADMAP.md](ROADMAP.md) for long-term vision._
