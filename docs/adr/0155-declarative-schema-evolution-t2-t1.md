# ADR-0155: Declarative Schema Evolution — Named Upcast Ops (T2) + DomainConfig Declaration (T1)

- Status: Accepted (delegated approval via standing execution loop, 2026-10-09 — same pattern as
  the proposal §12 ruling; owner re-issued the loop instead of objecting)
- Date: 2026-10-09
- Deciders: owner (delegated), executing session
- Related: ADR-0123 (v5 unification — `system.DomainConfig` IS the declaration surface),
  ADR-0126 (store/journal transforms — the machinery T2 compiles onto),
  ADR-0136 (temporal composability — replayable/compensable rungs),
  ADR-0143 (journal survives resets — why rebuild-from-journal is always safe),
  ADR-0151 (Evolutions are the declaration — no runtime Infer revival),
  proposal ([`docs/planning/2026-10-09_v5-declarative-schema-evolution.md`](../planning/2026-10-09_v5-declarative-schema-evolution.md)),
  execution plan ([`…execution-plan.md`](../planning/2026-10-09_18-02_SUPERB-v5-schema-evolution-execution-plan.md))

## Context

Four fleet apps carry event payloads that evolve, and the repo had four disconnected
half-mechanisms for it (hand-rolled closure upcasters in bank-sync's `upcasting.go` at 116 lines,
DiscordSync's at 211, cqrs-htmx surfacing `SchemaVersion` with zero upcasters, and `schema/`'s
`Upcaster` interface nobody composes). The proposal (2026-10-09) adopted a two-axis model —
Axon's payload-upcasting axis plus LiveStore's declared-schema drift-detection axis — sequenced
T2-first. T2 (named upcast ops) and T1-core (the `system.DomainConfig.Schema` declaration)
shipped the same day, gate-green; this ADR records the implementation decisions so the semantics
are citable rather than inferred from code. It deliberately covers T2+T1 as ONE unit (D2 ruling,
execution plan decision log): the declaration without the ops is an inert list, and the ops
without the declaration are the same hand-wiring the proposal set out to kill.

## Decision 1 — Op set, matching, and versioning semantics (T2)

`schema.Compile(ops...)` compiles seven sealed ops (`Op` interface with unexported `op()`) into a
`Chain`: `RenameType(from, target)`, `RenameField(type, ver, from, target)`,
`AddField(type, ver, field, default)`, `RemoveField(type, ver, field)`,
`Transform(type, ver, fn)`, `Split(type, ver, outputs...)` via `Producing(t, fn)`, and
`Drop(type)`.

- **Matching is most-specific-first and order-independent**: exact `(type, version)` beats
  type-only (`RenameType`, `Drop`). Duplicate exact matches are compile errors; duplicate
  rename targets and rename cycles are compile errors (iterative DFS detection).
- **Versioning follows Axon's rule**: payload ops (`RenameField`/`AddField`/`RemoveField`/
  `Transform`) advance the schema version by exactly +1 and never touch identity; only
  `RenameType` changes identity (and keeps the version). `Split` outputs get fresh event IDs,
  inherit stream/position/metadata/encoding/timestamp, and land at source version +1.
- **Identity is preserved for 1:1 ops** — event ID, stream, position, timestamp, metadata, and
  encoding stamp all survive an upcast. The hand-rolled closure style could silently mint new
  IDs (a real dedup hazard); the chain cannot.
- **Codec honesty**: decode uses the event's own `Encoding()` stamp; re-encode uses the same
  codec. Mixed JSON/CBOR streams upcast correctly with zero configuration.
- **Decode failures are per-op policy**: `Fail` (default) | `Passthrough` | `Drop`, set via
  `WithDecodePolicy`. No skip-as-default (Equinox's stance rejected — silent skipping loses
  facts a fold needs).
- **NO gap validation**: shape-preserving version bumps legitimately need no op, so op source
  versions are never required contiguous. Compile errors are reserved for ambiguity (duplicates,
  cycles, target conflicts), not absence. This is a documented delta from the proposal sketch.

## Decision 2 — Batch `SourceTransform` vs the `Upcasters` bridge

`Split` (1→N), `Drop` (1→0), and `RenameType` (identity change) compile into the batch-level
`Chain.SourceTransform()` (`func([]Event) ([]Event, error)` — ADR-0126's `SinkTransform`/
`SourceTransform` shape) and CANNOT map onto the single-event `Upcaster` interface, which
presupposes one event in, one event out, same identity. `Chain.Upcasters()` — the bridge for
consumers wiring the legacy `Upcaster` registry — therefore rejects a chain containing any of
those three (Rejection-family `ErrBatchOpNotConvertible`) instead of silently omitting them.
Silent omission was an implementation bug caught before ship: a chain that quietly dropped a
`Split` from its Upcasters view would upcast differently depending on which seam consumed it.

## Decision 3 — `DomainConfig.Schema` composition point (T1)

`schema.Event(t, currentVersion, ops...)` builds an `EventSchema`; `schema.Declare(schemas...)`
validates the set (non-empty, unique wire names, positive versions, op targets matching declared
types, op source < current) and compiles one `Chain`. `system.DomainConfig.Schema
[]schema.EventSchema` is the ONLY composition point (ADR-0123: extend the config, never a third
registry). When non-empty, `system.New` decorates `sys.eventStore` via `event.DecorateStore`
AFTER every store-assignment path (instance wiring, cache wrapper, memory fallback) — one
decoration covers both read seams: decider loads and the projection-host journal. Writes and
stored bytes stay untouched (upcasting is a read-time projection; the journal is immutable
truth, ADR-0143).

Declared schema event types join `DomainConfig.Events` for the coeffect gate — a payload-contract
declaration is also a journal-universe declaration. Layering is system (Tier 5) → schema
(Tier 2): downward, `check-arch`-legal; the reverse (schema importing system) would violate the
layer gate and is not attempted.

## Alternatives rejected

Ported from proposal §11 plus implementation-time rejections:

- **Status-quo closures** (bank-sync/DiscordSync hand-rolled upcasters) — proven to work but
  unauditable, per-app, and silent-ID-minting; the exact drift the proposal measured.
- **Per-op `Upcaster` interface only** — cannot express `Split`/`Drop`; rejected as the sole
  compilation target (Decision 2 keeps it as a guarded bridge for legacy consumers).
- **Gap validation** (contiguous op versions required) — rejected: version bumps without shape
  change are legitimate; absence of an op is not ambiguity.
- **Schema-registry service** — reintroduces client/server coupling the library refuses.
- **Cross-event migrations** (enrichment, stream merges) — one-shot runbook tools, not library
  semantics (AxonIQ drew the same line).
- **Downcasting** — deployment-ordering concern, not a library concern.
- **Skip-as-default decode policy** — loses facts; deliberate `Drop` op + per-op policy instead.
- **Version-prefixed event names** — rejected twice in research (§5, §8 of the proposal).

## Consequences

- **Positive**: one declaration drives read-path upcasting for every `system.New` consumer;
  identity-preserving semantics close the dedup hazard; compile-time validation catches
  ambiguity before boot; bank-sync's 116-line upcasting package collapses to a declarative
  `UpcastChain()` (proven via the preserved pilot patch).
- **Negative**: `Upcasters()` users cannot express batch ops through the legacy bridge (by
  design); the declaration is in-process only (no cross-service governance — deliberate).
- **Neutral**: the chain runs on every event load; cost is benchmarked in the execution plan's
  trust tier (M14) rather than assumed.
- **Follow-ups** (execution plan): TypeDecoder derivation (M6), catalog render + semver bridge
  (M7), cqrs-lint undeclared-event rule (M8) — the declaration becomes load-bearing for
  decoders, governance export, and lint; snapshot state-shape stamp (M11–M13) keeps snapshots
  honest under upcasting.
