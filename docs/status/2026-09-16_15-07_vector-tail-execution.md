# Status Report — Vector Verification Tail Execution — 2026-09-16 15:07 CEST


> **RESOLVED (2026-09-16, same-day execution pass):** every §g next-step item
> 1-14 executed. Key outcomes: DuckDB DDL concat fixed + full cgo suite green
> (83s); bisection proved the break born-broken in `284d78ebe` (18:28 wave) —
> the 18-32 "CGo suite green" claim was vacuous (no `-tags cgo`); benchmark doc
> written with quiet-machine medians (libSQL 0.76ms / sqlite Go-scan 0.97ms /
> DuckDB 1.22ms — the earlier 1.09/1.75 figures were load noise);
> `#integration-dgraph` green 7× (incl. the previously-hanging seed replay)
> after four real fixes: `dims`→`vecs` DQL root name (dimension guard never
> fired), nil-map init in `ensureVectorSchema`, `errIndexingInProgress` retry +
> gRPC deadline interceptor, serial reset tests (the RWMutex gate proposed in
> §g deadlocked and was reverted — see the CHANGELOG 2026-09-16 Fixed section);
> MariaDB leg green after the DECIMAL probe fix + serial reset tests; PG leg
> green live; metaengine full suite green; lint clean on all touched modules
> (repo-wide lint has ~50 pre-existing findings in queue-family + doc-check +
> otel/otlp + api-stability files this session did not author — attribution
> with the parallel session); file-size, duplication (6 accepted dialect-twin
> groups incl. the gate-script `turso_preset` literal-claim fix), and
> error-taxonomy gates all green. §g questions 1-3: 1 — attribution above;
> 2 — VectorCounter stays local-only (no promotion); 3 — live legs done
> locally (MariaDB userspace + ephemeral nix PG/Dgraph), no CI dependency.

> Scope: this session only (started ~14:00). Task: execute the TODO_LIST
> "Vector-search verification tail" (items a–h from the 2026-09-15 18:32
> report) plus the Dgraph floor decision and ADR-0140. Tree state at report
> time: 17 dirty files (latest wave not yet daemon-absorbed); all earlier
> waves absorbed into `chore: auto-commit` commits (52e165ca1 et al.).
> A PARALLEL session is active (its report landed 15:02, 5 min before this
> one) — ownership boundary respected throughout.
>
> Format note: written as Markdown per explicit user instruction (skill's
> HTML default overridden — same override as the 2026-09-15 18:32 report).

## a) FULLY DONE (verified this session)

| Item | Verification |
| ---- | ------------ |
| (b) `system` module tests + `example/metaengine-quickstart` re-run after `SQLiteEngineProfile` gained ADTVector | both `GOWORK=off go test ./...` PASS |
| **ADR-0140** — vector distance-semantics contract (cosine=1−cosSim, dot=NEGATED, euclidean=L2, ascending=nearest, degrade-everywhere, pre-filter AND) | `docs/adr/0140-vector-distance-semantics-contract.md` written; indexed in `docs/adr/README.md` (also added the missing 0139 row, date corrected to 2026-09-13) |
| (e) lazy libSQL probe — sqliteengine `vectorSQL` is now `sync.OnceValue`, probed on first vector use, construction query-free | sqliteengine build + full local vector suite PASS |
| (d) `metaengine.VectorPathReporter` (+ `VectorPathPushdown`/`VectorPathScan`) — ExplainPlan and Doctor now render a per-engine `k-NN path <label>` line; implemented on sqlite (probed), duckdb (pushdown), mysql/pg/dgraph/pebble/bbolt/badger/memory (go-scan), iroh (forwards, `""` when local can't report) | new `metaengine/vector_path_test.go` PASS (path line rendered, reporter-less engines skipped, no-vector stores keep old output); existing Doctor/ExplainPlan assertions still green |
| explain.go ratchet rescue — `explainVectorWarnings`/`vectorDoctorSection` moved out of growth-locked explain.go | explain.go 515 → **430** lines; build+vet green |
| vector_search.go ratchet rescue — `MemoryVectorIndex` extracted to new `vector_memory.go` | vector_search.go 453 → **348** lines; build green; memory engine delegates through it unchanged |
| (a) irohengine vector passthrough VERIFIED — `TestReplicatedVectorPassthrough` proves insert + k-NN ordering + filtered k-NN + path forwarding through `Replicated(memory)`; discovered the real surface: `VectorBackend` + `VectorFilterBackend` forwarded, `VectorCounter` deliberately NOT (forwarding policy) | irohengine suite PASS; CHANGELOG "every engine" enumeration completed with the honest iroh bullet |
| (f) **Dimension lock on every engine** — first insert establishes a collection's dimension; mismatching/zero-dim inserts rejected with `metaengine.ErrVectorDimensionMismatch` (errorfamily Rejection). Shared `metaengine.CheckVectorDimension`; per-engine established-dim probes (SQL `LENGTH(vec)/4` / `len(vec)` / `jsonb_array_length`, KV `keycodec.VectorPrefix` first-record seeks, dgraph DQL `first: 1`); `adttest.AssertVectorDimensionGuard` shared assertion called from 10 modules | PASS live: sqlite (modernc), turso (embedded libSQL), bbolt, pebble, badger, memory core. Written, gated on live servers: mysql (DSN), pg (testcontainer), dgraph (ephemeral nix), duckdb (cgo — see §d) |
| (g) Vector persistence in the shared restart-safety harness — phase 1 inserts 2 vectors (with metadata), phase 2 asserts k-NN ordering across reopen + a post-restart insert that exercises the dimension lock against persisted rows | `enginetest.RunRestartSafetyTest` PASS on sqlite, bbolt, pebble, badger (duckdb leg = cgo-tagged, see §d) |
| Dgraph v24 floor DECIDED + implemented — vector predicates (`float32vector`) moved OUT of construction-time `init()` into `ensureVectorSchema` (mirrors `ensureEdgeSchema`, `appliedSchemas` cache), wired into all 5 vector entry points; old servers keep booting and serving every other ADT; first vector op fails with an actionable "vector predicates require Dgraph v24+" error. README gained a "Version compatibility" section | dgraphengine build green; NOT yet live-tested (ephemeral-dgraph run is the next step) |
| Benchmark trio written — `BenchmarkVectorSearch_GoScan` + `BenchmarkVectorInsert_GoScan` (sqliteengine), `BenchmarkVectorSearch_LibSQLPushdown` (tursoengine), `BenchmarkVectorSearch_SQLPushdown` (duckdbengine, cgo-tagged); 1000×64-dim corpus, k=10, cosine; asserts the pushdown label before measuring | libSQL pushdown ≈ **1.09 ms/op (8.4 KB, 343 allocs)** vs Go scan ≈ **1.75 ms/op (944 KB, 10 036 allocs)** — pushdown ~1.5× faster with ~112× fewer bytes; 3×2s runs each. DuckDB bench blocked (§d1) |
| API golden regenerated twice in the same edit as the symbol changes | +22 lines (VectorSearchPath ×10 engines + interface + 2 consts), then +3 (`CheckVectorDimension`, `ErrVectorDimensionMismatch`, `adttest.AssertVectorDimensionGuard`) — all verified session-owned; `TestEvery` PASS |
| AGENTS.md contract #26 extended — ADR-0140 link, dimension lock, VectorPathReporter, lazy probe | written |
| CHANGELOG updated — iroh enumeration bullet, VectorPathReporter bullet, dimension-lock bullet, lazy dgraph schema bullet, benchmark numbers, sqlite bullet reworded to lazy probe | written (one bad cross-reference, see §d3) |

## b) PARTIALLY DONE

- **DuckDB (all legs).** The full duckdb cgo suite has NOT run this session.
  My two earlier "PASS duckdbengine" runs were **vacuous** — invoked without
  `-tags cgo`, so every cgo-tagged test file (including my new
  dimension-guard test, the restart-safety leg, and the benchmark) never
  compiled. Caught only when the benchmark finally ran with explicit tags —
  and then construction itself failed (§d1).
- **metaengine full non-short suite (item h)** — not run yet; planned as the
  last heavy step before composed gates.
- **Dgraph "Transaction has been aborted" flake** — not investigated; only
  context gathered (retryOnContention covers standalone ops; parallel-load
  aborts during adt matrices remain unexplained).
- **Live-server legs of the dimension guard** — mysql (needs `MYSQL_TEST_DSN`
  userspace MariaDB), pg (needs Docker/testcontainers), dgraph (needs
  `.#integration-dgraph`) — tests written, none executed.
- **Composed gates** — `#verify`, `#verify-fast`, `#check-coverage`,
  `#vulncheck`, `doc-check`, per-module lint over the ~14 touched modules:
  none run yet.
- **TODO_LIST checkbox updates** for the tail items — not done yet.
- **`check-error-taxonomy` exposure** — `ErrVectorDimensionMismatch` adds a
  new errorfamily code (`metaengine.vector_dimension_mismatch`); the
  bidirectional drift gate against `docs/error-taxonomy.md` has not been run
  and may require a taxonomy entry.

## c) NOT STARTED

- Remaining 18-32 §f items untouched this session: remote-Turso pushdown
  verification; Doctor `--- Vectors ---` live test on a real new engine;
  native ANN paths (Dgraph similar_to, DuckDB VSS, sqlite-vec, pgvector,
  MariaDB VECTOR); `docs/agents/module-map.md` engine vector notes;
  quickstart Turso-pushdown demo variant; empty-query-vector doc+test;
  OTel spans for vector ops; `#check-coverage`/`#vulncheck` runs.
- `verify-fast` pre-existing master red — not touched (still unowned).
- iroh `VectorCounter` forwarding promotion — needs a policy decision, not
  silently made this session (surface documented instead).
- Bulk/batch vector insert API (the per-insert dimension probe costs +1
  query per insert — +1 RTT on remote engines).

## d) TOTALLY FUCKED UP

1. **DuckDB engine construction is BROKEN in the committed tree.**
   `duckdbengine.init` concatenates the `meta_graph_edges` DDL (ends `)`, no
   semicolon) directly with `vectorTableDDL` (engine.go:127), producing an
   invalid multi-statement string → `Parser Error: syntax error at or near
   "CREATE" ... LINE 7: CREATE TABLE IF NOT EXISTS meta_vector` at engine
   construction. Every vector claim for DuckDB is currently undeployable.
   Yesterday's report §a claims "full CGo suite green (113 s)" — that is now
   inconsistent with the tree (the DDL-const refactor or a later absorbed
   edit reintroduced it; my session's diff does not touch those lines).
   Needs: bisect the auto-commits, one-line fix (`;` separator), full cgo
   suite rerun. NOT fixed yet — report-first instruction.
2. **Vacuous-green duckdb runs (twice).** I ran the duckdb suite twice
   without `-tags cgo`, accepted the green, and moved on. The exact
   "pipeline-masking / gate-must-prove-it-measured" lesson from AGENTS.md
   (2026-09-11 collector-utils port) — repeated the class anyway. The
   vacuous runs masked both §d1 and the unverified status of my own duckdb
   changes.
3. **CHANGELOG cites a file that does not exist** —
   `docs/benchmarks/2026-09-16_vector-search-paths.md` referenced in the
   benchmark bullet but not yet written. Shipped a dangling doc reference
   (the fiction class `check-changelog-symbols`-style gates exist for).
4. **Filler line in the first turso benchmark draft** (`_ =
   adttest.AssertVectorDimensionGuard` no-op) — the exact §d2 sin from
   yesterday's report. Self-caught and removed pre-commit.
5. **Cross-directory scripting bug** — one python wiring step ran from the
   repo root (`FileNotFoundError`) while the chained `go build` still
   printed OK from the wrong context; caught immediately, re-run in the
   correct directory, verified by `rg -c` on the wired call sites.
6. **Process friction (no damage):** the stale LSP error on
   sqliteengine/vector.go:108 persisted red in tool output for the whole
   session while the real compiler was green — distrusted correctly per the
   "verify tool output" memory rule, but it cost repeated re-reads.

## e) WHAT WE SHOULD IMPROVE

- **Tag-gated suites need tag-gated invocations, always.** A green
  `go test ./...` on a module whose tests are `//go:build cgo`-gated proves
  nothing. For duckdb the invocation must be
  `-tags "cgo goexperiment.jsonv2"` — bake this into the module README and
  my muscle memory; better, add a `#test-duckdb`-style nix app so nobody
  hand-types tags again.
- **Never write a cross-reference before the target exists** (§d3) — file
  first, citation second.
- **The green runs I accept must each prove they exercised the code** —
  when a test file is new, run `-run <ItsTestName>` explicitly at least once
  before trusting the package-level `ok`.
- **Keep `cd` + build verification in one command** when scripting
  multi-file edits — today's §d5 split nearly laundered a wrong-directory
  no-op into a green banner.
- **Bisect before attributing breakage** — §d1 sits on top of a claim
  ("suite green") from yesterday; the honest default is "never verified in
  this exact tree" until the bisect says otherwise.
- **Dimension-lock probe cost is documented but unmeasured on remote
  engines** — one extra indexed query per VectorInsert; fine locally,
  +1 RTT per insert on pg/mysql/dgraph. A bulk insert API or probe caching
  (with a documented staleness tradeoff) is the follow-up.
- **art-dupl exposure unknown** — the 10 new per-engine
  `vector_dimension_test.go` files are near-identical dialect twins; they
  may trip `#check-dupl` without `//art-dupl:accept` directives. Unchecked.

## f) Things to get done next (impact-ordered, 30 real items)

1. **Fix the DuckDB DDL concat** (missing `;` before `vectorTableDDL`) and
   rerun the FULL duckdb cgo suite (`-tags "cgo goexperiment.jsonv2"`,
   ~113 s) — includes dimension guard, restart legs, pushdown tests.
2. **Bisect which auto-commit broke duckdb construction** and annotate the
   18-32 report's "suite green" claim (docs-health ANNOTATE).
3. **Write `docs/benchmarks/2026-09-16_vector-search-paths.md`** (or fix the
   CHANGELOG reference) — kill the dangling citation.
4. **Run the DuckDB pushdown benchmark**, add the third number to the
   CHANGELOG bullet.
5. **Run `nix run .#integration-dgraph`** (unfiltered, soak included) —
   live-verifies the lazy vector schema + dgraph dimension guard + Vector
   matrix; collects "Transaction has been aborted" flake evidence.
6. **Investigate the abort flake** against the collected evidence —
   `retryOnContention` classification gap vs distinct contention class.
7. **MariaDB leg**: userspace MariaDB flow → `MYSQL_TEST_DSN` → dimension
   guard + `TestMySQLVectorRoundtrip`.
8. **PG leg**: testcontainers → `TestVectorDimensionGuard` on pgengine.
9. **metaengine FULL non-short suite** (item h; SOAK_SKIP_* policy per
   gotchas-testing.md).
10. **Per-module lint** over all touched modules (metaengine + 10 engine
    modules + enginetest + adttest consumers).
11. **`nix run .#check-file-size`** — confirm the two shrunk baselines and
    all new files (vector_path.go, vector_memory.go, vector_dimension.go,
    adttest/vector_dimension.go) are clean.
12. **`nix run .#check-duplication`** — add `//art-dupl:accept` directives to
    the 10 dialect-twin test files if the gate trips.
13. **`nix run .#check-error-taxonomy`** — add
    `metaengine.vector_dimension_mismatch` to `docs/error-taxonomy.md` if
    the drift gate demands it.
14. **`cd cmd/doc-check && GOWORK=off go run …`** over SKILL.md + references
    + AGENTS.md (AGENTS.md changed; contract #26 now cites ADR-0140).
15. **`nix run .#verify-fast`** — attribute my delta vs the pre-existing red.
16. **TODO_LIST harvest**: check off verification-tail items (b), (d), (e),
    (a), (f-partial), (g), Dgraph floor decision, ADR-0140; leave pointers
    to this report.
17. **FEATURES.md vector row** — add dimension lock + path reporter +
    Dgraph lazy schema one-liners.
18. **Skill references** — `.agents/skills/go-cqrs-lite/references/modules.md`
    + `faq.md`: Dgraph v24 vector floor, `ErrVectorDimensionMismatch`
    semantics, ExplainPlan path labels.
19. **`docs/agents/module-map.md`** engine vector notes (18-32 §f23).
20. **Decide iroh `VectorCounter` promotion** (policy decision; owner input
    via §g2 below).
21. **Bulk vector insert API** (amortize the dimension probe on remote
    engines) — ROADMAP fuel unless ANN work lands first.
22. **Remote-Turso pushdown verification** (needs credentials — 18-32 §g2).
23. **Doctor `--- Vectors ---` live test** on one new engine (18-32 §f15).
24. **Quickstart demo variant exercising Turso pushdown** (18-32 §f24).
25. **`#check-coverage` + `#vulncheck`** over the changed modules (18-32 §f25).
26. **Empty-query-vector behavior** doc+test (18-32 §f28 — interacts with my
    zero-dim insert rejection; search-side empty-query behavior still
    duckdb-specific).
27. **OTel spans for VectorInsert/VectorSearch** if convention supports
    (18-32 §f29).
28. **benchmark-regression gate decision record** — deliberately not wired
    this session (corpus-sensitive O(N) paths vs 25% CI threshold); record
    the rationale where the gate docs live.
29. **`nix run .#verify`** composed run in a quiet window (standing BLOCKED
    row — still subject to the quiet-window rule).
30. **Annotate the 18-32 report** with this session's resolution of §b/§c
    items (docs-health ANNOTATE pass).

## g) Questions I can NOT figure out myself

1. **DuckDB breakage attribution:** yesterday's 18-32 report claims the full
   CGo suite ran green AFTER the DDL-const refactor, but the committed tree
   fails construction on the concatenated DDL. Do you know of a parallel
   session editing duckdbengine after that green run (I see a parallel
   session active right now), or should I bisect the auto-commits and treat
   the "green" claim as never-verified-in-this-tree?
2. **iroh `VectorCounter` promotion:** the forwarding policy deliberately
   does NOT promote VectorCounter through the `Replicated` wrapper (no size
   introspection across the wrapper). Now that vector support is
   enumerated, do you want a policy exception (forward read-only
   count/collections to the local engine, documented as non-converging), or
   does the wrapper stay honest-by-omission?
3. **Live-server legs now or CI?** MariaDB (userspace init at /tmp/mariadb-cqrs,
   documented in gotchas-testing) and PG (testcontainers/Docker) — should I
   spin both up locally this session to execute the dimension-guard legs,
   or leave them to the CI integration matrix?
