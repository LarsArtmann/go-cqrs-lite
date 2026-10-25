# Engine Coverage & Vector Substrate Research — Session Status

**Date:** 2026-10-25 15:13 CET (Sunday)
**Session shape:** question-driven research, not plan execution. Six operator questions about the storage-engine fleet were answered from source, and one new repository doc was written to stop the answers being re-derived.
**Artifacts written:** `docs/engine-coverage-and-gaps.md` (new), `docs/README.md` (Guides index row + stale count fix), `docs/METAENGINE_DOMAIN_LANGUAGE.md` (stale count fix).
**Gates run:** `scripts/check-readme-links.sh` (green: 872 relative link targets, 107 files, 0 broken), `nix run .#check-md-go` (green: 1519 code blocks valid, 84 skipped, 105 baselined archived errors, no new errors).
**Not committed:** Crush harness forbids commits without an explicit user request; the auto-commit daemon picks the tree up.

---

## a) FULLY DONE

| # | Item | Evidence |
| --- | --- | --- |
| A1 | **"Do we support RocksDB?" answered — no, with reasons.** Repo-wide search found 13 incidental mentions (a test comment grouping LSM backends, research prose, two planning docs) and **zero driver, engine module, ADR, or TODO row**. | grep across `*.go`/`*.md`; `docs/engine-capabilities.md` roster |
| A2 | RocksDB refusal rationale established: (i) RocksDB CGo bindings would create a **second CGo island** (policy: CGo is isolated to duckdb, ADR-0071); (ii) the LSM niche is already covered by pure-Go Pebble (ADR-0074) and Badger (ADR-0118); (iii) an engine add is a module + `register.go` + `enginetest` conformance + multi-gate registration, so it must clear the ADR-0142 A/B gate; (iv) the extension point (implement `metaengine.Engine` as a programmatic engine, iroh/graphadapter pattern) is available to any consumer. | AGENTS.md dependency rules; ADR-0142; `metaengine/irohengine`, `metaengine/graphadapter` |
| A3 | **Missing-class inventory built** (10 classes): server columnar/DWH (ClickHouse), distributed wide-column (Cassandra/ScyllaDB), native graph server (Neo4j/Memgraph — deliberately superseded by dgraph + SQL CTE fallback, `graph.GraphDriver` is the extension point), document (MongoDB), full-text search (Elasticsearch/Meilisearch/Typesense), vector-native ANN (Qdrant/Milvus/Weaviate/pgvector), NewSQL (CockroachDB/TiDB/Yugabyte — **marked unverified** re: PG wire), cloud beyond GCP (DynamoDB), time-series (InfluxDB/Timescale), object/blob (S3/GCS). | vision docs `docs/planning/meta-engine-design.md` (ClickHouse/Scylla/Neo4j deployments), `graph/graph.go:29-32`, 2026-09-13 reconciliation audit |
| A4 | **Deliberately-refused list separated from gaps**: Redis (transport/broker layer, ADR-0142 gate A: transport is not durable state), RocksDB/LevelDB/LMDB (covered + CGo), per-engine reimplementation of shared storage capabilities (ADR-0142: one shared runtime per class in metaengine core/claimkit/Map runtimes). | ADR-0142 table (dgraph/iroh REFUSED rows for claims), AGENTS.md contract 30 |
| A5 | **New doc written: `docs/engine-coverage-and-gaps.md`** — 7 sections: fleet table (13 engines by storage class), missing classes, refused candidates, extension points (registry driver / programmatic engine / `GraphDriver`), add-an-engine checklist, provenance. Class-level only; per-engine capability facts deliberately **linked** to the generated `engine-capabilities.md` rather than restated (AGENTS.md rule). | file read-back after write; gates green |
| A6 | Doc indexed in `docs/README.md` Guides list so it is discoverable, not orphaned. | `docs/README.md:27` |
| A7 | **"Best engine for vector data?" answered with measurement**: turso/Turso-Database SQL pushdown 759,423 ns/op (343 allocs, 8.4 KB) > sqlite go-scan 967,391 ns/op (10,036 allocs, 944 KB) > duckdb pushdown 1,217,911 ns/op (127 allocs, 4.1 KB) at 1000 vectors × 64 dims, k=10, cosine. Caveat stated: all three are O(N) brute force; no ANN ships anywhere. | `docs/benchmarks/2026-09-16_vector-search-paths.md` |
| A8 | **"libSQL vs modern Turso — what do we use?" answered**: we embed **Turso Database, the Rust SQLite rewrite**, via `turso.tech/database/tursogo v0.8.2` (driver name `turso`), CGo-free through `purego`; libSQL is the legacy C fork. The driver is **embedded-only** — no remote Hrana/`libsql://` client, so remote DSNs are planner-level RTT priors. | `metaengine/tursoengine/go.mod`, `README.md`, `docs/agents/gotchas-tooling-build.md:51` |
| A9 | **"Why no SQL pushdown in sqlite?" corrected to the accurate answer**: we DO have conditional pushdown — a lazy probe (`SELECT vector_distance_cos(vector32('[1]'), ...)`) fires on first vector use and routes to `vectorSearchPushdown`; embedded it cannot fire because (i) `vector32`/`vector_distance_*` are libSQL/Turso features absent from stock SQLite, (ii) modernc's `database/sql` surface has no `LoadExtension` so sqlite-vec cannot load, (iii) CGo sqlite + extension would breach the CGo-isolation policy. Tracked future path: a pure-Go `vector32()` registration (ROADMAP, go-graph-rag feedback #1). | `metaengine/sqliteengine/vector.go:46-48,123,141`; `ROADMAP.md` ANN row |
| A10 | **"Pebble vector performance?" profiled from source**: LSM writes are the cheapest in the fleet (one `Set` of a binary float32 payload, `EncodeVectorBinary`, optional metadata key); reads are O(N·D) brute force, declared `ComplexityON` + degraded, `VectorSearchPath = go-scan`, with the lowest per-row decode cost of the brute-force engines (direct prefix iterator, no `database/sql`, binary not JSON; filtered path loads metadata per row and filters **before** scoring). No published benchmark exists for it. | `metaengine/pebbleengine/vector.go:16-24,47,95,113-116,189-191` |
| A11 | **"Do we use extensions?" answered — zero, deliberately**: no pgvector (stock-PG JSONB scan, `pgengine/vector.go:13-19` explicitly tells operators to deploy pgvector themselves for production scale), sqlite-vec impossible, DuckDB VSS unused (core `array_*` only), MariaDB `VECTOR` is a ROADMAP item, Turso uses native engine functions, Dgraph `similar_to`+HNSW not wired. Rationale: capability probing + honest degradation keeps engines interchangeable (ADR-0142) and pure-Go buildable. | per-engine `vector.go` files; `docs/adr/0140-*.md:80`; CHANGELOG MariaDB row |
| A12 | **Two stale engine counts fixed on sight** (`9 engines` → `13 engines`, grounded in the generated roster, with a pointer to `engine-capabilities.md`): `docs/README.md:26` and `docs/METAENGINE_DOMAIN_LANGUAGE.md:23`. The first sat directly above the line this session added — leaving it would have put a fresh, verified fact next to a stale one. | edits + link gate green |
| A13 | Session-answer durability: the fleet/vector/extension knowledge now lives in a repository doc instead of only in chat, per the repo's "write it down" memory discipline. | A5 |

---

## b) PARTIALLY DONE

| # | Item | What is missing |
| --- | --- | --- |
| B1 | **The gaps doc has no demand evidence.** Each missing class is listed with rationale but no "who would need this / what would have to be true to build it" column, so the table is informational, not actionable. | Revisit-trigger column + fleet-census cross-check per class |
| B2 | **One claim in the new doc is explicitly unverified** — CockroachDB speaking PostgreSQL wire so `pgengine` may work. Flagged inline as unverified (honest), but not tested. | Either verify against a real CockroachDB or drop the row to a footnote |
| B3 | **Add-an-engine checklist compressed from procedure, not audited.** It lists the ADR-0142 gate, module + dep budget, `register.go` + `enginetest`, the gate registrations (`go.work`, `flake.nix`, `check-module-layers.sh`, `cmd/api-stability`, `cmd/cqrs-lint`), and the `engine-capabilities.md` regeneration — but I did not execute or read each gate in this session; the list is AGENTS.md-derived. | Walk one real engine add against the checklist (or diff it against `AGENTS.md` §Add a New Module line by line) |
| B4 | **Cross-links one-directional.** The new doc points at `engine-capabilities.md` and ADR-0142; nothing points back from `ROADMAP.md`'s "Native ANN vector indexes" row or `metaengine/README.md`. | Add back-links |
| B5 | **No HARVEST.** §f below is not yet in `TODO_LIST.md`/`ROADMAP.md`; per the status-report skill the loop closes only after HARVEST. Deferred because the user asked to wait for instructions. | Run `docs-health` HARVEST on §f |
| B6 | **`docs/status/` hygiene untouched.** The link gate printed: "docs/status/ holds 127 live reports (>10) — move finished snapshots to docs/status/archived/ and index the rest in docs/status/README.md." Pre-existing debt, not caused here, but the gate NOTE recurred. | Bulk archive + index pass |
| B7 | **Vector answers are accurate but their caveats were spread across turns** — the turso defect context (IVM) arrived only while answering question 5, after the "best engine" recommendation in question 4. | Consolidate the caveats in the doc/faq so a reader sees recommendation + caveats together |
| B8 | **No FEATURES.md presence** for the coverage doc. Judged as doc-hygiene rather than feature inventory (defensible), so left alone — but the judgment is undocumented. | Record the judgment or add a row |

---

## c) NOT STARTED

1. HARVEST of §f into `TODO_LIST.md` / `ROADMAP.md` (skill-mandated next step).
2. Revisit-trigger/demand-evidence pass over the gaps doc (B1).
3. CockroachDB PG-wire verification (B2).
4. Back-links from `ROADMAP.md` and `metaengine/README.md` (B4).
5. `docs/status/` archive-and-index hygiene pass (B6).
6. Any actual vector-roadmap work: sqliteengine pure-Go `vector32()`, DuckDB VSS gating, pgvector adapter, ANN capability flag on `EngineProfile`, `VectorCount` advisory for Doctor — all still trigger-gated.
7. Annotation of `docs/benchmarks/2026-09-16_vector-search-paths.md`: its environment table still claims Go 1.26.x and `-tags "goexperiment.jsonv2"`, a no-op since the 2026-09-19 graduation sweep. Noticed, deliberately not rewritten (dated record → `docs-health` ANNOTATE mode), but the note was never written either.
8. `nix run .#check-engine-capabilities` (the drift gate) was not executed this session — the generated file was read, which is content-equivalent, but the gate itself has no run receipt.
9. `nix run .#verify-fast` on the docs change (only the two cheap doc gates were run).
10. Repo-wide sweep for other stale engine counts beyond the two fixed.
11. Question-answer capture in `references/faq.md` (RocksDB refusal, turso-vs-libSQL terminology, extension policy) so the next session does not re-derive it.
12. No new ADR was written — considered and declined (the gaps doc is a reference, not a decision). The decline is only recorded here.

---

## d) TOTALLY FUCKED UP

| # | What I got wrong | Severity | Self-caught? |
| --- | --- | --- | --- |
| D1 | **Recommended turso/libSQL pushdown as "best for vector data" without disclosing that (i) the `turso-go` driver carries verified upstream IVM defects through v0.8.2** (grouped materialized views return silently wrong SUMs; deterministic COMMIT abort at ~27k view-maintained rows per process; post-abort reads through the same engine lie) **and (ii) `TestVectorSearch_LibSQLPushdown` is a known intermittent CI red** (turso-go native-lib hash-mismatch/lazy-init defect class, `TODO_LIST.md:2686`). The performance claim itself stands — the IVM defects are view-scoped, not vector-scoped, and the benchmark ran solo on a quiet machine — but I presented a flaky-harness number as steady state, which is exactly the overclaim class this repo keeps catching. | Medium | Yes, during report prep (from `docs/agents/gotchas-tooling-build.md:49-51` + TODO_LIST) |
| D2 | **Wrote a provenance citation into a new committed doc from grep output alone.** `docs/engine-coverage-and-gaps.md` cited `docs/status/archived/2026-09-13_18-35_event-query-model-truth-reconciliation-execution.md` as the Neo4j evidence before ever opening that file — the citation came from a search-result line, not a read. Fixed during report prep by opening the file: line 37 ("T8 — §12 corrected: Dgraph replaces Neo4j") and line 48 (T19 Dgraph-vs-Neo4j) confirm the claim, so the citation now stands on a read. | Medium | Yes, during report prep |
| D3 | Minor, same family: the two stale "9 engines" counts sat in files this session touched, and I initially shipped past them. Caught and fixed (A12). | Low | Yes |

No other damage: nothing was deleted, no code touched, no test or gate was left red, and the tree's only modifications are the three doc files plus this report.

---

## e) WHAT WE SHOULD IMPROVE

**On my own behaviour this session**
1. **Pair every performance recommendation with the winning path's defect status.** The turso number is real, but a reader needs "and the harness is flaky in CI, and the driver has three verified upstream defects" in the same breath (D1).
2. **Read before citing.** No citation may enter a file from a grep hit; the read is part of writing (D2).
3. **Sweep the neighbourhood of an edit.** I edited `docs/README.md` without noticing the stale count one line up (D3). A post-edit eye over the surrounding lines is cheap.
4. **State caveats once, completely, at recommendation time** — not incrementally as follow-up questions arrive (B7).
5. **Answer questions with a "where this lives" pointer** so the answer is durable even when the doc isn't written yet.

**On the state of the project (observed, not caused here)**
6. **Engine counts in prose drift from the generated roster.** `engine-capabilities.md` is gated; the digits in prose are not. Two were stale by 4 (9 vs 13) across two files.
7. **The extension-free posture is an implicit policy with no single written home.** It is derivable from the CGo-isolation rule, the degradation design, and ADR-0140/0142 — but the question "do we ever use extensions?" took four files to answer. It deserves one paragraph in the new coverage doc (or an ADR addendum).
8. **Benchmark docs carry stale environments.** The 2026-09-16 vector doc names a Go version and build tag that the repo has since removed everywhere; dated records deserve an ANNOTATE banner rather than silence.
9. **`docs/status/` has grown past the gate's own comfort threshold** (127 live reports, >10) — the link gate says so on every run, and it is now noise that hides new signal.
10. **Vector capability reporting is per-engine and honest, but there is no fleet-level view** — "who can actually do ANN?" requires reading 13 `vector.go` files. The new doc's fleet table is class-level only; a path column (go-scan vs pushdown vs native) would finish it.
11. **The "best engine" question recurs** (vector today, graph previously). A short decision table — question → authoritative doc → one-line answer — would cut the derivation cost each time.
12. **Trigger-gated roadmap rows have no revisit trigger recorded** (ANN at ~100K vectors is the only quantified one).

---

## f) Up to 50 things we should get done next

Priority: 🔥 do next · ⚡ cheap win · ◦ backlog/ROADMAP fuel

### Doc integrity & hygiene
1. 🔥 HARVEST this §f into `TODO_LIST.md`/`ROADMAP.md` (docs-health HARVEST) — do not entomb it here.
2. ⚡ Add a "revisit trigger" column to the gaps doc's missing-classes table (B1).
3. ⚡ Add an explicit "extensions: none, deliberately" section to `docs/engine-coverage-and-gaps.md` with the four-file rationale condensed (improve #7).
4. ⚡ Back-link the gaps doc from `ROADMAP.md`'s "Native ANN vector indexes" row and `metaengine/README.md` (B4).
5. ⚡ Annotate `docs/benchmarks/2026-09-16_vector-search-paths.md`: Go 1.26.x → current, jsonv2 tag is a no-op, plus a flaky-harness note for the turso leg (D1, improve #8).
6. ⚡ Verify or demote the CockroachDB PG-wire claim (B2).
7. ◦ Decide gaps-doc placement vs FEATURES.md and record the judgment (B8, c12).
8. ◦ Add the fleet census date + gate-run receipts to the doc's provenance on every touch (provenance template).
9. ◦ Give the coverage doc a per-engine **vector path column** (go-scan / SQL pushdown / native) so the class table is complete (improve #10).
10. ◦ Add the vendor/refused refusals (`RocksDB`, `Redis`) to a stable FAQ answer so the question stops being re-asked from zero.

### Vector roadmap (all still trigger-gated)
11. 🔥 Decide the ANN question (§g Q1) and record the decision in the coverage doc + ROADMAP.
12. ⚡ File the `sqliteengine` pure-Go `vector32()` option as a scoped TODO (ROADMAP's stated first candidate).
13. ⚡ Add an `ANN` capability flag to `EngineProfile` so the planner prefers ANN engines by declaration instead of cost priors alone.
14. ⚡ Add the `VectorCount` optional capability so `Doctor`/`EXPLAIN` can warn *before* a collection crosses the brute-force cliff (VECTOR-SEARCH-AT-SCALE spike proposal H).
15. ⚡ Publish a pebble/bbolt/badger vector benchmark — the 2026-09-16 doc covers only sqlite/turso/duckdb, so every LSM claim is inference today.
16. ◦ Spike DuckDB VSS (HNSW) behind the existing CGo build tag (no new CGo island required).
17. ◦ Spike an operator-opt-in pgvector engine that probes for the `vector` type and degrades honestly when absent.
18. ◦ Wire Dgraph's upstream `similar_to` + HNSW if the pinned version supports it (ADR-0140:80 lists it as a native-ANN candidate).
19. ◦ Evaluate Qdrant/Milvus as an adapter-style external ANN route instead of an in-repo engine.
20. ◦ Define the vector-at-scale trigger metric (collection size, p99, or memory) and encode it in Doctor thresholds.
21. ◦ Filtered-search cost model: document that the LSM filtered path pays a metadata read per row before scoring.
22. ◦ Consider float16/quantized payloads in `EncodeVectorBinary`, with wire-format versioning consequences spelled out.
23. ◦ Add recall/parity floors to the test suite **before** any ANN path lands (brute-force parity exists; ANN needs recall assertions).
24. ◦ Cross-engine edge audit: filtered vector search with null metadata.
25. ◦ Decide whether vector-family wire-format duality (binary vs JSON across engines) deserves a one-line WHY in `modules.md`.

### Engine coverage decisions
26. 🔥 Answer §g Q3 (demand-pulled vs proactive for ClickHouse/Scylla/document) and record it.
27. ⚡ Run `bash scripts/census-fleet.sh` and check whether any fleet consumer asks for vector/ANN or a missing class.
28. ◦ If ClickHouse proceeds: confirm `clickhouse-go` purity, dep budget, and the columnar ADT mapping.
29. ◦ If Scylla/Cassandra proceeds: define claim/lease semantics per ADR-0142 (partition-key claims) before any driver work.
30. ◦ If a graph server is ever revisited: reuse `graph.GraphDriver` + `graphtest` — no new `Engine` surface.
31. ◦ Document why Bigtable is the only cloud engine (GCP) and what a DynamoDB driver would need (claim story, dep budget).
32. ◦ Object storage: record the gate-A verdict on journal-over-S3 (durable state vs build artifact) instead of leaving it implicit.
33. ◦ `ADTSearch` exists as a capability with no engine pushdown — decide degraded-only or one reference implementation.
34. ◦ `ADTSpatial` same question (`spatial.go` exists; which engine owns pushdown).
35. ◦ Governance: if RocksDB/Redis keep coming up, promote the refusals from prose into an ADR addendum so the answer has a stable home.

### Gates, verification, hygiene
36. ⚡ Run `nix run .#check-engine-capabilities` and record the receipt (c8).
37. ⚡ Run `nix run .#verify-fast` over this docs change (only the two cheap doc gates ran).
38. ⚡ Repo-wide sweep for stale engine counts (`9 engines`, `10 engines`, `12 engines…`) and add a gate for them (improve #6).
39. ◦ Extend `check-engine-capabilities.sh` (or `check-canonical-facts.sh`) to flag prose engine counts that disagree with the generated roster.
40. ◦ Consider making `check-readme-links.sh` fail — not NOTE — when `docs/status/` exceeds N live reports.
41. ◦ Execute the `docs/status/` archive-and-index pass (127 live reports) (B6).
42. ◦ Confirm `TODO_LIST.md:2686` (libSQL pushdown intermittent) is still open and annotated with today's re-observation (D1).
43. ◦ Add a metadata-filter regression test for the pebble filtered path (filter-before-score ordering is a correctness-relevant micro-optimization).

### Process
44. ⚡ Write the three session answers into `.agents/skills/go-cqrs-lite/references/faq.md` (RocksDB refusal, Turso-vs-libSQL, extension policy) so they stop being re-derived.
45. ⚡ Add a one-line "read before cite" rule to the session checklist in `AGENTS.md`/skill docs (D2).
46. ◦ Add a short "answering engine questions" cheat sheet to the go-cqrs-lite skill: sources are `engine-capabilities.md`, the vector benchmark doc, `gotchas-tooling-build.md:51`, ADR-0140/0142.
47. ◦ Adopt a "recommendation + defect status in the same breath" rule for performance answers (D1).
48. ◦ Keep the new coverage doc on a "touched when the fleet changes" list (the engine-census gates already run; extend their checklist to include this doc).
49. ◦ Consider a tiny renderer that prints the fleet table from `register.go` for ad-hoc answers (13 engines, revisited often).
50. ◦ Record this report's `.md` override (status-report skill is HTML-canonical) so the divergence is visible, and re-read the skill's exception list before the next dispatch.

---

## g) Questions I cannot answer myself

1. **ANN acceleration: promote now, and if so which first?** Options in fleet terms: (a) pure-Go `vector32()` in sqliteengine (no new deps, expression-level scoring only — still O(N)), (b) DuckDB VSS/HNSW behind the existing CGo tag, (c) an operator-opt-in pgvector engine, (d) a dedicated vector engine (Qdrant/Milvus adapter). Each has different cost/dependency/portability consequences; ROADMAP currently says trigger-gated at ~100K vectors with (a) as the first candidate. I cannot decide whether a consumer actually needs sublinear search now.
2. **Is "no extensions" a permanent invariant or an operator-opt-in path?** Today it is enforced implicitly (probing + honest degradation). I cannot tell whether you want that formalized as a permanent rule, or whether you want a sanctioned probe-then-use-extension path (pgvector/VSS) for operators who opt in.
3. **Should the missing engine classes be built proactively or strictly demand-pulled?** ClickHouse / Scylla / document / search are named in the original vision docs but have no consumer ask. Building any one is a multi-day module effort; waiting means the gaps doc stays a wish list. I cannot judge whether the fleet has demand I cannot see (private consumers, planned apps).

**WAITING FOR INSTRUCTIONS.**
