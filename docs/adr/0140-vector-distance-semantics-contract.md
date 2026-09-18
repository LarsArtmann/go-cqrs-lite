# ADR-0140: Vector Distance-Semantics Contract (One Ranking, Every Engine)

**Date:** 2026-09-16
**Status:** Accepted
**Supersedes:** N/A
**Related:** [ADR-0085](0085-metaengine-new-adts.md) (ADTVector introduction), ADR-0112 (ES-native planner), ADR-0124 (operator-driven layouts), `metaengine` README capability table

## Context

The Vector ADT shipped on every first-party engine on 2026-09-15. Because the
planner may route a collection to any engine the operator picks at deployment
time, the SAME query text must produce the SAME ranking on every backend. But
each engine's native vector support disagrees by default:

- **libSQL/Turso** `vector_distance_cos` is cosine distance (1 − cosSim),
  `vector_distance_l2` is Euclidean, and `vector_distance_dot` returns the
  **NEGATED** dot product (identical vectors → −1) — a sort-ascending trap
  that bit the first implementation.
- **DuckDB** core `array_cosine_distance`, `array_negative_inner_product`,
  and `array_distance` happen to align, but the core `array_*` binders only
  accept fixed `FLOAT[n]`, not `FLOAT[]` lists — queries must cast both
  sides with the dimension formatted into the SQL text.
- **Dgraph** `float32vector` + `similar_to` sorts by its own metric spelling
  ("cosine"/"euclidean"/"dot") and returns uid-only results, so scoring is
  done in Go for exact parity.
- **Go brute-force paths** (memory, sqlite/modernc, mysql, KV engines) use
  whatever loop the author wrote unless pinned to one shared function.

Without a written contract, each new engine re-litigates these semantics and
a near-miss (e.g. double-negating the dot) ships as a silently wrong ranking
— the split-brain class the parity tests exist to kill.

The second decision this ADR records is **degrade-everywhere**: engines
without native ANN still serve k-NN. The alternative — failing vector
queries on engines without an index — would make the planner's
deployment-time engine choice a silent correctness cliff, violating the
"operators pick engines, code doesn't care" vision.

## Decision

### 1. Distance semantics (the contract)

Metric names are the strings `"cosine"`, `"dot"`, `"euclidean"` (`""` and any
unknown metric default to euclidean, matching the ADT's graceful-degradation
stance). For query vector `q` and stored vector `v`:

| Metric      | `metaengine.VectorDistance(q, v, metric)` | Range   |
| ----------- | ----------------------------------------- | ------- |
| `cosine`    | `1 − cosSim(q, v)`                        | [0, 2]  |
| `dot`       | `−(q · v)` (NEGATED dot)                  | (−∞, ∞) |
| `euclidean` | L2 norm of `q − v`                        | [0, ∞)  |

- **Ascending sort is always nearest-first.** `TopKNearest` is the shared
  sort + truncate; the negated dot is what makes that single rule hold for
  all three metrics.
- **`metaengine.VectorDistance` is the single Go implementation.** Engine
  code must not hand-roll a second scorer; brute-force paths call it.
- **Engine-native SQL expressions must reproduce the contract exactly** and
  are pinned by the adttest parity matrix (same fixtures, same expected
  distances, run per engine). Known driver gotchas, verified empirically
  2026-09-15 and pinned in AGENTS.md contract #26:
  - libSQL `vector_distance_dot` ALREADY returns the negated dot — do NOT
    negate again.
  - libSQL `vector_distance_cos` is `1 − cosSim` — no adjustment.
  - DuckDB core `array_*` only bind fixed `FLOAT[n]`: cast both sides with
    the query dimension formatted into the SQL text.
- **Zero vectors** score cosine distance 1 (orthogonal-equivalent), not an
  error — brute-force and SQL paths must agree on this too.

### 2. Degrade-everywhere (no vector capability cliff)

Every first-party engine implements `VectorBackend`. Engines without a
native distance function execute a brute-force scan (`O(N·D)` per query),
scored in Go via `VectorDistance`; engines with one push the scoring into
SQL (libSQL, DuckDB) — same `O(N)` class, engine-side arithmetic. The
profile declares `ADTVector` honestly as degraded/`O(N)` so the planner's
cost model routes accordingly, and missing `VectorCounter` triggers the
ExplainPlan/Doctor full-scan WARN rather than pretending cheap queries.

Native ANN (Dgraph `similar_to`+hnsw, DuckDB VSS, sqlite-vec, pgvector,
MariaDB VECTOR) remains future opt-in work (ROADMAP); this ADR fixes only
the floor: **vector queries degrade, never fail, on any engine.**

### 3. Filtered k-NN (pre-filter, AND)

`VectorSearchFiltered` applies metadata filters BEFORE ranking, so k results
are the k nearest MATCHING neighbors (post-filtering a bare top-k can return
fewer than k while matches exist). Filter ops are the standard `FilterOp`
constants with AND semantics; `VectorMatchesFilters` is the shared predicate
so missing-field behavior is identical across engines.

## Consequences

**Positive**

- Cross-engine parity is mechanical: the adttest matrix runs identical
  vector fixtures per engine, so a new engine cannot ship a divergent
  ranking without a red test.
- Operators can move a collection between engines at deployment time
  without changing result order — the planner's core promise extends to
  the vector ADT.
- One Go scorer and one SQL-expression table per engine; review surface for
  future engines is a fixed checklist (this ADR + AGENTS contract #26).

**Negative / accepted costs**

- The negated dot is unintuitive; the cost is permanent documentation
  weight (this ADR, AGENTS #26, `TopKNearest` doc) in exchange for one sort
  rule everywhere.
- Brute-force `O(N)` is the shipped performance floor until the ANN paths
  land; large collections need an operator-side engine choice, which is the
  deployment-time model anyway.
- `"dot"` returns negative "distances" (identical vectors → −1); callers
  must not assume non-negativity. The contract, not the caller, owns the
  sign.

## Verification

- adttest Vector/VectorFiltered matrix per engine (parity, all engines).
- `TestLibSQLDistanceParity`-style exact-parity tests on the embedded libSQL
  driver (identical distance values Go vs SQL, all three metrics).
- Capability audit asserts profiles declare `ADTVector` on every engine —
  no over- or under-declaration.
