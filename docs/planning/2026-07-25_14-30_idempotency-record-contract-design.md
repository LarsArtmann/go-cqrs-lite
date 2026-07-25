# Design Note: Idempotency `Store.Record` Contract

**Date:** 2026-07-25
**Status:** Decided — Option A implemented (2026-07-25)
**Trigger:** Three `Record` implementations disagree on TTL semantics.

> **Resolution:** `idempotency/kvstore.Record` now uses `SetIfAbsent` (no-op on
> existing key, TTL not extended), matching `MemoryStore`, `sqlstore`, and the
> documented `Store.Record` contract. The open question below ("does kv.Store
> support SetNX?") was answered YES: `KVBackend` already requires
> `kv.ConditionalWriter.SetIfAbsent`, which `CheckAndRecord` already used.
> Regression tests: `TestStore_Record_DoesNotExtendTTL` and the cross-impl
> `TestStore_Record_MatchesMemoryStoreContract` guard the unified contract.

---

## The Split-Brain

| Implementation | Behavior on existing, non-expired key | Extends TTL? |
|---|---|---|
| `idempotency.MemoryStore.Record` | No-op (`set only if absent`) | **No** |
| `idempotency/sqlstore.Store.Record` | No-op (`INSERT ... ON CONFLICT DO NOTHING`) | **No** |
| `idempotency/kvstore.Store.Record` | **Overwrites** (calls `backend.Set` unconditionally) | **Yes** |

The interface doc (`idempotency/store.go:34-36`) says:
> *"If the key is already recorded, it is a no-op (the TTL is not extended)."*

Two implementations follow the doc. One violates it.

---

## The Two Contracts

### Option A: No-op on existing (don't extend TTL)

**Semantics:** Once a key is recorded, its dedup window is fixed at the original TTL. A retry arriving within the window is deduped; a retry arriving after the window expires is treated as a new request.

**Pros:**
- Bounded dedup window — every key is deduped for exactly `ttl`, then eligible again. Predictable.
- Protects against infinite-retry storms: a broken consumer retrying every 100ms won't keep extending the TTL forever.
- Matches the documented contract (2 of 3 implementations + the interface doc).
- Simpler mental model: "the first call within `ttl` wins; after `ttl`, the door reopens."

**Cons:**
- A long-running operation that takes longer than `ttl` to complete will be re-executed if the caller retries. The dedup window doesn't cover the full processing time.
- Requires the caller to choose a `ttl` that exceeds the worst-case processing time.

**At-least-once implication:** Keys are re-executed after `ttl` even if the first execution is still in-flight. This is **correct for at-least-once** — the delivery guarantee is "at least once," not "exactly once," and the idempotency layer's job is to suppress *duplicate deliveries within a window*, not to guarantee single execution forever.

### Option B: Overwrite on every call (refresh TTL)

**Semantics:** Every `Record` call (including retries) resets the TTL clock. A key stays deduped as long as retries keep arriving within the window.

**Pros:**
- Self-extending dedup window — a retry storm keeps the key alive, preventing re-execution as long as retries are flowing.
- Better for long-running operations where the caller polls/retries until success.

**Cons:**
- Unbounded dedup window under retry storms — a broken consumer retrying forever keeps the key alive forever, permanently blocking the operation.
- Violates the documented contract.
- Only 1 of 3 implementations does this (kvstore).

**At-least-once implication:** Retries within the window are deduped, but the window extends with each retry. If retries stop, the key expires normally. This is **also correct for at-least-once** but has different operational characteristics.

---

## Retry Semantics Matrix

| Scenario | Option A (no-op) | Option B (overwrite) |
|---|---|---|
| First delivery, processing completes in 2s, TTL=5m | Deduped for 5m, then reopens | Same |
| First delivery, processing takes 10m, TTL=5m, caller retries at 6m | **Re-executed** (TTL expired) | Deduped (TTL refreshed at 6m) |
| Broken consumer retries every 100ms forever | Key expires after 5m, operation re-executes once | Key **never expires**, operation **never re-executes** (but also never completes if blocked) |
| At-least-once with message redelivery after consumer crash | If crash+restart < 5m: deduped. If > 5m: re-executed (correct) | If retries resumed < 5m: deduped. Otherwise: re-executed (correct) |

---

## Recommendation: **Option A (no-op on existing)**

**Reasoning:**

1. **Documentation already says it.** The interface doc, MemoryStore, and sqlstore all implement Option A. Only kvstore diverges. Aligning to the majority is the lowest-risk change.

2. **Bounded dedup windows are safer for production.** An unbounded TTL (Option B under retry storms) can silently block operations forever. Option A's fixed window forces the system to self-heal: after `ttl`, stuck operations become eligible for re-execution, which is the correct at-least-once behavior.

3. **The caller controls the window.** If a caller needs a longer dedup window (long-running operation), they pass a longer `ttl`. The API already supports this. Option B removes that control by making the window depend on caller behavior.

4. **The kvstore fix is trivial.** Change `backend.Set(key, val)` to a conditional write (SetNX semantics). Most KV backends support this natively (Redis `SET NX`, etcd `CreateRevision == 0`, etc.).

5. **Option B's only advantage** (self-extending window for long-running ops) is better solved by a different mechanism: a `RefreshTTL(key, ttl)` method that the processing goroutine calls explicitly while work is in progress. This separates "dedup a duplicate delivery" from "extend the processing lease."

---

## Proposed Change (if approved)

1. **`idempotency/kvstore/store.go:74-86`** — change `Record` to use conditional write (SetNX). If the KV backend doesn't support conditional writes, fall back to Get-then-Set with a note that this is not atomic (and recommend sqlstore for cross-process dedup).

2. **`idempotency/store.go:34-36`** — strengthen the doc comment to explicitly call out that expired keys are not refreshed by Record (documenting the sqlstore behavior discovered in this session).

3. **Add `RefreshTTL(ctx, key, ttl) error`** to the Store interface as an OPTIONAL capability (not all backends need to support it). This gives callers that need Option B's behavior an explicit mechanism instead of overloading Record.

4. **Add a cross-implementation contract test** that verifies all three implementations behave identically for the no-op-on-existing contract.

---

## Open Question for Decision

The kvstore backend (`idempotency/kvstore/store.go`) wraps a `kv.Store` (the layer-0 KV abstraction). Does `kv.Store` support a conditional SetNX operation? If not, the fix requires either:
- Adding `SetNX` to the `kv.Store` interface (API change in the kv module), or
- Implementing Get-then-Set in kvstore.Record (non-atomic, documented limitation), or
- Documenting that kvstore.Record is best-effort and recommending sqlstore for correctness-critical dedup.

This needs investigation of the `kv.Store` interface before implementation.
