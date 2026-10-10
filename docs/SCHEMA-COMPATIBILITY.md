# Schema Compatibility Policy

How event payload schemas may evolve without breaking stored events, read
models, or the fleet. This is the policy the `schema` module's machinery
(upcast chains, fingerprints, snapshot stamps) enforces or observes; the
`cqrs-lint` rules E021/E022 statically check the authoring side.

## The one rule

Stored events are immutable facts. Evolution happens on the READ path via
declared upcast ops; nothing is rewritten in place. Every change to a payload
shape is therefore a question about OLD events: can they still be read?

## Safe without a version bump (additive)

| Change                                | Why safe                                                                    |
| ------------------------------------- | --------------------------------------------------------------------------- |
| Adding a field with a zero value      | Old events decode with the zero value; `AddField` can default it explicitly |
| Adding `omitempty` to a new field     | Wire shape of old writers unchanged                                         |
| Adding a NEW event type               | Nothing old references it                                                   |
| Widening an int field (int32 → int64) | All old values representable (verify per codec)                             |
| Adding an index to a layout plan      | Read-path only; record the new fingerprint                                  |

Additive changes never bump `EventSchema`'s current version and never add
ops. Bump the DECLARED-shape fingerprint consumers only if you want the
ledger to note the change (fingerprints change automatically).

## Version-bump triggers (write an op, bump current version)

| Change                       | Required op                                               |
| ---------------------------- | --------------------------------------------------------- |
| Renaming a field             | `RenameField(type, oldVersion, from, to)`                 |
| Removing a field             | `RemoveField(type, oldVersion, field)`                    |
| Changing a field's type      | `Transform(type, oldVersion, ...)` converting values      |
| Restructuring (flatten/nest) | `Transform(type, oldVersion, ...)` reshaping the payload  |
| Renaming the event type      | `RenameType(from, target)` in the FROM type's declaration |

Bump the declaration's current version BY ONE per migration step and declare
an op FROM the previous version. The ladder must be CONTINUOUS: every version
below the current one needs an op (cqrs-lint **E022** flags gaps — a stored
event at a gap version would never upcast).

## Breaking (never do these)

- Removing or re-versioning an op that already stamped events — old events
  silently stop upcasting.
- Changing an op's semantics without changing the payload shape — upcasting
  is idempotent per (type, version); a changed body re-transforms already-read
  events inconsistently. Fingerprints deliberately do NOT capture closure
  bodies; treat op internals as frozen once deployed.
- Editing the event journal in place to "migrate" — the journal is the
  survivor of every reset (ADR-0143); mutations destroy the replay source.
- Changing a snapshot State's shape without bumping
  `decider.WithSnapshotStateVersion` — stale snapshots must be discarded and
  rebuilt, never decoded into the new shape.

## Observability ladder

1. Stamp on write (`DomainConfig.StampSchemaFingerprints`) and watch
   `schema.FingerprintDrift` — advisory ledger while shapes settle.
2. After a quiet deployment cycle, flip drifted collections to
   `schema.FingerprintDriftHard` (Corruption on mismatch).
3. Layout plans: `RecordLayoutStamps` at apply time, `LayoutStampDiffs` at
   boot, rebuilds through the ADR-0124 gating (`RebuildThreshold` auto /
   `ConfirmRebuild` operator gate), `MarkReplayComplete` when a rebuild's
   replay finishes.
