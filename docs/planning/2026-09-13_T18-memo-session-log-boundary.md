# T18 Decision Memo: Session Log and the `queue/` Boundary

**Date:** 2026-09-13
**Status:** Proposal for decision (no code changed)
**Parent plan:** [`2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md`](2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md) (T18)
**Evidence:** repo-wide zero hits for session event types · `cqrs-htmx/identity-model` (external) · [`claiming/`](../../claiming/) extraction note

---

## What the design asked for vs. reality

The 2026-07-23 design (§10, "Sessions as Event Streams") proposed `SessionStarted` / `SessionEnded` / `SessionRevoked` as first-class event streams with analytics/audit/security projections.

**Reality (verified 2026-09-13):**

- Zero session event types exist repo-wide. No `SessionStarted` anywhere in production code.
- Sessions live in `cqrs-htmx/identity-model` as ephemeral runtime objects — the design text itself acknowledges this and argues they *should* be event-streamed.
- Actor attribution in this repo is already typed: `record.CommonMetadata.Actor` / `id.ActorID` end to end, so any future session events would integrate without new attribution plumbing.

## The `queue/` question

A planned `queue/` module is documented in the skill reference: built on the extracted `claiming/` module ("Dialect-correct lease-claim SQL core — owns NO store", extracted from `scheduling/sqlstore` 2026-09-13).

**Intersection analysis:**

| Concern | `queue/` | Session log |
| ------- | -------- | ----------- |
| Subject | Work items / jobs (claim, lease, retry, complete) | Who is logged in (start/end/revoke) |
| Lifecycle events | Job lifecycle (claim/complete/fail) | Session lifecycle |
| Shared pieces | `claiming/` lease SQL; possibly commandlifecycle-style event recorder | Actor attribution; possible commandlifecycle-recorder template |

They share *patterns*, not *domain*: a queue is not a session store, and sessions are not work items. Coupling them would create a module with two unrelated reasons to change.

## Options

| # | Option | Effort | Outcome |
| - | ------ | ------ | ------- |
| A | **Sessions permanently external** — document the boundary; identity-model owns session state | ~15min doc | Clear responsibility; no session audit in this repo |
| B | **Future `sessionlifecycle` module** — event-sourced sessions modeled on `commandlifecycle` (3-5 event types, projections for concurrency/revocation), possibly reusing its recorder template | ~1-2d when triggered | The doc's analytics/audit benefits; more surface to maintain |
| C | Fold sessions into the planned `queue/` module | not recommended | Two domains in one module; unclear ownership |

## Recommendation: **A now, B only when a concrete consumer asks**

- There is no in-repo consumer for session events today; the identity-model project deliberately made sessions ephemeral. Event-sourcing them now is speculative surface.
- Keep the §10 section marked NOT SHIPPED (done in the reconciliation) as the standing design argument.
- If an audit/compliance consumer appears (e.g. "who was logged in when X happened"), implement option B as a sibling of `commandlifecycle` — the pattern is proven, and ActorID is already typed. Do not merge it into `queue/`.
- The `queue/` module should stay scoped to work-item claiming/leasing; the only shared code worth reusing is `claiming/` (already extracted for exactly that).

## Decision

- [ ] A — sessions stay external; boundary documented (recommended)
- [ ] B — add `sessionlifecycle` module later (trigger: concrete audit consumer)
- [ ] C — fold into `queue/` (not recommended)
