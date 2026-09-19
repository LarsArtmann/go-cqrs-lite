# ADR-0143: Engine Reset Never Deletes the Journal

**Date:** 2026-09-19
**Status:** Accepted
**Related:** ADR-0136 (temporal-composability / invertibility ladder), ADR-0142 (universal storage substrate)

## Context

ADR-0142 makes engines the ONE substrate for durable writes — including the
domain event journal itself: a `system` deployment can host its source of
truth on an engine (`RoleSourceOfTruth` → `NewEventAdapter` over
`StreamLogBackend`, journal rows in `meta_stream_log` / the `sl`/`jl`/`l`
keycodec prefixes).

ADR-0136's reset ladder made every first-party engine implement
`EngineResetter`, and the original implementation cleared EVERYTHING —
including the journal tables — reasoning that "the journal is rebuilt by the
replay". That is true only when the journal is a local cache of an external
event store (the classic metaengine-as-projection deployment). When the
journal IS the event store, the reset deleted the facts it was supposed to
replay from.

This surfaced as `TestSystem_ResetProjection_RestartAndReplay` failing
deterministically in workspace mode (`journal holds 0 event(s) from sys2's
view` — phase 2's fresh system replayed from a journal the phase-1 reset had
emptied) while passing under `GOWORK=off`, where the published engine modules
pre-dated `ResetEngine`. The failure had been mis-attributed to
load-sensitivity for days because the per-module (`GOWORK=off`) test mode
runs against PUBLISHED dependency versions, masking the local-tree regression.

## Decision

**Engine reset clears materialized state; the journal survives.**

- Journal entries (log ADT, stream-log ADT, the fact journal) sit on the
  FACTS rung of the ADR-0136 invertibility ladder — the replay SOURCE a reset
  rebuilds FROM, never derived data a reset clears.
- Concretely, per engine: `meta_log`/`meta_stream_log` left out of the SQL
  reset-table lists; `l`/`sl`/`jl` prefixes left out of the KV reset ranges;
  `LogEntry`/`StreamLogEntry` left out of the dgraph reset type sweep; the
  memory engine's `logs`/`streams`/`streamJournal` maps carried across
  `ResetEngine`.
- Journal position counters (AUTOINCREMENT/BIGSERIAL/sequences) keep their
  monotonicity guarantees — trivially, since the rows persist.
- Every engine's reset test now PINS journal survival (read-back non-empty
  after reset) alongside the existing derived-collections-cleared pins.

## Consequences

- A reset is now safe on shared-engine deployments: events hosted on the
  same engine as a projection survive `ResetProjection` / `Store.Reset`.
- `CatchUpEngine` (reset + fold replay) is unaffected: it folds the
  in-process event log into MATERIALIZED collections; journal rows were never
  its input.
- Consumers who (incorrectly) used an engine journal as scratch space that a
  reset would clear must switch to a derived collection (map/set/list) — the
  ladder's answer for rebuildable state.
- The verification lesson is recorded in the testing gotchas: a
  workspace-only regression is invisible to `GOWORK=off` per-module tests —
  run the module in BOTH modes after touching cross-module contracts.
