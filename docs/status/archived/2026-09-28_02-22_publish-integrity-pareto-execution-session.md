# Publish-Integrity Pareto Plan — Execution Session Status

> **RESOLVED-BY-ROUTING (docs-health 12th pass, 2026-09-28):** Publish-integrity session 1 — M1-M13 receipts rowed in TODO_LIST (stalled-train M1, surgery M2, race M3, junk M6, counts M7, placements M8/M9, flake M10, ADR-0149 M11, README note M12); M14 finished + M19 filed by session 2 (05-40); M15 rows added; M16 gates green; M22 owner Q3 is now a TODO row. Open: M5+M20 (armed quiet campaign — TODO calibration row + plan §8 addendum), M22, and the owner mechanics (train completion, tursoengine v4.2.1).

**Written:** 2026-09-28 02:22 CEST
**Scope:** THIS SESSION ONLY — execution of [`docs/planning/2026-09-28_01-26_SUPERB-publish-integrity-pareto-plan.md`](../planning/2026-09-28_01-26_SUPERB-publish-integrity-pareto-plan.md) (M-tasks), stopped mid-M14 on the user's report demand. Prior-session work (the plan itself, the QA session) is context, not inventory here.
**Format note:** Markdown per explicit user instruction (skill default is HTML dashboard; override flagged, not propagated).

---

## a) FULLY DONE

### M1.1 — Release-train gate assessment (with a surprise)

- `go.work`/`go.work.sum` restored, no `.hold`; all 15 verify-lock files tested with `flock -n`: **FREE** (stale leftovers, no live verify).
- Tree NOT clean: 7 modified `metaengine/rule_*.go` files = ANOTHER SESSION's in-flight `declaredQuery(ctx, q)` refactor (mechanical extraction of `ctx.Store.queries[q.QueryName]`, semantically neutral). Read in full, left untouched, never reverted.
- One transient build break raced the other session's live edits (`../fold.go:258: undefined: TypeName` — retry green seconds later).

### M1 — 7-tag train verification: **THE TRAIN STALLED AT 1/7**

- The prep commit `2d1669f78` describes 7 intended tags: `dispatcher/v4.5.0`, `middleware/v4.7.0`, `metaengine/v4.15.0`, `system/v4.10.0`, `event/v4.12.0`, `command/v4.12.0`, `query/v4.9.0`.
- Reality: ONLY `dispatcher/v4.5.0` exists (local + origin + proxy + pkg.go.dev; tagged 09-27 23:53). The other 6 are cut **nowhere**. Their release content IS in-tree; dependents' go.mod already pin `dispatcher/v4.5.0` (published); CHANGELOG `[Unreleased]` honestly still carries the 6 entries.
- `batch-release.sh` keeps **no logs** — the stall cause is undiagnosable from artifacts.
- Fresh-consumer verification: `dispatcher/v4.5.0` `go get` + build green in a scratch module.
- pkg.go.dev spot check: dispatcher v4.5.0 listed **but docs hidden (License: UNKNOWN)**. Investigated root cause: the root LICENSE is deliberately **PROPRIETARY**; pkg.go.dev hides docs for non-redistributable licenses by design. Verified the counter-example (`benchkit/` carries its own LICENSE copy and is hidden the same way) → the "copy LICENSE into 97 module dirs" fix idea is **dead on arrival** — not a defect, an intended consequence of the licensing choice. Also noted `benchkit/LICENSE` says "Unknown Author" (template artifact) — owner's legal file, recorded not touched.
- Receipt rowed into the TODO_LIST tag-wave row.

### M2 — Poisoned-tag surgery verification

- **Q1 executed path = leave-published + supersede:** `storage/v4.10.0` stays on the proxy un-retracted; `storage/v4.10.1` (09-22) is @latest. Verified `storage/v4.10.1`'s published go.mod carries only the ancient `retract v4.7.0`.
- **`system/v4.9.0` graph fully resolves + compiles** in a fresh module — and it references NO `storage/v4` at all. The feared "retraction breaks system/v4.9.0's graph" branch never existed.
- **Q2 executed path = tag-delete:** `tursoengine/v4.2.0` tag deleted from origin; proxy `@v/v4.2.0.mod` 404s (cached absence) while `@v/list` still names it → the binary-junk zip **never reached any consumer**. No retract shipped.
- **RESIDUAL DEFECT found + rowed:** the module's `@latest` 404s entirely (fresh `go get .../metaengine/tursoengine/v4@latest` fails). Fix proposal in the TODO row: cut `tursoengine/v4.2.1` from the clean tree via `tag-release.sh` (no retract needed — v4.2.0 was never proxy-published). Owner action.

### M3 — CatchUpEngine race verification: **STALE CLAIM, already fixed**

- Code read: the current algorithm (`failover.go:137-189`) replays via offset-keyed stabilize passes and lifts quarantine only inside the append-blocked stability gate `reactivateIfStable` (`consistency.go:92-103`); the live path records into the EventLog BEFORE the routing decision under one `s.mu.RLock` (`store.go:495-511`) — the once-taken-Events-snapshot hole is closed by construction.
- Fix landed 2026-09-13 (commit-dated); the stress test `TestEngineHealth_CatchUpUnderConcurrentApplies` exists to pin it; the TODO 🔥 row the FEATURES caveat pointed at no longer exists.
- Empirical: **15/15 green under `-race`** (5 + 10 counts) at ~0.08s/run.
- Both remaining `Events()` snapshot callers verified sound: `DemoteEngine` snapshots under the SAME write lock as the role flip (documented invariant); `Backfill` is point-in-time by contract.
- FEATURES.md:326's "Known fast-follow" sentence replaced with the verified-fixed statement.

### M6 — Junk-file forensics + gotcha

- Exact dropped set (from `2d1669f78`): `\006` (4 KB), `\006-wal` (169 KB), `P\021B` (4 KB), `P\021B-wal` (250 KB) — two SQLite DB+WAL pairs.
- **Zip-poison vector pinned:** `git ls-tree metaengine/tursoengine/v4.2.0` shows the `\006` pair INSIDE the tag's tree. The junk was added by daemon commits 09-19/09-20 (after the v4.1.0 tag, before v4.2.0's cut).
- **The CHANGELOG's side-claim was FALSE:** "the `\006` pair already shipped inside published v4.14.0" — downloaded the real `metaengine/v4.14.0` proxy zip and inspected: 345 entries, zero tursoengine entries, zero >100 KB files. Corrected the CHANGELOG line to the verified truth.
- Leak-producer grep: every current test DSN goes through `t.TempDir()`; full tursoengine suite run (167 s green) left the tree **clean** — the producer died with the 09-19/09-20-era code. M13 (leak fix) correctly NOT triggered.
- Gotcha entry written: `docs/agents/gotchas-tooling-build.md` — "Test-dropping junk files with control-char names" (invisibility via `cat -v`/`ls -b`, daemon-commit vector, tag-freeze risk, the three guards, poisoned-tag runbook).

### M7 — Engine/driver/ADT count truth audit (6 docs fixed)

- Census: **12 engine implementations** (memory in-core + 11 modules), **11 registered drivers** (memory + 10 module `register.go`; iroh does not self-register), **12 ADTs** (Map/Set/Counter/SortedMap/Log/Multimap/Graph/Vector/Search/Spatial/DueClaim/Dedup). bigtable's `EngineResetter` verified as a real implementation, not a comment.
- Fixed: `FEATURES.md:325` (bigtable missing from the 12-engine enumeration), `FEATURES.md:1477` ("10 engines, 10 ADTs" → 12/12 with full ADT list), `FEATURES.md:1500` ("all 10 drivers" → 11 with parenthetical), skill `core.md:205` (10→12 ADTs), skill `modules.md:104` (stale 10-ADT enumeration that even listed a nonexistent "Scan" ADT → correct 12), `ROADMAP.md:74` (10→12 ADTs).

### M8 + M9 — Explain Volume/placement gap: confirmed, then IMPLEMENTED

- M8: gap still present — `System.Explain` printed drivers/engines/collections-count only (`system/introspection.go:195-223`).
- M9 shipped:
  - `metaengine`: exported `QueryPlacement` struct + `Store.QueryPlacements()` (name-sorted; engine, ADT, **declared Volume hint**, plan estimate, complexity) in `explain.go` (baselined-file headroom respected: 430→~485 of 515).
  - `system`: `Explain` now renders `~ <query>: <engine> (<adt>, volume=N/s, est=Xms)` per projection; no-store path pinned.
  - Tests: `TestQueryPlacements` (metaengine), `TestExplainRendersQueryPlacements` + `TestExplainWithoutProjectionStore` (system) — green.
  - Discovered mid-test: the planner estimates latency with a DEFAULT volume even when no hint is declared — accessor doc corrected to say so.
  - api-stability golden regenerated (+8 exports); `check-changelog-symbols.sh` green (42 citations); CHANGELOG `[Unreleased]` Added entry; TODO:1099 closed with receipt.

### M10 — Health catch-up test flake: evidence gathered, kept observe-only

- 15/15 green under `-race` isolated; full metaengine package suite green **twice** this session (41 s, 46 s — the second run concurrent with another session's edits). Receipt appended to the row; no recurrence since 2026-09-13.

### M11 — Durable checkpoint/DLQ design: premise was STALE, ADR written instead

- **Durable checkpoints already SHIPPED in-tree:** `system.NewEngineCheckpointStore(backend)` persists checkpoints as `system_checkpoints` Map-collection entries on the deployment-declared engine (engine named "checkpoints" wins, Map-ADT gated, documented in-memory fallback, `DomainConfig.CheckpointStore` override, restart-durability pinned by `systemtest.TestEngineCheckpointStoreRestartDurability`). This is the untagged system/v4.10.0 content.
- **Durable DLQ: no separate store exists BY DESIGN** (ADR-0117): `WithCommandLifecycle(store)` records lifecycle events into the caller's event store; DLQ/FailureLog/RejectionLog are projections over them.
- Wrote **ADR-0149** (`docs/adr/0149-durable-checkpoints-and-dlq-by-events.md`) documenting the shipped design and superseding the TODO premise; M17 (checkpoint impl) = already satisfied, M18 (DLQ store extraction) = obsolete.
- **Doc rot found + fixed:** ADR-0148 was MISSING from `docs/adr/README.md`'s index even though the CHANGELOG claims "the ADR index gained 0148". Added 0148 + 0149 rows.
- TODO row closed; residual gap = PUBLISHING (the stalled wave), not design. cqrs-htmx `NewProjectionLayer` unblocks when system/v4.10.0 ships.

### M12 — sqliteengine README note

- Added the operator-only limitation note: modernc.org/sqlite's `database/sql` surface exposes no `LoadExtension` (v1.59.0) → C extensions (sqlite-vec, vec0 indexes) are impossible inside this engine; native vector SQL requires an operator-managed libSQL server via `tursoengine`. (doc-check batch pass = M16, still pending.)

---

## b) PARTIALLY DONE

### M14 — IVM defect-A onset bisect (interrupted here)

- `metaengine/tursoengine/ivm_bisect_test.go` WRITTEN (build tag `ivmrepro`, opt-in `TURSO_IVM_BISECT=1`): fresh-DB-per-config workload sweeping three dimensions — chunk/tx-size {2000,1000,500,100,10} at 2k rows; groups {1,2,8,64,316,2000} at 500-chunk; rows {500,1000,2000,4000} at 500-chunk — with per-transaction divergence detection and defect-C wall detection.
- First run FAILED at chunk=10/tx#93: hit defect C's commit wall ("cannot commit - no transaction is active") — my per-tx `GroupedAggregate` scans shrink the wall (documented behavior I initially treated as harness failure).
- Harness fixed: wall now recorded as DATA (`wallTx` verdict) instead of `t.Fatalf`; `go vet -tags ivmrepro` green.
- **NOT YET DONE: the sweep itself has never run to completion.** Next step: re-run, harvest the matrix, record in `docs/benchmarks/` + TODO:208 receipt, then M19 (upstream filing) unblocks.

### Inline receipts into TODO_LIST (part of M15's job, done incrementally)

- Rows updated with dated receipts this session: tag-wave row (M1), poisoned-tag-surgery row (M2), catch-up-race FEATURES fix (M3), health-test flake row (M10), Explain row closed (M8/M9), checkpoint/DLQ row closed (M11). M15's remaining work: add genuinely-new rows (see f).

---

## c) NOT STARTED (plan tasks)

- **M25** — quiet-window orchestrator wrapper for M5/M20 (`#quiet-window-run` batch + self-test + plan-tail documentation).
- **M5** — ADTSet G-T13 MySQL VM leg (quiet-window gated; needs M25 first; nspawn variant may need root, VM variant ~131 s always works).
- **M15** — full HARVEST of the plan into TODO_LIST (partially done inline; the diff/verify-columns pass remains).
- **M16** — doc gates batch: `cmd/doc-check` (AGENTS scan set incl. the new gotcha + README note), `check-readme-deprecated`/`check-readme-links` (my doc edits happened after their last nightly).
- **M19** — turso-go zombie-tx/IVM upstream filing (gated on M14 completion; requires loading `verify-before-filing` + `github-voice` skills; gh + network untested).
- **M20** — dgraph calibration constants re-anchor + SearchQuery count=5 (quiet-window gated).
- **M21** — bigtableengine real-GCP run (access-gated; likely unavailable — untested).
- **M22** — Q3 codification — **BLOCKED on the owner's Q3 answer** (report-artifact policy for narrow skill triggers).
- **M23** — v5-prep removal census (documentation-only).
- **M24** — T19–T21 blocked-status confirmation (documentation-only).
- Plan-scope notes: M13 correctly not triggered (no leak producer). M4 skipped (M3 verdict = stale claim). M17/M18 obsolete (already shipped, per M11 verification).

---

## d) TOTALLY FUCKED UP! (honest mistakes — all caught + fixed in-session)

1. **Bisect harness, run 1:** treated defect C's commit wall as a harness FAILURE (`t.Fatalf`) instead of data — the wall is a characterized defect and its position IS bisect output. Also shipped a draft leftover `_ = stepErrUnused` line in one edit (caught and removed before running).
2. **Bisect harness design, pre-run:** per-tx `GroupedAggregate` reads naively shrank defect C's write budget — I designed the sweep against a documented constraint I had already read (the ivm_repro_test.go comment warns about exactly this). Wall-as-data fixed it, but chunk=10's onset number will carry an asterisk (wall at tx#93 = 930 rows in).
3. **`TestQueryPlacements` took three iterations to green:** guessed `On("item.created", ...)` (string arg — signature takes a SAMPLE, panicked), then declared a fold-less Query (panics by design), then asserted `est==0` for the no-hint query (wrong — the planner assumes a default volume). All three were avoidable by reading `fold.go`/`query.go` signatures BEFORE writing the test.
4. **Guessed a nonexistent API in the M1 scratch test** (`dispatcher.Handler`) instead of checking exports first — replaced with a blank-import build check, which was the honest minimal verification anyway.
5. **Tried proxy probes through shell-style `fetch` flags** (failed — `fetch` is a tool, not a curl); switched to the fetch tool properly. One `sed`-from-grep command also died on a bad expression (retried differently).
6. **Edited ADR README + CHANGELOG before viewing exact context twice** — the edit tool refused; wasted round trips. Self-inflicted, zero damage.
7. **Session hygiene:** scratch artifacts left in `/tmp` (`goget-m1/`, `goget-m2/`, `metaengine-v4.14.0.zip`) — harmless but should be trashed at session end.

---

## e) WHAT WE SHOULD IMPROVE!

1. **`batch-release.sh` writes no run log** — a stalled 6-of-7-tag train is currently undiagnosable from artifacts. A one-line-per-module log (tagged, verify result, smoke result) would have made M1 a 5-minute read instead of forensics.
2. **Hand-maintained counts in docs keep rotting** (10-vs-11-vs-12 engines/ADTs/drivers across FEATURES/skill/ROADMAP). The repo already has the canonical-facts gate pattern — engine/ADT/driver counts should be gate-derived (census command) like the go.mod count is.
3. **Verify-before-acting keeps paying off** — 4 stale claims killed this session (race caveat, checkpoint/DLQ "in-memory-only", CHANGELOG "shipped inside v4.14.0", ADR-index-has-0148). The plan's verify-first gating is the right shape; keep it for every future "known-broken" row.
4. **Read the target API before writing tests** — see d.3; three wasted iterations on a 60-line test.
5. **Authored commits vs daemon:** M9 (exported API + golden + ADR) would deserve an authored commit; I let the daemon absorb it (the documented race makes immediate authored commits the correct pattern when history matters — apply next time).
6. **pkg.go.dev hidden docs** is a permanent consequence of the proprietary license — worth one FAQ/README sentence so the next session doesn't re-derive it (I spent two fetches on benchkit to disprove my own fix idea).
7. **`metaengine` README module-list row is a 4 KB wall of text** (modules.md:104) — counts drift INSIDE it invisibly; split or gate-check the enumerable claims.

---

## f) Top things to get done next (impact-sorted, ≤50)

1. ~~**Re-run the M14 bisect sweep** (`TURSO_IVM_BISECT=1 go test -tags ivmrepro -run TestIVMReproDefectAOnsetBisect`) — harness is fixed, numbers pending.~~ done — 2026-09-28 — harness fixed; matrix recorded (M14, session 2)
2. ~~Record the bisect matrix in `docs/benchmarks/` + TODO:208 receipt (M14.5).~~ done — 2026-09-28 — onset-matrix doc + TODO receipt
3. ~~**M19: file the turso-go upstream issue** (load `verify-before-filing` + `github-voice`; minimal repro; check latest turso-go first; `gh` auth check).~~ done — 2026-09-28 — turso#9391 filed (M19)
4. ~~**M16: doc gates** — `cmd/doc-check` over SKILL.md/references/AGENTS.md (new gotcha entry + README note are in the scan set), `check-readme-deprecated`, `check-readme-links`.~~ done — 2026-09-28 — doc gates green (M16)
5. ~~**M15: HARVEST remainder** — add new TODO rows: (a) batch-release run-log improvement, (b) gate-derived engine/ADT/driver counts, (c) the "verify-columns convention" from the plan, (d) pkg.go.dev-license FAQ note; verify 3 claims docs-health style.~~ done — 2026-09-28 — M15 rows added
6. ~~**M25: quiet-window orchestrator** wrapper + `--self-test` (unblocks M5/M20 mechanics).~~ done — 2026-09-28 — quiet-campaign.sh shipped (M25)
7. ~~**M5: G-T13 MySQL leg** via the orchestrator (VM leg if nspawn needs root) — closes the FEATURES "MySQL VM leg pending" caveat with a receipt.~~ done — routed — TODO metaengine tag-wave row (M5 leg)
8. ~~**M20: dgraph constants re-anchor + SearchQuery count=5** when `calibration-gate.sh` passes; provenance lines + drift check.~~ done — routed — TODO calibration row (M20 leg shipped)
9. ~~**M23: v5-prep census** (Deprecated markers vs api golden: On/OnTyped, Infer, stack presets, shells) — doc-only, 30 min.~~ done — 2026-09-28 — M23 receipt: census is gate machinery
10. ~~**M24: T19–T21 ADR-0142 citation currency check** — doc-only, 30 min.~~ done — 2026-09-28 — M24 receipt: ADR-0142 citation current
11. ~~**M21: bigtable GCP run** — first check credentials exist; record BLOCKED if not.~~ done — 2026-09-28 — M21 BLOCKED recorded (FEATURES:230)
12. ~~**Trash `/tmp/goget-*`, `/tmp/metaengine-v4.14.0.zip`** (session hygiene).~~ done — session hygiene executed (session-2 report)
13. ~~**Stale-lock cleanup consideration:** 15 stale verify-lock files in `.gotmp` confuse gate triage — a lock-sweep or PID-encoding idea for the maintainers.~~ done — routed — locks tested FREE 2026-09-28 (M1.1); sweep note is owner env work
14. ~~**tursoengine v4.2.1 cut** (owner mechanics) — repairs the module's 404 `@latest` (rowed in M2 receipt).~~ done — routed — TODO data-mesh tail (tursoengine v4.2.1)
15. ~~**Complete the stalled 6-tag wave** (owner mechanics: `batch-release.sh --from-manifest` if a manifest exists) — the single highest-consumer-impact action in the repo right now.~~ done — routed — TODO metaengine tag-wave row (M1 receipt)
16. ~~Lint pass on the two changed modules (`metaengine`, `system`, `metaengine/tursoengine` test file) — golangci via buildflow or `nix run .#lint` before the next verify.~~ done — 2026-09-28 — lint zero findings on session files
17. ~~`nix run .#check-file-size` — confirm the explain.go growth stays under the 515 baseline after M9.~~ done — 2026-09-28 — file-size gate GREEN
18. ~~`cmd/api-stability` meta-test: `GOWORK=off go test -run TestEvery .` after the golden regen.~~ done — 2026-09-28 — TestEvery green
19. ~~Sweep `docs/agents/module-map.md` for the engine/ADT count drift class M7 found (not yet checked).~~ done — 2026-09-28 — module-map swept in the M7 count fix
20. Consider a `QueryPlacements()` leg in Doctor parity (plan 9.2 said "if applicable" — currently n/a because system has no Doctor; metaengine's Doctor could render placements cheaply).
21. ~~CHANGELOG `[Unreleased]`: when the wave completes, the six entries move under a dated section — remember the changelog-coverage gate.~~ done — routed — TODO metaengine tag-wave row
22. ~~cqrs-htmx unpins `NewProjectionLayer` after system/v4.10.0 ships (external repo follow-up; named in ADR-0149).~~ done — routed — external repo follow-up (ADR-0149 names it)
23. ~~The other session's `declaredQuery` refactor is STILL uncommitted in-tree — coordinate so it doesn't rot (not mine to finish).~~ done — the declaredQuery refactor landed in-tree (consumed by the dedup campaign)
24. ~~`example/taskmanager` golden pins version numbers (TODO:331) — will move again when the wave ships; pre-check before tagging.~~ done — routed — TODO Release section (V006 golden row)
25. Nightly gates will hit the new gotcha file (md-go fence policy) — M16's doc-check run covers it; verify zero warnings.
26. ~~Optional polish: `benchkit/LICENSE` "Unknown Author" → real name (owner's legal call, noted in a).~~ done — routed — TODO benchkit/LICENSE row (12th-pass harvest)
27. If quiet windows don't open tonight: schedule M5/M20 via the SystemNix nightly timer pattern instead of blocking a session.
28. After M9: consider rendering `~` placements in `ProjectionExplain` too (currently only `Explain`) for symmetry — tiny, cosmetic.

---

## g) Top questions I can NOT figure out myself

1. **The stalled release train:** the 2026-09-27 7-tag wave stopped after `dispatcher/v4.5.0` with no logs. Should I complete the remaining 6 tags via `batch-release.sh` (verify-lock + zip guard + smoke) — or is the owner re-running/abandoning the wave? Every tagging decision in this repo's history is owner-gated, so I did not touch release mechanics.
2. **tursoengine `@latest` repair timing:** cut `tursoengine/v4.2.1` now from the clean tree, or fold it into the next wave? (Both are owner tag operations; the 404-`@latest` is a live consumer-facing defect until one of them happens.)
3. **Q3 (still open from the prior session):** what IS the report-artifact policy for narrow skill triggers — chat-answer + "report on request" as a sanctioned exception, or always write the artifact? It gates only M22 now. (Sub-question, your call, zero urgency: may `benchkit/LICENSE`'s "Unknown Author" be corrected to your name? It is a legal notice — I will not touch it without instruction.)

---

_Arte in Aeternum — execution resumes on instruction._
