# V5 Declarative Schema Evolution — Lessons from Axon Framework and LiveStore

- **Status:** Proposal (not accepted). Awaiting owner ruling; on acceptance this becomes an ADR (0153 or later slot).
- **Date:** 2026-10-09
- **Author:** agent research session (repo study + external research, all external claims re-derived from vendor docs)
- **Related:** ADR-0111 (Record + SchemaVersion), ADR-0126 (SourceTransform upcasting), ADR-0114 (tombstones as events), ADR-0136 (invertibility ladder), ADR-0143 (engine reset never deletes the journal), ADR-0151 (Evolutions are the declaration), ADR-0152 (v5 dual-support topology)
- **External sources:** [Axon event versioning reference](https://github.com/AxonIQ/reference-guide/blob/master/axon-framework/events/event-versioning.md), [Axon 5 upcaster migration path](https://docs.axoniq.io/axon-framework-reference/5.3/migration/paths/upcasters/), [LiveStore concepts](https://docs.livestore.dev/overview/concepts/), [LiveStore schema evolution](https://docs.livestore.dev/building-with-livestore/state/sqlite-schema/), [LiveStore migrations internals](https://github.com/livestorejs/livestore) (`@livestore/common/src/schema-management/`)

## Executive summary

go-cqrs-lite's evolution story is fragmented across four mechanisms that do not know about each other. Axon Framework (esp. its v5 rewrite) shows how to make **payload evolution** declarative and audit-friendly; LiveStore shows how to make **read-model evolution** a rebuild instead of a migration, guarded by hash-based drift detection. go-cqrs-lite already owns the hardest half of each idea (`UpcastSourceTransform` chains; journal-survives-reset + `ConfirmRebuild`), so the v5 move is not to adopt foreign machinery but to **declare** what today is implicit: one schema declaration per app that (a) binds event wire names to versions/codecs/Go types, (b) compiles to the existing upcast chain, and (c) is checked at boot against stored drift — warn-first, per the repo's reset-warn-first v4.x precedent (ADR-0136).

Seven increments (T1-T7) are proposed. The Pareto core is T1 (the declaration) + T2 (named upcast ops) + T4 (snapshot state-shape stamp): ~80% of the evolvability win for a fraction of the effort.

## 1. Problem: four disconnected half-mechanisms

Verified current state (file:line evidence):

| Concern | Today | Where | Gap |
| --- | --- | --- | --- |
| Payload version | `Record.SchemaVersion int`, set once at creation | `record/record.go:104-107` (ADR-0111) | An integer with no declaration of what version N means |
| Payload transformation | Imperative `Upcaster` = full event-rebuild closure per (type, version) | `schema/upcaster.go:7-11`, `schema/doc.go:1-24` | The 80% ops (add field with default, rename field, rename type, split) have no shorthand; every step hand-writes `event.NewEvent(...)` |
| Event identity at governance level | `catalog.Event(id, direction, WithVersion("1.0.0"))` — docs/EventCatalog export only | `catalog/message_config.go:89-90,214-245` | Never consulted at runtime; the wire-facing and doc-facing schema surfaces can silently diverge |
| Read-model shape change | `ReplanLayout` diffs in-process layout OPTIONS, `ConfirmRebuild` replays from EventLog | `metaengine/relayout.go:64,171` | Only within a live process. Nothing persists a per-collection layout fingerprint, so a fold changed while the process was down is invisible at boot; stale materialized state serves queries until someone notices |
| Snapshots | ADR-0044 envelope stamps codec, not state shape | `snapshot/` (no revision/state-hash field anywhere in `snapshot/*.go`) | A snapshot written by old fold code loads as if current; silently wrong state until first replay |

The question this document answers: **what would one declared, evolvable schema for v5 look like, and what do Axon and LiveStore teach about each piece?**

## 2. The two-axis insight (LiveStore's core lesson)

LiveStore's foundational stance: the eventlog is the source of truth; SQLite state is *pure derived data* that "can always be rebuilt" (docs.livestore.dev/overview/concepts). That splits schema evolution into two independent axes:

- **Axis A — payload evolution** (stored events must decode into current code). Axon's domain: revisions + upcast chains, read-time, non-destructive.
- **Axis B — read-model evolution** (fold functions / layouts change freely). LiveStore's domain: migration IS rematerialization; the only rule is "never touch the eventlog".

go-cqrs-lite already owns the hard half of BOTH axes:

- Axis A: `schema.UpcastSourceTransform` composed via `event.DecorateStore`/`DecorateJournal` is capability-preserving read-time transformation (ADR-0126) — the same architecture as Axon's lazy `IntermediateEventRepresentation` stack.
- Axis B: every engine implements `EngineResetter` and resets never delete the journal (ADR-0143); `ConfirmRebuild` replays from the EventLog into exactly the drifted layout; the reprobe path (`ADR-0137`) already runs reset+replay as a routine operation.

**Conclusion:** the missing piece is not machinery. It is a *declaration* that names the events, their versions, their codecs, and the expected shape of derived state — so drift becomes detectable instead of discovered. LiveStore stores exactly such fingerprints (`__livestore_schema` per-table hash, `__livestore_schema_event_defs` per-event-name hash); Axon carries the equivalent on the message (`SerializedType` = name + revision).

This also sits cleanly on the ADR-0136 ladder: both axes are **replayable** effects (read-time transform; reset+replay), with a well-defined inverse — the eventlog itself.

## 3. Axon Framework: adopt / reject

| Axon mechanism | Verdict | Why (repo-specific) |
| --- | --- | --- |
| `@Revision` stamped into `SerializedType` (name + revision) travels with every stored event | **Adopt (have it)** | `Record.SchemaVersion` (ADR-0111) already rides the record. What Axon adds: the *current* version is declared next to the type, so "what version is this type now" is one lookup, not archaeology |
| AF5 `EventTransformation`: identity = logical `MessageType` (qualified name + version, unversioned defaults to `0.0.1`), decoupled from Java class names | **Adopt the identity rule** | Wire name + version, never Go struct name. We already do this (event.Type aliases record.Type; ADR-0111's cross-type lockstep tests). Codify it as the declared identity in T1 |
| AF5 named ops: `EventTransformation.rename(from,to)`, `.split(...).producing(...)`, typed `.transform(JsonNode.class, mapper)` | **Adopt as T2** | Our `NewUpcaster` closure forces hand-rebuilding the whole event for even a one-field change. Named ops compile to the existing `Upcaster` interface — zero change to the read path |
| Most-specific-match, order-independent chain (AF5) vs AF4's registration-order `EventUpcasterChain` | **Adopt** | Order-dependent chains are a silent-corruption farm (Axon's own migration docs concede the ordering burden). Our registry already keys on (type, version) — extend match to "most-specific wins" |
| Lazy stacked representations, content only converted when pulled | **Have equivalent** | Go closures over `event.Event` are cheap; no need for Axon's `ContentTypeConverter`/`ChainingConverter` graph (its absence is a real Axon pain point: `CannotConvertBetweenTypesException` when no conversion path exists). Our codec boundary is already stamped per event (`Record.Encoding`) |
| `RevisionSnapshotFilter`: snapshots carry the aggregate revision; stale revisions are *ignored* and the aggregate is rebuilt from (upcasted) events | **Adopt as T4** | Direct fix for the snapshot gap above. Snapshots are pure optimization — correctness comes from the journal — which is exactly the replayable rung of ADR-0136 |
| Handling-time conversion to each handler's requested type (AF5) | **Have** | `DecodePayloadAuto` + self-describing events (mixed JSON/CBOR streams decode correctly) |
| `ContextAwareSingleEventUpcaster` (cross-event context within a stream window) | **Reject** | Non-deterministic across read windows (token position changes the context Axon's own staff warn). AF5 deliberately dropped cross-event transformations. Merging/enrichment across events stays out of scope (see §6) |
| FQCN-based identity (AF4) | **Reject** | The exact coupling that forced `EventTypeUpcaster` rituals for mere package renames. We never had it; keep it that way |

## 4. LiveStore: adopt / reject

| LiveStore mechanism | Verdict | Why (repo-specific) |
| --- | --- | --- |
| One `makeSchema({ events, state: { tables } })` declaration per app; types derive end-to-end from it | **Adopt as T1** | This is the "defined Schema" ask. In Go the analogue is a declarative registry: event wire name, current version, codec, Go type binding, and (when needed) the upcast chain — one object, multiple consumers |
| Hash-based state migration: per-table AST fingerprint persisted in `__livestore_schema`; boot compares and rebuilds mismatched tables by rematerializing from the eventlog | **Adopt as T5** | We have the rebuild (`ConfirmRebuild` + journal-survives-reset) but not the persisted fingerprint. Store a per-collection layout fingerprint at each engine; at boot, declared-vs-persisted drift produces `LayoutDiff`s and flows through the existing `RebuildThreshold`/`ConfirmRebuild` gate |
| Event-def hash ledger (`__livestore_schema_event_defs`): schema hash stored per event name, mismatches recorded | **Adopt as T3** | Wire-level drift detection. Stamp a payload-schema fingerprint alongside the existing `Encoding` stamp (envelope or metadata — see Open Questions); on read, compare against the declaration, warn-first. This catches "my stored events no longer mean what my structs say" — today undetectable |
| Compat rules: add field with default/optional = safe; remove field = safe (forward-compatible); **removing an event definition = never** | **Adopt as T6** | Small, teachable policy; encode as a cqrs-lint rule + docs. Removes the "is this change safe?" folklore |
| Version-prefixed event names (`v1.TodoCreated`) with per-version materializers | **Reject** | LiveStore needs it because materializers key on the name. Our chain normalizes payloads BEFORE the fold (upcast-before-fold), so one Evolution per logical type suffices — simpler, and `event.Type` stays alias-stable for catalogs. Document the rejection in T1's ADR |
| Deterministic pure materializers → replay = rebuild | **Have (restate)** | Fold discipline; worth one sentence in the v5 declaration docs, since the whole T5 mechanism leans on it |
| Weak enforcement today (hash mismatch only recorded; strict check is upstream TODO #69) | **Invert** | Repo precedent is warn-first → hard (reset warnings ADR-0136; `Infer` removal ADR-0151). Ship drift checks advisory in v5.0, promote once burn-in proves low false-positive rate |
| Sync backend / client documents / resets | **Out of scope** | Server library; watermill owns distribution |

## 5. The proposal: one declaration, seven increments

### T1 — `schema.Declaration`: the defined schema (the ask)

```go
// skip-validate
schema := schema.Declare(
    schema.Event[user.Created]("user.created", 2),            // wire name + current version
    schema.Event[user.Renamed]("user.renamed", 1,
        schema.From(1, schema.Rename("name", "displayName"))), // chain declared inline
    schema.DefaultCodec(codec.CBORCodec{}),
)
```

- One registry binding wire name, current version, codec, and Go type. Compiles at construction: missing links in a chain, duplicate (name, version) pairs, and version gaps are construction errors (cycle detection already exists in `schema/registry.go`).
- **Unifies the split-brain:** `catalog` declarations and the runtime consult the same declaration (catalog already models `WithVersion("1.0.0")` — `catalog/message_config.go:89-90` — but only for export). Governance export becomes a *view* of the runtime declaration.
- cqrs-lint gains a rule: emitted/consumed event types must be declared (symmetric to E018's coeffect gating).
- Files: `schema/declaration.go` (new), `catalog/` bridge, `cmd/cqrs-lint` rule.

### T2 — Named upcast ops (Axon AF5's best idea, on our chain)

```go
// skip-validate
schema.RenameType("user.created.v1", "user.created")       // rename wire type
schema.RenameField("user.created", 1, "name", "fullName")  // rename payload field
schema.AddField("user.created", 1, "country", "US")        // add field w/ default
schema.RemoveField("user.created", 2, "legacyToken")
schema.Split("user.checkout", 3, producing("cart.checked_out", "payment.requested"), splitFn)
```

- Each op compiles to an `Upcaster` (the `UpcastSourceTransform` read path is untouched — capability-preserving per ADR-0126).
- Field ops work on the decoded representation (`Encoding`-aware: decode via stamped codec, transform the map/struct generically, re-encode). Typed closures remain available for the 20%.
- Chain matching: (type, version) exact, most-specific-first, order-independent; construction-time chain validation already covers cycles.
- Files: `schema/ops.go` (new) + table-driven tests; `schema/README.md`.

### T3 — Event payload fingerprint ledger

- Stamp a payload-schema fingerprint (hash of declared shape at write time) next to the existing `Encoding` stamp — candidate locations: metadata field vs a new `Record` stamp (v5 is the breaking window; envelope extension mirrors ADR-0044 for blind stores).
- On read: fingerprint ≠ declaration → advisory warning (v5.0), promotable later. Gives the LiveStore guarantee ("stored data is what my code thinks it is") without a registry service.
- Files: `record/` (stamp), `schema/` (check), engines write paths unchanged (stamp rides the record).

### T4 — Snapshot state-shape stamp (Axon `RevisionSnapshotFilter` analog)

- Snapshot envelope gains the state-shape version (fold layout version or declaration hash) of the writing code.
- Loader on mismatch: **discard silently (counted stat) and rebuild from the journal** — snapshots are optimization, journal is truth (ADR-0136 replayable rung; ADR-0143 guarantees the journal survived).
- Files: `snapshot/` envelope + `decider/` snapshot cache path; conformance test in `eventtest`-style suite.

### T5 — Persisted layout fingerprints + boot drift gate

- Each engine persists a per-collection fingerprint (declared layout hash) alongside materialized data; cleared together with collections on reset (journal exempt per ADR-0143).
- Boot: declared vs persisted → `LayoutDiff`s → existing `RebuildThreshold` auto-rebuild / `ConfirmRebuild` confirmation. Closes the "fold changed while down" hole; reuses PlanDiff machinery (`metaengine/plan_diff.go:47`).
- Files: `metaengine/` (fingerprint store + boot check), one engine first (sqliteengine) as the reference implementation.

### T6 — Compatibility policy + lint

- Codify: add field w/ default = safe; remove field = safe (forward-compatible reads); rename/retype = requires declared op (T2); remove event definition = never. Encode as a cqrs-lint rule against the T1 declaration and document in `schema/README.md`.

### T7 — Docs + ADR promotion

- On acceptance: split this proposal into implementation ADRs (T1+T2 declaration/ops; T3+T4 stamping; T5 boot gate), reconcile `schema/README.md`, add a recipes.md section, and update `.agents/skills/go-cqrs-lite/references/core.md` conventions.

### Suggested sequence

T2 → T1 → T4 → T3 → T5 → T6/T7. Rationale: T2 lands immediate ergonomics on the existing path with zero surface risk; T1 then has ops to declare; T4 is small and independently correct; T3/T5 add persistence-layer coordination; policy and docs follow the code.

## 6. What we deliberately do NOT build

- **No schema-registry service** — the declaration is in-process and versioned with the app; a registry reintroduces the client/server coupling the library refuses (no opinionated transport, per README).
- **No cross-event migrations** (enrichment from external data, merging across streams) — same line Axon drew in AF5; these are one-off copy-and-replace migrations or stateful projections, not schema evolution.
- **No downcasting** — old code reading new events is a deployment-ordering concern, not a library concern.
- **No revival of runtime `Infer`** — dead per ADR-0151; the declaration is explicit and auditable, which is the point.
- **No version-prefixed event names** — rejected in §4; per-record `SchemaVersion` plus upcast-before-fold covers the same need with one Evolution per logical type.

## 7. Open questions for the owner

1. **Home of the declaration:** `schema/` module (runtime home) with a `catalog/` bridge, or declare in `catalog/` (governance home) and let `schema/` consume? Split-brain risk exists either way; the deciding factor is which module can depend on the other without violating `check-arch` tier budgets.
2. **T3 stamp location:** metadata field (cheap, no `Record` change) vs a first-class `Record` stamp (v5 breaking window, self-describing like `Encoding`). First-class is cleaner; metadata ships sooner.
3. **Promotion timing** for warn-first drift checks (T3/T5): burn-in duration or named-consumer gate, mirroring the ADR-0151 evidence-gate style.
