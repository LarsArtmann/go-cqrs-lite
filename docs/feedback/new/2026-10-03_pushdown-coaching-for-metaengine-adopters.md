# Fix Proposal: Pushdown Coaching for Metaengine Adopters

**From:** nsfw-classifier remediation sessions (2026-10-03)
**Perspective:** AI agent operating cqrs-lint on a production `system/v4` +
`metaengine/v4` consumer (sqlite engine, three domains, ~100k-row ceiling)
**Tone:** Honest, direct; every claim verified against cqrs-lint and
metaengine source before writing

**Related:** `2026-10-03_nsfw-classifier-cqrs-lint-feedback.md` (same
directory) — this file expands one item into a concrete fix plan.

---

## The gap in one sentence

The entire pushdown coaching family (F022 sort, F023 filter, F024
pagination, F025 count) treats "imports metaengine" as *adopted* and skips
the project — but adopting metaengine and adopting **pushdown** are
different steps, and the second one is never coached.

Evidence (read from source):

- `adoption/f022.go`: `if importsPathIn(sc.files, "go-cqrs-lite/metaengine") { continue }`
- `adoption/f023_f024_f025.go:29,76,123`: the same skip in all three rules.

Consequence, from a real consumer: nsfw-classifier declares three
`metaengine.Query` declarations, does Go-side filtering/sorting over
room-item rows (threshold badges, status filters, full-grid reads), has
`store: sqlite` — and `pushdown: false` in the profile. No rule ever fired,
because we import metaengine. The profile line is accurate but not
actionable, and the coaching population that needs it most (metaengine
users at scale) is the one excluded.

## Why this matters at scale (the consumer case)

Our room grid currently re-reads full state and re-derives NSFW/Safe badges
from scores in Go (`GridData.Threshold`). At hundreds of rows this is fine —
which is why we did nothing. At library scale (our own `Volume(100_000)`
declaration) the correct shape is engine-side filtering:

```go
// Declaration — allow-list the pushable columns (verified signatures:
// metaengine/query_config.go:183,196):
metaengine.Query[itemsScan, itemView](
    roomItemsCollection,
    metaengine.Volume(estimatedRoomItems),
    metaengine.FilterOnField[itemView]("room_id", metaengine.FilterEq),
    metaengine.FilterOnField[itemView]("score",  metaengine.FilterGe),
    metaengine.SortOnField[itemView]("item_id", false),
    // ...folds
)

// Read — bind values per call (verified: metaengine/typed_reader.go:15,
// metaengine/explain.go:36):
rows, err := reader.Scan(ctx,
    metaengine.WithFilter("room_id", metaengine.FilterEq, roomID),
    metaengine.WithFilter("score",  metaengine.FilterGe, threshold),
    metaengine.WithLimit(pageSize),
)
```

(Illustrative sketch — the point is the two-layer API shape: declaration
allow-lists columns, read-time options bind values.)

## Fix plan

### 1. Split each rule's trigger: adoption vs utilization

Keep the current behavior for non-importers (coach metaengine adoption).
For importers, don't `continue` — switch to a utilization check:

- **F022 (sort):** fires when a `sort.Slice`/`slices.SortFunc` call site
  operates on a slice whose element type is a registered Query's `R` type,
  and that Query declaration lacks `SortOnField`. Suggestion:
  `SortOnField[R](...)` + read-time sort binding.
- **F023 (filter):** fires when a `range` loop over Query-`R`-typed values
  contains comparisons (`if row.Score >= x`, `row.Status == y`) on fields
  that exist in `R`, and the Query lacks `FilterOnField` for those fields.
  The compared field names ARE the suggested columns — the finding can name
  them (`FilterOnField[itemView]("score", FilterGe)`).
- **F024/F025 (pagination/count):** same pattern for manual `len()` counts
  and slice-window pagination over `R`-typed collections.

The analyzer's query registry already knows `R` per Query declaration; the
new work is linking `R` to filter/sort call sites cross-package (type
identity, not file locality).

### 2. Use `Volume` as the scale signal

The declaration already carries `metaengine.Volume(n)` (consumers set it for
the planner). Read the literal `n` in the analyzer and gate utilization
coaching on it: below ~1k rows, Go-side filtering is legitimately fine —
coach only when Volume is large or the collection is filtered in a hot path
(SSE re-broadcast, HTTP handler). This keeps small consumers noise-free,
which is presumably why the skip existed.

### 3. Make the profile line actionable

`pushdown: false` should explain itself, doctor-style:

```
pushdown:    false (3 queries, 0 filterable — nothing to push down)
pushdown:    false (query room_items: score/room_id filtered in Go — F023)
```

### 4. Memory-engine honesty in the suggestion text

Pushdown pays only when a SQL engine serves the collection. Suggestion text
should say: declare it anyway — the planner ignores pushdown on the memory
engine, so the declaration is free and becomes live the moment a SQL DSN is
configured. (Consumers running memory-by-default, like us, need this
framing or they will defer the declaration as pointless.)

### 5. Cookbook pattern: legacy grid → read model + pushdown

A common consumer shape (ours) serves collections from legacy in-memory
snapshots while the metaengine read model sits idle. A COOKBOOK recipe for
"migrate a full-grid SSE surface onto the read model with pushdown +
pagination (browser-cached thumbnails make full-grid swaps cheap until they
aren't)" would give the F0xx findings a landing doc.

## Honest limits (known before you start)

1. **Cross-package scope blindness interacts with this.** Our Go-side
   filtering lives in `internal/server`, which doesn't import go-cqrs-lite
   and is invisible to the analyzer (see the scope item in the companion
   feedback file). Fix 1's type-linking helps only for filter sites in
   importer packages; the full win needs scope widening or the
   `R`-type-link to cross package boundaries via the registry, not file
   locality.
2. **Filter values are runtime data** — the analyzer can propose columns
   and ops from comparison expressions, but not invent the read-time
   `WithFilter` call; the suggestion must show the two-layer shape or
   consumers will try to put values in the declaration and fail.

## Acceptance criteria

On nsfw-classifier's current tree, after the fix:

- `pushdown: false` remains correct (our readers are point-`Get`; grid is
  legacy-memory — nothing coachable *yet*, and the profile line should say
  so per Fix 3).
- If the grid moves onto `roomItemsQuery` reads with Go-side threshold
  filtering, F023 fires naming `score`/`room_id` with the two-layer
  suggestion — without any config change, and without firing on the ~100
  row rooms we serve today below the Volume gate.
