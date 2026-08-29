# Status Report — 2026-08-16 19:51: Wave5 TODO Sweep (conformance loop, capability surfacing, DuckDB graph, Dgraph hardening)

Session scope: the 7-item paste from TODO_LIST (wave4 follow-ups). Everything
below reflects only this session's work and what was directly observed.

---

## a) FULLY DONE

1. **9-engine capability conformance loop, ALL GREEN** (TODO item: "Run the
   9-engine conformance loop under `#test-integration`").
   - Real-server rows executed: pgengine via `nix run .#integration-pg`
     (ephemeral PG, `go test -C` passthrough, 0.167s); dgraphengine via
     `nix run .#ephemeral-dgraph` (0.842s); mysqlengine via ephemeral
     `mysql:8.4` Docker container + `MYSQL_TEST_DSN` (0.175s, container
     cleaned up after).
   - Local rows: pebble, bbolt, badger, iroh, turso, duckdb (cgo tag) — all
     EXIT=0; root module (memory+sqlite) green.
   - Operational gotchas discovered and recorded in TODO_LIST: `go test -C
     <dir>` MUST be the first flag through the ephemeral scripts' `go`
     passthrough (two failed round trips before this); `GOWORK=off` must be
     exported BEFORE the passthrough; host `go test` outside the nix shell
     lacks `dgraph` on PATH (must use the nix app).
2. **irohengine optional-capability forwarding audit** — full policy written
   into `engine_passthrough.go` as a doc-comment table:
   - Closer: forwarded (transport closes first, then local).
   - MapUpdater/ScanBackend/Vector/Search/Spatial/graph: local passthrough
     (write-shaped members do NOT replicate — no WriteOp wire kinds).
   - Transactional, StreamLogBackend/SeqSeekableStreamLog/AtomicAppender:
     DELIBERATELY not forwarded — forwarding would let writes bypass
     `publish()` and silently diverge peers; the system adapters' LogBackend
     fallback is the converging route.
   - Prober/TransactMeasurer: deliberately not forwarded — a forwarded probe
     measures local RTT (~0) and live calibration would override the honest
     replication-derived NetworkRTT.
   - Pinned by new `engine_capability_forwarding_test.go` (Closer present;
     tx/streamlog/seqseek/atomic/prober/measurer absent — each with a policy
     pointer). Method-set diff (memory vs wrapper) documented as
     dropped-by-design: temporal reads (MapGetAsOf, StreamReadAsOfVersion,
     StreamVersion), VectorSearchFiltered, SnapshotBackend — flagged for
     future triage rather than silently ignored.
3. **Surface capability drift beyond tests** — three deliverables:
   - Doctor `--- Capability ---` section now prints an honest-degradation
     note for replicated graph engines: "graph writes are local-only … edges
     do NOT converge across peers" (`capability_audit.go`).
   - `ExplainPlan` renders `--- Capability Warnings ---` banner with one
     `WARN capability drift:` line per CapabilityAudit violation; healthy
     plans stay banner-free (verified both directions).
   - `TestAdttestStaysDelegatingOnly` meta-test (source-level AST scan of
     `metaengine/adttest/*.go`): verdict literals forbidden outside
     metaengine; every CapabilityAudit call must route through the
     `metaengine` package; fails if delegation disappears. Comment-stripping
     added so doc prose doesn't false-positive.
   - Behavioral tests: `TestDoctorNotesGraphNonReplication` (positive +
     negative — non-replicated engine gets no note),
     `TestExplainPlanShowsCapabilityDriftBanner`.
4. **iroh graph WriteOp replication** — resolved by choosing option B (keep
   the honest local-only note): Doctor note (above) + policy table + pinned
   tests. Rationale: replicating edges needs a new wire WriteOp kind plus
   CRDT edge-set semantics — no consumer demand; documented revisit criteria.
5. **Dgraph engine hardening** — all three sub-items:
   - Transactional: **ADR-0129** (`docs/adr/0129-dgraph-engine-transactional-deferred.md`)
     accepted — per-op dgo.Txn is the current unit of work; ambient-tx
     plumbing, ErrAborted→Conflict mapping, and `enginetest.RunTransactionalTest`
     gate sketched for when a consumer needs it. ADR README index updated
     (also backfilled the missing 0128 row).
   - Per-test isolation: `uniqueCollection(tb, base)` helper (pid + atomic
     counter suffix) in `helper_test.go`; replaced every fixed collection
     literal in the suite (injection×3, stream_log×6, multimap/log×2,
     products, events_parity). Full dgraphengine suite re-run against real
     ephemeral Dgraph: GREEN (56s).
   - CI/test-all-backends: `test-all-backends.sh` already had Dgraph (Phase 4)
     — the gap was CI + docs. New `dgraph` job in ci.yml runs
     `nix run .#integration-dgraph`; AGENTS quick-ref updated to list Dgraph.
6. **DuckDB real aggregation pushdown (AggregateReader)** — resolved STALE:
   the premise no longer holds. `duckdbengine/aggregations.go` implements the
   full pushdown family (Aggregate, GroupedAggregate, MultiAggregate,
   MultiGroupedAggregate, DistinctValues; planned + standard JSON paths) and
   `CounterGet` is a single SQL SELECT over a dedicated `meta_counter` table.
   All aggregation tests re-run green (cgo). Annotated as resolved-stale in
   TODO_LIST rather than "done" to keep the history honest.
7. **DuckDB native graph via recursive CTE** — implemented:
   - `meta_graph_edges` table added to init DDL (PK collection,from,to).
   - `GraphAddEdge`: `INSERT … ON CONFLICT DO NOTHING` (idempotent).
   - `GraphNeighbors`: single `WITH RECURSIVE walk` CTE mirroring pgengine's
     semantics (UNION dedup of (node,depth), DISTINCT nodes, start excluded
     so cycles never re-admit it).
   - Profile upgraded: ADTGraph O(N)→O(degree^depth), removed from
     DegradedADTs. Capability conformance re-run GREEN (would have caught
     over-declaration).
   - `graph_cgo_test.go`: depth 1-3 traversal, cycle safety, dedup,
     duplicate-edge idempotency, integer node keys, depth-0/empty honesty.
     Full duckdbengine suite green (73.9s).
   - Node-key encoding mirrors sqlite/pg/mysql with `art-dupl:accept
     cross-module SQL engine pattern` comment; duplication gate shows ZERO
     new clones from my files.

Supporting work: api-stability golden regenerated (4147→4186 exports — mostly
a concurrent session's symbols; mine were exactly the duckdb graph pair);
doc-check green (898 refs); lint green on irohengine (0 issues), duckdbengine
(0), my metaengine files, my dgraphengine files; `nix fmt` applied; vet green;
CHANGELOG section added; TODO_LIST items marked done with detail.

---

## b) PARTIALLY DONE

1. **Full `#verify` gate NOT run.** Per AGENTS.md the "stale GREEN" rule
   requires `nix run .#verify` before claiming GREEN; I ran targeted suites
   (all modules I touched, full irohengine/duckdbengine/dgraphengine suites,
   metaengine slices) but not the whole-gate sweep. Blocking context: the
   metaengine root package currently fails 5 tests from a concurrent
   session's in-flight file (see d), so a full gate would fail on code I
   don't own.
2. **metaengine root module full test run** — my slices green, but a full
   `go test .` is red from `graph_vector_features_test.go` (not mine).

---

## c) NOT STARTED (adjacent TODO_LIST items observed, untouched)

- Turso explicit CTE-probe test (tursoengine probe over remote protocol).
- Badger engine vector + graph parity audit.
- Vector search at scale (quantization/HNSW spike).
- `VectorResult` filtered k-NN; `GraphRemoveEdge`; graph directed-vs-undirected.
- mysqlengine upsert semantics audit; MariaDB functional-index alternative.
- enginetest per-run collection suffixes (the TODO I solved locally for
  dgraphengine exists as a cross-engine item — my `uniqueCollection` pattern
  could be generalized into enginetest).
- adttest graph depth>2 + cycle scenarios; Vector scenario on pgengine.
- Convergence-suite order-tolerance audit; quic pooled-stream ordering doc.
- CTE-vs-BFS crossover bench; MariaDB dual-key sort bench.
- `nix run .#integration-mysql-nspawn` real-env run (needs root).

---

## d) TOTALLY FUCKED UP / PROBLEMS HIT

1. **Round trips wasted on `go test -C` ordering** — two full ephemeral
   server spin-ups (PG ~30s, Dgraph ~40s) burned because `-C` was not the
   first flag. Discovered the rule, recorded it — but should have read the
   flag docs before the first run.
2. **Host env traps**: `GOTOOLCHAIN=auto` needed everywhere (go.work wants
   1.26.6, host has 1.26.5 — AGENTS documents this, I still hit it);
   `/mnt/buildcache` went missing mid-session ("no such device") forcing
   `GOCACHE=/tmp/... GOMODCACHE=$HOME/go/pkg/mod` workarounds on every
   command. The nix develop shell was unaffected.
3. **`storage TestPostgresEventStore_CRUD` flake observed** ("expected 2
   events, got 27") when the ephemeral-pg script ran its FULL default module
   list — shared-journal cross-test contamination. Pre-existing, not caused
   by this session, not fixed; noted in TODO_LIST annotation. This flake also
   cost one failed run before I switched to per-module passthrough.
4. **NOT MINE, STILL RED — concurrent session collision**: untracked
   `metaengine/graph_vector_features_test.go` fails 5 tests
   (`invalid FoldKind: "edge_remove"`, VectorFilterBackend error text) plus
   its accompanying dirty production files (`metaengine/dispatch.go`,
   `fold_classify.go`, `enum_validation.go`, `adttest/harness.go`,
   sqlite/badger/pebble graph+vector files, catalog/*). I deliberately did
   NOT touch or fix these (AGENTS: never revert changes you didn't author).
   Same for the 11 new art-dupl clone groups (mysql/sqlite
   `graph_undirected.go`, quickstart examples) — the gate is red on their
   account, zero from mine.
5. **Edit-tool churn** — several edit failures from writing files before
   reading (write-after-modify guard) and a broken multiedit on
   `capability_surface_test.go` (constructor name guessed as `NewStore`;
   actual is `Plan`). Cost ~4 round trips. Should have grepped the API first.
6. **api-stability golden regen picked up a concurrent session's 39
   exports** (undirected graph, edge removal, vector filter APIs) in the same
   update as mine. The diff is honest, but my commit boundary is now
   entangled with theirs — worth splitting or at least noting at commit time.

---

## e) WHAT WE SHOULD IMPROVE

1. **Concurrent-session collision policy is undefined.** Two agents working
   the same repo (undirected-graph person + me) produced entangled golden
   files, a red root package, and a red duplication gate. Need a convention:
   either lock files per work-stream, or accept golden/baseline regen as
   "last writer merges".
2. **`nix develop` vs host Go env** — the host Go caches broke mid-session.
   Standardize: all go commands through the dev shell (I ended up there).
3. **The ephemeral scripts' `go` passthrough docs** — the `-C must be first`
   rule should be a comment in `ephemeral-pg.sh`/`ephemeral-dgraph.sh`
   headers (I only recorded it in TODO_LIST).
4. **`uniqueCollection` should graduate into enginetest** — the dgraph fix
   solves one module; the same fixed-name hazard is the existing
   "enginetest per-run collection suffixes" TODO. Generalize.
5. **Verify gate hygiene** — when a foreign in-flight file blocks the full
   gate, there's no sanctioned "verify mine only" mode. A `verify-fast
   --exclude-dirty-foreign` or just a documented workflow would help.
6. **Storage PG CRUD flake** — the shared-journal contamination should get a
   dedicated TODO (it doesn't have one yet, only my inline note).

---

## f) NEXT — up to 50 things (ordered by leverage)

**Gates & hygiene**

1. Run full `nix run .#verify` once the concurrent graph/vector session lands.
2. Triage the 11 art-dupl groups (undirected-graph session's); accept-comment
   or baseline regen after their owner confirms.
3. Split the api-stability golden diff at commit time (mine: duckdb graph ×2).
4. File a TODO for `TestPostgresEventStore_CRUD` shared-journal flake.
5. Fix `/mnt/buildcache` mount or stop depending on it.
6. Add `-C first flag` note to ephemeral script headers.

**DuckDB follow-ups**
7. Add duckdb graph row to the adttest cross-engine matrix (parity vs pg).
8. EXPLAIN AGGREGATE/GRAPH section for duckdb graph (`explain.go`).
9. `GraphRemoveEdge` on meta_graph_edges (DELETE; pairs with EdgeRemoval fold).
10. Graph directed-vs-undirected option (duckdb `UNION ALL` both directions).
11. Bench: duckdb recursive CTE vs pg at depth 3-5 (cost-model calibration).
12. DuckDB secondary index on (collection, from_node) if PK insufficient.

**Dgraph follow-ups**
13. Implement Transactional per ADR-0129 sketch when a consumer needs it.
14. dgraphengine CI job: add `-race` variant if runtime allows.
15. Generalize `uniqueCollection` into enginetest helpers (existing TODO).

**iroh follow-ups**
16. Temporal reads through Replicated (MapGetAsOf/StreamReadAsOfVersion) —
forward or document per the new policy table.
17. VectorSearchFiltered through Replicated — same decision needed.
18. Graph WriteOp wire kind + CRDT edge-set if a consumer demands convergence.
19. iroh conformance test parity with loopback/quic matrices (currently
separate `TestLoopbackADTMatrix`/`TestQuicADTMatrix`).

**Capability surfacing follow-ups**
20. EXPLAIN banner: include the replicated-graph non-convergence note too
(currently Doctor-only).
21. Doctor: surface dropped-by-design iroh capabilities as informational lines.
22. Add CapabilityGaps entries where engines legitimately diverge (self-doc).

**Adjacent TODO_LIST items (not started, c-section)**
23. Turso CTE-probe test. 24. Badger vector+graph parity audit.
25. VectorResult filtered k-NN. 26. HNSW/quantization spike.
27. GraphRemoveEdge (generic). 28. mysqlengine upsert audit.
29. MariaDB generated-column indexes. 30. adttest depth>2 cycle scenarios.
31. pgengine Vector scenario in adttest. 32. Convergence order-tolerance sweep.
33. quic pooled-stream ordering doc. 34. CTE-vs-BFS crossover bench.
35. MariaDB dual-key sort bench. 36. `#integration-mysql-nspawn` real run.
37. Per-run suffixes in enginetest (same as 15 — keep one).
38. Doctor routing-section test for the new Capability notes (golden-ish).

**Docs & meta**
39. Update `.agents/skills/go-cqrs-lite/references/*` for duckdb native graph
(module docs still say degraded).
40. metaengine README 9-engine matrix → include graph complexity column.
41. Update TODO_LIST "Badger engine vector + graph parity audit" with the new
duckdb precedent as reference implementation.
42. Consider CI job that runs the 9-engine conformance loop nightly (the
thing this session ran manually).
43. CHANGELOG: separate my section from the concurrent session's entries at
release-cut time.

---

## g) QUESTIONS (cannot determine myself)

1. **Concurrent session ownership**: `graph_vector_features_test.go` and the
   undirected-graph/vector production changes are dirty in the tree — is
   another agent actively mid-flight, and should I leave the tree as-is, or
   do you want me to stash/coordinate before any commit of my work?
2. **api-stability golden**: it now contains a concurrent session's 39
   exports mixed with my 2. Commit the golden as-is (their code presumably
   lands anyway), or do you want a hand-filtered golden containing only my
   exports (which would then fail their check-in)?
3. **MySQL container vs nspawn for conformance**: I used an ephemeral
   `mysql:8.4` Docker container (fast, available). The TODO mentioned
   "under `#test-integration`" whose MySQL path is nspawn (needs root I don't
   have). Is Docker acceptable as the standing method, or should the
   conformance loop be wired into a nix app (`#conformance-loop`) that
   prefers nspawn/Docker/testcontainers in order?
