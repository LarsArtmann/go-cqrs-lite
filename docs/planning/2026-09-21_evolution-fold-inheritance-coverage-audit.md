# Evolution-Fold-Inheritance Coverage Audit — G-T09

**Date:** 2026-09-21
**Source task:** [`TODO_LIST.md` § metaengine Goal-closure follow-ups](../../TODO_LIST.md) G-T09 (plan ref: [`2026-09-17_05-49_SUPERB-metaengine-goal-closure-pareto-plan.md`](2026-09-17_05-49_SUPERB-metaengine-goal-closure-pareto-plan.md), R19–R21).
**Scope:** enumerate every `system` projection declaration shape, whether it inherits folds from a matching `Evolution` today, and where the gaps are. Evidence is `file:line` against master (`9efeb426e`).

## 1. The inheritance mechanism (verified)

`DomainConfig.Evolutions` declares, per RESULT type, how that type materializes
from events (`system/evolutions.go:83-140`). `system.New` indexes evolutions by
result type (`system/projection_builder.go:75-85`); each projection
declaration's build closure looks itself up in that index and, on a hit, reuses
the evolution's folds instead of requiring its own `.On(...)` samples
(`system/query_constructors.go:80-99`, `:212-231`).

Fold generation from an evolution (`system/evolutions.go:211-228`):

- **Convention samples** (`.On` with zero fold closures) →
  `metaengine.AutoCRUDByNamedEvents[R]` (`metaengine/auto_named_events.go:51-130`):
  `*Created` → insert fold, `*Updated` → update fold, `*Deleted` → **remove fold**
  (`metaengine/auto_naming.go:104-119` — the ADR-0114 type-driven tombstone
  Remove), matched by wire event-type string.
- **Explicit folds** (`.On` with a closure) → `makeExplicitFold`
  (`system/evolutions.go:144-160`), typed mutation with reify.
- A `*Created` sample is mandatory; `*Deleted`/`*Updated` optional
  (`metaengine/auto_named_events.go:102-106`).

## 2. Shape × inheritance matrix

| # | Declaration shape | Constructor | Result type | Inherits evolution folds | Evidence |
|---|-------------------|-------------|-------------|--------------------------|----------|
| 1 | Point lookup | `Lookup[R](name)` | `R` | **YES** | `system/query_constructors.go:80-99` |
| 2 | Filtered/sorted collection | `QuerySet[R](name)` (+`Filterable`/`Sortable`/`Priority`) | `R` | **YES** (filter/sort opts compose with inherited folds) | `system/query_constructors.go:212-231` |
| 3 | Counter aggregate | `Count(name).On(type, sample, delta, key)` | `map[string]int64` | **NO — by design** (delta arithmetic, not row materialization; build closure ignores the evolution index) | `system/query_constructors.go:297-306` |
| 4 | Raw passthrough | `RawQuery(decl)` | any | **NO — by design** (opaque QueryDecl; also opaque to the coeffect gate) | `system/projection_builder.go:43-54`, `:111-112` |
| 5 | Runtime scan over a registered projection | `Find[R](ctx, sys, name, ...)` | `[]R` | reads the projection's existing folds (inherits whatever its declaration wired) | `system/runtime.go:111-140` |

Runtime readers that consume inherited folds (no new wiring): `system.Get[R]`
(point), `system.GetCount` (counter), `system.Find[R]` (scan; note: inherits
the documented-100 default cap when no `Limit` is passed — G-T14's consumer
census already covers this, `system/runtime.go:124-129`).

## 3. What the tombstone auto-fold already covers (G-T11 ground truth)

- `*Deleted` convention samples produce `removeFold` — a type-driven
  `MapDelete`-class removal, i.e. ADR-0114's tombstone-as-domain-event mapping —
  inside `AutoCRUDByNamedEvents` (`metaengine/auto_named_events.go:122-127`).
  The deprecated metadata path (`event.TombstoneMark`) is NOT involved.
- **Rebirth** is structural: `insertFold` is a fresh upsert
  (`metaengine/auto_naming.go:36-49`, `reflect.New(resultType)`), so a
  `*Created` after a `*Deleted` re-materializes the row. No code blocks this;
  it is pinned only implicitly today.
- End-to-end proof exists in the flagship example: `example/goal-shaped-app`
  declares ONE evolution (Created/Updated/Deleted convention), Lookup+QuerySet
  inherit by result type, and its sqlite e2e test asserts the tombstone
  (`awaitGone`) through the INHERITED path on a planned collection
  (`example/goal-shaped-app/main_test.go`).

Remaining G-T11 delta (done this session): an explicit system-level test pair —
tombstone AND rebirth through the inherited-fold path — plus a CHANGELOG entry
stating the semantics, so the behavior is pinned against regression rather than
demonstrated only by the example.

## 4. Gap table (severity-ordered)

| Gap | Severity | What happens today | Disposition |
|-----|----------|--------------------|-------------|
| **A. Partial-sample projection drops evolution events silently** | **High** | A projection with its OWN `.On` samples wins outright (`query_constructors.go:74-78`, `:206-210`); if its samples omit an event the matching Evolution declares — typically the `*Deleted` tombstone — deleted entities persist as ghost rows in that projection. No warning anywhere. | **Close warn-first (G-T10, this session):** when a projection carries its own samples and a matching evolution exists, compare event-type sets; on a missing tombstone-class (or any missing) event type, emit a Doctor-visible warning naming the projection, the missing types, and the ghost-row consequence. Warning only — behavior unchanged. |
| **B. Count/RawQuery never inherit** | Low (by design) | Counters are delta arithmetic; RawQuery is deliberately opaque. Making them inherit would need new evolution shapes (counter evolutions) — no consumer demand evidenced. | Document in this audit + core.md note. No code. |
| **C. Duplicate result-type evolutions: last-wins silently** | Low | `evoIndex[rt] = es` overwrites (`projection_builder.go:84`); two Evolutions for the same result type = second silently replaces first for every inheriting projection. | Warn-first candidate; parked (no known consumer shape hits it; warn noise risk for legitimate Internal evolutions until enforcement lands). Tracked here for the ruling's evidence pack. |
| **D. `Internal()` marker unenforced** | Low | Reserved v5 (documented at `system/evolutions.go:46-52`). Lookup/Find may still inherit internal evolutions. | Already documented in-code; skip. |

## 5. Enumeration basis (R19)

- All exported `system` constructors (`grep '^func [A-Z]' system/*.go`): the
  five shapes above are the complete projection-declaration surface; everything
  else is command/query/decider registration or config.
- Test suite: `system/system_auto_projection_test.go` (memory engine e2e +
  backward-compat samples path), `system/system_projection_test.go`,
  `system/evolutions_test.go`, `system/evolutions_options_test.go`.
- Consumer shapes: `example/goal-shaped-app` (Evolutions + Lookup + QuerySet —
  the full-inheritance path), skill recipes (`references/core.md` § The Goal in
  5 minutes; `recipes.md` §2.39). CV (external consumer) declared shape is not
  re-censused here; the G-T14 scan survey (2026-09-21) already recorded CV's
  usage class for the Find path.

## 6. Consequences for the Direction Ruling (feeds G-T01)

- The non-deprecated `system` surface ALREADY delivers "declare once, wire
  everything" for Lookup+QuerySet — the two shapes the flagship example and the
  skill's Goal story use. The gaps are warnings-and-docs sized, not
  architecture sized.
- Count-inheritance is the only *shape* the inheritance model cannot express
  today; it is also the shape least likely to be expressible by convention at
  all (deltas carry intent). Any ruling document should scope "declare ONLY" to
  row-materializing projections and state the counter story explicitly
  (`Count(...).On(...)` IS the declaration).
