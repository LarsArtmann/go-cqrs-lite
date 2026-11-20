# Status — v5 Core-Type Plan Execution: Wave 0 complete, Wave 1 started (2026-11-20 20:17 CET)

**Session window:** 2026-11-20, ~17:50–20:17 CET · **Head at start:** `8828ea837` · **Head now:** `baa8f94a8` (6 authored commits, all local — NOT pushed)
**Plan of record:** [`docs/planning/2026-11-19_20-52_SUPERB-v5-core-type-finalization-pareto-plan.md`](../planning/2026-11-19_20-52_SUPERB-v5-core-type-finalization-pareto-plan.md) (v2.1) · Rulings: D-sheet **D32**
**Mode:** executing the commissioned plan under the blanket execute order; this report pauses at the user's explicit "WAIT HERE".

---

## a) FULLY DONE (6 commits, each gate-green at its boundary)

1. **M22 — tag trigger pinned** (`5f0dd00cf`). Read all three sources (TODO_LIST :395 + cut-readiness Layers 8/13 + parent-plan M42/M43 rows). Verdict: the first `core/v5.x.y` tag rides the **full v5.0.0 cut** (M42; no early tag anywhere); TODO_LIST :395 is downstream of the cut. Deadline axis = **gate-ordered**, written into the plan header + §1(b) + revision log v2.1; wave sizing unchanged.
2. **M01 — owner ruling pack drafted** (`8c7ea3007`). One-pager at `docs/planning/2026-11-20_18-10_v5-core-type-owner-ruling-pack.md`: (A) T26 reconciliation, (B) ADR-0166 flip-at-M04, (C) brand-now — each with evidence, options, veto consequences. Recorded as **D32** (EXECUTING PER RECOMMENDATION; retro-veto = single wave-commit revert). D-sheet table realigned by dprint (no content change).
3. **M02 — review errata applied** (`bff08dfc6`). R1 card got the T26-gated amendment callout (replacing "execute the cut now" as the operative fix); methodology → honest inventory (24/91 files, census = raw grep, 10-17 body not re-read); R10 badge `issue-nice`→`issue-important`; hero subtitle scope-corrected; severity-matrix row + footer note. Re-validated: tag balance none-errors, all nav anchors resolve, `callout-amber` class exists.
4. **M17 — HARVEST + rowing** (`8501b4d54`). TODO_LIST gained the execution-ledger section (24 rows: M00–M24 + 2 new tooling rows); postmortem §f's 22 items fully routed (rows / plan tasks / explicit NOT-DO verdicts with reasons); dated pre-M43 dependency banner added to the 11-13 parent plan header.
5. **M00 — retype spike + calibration** (`a3ee4971a`). Scratch branch, full retype (brands + `Stream StreamRef` single-home + id-alias): fallout = **46 type errors in 5 files** — command/asrecord 22, query/asrecord 12, event/asrecord 7, +2 test files; kv/metadata compile untouched. Calibration written into plan §8: M03/M04 are declaration work; M06/M09 absorb the bridge toll; `go vet` (not `go build`) is the honest fallout counter. Branch discarded, tree restored clean.
6. **M03 — core/v5/record brands** (`baa8f94a8`). New `brands.go`: `InstanceID`, `Version` (uint64, 0=unversioned), `CorrelationID` string brands + validating `Parse*` doors + sentinels. Record.ID/.Type/.Version + CommonMetadata.CorrelationID retyped; bridges convert at the boundary (event/command/query AsRecord ×4); 8 test-compare sites updated; `brands_test.go` (parse doors, zero-value semantics, Type basics). Full core/v5 suite **green** (11 packages). Api golden regenerated **in the same edit** (6501→6511, 10 new exports — after one aborted commit attempt, see d)1). Deviation from plan letter (recorded): Record.StreamType NOT retyped in M03 — the field dies in M04's single-home collapse ("no StreamRef+StreamType", vision pillar 1); retyping it first would be churn.

**Wave 0 = complete (4/4). Wave 1 = M00+M03 done, M04/M05 next. 6 of 25 medium tasks.**

## b) PARTIALLY DONE

1. **M03's commit ritual** — code + tests + golden all landed, but the FIRST commit attempt aborted at the api-surface gate (see d)1); the second attempt succeeded. Net state: done, with one wasted hook cycle (~2 min).
2. **TODO_LIST execution ledger** — created with M00/M03 rows still unchecked ([ ]); ticks owed when execution resumes (the ledger was written before M00/M03 completed; rows for M22/M01/M02/M17 are ticked with receipts).
3. **api golden freshness discipline** — golden is current AS OF M03; every later code task (M04+) must regen per commit (rule re-learned the hard way, now baked into the remaining wave).

## c) NOT STARTED (19 of 25 medium tasks)

- **Wave 1 remainder:** M04 (ADR-0166 StreamRef alias + single-home Stream field + validating ctors + ADR flip + `#check-adr-numbering` — checks off cut-readiness **Layer 8**), M05 (Payload{Data,Enc} + EncodingEnvelope + tri-state tests).
- **Wave 2:** M06 (bridge stamps Received-at-intake, query zero-Stream, zero-struct guards), M07 (cause first-class on ImmutableEvent; incl. the event.Version→record.Version alias decision), M08 (structural Actor in Tracing), M09 (command/query twins + trinity contract test).
- **Wave 3 gates:** M10 (golden+TestEvery+untested-exports verification pass), M23 (v4-untouched negative check), M11 (v007 twin-table sync), M12 (doc mirrors + md-go), M13 (duplication/file-size).
- **Wave 4:** M14 (cqrs-upgrade v5 rules), M15 (migration guide sections), M16 (DOMAIN_LANGUAGE collapse), M18 (AGENTS.md memory), M19 (reviews README).
- **Wave 5:** M20 (scoped suites + lint-scope + full `#verify` exclusive), M21 (post-landing CI watch — **blocked on push authorization**), M24 (downstream compat review, pre-M43).

## d) TOTALLY FUCKED UP (honest ledger)

1. **M03 first commit aborted — I deferred the api golden to M10 against the repo's same-edit rule** (AGENTS Ground Rule 5: "API-surface change ⇒ api golden regen in the same edit"). The plan had scheduled golden regen as M10 (once, after M03–M09); I followed the plan letter instead of the repo contract. The hook caught it (6501 vs 6511), the commit aborted mid-hook, and the hook's formatter rewrote 2 staged files — recoverable but sloppy. Fix: golden regens EVERY code-task commit from here on; M10 becomes a verification-only pass. This plan-vs-repo-rule conflict should have been spotted at plan time (the self-audit missed it).
2. **multiedit mid-struct truncation** — my query/asrecord.go edit used an old_string that ended mid-struct, briefly creating a duplicated `MetaData:` block; caught by immediate re-view, fixed. Lesson applied: full-block context for bridge edits.
3. **Over-clever test** — brands_test.go's first draft used an `any`/func-indirection dance to "prove" the alias; rewritten as an honest basics test (the real cross-type pins already exist in event/command/query test files — verified by grep before writing, which saved a duplicate).
4. **Spike branch mechanics** — the working-tree edits rode across `git switch` (expected, but required a `git restore` of record.go on master); also learned `go build ./...` masks dependent-package errors (only query showed) while `go vet` reports all 5 files — recorded in the calibration note so M06/M09 use vet.
5. **Edit-tool read-before-edit failures ×4** — batched test fixes via a python one-shot script instead (worked; the edit tool requires a prior `view` of each file in-conversation).

## e) WHAT WE SHOULD IMPROVE

1. **Per-commit golden regen** for every API-touching task (now baked in; M10 downgraded to verification).
2. **Pre-commit hook cost awareness:** each authored commit pays ~1.5–2 min of BuildFlow (report-only noise incl. 235 baselined file-size findings + timing regressions that are cold-cache artifacts, not code regressions). With ~15 commits left in the plan that is ~30 min of hook time — acceptable, but batch related fine-tasks into one commit per medium task where the plan allows.
3. **Ledger hygiene:** tick TODO_LIST execution-ledger rows as tasks complete (M00/M03 owed), not at wave end.
4. **The BuildFlow "regression verdict" block is noise during cold-cache runs** — ignore unless a step is genuinely slower warm.
5. **mesh-demo/ still unruled** (carried from the postmortem; flagged again in g).

## f) NEXT TASKS (ranked, no padding — 22 real items)

1. **M04** — ADR-0166: `record.StreamRef = id.StreamRef` alias; collapse StreamID+StreamType → single `Stream` field; validating `NewStreamRef`/`ParseStreamRef` (empty type ILLEGAL, zero ID ILLEGAL); StreamKey disk truth; adapt bridge call sites (NewStreamRefOrZero dies); flip ADR-0166 Proposed→Accepted + `#check-adr-numbering`; golden regen; commit. **This checks off cut-readiness Layer 8 (its only row).**
2. **M05** — Payload{Data,Enc} + EncodingEnvelope; tri-state tests (absent/opaque/wrapped); recordEncoding bridge one home; golden regen; commit.
3. **M06** — bridge stamps: Received=NewStamp(intake clock), Created=client clock only; query Stream zero (kills fabricated ref); zero-struct guards in Reconstruct/AsRecord; tests; commit.
4. **M07** — `cause record.Cause` field on ImmutableEvent stamped by WithCommandCausality; Causation.CommandType → record.Type; decide event.Version→record.Version alias + R11 method shrink; commit.
5. **M08** — Tracing carries structural Actor{Kind,Raw}; ActorString precedence documented as THE rule; round-trip tests.
6. **M09** — command/query twins aligned (brands, zero-Stream, Version=0); trinity contract test.
7. **M10** — verification pass: TestEvery, module-count sanity, `#check-untested-exports --update` (new symbols must be test-named).
8. **M23** — v4-negative check: `git diff --stat` over the nine v4 module dirs (excl. core/v5) = empty; golden diff scoped to core/v5 section; receipt rowed.
9. **M11** — v007 v5-twin table drift scans; update changed replacement texts (AsRecord rows); NO twin deletions.
10. **M12** — doc-check canonical run (six mirror aliases still informational); md-go dry-run.
11. **M13** — `#check-duplication` post-divergence triage (specview surgical insert only if genuinely new shapes); `#check-file-size`.
12. **M14** — cqrs-upgrade `rewrite_v5.go`: StreamRef ctor rewrites, Type coercions, Causation→cause field, query zero-Stream; fixture tests + apply-mode smoke.
13. **M15** — V5-MIGRATION-GUIDE before/after sections (brands, StreamRef, Payload, Cause, stamps).
14. **M16** — DOMAIN_LANGUAGE: single StreamRef row (ADR-0166), Record/CommonMetadata core/v5 rows; doc-check exit 0.
15. **M18** — AGENTS.md: deep-dive row for the 11-19 review + this plan; core/v5 free-window fact; v007-v5-twin-table pointer.
16. **M19** — docs/reviews README (series index) + read the 10-17 findings table + 2 real cross-refs.
17. **M24** — enumerate v4 metaengine/system record.Record field consumers; map onto the retype; design notes on TODO_LIST :2585.
18. **M20** — scoped suites (core/v5, cqrs-upgrade, api-stability, cqrs-lint v007) → `GT_LINT_MODULES=... #lint-scope` → full `nix run .#verify` (exclusive window); receipts rowed.
19. **M21** — first-green-master-CI watch + untagged-API gate observation (BLOCKED on push).
20. Tick ledger rows M00/M03 (+ later tasks) in TODO_LIST.
21. CHANGELOG `[Unreleased]` row for the core/v5 type surface at wave close (cite `record.InstanceID` etc.; `check-changelog-symbols` gate).
22. `#check-package-surface` on core/v5 (new symbols count against the 150/new-package ceiling — record is an existing package, but verify no gate surprise at M20).

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Push authorization:** master is 6 authored commits ahead of origin (746MB-free planning + first code wave). M21 (first-green-CI watch, the window "closing loudly") is impossible without a push. Push now, after Wave 1 (M04+M05), or only when the whole plan lands?
2. **D32 ratification:** the three rulings are executing per recommendation under the blanket order — (A) deprecated twins stay until T26, (B) ADR-0166 flips Accepted when M04 lands (vs. at the full cut), (C) brand-now over string-compat. Veto any, or ratify as executed?
3. **`mesh-demo/` disposition** (carried owner question): the empty untracked directory is still unruled — trash it, keep it as staging for a sibling project, or move it out of the repo?

---

**Resume point:** M04, exactly as specified in plan §4 (M04·a–f); the M00 calibration note (§8) holds the measured fallout map (bridges = the work; vet = the counter; golden = per commit).
