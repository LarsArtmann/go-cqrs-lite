# ADR-0163: Schema compatibility is a structural differ over the schema model, not a CUE subsume

| Field      | Value                                                                                                       |
| ---------- | ----------------------------------------------------------------------------------------------------------- |
| Status     | Accepted                                                                                                    |
| Date       | 2026-11-02                                                                                                  |
| Deciders   | Lars Artmann                                                                                                |
| Related    | ADR-0111 (Record), ADR-0151 (evolutions are the declaration), `docs/SCHEMA-COMPATIBILITY.md`                |
| Supersedes | nothing — this is the first decision about who owns the additive-vs-breaking question                       |

## Context

`docs/SCHEMA-COMPATIBILITY.md` states the policy that governs payload-shape evolution: some changes
are additive (old events still decode), others owe an explicit op — `RenameField`, `RemoveField`,
`Transform`, `RenameType`. The repo had three partial answers and none of them can answer the actual
question:

| Existing mechanism | What it can say | What it cannot |
| --- | --- | --- |
| The written policy | the general rule | whether *this* delta is additive |
| cqrs-lint E021/E022 | the op ladder is satisfied | whether the op was *needed* |
| `schema.EventSchema.Fingerprint` | the shape changed | **how** it changed |

So "the shape moved and nobody wrote an op" fails in production replay rather than in CI. That is
the gap this decision fills.

The plan ([`docs/planning/2026-11-02_05-31_SUPERB-catalog-cue-compat-and-unification-pareto-plan.md`](../planning/2026-11-02_05-31_SUPERB-catalog-cue-compat-and-unification-pareto-plan.md))
proposed delegating subtyping to CUE, whose `Value.Subsume` documents precisely this question, and
gated the whole effort on a fidelity spike (G0) with a recorded fallback.

## Decision

**The compatibility oracle is a structural walk over this repository's own schema model**
(`catalog/compat`), not CUE subsumption.

`catalog/compat.Compat(old, new *schema.Schema) Report` walks two `catalog/schema.Schema` trees and
classifies every difference against the policy's own rows, producing `Finding` values that name the
**op** the change owes (`OpAddField`, `OpRemoveField`, `OpRenameField`, `OpTransform`). Exactly one
class blocks: a breaking shape change where the message's declared version did not move
(`KindVersionNotBumped`). A breaking change *with* a version bump passes, because whether the
required op actually exists is `schema.EventSchema`'s job and cqrs-lint E021/E022's enforcement.

The package lives **inside the existing `catalog` module** as `catalog/compat`, not as a new
`catalog/cue` module. The spike removed the only reason to isolate it: there is no CUE dependency to
quarantine, so a new module would have bought eight gate registrations and zero consumer benefit.

## Why not CUE (G0)

`cuelang.org/go v0.17.1`, `encoding/jsonschema.Extract` → `cue.Value.Subsume`, measured over 24
fixtures drawn from the policy's own tables plus the shapes a subtype check typically misses
(closing, format swaps, newly constrained lists, restructuring).

| Configuration | Score | False `compatible` (unsafe) |
| --- | ---: | ---: |
| `Config{}` + none | 21/24 | 3 |
| **`Config{}` + `cue.Raw()`** | **22/24** | **2** |
| `Config{}` + `cue.Final()` | 16/24 | 8 |
| `Config{}` + `Raw`+`Final` | 16/24 | 8 |
| `Config{Strict:true}` + none | 20/24 | 3 |
| `Config{Strict:true}` + `Raw` | 21/24 | 2 (and `int32`/`int64` fail extraction) |

Three findings settle it:

1. **The classification is good but not sound.** 22/24 is genuinely impressive, and it correctly
   handles every additive row including "add an optional field".
2. **Both residual misses are structural, not tuning.** A `format` swap (`date-time` → `uuid`) is
   invisible to CUE because `format` is not a constraint; a newly constrained array `items` is
   invisible because an open list subsumes a constrained one. No option combination closes either.
   Both are policy rows that owe `Transform`, and both are first-class fields (`Format`, `Items`) on
   `catalog/schema.Property`, so the structural walk sees them for free.
3. **The errors point the unsafe way.** A false `breaking` costs a human a review. A false
   `compatible` licenses shipping a replay-breaking change. A gate whose failure mode is "silently
   approve" is worse than no gate, so the tolerance is zero rather than "nearly all".

`cue.Final()` — the option that reads most naturally, since a released schema *is* closed — is the
worst of the six (16/24), converting `remove field`, `rename field` and `add required field` into
"compatible". That is exactly backwards.

Evidence, fixture table, per-config matrix and reopen triggers:
[`docs/evidence/2026-11-02_cue-subsume-spike.md`](../evidence/2026-11-02_cue-subsume-spike.md).

## Consequences

**Positive**

- Dependency-free and v0.x-free: no new module, no eleven indirect dependencies, no 24 MB binary,
  no "bridge is the upgrade tax" class of churn.
- The oracle sees `Format`, `Items`, `Pattern`, `Enum`, `Nullable`, `Deprecated` because it walks the
  model the repo actually renders payloads from, so a policy row cannot be invisible to it.
- Findings name the owed op, so a CI failure reads as an instruction rather than a verdict.
- The same walk powers `CompatCatalog` (catalog-to-catalog), `Conflicts` (two-catalog unification
  detection) and `Validate` (cross-resource invariants), all CUE-free.

**Negative**

- ~900 lines of Go that CUE would have provided for free, and that must be maintained as the schema
  model grows. Mitigation: a 25-case policy table test pins every row, and adding a
  `catalog/schema` field without a policy row is visible as an unclassified difference.
- The classifier is hand-written, so its completeness is an argument rather than a proof. Mitigation:
  the table test is the policy in executable form, and the raise-the-bar rule applies (a new schema
  field must land with its compat classification or the table fails).
- No CUE-authored payload schemas. That was never in scope (ADR-0151 already made evolutions the
  declaration), but this ADR closes the door explicitly.

**Rejected alternatives**

| Alternative | Why rejected |
| --- | --- |
| CUE `Subsume` as oracle | 2 of 24 policy rows invisible, and both fail in the unsafe direction (above) |
| Hand-written CUE schemas fed to `Subsume` directly, bypassing `Extract` | Would dodge the extraction limits but recreate the entire schema model in a second language — a split brain of exactly the kind this repo keeps eliminating |
| Extend `Fingerprint` to report the delta | A fingerprint is a hash; deriving semantics from it means reinventing the walk with less information |
| Leave it to cqrs-lint E021/E022 | Those check that the op ladder is consistent, not whether an op was owed |

**Reopen triggers** (all three required):

1. `encoding/jsonschema` graduates out of v0.x and models `format` and constrained lists as
   comparable constraints.
2. A named consumer asks for CUE-authored payload schemas.
3. `Generate` leaves experimental, so one CUE source could own both directions.
