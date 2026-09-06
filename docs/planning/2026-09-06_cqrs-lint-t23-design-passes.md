# cqrs-lint Design Passes (T23) — F089/F090/F091

**Date:** 2026-09-06 · **Status:** DESIGN (not implemented unless noted)
Decision gates ⛔Q2 resolved to plan defaults (V007 stays `warning`); these
designs show the opt-in mechanisms consumers can use instead of forcing the
default.

---

## F089 — `v5-ready` preset (severity-escalation mechanism)

**Problem.** V007 fires at `warning`, which reports but does not block CI.
Consumers whose v5 migration is complete (or who want a hard deadline)
need an opt-in escalation, without changing the default for everyone else.

**Current mechanics.** `PresetDefinitions` (analyzer/feature_profile.go) is
the single source of truth consumed by both `init` (config generation) and
the runtime (feature + rule resolution). `RulesConfig` supports
`disable` only; severity comes from the catalog, mutated at runtime by
`applyDomainBias` (financial-domain escalation).

**Design.**

1. Add `rules.severity-overrides` (`map[ruleID]severity`) to `RulesConfig`,
   validated against catalog IDs (unknown IDs warn, same as unknown keys
   today) and rejected for rules that don't exist in the catalog.
2. Add preset `v5-ready`:
   ```go
   PresetV5Ready: {
       Rules: RulesConfig{
           SeverityOverrides: map[string]string{"V007": "error"},
       },
   },
   ```
   Escalation flows through the same choke point as `applyDomainBias`
   (post-detection, pre-filter), so triage/fix behavior is unchanged and
   `--min-severity` keeps working (`error` findings always pass it).
3. Precedence, fixed and documented: catalog → preset override →
   config-file override → domain bias → CLI `--min-severity` (filter only,
   never rewrites severity). Later stages win; nothing may lower a
   severity below the catalog value except an explicit config override.
4. `init` gains `--preset v5-ready`; README preset table gains a row.

**Cost.** ~1 day incl. tests. Risk: low — the override map is additive;
the choke point already exists.

**Non-goal.** Per-rule confidence overrides; escalations that silently
re-enable disabled rules.

---

## F090 — Dot-import detection (V007 scope extension)

**Problem.** V007 matches selectors (`pkg.Symbol`) and resolves qualifiers
through import declarations. A dot-import
(`import . "github.com/larsartmann/go-cqrs-lite/storage/v4"`) leaves
removed-API calls qualifier-less, so V007 cannot attribute them — a real
hole during a v5 migration, exactly when consumers scan for these calls.

**Options.**

- **(a) Flag the dot-import itself (recommended now).** When a
  go-cqrs-lite import path appears with `ast.ImportSpec.Name == "."`, emit
  a V007 finding at the import position: "dot-import of a go-cqrs-lite
  module hides v5-removed-API usage from this linter; name the import".
  False-positive risk ≈ 0 (dot-importing library internals is rare and
  already discouraged); implementation is a small walk of `GoFiles`
  import specs, no new machinery; the message teaches the fix.
- **(b) Type-based attribution (deferred to F091).** Resolve dot-imported
  idents via `packages.TypesInfo` and fire V007 only on actual removed
  symbols. More precise, but needs the typed-info integration (below) and
  `NeedTypes` load mode — do it once, not twice.

**Decision.** Ship (a) as part of V007; revisit (b) after F091 lands.
The existing V007 drift meta-tests extend naturally (a fixture with a
dot-import must fire).

---

## F091 — Typed-info integration for name-heuristic rules

**Problem.** C008 (money in float64) and C035 decide from NAMES (field,
struct, package). Names are the right first-order signal but misfire on
unrelated domains (metrics, rates) and miss euphemisms; the current
strong/weak + corroboration tables are curated guesswork.

**Design — incremental, three tiers.**

1. **Tier 1: type resolution for qualifier rules (cheapest, widest).**
   Load with `packages.NeedTypes|NeedTypesInfo` and resolve every
   selector's referenced object through the import table. Kills the whole
   alias-blindness class (A014's original bug) by construction and makes
   F090(b) possible. Measure first: `NeedTypes` roughly doubles load cost;
   the F044 benchmark (V007 sub-noise) shows headroom, but the budget
   gate is "no visible wall-time regression on the repo-root corpus".
2. **Tier 2: usage-confirmation for C008.** Escalate a weak-signal field
   to a finding only when type info shows the struct instance flows into
   an event payload position (`event.New...` argument, `Payload()`
   producer, CBOR/JSON marshal of the struct). Weak-signal fields with no
   payload flow stay silent — this replaces the corroboration tables with
   evidence. Strong signals keep firing without confirmation (cheap,
   catches the obvious cases before Tier-2 analysis runs).
3. **Tier 3: payload-shape confirmation for C035/C013-class name rules.**
   Same pattern: name selects candidates, type info confirms or rejects
   via structural usage (field read in a fold/decide body, serialization
   reflection points).

**Failure modes to design against.** Type info is unavailable on
non-compiling projects (the linter's "could not analyze" path) — name
heuristics MUST remain the fallback so partial results survive broken
builds. Workspace consumers (go.work with many modules) inflate type-check
cost — measure per-corpus, not per-file.

**Sequencing.** Tier 1 is the prerequisite for both other tiers and
independently valuable; ship it alone first behind no flag (load mode
change only), then Tier 2 behind `--typed-info=auto` (default on when
type load succeeded, off on fallback).
