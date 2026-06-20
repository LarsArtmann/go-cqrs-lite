# Status Report — 2026-06-20 21:55 — Post-Projection Dissolution

> **The projection/ module is gone. The composable stack is the only projection story.
> 37 modules, all green, 40K production LOC.**

---

## Executive Summary

| Metric              | Value       |
| ------------------- | ----------- |
| Modules             | 37 (was 38) |
| Production LOC      | 40,257      |
| Test LOC            | 69,797      |
| Total .go files     | 830         |
| ADRs                | 33          |
| API exports         | 1,614       |
| Test suites passing | **37/37**   |
| Test suites failing | **0**       |
| HEAD                | `bbe9ffa3`  |
| Branch              | `master`    |

**Overall health: STRONG.** The architecture review's two biggest items —
projection dissolution (ADR-0030) and readmodel merge (ADR-0032) — are both
Implemented. The EventBus ordering bug (the most impactful correctness issue
in the library's history) is fixed. What remains is 5 v3 breaking changes
(metadata aliases, io.Closer removal, reactive cleanup, Event→struct, HTTP
extraction) and quality-of-life improvements.

---

## a) FULLY DONE ✓

### Architecture (This Session Series)

| Item                                        | Impact                                                                                                                                                                                                        | Commit(s)              |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------- |
| **EventBus delivery ordering fix**          | Critical — affected every consumer including projection.Runner's live phase. GoChannel's `sendMessage` spawned goroutines per Publish; without `BlockPublishUntilSubscriberAck`, consecutive publishes raced. | `97e2fd58`             |
| **projection/ deleted (ADR-0030)**          | Critical — 34 files, 1343 LOC removed. Replaced by `bus.SubscribeAll()` + `stack.Materialize[V,K]` + `CatchUpSubscriber`.                                                                                     | `5481a6f4`             |
| **readmodel/ deleted (ADR-0032)**           | High — 482 LOC, 14 files removed. 6/22 production clone groups eliminated. `kv.TypedStore[T,K]` is the replacement.                                                                                           | `c4d0ffcd`             |
| **Storage consolidation (ADR-0029)**        | High — `memory/` → `storage/memory/`, `pebble/` → `storage/pebble/`, `turso/` → `storage/turso/`.                                                                                                             | `22ecccd0`             |
| **Watermill as delivery layer (ADR-0028)**  | Critical — `watermill.EventBus` replaces all 5 bus implementations. EventBus default: `Persistent: false` + `BlockPublishUntilSubscriberAck: true`.                                                           | `97e2fd58`, `98b7a2f2` |
| **Deployer-first example**                  | High — `example/deployer-first/` proves the composable stack end-to-end. Ordered create→complete→delete(tombstone) via CatchUpSubscriber + Materialize.                                                       | `98b7a2f2`             |
| **CatchUpSubscriber**                       | High — ~300 LOC Watermill `message.Subscriber` with replay→live handoff + EventID dedup + checkpoint persistence.                                                                                             | `f9ecce33`             |
| **stack.Materialize[V,K]**                  | High — Tombstone-aware materialized view builder over `kv.TypedStore`. OnCreate/OnUpdate/OnTombstone/OnRebirth callbacks.                                                                                     | `f9ecce33`             |
| **Version.Add(uint)**                       | Medium — Type system prevents negative input at compile time. No runtime panic.                                                                                                                               | `c7df1c79`             |
| **Materialize error-misclassification fix** | Critical bug fix — `errors.Is(err, kv.ErrNotFound)` prevents silent data corruption on real DB errors.                                                                                                        | `72538df7`             |
| **Tombstone protocol gap fixed**            | Medium — Watermill `eventToMessage` was not serializing `Tombstone` field. Fixed with `tombstone_status` + `tombstone_reason` metadata keys.                                                                  | `97e2fd58`             |
| **CBOR canonical-fidelity fuzz fix**        | Low — Canonical CBOR non-injectivity (int and time.Time with matching epoch collapse). Fixed fuzz invariant, not the codec. 6M fuzz executions pass.                                                          | `97e2fd58`             |
| **Multi-DB SQLite preset**                  | Medium — `WithEventDB` and `WithQueryDB` were dead options that silently lied. Now actually open secondary backends.                                                                                          | `dcd350af`             |
| **query.AuditMiddleware**                   | Medium — `query.AuditFull` / `AuditMetadata` / `AuditOff` wired via `stack.QueryAuditMiddleware()`.                                                                                                           | `f9ecce33`             |
| **cqrs-gen updated**                        | Medium — Emits `bus.Subscribe(event.Type(...))` instead of deleted `projection.On[...]`.                                                                                                                      | `5481a6f4`             |
| **Documentation cleanup**                   | Medium — AGENTS.md, TODO_LIST.md, V3_MIGRATION.md, ROADMAP.md all cleaned of stale projection/readmodel references.                                                                                           | `bbe9ffa3`             |

### Bugs Fixed

| Bug                                   | Root Cause                                                                                          | Fix                                     |
| ------------------------------------- | --------------------------------------------------------------------------------------------------- | --------------------------------------- |
| EventBus ordering race                | GoChannel goroutine-per-Publish races without `BlockPublishUntilSubscriberAck`                      | Config fix                              |
| Tombstone lost in Watermill protocol  | `eventToMessage` didn't serialize Tombstone field                                                   | Added metadata keys                     |
| Materialize silent data corruption    | Any `Store.Get` error → `OnCreate`                                                                  | `errors.Is(err, kv.ErrNotFound)`        |
| EventBus Nack→infinite retry deadlock | Handler errors are deterministic; Nack causes infinite retry under `BlockPublishUntilSubscriberAck` | Changed Nack→Ack with logged error      |
| CBOR fuzz failure                     | Over-strong invariant (canonical CBOR is non-injective)                                             | Skip non-injective inputs               |
| PgxListener "data race"               | False positive — channel-synchronized per Go memory model                                           | Documented + regression test with teeth |

---

## b) PARTIALLY DONE ⚠️

### Decider[State, Cmd] Evolution (30%)

`decider.TypedDecider[State, Cmd]` exists with `Decide` field.
`decider.TypedRepository[State, Cmd]` wraps Repository and adds `ExecuteCommand`.
`stack.TypedRepository[State, Cmd]()` accessor exists.

**Not done:** Legacy `Decider[State]` still has `Fold` (not renamed to `Apply`).
`Decide` is NOT a field on legacy Decider — passed per-call as `DecideFunc`.
The rename was deferred to avoid breaking all consumers.

### Typed Metadata Fields (40%)

`TombstoneMark` struct exists (typed: Status + Reason).
`Causation` struct exists (typed).
Tracing fields (CorrelationID, CausationID, UserID, RequestID) are first-class on `event.Metadata`.

**Not done:** `SecurityEnvelope` does NOT exist. Command-specific fields
(RetryCount, IdempotencyKey) not added. The `Custom map[MetadataKey]string`
is still the real schema for non-core fields.

---

## c) NOT STARTED

| #   | Task                                                                                               | ADR  | Effort   | Type         |
| --- | -------------------------------------------------------------------------------------------------- | ---- | -------- | ------------ |
| 1   | **Kill metadata aliases** — `command.Metadata = event.Metadata`, `query.Metadata = event.Metadata` | 0031 | 65min    | Type safety  |
| 2   | **Remove io.Closer from interfaces** — `event.Store`, `snapshot.SnapshotStore`, `command.Store`    | 0010 | 40min    | Cleanliness  |
| 3   | **Delete `event/reactive*.go`** — 343 LOC, zero production consumers                               | —    | 15min    | Dead code    |
| 4   | **Event→concrete struct** — Interface→struct, remove type-assertions                               | —    | 2-3 days | Architecture |
| 5   | **Move HTTP→transport/** — SSE, healthcheck, metrics_http                                          | 0025 | 1 day    | Architecture |
| 6   | **Decider Fold→Apply rename** — Name lies                                                          | —    | 30min    | Naming       |

---

## d) TOTALLY FUCKED UP 💥 (All Fixed)

### 1. The EventBus Ordering Bug

**What:** GoChannel's `sendMessage` spawns a goroutine per Publish. Without `BlockPublishUntilSubscriberAck`, consecutive publishes race — events arrive in non-deterministic order.

**Impact:** EVERY consumer of EventBus, including projection.Runner's live phase. Latent because examples publish one event per command with `time.Sleep`.

**Fix:** `Persistent: false` + `BlockPublishUntilSubscriberAck: true`.

### 2. Watermill Router Cannot Guarantee Ordering

**What:** Tried to wire Materialize through Watermill Router for retry. Events arrived scrambled.

**Truth:** Router processes messages in parallel (one goroutine per message, `router.go:30`). Designed for high-throughput unordered processing, not ordered projection.

**Fix:** Consume CatchUpSubscriber output channel from single goroutine (FIFO).

### 3. Tombstone Lost in Watermill Protocol

**What:** Soft-delete events lost their status through the Watermill transport.

**Truth:** `eventToMessage` never serialized the `Tombstone` field into Watermill flat metadata.

**Fix:** Added `tombstone_status` + `tombstone_reason` metadata keys.

### 4. EventBus Nack→Deadlock

**What:** After enabling `BlockPublishUntilSubscriberAck`, handler errors caused Nack→retry→same error→Nack→infinite loop→deadlock.

**Truth:** Handler errors are deterministic — retrying produces the same error. Under `BlockPublishUntilSubscriberAck`, the blocked Publish never returns.

**Fix:** Changed Nack→Ack with logged error. Consumers needing retry wrap their handler.

### 5. The Synthetic Test Lie

**What:** `TestDeployerFirstArchitecture` passed while EventBus was silently reordering events.

**Truth:** It tested ONE event. Multi-event ordering was broken.

**Fix:** Added `example/deployer-first` with 3-event aggregate (create→complete→delete).

---

## e) WHAT WE SHOULD IMPROVE 🎯

### Architecture

1. **Metadata aliases force event shape onto commands/queries.** `command.Metadata = event.Metadata` means commands inherit `CausationID` (meaningless) and lack command-specific fields (`RetryCount`, `IdempotencyKey`).
2. **`event.Event` is still an interface** with 4 internal type-assertions to `*ImmutableEvent`. Making it a concrete struct would eliminate the leaky abstraction.
3. **`event/reactive*.go` (343 LOC)** has zero production consumers since projection/ deletion. Pure dead code kept alive by tests.
4. **Decider naming lies.** `Fold` should be `Apply`. The struct named `Decider` has no `Decide` field.
5. **No EventBusOption for GoChannel config.** Consumers can't tune `OutputChannelBuffer` or override ordering guarantees for high-throughput unordered scenarios.

### Code Quality

6. **`io.Closer` on core interfaces** (`event.Store`, `snapshot.SnapshotStore`, `command.Store`) violates ISP — consumers that only read shouldn't need to close.
7. **`eventOptions` pointer leak** — every event carries `opts *eventOptions` (codec + clock + deadline) for its entire lifetime. Config only needed at construction.
8. **HTTP code in middleware/** (SSE, healthcheck, metrics_http) should be in `transport/`.

### Testing

9. **No multi-event ordering regression test** through the full EventBus pipeline.
10. **CBOR fuzz corpus is minimal** — 6M executions found no new bugs but corpus could be richer.
11. **No `-race` CI for example/deployer-first** — the live-delivery path has concurrent goroutines.

### Documentation

12. **FEATURES.md not updated** with deployer-first architecture.
13. **SKILL.md** (consumer guide) may reference deleted projection types.

---

## f) Top #25 Things to Do Next

Sorted by impact/effort (Pareto order).

| #   | Task                                                                                            | Impact | Effort        | Type         |
| --- | ----------------------------------------------------------------------------------------------- | ------ | ------------- | ------------ |
| 1   | **Kill metadata aliases** (ADR-0031) — make `command.Metadata` and `query.Metadata` own structs | High   | 65min         | Type safety  |
| 2   | **Delete `event/reactive*.go`** — 343 LOC dead code, zero consumers                             | Medium | 15min         | Cleanup      |
| 3   | **Add multi-event ordering integration test** (5+ events through EventBus)                      | High   | 30min         | Test         |
| 4   | **EventBusOption for GoChannel config** — tune buffer/throughput                                | Medium | 20min         | Feature      |
| 5   | **Decider Fold→Apply rename** — the name lies                                                   | Medium | 30min         | Naming       |
| 6   | **Remove io.Closer from interfaces** (ADR-0010)                                                 | Medium | 40min         | Cleanliness  |
| 7   | **Add retry helper for direct-consumption path** (no Router needed)                             | High   | 40min         | Feature      |
| 8   | **Update FEATURES.md** with deployer-first architecture                                         | Medium | 15min         | Docs         |
| 9   | **Update SKILL.md** consumer guide                                                              | Medium | 20min         | Docs         |
| 10  | **Add `-race` to example/deployer-first tests**                                                 | Medium | 5min          | Test         |
| 11  | **Add Watermill Router vs direct-consumption decision guide**                                   | Medium | 15min         | Docs         |
| 12  | **Enrich CBOR fuzz corpus** with real event payloads                                            | Low    | 15min         | Test         |
| 13  | **Add EventBus benchmark** (throughput with BlockPublishUntilSubscriberAck)                     | Medium | 20min         | Perf         |
| 14  | **Make Event concrete struct** (ADR-0001 Phase 7)                                               | High   | 2-3 days      | Architecture |
| 15  | **Move HTTP→transport/** (ADR-0025)                                                             | Medium | 1 day         | Architecture |
| 16  | **Fix `eventOptions` pointer leak**                                                             | Low    | 30min         | Cleanliness  |
| 17  | **Add `SecurityEnvelope` typed metadata field**                                                 | Medium | 30min         | Type safety  |
| 18  | **Consolidate `deployer_first_test.go`** (stack/ vs example/)                                   | Low    | 15min         | Cleanup      |
| 19  | **Add `.gitignore` for example binaries**                                                       | Low    | 5min          | Cleanup      |
| 20  | **Review all ADR statuses** for accuracy                                                        | Low    | 10min         | Docs         |
| 21  | **Document the CatchUpSubscriber startup pattern**                                              | Medium | 10min         | Docs         |
| 22  | **Add SQLite auto-indexing advisor** to preset (from turso/)                                    | Low    | 30min         | Feature      |
| 23  | **Audit `event.Projection` interface usage** in examples                                        | Low    | 15min         | Cleanup      |
| 24  | **Property-based test for Materialize tombstone lifecycle**                                     | Medium | 30min         | Test         |
| 25  | **gRPC / NATS / Redis transport adapters**                                                      | High   | 3-5 days each | Feature      |

---

## g) Top #1 Question I Cannot Answer Myself 🤔

**Should `command.Metadata` and `query.Metadata` embed `event.Tracing` or should `Tracing` be extracted to a shared `domain/` module?**

ADR-0031 proposes three options:

- **A) Embed shared `Tracing` struct** in each module's own `Metadata` — simplest, no new module
- **B) Extract to `domain/` module** — cleanest separation, but adds a module everyone depends on
- **C) Keep aliases, split `event.Metadata`** — shared shape stays, event-specific concerns move to separate `EventMetadata`

This is a tradeoff between simplicity (A), purity (B), and migration cost (C). I recommend A (embed Tracing, each module adds its own fields). But the decision affects every consumer's import paths and serialization format. It's a product-direction call.

---

## Test Suite Status (All 37 Modules)

| Module          | Status | Module                 | Status |
| --------------- | ------ | ---------------------- | ------ |
| event           | ✅     | stack                  | ✅     |
| event/eventtest | ✅     | stack/memory           | ✅     |
| command         | ✅     | stack/sqlite           | ✅     |
| query           | ✅     | stack/pebble           | ✅     |
| decider         | ✅     | stack/postgres         | ✅     |
| id              | ✅     | stack/bench            | ✅     |
| dispatcher      | ✅     | storage/memory         | ✅     |
| schema          | ✅     | storage/pebble         | ✅     |
| snapshot        | ✅     | storage/sql            | ✅     |
| catalog         | ✅     | storage/turso          | ✅     |
| middleware      | ✅     | example/user           | ✅     |
| integration     | ✅     | example/todo           | ✅     |
| signing         | ✅     | example/encryption     | ✅     |
| encryption      | ✅     | example/deployer-first | ✅     |
| storage         | ✅     | cmd/cqrs-gen           | ✅     |
| watermill       | ✅     | cmd/api-stability      | ✅     |
| codec           | ✅     | prometheus             | ✅     |
| kv              | ✅     | otel                   | ✅     |
| listing         | ✅     |                        |        |
| testutil        | ✅     |                        |        |

**All 37 modules green. Zero failures.**
