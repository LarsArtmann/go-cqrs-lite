# T17 Decision Memo: Command-Log Audit Scope

**Date:** 2026-09-13
**Status:** Proposal for decision (no code changed)
**Parent plan:** [`2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md`](2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md) (T17)
**Evidence:** [`commandlifecycle/`](../../commandlifecycle/) source · [15:55 deep dive](../status/2026-09-13_15-55_event-query-model-not-shipped-vs-reality.md)

---

## What shipped vs. what the design asked for

The 2026-07-23 design (§10) envisioned "a FULL COMPREHENSIVE audit log — who did what, when, and what did it cause", built from `CommandSucceeded`/`CommandRejected` records carrying `{Type, Payload, Metadata, Timestamp}` plus a `CommandsByUser` projection.

### Shipped today (better than designed in shape, narrower in scope)

| Piece                                                                       | Where                                                                            |
| --------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| 5 event types: `command.received/failed/retried/dead-lettered/completed`    | `commandlifecycle/events.go:51-64`                                               |
| Two streams: `Command/<id>` (journal) + `CommandLifecycle/<id>` (lifecycle) | `commandlifecycle/events.go:42-48`, `command/store.go:141-160`                   |
| Projections: dead-letter queue, retry count, failure log, processing time   | `commandlifecycle/projections/projections.go:42-124`                             |
| Middleware + recorder + `system.WithCommandLifecycle` wiring                | `commandlifecycle/middleware.go`, `recorder.go:99-158`, `system/lifecycle.go:50` |
| Causation link to the command (`WithCausation(cmd.Type(), cmd.ID())`)       | `commandlifecycle/recorder.go:180`                                               |

### Gaps vs. the doc's vision

| Gap                                          | Reality                                                                                                                                                                                                      | Severity                                                                                      |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------- |
| **Per-actor projection** (`CommandsByUser`)  | Does not exist. Actor attribution exists in event metadata (`record.CommonMetadata.Actor`), but no projection groups commands by actor.                                                                      | Medium — the doc's headline use case ("who did what") is half-answered (what, when; not who). |
| **Payload capture**                          | Lifecycle payloads capture type/ID/error/attempt/timestamps, never the command payload (`ReceivedPayload`, `events.go:72-139`). The doc's `CommandRecord.Payload` does not exist.                            | Medium by design — capturing every payload has PII/storage cost.                              |
| **Distinct rejection event**                 | No `command.rejected`. A business rejection surfaces as `command.failed` with the error text (`FailedPayload.Error`, `events.go:87-100`); errorfamily classification exists but is not stamped on the event. | Medium — "rejected by business rule" vs "broke" are different audit answers.                  |
| **`FailedPayload` has no `CommandID` field** | `Received`/`Completed` carry `CommandID`; failed/retried/dead-lettered rely on the stream ref. Not wrong (streams are keyed by command), but projections must key off the stream, not the payload.           | Low — fix opportunistically if payloads are touched.                                          |

## Options

| # | Option                                                                                                                                                                  | Scope                           | Effort |
| - | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------- | ------ |
| A | **Complete the audit scope** — add `CommandsByActor` projection; add `command.rejected` + recorder/middleware classification via errorfamily; leave payload capture out | projection + event + middleware | ~3-4h  |
| B | **Minimal extension** — add only the `CommandsByActor` projection; rejections stay inside `command.failed`                                                              | projection                      | ~2h    |
| C | **Declare scope done** — DLQ/retry/failure-log is the finished surface; per-actor and rejection are out of scope                                                        | docs only                       | ~15min |

## Recommendation: **B now, A if audit demand is real**

- The per-actor projection is the highest-value, lowest-risk gap: all data already exists (metadata Actor on the lifecycle events, command ID on received/completed), it is purely additive, and it directly answers "who did what".
- A distinct rejection event (A) is the right long-term shape, but it changes middleware semantics and needs a classification contract (which error families count as rejection) — worth doing deliberately, not as a side effect of reconciliation. Ship it when an actual audit/compliance consumer needs the distinction.
- Payload capture stays out by default; if needed, make it an explicit recorder option with size limits rather than a default (PII, storage amplification).
- `FailedPayload.CommandID` can be added in any of the options without breaking wire format (additive JSON field).

### Sketch: `CommandsByActor` (option B)

1. Query keyed by `id.ActorID` from `record.MetaData.Actor` on `command.received` (+ update on completed/failed for outcome fields).
2. Result: `ActorCommands{ActorID, Counts{Received, Completed, Failed, DeadLettered}, LastAt}` or a cursor-paginated event list per actor (decide at implementation).
3. Wire into `commandlifecycle/projections` with a test that feeds the middleware pipeline end-to-end.
4. Do not touch existing projections' output shapes.

## Decision

- [x] A — full audit scope (rejection event + per-actor; no payload capture) — rejection half EXECUTED 2026-09-15 (see Update below); payload capture stays out by design
- [x] B — per-actor projection only (recommended) — EXECUTED 2026-09-13 (`projections.CommandsByActor`)
- [ ] C — scope done as shipped

---

## Update 2026-09-13 (executed)

Option B was implemented the same day: `projections.CommandsByActor` folds `command.received`
into a Multimap keyed by the record's typed Actor (`"kind:raw"`), with `CommandsByActorQuery` /
`CommandRecordEntry` / `CommandsByActorResult`, included in `projections.All()` and covered by a
new test. Still deferred (recommendation unchanged): a distinct `command.rejected` event and any
payload capture.

## Update 2026-09-15 (executed: option A rejection half)

The distinct rejection event shipped (the remaining deferred piece is payload capture, still
opt-out by design — PII/storage, per the memo):

- **Classification contract**: `commandlifecycle.DefaultRejectionFamilies` = {Rejection,
  Conflict} ("no state changed; caller must act"), overridable via
  `WithRejectionFamilies` (empty list disables). Unclassified errors classify as Transient
  (errorfamily fail-open) and therefore stay failures. The default pairs with retry
  middleware (`IsRetryable` = Transient-only), so rejected commands are never re-attempted.
- **Event + recorder**: `TypeRejected` ("command.rejected") with `RejectedPayload` stamping
  the classified family + error code at rejection time; `Recorder.RecordRejected` +
  `Recorder.IsRejection`.
- **Middleware semantics (the deliberate change the memo warned about)**: rejection-classified
  errors emit `command.rejected` INSTEAD of `command.failed` per attempt, and the outer
  middleware suppresses `command.dead-lettered` for rejections — the DLQ becomes ops-only
  (rejections are never retried, so dead-lettering them is noise). Non-rejection flows are
  byte-identical to before. `FailureLog` is therefore failures-only by construction;
  new `projections.RejectionLog` (in `projections.All()`) folds rejected events.
- **Opportunistic gap closed**: `FailedPayload.CommandID` added (additive JSON field).
- **Evidence**: `commandlifecycle/classification.go`, `events.go`, `middleware.go`,
  `projections/projections.go`; table+integration tests in `classification_test.go`,
  `middleware_classification_test.go`, `projections_test.go`; system wiring count test
  updated to track `projections.All()`.
