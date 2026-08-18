# Status Report — Vector Binary Phase 0 + mysqlengine Depth-1 Graph Short-Circuit

**Date:** 2026-08-17 14:25
**Scope:** This session only (TODO_LIST items "mysqlengine: depth-1 graph short-circuit" and "🔥 Vector payload binary encoding (spike Phase 0)"). No unrelated research.
**Commit state:** All session work was captured (by the auto-commit daemon) inside `15a7cfe6d` "chore: workspace actor pin sweep, system/v4 review outcomes, scheduling actor envelope" — mixed together with a concurrent session's actor-pin/go.mod-sweep work. Working tree is now clean.

---

## a) FULLY DONE

### 1. mysqlengine depth-1 graph short-circuit (Effort: XS — done, verified live)

- `metaengine/mysqlengine/graph.go`: `GraphNeighbors(depth == 1)` routes to a new
  `mysqlGraphNeighborsDepth1` direct adjacency query (`... AND to_node <> ?` preserves
  start-node exclusion) **before** the CTE/iterative mode switch — so every server
  (MySQL 8.x, MariaDB 10.2+, and old no-CTE servers) gets the fast path.
  The composite PK `(collection, from_node, to_node)` makes UNION/DISTINCT unnecessary.
- `metaengine/mysqlengine/graph_undirected.go`: undirected twin short-circuits to the
  CTE's seed union (derived table + UNION) with start-node exclusion — provably
  equivalent at depth 1 because the CTE's recursive arm contributes zero rows there.
- Consolidated 3 hand-rolled single-query drains into `queryGraphRows(ctx, query, args...)`;
  `scanGraphRows` remains the row-drain primitive.
- Tests (`graph_internal_test.go`): extracted duplicated `sortNeighbors` closure into a
  shared helper; added `TestGraphNeighbors_Depth1ShortCircuitMatchesCTE` and the
  undirected twin — self-loop exclusion, no depth-2 leakage, CTE parity, both modes.
- **Live verification** against a relaunched userspace MariaDB 11.4.12 (datadir
  `/tmp/mariadb-cqrs`, port 33061): full mysqlengine suite GREEN, `-count=1` and
  `-race`. Depth-1 latency now ~83-133µs (was 137-253µs via CTE).
- TODO_LIST item marked done; LATENCY-MODEL §9 finding #2 updated from
  "Unimplemented; see TODO_LIST" to implemented.

### 2. Vector payload binary encoding — spike Phase 0 (Effort: M — done, measured)

- New `metaengine/vector_binary.go` (+`vector_binary_test.go`): exported
  `EncodeVectorBinary` / `DecodeVectorBinary` / `DecodeVectorAuto`. Wire format:
  `'b' | dim uint32 LE | dim × float32 LE`. `'b'` can never begin a JSON text, so
  the format sniff is unambiguous; the dim header makes torn writes detectable
  (payload-length check with uint64 arithmetic to stay 32-bit-safe).
- Engines switched (write binary, read via `DecodeVectorAuto` so legacy JSON rows
  keep decoding): **pebbleengine**, **bboltengine**, **badgerengine** —
  `VectorInsert` + `VectorSearch` + `VectorSearchFiltered` in each.
- **pgengine deliberately stays JSON** — its vector column is typed JSONB; binary
  needs a BYTEA DDL migration. Documented in code (pgengine/vector.go comment),
  CHANGELOG, TODO_LIST, and spike doc §4. This was my judgment call on scope.
- Tests: codec unit tests (bit-exact roundtrip incl. NaN/±Inf/-0, wire layout pin,
  marker-vs-JSON collision, malformed payloads, legacy-JSON readability incl. the
  `null` a nil slice produces, cross-format equivalence at D=128); per-engine
  `TestVectorSearch_LegacyJSONPayloadReadable` in all three KV engines seeding raw
  legacy JSON rows and asserting mixed-format collections search correctly.
- **Measured win** (LSM validation bench, D=128, k=10, cosine, 20x):
  pebble VectorSearch **15.9ms → 459.9µs (1K)**, **172.2ms → 5.63ms (10K)** —
  ~31-35x, ~460-560ns/vector vs the same-day in-RAM ceiling of ~80-95ns/vector.
  LSM now within ~6x of the ceiling instead of ~190x. Exactly the spike's prediction
  that the 190x constant was JSON decode, not the scan.
- Docs/process: api-stability golden regenerated (+3 exports, meta-tests GREEN);
  spike doc updated (§2 post-Phase-0 table, §4 implemented note, §5 phase row,
  §7 Q1 answered: value-level marker, not a new key family); CHANGELOG
  `[Unreleased]` section added; `check-changelog-symbols` (23 citations) and
  `doc-check` (916 refs) both GREEN.
- Gates: build+vet+lint+test+race GREEN on all 5 touched modules; workspace-wide
  build EXIT=0; `#check-duplication` **0 new clones** (baseline 111 groups);
  `#verify-fast`: 119 modules ok.

### 3. Session side-quests (small, done)

- Relaunched the userspace MariaDB from the prior session's persisted datadir and
  recorded the exact relaunch recipe in AGENTS.md (Testing section).
- Perfsprint finding fixed (`errors.New` sentinel instead of format-less `fmt.Errorf`).
- Formatter failure on `system/journal_error_test.go` correctly identified as a
  concurrent session's mid-write corruption and left alone (they fixed it).

---

## b) PARTIALLY DONE

- **Bench coverage is pebble-only.** bbolt and badger also switched to binary, but I
  measured only pebble (`BenchmarkPebbleVectorSearch_*`). bbolt has no vector bench
  file at all; badger's state unverified this session. The ~31-35x claim is strictly
  a pebble number.
- **`VectorSearchFiltered` not re-measured** — it benefits identically (same decode
  path) but no number exists.
- **Verification stopped at `#verify-fast`** (per the documented rule that a full
  `#verify` must run exclusively). Three benchkit timing tests (`TestRun_ClosedStore`,
  `TestRun_ClosedStore_ErrorMessage`, `TestRun_ReplayOnly_SQLite`) flaked under the
  parallel gate load and all passed in isolation — the pre-documented load-sensitive
  class, unrelated to this diff — but a clean exclusive full `#verify` has NOT been run
  since these changes.
- **Depth-1 benchmark semantics degraded (see d):** the forced-mode graph benches no
  longer measure what their labels say at depth 1.

---

## c) NOT STARTED

- **Phase 1 — int8 scalar quantization + exact re-rank** (deliberate: spike says only
  if post-Phase-0 p99 still above budget at real N).
- **Phase 2 — optional ANN (HNSW/IVF) with filter-aware fallback** (deliberate deferral).
- **pgengine BYTEA migration** (deliberate scope cut, documented).
- **`VectorCount` optional capability + Doctor/EXPLAIN WARN advisory** (spike §5
  "cheap to add anytime" row — untouched).
- **Re-pinning `#check-duplication` baseline** — not needed (0 new clones), noted for
  completeness only.
- **`#load-sweep` before verify** — AGENTS.md prescribes it after touching timing
  paths; my changes touch perf paths but not the `Latency|Timer|Deadline`
  timing-assertion suites. Skipped on that reading; a stricter reading would have run it.

---

## d) TOTALLY FUCKED UP (honest ledger)

1. **The depth-1 forced-mode benchmarks are now mislabeled — and my §9 wording leans
   on the mislabel.** `runGraphNeighborsBench` forces `e.graphCTE = useCTE`, but the
   short-circuit checks `depth == 1` BEFORE consulting `graphCTE`, so
   `BenchmarkGraphNeighbors_CTE/size-X/depth-1` no longer executes the CTE at all —
   both forced modes now run the identical direct query. My post-change "measurement"
   ("both forced modes converge to ~83-133µs") and the LATENCY-MODEL §9 sentence I
   wrote ("re-verified: both forced modes converge") are technically true but
   **misleading**: they converge because they execute the same SQL now, not because
   the CTE improved. The honest comparison (short-circuit vs actual CTE at depth 1)
   requires calling `graphNeighborsCTE` directly. Unfixed.
2. **Cheap self-inflicted roundtrips (caught instantly, no damage):** first test
   compile failed on `float64`→`float32` literals (`math.Inf` etc.); lint failed on
   perfsprint; one `edit` hit a stale-read because the formatter had re-wrapped the
   file between my read and write. Each cost one cycle; all caught by gates.
3. **Minor convention deviation:** AGENTS.md prescribes the gosec-G115
   helper-isolation pattern for integer conversions; I used an inline
   `//nolint:gosec` with justification instead (single short line, but not the
   documented house pattern).

---

## e) WHAT WE SHOULD IMPROVE

- **Bench honesty when adding dispatch short-circuits:** any forced-mode bench must
  either bypass the production dispatch (call the internal method directly) or skip
  the short-circuited configuration — otherwise labels lie. This bit both the mysql
  graph bench and my §9 wording; a general lesson for the engine benches.
- **Measure every engine you switch, not one representative.** The spike's claim is
  about the LSM family; one data point (pebble) is evidence, three is the claim.
- **A codec-level micro-bench is missing:** `DecodeVectorBinary` vs
  `DecodeVectorJSON` at D=128/1536 would pin the pure decode cost independently of
  any engine, making future format work (int8) comparable without LSM noise.
- **Test-collection hygiene on the shared MariaDB datadir:** engine tests use
  per-run-scoped collection names that accumulate forever in `/tmp/mariadb-cqrs/data`;
  nothing prunes them.
- **Daemon commit mixing:** this session's coherent feature work landed inside a
  commit named for a different workstream ("actor pin sweep"). Nothing broken, but
  archaeology suffers; consider smaller commit cadence or session-scoped staging.
- **gofmt-dirty `system/system_test.go`** observed mid-session (concurrent session's
  tree) — resolved by them; a `check-staged-go`-style gate on ALL changed files
  (not just staged) would have caught the earlier `journal_error_test.go` breakage
  sooner.

---

## f) NEXT — prioritized backlog from this session (not researched beyond it)

**Directly from this session's work:**

1. Fix the forced-mode graph bench: route forced CTE depth-1 through
   `graphNeighborsCTE` directly (or skip depth-1 when forcing); re-measure and append
   honest numbers to LATENCY-MODEL §9.
2. Correct the §9 sentence + CHANGELOG wording so "convergence" isn't attributed to
   mode parity (see d-1).
3. Add `vector_bench_test.go` for bboltengine (mirroring pebble's) and run badger's —
   get all three LSM engines measured post-Phase-0.
4. Add `BenchmarkVectorSearchFiltered` (pebble) — filtered k-NN win number.
5. Add codec micro-bench: `DecodeVectorBinary` vs `DecodeVectorJSON`, D=128/1536,
   plus encode benches.
6. Re-run full exclusive `nix run .#verify` (and `#load-sweep` first) — the session
   only banked `#verify-fast`.
7. Replace inline `//nolint:gosec` in `EncodeVectorBinary` with the house
   helper-isolation G115 pattern.
8. Decide + possibly implement `VectorCount` capability + Doctor WARN (spike §5
   cheap advisory) — closes the "operator can see degraded collections" loop.
9. Evaluate Phase 1 (int8) trigger honestly: with pebble at ~460-560ns/vector, is p99
   still above budget at real N? If yes, spec int8 per spike §4-B.
10. pgengine BYTEA design sketch (dual-read migration) — only if PG vector scan cost
    ever matters in practice; keep documented deferral otherwise.
11. Prune `/tmp/mariadb-cqrs/data` test collections (DROP/recreate `cqrs_test`) or add
    a cleanup helper to the relaunch recipe.
12. Check whether `benchmark-regression.sh`'s stored medians reference
    `BenchmarkGraphNeighbors_CTE/depth-1` or `BenchmarkPebbleVectorSearch_*` — big
    improvements changed those series' meaning/values; re-pin if the gate keys on them.

**Session-observed, pre-existing (not investigated further):**
13. benchkit timing tests flake under parallel gate load (3 tests, this session) —
isolate/quarantine or relax bounds under load (already a known class).
14. `[BLOCKED] nix run .#integration-mysql-nspawn` still needs root — unchanged.
15. TODO_LIST F46 (go-codec tagging shifting workspace alloc pins) — still open.
16. `/mnt/buildcache` corruption status — caches still redirected to /tmp per AGENTS.md.
17. go.mod version-drift sweep across ~40 modules (id/v4 v4.4.0→v4.5.0 etc.) landed
via the concurrent session — verify it was intentional and tag-complete
(`#vulncheck` / version-sequence check before next release).

**Backlog beyond f-17:** nothing else was observed this session; the existing
TODO_LIST.md remains the authoritative backlog (cqrs-lint section, F-items,
metaengine live-latency P2/P3 items — untouched by this session).

---

## g) QUESTIONS (cannot be answered from the repo)

1. **Was the concurrent go.mod version-drift sweep sanctioned?** ~40 modules' go.mod
   got dependency bumps (e.g. `id/v4 v4.4.0 → v4.5.0`) plus untracked test files in
   `event/`, `command/`, `scheduling/` during this session, all committed together
   with my work. I left them untouched per the "respect existing changes" rule — but
   whether they should ship needs your call (tag existence/sequence checks not run).
2. **Should Phase 0 be called done with pebble-only measurement, or do you want
   bbolt/badger numbers (f-3) before ticking it fully verified in TODO_LIST?** I
   marked it done on the strength of shared-code + shared-format + per-engine
   correctness tests; the perf claim is single-engine.
3. **For LATENCY-MODEL §9:** append a corrected short-circuit-vs-raw-CTE depth-1
   measurement now (f-1/f-2), or leave the table as the historical pre-change record
   and only fix the wording? (Append-only keeps provenance; re-measuring keeps it
   current — both defensible, your preference.)

---

## h) FOLLOW-UP EXECUTED (2026-08-17, later session — backlog items f-1..f-7, f-11, f-12)

All actionable follow-up items from §f executed and verified:

1. **f-1 FIXED — forced-mode graph bench now honest.** The forced-CTE arm of
   `runGraphNeighborsBench` calls `e.graphNeighborsCTE` directly, bypassing
   the depth-1 dispatch short-circuit (documented as the "honesty rule" in
   the file header). The forced-iterative arm keeps the public entry point
   (a no-CTE server takes the short-circuit at depth 1 by design).
2. **f-1/f-2 MEASURED — the "convergence" claim was indeed false.** Honest
   depth-1 numbers (MariaDB 11.4, 20x, 1k-100k graphs): short-circuit
   53-59µs vs TRUE CTE 94-129µs — **1.6-2.4x win** on this run, consistent
   with the original §9 table's 2-4x. §9 finding #2 and the CHANGELOG entry
   now carry these numbers and note the earlier convergence observation as a
   bench artifact. Historical §9 table left intact (append-only provenance —
   superset-safe under open question 3).
3. **f-3 DONE — all three LSM engines measured** (new
   `bboltengine/vector_bench_test.go` + `badgerengine/vector_bench_test.go`,
   `art-dupl:accept`-annotated mirrors): pebble 457.4µs/5.23ms, bbolt
   425.7µs/5.79ms, badger 646.7µs/5.85ms (1K/10K). Same ~430-650ns/vector
   band — the win is the shared format, not a pebble artifact. Spike doc §2
   follow-up table added.
4. **f-4 DONE — `BenchmarkPebbleVectorSearchFiltered_1K`:** 1034.5µs vs
   457.4µs unfiltered (half the collection matching) — per-row metadata
   read + filter eval ≈ 2.2x over the bare scan.
5. **f-5 DONE — codec micro-bench** (`metaengine/vector_binary_bench_test.go`):
   decode 196ns binary vs 8.51µs JSON @D=128 (**43x**), 1.81µs vs 110.1µs
   @D=1536 (**61x**); 1 alloc vs 8-13. Encode: 152ns/1.4µs, 1 alloc.
6. **f-7 DONE — gosec G115 house pattern applied** (`vectorBinaryDim` helper,
   mirroring `transport/grpc/event_version.go`); metaengine lint clean.
7. **f-11 DONE — MariaDB test hygiene:** `cqrs_test` dropped/recreated (11
   accumulated tables pruned); `cqrs` TCP user re-provisioned; full
   mysqlengine suite green (31 top-level tests, 0 skips) + honest benches.
8. **f-12 CHECKED — no re-pin needed:** `benchmark-regression.sh` gate set is
   `BenchmarkFullPipeline_Memory|BenchmarkBenchkitSuite_Memory$` (stack/bench);
   baseline contains 0 references to GraphNeighbors/VectorSearch series.
9. **Q1 (go.mod sweep) ANSWERED BY EVENTS:** the concurrent session's own
   CHANGELOG entry ("stale id/v4.4.0 pins silently dropped ActorID in CBOR —
   all 59 modules bumped to v4.5.0") documents the sweep as an intentional
   bugfix. No action needed from this workstream.
10. **Open questions 2 and 3 remain user-gated** but are now superset-safe:
    the three-engine numbers exist (satisfies the strictest Phase-0 reading),
    and §9 keeps its historical table plus an honest dated addendum.

**Remaining from this follow-up:** full exclusive `#load-sweep` + `#verify`
(see tracked todos) — everything else banked.
