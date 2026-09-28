# ADR-0149: Durable Checkpoints and DLQ-by-Events in system.New

- Status: Accepted
- Date: 2026-09-28
- Deciders: owner release-prep commits 2026-09-19..27 (design shipped in-tree); codified here after verification
- Supersedes: the "system.New checkpoint/DLQ in-memory-only" TODO premise (TODO_LIST row "system.New checkpoint/DLQ store options") and the cqrs-htmx ADR-0051 accepted-limitation that pinned `NewProjectionLayer` consumers
- Verification session: docs/planning/2026-09-28_01-26_SUPERB-publish-integrity-pareto-plan.md (M11)

## Context

The recorded claim (session status report 2026-09-27, TODO_LIST): "`system.New`
uses an internal in-memory checkpoint store, so consumers needing durable
checkpoints or dead letters cannot use it" — with design tasks M17 (engine-backed
checkpoint store) and M18 (durable DLQ store extraction) planned behind an
owner sign-off gate.

Verification against the current tree (2026-09-28) found the premise STALE on
both halves: the durable designs are already implemented, tested, and sitting
in the untagged `system/v4.10.0` content.

## Decision (as shipped)

### 1. Durable checkpoints: a Map collection on the deployment engine

`system.NewEngineCheckpointStore(backend metaengine.MapBackend)` persists each
projection's checkpoint as an entry of the `system_checkpoints` Map collection
(`system/checkpoint_engine.go`). The constructor resolves the backing engine
from the deployment — the engine named `"checkpoints"` wins, else the first
configured engine — and only engines carrying the Map ADT qualify; without a
qualifying engine the System falls back to the in-memory store and the config
comment says so loudly. Consumers override via `DomainConfig.CheckpointStore`.

This is ADR-0142's "the last trivial satellite rides engines" pattern: zero
dedicated infrastructure, durability equals the engine's durability.

### 2. Durable dead letters: there is no DLQ store, by design

ADR-0117 (command lifecycle as event streams) already decided the shape: the
`commandlifecycle.Recorder` persists lifecycle events — including failures that
dead-letter — into the caller's `event.Store` via `system.WithCommandLifecycle`;
the DLQ, FailureLog, RejectionLog, and retry-count projections derive from
those events (`commandlifecycle/projections`). Dead letters are durable to the
extent the event store is durable; a separate durable DLQ store would be a
second source of truth for the same facts.

## Consequences

- The planned M17 (checkpoint impl) is already satisfied, including the
  restart-persistence leg (`systemtest/checkpoint_restart_test.go`
  `TestEngineCheckpointStoreRestartDurability`, reopen-the-database over a real
  SQL engine) and the resolution pin (`TestCheckpointEngineResolution`).
- The planned M18 (DLQ store extraction) is OBSOLETE: there is nothing to
  extract; the recorder is the write path and the projections are the read path.
- The remaining consumer-facing gap is NOT design but PUBLISHING: the
  `system/v4.10.0` tag carrying this content was never cut (the 2026-09-27
  7-tag release wave stalled at 1/7 — see the tag-wave TODO row). Consumers on
  published `system/v4.9.0` still see the in-memory behavior.
- cqrs-htmx `NewProjectionLayer` consumers can unpin once `system/v4.10.0`
  (or later) ships: `NewEngineCheckpointStore` is the public seam.
