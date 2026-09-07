# ADR-0135: Materialized Views as an Operator Option (Turso IVM)

- Status: Accepted
- Date: 2026-09-07
- Deciders: Lars Artmann
- Related: ADR-0046 (seven-tier model), ADR-0124 (layout priorities),
  ADR-0133 (ReadAggregate cost model), metaengine north-star invariants

## Context

Turso (libSQL) supports **materialized views with incremental view
maintenance (IVM)**: `CREATE MATERIALIZED VIEW ... AS SELECT ...` is kept
consistent with its base tables inside every write transaction — no manual
REFRESH step. This is exactly the shape of metaengine's aggregate
acceleration problem: today, scalar aggregates over `meta_map`
(`SELECT SUM(json_extract(value, '$.amount')) FROM meta_map WHERE collection =
?`) scan and JSON-parse every row on every query, and grouped aggregates
re-scan and re-group per query.

metaengine's north star says storage placement is a **deployment-time
concern**: developers declare commands, events, queries, and relationships;
operators decide where data lives. A read-model acceleration that changes the
physical storage layer (a precomputed aggregate table) must therefore be
declared by the OPERATOR, not the developer.

## Decision

1. **Declarative specs, not raw SQL.** The operator declares WHAT to
   accelerate — `(collection, fn, column, groupBy?)` as
   `metaengine.MaterializedViewSpec` (`system.EngineConfig.MaterializedViews`
   in YAML). The engine derives the `CREATE MATERIALIZED VIEW` DDL, so the
   mapping between spec and view columns is exact by construction; the library
   never guesses what an arbitrary operator-written SELECT means.

2. **Construction-time creation, loud failure.** Specs flow
   `system → metaengine.DriverConfig → driver factory` and the views are
   created (`IF NOT EXISTS`, restart-idempotent) during engine construction.
   Engines that cannot maintain materialized views (plain SQLite) FAIL
   CONSTRUCTION with a hint naming the required Turso feature
   (`experimental=views` DSN param / server flag) — a silent ignore would
   yield a green deployment with zero acceleration.

3. **Automatic feature enablement on embedded DSNs.** When specs are present,
   the turso driver appends `experimental=views` to file/`:memory:` DSNs
   (deduplicated). Remote DSNs pass through untouched: the flag is a
   server-side setting there, and construction fails with the server's own
   error if the feature is missing.

4. **Exact serving rules; honest fall-through.** The engine serves an
   aggregate from a view only when the rewrite is algebraically EXACT:
   - unfiltered scalar aggregate ↔ scalar view of the same shape (AVG views
     store SUM and COUNT columns; the average is the quotient), or exact
     derivation from a grouped view of the same shape (SUM of group sums,
     SUM of counts, MIN of mins, MAX of maxes, weighted average);
   - unfiltered grouped aggregate ↔ grouped view of the same shape
     (per-group SUM/COUNT quotient for AVG).
     Everything else — filtered aggregates, planned-table collections,
     multi-aggregates, DISTINCT — falls through to the base path. The
     planned-table guard is a correctness rule, not an optimization detail:
     `ApplyLayout` creates an EMPTY planned table with no backfill, so a view
     over `meta_map` cannot serve a collection once it has migrated.

5. **Observability.** Engines report their views via
   `MaterializedViewsReporter`; `Store.Doctor` renders a
   `--- Materialized views ---` section (view name, shape, live row count)
   and `ExplainAggregateQuery` returns the view SQL the serving path would
   run — EXPLAIN never lies about what executes.

## Consequences

- Unfiltered rollups (totals, per-key counts/sums/averages, per-customer
  groupings) drop from O(rows × json_extract) to O(1) (scalar) or O(groups)
  (grouped) reads; writes pay IVM maintenance per view (measured in
  `docs/benchmarks/2026-09-07_turso-materialized-views.md`).
- Operators gain a tuning surface that requires NO code change — declare a
  view, restart, read the acceleration in Doctor.
- v1 scope limits: `meta_map` (standard-path) collections only; planned-table
  acceleration would need view creation to be ordered with `ApplyLayout`
  (and backfill semantics) — deferred until a deployment needs it.
- Upstream constraint (turso-go v0.7.2, repro verified): COMMIT of
  transactions that maintain materialized views fails deterministically
  once a process has written ~27k view-maintained rows ("no transaction is
  active"); smaller shapes fail probabilistically near the boundary. Keep
  cumulative view-maintained writes per process under that ceiling — chunk
  transactions to ≤ ~1k statements AND rotate process/file beyond the
  budget. See AGENTS.md and the upstream issue draft
  (`docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md`).

## Alternatives considered

- **Operator-written view SQL + query-shape matching** — rejected: matching
  arbitrary SQL to aggregate shapes is heuristic, silently wrong matches
  produce wrong numbers, and the operator burden is higher for less safety.
- **Engine-internal automatic matviews ("the engine decides")** — rejected:
  violates the developer/operator separation; IVM costs write throughput, so
  the choice of what to accelerate is a workload/business judgment.
- **Only expose DSN passthrough (`experimental=views`) and let operators
  hand-create views** — rejected: the planner/serving layer could not use
  them safely without the same shape metadata, so the feature would be dead
  configuration.
