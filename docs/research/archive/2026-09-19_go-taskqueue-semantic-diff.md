> **RESOLVED-BY-ROUTING — docs-health 9th pass (2026-09-20):** probe complete — T23 of the substrate plan
> delivered; the verdict below fed the P5 review, and the substrate shipped in the 92-tag wave
> (2026-09-19). No open tasks.

# go-taskqueue semantic-diff probe — production contract vs the ADR-0142 substrate

**Date:** 2026-09-19
**Task:** T23 of the metaengine-universal-storage-substrate plan ("go-taskqueue
semantic-diff probe: ADT vs production contract memo (P5 input)")
**Compared:**

- Donor: `~/projects/go-taskqueue` `internal/queue/queue.go` (Store interface,
  521 lines, read in full 2026-09-19) — the production-proven contract the
  queue/ family was upstreamed FROM ("upstream the PROVEN semantics; do not
  invent new ones").
- Library: `queue/store.go` (`Store[T]`, 33 methods),
  `metaengine/dueclaim.go` (`DueClaimer`/`FactSink`),
  `metaengine/dedup.go` (`DedupStore`).

**Verdict: the core semantics carried over 1:1; every divergence is either a
deliberate strengthening, a genericity change, or a product-specific surface
that correctly stayed in the consumer. No silent semantic drift found.**

## 1. Carried over unchanged (the load-bearing core)

| Donor semantic                                                                                                                          | Library home                                                                                                                                                 |
| --------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Enqueue assigns ID/defaults, records `task.enqueued`                                                                                    | `Store[T].Enqueue` (+ `task.New[T].Normalize`)                                                                                                               |
| ClaimDue: pending + NotBefore passed + deps met, sets Running + lease, `ErrNoTaskDue` when empty                                        | `Store[T].ClaimDue` (deps gating is OUR addition, §3)                                                                                                        |
| Fail: attempts remain → Pending + NotBefore=backoff; else dead-letter; evidence on the fact                                             | `Store[T].Fail`                                                                                                                                              |
| FailPermanent: dead-letter now, attempt still counted, class "permanent"                                                                | `Store[T].FailPermanent`                                                                                                                                     |
| Requeue: back to Pending WITHOUT counting an attempt                                                                                    | `Store[T].Requeue`                                                                                                                                           |
| Cancel / CancelRunning / CancelRequested / CancelOwned cooperative-cancel flow, idempotent request fact                                 | identical set in `Store[T]`                                                                                                                                  |
| MarkOrphaned: fact-only, no state change, idempotent                                                                                    | `Store[T].MarkOrphaned`                                                                                                                                      |
| RescueDead / DismissDead DLQ off-boarding with reason+by on the cancelled fact                                                          | identical pair                                                                                                                                               |
| UpdatePendingPriority: pending-only, same-value no-op, `task.reprioritized` fact in the same tx                                         | `Store[T].UpdatePendingPriority`                                                                                                                             |
| Facts journal: Facts(after, limit), FactsForTask (tail-bounded, ascending), HeadSeq watermark, Watermark/SaveWatermark monotonic upsert | identical set (+ `Watermarks` list, §3)                                                                                                                      |
| Priority aging: effective priority = stored + bounded age bonus computed in the claim query                                             | `queue/priority.go` (defined once, both engines)                                                                                                             |
| Facts-in-same-tx invariant ("every mutating operation appends its fact in the same transaction")                                        | generalized into the `metaengine.FactSink` ADT (ClaimDueFacts/ClaimDeleteFacts) + `queue.FactTx` — pinned by `adttest.AssertFactSink` on every native engine |

## 2. Deliberate strengthenings (library is STRICTER than the donor)

1. **Claim fencing is token-based, not owner-based.** Donor finalizers take
   `owner string`; the library takes `token string` (ADR-0134:
   `queue.Claim.Token`, `lease_token` column). The donor's owner check cannot
   distinguish two workers claiming the same owner identity; the token makes
   theft and same-owner double-finalize structurally impossible
   (`queue.ErrLeaseNotHeld`). `ClaimDue` correspondingly returns a
   `Claim[T]` (task + token) rather than a bare task.
2. **Enqueue dep validation.** Donor has no dependency graph. The library's
   `task.New[T].Deps` + existence check (`queue.ErrDanglingDep`, cycles
   unrepresentable by construction — store-minted IDs) rejects dangling edges
   AT ENQUEUE, and claim-time gating makes blocked tasks unselectable
   (queue plan T14; pinned by `queue/conformance` pinDepGating /
   pinDeadDepGates).
3. **Typed payloads.** Donor payloads/evidence are `jsontext.Value`
   (JSON-locked). The library is `Store[T]` with a pluggable codec
   (`WithCodec[T]`); evidence is `[]byte`. Same wire shape, no JSON
   dependency at the contract level.

## 3. Divergences to be aware of (all justified, none silent)

| Donor surface                                                                                   | Library status                                                                                                                              | Why                                                                                                                                                               |
| ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `RecordAnswer` + question types + `QuestionAskedDetail`/`QuestionAnsweredDetail`/`AnswerRecord` | NOT carried                                                                                                                                 | PapDashboard product domain (human-in-the-loop question flow). Belongs to the consumer; it composes over `Facts` + payload mutation like any consumer projection. |
| `SavePriorityScore`/`PriorityScore` (ADR-0015 AI score cache)                                   | NOT carried                                                                                                                                 | A product scoring cache, not queue state. Rides any KV/Map engine (metaengine) when a consumer needs it.                                                          |
| `resumeCloseout` flag on Requeue                                                                | folded into fact evidence semantics                                                                                                         | Rate-limited-close-out is a product flow; the journal distinguishes via fact detail, not a contract boolean.                                                      |
| `LastFacts(limit)`                                                                              | absent — `FactsForTask`/`Facts` limits are tail-bounded and ascending (pinTailBound), covering feed renders                                 | Composable from existing reads; add only on consumer demand.                                                                                                      |
| `CountFacts(ftype, since)`, `ProjectCounts`                                                     | absent — `CountTasks(Filter)` covers per-project counts caller-side via the Filter's project term; fact-type counting composes over `Facts` | COUNT pushdowns are conveniences; the library keeps the contract minimal (growing `Store[T]` waits for v5, contract 21g discipline).                              |
| —                                                                                               | `Watermarks()` ADDED                                                                                                                        | List-all-consumers cursor read (operator/DLQ tooling) the donor lacked.                                                                                           |
| —                                                                                               | `Store[T]` registers as metaengine drivers (`queue-sqlite/postgres/mysql`) + claimkit capabilities                                          | ADR-0142 absorption: the SAME store is an engine citizen (DueClaimer/DedupStore/FactSink via claimkit), so timers/dedup/claims share one substrate.               |

## 4. ADT extraction fidelity (the P5 question: did the ADT lose anything?)

The donor's claim core maps onto `metaengine.DueClaimer` as:

- `ClaimDue(owner, lease)` → `ClaimDue(ClaimDueRequest{Owner, Lease, Now, Limit})`
  over collections (batch-capable, clock-injectable — both needed by
  `scheduling/engine`).
- `Heartbeat(owner, extend)` → `RenewLease(collection, key, owner, extend, now)`.
- finalize-after-claim → `ClaimDelete` / `ClaimDeleteIfDue` (the epoch guard
  the donor did not need because its keys never re-schedule; timers do).
- enqueue-dedup (`DedupKey`) → generalized as `DedupStore.CheckAndRecord`
  TTL windows (the idempotency primitive extracted on its own).

What the ADT deliberately does NOT model: status machines, attempt budgets,
backoff ladders, priorities/aging, DLQ semantics — those remain
engine-owned in `queue/` per the facade-not-rewrite guardrail, and the
`queue/conformance` suite (which the donor's mirrored suites inspired)
pins them identical across all three engines. The scheduler facade
(`scheduling/engine`) proves the extraction is sufficient for the OTHER
consumer (timers) without growing the ADT.

**P5 input conclusion:** upstreaming did not fork the semantics — it
subtyped them. The donor can adopt the library (or the substrate directly)
without behavior change for everything in §1; §2 differences are upgrades;
§3 differences are product surface that was never the library's to take.
