# Owner Decision Pack — BDD Depth Follow-Ups (2026-10-17)

> One artifact, three rulings requested (supersedes the three separate memos
> of plan v1 M14+M15+M16). Source:
> [plan v2](../planning/2026-10-17_20-18_SUPERB-bdd-depth-followups-pareto-plan.md)
> §2. Context: [self-review](../status/2026-10-17_20-13_bdd-harness-depth-specs-execution-and-self-review.md).

## D1 — Tag timing for the depth-spec wave → RESOLVED BY EVENTS (no ruling needed)

**Was:** tag the corruption reclassification + nextVersion fix now, or batch?

**Now resolved threefold:**

1. Owner directive 20:36 ("GET SHIT DONE — the WHOLE TODO LIST") = tag-now.
2. A parallel session's batch release (20:47-20:48) already tagged
   `metaengine/projectionadapter/v4.6.0` **containing the corruption
   classification fix**, plus event/v4.15.0, command/v4.14.0, signing/v4.4.1,
   encryption/v4.4.5, kv/v4.3.5.
3. Companion verification (plan M01/M02): cqrs-htmx and go-appkit full
   suites are GREEN against the workspace tree containing the fixes — the
   blast radius is measured, not assumed.

**Remainder (executed as plan M12):** tag `system` (13 untagged commits,
needs the projectionadapter v4.5.3→v4.6.0 pin bump to ship the fix through
`system.New`) and `systemscenario` (28 untagged commits: depth specs +
nextVersion fix), then `pin-sweep` so standalone (GOWORK=off) legs go green
— M04 proved both modules' standalone suites are red purely on stale pins
(the DLQ spec fails against pre-fix adapter exactly as designed).

## D2 — Erroring consumer folds → RULING REQUESTED

**Question:** should `system.Lookup`/projection declarations expose
fold-level errors in v4.x (additive), defer to v5, or never?

**Why it matters:** today consumer folds (`Lookup` funcs) cannot fail, so
the DeadLetterStore is reachable ONLY for decode poisons (the
`TestDepth_PoisonEventDeadLetters` spec). DOMAIN poisons — a fold that can
look at a healthy payload and still reject it (invariant violations,
cross-aggregate checks) — cannot dead-letter; they cannot even signal.
The DLQ story is half-built until folds can error.

**Options:**

| Option | Consequence |
| --- | --- |
| v4.x additive (e.g. `LookupE`/erroring fold variant) | DLQ reachable for domain poisons; unlocks the retry-journey depth spec (plan M09 half); grows v4 declaration API surface during the v5 dual-support window |
| v5 | Keeps v4 frozen; DLQ stays decode-only until the v5 cut; the depth spec waits |
| never | DLQ is a decode-poison quarantine, documented as such; spec story closed as-is |

**My recommendation (for the record): v5.** The v4.x surface is in
maintenance mode (ADR-0152 dual-support), erroring folds change the
declaration API shape (the vision statement's "one Evolution per read
model" is a v5-native fit), and nothing in the fleet currently asks for it.
The retry-journey spec is deferred with a note either way (its DoD accepts
D2-deferred).

## C1 — Error classification at the projection boundaries → RULING REQUESTED

The depth-spec session fixed ONE site (projectionadapter decode →
Corruption, shipped in v4.6.0). Two sibling sites remain unclassified
(unclassified defaults to Transient = retryable = worker crash-loop when no
DLQ is configured — the exact failure mode the decode fix removed):

### C1a — `projectionadapter/adapter.go:159` (apply wrap)

```go
if err := a.store.ApplyRecord(ctx, event.AsRecord(evt), decoded); err != nil {
    return fmt.Errorf("projectionadapter: apply %s: %w", eventType, err) // → Transient
}
```

The cause is an ENGINE failure (sqlite/pg write, constraint, lock). Unlike
decode failures these are NOT deterministic-by-inspection: a locked DB is
transient; a constraint violation on deterministic data is not.

**Proposal: pass the cause's family through** —
`errorfamily.Wrapf(err, errorfamily.Classify(err), …)` — so classified
engine errors keep their family and unclassified ones stay Transient
(the correct default for engine I/O). No behavior change for correctly-
classified engines; crash-loop only remains for engines that never classify.

### C1b — `system/projection_builder.go:188-200` (buildEventDecoder)

Three fmt.Errorf wraps on the `system.New` decode path — the system-level
twin of the adapter fix that never got the same treatment:

- `:188` "no decoder registered for event type %q" — a wiring gap,
  deterministic. Proposal: **Rejection** (deterministic, not data damage;
  DLQ-able, not retry-fixable).
- `:196` "codec for encoding %q" — deterministic config/codec error.
  Proposal: **Corruption**.
- `:200` "decode %s" — THE decode failure itself. Proposal: **Corruption**,
  identical rationale to the shipped adapter fix (symmetry between the two
  decode paths; today the system path crash-loops where the adapter path
  dead-letters).

**Why these are not drive-bys:** each changes worker/DLQ behavior for every
`system.WithCommandLifecycle`-style consumer; the 20:13 review ruled the
apply-boundary split "genuinely judgment, needs an owner call". A green
light = three small wrapped-error changes + depth-spec pins for each.

## Ruling log

| Gate | Ruling | By | When |
| --- | --- | --- | --- |
| D1 | tag-now (directive + parallel wave already shipping) | owner directive + events | 2026-10-17 20:36 |
| D2 | *pending* | | |
| C1 | *pending* | | |
