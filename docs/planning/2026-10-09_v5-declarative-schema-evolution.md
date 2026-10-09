# V5 Declarative Schema Evolution — Lessons from Axon Framework, LiveStore, Equinox, and Akka

- **Status:** Proposal (not accepted). Awaiting owner ruling; on acceptance this splits into implementation ADRs.
- **Date:** 2026-10-09 (upgraded same-day after verification pass: external claims re-verified against primary sources, fleet evidence added, recommendations added)
- **Related:** ADR-0111 (Record + SchemaVersion), ADR-0126 (SourceTransform upcasting), ADR-0114 (tombstones as events), ADR-0136 (invertibility ladder), ADR-0143 (engine reset never deletes the journal), ADR-0151 (Evolutions are the declaration), ADR-0152 (v5 dual-support topology)

## Verification status

Every load-bearing external claim below was verified against a primary source (raw fetch + mechanical extraction, not summarizer paraphrase). Fleet claims verified by reading the sibling checkouts.

| Claim                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | Status                                                                                                                       | Source                                                                                                                               |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Axon upcaster model: revision x → x+1 steps, chain, `RevisionResolver` variants (`Annotation`/`SerialVersionUID`/`FixedValue`/`Maven`), `IntermediateEventRepresentation` laziness, payload+type+metadata-additions only, ordering burden on `EventUpcasterChain`/`@Order`                                                                                                                                                                                                                                                                                                                | ✅ Verified                                                                                                                  | `AxonIQ/reference-guide` `axon-framework/events/event-versioning.md` (raw)                                                           |
| Axon context-aware upcaster non-determinism ("the context contains different state depending on what's included in the event stream" — token position / aggregate scope)                                                                                                                                                                                                                                                                                                                                                                                                                  | ✅ Verified                                                                                                                  | same page, verbatim                                                                                                                  |
| AxonIQ Framework 5.2+ `EventTransformation`: `from(MessageType)`, `rename`, `split(...).producing(...)`, `drop`, `transform(Class<T>, fn)`; exact match beats predicate "independent of registration order"; duplicate exacts rejected at build time (`ChainConfigurationException`); `transform` may change only the version, only `rename` may change the name; `@Event` without version defaults `0.0.1`; "declarative successor to the Axon Framework 4 upcaster"                                                                                                                     | ✅ Verified                                                                                                                  | `AxonIQ/axoniq-framework` `axoniq-message-transformation/.../EventTransformation.java` (source) + AF 5.3 reference guide             |
| Axon `RevisionSnapshotFilter` (stale-revision snapshots skipped, aggregate rebuilt from events)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | ⚠️ Unverified lead: reported by research agent, not independently confirmed by primary fetch — T4 below does not depend on it | apidocs (agent-cited)                                                                                                                |
| LiveStore: fingerprint-based migration (`__livestore_schema` per-table hash + `__livestore_schema_event_defs` per-event-name hash, both with `updatedAt`), boot compare + rematerialize, state tables "SAFE TO CHANGE... automatically rebuilt from eventlog", eventlog tables "NEVER modify... causes data loss", `__livestore_rebuild` marker ("only a completed replay and successful migration hooks may insert the marker row"), storage-format versioning is a TODO, `validateSchema`, migration behaviours `create-if-not-exists`/`drop-and-recreate`, OTel span on `migrateTable` | ✅ Verified                                                                                                                  | `livestorejs/livestore` `packages/@livestore/common/src/schema-management/migrations.ts` + `.../system-tables/state-tables.ts` (raw) |
| Equinox: `FsCodec.IEventCodec` contract, explicit `Codec` = "an explicitly coded pair of `encode` and `tryDecode` functions", NewtonsoftJson codec = "versionable convention-based approach... serializer-agnostic schema evolution with minimal boilerplate", snapshots/unfolds are events with "support for multiple co-existing compaction schemas", "Events are never destroyed, updated or touched in any way, ever", un-matching old events are "as good as not there" (skip semantics)                                                                                             | ✅ Verified                                                                                                                  | `jet/equinox` `README.md`@master (raw)                                                                                               |
| Akka: `akka.persistence.typed.EventAdapter[E, P]` — to/from-journal conversion, doc-listed purposes include "migration by splitting up events into sequences of other events" and "migration filtering out unused events, or replacing an event with another"                                                                                                                                                                                                                                                                                                                             | ✅ Verified                                                                                                                  | `akka/akka-core` `akka-persistence-typed/.../EventAdapter.scala` (source)                                                            |

## 1. Problem: four disconnected half-mechanisms

Verified current state (file:line evidence):

| Concern                            | Today                                                                                  | Where                                                         | Gap                                                                                                                                                                                                    |
| ---------------------------------- | -------------------------------------------------------------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Payload version                    | `Record.SchemaVersion int`, set once at creation                                       | `record/record.go:104-107` (ADR-0111)                         | An integer with no declaration of what version N means; every engine defaults it to 1 (DiscordSync's journal comment documents the fleet-wide effect)                                                  |
| Payload transformation             | Imperative `Upcaster` = full event-rebuild closure per (type, version)                 | `schema/upcaster.go:7-11`, `schema/doc.go`                    | The 80% ops (add field, rename field, rename type, split) have no shorthand; every step hand-writes decode/validate/reshape/re-encode/`event.New`                                                      |
| Event identity at governance level | `catalog.Event(id, direction, WithVersion("1.0.0"))` — EventCatalog export only        | `catalog/message_config.go:89-90,214-245`                     | Never consulted at runtime; wire-facing and doc-facing schema surfaces can silently diverge — and they already disagree on TYPE: catalog versions are semver strings, `Record.SchemaVersion` is an int |
| Read-model shape change            | `ReplanLayout` diffs in-process layout OPTIONS, `ConfirmRebuild` replays from EventLog | `metaengine/relayout.go:64,171`                               | In-process only. No persisted per-collection fingerprint, so a fold changed while the process was down is invisible at boot; stale materialized state serves queries                                   |
| Snapshots                          | ADR-0044 envelope stamps codec, not state shape                                        | `snapshot/` (no revision/state-hash field in `snapshot/*.go`) | A snapshot written by old fold code loads as if current                                                                                                                                                |

## 2. Fleet evidence: the pain is real, current, and measured

First-party consumers already pay the tax this proposal removes (per ADR-0152 consumer scope, these ARE the market):

- **bank-sync** wrote a ~45-line closure for ONE evolution step (`internal/cqrs/upcasting.go:29-72`: unmarshal → validate `currency` present → reshape flat `amountCents`/`currency`/`reservedCents` into nested `amount`/`reserved` Money objects → delete three fields → marshal → `event.New`), wired via `event.DecorateStore` at TWO call sites (`infrastructure.go:374-377, 571-574`). It then hand-rolled `NewFieldRenameUpcaster` (`upcasting.go:75+`) — a consumer-side reimplementation of exactly the named-op layer T2 proposes. And because upcasting alone was not the whole story, bank-sync also maintains a destructive `migrate-journal` runbook (`docs/runbooks/migrate-journal.md`), with the comment that upcasting is "defensive rather than load-bearing."
- **DiscordSync** maintains a 211-line `internal/eventschema/upcasters.go` package whose contents are two TYPE RENAMES (`AttachmentMigrated → AttachmentBackedUp`, `EmbedMediaMigrated → EmbedMediaBackedUp` — the Axon `EventTypeUpcaster` / AxonIQ `rename()` case) plus add-derived-field upcasters, and had to reason in package docs about idempotency and best-effort error policy per upcaster — policy the library should own.
- **cqrs-htmx** surfaces `evt.SchemaVersion()` to its consumers (`sync_pull.go:282`, `event_catalog.go:24`) but registers no upcasters; **go-appkit** never touches schema evolution.

Implications, taken seriously:

1. T2 (named ops) has demonstrated demand — a consumer built it locally. Ship it first.
2. A **`Transform`** (typed closure) op must remain first-class: bank-sync's Money-object restructuring is NOT expressible as field renames; the 20% needs the escape hatch.
3. Error policy belongs in the op layer: bank-sync's Corruption-family wrapping and DiscordSync's "best-effort passthrough" should be declarative (`OnDecodeError: Fail | Passthrough | Drop`, default Fail with Passthrough/Drop explicit) — today every consumer invents it.
4. The §11 boundary is consumer-confirmed: bank-sync's `migrate-journal` runbook exists precisely because cross-event/one-shot migrations are OUT of upcasting's league. Do not pull them in.

## 3. The two-axis insight (LiveStore's core lesson, primary-verified)

LiveStore's migration source opens with the "CRITICAL DISTINCTION": state tables are "SAFE TO CHANGE" because changes trigger rematerialization from the eventlog; eventlog tables must "NEVER" change because "changes cause data loss." That splits schema evolution into two independent axes:

- **Axis A — payload evolution** (stored events must decode into current code). Axon's domain: revision steps + chains, read-time, non-destructive ("the complete event history remains intact").
- **Axis B — read-model evolution** (folds/layouts change freely). LiveStore's domain: migration IS rematerialization.

go-cqrs-lite already owns the hard half of both axes: Axis A via capability-preserving `UpcastSourceTransform` chains (ADR-0126 — same lazy, non-destructive architecture Axon uses); Axis B via `EngineResetter` everywhere + journal-survives-reset (ADR-0143) + `ConfirmRebuild` replaying from the EventLog. The missing piece is not machinery — it is the **declaration** that makes drift detectable instead of discovered. LiveStore persists exactly such fingerprints; AxonIQ carries the equivalent on every message (`MessageType` = name + version).

Both axes sit on the ADR-0136 ladder's **replayable** rung: their inverse is the journal itself.

## 4. Axon / AxonIQ Framework: adopt / reject

| Mechanism                                                                                                                | Verdict                                                                                                                                             | Why (repo-specific)                                                                                                                                                                                                           |
| ------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Revision stamped into stored type; current revision declared next to the payload type                                    | **Adopt (half-have)**                                                                                                                               | `Record.SchemaVersion` rides the record; what's missing is the declaration of the CURRENT version — T1 adds it                                                                                                                |
| Logical identity decoupled from class names (`MessageType` = QualifiedName + version)                                    | **Adopt**                                                                                                                                           | We already use wire names (`event.Type` = `record.Type`, ADR-0111 lockstep). Codify: identity is (wire name, version), NEVER the Go struct name                                                                               |
| Named declarative ops: `rename`, `split(...).producing(...)`, `drop`, `from(...).to(...).transform(...)`                 | **Adopt as T2**                                                                                                                                     | Kills the one-closure-per-step boilerplate (§2). Each op compiles to the existing `Upcaster`; read path untouched                                                                                                             |
| "Most specific match wins... independent of registration order"; duplicate exact identities rejected at chain BUILD time | **Adopt**                                                                                                                                           | Order-dependent chains are a silent-corruption farm; build-time validation beats runtime surprises. Our registry already keys on (type, version) — extend with build-time duplicate/gap checks                                |
| `transform` may change ONLY the version; only `rename` may change the name                                               | **Adopt**                                                                                                                                           | A blast-radius constraint that makes chains auditable: type identity changes are greppable by construction                                                                                                                    |
| Lazy stacked representations; content converted only when pulled                                                         | **Have**                                                                                                                                            | Go closures over `event.Event` are already cheap; Axon's `ContentTypeConverter` graph is a pain point we don't need (codec is stamped per event, `Record.Encoding`)                                                           |
| Context-aware upcasters (pull state from earlier events)                                                                 | **Reject**                                                                                                                                          | Axon's own docs state the context "contains different state depending on what's included in the event stream" — non-deterministic across read windows. Cross-event work stays a migration (§11), exactly where AxonIQ left it |
| `RevisionSnapshotFilter`                                                                                                 | **Unverified lead** (see verification table) — T4 is justified independently: snapshots are pure optimization, the journal is truth (ADR-0136/0143) |                                                                                                                                                                                                                               |

## 5. LiveStore: adopt / reject

| Mechanism                                                                                                 | Verdict              | Why (repo-specific)                                                                                                                                                                                                                         |
| --------------------------------------------------------------------------------------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| One declared schema (events + tables) per app                                                             | **Adopt as T1**      | The "defined Schema" ask. In Go: a declarative registry binding wire name, current version, codec, Go type, and evolution chain — one object, multiple consumers                                                                            |
| `__livestore_schema`: per-table fingerprint persisted with `updatedAt`; boot compares, mismatches migrate | **Adopt as T5**      | We have the rebuild (`ConfirmRebuild` + ADR-0143) but not the persisted fingerprint. Engine-side per-collection fingerprint + boot diff closes the "fold changed while down" hole via the existing `RebuildThreshold`/`ConfirmRebuild` gate |
| `__livestore_schema_event_defs`: per-event-name schema hash ledger                                        | **Adopt as T3**      | Wire-level drift detection: "stored data is what my code thinks it is" without a registry service                                                                                                                                           |
| `__livestore_rebuild` marker: only a completed replay may insert it                                       | **Adopt (into T5)**  | Atomic rebuild-completion marker prevents serving half-rebuilt state after a crash mid-replay — a failure mode our reset path currently papers over with lock discipline                                                                    |
| Compat rules: add-with-default safe, remove-field safe, removing an event DEFINITION never                | **Adopt as T6**      | Small, teachable, lintable                                                                                                                                                                                                                  |
| Eventlog tables NEVER modified; storage-format versioning honestly TODO                                   | **Restate as axiom** | Matches ADR-0143 verbatim in spirit: the journal is the one thing no migration touches                                                                                                                                                      |
| Weak enforcement (hash mismatch recorded, strict check TODO upstream)                                     | **Invert**           | Repo precedent is warn-first → hard (ADR-0136/0151). Ship advisory with opt-in hard mode; promote on fleet burn-in                                                                                                                          |
| Version-prefixed event names (`v1.TodoCreated`) with per-version materializers                            | **Reject**           | Needed there because materializers key on the name; our upcast-before-fold normalizes payloads BEFORE the Evolution runs, so one fold per logical type suffices. Rejected again in §8 (semver analysis)                                     |
| Sync backend, client documents, rebasing                                                                  | **Out of scope**     | Server library; watermill owns distribution                                                                                                                                                                                                 |

## 6. Ecosystem cross-check: Equinox and Akka

| System                | Mechanism (verified)                                                                                                                                                                                                                                                                                                                                                                            | Takeaway for v5                                                                                                                                                                                                                                                                                                                                                                                                |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Equinox (jet/equinox) | `IEventCodec` per event type: explicit `encode`/`tryDecode` pair; convention-based codec gives "serializer-agnostic schema evolution with minimal boilerplate"; old events the codec's union doesn't match are skipped ("as good as not there"); snapshots/unfolds are events with "multiple co-existing compaction schemas"; "Events are never destroyed, updated or touched in any way, ever" | Confirms the declaration-per-type shape (T1) and journal immutability. Its skip-instead-of-upcast stance is REJECTED as a default: skipping silently drops data from folds — bank-sync must count every `BalanceUpdated`, not skip undecodable ones. But deliberate drop must exist as an op (AxonIQ's `drop`; our `Drop` covers it), and Equinox's multiple co-existing snapshot schemas reinforce T4's stamp |
| Akka Persistence      | `EventAdapter[E, P]` to/from-journal conversion; documented purposes include splitting one event into a sequence and filtering/replacing events                                                                                                                                                                                                                                                 | Confirms split/drop as first-class evolution verbs across ecosystems — not Axon exotica. Akka's adapter is write-read-path-coupled; our compile-to-`Upcaster` composition stays capability-preserving (ADR-0126)                                                                                                                                                                                               |

## 7. The proposal: one declaration, seven increments

### Integration rule first (ADR-0123): `system.DomainConfig` IS the declaration surface

The composition root already declares the domain — verified in code:

- `DomainConfig.Events []event.Type` — wire names, coeffect-gated (`system/config_types.go:49-57`)
- `DomainConfig.Evolutions []EvolutionSpec` — the fold declarations, ADR-0151's sanctioned form (`config_types.go:44-47`)
- `DomainConfig.Projections []ProjectionDeclaration` — sealed, typo-proof (`config_types.go:36-42`)
- `DomainConfig.ProjectionTypeDecoder` — typed payload decoding via `projectionadapter` (`config_types.go:71-78`)

Therefore the v5 schema declaration **extends `DomainConfig`, never sits beside it**. Concretely: a `DomainConfig.Schema []schema.EventDef` (typed superset of `Events`: name + current version + codec + Go type + chain) with `Events` derived when `Schema` is present — one declaration, four consumers (coeffect gate, projection decoders — `TypeDecoder` registration becomes DERIVABLE from `Schema`, cqrs-lint, catalog export). `schema.Declare` (below) is the Tier-2 builder; `system.New` is the only composition point consumers touch. Layering holds: system (Tier 5) → schema (Tier 2).

**The missing seam, verified:** `system/` and `metaengine/` contain ZERO upcaster references — a `system.New` consumer cannot inject upcasting today. bank-sync/DiscordSync wire `event.DecorateStore` only because they compose lower layers directly. The fix is part of T2/T1: upcasters declared via `DomainConfig` are applied INSIDE system's adapter layer (`EventAdapter.ReadFrom`, `system/adapter_event_journal.go:16`, for the projection-host path; the decider load path likewise) — capability-preserving decoration becomes the composition root's job, not each consumer's.

### T2 — Named upcast ops (first; demonstrated demand)

Illustrative signatures (the `v5schema` qualifier names the proposed API — none exist yet):

```go
// skip-validate
v5schema.RenameType("attachment.migrated", "attachment.backed_up")     // DiscordSync's rename family
v5schema.RenameField("user.created", 1, "name", "displayName")
v5schema.AddField("user.created", 1, "country", "US")                  // default-valued
v5schema.RemoveField("user.created", 2, "legacyToken")
v5schema.Transform("balance.updated", 1, reshapeToMoney)               // bank-sync's case: typed/map-level fn, the only domain logic
v5schema.Split("checkout.completed", 1, v5schema.Producing("cart.checked_out", cartFn), v5schema.Producing("payment.requested", payFn))
v5schema.Drop("audit.legacy_ping")                                     // deliberate skip (Equinox's verb, explicit)
```

- Each op compiles to the existing `Upcaster` interface; `UpcastSourceTransform`/`DecorateStore`/`DecorateJournal` untouched (ADR-0126 capability preservation). Following AxonIQ: transform ops may change only the version; only `RenameType` changes identity.
- Chain construction validates at BUILD time: duplicate (type, version) exact matches rejected; most-specific-first matching, order-independent (adoption of AxonIQ semantics; cycle detection already exists in `schema/registry.go`).
- Declarative error policy per op: `OnDecodeError` Fail (default) | Passthrough | Drop — fleet-confirmed need (§2.3).
- Blast radius: `schema/` only (new `ops.go`), table-driven tests, api-stability golden regen, doc-check. **Additive, non-breaking, v4.x-shippable.** (The system seam above lands with T1 — ops are usable standalone via `UpcastSourceTransform` immediately.)

### T1 — The declared schema, extending `system.DomainConfig`

```go
// skip-validate
// Illustrative v5 surface: `Schema` is the proposed DomainConfig field; the
// v5schema qualifier names the proposed builder API — none exist in the repo yet.
sys, err := system.New(ctx,
    system.DomainConfig{
        Schema: v5schema.Declare(
            v5schema.Event[user.Created]("user.created", 2),
            v5schema.Event[user.Renamed]("user.renamed", 1,
                v5schema.From(1, v5schema.Rename("name", "displayName"))),
            v5schema.DefaultCodec(codec.CBORCodec{}),
        ),
        Evolutions: []system.EvolutionSpec{ /* folds — unchanged, ADR-0151 */ },
    },
    system.DeploymentConfig{ /* engines — unchanged */ })
```

- One registry binding wire name + CURRENT version + codec + Go type + evolution chain. Construction-time validation covers duplicate names, version gaps, and chain integrity; `Events` derives from it when present (coeffect gate unchanged in behavior).
- **Unifies the split-brain at the root:** `catalog` (which already models `WithVersion` — `catalog/message_config.go:89`) renders the SAME declaration for governance export; `projectionadapter.TypeDecoder` registrations become derivable from it; cqrs-lint's undeclared-event rule reads it.
- Upcasters declared here are applied inside system's adapters (see integration rule) — `system.New` consumers get upcasting without hand-wiring.
- Blast radius: `schema/declaration.go` (new) + `system/config_types.go` (field + adapter application) + `catalog/` bridge + one cqrs-lint rule, api golden, doc-check, module-catalog test untouched (no new module). **Additive; v4.x-shippable with `Schema` optional at first.**

### T4 — Snapshot state-shape stamp

- Snapshot envelope gains the state-shape version of the writing code (declaration hash or explicit layout version). Missing stamp (pre-v5 snapshots) = accept (compat default); present + mismatch = discard (counted stat) and rebuild from journal — snapshots are optimization, journal is truth (ADR-0136 replayable rung; ADR-0143 guarantees the journal survived).
- Blast radius: `snapshot/` envelope + `decider/` snapshot path + conformance test. **Non-breaking (additive envelope field).**

### T3 — Event payload fingerprint ledger

- Write side: stamp a payload-shape fingerprint (hash of the DECLARED shape, never the ciphertext — see §10) next to the existing `Encoding` stamp. Read side: compare against the declaration; mismatch = advisory warning (LiveStore's `__livestore_schema_event_defs` analog).
- Staged stamp location: metadata field first (no `Record` change, v4.x-able), first-class `Record` stamp at the v5 breaking window if burn-in favors it.
- Blast radius: `record/` stamp helper + `schema/` check + write paths via options. **Advisory-only initially.**

### T5 — Persisted layout fingerprints + boot drift gate

- Engines persist a per-collection layout fingerprint — the hash of the metaengine **LayoutPlan** (`BuildLayoutPlanFromType` output, the shape `LayoutPlanApplier` materializes) — alongside materialized data (cleared with collections on reset; journal exempt per ADR-0143). Boot compares declared vs persisted → `LayoutDiff`s → existing `RebuildThreshold` auto-rebuild / `ConfirmRebuild` operator gate. Adds LiveStore's completed-replay marker so a crash mid-rebuild never serves half-rebuilt state.
- `system.New` boot runs the check as part of composition (the operator gate surfaces through the existing Doctor/health surfaces).
- Blast radius: `metaengine/` (fingerprint store + boot check; reuses `PlanDiff` machinery, `plan_diff.go:47`), sqliteengine as reference implementation, then engine conformance suite, `system/` boot hook. **L; the only increment needing per-engine work.**

### T6 — Compatibility policy + lint

- Codify: add field with default = safe; remove field = safe for forward reads; rename/retype = requires a declared op; remove event definition = never. Encode as a cqrs-lint rule against the T1 declaration; document in `schema/README.md` + recipes.

### T7 — Docs + ADR split

- On acceptance: T1+T2 → declaration/ops ADR; T3+T4 → stamping ADR; T5 → boot-gate ADR. Reconcile `schema/README.md`, add a recipes section, update core.md §3 conventions.

### Sequence

**T2 → T1 → T4 → T3 → T5 → T6/T7.** T2 first (demand-proven, additive, unblocks the others' vocabulary); T1 then has ops to declare; T4 small and independently correct; T3/T5 add persistence coordination; policy and docs follow code. Pareto: T2+T4 ≈ 80% of the evolvability win at ~20% of the effort.

## 8. Version identity: keep the int, derive the semver

The repo already has TWO version grammars: `Record.SchemaVersion int` (wire, every engine, cqrs-htmx passthrough `sync_pull.go:282`) and catalog's `"1.0.0"` semver string (governance export). AxonIQ chose semver strings (`MessageType` `"1.0.0"`, default `"0.0.1"`), but they never had our installed base of integer columns defaulting to 1.

**Recommendation: the wire keeps the int; the declaration owns semantics; catalog's semver string becomes derived display metadata.**

- Chains compose naturally on +1 steps; our compat classes are ENFORCED by declaration + ops (T1/T6), not parsed out of a version string — semver's MAJOR/MINOR would be decorative.
- Changing the wire type to string breaks every stored row, every engine DDL, and cqrs-htmx's passthrough for zero enforced guarantee.
- `WithVersion("2.0.0")` keeps working: the declaration maps it to the int current-version (2) for checks and renders it back on export. Split-brain resolved at the declaration, not at the wire.

## 9. Worked example: bank-sync's BalanceUpdated, before/after

Today (~45 lines + wiring at two call sites, `internal/cqrs/upcasting.go:29-72`):

```go
// skip-validate
import "github.com/larsartmann/go-cqrs-lite/schema/v4"

schema.NewUpcaster(EventBalanceUpdated, schemaV1, func(evt event.Event) (*event.ImmutableEvent, error) {
    var raw map[string]any
    if err := json.Unmarshal(evt.Payload(), &raw); err != nil {
        return nil, errorfamily.WrapCorruption(err, "upcast.unmarshal", "unmarshal BalanceUpdated v1 payload")
    }
    currency, ok := raw["currency"]
    if !ok {
        return nil, errorfamily.NewCorruption("upcast.missing_currency", "BalanceUpdated v1 payload missing required 'currency' field")
    }
    raw["amount"] = map[string]any{"cents": raw["amountCents"], "currency": currency}
    raw["reserved"] = map[string]any{"cents": raw["reservedCents"], "currency": currency}
    delete(raw, "currency"); delete(raw, "amountCents"); delete(raw, "reservedCents")
    payload, err := json.Marshal(raw)
    if err != nil {
        return nil, errorfamily.WrapCorruption(err, "upcast.marshal", "marshal BalanceUpdated v2 payload")
    }
    return event.New(evt.Type(), evt.StreamID(), evt.StreamType(), evt.Version(), payload, event.WithCodec(codec.JSONCodec{}))
})
```

Proposed (the reshaping function — map-level, ~6 lines — remains the ONLY domain logic):

```go
// skip-validate
v5schema.Transform("balance.updated", 1, func(raw map[string]any) error {
    raw["amount"] = map[string]any{"cents": raw["amountCents"], "currency": raw["currency"]}
    raw["reserved"] = map[string]any{"cents": raw["reservedCents"], "currency": raw["currency"]}
    delete(raw, "currency"); delete(raw, "amountCents"); delete(raw, "reservedCents")
    return nil
})
```

Decode, codec selection, validation-ordering, re-encode, event reconstruction, version stamping, error-family wrapping: library-owned, uniform across the fleet. DiscordSync's two type renames become two `RenameType` one-liners; its add-derived-field upcasters become `Transform` closures without the ceremony.

## 10. Failure modes and interactions (asked before shipping, not after)

1. **Encryption/signing:** T3 fingerprints hash the DECLARED SHAPE, never payload bytes — ciphertext changes (key rotation) must not read as schema drift. Signature verification stays BEFORE upcasting (already true: transforms run post-read); upcasters emit new immutable events and never re-sign.
2. **Rolling deploys:** two app versions writing v1/v2 of the same type concurrently. The declaration stamps the writer's current version; readers normalize via the chain; the ledger warns on older-shape rows — mixed-version streams are the normal state, not an error (same as Axon/LiveStore).
3. **Rename mid-flight:** `RenameType` leaves two logical names in the journal. The declaration must register the alias (rename op covers reads); cqrs-lint + coeffect checks must accept both names as one identity — a T1/T6 test case, not an afterthought.
4. **Hash mechanics:** fingerprints hash shape (field names + types, canonicalized), not content — renames are drift by design; collisions are irrelevant at shape-space entropy.
5. **Rebuild atomicity (T5):** crash mid-replay must not serve half-rebuilt state — LiveStore's completed-replay marker row; `ResetResult.Partial()` discipline (ADR-0136) already names the contract.
6. **CatchUp reprobe (ADR-0137):** quarantine reprobe replays the journal through `projectionadapter`; the decorated (upcasting) journal must be the one attached there too — existing wiring pattern, becomes a T5 conformance assertion.
7. **Snapshots + upcasting:** upcasters run on the journal; snapshots bypass them by design — T4's stamp is what keeps that honest (stale-shape snapshot = discard + fold the upcasted journal).

## 11. What we deliberately do NOT build

- **No schema-registry service** — the declaration is in-process, versioned with the app; a registry reintroduces client/server coupling the library refuses.
- **No cross-event migrations** (enrichment from external data, merges across streams) — AxonIQ drew the same line; bank-sync's `migrate-journal` runbook shows these are one-shot tools, not library semantics.
- **No downcasting** — old code reading new events is a deployment-ordering concern.
- **No skip-as-default-policy** (Equinox's stance) — deliberate `Drop` op only; silent skipping loses facts the fold needs (bank-sync must count every balance event).
- **No version-prefixed event names** — rejected twice (§5, §8).
- **No revival of runtime `Infer`** — dead per ADR-0151; the declaration is explicit and auditable, which is the point.

## 12. Open questions → recommendations

> **Ruling (2026-10-09, same day):** the owner re-issued the standing execution loop ("execute and verify, repeat until done") instead of objecting to any recommendation — adopted as delegated approval of all three recommendations below plus the **T2-first sequence (T2 → T1 → T4 → T3 → T5 → T6/T7)**, with T2 landing v4.x-additive and fleet apps migrating opportunistically (bank-sync pilot first). Implementation ADRs split out as each increment lands (T7).
>
> **T2 SHIPPED 2026-10-09 (same day):** `schema/ops.go` + `schema/chain.go` + `schema/chain_engine.go`. All seven ops land as specified; the bank-sync pilot ran via a temporary sibling replace (build + `internal/cqrs` tests + race green; collapse = 116-line upcasting.go → declarative `UpcastChain()`; patch preserved at `docs/planning/2026-10-09_t2-pilot-bank-sync.patch`, adoption rides the next schema/v4 tag). Design deltas from the sketch, all verified in tests: (a) `Split`/`Drop`/`RenameType` compile into the batch-level `Chain.SourceTransform()` — they cannot map onto the single-event `Upcaster` interface (count/identity changes), which `Chain.Upcasters()` enforces by rejecting them; (b) no "gap validation" — shape-preserving version bumps legitimately need no op, so source versions are never required contiguous (only duplicates and rename-graph ambiguity are compile errors); (c) ops preserve event ID/metadata/timestamp (the closure style could silently mint new IDs — a real dedup hazard).
>
> **T1 CORE SHIPPED 2026-10-09 (same day):** `schema/declaration.go` (`EventSchema` + `Event` + `Declare`) + `system.DomainConfig.Schema` + `system/schema.go`. One decoration of `sys.eventStore` (placed after every assignment path) covers both read seams — decider loads and the projection-host journal; Schema types join the coeffect universe. Remainder for the follow-up increments: TypeDecoder derivation, catalog render, cqrs-lint rule, and the tag wave that publishes schema v4.6.0 + the system bump together (GOWORK=off per-module builds need the co-released pins).

1. **Declaration home?** → **Type lives in `schema/` (Tier 2 builder); composition point is `system.DomainConfig` (ADR-0123); `catalog/` renders.** No third registry: `DomainConfig.Schema` derives `Events` (coeffect gate), decoder registrations, and the catalog export from one list. Layering: system (Tier 5) → schema (Tier 2) is downward and legal; the reverse would violate `check-arch`. (Confirm with a `#check-arch` dry run at implementation time.)
2. **T3 stamp location?** → **Metadata field now; first-class `Record` stamp at v5 only if burn-in favors it.** Metadata ships in v4.x with zero schema churn; the v5 window stays open for the cleaner form.
3. **Warn-first promotion?** → **Advisory + opt-in hard mode from day one; promote to hard after DiscordSync + bank-sync each run one clean minor cycle with the ledger on.** Mirrors the ADR-0151 evidence-gate style: named consumers, named criterion, not a date.
