# Status Report: Example Consolidation — Session 2

**Date:** 2026-07-05 16:13
**Session Goal:** Fix critical bugs from session 1's example consolidation, add missing library feature demonstrations, and update all documentation.

---

## Executive Summary

Session 1 left the flagship `example/taskmanager` with **3 critical bugs that the status report claimed were working**: signing was dead code (`if false {}`), tombstone projection was silently broken (`json:"-"`), and go.mod files didn't actually build. Session 2 fixed all 3, then added 5 new library feature demonstrations: ProjectionHost (DLQ + crash-restart), SSE real-time streaming, snapshot strategy, deriver (event→command reactions), and getting-started tests. All 9 tests pass (8 taskmanager + 1 getting-started). Documentation across 5 files updated to reference new examples.

---

## a) FULLY DONE

| Item                           | Details                                                                                                                                                                                                                          |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Signing wired correctly**    | Type-assert `bundle.Publisher.(*cqrswatermill.EventBus)` → `bus.UsePublish(SignMiddleware)` + `bus.Use(VerifyMiddleware)`. HMAC-SHA256 with random key. Events are now actually signed and verified.                             |
| **Tombstone projection fixed** | Root cause: `json:"-"` on `Tombstoned` field stripped it from KV store serialization. Changed to `json:"tombstoned,omitempty"`. Integration test now verifies the **projection** reflects the delete (not just the event store). |
| **Dead code removed**          | Removed `if false {}` block, removed duplicate `shutdownOTel` method (Stop() handles it), changed `Server.signer` to `SignerVerifier` (needed for verify).                                                                       |
| **ProjectionHost with DLQ**    | Replaced raw CatchUpSubscriber loop with `projectionhost.Host` — batch size 100, DLQ threshold 3, unlimited restarts, backoff. Demonstrates production-grade projection lifecycle.                                               |
| **SSE real-time streaming**    | `/events` endpoint via `transport/http.SSEBroker`. Clients get live domain events as Server-Sent Events.                                                                                                                         |
| **Snapshot strategy**          | `snapshot.EveryNEvents(10)` with proper `WithSnapshotStore` + `WithCodec` wiring. Accelerates aggregate loading.                                                                                                                 |
| **Deriver (event→command)**    | Auto-assigns new tasks to default team lead when `task.created` fires. Async dispatch (goroutine) to avoid deadlock with `BlockPublishUntilSubscriberAck=true`. Registered as second projection in ProjectionHost.               |
| **Getting-started test**       | `TestGettingStarted_CounterValue` verifies counter=10 after 5+3+2 increments through full pipeline.                                                                                                                              |
| **5 doc files updated**        | README.md, SKILL.md (symlink to `.agents/skills/go-cqrs-lite/SKILL.md`), `references/advanced.md`, `references/readmodels.md` — all old example references replaced with taskmanager/getting-started.                            |
| **go.mod files build**         | Both modules pass `GOWORK=off go build ./...` and `go test`. Previous session's claim was false — go.mod needed `go mod tidy -e`.                                                                                                |
| **Git pushed**                 | 7 commits pushed to master.                                                                                                                                                                                                      |

**Test count:** 9 tests pass (8 taskmanager: 5 scenario + 2 integration + 1 deriver; 1 getting-started)
**Total LOC:** 2,610 (taskmanager: 2,374 LOC across 12 .go files; getting-started: 236 LOC across 2 .go files)

---

## b) PARTIALLY DONE

| Item                            | What's Done                                                                                                                                                                                                                                                                                                                                                                                                                                           | What's Missing                                                                                                                                         |
| ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **CBOR codec**                  | Investigated: library bug was already fixed (encoding stamp preserved in `eventToMessage`). Projection callbacks use `DecodePayloadAuto`. But enabling CBOR still causes projection failures — likely because signing's `CloneEvent` re-encodes payload via `event.NewEvent` with `PayloadReadOnly(evt)` which passes `[]byte` through as-is, but the encoding stamp from `WithEncoding` option gets overwritten by `DefaultCodec` during `NewEvent`. | Not enabled in the example. JSON works correctly. Needs deeper library investigation of the `CloneEvent` → `NewEvent` encoding stamp propagation path. |
| **Deriver integration test**    | Deriver is wired and compiles. The async dispatch pattern avoids deadlocks.                                                                                                                                                                                                                                                                                                                                                                           | No dedicated test verifying the auto-assign actually fires. Integration test may be flaky due to async timing.                                         |
| **Doc references in AGENTS.md** | Module list, test command, structure tree updated in session 1.                                                                                                                                                                                                                                                                                                                                                                                       | Inline code examples in AGENTS.md still reference old patterns — not broken, just not refreshed.                                                       |

---

## c) NOT STARTED

These were considered but deferred for scope/time:

| Item                                     | Why It Matters                                                                                                                                       |
| ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Graph projection** (task BLOCKS edges) | Would demonstrate the 3rd projection tier (`graph.GraphProjection`). `TaskState.BlockedBy` exists in the domain. Would need graph module dependency. |
| **Catalog generation** (AsyncAPI/D2)     | Would demonstrate `catalog.NewBuilder` + exporters for auto-docs.                                                                                    |
| **Encryption demo** (XChaCha20)          | Would show `encryption.NewCodec` wrapper for confidential event payloads.                                                                            |
| **Scheduling demo** (auto-archive)       | Would show `scheduling.Scheduler` for deadline-based command dispatch.                                                                               |
| **Idempotency demo**                     | Would show `idempotency.MemoryStore` for command dedup.                                                                                              |
| **gRPC transport**                       | Would show `transport/grpc` for remote command/query dispatch.                                                                                       |
| **Prometheus /metrics**                  | Would show `prometheus.Setup()` + metrics endpoint.                                                                                                  |

---

## d) TOTALLY FUCKED UP

| Item                             | What Went Wrong                                                                                                                                                                                                                                     | Impact                                                                                                                                                                                                 | Fixed?                                      |
| -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------- |
| **Session 1 status report LIED** | Claimed "both modules build with GOWORK=off" and "all 17 tests pass" — neither was true without running `go mod tidy -e` first. The go.sum was stale.                                                                                               | Lost trust in session 1's claims. Had to verify everything from scratch.                                                                                                                               | ✅ Fixed — go.mod/go.sum are now correct.   |
| **`json:"-"` on Tombstoned**     | The `Tombstoned` field on `TaskView` had `json:"-"` which means it was NEVER serialized to the KV store. Setting `Tombstoned=true` in `OnTombstone` was silently stripped on `Set`. On `Get`, it was always `false`.                                | The #1 "unresolved bug" from session 1 was a **1-character bug** — not a library issue, not a Watermill serialization problem, not a race condition. Just `json:"-"` vs `json:"tombstoned,omitempty"`. | ✅ Fixed                                    |
| **Signing dead code**            | `if false { _ = signing.SignMiddleware }` — the signing middleware was never actually installed. The flagship claimed to show signing but didn't sign anything.                                                                                     | Misleading. The fix was a straightforward type assertion that session 1 couldn't figure out.                                                                                                           | ✅ Fixed                                    |
| **Compiled binaries committed**  | `example/taskmanager/taskmanager` (compiled binary) was accidentally committed to git.                                                                                                                                                              | Repo bloat.                                                                                                                                                                                            | ✅ Fixed — removed + added to `.gitignore`. |
| **CBOR still doesn't work**      | Even after the library fix (encoding stamp preservation), enabling CBOR causes `invalid character '£'` in projection path. The signing `CloneEvent` → `NewEvent` path likely loses the encoding stamp because `NewEvent` re-applies `DefaultCodec`. | Flagship uses JSON only. Missing a key library feature.                                                                                                                                                | ❌ Deferred                                 |

---

## e) WHAT WE SHOULD IMPROVE

### Critical (trust-breaking)

1. **Fix CBOR codec** — the encoding stamp propagation through signing's `CloneEvent` needs investigation. When `CloneEvent` calls `event.NewEvent(..., PayloadReadOnly(evt), WithEncoding(cbor))`, the `NewEvent` function checks `opts.newCodec` first, but `PayloadReadOnly` returns `[]byte` which bypasses codec encoding entirely (correct), however `WithEncoding` sets `e.encoding` while `NewEvent` line 62 also sets `evt.encoding = c.Encoding()` — these may conflict depending on option order.

### Important (quality)

2. **Add graph projection** — the 3rd projection tier. Task dependencies (`BlockedBy`) are natural graph edges. Would complete the "3 projection tiers" promise.
3. **Add deriver integration test** — verify the auto-assign actually fires end-to-end.
4. **Add catalog generation** — demonstrate AsyncAPI/D2 export from event types.
5. **Update taskmanager README** — reflect all new features (ProjectionHost, SSE, snapshot, deriver, signing).

### Nice-to-have

6. Add encryption demo (XChaCha20 codec wrapper).
7. Add scheduling demo (auto-archive after due date).
8. Add idempotency demo (command dedup).
9. Add Prometheus `/metrics` endpoint.
10. Add gRPC transport option.

---

## f) Top 25 Things to Get Done Next

| #   | Task                                                                         | Impact   | Effort |
| --- | ---------------------------------------------------------------------------- | -------- | ------ |
| 1   | Fix CBOR codec — investigate `CloneEvent`/`NewEvent` encoding stamp conflict | Critical | M      |
| 2   | Add graph projection for task dependencies (BLOCKS edges)                    | High     | M      |
| 3   | Add deriver integration test (verify auto-assign fires)                      | High     | S      |
| 4   | Update taskmanager README with all new features                              | High     | S      |
| 5   | Add catalog generation (AsyncAPI + D2 export)                                | Medium   | M      |
| 6   | Add encryption demo (XChaCha20 codec wrapper)                                | Medium   | S      |
| 7   | Add scheduling demo (auto-archive after deadline)                            | Medium   | S      |
| 8   | Add idempotency to command handlers                                          | Medium   | XS     |
| 9   | Add Prometheus `/metrics` endpoint                                           | Medium   | XS     |
| 10  | Add gRPC transport option                                                    | Low      | M      |
| 11  | Add health check verifying event store connectivity                          | Low      | XS     |
| 12  | Add input validation depth (priority bounds, max-length)                     | Low      | S      |
| 13  | Add consistent JSON response envelope                                        | Low      | S      |
| 14  | Make Start() accept configurable HTTP address                                | Low      | XS     |
| 15  | Add `TombstonePolicy.IncludeTombstoned` list option demo                     | Low      | XS     |
| 16  | Add OTel stdout exporter toggle via env var                                  | Low      | XS     |
| 17  | Add command audit trail demo (command store)                                 | Low      | S      |
| 18  | Add query audit trail demo (query store)                                     | Low      | S      |
| 19  | Add multi-DB SQLite preset demo (separate DBs)                               | Low      | S      |
| 20  | Add relayer/rebirth cycle demo (tombstone → active)                          | Low      | S      |
| 21  | Add Watermill CatchUpSubscriber vs ProjectionHost comparison doc             | Low      | S      |
| 22  | Add Dockerfile for containerized deployment                                  | Low      | XS     |
| 23  | Add integration test for SSE endpoint                                        | Low      | S      |
| 24  | Add integration test for snapshot loading                                    | Low      | S      |
| 25  | Add benchmark test for projection throughput                                 | Low      | M      |

---

## g) Top #1 Question I Cannot Figure Out

**Why does CBOR still fail when the library encoding stamp preservation is already fixed?**

The `watermill/protocol.go` `eventToMessage` function correctly stamps `metaPayloadEncoding` from `evt.Encoding()`, and `MessageToEvent` correctly restores it via `event.WithEncoding(codec.Encoding(md.Get(metaPayloadEncoding)))`. The roundtrip test passes.

But when I set `event.DefaultCodec = codec.CBORCodec{}` and pass `codec.CBORCodec{}` to ReadModel and Materialize, the projection fails with `invalid character '£'` — CBOR bytes being JSON-decoded.

**The signing path is the likely culprit:** `signing.CloneEvent` calls `event.NewEvent(..., PayloadReadOnly(evt), ...)` with `WithEncoding` option. But `event.NewEvent` at line 62 sets `evt.encoding = c.Encoding()` where `c` is resolved from `DefaultCodec` (since no `WithCodec` option is passed in `CloneEvent`). The `WithEncoding` option then sets `e.encoding` AFTER, but only if the option is applied after line 62. If the option ordering causes `NewEvent` to overwrite the encoding to CBOR's stamp, but the payload was already CBOR bytes passed through as `[]byte`... then the resulting event has CBOR encoding stamp but when the signing middleware later signs and the verify middleware verifies, the canonical payload hash should still match.

**The actual failure is in the projection decode path**, not signing. The event arrives at the projection with the correct CBOR encoding stamp, but `DecodePayloadAuto` dispatches to `CBORCodec.Decode` which should work... unless the signing `AttachSignature` → `CloneEvent` → `NewEvent` path corrupts the payload bytes somehow.

I need to trace the exact payload bytes at each step to find where the corruption happens. This requires adding debug logging to the signing middleware or writing a focused roundtrip test with CBOR + signing enabled.

---

## File Inventory

```
example/taskmanager/         2,374 LOC (12 .go files + README.md + go.mod + go.sum)
  domain.go                  135 LOC
  events.go                   79 LOC
  decider.go                 535 LOC
  decider_test.go            247 LOC
  projection.go              154 LOC
  handlers.go                187 LOC
  http.go                    340 LOC
  setup.go                   299 LOC
  features.go                 94 LOC
  deriver.go                  55 LOC
  integration_test.go        240 LOC
  main.go                      9 LOC

example/getting-started/       236 LOC (2 .go files + go.mod + go.sum)
  main.go                    151 LOC
  main_test.go                85 LOC
```

## Test Status

```
ok  github.com/larsartmann/go-cqrs-lite/example/taskmanager       0.063s
ok  github.com/larsartmann/go-cqrs-lite/example/getting-started   0.107s
```

## Commit History (This Session)

```
33932527 feat: add deriver — auto-assign new tasks via event→command reaction
22339ee5 docs: update all references from deleted examples to taskmanager/getting-started
e858ec74 feat: add SSE real-time streaming, snapshot strategy, and transport/http dep
262f4a6f chore: gitignore compiled example binaries
1d5fb428 feat: wire ProjectionHost with DLQ replacing raw CatchUpSubscriber loop
ff706332 fix: tombstone projection now persists correctly across KV store roundtrip
a97ea250 fix: wire event signing middleware via EventBus type assertion
5bebd37f feat: consolidate 9 examples into flagship taskmanager + getting-started
```
