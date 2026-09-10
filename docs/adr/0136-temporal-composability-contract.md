# ADR-0136: The Temporal Composability Contract (Invertibility Ladder)

- Status: Accepted
- Date: 2026-09-10
- Deciders: Lars Artmann
- Related: ADR-0114 (tombstone as domain event), ADR-0123 (v5 wave),
  ADR-0124 (layout planning / rebuild gating), ADR-0004 (saga),
  [Cordis mapping](../architecture-understanding/2026-09-10_cordis-spatiotemporal-composability-mapping.md)

## Context

The Cordis spatiotemporal-composability mapping surfaced that this repo already
implements "revertible effects" — but implicitly, scattered across modules, with
no single statement a user can consult to answer: **"can I undo this, and how?"**
Two enforcement gaps were found and fixed in the same session (M-06/M-07):

- `projectionhost.Host.Reset` on a non-`Resettable` projection silently cleared
  only the checkpoint — a *partial revert* that left stale read-model state.
- The `metaengine` Store had **no reset primitive at all**; "rebuild" meant
  constructing a new Store.

Both shipped warn-first in v4.x (hard error rides the ADR-0123 v5 wave). This
ADR codifies the contract those mechanisms enforce, so future surfaces (new
engines, new adapters, `deriver` sagas) have one normative ladder to follow
instead of rediscovering it per module.

## Decision

Every effect a go-cqrs-lite consumer can produce sits on a three-rung
**invertibility ladder**. Rung is a property of the *effect*, not of the API
that triggered it.

### Rung 1 — Replayable (mechanically invertible)

The effect is **derived data**: state computed purely by folding events
(projections, read models, metaengine stores, cached views). The inverse
exists by construction — drop the derived state, replay the journal. Nobody
compensates anything; the runtime holds the inverse.

Contract:

- Declaring it as a projection/read model **grants Reset for free** — the
  consumer never writes un-revertible derived state by hand.
- Reset must be **total or loud**: either the whole footprint clears, or the
  caller is told (structured `ResetResult.Partial()` + log warn in v4.x;
  hard error in v5 — M-06 `Host.Reset` warn-guard, M-07 `EngineResetter`).
- Engine capability ladder: `memory` implements `EngineResetter` (full);
  persistent engines report themselves via `UnclearableEngines` until they
  implement it (`sqliteengine` is the prioritized follow-up: 8 `meta_*`
  tables + planned tables + matviews + `multiSeq` state).
- Rebuild cost is priced before it runs (`LayoutDiff.EstimatedRebuildEvents`,
  `RebuildThreshold` / `ConfirmRebuild`, ADR-0124).

### Rung 2 — Compensable (business-level inverse)

The effect crosses the process boundary into a system that does not hold our
journal (payment captured, email sent, warehouse reserved). No mechanical
inverse exists; the inverse is a **compensating business action**, and in an
event-sourced system a business action is an event. This is the `deriver`
saga pattern (ADR-0004): `payment.refunded`, `reservation.released`.

Contract:

- Compensations are **events, not `Close()`/disposers**. `Closer` releases
  resources (files, connections, goroutines); it must never attempt semantic
  undo — a `defer` cannot append to the journal.
- A saga's forward and compensating steps both live in the journal, so the
  compensation itself is auditable and idempotent.

### Rung 3 — Must-be-an-event (facts, not revertible at all)

Domain decisions (`user.registered`, `order.placed`) are **facts**. Facts are
never undone — history is append-only. The only honest "inverse" is a
**negating fact appended forward**: `user.deleted` (tombstone) with rebirth
as its inverse (ADR-0114). Mutating or deleting history is corruption, not
reversion.

### The user decision rule

Before writing an effect, ask **"what is its inverse?"** and route by the
answer:

| The effect is… | Rung | Do this |
| --- | --- | --- |
| Recomputable from events | 1 | Declare it a projection/read model; Reset + replay is the inverse. |
| External but business-reversible | 2 | Model forward + compensation as events (`deriver` saga). |
| A domain decision | 3 | Append it, and model retraction as a negating event (tombstone). |

Anti-patterns the rule forbids: deleting a projection's rows by hand while
leaving the checkpoint (silent partial revert — now warned); performing
semantic undo in `Close()`; "updating" an emitted event in place.

## Consequences

- **v4.x:** every revert path either completes or warns (`Host.Reset`
  non-Resettable warn, `Adapter.Reset` partial warn). No v4 caller breaks.
- **v5:** warns become errors (ADR-0123 train); the ladder becomes
  mechanically enforced, not advisory.
- New engines should implement `EngineResetter` or document why they cannot;
  `Doctor`/`GetEngineStats` will surface reset capability (follow-up).
- Skill references carry the revert-and-rebuild recipe as the canonical
  Rung-1 user flow (M-10).
- Vocabulary stays internal (mapping doc) until the M-26 decision gate; this
  ADR uses plain terms (replayable/compensable/fact) that stand alone.
