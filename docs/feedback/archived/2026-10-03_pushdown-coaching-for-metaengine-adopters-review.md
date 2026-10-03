# Review: Pushdown Coaching for Metaengine Adopters

**Source proposal:** [`2026-10-03_pushdown-coaching-for-metaengine-adopters.md`](2026-10-03_pushdown-coaching-for-metaengine-adopters.md)
**Date reviewed:** 2026-10-03
**Outcome:** Core proposal ACCEPTED and implemented (F022/F023 utilization
split with Volume gate, actionable profile line, memory-engine framing). One
premise correction, two deferrals.

---

## Premise verification

- **"F022-F025 `continue` on metaengine importers"** — CONFIRMED at
  `adoption/f022.go` and `adoption/f023_f024_f025.go` (three sites).
- **"The analyzer's query registry already knows R per Query declaration"** —
  **FALSE** (the verify-before-acting catch): no registry of
  `metaengine.Query` declarations existed; only the boolean
  `MetaenginePushdown` (any FilterOnField/SortOnField call anywhere). The
  registry is therefore NEW work delivered with this change:
  `CQRSRegistry.MetaengineQueries` (`QueryDeclInfo`: collection, R type name,
  Volume literal, declarative flags, position), scanned by
  `scanMetaengineQueryDecl` (generic-instantiation calls are IndexListExpr —
  invisible to the SelectorExpr-based call scanner, hence the dedicated walk).
- Metaengine API claims verified: `FilterOnField[R](field, op)` /
  `SortOnField[R](field, desc)` (metaengine/query_config.go:183,196) and
  read-time `WithFilter`/`WithLimit` (metaengine/scan_options.go:31,97). The
  proposal's `typed_reader.go:15` citation for WithFilter is slightly off
  (scan_options.go:31); signatures otherwise as claimed.
- **Bonus finding:** the pre-existing non-importer F023 suggestion showed the
  exact API lie the proposal warns about — `FilterOnField[R]("column",
  FilterEq, value)` (three args with a VALUE). Fixed alongside: the suggestion
  now shows the two-layer shape.

## Fix plan triage

### 1. Split adoption vs utilization — IMPLEMENTED (F022, F023)

Importers no longer `continue`: `pushdown_utilization.go` links manual
sort sites (`sort.Slice`/`slices.Sort{,Stable}Func`) and range loops with
field comparisons to a registered Query's R type via REAL type information
(`packages.TypesInfo`), firing when the declaration lacks `SortOnField`/
`FilterOnField`. F023 names the compared fields (sorted, deduplicated) and
both suggestions embed the two-layer shape plus the runtime-data warning.

Design deviations from the sketch, both conservative:

- **R attribution is exactly-one:** when several Query declarations share one
  R type, the site cannot be attributed to a collection, so coaching stays
  silent — a missed hint, never a wrong-collection finding. Consumers with
  distinct view types per collection (the reported shape) get full coaching.
- **F024/F025 utilization DEFERRED:** pagination-window and manual-count
  detection over R-typed collections needs dataflow (which reader produced
  the slice) to avoid false positives; the proposal's own acceptance criteria
  only exercise F023. Tracked in TODO_LIST.

### 2. Volume as the scale signal — IMPLEMENTED

`pushdownVolumeThreshold = 1000`. The Volume literal is read from the
declaration (underscore separators accepted); an absent or unresolvable Volume
(constant/variable reference) keeps the project noise-free — matching the
"below ~1k rows Go-side filtering is legitimately fine" contract and the
acceptance criterion "without firing on the ~100 row rooms".

### 3. Actionable profile line — IMPLEMENTED

```
pushdown:    false (3 queries, 0 declarative — declare FilterOnField/SortOnField; free on the memory engine)
pushdown:    true (1/3 queries declarative)
```

New profile fields `metaengineQueryCount`/`metaengineDeclarativeQueries`
(doctor JSON golden regenerated). The bare `%t` remains only when no Query
declarations exist.

### 4. Memory-engine honesty — IMPLEMENTED

Every utilization suggestion ends with: "Declare it even on the memory
engine — the planner ignores pushdown there, so the declaration is free and
goes live the moment a SQL DSN is configured." Same framing in the profile
line.

### 5. Cookbook recipe — DEFERRED

"Legacy grid → read model + pushdown" belongs in the skill references /
COOKBOOK, not the linter. Tracked in TODO_LIST.

## Honest limits — confirmed and honored

1. **Scope blindness interacts:** utilization links only filter/sort sites in
   importer packages. The reporter's `internal/server` grid stays invisible
   until load-scope widens (see companion review, scope item).
2. **Filter values are runtime data:** both suggestion texts show the
   declaration-vs-read two-layer shape and say values never go in the
   declaration.

## Acceptance criteria

The nsfw-classifier tree is external and cannot be executed from this repo;
the equivalent shapes are pinned by the committed typed fixture
`cmd/cqrs-lint/testdata/scanfixture` (a real replace-based consumer module,
same harness pattern as typedfixture):

- Point-Get reader + bufio loop: no F022/F023/F026/F031 findings (nothing
  coachable — the "grid is legacy-memory" state stays silent).
- Volume(100_000) query + Go-side threshold/room filtering → exactly one F023
  naming `RoomID`/`Score` with the two-layer suggestion.
- Volume(100) query and a declarative query with identical filtering → no
  findings (the Volume gate and the declarative escape hold).
- `slices.SortFunc` over the large query's rows → exactly one F022 with
  `SortOnField[itemView]` in the suggestion.
