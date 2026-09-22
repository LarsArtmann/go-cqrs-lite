# Direction Ruling — Evidence Pack + 3-Option Decision Memo (G-T01)

> **Status: DELIVERED — the ruling itself remains owner-gated (G-T02).**
> **Date:** 2026-09-21 · **Source:** goal-closure plan §1.1 / G-T01–G-T03
> ([plan](2026-09-17_05-49_SUPERB-metaengine-goal-closure-pareto-plan.md)).
> **Routing:** the ruling lands as **ADR-0147** (re-slotted 2026-09-22: the
> SingleWriter lease one-pager claimed ADR-0146 first; the ADR-0141
> the plan named was taken by temporal versioned cells) plus an AGENTS.md Goal
> sentence amendment. Ruling session = XS from here: §6 states what is drafted
> for each outcome.
> **Contract honored:** no deprecated machinery is revived by this memo; it
> only prices the options (plan anti-Verschlimmbesserung clause).

## 1. The question being ruled

The Goal (AGENTS.md, verbatim): _"Developers declare ONLY Commands + Events +
Queries and their relationships. We should be able to build superb projections
(materialized views) and developers never need to worry about anything else,
while where data lives is up to operators at DEPLOYMENT time."_

Since 2026-09-16 the only mechanism that literally satisfied "ONLY" — runtime
`Infer`/`InferFromNamedEvents` — is **Deprecated, removal at v5**
(`metaengine/fold_inference.go:58-62`: "hides projection semantics behind
naming conventions and reflection"; `metaengine/infer_named.go:29-33`). The
Goal's signature claim is therefore currently expressed by machinery the
library itself tells consumers not to use. Until the owner rules what
"declare ONLY" means instead, G1 is undefinable and Gate A stays blocked.

## 2. Evidence

### R01 — What `Infer` covered (the deprecated surface)

| Capability                | Mechanism                                                                          | Evidence                              |
| ------------------------- | ---------------------------------------------------------------------------------- | ------------------------------------- |
| Convention classification | Go struct NAME suffix `*Created`/`*Updated`/`*Deleted` → insert/update/remove fold | `metaengine/fold_inference.go:96-100` |
| Row-materializing folds   | Field-name matching, nested struct flattening                                      | `metaengine/fold_inference.go:40-42`  |
| Key detection             | Single non-pagination Q field's Go type, fallback `"ID"`                           | `metaengine/fold_inference.go:33-35`  |
| Filter inference          | Q fields beyond key matching R fields → `FilterOnField`                            | `metaengine/infer_filters.go`         |
| Sort inference            | Temporal-field detection (`CreatedAt`, …), descending                              | `metaengine/infer_sort.go`            |
| Composite keys            | Multi-field key detection                                                          | `metaengine/infer_composite.go`       |
| Wire-event variant        | `InferFromNamedEvents` pairs dot-separated wire types with struct samples          | `metaengine/infer_named.go:43`        |
| Evaluation point          | Per-`QueryDecl`, at `Plan()` time, via reflection                                  | `metaengine/fold_inference.go:10-16`  |

What it never covered: counter/delta arithmetic, graph/traversal shapes,
anything ambiguous (hard errors instead). In-repo callers today: **zero
production/example call sites** — only the two definition files, `override.go`
(Layer-2 override plumbing), and tests.

**Why it died:** the fold mapping is invisible at the call site; the docs and
ADR-0116 steer every production model to explicit folds
(`fold_inference.go:23-31`, ADR-0116 §Implementation-Status recommendation
block). Matching by struct name while the rest of the library speaks
dot-separated wire types added a second mismatch.

### R02 — What the `system` surface covers today (the sanctioned surface)

`DomainConfig.Evolutions` declares, once per RESULT type, how that type
materializes from events; projections inherit the folds by result type
(`system/config_types.go:44-47`, `system/evolutions.go:83-140`,
`system/query_constructors.go:80-99`, `:212-231`; inheritance audit:
[2026-09-21 coverage audit](2026-09-21_evolution-fold-inheritance-coverage-audit.md)).

| Capability                       | `Infer` (dead)                        | Evolution path (live)                                                                                                                                                                  |
| -------------------------------- | ------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CRUD row folds from convention   | struct-name suffix, per-query samples | same suffix classes via `AutoCRUDByNamedEvents` (`metaengine/auto_named_events.go:51-130`), **keyed by wire event type declared at the DomainConfig** (`system/evolutions.go:104-140`) |
| Tombstone removal                | `*Deleted` suffix → remove            | same, type-driven Remove per ADR-0114 (`metaengine/auto_naming.go:104-119`), pinned by tests (`system/evolution_tombstone_test.go`)                                                    |
| Rebirth after delete             | upsert semantics                      | structural: insert fold is a fresh upsert (`metaengine/auto_naming.go:36-49`)                                                                                                          |
| Filters / sorting                | auto-inferred, invisible              | consumer opts per declaration: `Filterable`/`Sortable` (visible)                                                                                                                       |
| Partial/missing coverage         | silent                                | **warn-first guard** naming projection + missing types (G-T10, `system/evolutions.go:234+`)                                                                                            |
| Event-universe consistency       | none                                  | `DomainConfig.Events` coeffect gate — hard error on undeclared consumption (`system/config_types.go:49-57`)                                                                            |
| Escape hatch for the 20%         | `Override(...)` wrappers              | explicit fold closures on the Evolution (`.On(wireType, sample, closure)`, `system/evolutions.go:144-160`)                                                                             |
| Non-row shapes (Count, RawQuery) | not covered                           | declared directly — `Count(...).On(...)` IS the declaration (audit §4 gap B: by design)                                                                                                |

2026-09-21 hardening closed the audit's top gap (ghost-row warning), pinned
tombstone+rebirth through the inherited path, and shipped
`Store.BackfillPlannedTables` (register-after-data). Audit §6 verdict: the
remaining gaps are "warnings-and-docs sized, not architecture sized", and
Count is the one shape least expressible by ANY convention (deltas carry
intent).

### R03 — Consumer declared shapes

- **`example/goal-shaped-app`** (flagship, in-repo): zero fold closures; ONE
  Evolution (Created/Updated/Deleted convention samples); `Lookup`+`QuerySet`
  inherit by result type; tombstone e2e asserted through the inherited path
  (`example/goal-shaped-app/app.go:87-111`, `domain.go:24-56`,
  `main_test.go`). This app already lives entirely inside the sanctioned
  surface — it needs no ruling to keep compiling.
- **CV** (external consumer, most rigorous evaluator): runs released tags
  (system v4.7.0) and hand-rolls what master already does (excellence plan
  context); its evaluation asked whether metaengine's Vector+Graph ADTs can
  replace its hand-rolled SQLite store
  ([feedback](../feedback/reviewed/archived/2026-09-15_go-graph-rag_metaengine-system-evaluation-feedback.md):276),
  and it hit the Scan-100 silent truncation (G-T14 survey census). Its shape
  class (funnel/graph/vector) sits OUTSIDE row-materialization convention —
  it would use explicit declarations under every option. CV verdict on
  parity risk: "fold-as-read-model is our own pattern" (plan §6 risk table) —
  an acceptable honest outcome if the benchmark says so.
- **No consumer evidence demands runtime `Infer` back**: zero in-repo
  production call sites; the deprecation decision (2026-09-16) documented the
  same steer in FAQ, modules.md, and FEATURES.

## 3. The options

### (a) REFRAME — "declare Evolutions + Queries once; system wires projections"

The Goal's "ONLY" means: declare Commands, Events, Queries, their
relationships, and ONE fold declaration per read model (the Evolution); the
system builds every projection and the planner routes. `Infer` stays dead;
ADR-0116's Layer-1 section gets a status addendum (runtime-reflection form
retired; the Evolution-convention declaration is the sanctioned Layer 1).

- Cost: **XS** — ADR + AGENTS sentence + doc pass. Gate A unblocks now.
- Gain: the Goal becomes true on shipped, hardened, warn-loud surface; the
  declaration is _more_ auditable than `Infer` ever was (wire types listed,
  coeffect-gated, Doctor-visible, lint-covered).
- Risk: "ONLY" gains one keyword; readers dreaming of literal zero-fold
  declarations must accept that the Evolution IS the declaration. Counters
  are explicitly declared, never inferred.

### (b) REVIVE — re-commit Layer-1 inference, compile-time and visible

`cqrs-gen` (or `go:generate`) generates fold declarations from type
inspection; output is diffable, committed, tested; runtime `Infer` stays
dead so the auditability objection is answered by construction.

- Cost: **M+** — `cqrs-gen` today only emits typed handler-registration
  stubs from `//cqrs:` marker comments (`cmd/cqrs-gen/main.go:1-14`); fold
  generation (AST → fold-spec → rendered declaration) does not exist.
  ADR-0116 explicitly REJECTED codegen as the sole path ("complementary, not
  the primary mechanism", ADR-0116 §Alternatives-B). Gate A stays blocked for
  the design + build + proof window.
- Gain: literal zero-declaration CRUD views return, with visible output.
- Risk: builds machinery no current consumer asked for (R03); risks
  re-importing the convenience-over-clarity drift the deprecation just
  closed; competes with G-T04/G-T05/G-T16 (AggregateOn, routing, parity
  benchmark) for the same attention.

### (c) HYBRID — Reframe NOW, codegen as evidence-gated opt-in later

Rule (a) immediately: AGENTS sentence + ADR-0147 (see routing note) + ADR-0116 addendum. Park a
ONE-PAGER (G-T03, doc only, no code) describing the `cqrs-gen` fold-generation
path as an opt-in for CRUD-shaped views, to be picked up only if the G-T16
parity benchmark or consumer evidence shows the Evolution declaration is a
real adoption blocker.

- Cost: **S** — everything in (a) plus a design one-pager.
- Gain: Gate A unblocks now; the revival door stays open WITHOUT committing
  the M-effort or blocking the release train.
- Risk: a parked one-pager can rot — mitigated by tying its activation to a
  named evidence gate (G-T16 numbers or a named consumer ask).

## 4. Recommendation

**(c) HYBRID** — with the reframe leg executed immediately and unconditionally.

Reasoning from the evidence: the sanctioned surface already delivers
declare-once-wire-everything for every row-materializing shape, hardened
warn-first as of 2026-09-21 (R02); the flagship example needs zero fold
closures today (R03); the only non-inheritable shape (Count) is also the
least convention-expressible in principle, so no codegen path would remove
its declaration either (audit §6); and no consumer — including CV — is
blocked by the Evolution keyword (R03). Revive-as-primary would spend M+
effort re-solving a problem the surface already solved, against the plan's
own warning that casual revival of deprecated machinery IS the regression
this plan guards against. If the owner prefers maximal minimalism, pure (a)
is defensible by dropping the one-pager; nothing in this memo makes (b)
standalone defensible today.

## 5. AGENTS.md Goal sentence — proposed amendment (pending ruling)

Current: _"Developers declare ONLY Commands + Events + Queries and their
relationships. …"_

Proposed (reframe/hybrid meaning):

> _"Developers declare ONLY Commands + Events + Queries, their relationships,
> and one Evolution per read model (the fold declaration). We should be able
> to build superb projections (materialized views) — wired and routed by the
> system — and developers never need to worry about anything else, while where
> data lives is up to operators at DEPLOYMENT time."_

(Ruling may redline; the operative delta is the Evolutions clause + "wired and
routed by the system".)

## 6. What happens mechanically under each ruling (XS session script)

| Step                                                         | REFRAME                 | HYBRID (recommended)                  | REVIVE                                  |
| ------------------------------------------------------------ | ----------------------- | ------------------------------------- | --------------------------------------- |
| ADR-0147 draft (decision + consequences + ADR-0116 addendum) | ✓                       | ✓                                     | ✓                                       |
| AGENTS.md Goal sentence amended                              | ✓                       | ✓                                     | ✗ (sentence stays, meaning re-expanded) |
| G-T03 `cqrs-gen` fold-codegen one-pager                      | doc note: declined, why | ✓ parked one-pager w/ activation gate | ✓ promoted to build plan                |
| TODO_LIST 🔥 row closed                                      | ✓                       | ✓                                     | partially (build rows open)             |
| CHANGELOG [Unreleased] entry (Goal story)                    | ✓                       | ✓                                     | at build time                           |

Shared after any ruling: TODO_LIST G-T14 scan-default decision recorded the
same session (survey recommendation: flip to unbounded at the v5 cut);
v5-deprecation list gains nothing new (`Infer` already listed).

## 7. Evidence index (all verified 2026-09-21, master `9efeb426e`+)

`fold_inference.go:58-62` (deprecation) · `fold_inference.go:23-31` (steer) ·
`infer_named.go:29-33` · `auto_named_events.go:51-130` · `auto_naming.go:36-49`,
`:104-119` · `system/config_types.go:44-57` · `system/evolutions.go:104-160`,
`:234+` · `system/query_constructors.go:80-99`, `:206-231` ·
`system/evolution_tombstone_test.go` ·
[inheritance audit](2026-09-21_evolution-fold-inheritance-coverage-audit.md)
§2/§4/§6 · [scan survey](2026-09-21_scan-default-v5-survey.md) ·
[ADR-0116](../adr/0116-layered-auto-projection.md) §Layer-1 + §Alternatives-B ·
[ADR-0114](../adr/0114-tombstone-as-domain-event.md) ·
[CV feedback](../feedback/reviewed/archived/2026-09-15_go-graph-rag_metaengine-system-evaluation-feedback.md):276 ·
`cmd/cqrs-gen/main.go:1-14` · `example/goal-shaped-app/app.go:87-111`
