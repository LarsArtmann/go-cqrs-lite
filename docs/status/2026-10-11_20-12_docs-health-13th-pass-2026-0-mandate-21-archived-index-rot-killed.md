# Status Report — 13th Docs-Health Pass: `**/2026-0*` Mandate, 21 Files Archived, Index-Rot + Gate-Debt Killed

**Written:** 2026-10-11 20:12 CEST
**Session scope:** the standing mandate — "View ALL `**/2026-0*` files, execute docs-health PROPERLY, six living docs SUPERB, archive fully-done files with inline strikethrough." No unrelated work researched.
**Companion artifact:** the inline health report (Accuracy 2.25 / Fitness 8.5, as-found scoring) printed in-session per `health-report-format.md`.
**Tree state at close:** clean (the auto-commit daemon absorbed the pass as a series of `chore:` commits, ending `de3eea380`); no authored commits (harness contract).

---

## a) FULLY DONE (verified green at close)

1. **Skill loaded properly:** docs-health SKILL.md + governing references (harvest-guide, verify-checklist, health-report-format; annotate.go toolkit) read up front; all bulk table annotation ran through `annotate.go` with `--dry-run` first (50-row §f table on the 09-30 report: dry-run → real run → shape-verified).
2. **Full inventory of the mandate scope:** every non-archived `2026-0*` file enumerated (57: 12 status, 12 planning incl. .d2/.svg/.html, 9 reviews, 10 benchmarks, 7 architecture-understanding, 3 research, 3 raw bench `.txt`) and classified per the skill table. Archived dirs stayed closed per the standing rule.
3. **ANNOTATE — 7 status reports fully resolved** (2026-09-28_15-10, 09-28_17-57, 09-29 ×4, 09-30_15-31): every one bannered `RESOLVED-BY-ROUTING (docs-health 13th pass)` with **255 inline strikes** (32/18/24/34/73/16/58), each citing its landing receipt (successor report §, issue-closure timestamp, tag, or TODO row). The 09-30 report's 50-row §f table struck via `annotate.go rows`; its §b/§c/§g prose via `annotate.go prose`; 4 rows are `r`-kind routed verdicts (bare line + `→` pointer, by tool design).
4. **ANNOTATE — 6 planning/review docs bannered with dated verdicts:** goal-closure plan (EXECUTED), direction-ruling memo (RULED via ADR-0151), evolution-fold audit (point-in-time), publish-integrity plan (EXECUTED), post-wave plan (EXECUTED), cv-verdicts reflection (ROUTED to TODO_LIST §CV).
5. **ARCHIVE — 21 `git mv`s + manifest:** status ×7, planning ×8 (incl. the 09-30 plan `.d2`/`.svg`/`.html` trio), reviews ×6 (4 full-code-review dashboards + dogfooding HTML under the standing 5th-pass exemption + cv-verdicts). The one-line-per-file manifest lives in the 13th-pass ledger entry in `docs/status/README.md`.
6. **LEAVE-ALIVE classifications recorded inline:** the 4 standing KEEP-LIVE/SKIP status docs (09-13 ×2 evidence, 09-17 fp-sweep baseline, 09-21 exempt HTML, 09-23 crash draft), 5 binding/owner-pending planning docs (350-line memo, substrate proposal, AggregateOn + single-writer-lease one-pagers, clock-seam design), 3 decision memos in reviews (ratification, ADR-0139 owner memo, pebble/bbolt scope). The 12th pass's open §g1 ruling (benchmarks/architecture lifecycle) was ANSWERED inline: LEAVE-ALIVE evidence/mappings — recorded in the archived 12th-pass file itself.
7. **Status live-index rot killed (the pass's Critical):** 32 missing rows added (2026-10-09_06-16 → 2026-10-11_19-27; 31 built from inventory + 1 caught by the gate), 7 archived rows removed, header → 2026-10-11, 13th-pass ledger written. `check-canonical-facts` status leg: missing=0 dangling=0.
8. **Archived-count truth restored:** intro claim 1,267 → **1,274** (gate-derived; the pre-existing off-by-one — dir held 1,268 before this pass — died here); day-table gained `2026-10-11 | 21`.
9. **HARVEST executed:** 4 new TODO rows (🔥 idempotency-retirement ADR + decision routing; 🔥 `cmd/*` gate-coverage decision; gci re-add durable guard; #26 publish-watch residue), 3 fully-done receipt rows deleted per header policy (W0/W1/W4 — W0's own "delete at next pass" marker honored), header ledger extended with the 13th pass.
10. **Living docs repaired:** ROADMAP (98→104 `go.mod` recount with a gate-blind-spot note; `[Unreleased]` window 09-06..28 → 09-06..10-11; new `2026-09-29..10-11` release-history segment), AGENTS (103→104 go.mod / 99→100 modules; 86/86→98/98 recipes; #6 extended with the 10-10/10-11 gci re-add evidence), ADR-0065 status drift fixed in BOTH index rows ("Proposed" → "Accepted (partially executed — see Addendum)").
11. **Broken-gate + broken-link debt cleared:** `#check-md-go` was UNBUILDABLE for the whole repo (md-go-validator `?ref=master` moved past the 10-04 vendorHash) — re-pinned via the fake-hash dance, gate green (1509 blocks). 23 broken links in `core/v5/*/README.md` (written at root-depth, modules are nested two deep) rewritten; 11 doc-links fixed (8 wrong-depth skill-reference links from the 10-10 wave, the 0157→0123 slug, 3 archival-move repoints); 17 deprecated-symbol citations resolved (deprecation banners on `idempotency/{kvstore,sqlstore}/README.md` matching the gate's exact pattern + `kv.Store` qualification in the root README).
12. **Fixes on sight:** `2026-09-30_14-28_github-issue-plan.svg` permission 600→644 (the 09-30 report's own d-2 bug); stray gitignored `event/event.test` binary trashed; inbound references to all 21 archived files repointed (TODO_LIST ×6, ADR-0151, ADR-0157, gt03 memo, 4 newly-archived reports' internal links, 1 already-archived review).
13. **All 7 doc gates green at close:** canonical-facts (104 go.mod, live index clean, archived 1274=1274, engines 12/11/11) · doc-links 1038 targets / 0 broken · readme-links 834 / 0 · readme-deprecated clean (0 new, 0 grandfathered) · doc-check 1,284 refs / 57 packages · check-md-go 1,509 blocks · adr-numbering 155 ADRs in lockstep.
14. **Health report printed inline** with as-found scoring, visible substitution math, and the fixes-reported-not-subtracted ruling honored (Accuracy 2.25 = 10 − 4·1 − 7·0.5 − 1·0.25; Fitness 8.5 = 10 − 0.75 − 0.75).

## b) PARTIALLY DONE

1. **"View ALL" depth is uneven by role, and stated as a ruling:** benchmarks (10 .md), architecture-understanding (7), research (3) were opened, classified, and freshness-checked via the link/canonical gates — but NOT line-by-line claim-verified. The LEAVE-ALIVE ruling is recorded inline; the bodies' internal claims (e.g. the superseded 07-31 backend comparison) were not individually re-proven.
2. **The ~28 live 2026-10-01..11 reports:** outside the letter of the `2026-0*` mandate. The newest 3 were read in full (HARVEST sources); the rest got honest filename-derived index rows but their bodies were NOT read, and they were NOT annotated/archived. They are the next pass's obvious backlog.
3. **TODO_LIST "superb" = structural surgery, not a full audit:** rows deleted/added/ledgered, but the ~110 open rows across 28 sections were not re-verified one by one against code (the pass relied on the row-level receipts already present + the gates).
4. **FEATURES.md:** counts are gate-pinned and verified green; the row-by-row status truth still rests on the 09-28 M7/M15 sweeps rather than a fresh sweep.
5. **doc-check's 6 pre-existing ambiguity warnings** (advanced/readmodels/recipes aliases mapping to both v4 and `core/v5` packages, "verified via union") left as-is — cosmetic-but-real, needs import-scoping in the fences.
6. **§g2 of the 12th pass (strike-depth contract):** answered with a WORKING ruling (banner-routing + individually-classified items = closure for §f brainstorms), recorded inline — still not owner-ratified.
7. **`nix fmt` over this pass's markdown:** not run — the 6-passes-carried tail (is markdown in treefmt scope?) remains unresolved and was deliberately not re-litigated mid-pass.

## c) NOT STARTED (deliberate or out of scope)

- Per-module test suites / `#verify` / race — docs-only pass; the composed `#verify` remains quiet-gated (load never cooperated since 09-20).
- The owner question stack: push `master` (branch was ahead 4 at check time), quiet-window strategy for M04/M05/M11/M14.3, EventCatalog crash draft + turso A+B/a-b-c filings (all owner-gated, all routed).
- The idempotency-retirement ADR itself — harvested and rowed (TODO 🔥), not written.
- The 2026-10-11_19-27 triage session's own follow-ups (rerun `#build` to green, `TestRenderDoctorFeatureProfile` diagnosis, parallel-session ownership) — that session is live; its items stay its own.
- CHANGELOG entry for the pass — deliberately none, matching the docs-only pass convention (10th/11th/12th precedent).

## d) TOTALLY FUCKED UP (honest ledger)

1. **A dead loop that looked like work: my core/v5 link-fix script ran "successfully" and changed 0 files.** Root cause: `target.split("/")[0]` on a path starting with `../` yields `".."`, so the existence condition was trivially true-but-useless and nothing matched — yet the script printed clean output. It took a standalone `findall` probe, a condition probe, and an `iter` probe (three wasted rounds) to see it. The standalone probe WORKED, which is exactly why the loop bug hid: I tested the regex, not the loop's data path.
2. **Trusted recomputed arithmetic over the gate — the exact 12th-pass d1 class.** I computed the archived count as 1,275 (from `ls | wc -l` = 1268 + 7 moves) and pinned it; the gate's derived count said **1,274** (`ls` counts entries, the gate counts files). Caught by the gate on the next run, not by me. Gate-derived numbers beat hand arithmetic — I KNEW this rule and still reached for `wc -l`.
3. **My inventory had a hole the gate found:** the 32-row index list missed `2026-10-09_06-16` because I built rows from a `ls | tail -25` slice instead of the sorted `find` I had used for the 2026-0* inventory. Inconsistent sourcing, silent truncation, gate caught it.
4. **A script that died mid-way, twice:** the link-fix script's trailing assert fired on a SECOND unpatched occurrence in recipes.md — aborting before the ADR-0151/0157/gt03/event-command fixes ran; and the index-row insertion script asserted on a wrong anchor pattern (`[filename](` vs the file's `[Title](`), leaving the count fix unapplied. Both were only caught because I re-ran the gates; neither partial state shipped, but both cost a re-run.
5. **multiedit failed 2-of-3 on the 04-58 report** because I reconstructed §f/§g text from my earlier read instead of copying the exact view (item 17's wording differed by one word; §g2's bold placement differed). Re-viewed, retried, landed — but the failure was preventable by following my own exact-match discipline the first time.
6. **Composed the deprecation banner from memory of the gate's pattern** (`> **⚠️ DEPRECATED MODULE…`) instead of copying the gate's regex (`^> ?\*\*Deprecated`) — 16 of 17 citations stayed flagged until a rewording round. The gate's self-test fixture literally documents the accepted shape; I didn't re-read it before writing.
7. **Minor:** typed "PARTYAL" into a strike verdict on the 12th-pass report (caught and fixed in the next edit); my first `for` loop over archived files globbed in the project root (bare `2026-09-28_15-10*` matched nothing) producing seven noisy grep errors before I re-grounded the paths.

## e) WHAT WE SHOULD IMPROVE (process/systemic)

1. **Gates-first baseline:** a docs-health pass should run ALL doc gates BEFORE the first edit, so the findings table is complete on arrival (this pass discovered gate-red states one by one while working — the 12th pass's e5 recommended exactly this; it is still not habit).
2. **One inventory source, always sorted `find`** — never `ls | tail` slices — for any list that feeds row/manifest generation (d3).
3. **Gate-derived numbers only** for any count a gate pins; hand arithmetic is for scratch, never for docs (d2).
4. **Test the loop's data path, not just the pattern:** one `repl()` invocation on a real match inside the real loop would have killed d1 in seconds.
5. **Fix-scripts must be resumable:** asserts first (validate ALL targets), then mutate — or per-file try/except with a summary — so a mid-script abort can't strand half the fixes (d4).
6. **Copy gate-matching patterns from the gate's own fixture/self-test,** never from memory (d6).
7. **Exact-match discipline applies to text I read an hour ago:** re-view before every multiedit on prose (d5).
8. **The index rot class needs a ritual, not vigilance:** 32 rows accrued in ~2 days because no session runs the status-index leg. Candidate: session-close `canonical-facts` (or a fast index-only variant) as a standing checklist item.
9. **The `?ref=master` flake-input class needs a policy:** md-go-validator broke every consumer of `#check-md-go` silently for days. Pin to a rev + drift-watch, or accept chase-the-master with scheduled vendorHash checks (question g3).

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT (impact-ordered; ★ = rowed in TODO_LIST this pass)

**Owner rulings (unblock chains)**
1. ★ Ratify or reject the strike-depth ruling (12th-pass g2): banner-routing + classified strikes vs full §f sweeps — decides whether "sweep the unstruck remainders" passes are ever real work.
2. ★ Confirm the benchmark/architecture-understanding LEAVE-ALIVE ruling (12th-pass g1, answered inline this pass).
3. ★ AGENTS.md second-split decision (61 KB, ~1 contract/month growth): keep indexed-split style or split contracts into `docs/agents/contracts.md`.
4. Push `master` (re-check ahead-count first; asked by four consecutive reports).
5. Quiet-window strategy for M04 composed `#verify` / M05 MySQL legs / M11 calibration / M14.3 stamps (M15.1 pack still pending).
6. Fate of the six remote branches (merged/abandoned analysis → delete/close).
7. gci re-add loop: BuildFlow config-skip vs hook vs CI check (★ rowed; mechanism is the owner's call).
8. ★ `cmd/*` gate coverage: add to `testModules`/`lintModules` (+`check-arch`) or document the exclusion in module-map.md.

**Directly unblocked work from this pass**
9. ★ Write the idempotency-retirement ADR (retire `idempotency/{kvstore,sqlstore}` → `system`-exposed dedup facade over `metaengine.DedupStore`); `check-adr-numbering` before AND after; then the ADR-0152 ghost-verdict addendum.
10. Cut the next `stack/postgres` tag carrying `retract v4.2.0` → close #26 with the proxy receipt.
11. File the EventCatalog agent-changelog crash draft upstream (owner approval; verify-before-filing + github-voice).
12. File the turso A+B and turso-go a/b/c verified drafts (owner-held since 10-01).
13. Scope-import the 6 doc-check ambiguity warnings (advanced.md:50/763, readmodels.md:113, recipes.md:69 class) or add an accept-marker convention.
14. The 19-27 session's tail: rerun `source scripts/go-env.sh && nix run .#build` to a green conclusion; diagnose `TestRenderDoctorFeatureProfile`.
15. ★ Re-verify the 26 rewritten `core/v5` link TEXTS read correctly in v5 context (I fixed targets, not prose).
16. Sweep active docs for bare-name citations of other idempotency symbols (the banners only cover the two module READMEs; root README's `Store` was one collision of a class).
17. Decide whether README's versions-matrix row `idempotency/v4.4.0` (parent module deleted, ADR-0128) needs a retirement note under the tag-truth policy.
18. Run the gates this pass didn't (docs-only pass): `#verify`, `check-csp`, `check-eventcatalog`, `check-duplication`, vulncheck — next quiet window or next full-session close.

**Docs-health machinery (kill the recurrence classes)**
19. Fast session-close status-index check (canonical-facts is heavyweight; a missing=0/dangling=0-only variant would be ~1s) — or wire the existing leg into a closer ritual.
20. Teach `check-canonical-facts.sh` the backticked `` `N go.mod` `` form (the ROADMAP 98→104 drift was invisible to it; currently compensated by an inline note).
21. Pin or watch `?ref=master` flake inputs (md-go-validator class; see g3).
22. Recalibrate or silence the "N live reports (>10)" NOTE — 72 live reports is this repo's steady state, the threshold just cries wolf.
23. Decide the July-era archived files lacking `~~` strikes (240+ pre-convention bulk archives): rule them exempt or schedule a conversion sweep.
24. Add planning/reviews archived-dir local manifests (this pass's manifest lives in the status README; discoverability inside `docs/planning/archived/` is zero).
25. Codify the claim-marker convention into the crush-config docs-health skill (12th-pass f16, external repo, commit-gated).
26. Upstream this pass's lessons to crush-config `references/lessons.md`: "gate-derived numbers beat recomputed ones"; "test the loop's data path"; "gates-first baseline" (by commit, per memory rules).
27. Extend `check-readme-deprecated` self-test fixture into the docs (the banner pattern `^> ?\*\*Deprecated` is documented only in the script).
28. Health-report format: add a mandatory "as-found gates snapshot" section (the gates-first baseline output) to `health-report-format.md`.
29. Consider gating ROADMAP/AGENTS per-day history segments less densely (the new 09-29..10-11 segment is one mega-bullet; fine now, painful at v5-cut time).

**Deep-verification debts (the "VIEW ALL" letter, next pass)**
30. Line-by-line claim check of the 10 benchmark .md bodies (07-31 backend-comparison vs 09-19 variation doc first).
31. Line-by-line claim check of the 7 architecture-understanding bodies (cordis mapping is AGENTS-referenced → highest stakes).
32. Line-by-line claim check of the 3 research docs (graph-DB survey is 1,299 lines; the turso IVM draft stays owner-gated).
33. Read + annotate + archive the ~28 live 2026-10-01..11 reports (the ones this pass indexed but didn't read) — they are now the accumulated backlog the 13th pass just cleared for 09-2x.
34. Full row-by-row TODO_LIST freshness audit (~110 open rows / 28 sections) — this pass did structure, not rows.
35. FEATURES row-by-row re-verification beyond gate-pinned counts.
36. Verify no OTHER active docs reference the 6 archived reviews by path (this pass grepped a fixed filename list; a symbols/anchor-level sweep would be exhaustive).

**Downstream doc work surfaced in passing (from this session's reads, no new research)**
37. watermill skill: confirm backends.md's NATS plugin-gap entries match the test-file's documented workarounds 1:1 (M19 tail landed them; parity unverified).
38. `example/graph-native` entry-point choice: skill refs now link `domain.go` (my pick); confirm the recipes §2.44/advanced §6.13 authors would rather route to `composition.go`.
39. ROADMAP release-history: split the 09-29..10-11 mega-bullet per-day when the v5 cut forces a rewrite anyway.
40. TODO_LIST §CV section: the archived reflection pointer now resolves; confirm the section's BLOCKED lease row still cites the right ADR number post-ADR-0151/0152 reshuffle.
41. AGENTS quick-reference: the fp-sweep baseline doc is superseded-in-place (09-29); confirm the AGENTS pointer still targets the right section of that file.
42. status-report skill: this report is `.md` per your explicit instruction (HTML is the skill default) — same one-off override as the 09-30 and 10-11_19-27 reports; either bless `.md` for triage-style reports or stop overriding.
43. `idempotency/kvstore/README.md:105` (core/v5/kv points back out) — the chain this pass bannered; fold into the ADR work (item 9) rather than patching piecemeal.
44. Consider a `chore:`-commit tag convention for daemon-absorbed docs passes (reconstruction cost: this pass's history is 6 `chore:` commits).
45. The `2026-10-11_19-27` report was untracked at session start and is now daemon-absorbed — verify it's actually committed (I saw tree clean at close; confirm the file is IN a commit, not gitignored).
46. Sweep for other gitignored stray binaries like `event/event.test` (the 19-27 report's d1 daemon-committed-binaries class; the zip-guard only fires at cut time).
47. AGENTS gotcha candidate: "`ls | wc -l` counts entries, not files — archived-count class" (d2/d3 lessons into `gotchas-tooling-build.md`).
48. Improve `annotate.go` docs: the `r` kind leaves rows bare by design — document it where `check-rows` users will see it (its INCOMPLETE verdict on routed rows is noisy-but-correct).
49. Decide whether `check-rows` should treat a uniform `r`-verdict table as COMPLETE (tooling nit surfaced by the 09-30 annotation).
50. Then: resume the TODO_LIST's own priority order — v5 declarative schema evolution (M9+), BDD harness wave tails, graph-native recipe-polish trigger (CRM debut fired it), strong-ID wave follow-ups.

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF (3)

1. **Strike-depth contract (12th-pass g2, carried):** is "RESOLVED-BY-ROUTING banner + every item individually classified (resolved→struck with receipts, open→routed to TODO_LIST)" sufficient closure to archive a report — or must a future pass literally strike every line of §f brainstorms? This pass used the former; your ruling decides whether item-class sweeps ever exist.
2. **Scope boundary for the next pass:** the ~28 live 2026-10-01..11 reports were indexed but not read (outside the `2026-0*` letter). Should the NEXT docs-health pass process ALL live reports regardless of date pattern (my recommendation — the 09-28 12th pass did exactly that for its window), or do you want a `2026-10-*` sweep as its own explicit mandate?
3. **The `#check-md-go` vendorHash treadmill:** the gate's source input is pinned `?ref=master`, so every upstream push can (and did, twice) break the gate repo-wide until someone re-pins. Want me to pin the input to an immutable rev + add a nightly drift-watch leg, or keep chase-the-master?

---

_Point-in-time snapshot. No authored commits (harness contract) — the daemon absorbed the pass. The inline health report (Accuracy 2.25 / Fitness 8.5, as-found) is the scorecard; this file is the ledger. Awaiting instructions._
