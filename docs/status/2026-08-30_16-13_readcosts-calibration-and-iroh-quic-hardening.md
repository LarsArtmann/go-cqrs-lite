# Status: ReadCosts Calibration (badger/bbolt/pebble) + iroh QUIC Test Hardening

> Session date: 2026-08-30, ended ~16:15 CEST
> Work captured by auto-commit daemon as `d1884e158` ("feat(metaengine): calibrate badger/bbolt/pebble ReadCosts, remove NsPerRead")
> Gate at session end: `nix run .#verify-fast` EXIT=0 (build + vet + test-short + race-short + lint + arch + depguard + docserver-css + duplication + coverage + api-stability), doc-check ✓ (938 refs), check-changelog-symbols ✓ (120 citations), check-duplication ✓ (0 new clones).

---

## Summary

Two TODO_LIST items executed to done:

1. **Engine per-pattern `ReadCosts` calibration** — badger/bbolt/pebble migrated off the deprecated `NsPerRead` scalar onto 4-field `ReadCosts`, with NEW per-pattern calibration benches (FilteredScan / CounterScan / FullScan) run 3x each in a moderate-load window (load ~3.8, medians taken). The `.golangci.yml` SA1019 exclusion for `EngineProfile).NsPerRead` is DELETED (zero internal uses remain). Exported constants kept (api golden untouched) and re-anchored to fresh medians: bbolt 1500→750 (old estimate ~2x conservative), pebble 1300→700, badger 1200→1100.
2. **iroh QUIC test hardening** — verified 4 of 5 sub-items had ALREADY landed (concurrent session): `normalizeAny` table tests, pooled-evict error-injection + `TestQuicPooledThousandOps`, framing-constant dedup (both transports alias `irohengine.FrameHeaderSize`/`ErrFrameTooLarge`), `WithStreamPooling()` README entry. Added the one missing piece: `TestRing_ProductionCapacity10K` in dedup (production capacity 10_000 per `quic/transport.go`, 30K adds, pins bounded Len + exact eviction window + graceful re-add).

Ledger updated: TODO_LIST.md (both items [✓] with full detail), CHANGELOG.md (2 new dated sections), AGENTS.md (KV-engine ReadCosts semantics gotcha), FEATURES.md (cost-model row refreshed — caught stale during this retro, fixed post-gate).

Measured constants now shipped (point-lookup / filtered-scan / aggregate / scan, ns):

| Engine | PointLookup | FilteredScan | Aggregate (CounterGet) | Scan | Benches |
|---|---|---|---|---|---|
| badger | 1100 (was 1200) | 650 | 165 | 630 | `BenchmarkCalibration_Badger_{FilteredScan,CounterScan,FullScan}` |
| bbolt | 750 (was 1500) | 620 | 100 | 660 | `BenchmarkCalibration_Bbolt_*` |
| pebble | 700 (was 1300) | 830 | 125 | 700 | `BenchmarkCalibration_Pebble_*` (scan paths drive `ScanRawValues`, its real `executeFilteredScan` route) |

Raw bench runs preserved at `/home/lars/projects/.gotmp/bench-{badger,bbolt,pebble}.log` (volatile — not committed; see improvement item E2).

---

## a) FULLY DONE

| # | Item | Evidence |
|---|---|---|
| A1 | 9 new calibration benches (3 engines × 3 patterns) with `rows-scanned` metrics, doc headers mapping bench→ReadCosts field, run commands | `metaengine/{badger,bbolt,pebble}engine/calibration_bench_test.go` |
| A2 | Quiet-window bench campaign, 3 counts each, medians; ambient load recorded in code comments + CHANGELOG | load avg 3.75–4.37 at 15:15; runs took medians |
| A3 | `Profile()` migration in all three engines: `NsPerRead:` line removed, 4-field `ReadCosts` added with per-field bench citations | `badgerengine/engine.go:134`, `bboltengine/engine.go:189`, `pebbleengine/engine.go:180` |
| A4 | Constants re-anchored and kept as single source of truth for point-lookup cost (`NsPerPointLookup: XxxNsPerRead`) | constant doc comments updated with re-calibration dates |
| A5 | SA1019 exclusion deleted from `.golangci.yml`; residual-use grep clean | exclusion block was lines 423-426; `rg "\.NsPerRead"` in the 3 modules = 0 hits |
| A6 | Per-module verification: badger/pebble/bbolt test suites GREEN (bbolt required documented `SOAK_SKIP_BOLT=1`), dedup GREEN, metaengine core GREEN | `/home/lars/projects/.gotmp/test-*.log` |
| A7 | Scoped golangci-lint GREEN on all 4 touched modules (after gofumpt alignment fix) | `lint3-*.log`: "0 issues" ×2, badger/dedup first pass 0 |
| A8 | Full `nix run .#verify-fast` EXIT=0 — all 16 stages | `verify-fast.log` |
| A9 | doc-check re-run AFTER the late AGENTS.md edit: 938 references valid | `doccheck.log` |
| A10 | iroh items 1/3/4/5 verified as landed by inspection (file content read, not trusted from TODO text) | `latency_internal_test.go`, `pool_evict_internal_test.go`, `pool_stress_test.go`, `loopback/frame.go`, `quic/frame.go`, `quic/README.md:135` |
| A11 | `TestRing_ProductionCapacity10K` added (30K adds @ cap 10K: Len bounded, evicted range exact, in-window all present, graceful re-add) — first run caught MY OWN off-by-one expectation, fixed, suite green | `dedup/ring_test.go` |
| A12 | CHANGELOG sections (Added + Changed, 2026-08-30) — 120 `pkg.Symbol` citations verified honest by gate | `changelog-syms.log` |
| A13 | Consumer builds: `stack/pebble`, `metaengine/bench`, `system` — all build GREEN against the changed constants | build matrix run |
| A14 | AGENTS.md gotcha added (KV-engine ReadCosts semantics: no pushdown → filtered≈scan; ReadAggregate=CounterGet; don't re-anchor via SA1019) | AGENTS.md, "Module & Dependency Management" |
| A15 | TODO_LIST items marked [✓] with honest DONE-verification notes (iroh: 4-of-5 pre-landed, inspection-verified) | TODO_LIST.md lines ~36-42 |

## b) PARTIALLY DONE

| # | Item | What's missing |
|---|---|---|
| B1 | **iroh QUIC hardening is inspection-verified, not execution-verified** — the 4 pre-landed items' tests (`//go:build cgo`, need Rust toolchain + iroh-go) were NOT executed in this session. Code was read and confirmed to pin the claimed behavior; nothing was run. | A CGo-enabled run of the quic suite (needs Rust/iroh-go environment) |
| B2 | **"No planner routing flip" is an absence-of-evidence claim** — no test in the repo pins cross-engine routing decisions against REAL badger/bbolt/pebble profiles (core metaengine tests use synthetic profiles). verify-fast green + core green says the math didn't break, not that no Store anywhere now picks a different engine. | A routing regression test constructing multi-engine Stores with the real profiles |
| B3 | **Bench statistical rigor is median-of-3 @ benchtime=1s** — bbolt FilteredScan showed a ~30% cold first run (8.48ms → 5.94–6.14ms); the 620 constant could be a few % high. Acceptable per repo precedent (pg/duckdb used single runs), below the honesty bar I'd set for a perf-sensitive product. | Warmup pass + `-count=5` re-run on a truly quiet window; numbers archived under `docs/benchmarks/` |
| B4 | **"ALL 7 engines now set ReadCosts" is roster-narrow** — true for pg/mysql/dgraph/duckdb/badger/bbolt/pebble (the TODO's roster), but `sqliteengine` (and memory/turso/iroh passthrough) do NOT set ReadCosts. SQLite is the highest-traffic embedded SQL engine in the repo and still falls back to the scalar. The FEATURES row wording could mislead a reader into thinking the engine roster is exhausted. | sqliteengine calibration (has PushdownMapScan + aggregations, benches are straightforward); wording tweak or engine-count honesty pass |
| B5 | **`docs/benchmarks/` baseline not written** — AGENTS requires protocol benchmarks run from workspace root with load recorded; I recorded load inline in code comments but did not file a baseline doc for the new calibration numbers. | `docs/benchmarks/calibration-2026-08-30.md` with raw runs |
| B6 | **dedup↔quic capacity coupling is comment-only** — `quic/transport.go` hardcodes `dedup.NewRing(10000)`; the new test pins 10_000 with a comment pointing at the quic file. If quic changes its capacity, nothing mechanical fails. | Named/exported constant or a meta-test |

## c) NOT STARTED (observed this session, deliberately out of scope)

| # | Item | Why not started |
|---|---|---|
| C1 | ReadAggregate semantic reconciliation for SQL engines — `execute.go:186` routes `ReadAggregate` → `CounterGet` for EVERY engine, yet pg/mysql/duckdb calibrate `NsPerAggregate` against SQL SUM-over-map-rows (AggregateReader workloads), not CounterGet. Either a latent miscalibration or an undocumented intentional modeling choice. Needs a design decision before touching constants. | Ambiguous business intent — flagged as question G1 |
| C2 | Tag wave for the constant changes (badger/bbolt/pebble + dependents) + consumer pin sweep | Behavior-affecting values; release decision is user's (see G2) |
| C3 | Tombstone consumer migration (storage/listing/watermill/stack-sqlite) → then delete the tombstone SA1019 exclusion block | Concurrent sessions' declared work; untouched |
| C4 | `nix run .#verify` (full, non-fast, with soaks) before the next release train | verify-fast was the proportionate gate for this change set |
| C5 | Turso CTE-probe test + calibration-vs-baseline run (C2 items from the 2026-08-29 Pareto closeout plan) | Separate planned work, not mine this session |
| C6 | iroh replicated `Profile()` honesty (replication overhead not reflected in cost fields — old archived review finding) | Not in scope of either TODO item |
| C7 | CI drift job running the new calibration benches on schedule and diffing against shipped constants | New idea from this session, needs a design pass |

## d) TOTALLY FUCKED UP (own failures, no varnish)

| # | Failure | Cost | Root cause |
|---|---|---|---|
| D1 | **My regression test shipped with an off-by-one**: first run of `TestRing_ProductionCapacity10K` FAILED — I asserted `2*capacity` ("20000") was evicted when it is exactly the oldest in-window ID. Caught by running the test (the failure mode worked as designed), fixed to `2*capacity-1`. | 1 cycle | Wrote boundary assertions from memory instead of deriving them on paper first |
| D2 | **bbolt full suite 10-minute timeout** — ran `go test ./...` without the documented `SOAK_SKIP_BOLT=1`; hit the 600s package timeout, wasted a full cycle. | 10 min | AGENTS documents this exact gotcha; I knew it and reached for it only AFTER the failure. Process failure, not knowledge failure |
| D3 | **Dropped the cache env chain on the gofumpt re-lint** — second lint run lacked `GOLANGCI_LINT_CACHE`/`GOCACHE`, failed initializing build cache at the dead `/mnt/buildcache`. The AGENTS gotcha (phantom failures after ENOSPC; golangci derives cache from GOCACHE's parent) describes this verbatim. | 1 round trip | Composed the "quick re-check" command lazily instead of reusing the mandatory env prefix |
| D4 | **Edit-without-viewing**: the bbolt constant comment multiedit guessed text that didn't exist ("Calibrated 2026-08-05...") — 1 of 2 edits failed, re-read, re-applied. I HAD viewed badger's constants but not bbolt's, and pattern-matched instead. | 1 round trip | Violated read-before-edit for the one file I hadn't fully inspected |
| D5 | **Edited AGENTS.md while verify-fast was mid-run** — the gate's docs stage didn't cover my edit; had to re-run doc-check manually afterward. Sequencing error, self-caught and compensated. | 1 extra command | Started the long gate before finishing doc edits |
| D6 | **Missed FEATURES.md staleness at session close** — my own migration made the FEATURES cost-model row stale; the `verify-fast` doc gates don't check FEATURES-vs-code drift, and I didn't sweep living docs either. Caught only during THIS retro (fixed post-gate, uncommitted-to-gate). | Doc drift window | docs-health mandate not executed as a closing step |

## e) WHAT WE SHOULD IMPROVE

1. **Bench protocol**: adopt a written calibration protocol — warmup run (discarded), `-count=5` medians, ambient load + commit recorded, raw output filed under `docs/benchmarks/`. Constant-value edits deserve the same rigor as timing-threshold edits (which already have the ×3 race rule).
2. **Pre-flight checklist for engine test suites**: before running any `go test ./...` in a module, grep the module for soak/skip env vars (`SOAK_SKIP_*`) — mechanical, would have saved D2.
3. **Env-chain discipline**: the mandatory cache env prefix belongs in every command verbatim, including "quick" ones — consider a shell function or a note that any golangci/go command without the full chain is invalid output (D3).
4. **Living-docs closing sweep**: after API/behavior-adjacent changes, sweep FEATURES.md/README cost tables alongside TODO_LIST/CHANGELOG — the CHANGELOG gates symbols but nothing gates FEATURES drift (D6). Candidate: extend a docs gate to grep FEATURES claims against source.
5. **Routing-flip coverage**: multi-engine routing tests with real engine profiles would convert "probably no flip" into "pinned" (B2).
6. **README cost tables**: badger/bbolt/pebble READMEs have capability tables but no per-pattern cost table citing the benches — cheap consumer-facing honesty.

## f) NEXT ITEMS (up to 50, prioritized; 🔥 = direct fallout of this session)

**Direct follow-ups (this session's work, highest leverage)**
1. 🔥 Decide C1: recalibrate pg/mysql/duckdb `NsPerAggregate` onto CounterGet (the actual ReadAggregate path) or document SQL SUM as the intentional model — then align constants + benches + AGENTS wording.
2. 🔥 Add multi-engine routing regression test with real badger/bbolt/pebble profiles (close B2).
3. 🔥 Calibrate `sqliteengine` ReadCosts (point lookup via PK, PushdownMapScan filtered scan, SQL SUM aggregate, full scan) — the biggest engine still on the scalar fallback (B4).
4. 🔥 Execute the iroh QUIC CGo suite (Rust/iroh-go env) — convert inspection-verified to execution-verified (B1).
5. 🔥 File `docs/benchmarks/calibration-2026-08-30.md` with raw runs + protocol notes (B5).
6. 🔥 Re-run bbolt FilteredScan with warmup + `-count=5` on a quiet window; adjust 620 if steady-state differs (B3).
7. 🔥 CI drift job: run the 7 engines' calibration benches on a schedule, diff vs shipped constants, warn on >25% drift (mirrors benchmark-regression gate).
8. 🔥 FEATURES.md wording pass: name the ReadCosts engine roster explicitly (avoid "ALL 7" ambiguity) + note sqlite gap until item 3 lands.
9. 🔥 Unify dedup↔quic capacity: exported constant (e.g. `quic.DefaultDedupCapacity`) or meta-test so the 10_000 coupling can't rot (B6).
10. 🔥 Add per-pattern cost tables to badger/bbolt/pebble READMEs citing the new benches.

**Calibration/planner deepening**
11. Calibrate memory engine ReadCosts (test-tier but used in dozens of planner tests — honest priors help).
12. Turso/iroh ReadCosts honesty: turso is RTT-dominated (per-row costs near-meaningless — document or zero deliberately); iroh passthrough should inherit local engine's ReadCosts, verify it does.
13. Per-pattern live trackers (archived backlog item 24): `NsPerPointLookup` stays compile-time while scalar gets EWMA — extend tracker to per-pattern map.
14. CheckRouting drift test: change ReadCosts via `SetCalibration` and assert the routing signature changes trigger a drift diagnostic (signature includes ReadCosts since the fix — is that behavior tested? verify).
15. EXPLAIN/Doctor: display each ReadCosts field's provenance (compile-time prior vs runtime calibration vs live EWMA).
16. Consider `NsPerScan` network-amortization interaction: KV engines have RTT=0 so scan fallback math is trivial, but document the invariant that embedded engines must never set `NetworkRTT`.
17. Extend `benchmark-regression.sh` gate to cover the new calibration benches (median ns/op, 25% threshold) so constant drift fails CI.

**Inherited pending (observed in TODO_LIST/status docs this session)**
18. Tombstone consumer migration (storage/listing/watermill/stack-sqlite), then delete the tombstone SA1019 exclusion block.
19. Tag wave for badger/bbolt/pebble constants + consumer pin sweep (G2 decides timing).
20. go-codec tagging + event alloc-expectation updates (TODO F46 — workspace-vs-published alloc drift).
21. Turso CTE-probe test (Pareto C2 leftover).
22. Calibration-vs-baseline run comparing shipped constants across engines (Pareto C2 leftover).
23. dgraph depth-bound parity: re-verify `TestDgraphADTMatrix/GraphDepthBound` post-recurse-fix (AGENTS says fixed 2026-08-30; TODO_LIST line 32 still says REMAINING — reconcile the ledger).
24. v5 train: stack preset removals per ADR-0123 (Bundle/Materialize/RunProjections), transport/http+grpc removal, graph module removal.
25. v5 Deprecated-shell removals from AGENTS contract 16 (schema.VersionedStore, signing.Rejecting*, encryption.ErrInnerStoreNot*, metadata.CustomData).
26. Drop middleware/encryption/signing sibling replaces once metadata/event tags carry the unpublished symbols (AGENTS Module & Dependency note).
27. ADR-0114 completion: migrate `stack.Materialize.OnTombstone/OnRebirth` off metadata-triggered tombstones to event-type branching.
28. commandlifecycle/projections: confirm DLQ/retry/failure-log projections all read canonical Cause (spot-check after the record-tier waves).
29. kvstore SA1019 permanent exclusion: copy the rationale into V5-OUTLINE §exclusions (outline references it; verify the section is written, not just planned).
30. Full `nix run .#verify` + `nix run .#vulncheck` before the next release train (stale-GREEN prophylaxis).

**Hygiene/tooling (small, from this session's friction)**
31. `scripts/check-engine-bench-protocol.sh`-style tripwire: fail CI if a calibration constant changes without a same-commit bench-log reference (or CHANGELOG citation).
32. Make bbolt AutoCRUD soak duration-bounded (configurable cap) so plain `go test ./...` doesn't need env-var knowledge (D2's class).
33. Module READMEs: add "Calibration" section template (bench names + date + load) so re-calibrations have a fixed home.
34. Pre-commit/docs gate candidate: grep FEATURES.md table claims for `NsPerRead`-style renamed symbols (would have caught D6 mechanically).
35. `SOAK_SKIP_*` discovery: add a `nix run .#test-fast` alias that sets all soak skips + short mode in one place.
36. Archive the raw bench logs pattern: `docs/benchmarks/` naming convention (`calibration-<engine>-<date>.log`) so re-calibration comparisons are diffs not archaeology.
37. Update `references/modules.md` (skill) if engine cost tables are added to module READMEs (keep SKILL in sync per AGENTS rule).
38. Add `TestEveryEngineSetsReadCosts` meta-test: enumerate engine modules, assert `Profile().ReadCosts` is fully set (fails for sqlite until item 3 — write it to document the expected roster).
39. Consider a tiny `benchkit` factory preset for "calibration bench" so new engines get the 3-pattern bench scaffold for free (pattern parity with enginetest).
40. Dedup package: consider renaming `TestRing_LargeCapacity_Wraparound` (1024) vs the new 10K test to make the capacity taxonomy obvious (1024=default, 10000=quic).

**Bigger swings (needs planning, not this session)**
41. Live-latency Phase-4: replace compile-time ReadCosts for networked engines with probe-derived per-pattern costs (METAENGINE-LIVE-LATENCY-MODEL's stated direction).
42. Cost-model honesty doc: single page mapping every planner cost field → its bench → its execution path (would have surfaced C1 immediately).
43. Routing decision telemetry: log when ReadCosts changes flip an assignment (observability for constant recalibrations).
44. Revisit `filterSelectivity` (unused for routing by design) — either remove or wire as diagnostic-only output; it's dead-adjacent weight in cost.go.
45. Engine conformance: extend `adttest.RunCapabilityConformance` with "profile completeness" checks (Supports/Degraded/ReadCosts coherence).

**Ledger/meta**
46. Reconcile TODO_LIST line 32 (dgraph REMAINING claim) with the 2026-08-30 recurse fix — annotate as re-verify-then-close (docs-health HARVEST).
47. Add this session's 3 questions (G1–G3) to ROADMAP "Open Questions".
48. Mark items 1–10 of this list into TODO_LIST.md's Metaengine section (they're actionable now, not vague).
49. After item 3 (sqlite) lands, update FEATURES row roster + AGENTS gotcha to "all embedded SQL + KV engines".
50. Schedule the next quiet-window bench day (Pareto C2 pattern) for: items 3, 6, 13, 17, 22 — one session, one environment, comparable numbers.

## g) QUESTIONS (cannot self-answer)

**G1 — ReadAggregate semantics for SQL engines.** `execute.go` routes `ReadAggregate` → `CounterGet` for every engine, but pg/mysql/duckdb calibrate `NsPerAggregate` against SQL SUM over map rows. Is the SQL-SUM calibration intentional (modeling AggregateReader/typed aggregate workloads I shouldn't touch), or a latent miscalibration I should fix by recalibrating those three onto CounterGet like the KV engines? This decides whether follow-up 1 is a constants change or a docs-only clarification.

**G2 — Release timing for the constant changes.** The badger/bbolt/pebble profiles now route differently (point-lookup scalars moved 2x for bbolt/pebble). Cut engine-module tags + consumer pin sweep NOW so published consumers get honest routing priors, or hold all constant recalibrations for the v5 train? (Second-order: should published `stack/*` presets that embed these engines re-tag in the same wave?)

**G3 — iroh CGo verification depth.** The 4 already-landed QUIC hardening items are inspection-verified only (their tests need the Rust toolchain + iroh-go CGo build, Linux). Do you want me to set up that toolchain and actually execute the quic suite (`TestQuicPooledThousandOps`, `TestEvictPooledStream_ReopenOnNextSend`, etc.) — accepting the environment work — or is code-reading verification acceptable for items whose pinned tests exist in-repo?

---

**Standing at session end:** `d1884e158` committed, working tree clean except the post-retro FEATURES.md fix. No branch, no push, no tags (per rules). Waiting for instructions.
