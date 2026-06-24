# TODO List

**Updated:** 2026-06-22
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).

## Legend

- `[ ]` = Open
- `[x]` = Done
- `[v3]` = Breaking change, deferred to next major (v3)
- `[v4]` = Breaking change, deferred to v4

---

## Open Items

### Architecture & Quality (2026-06-24 multi-skill review)

Surfaced by the code-quality, architecture, data-model, naming, and modularization reviews (see `docs/reviews/`, `docs/architecture-understanding/`, `docs/modularization/`).

- [ ] **Kill the test-dependency leak** [HIGH] — `event`, `command`, `decider`, `watermill` declare `storage/memory` as a direct production `require` but import it only from `_test.go`. Creates a require-graph cycle and undercuts the "event = 3 deps" claim. Swap to `event/eventtest` fakes or in-package stubs, then `go mod tidy` each.
- [ ] **Resolve genproto conflict, then wire transport/grpc** [HIGH] — `transport/grpc` builds only with `GOWORK=off` because the workspace merges monolithic `google.golang.org/genproto` against the split `genproto/googleapis/rpc` (grpc v1.81.1) → ambiguous `googleapis/rpc/status`. Align the workspace on the split modules, then add `transport/grpc` to `go.work` + `flake.nix` `testModules`.
- [ ] **Tune `.golangci.yml`** [HIGH leverage, low effort] — disable `noinlineerr` (anti-idiomatic), scope `makezero` + `exhaustruct` out of `_test.go` / optional-field structs (`event.Metadata`, `command.Metadata`, `http.SSEEvent`). Clears ~135 of 200 findings and surfaces real issues.
- [ ] **Add explicit `TombstoneUndetermined` case** [MEDIUM] — `stack/materialize.go:126` switch silently no-ops on Undetermined. Make intent explicit (case + comment or warning log).
- [ ] **Handle unsupported `reflect.Kind` loudly** [MEDIUM] — `storage/view_store_auto.go:159` switch misses many kinds and falls through silently. Return an error on unsupported types instead.
- [ ] **Exclude generated code from file-size gate** [LOW] — `flake.nix` `check-file-size` flags `transport/grpc/proto/cqrs.pb.go` (530 lines, generated). Exclude `*.pb.go` / `Code generated` files.
- [ ] **Migrate deprecated `query.Handler` usage** [LOW] — `middleware/generic.go:85` references the library's own deprecated type. Move to `TypedHandler[Q,R]`.
- [ ] **Remove `context.Context` from a struct** [LOW] — `watermill/event_bus.go:40` stores a ctx; pass it per-call instead.
- [ ] **Document `AggregateID` string backing** [LOW] — `id.AggregateID` is string-backed while the other 7 IDs are ULID-backed. Either document as intentional (natural keys) + add a validating constructor, or unify on ULID at the next major.

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

_v3.0.0 is tagged (2026-06-22). Open items are transport adapters (ADR-0025, waiting for consumer signal) and Go-stdlib-blocked experiments (jsonv2, arenas). See [ROADMAP.md](ROADMAP.md) for long-term vision._
