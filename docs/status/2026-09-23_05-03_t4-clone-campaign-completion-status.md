# t4 Clone-Elimination Campaign — Completion Status & Self-Review

**Report time:** 2026-09-23 05:03 CEST
**Scope:** This session's run (continuation of `docs/planning/2026-09-23_00-03_SUPERB-t4-clone-elimination-campaign.md`; prior state in `docs/status/2026-09-23_01-38_t4-campaign-mid-flight-status.md`).
**Verdict:** Campaign COMPLETE. 21/21 clone groups resolved. t4 AND t7 repo scans clean (0 actionable, verified by consecutive idle-tree double-runs). One owner-rejected decision (IN separator) reworked and re-verified. Format note: user requested `.md`; the status-report skill's HTML default was overridden by explicit instruction.

---

## a) FULLY DONE (verified green this session)

| Item | Evidence |
| --- | --- |
| F11/G11: `streamIDFromMessage` extracted (`watermill/stream_id.go`), `MessageToCommand`/`MessageToEvent` rewired | watermill suite `ok` 0.321s (run twice) |
| F11/G14: asymmetric bus-loop twins accepted (directives on region first lines) | watermill t4 scan: 0 shown, 15 suppressed |
| duckdb full-suite rerun (tree previously untested since F2) | `ok` 462.116s |
| F12: 8 accept-directives placed — G6 (queue/conformance ×2), G10 (testutil containers ×2), G12 (cattest/cqrs-upgrade ×2), G13 (commandtest/eventtest ×2) | scoped t4 scan 0 shown; all modules vetted + tested |
| F13/G15: pgengine hand-rolled tx-isolation test promoted onto `adttest.AssertTxIsolationFromForeignContext` | real-PG `ok` 77.4s; pair scan 0 shown |
| Residual group: third BFS copy eliminated — `metaengine.GraphBFS` in core; mysql `graphWalk` + sqlite `graphBFS` deleted; 4 call sites delegate with byte-identical label errors | sqlite full `ok` 10.4s; core graph tests `ok`; engines build |
| Residual groups: vector dimension-lock ladders accepted (4 directives, duck/pg + mysql/sqlite) | engines build; t4 idle-run 0 shown |
| F14: family completeness sweep — every engine's VectorInsert uses core `CheckVectorDimension`; 9 engine modules pin it via `adttest.AssertVectorDimensionGuard`; no copies of new SQL families in KV engines/memory/iroh | rg sweep + adttest coverage list |
| F15: api golden regenerated (+9 exports incl. `GraphBFS`), `TestEvery` green uncached, doc-check green (1200 refs) | `docs/api_surface.txt` diff; `ok` 3.901s / 9.741s |
| F16: CHANGELOG entry (rewritten after separator revert); honesty gate green | `Verified 35 pkg.Symbol citation(s)` |
| F17: `references/modules.md` metaengine row extended (GraphBFS + plumbing families) | doc-check green |
| F18: AGENTS internal contract #27 (engine plumbing in core, label-prefix errors, caller-owns-lock-mode, accept-don't-merge residue) | doc-check green |
| F20: `check-duplication` hardened — art-dupl nix-provisioned at v0.7.0 (`packages.art-dupl`, fetchFromGitHub + vendorHash resolved), silent SKIP replaced with hard error, dirty-tree guard kept | `nix build .#art-dupl` green; gate executes and reports 50 groups |
| F21: file-size ratchet violations fixed (relocations: `keyExtractorPtr` → fold_classify.go, `applyIndexEntries` → sort_index.go, `streamIDFromMessage` → stream_id.go); full test matrix green; t4+t7 clean | `check-file-size`: only the parallel session's eventcatalog offender remains |
| F22: TODO_LIST harvest (3 items) | TODO_LIST.md Code Quality section |
| Owner decisions executed: per-dialect IN separator restored (`AppendPlannedFilter` + `inSeparator` param; sqlite `","`, pg/mysql `", "`), work-down policy recorded, daemon commit policy confirmed | sqlite `ok` 11.8s, pg `ok` 35.6s, mysql `ok`; changelog gate green |

**Test matrix run this session (all mine green):** watermill, metaengine core (full ×2), sqliteengine (full ×2), pgengine (full ×2, real PG), duckdbengine (full, 462s), mysqlengine (compile + server-gated skip), pebbleengine (full, post-relocation), queue, catalog, cmd/cqrs-upgrade, testutil, storage/memory, command/commandtest, event/v4/eventtest, cmd/api-stability, cmd/cqrs-lint (pre-existing red only).

---

## b) PARTIALLY DONE

1. **mysqlengine BFS rewire — verified by compilation + reasoning, not execution.** No local MySQL server: the suite reports `ok 0.005s` (all server-gated tests skip). The `metaengine.GraphBFS` delegation preserves error labels byte-for-byte and sqlite exercises the identical core walker, but mysql's own `TestGraphNeighbors_IterativeMatchesCTE` / undirected twins never ran here. Verification route unresolved (see question 1).
2. **t3 work-down: decided but not started.** Owner chose NO baseline re-pin; the 50 pre-existing groups (down from ~69) must be worked down while the gate stays hard/red. Recorded in TODO_LIST.
3. **art-dupl under-load instability: characterized, not root-caused.** Repo-wide t4 scans under parallel CPU load (tests/fmt running) intermittently report `16 shown / 828 suppressed` instead of `0 shown / 874 suppressed`; two consecutive idle-tree runs are deterministic and clean. Hypothesis: directive association degrades while type-info loading contends — unproven; it's Lars's tool (see question 2).

---

## c) NOT STARTED

- **Burn-down of the 50 t3 groups** (the owner's chosen path; M-L effort).
- **`metaengine.graphNeighborsFallback` unification onto `GraphBFS`** (core's degraded-path BFS: `[]any` frontier, `typedNodeKey`, `nil`-for-depth<=0 semantics — needs a behavior decision; TODO_LIST item).
- **Real-server verification of mysqlengine graph paths** (blocked on server access).

---

## d) TOTALLY FUCKED UP

Nothing irrecoverable; one self-inflicted rework cycle:

- **The IN-list separator was unified to `", "` before owner sign-off.** The prior session made the change; THIS session then propagated the "unified everywhere" story into CHANGELOG/modules.md and only asked at the very end — the owner rejected it. Cost: one re-edit cycle (core param + 3 call sites + 2 doc rewrites + retest). Everything is now consistent with per-dialect wire strings, but the sequence was wrong: wire-visible behavior changes must be gated at the decision point, not post-hoc.

---

## e) WHAT WE SHOULD IMPROVE (self-review, this session)

1. **Gate cadence: run the ratchet/file-size gates per family, not only at campaign end.** Session-2 extractions grew `fold.go` (+4) and `layout_planner.go` (+1) past their baselines; my own G11 helper grew `protocol.go` (+10). All caught only at F21 and fixed by relocation. Running `check-file-size` after each extraction would have caught them at the source.
2. **Ask blocking questions at the decision point.** The three owner questions sat in the plan from the start; asking them up front would have avoided the separator rework entirely.
3. **Controlled experiments over pattern-matching.** I twice waved off the "16 shown" t4 artifact as a daemon race before running the definitive sequential double-run. First recurrence should have triggered the experiment.
4. **View tool discipline.** Three failed edits because I read targets via `sed`/`rg` then called `edit` (stale-read refusal). Wasted round-trips; View is mandatory pre-edit.
5. **Unverified gate coverage claim.** doc-check reported "1200 references" before AND after I added ~10 symbol mentions to modules.md; I did not confirm the new mentions are actually validated by the gate (they may be row-context prose, not `pkg.Symbol` citations). Small honesty gap — the new citations may have no mechanical gate.
6. **Cross-session hygiene note.** `nix fmt` reformatted `catalog/eventcatalog/writer.go` (the parallel session's file) — mechanical only, left in tree, but it should have been called out explicitly in the closing summary rather than implied.
7. **Directive-length robustness.** Several `//art-dupl:accept` reasons are >100-char single lines. If any formatter ever wraps comments, directive parsing (first-line anchored) breaks silently. Worth an upstream guard in art-dupl or a lint rule.
8. **Three-place documentation invariant.** The new helper families are now described in AGENTS #27 (contract), modules.md (lookup), CHANGELOG (history). Different audiences, justified — but future helper additions must touch all three + golden; the "Add a New Module"/"Change an Exported Symbol" procedures should grow a checklist line for the plumbing-family contract.

**Ghost systems check:** none. Every new symbol is wired (`GraphBFS` ← 2 engines ×2 paths; helpers ← 4 SQL engines; `streamIDFromMessage` ← 2 protocols; adttest promotion ← pgengine).
**Split-brain check:** the separator fact lived inconsistently in docs for part of the session (unified vs per-dialect) — now reconciled everywhere; no remaining known drift from this session's work.

---

## f) Next things (up to 50; brainstorm-graded, top items already in TODO_LIST)

**Queue (from this session's open ends):**
1. Work down t3 group #1-10 of the 50 (start with the largest by tokens; extract or accept each) — TODO_LIST item.
2. …continue through #50; the baseline shrinks naturally at the next structural re-pin.
3. Unify `graphNeighborsFallback` onto `GraphBFS` (nil-vs-empty decision + typed-key encode param) — TODO_LIST item.
4. Run mysqlengine against a real server (nspawn needs root / CI leg) to execute the iterative-graph tests.
5. File upstream art-dupl issue for the under-load suppression instability (with reproduction: parallel load → 874→828 suppressed).
6. Add an art-dupl/lint guard: `//art-dupl:accept` directives must stay single-line.
7. Verify doc-check actually validates the newly added modules.md symbol mentions; if not, extend its citation extraction.
8. Add the plumbing-family checklist to AGENTS "Change an Exported Symbol" procedure (AGENTS #27 + modules.md + CHANGELOG + golden in one edit).

**Wave-transient / release-train (noticed this session):**
9. Next tag wave: re-tag dispatcher (v4.4.1 lacks `Middleware[H]` that command/event now reference) — heals GOWORK=off standalone builds.
10. Same wave: `pin-sweep` to bump command/event dispatcher pins.
11. Same wave: refresh cqrs-lint taskmanager V003/V006 version-set goldens (pre-existing red confirmed again this session).
12. Check whether the 3 dependabot alerts seen at push time are still open; triage.
13. `graphNeighborsFallback`'s `depth <= 0 → nil, nil` vs engines' `[]any{}`: pick ONE contract and document it in ADR or contract #26/27.

**Duplication debt observed but out of campaign scope:**
14. `typedNodeKey` vs `encodeKey`/`encodeNodeKey` vs `encodeIndexValue` — three key-encoding vocabularies across core/engines; consider one naming/semantic pass.
15. The 499 "non-actionable" t4 groups — periodically triage whether art-dupl's non-actionable bucket hides real debt as the codebase grows.
16. systemtest fixtures carry accept-directives ("test-fixture twin of …") — a sign system/systemtest share helpers across module boundaries; evaluate one shared testutil export.
17. queue/mysql (and sqlite/postgres) carry many dialect-twin directives; once conformance pins are proven stable, consider whether a codegen/table-driven approach shrinks them.

**Gate/tooling hardening:**
18. Add `check-duplication` to a CI leg that runs on an idle runner or with `nice`/serialized load (avoid the under-load artifact class entirely).
19. Teach `check-file-size` to run in `--fix-suggest` mode listing the nearest relocation target for new offenders.
20. Add a golden assertion that `packages.art-dupl` version stays pinned (flake input bump alert) so the gate's semantics can't drift silently.
21. Consider `art-dupl check --jobs=1` (or a flag) if the tool gains one, for deterministic CI runs.
22. Add the `", "`/`","` separator facts to the FAQ (cross-engine wire-string differences) — noticed while documenting.

**Behavior parity tails:**
23. mysql `CAST(LENGTH(vec)/4 AS SIGNED)` vs sqlite `LENGTH(vec)/4` probe — confirm MariaDB vs SQLite integer-division behavior is pinned by a test on each (probe ladder accepted as twin; semantics ride adttest).
24. pg `mustMarshalMetadata` vs core `VectorMetadataArg` — pg still hand-rolls metadata marshaling in VectorInsert; evaluate rewiring for one path.
25. duck/pg `json.Marshal(emb.Values)` vs mysql/sqlite `EncodeVectorF32` — dual wire formats for vectors across engines are intentional (per contract #26) but deserve a one-line WHY in modules.md next to the vector family.

**Docs/truth:**
26. Annotate the prior mid-flight status report (2026-09-23_01-38) with a completion pointer to this report (docs-health ANNOTATE).
27. Record the art-dupl load-artifact in `docs/agents/gotchas-tooling-build.md` so future sessions don't re-diagnose it.
28. CHANGELOG: when the 50-group work-down starts, add entries per extracted family (keeps the honesty gate meaningful).
29. Consider a `docs/adr/` note for the GraphBFS promotion decision (3rd-copy elimination; fallback path left separate + why).

**Bigger swings (ROADMAP fuel, from this session's observations):**
30. Extract an `engine-scaffold` nix/go template for new engines (register.go, vector_dimension_test, dueclaim wiring are now visibly formulaic across 10+ engine modules).
31. Promote the accept-directive corpus into a machine-readable map (which groups are deliberate twins) rendered into Doctor-style docs for contributors.
32. Metaengine "engine conformance kit": bundle adttest.Assert* into one `RunFullEngineConformance` entry a new engine can adopt in one line.
33. Per-engine smoke profiles in benchkit for the shared helper families (scan/filter/BFS) so refactors get perf-verified, not just correctness-verified.
34. Investigate whether `nix fmt`+daemon interleaving can be serialized (daemon hook) to kill the moving-target class for scanners and agents alike.
35-50. Reserved: the 50-group work-down itself will generate its own item list once triage starts (each group = one item; do not pre-invent them here).

*Routing note: items 1-3 and 13 are already in TODO_LIST.md (harvested this session); the rest is ROADMAP/triage fuel per docs-health HARVEST rigor — a future docs-health pass should route, not blanket-copy.*

---

## g) Questions I cannot figure out myself

1. **MySQL verification route:** mysqlengine's graph paths (rewired this session) have never executed locally — the suite skips without a server, and `#integration-mysql-nspawn` needs root I don't have. Should I treat CI's MySQL leg as the verification authority for this change, or do you want to hand me a runnable server/DSN (e.g. `MYSQL_TEST_DSN`) for a one-time local run?
2. **art-dupl under-load instability:** repo-wide scans under parallel CPU load intermittently drop ~46 suppressions (874→828) and show 16 phantom groups; idle-tree runs are deterministic and clean. It's your tool — known quirk, or should I file an upstream issue with my reproduction?
3. **Work-down scheduling:** the 50 remaining t3 groups (your call: no re-pin) — start a dedicated SUPERB work-down campaign next, or queue it behind the current T01-T27 plan (`2026-09-22_01-25_SUPERB-unblock-prove-deliver-pareto-plan.md`)?

---

*Point-in-time snapshot. Annotate, never rewrite, when stale.*
