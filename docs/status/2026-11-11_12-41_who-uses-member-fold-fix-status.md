# Status: who-uses Member-Fold Fix — Implementation Session Report

**Date:** 2026-11-11 12:41 CET
**Session scope:** continuation of the 09:16 cmdguard forensics session. Lars ruled
*"who-uses is a fucking ~/projects/project-dependency-graph/ responsibility"*; this phase
implemented, tested, verified, and documented the attribution fix in
`~/projects/project-dependency-graph`, then annotated the 09:16 status report in go-cqrs-lite.

## Scope line — VERIFIED vs NOT verified

**Verified this phase (command/artifact for each):**

| # | Claim | Evidence |
| - | ----- | -------- |
| V1 | Fix compiles + passes across all 6 modules under race | canonical loop `for m in . discovery graph renderer types testutil; go build ./... && go test -count=1 -race ./...` → all ✅ |
| V2 | cmdguard tree: 77 → 34 consumers, 0 `via go-cqrs-lite` rows | `/tmp/pdg-fixed who-uses cmdguard --direct-only=false` → 32 direct + 2 indirect; `grep -c 'via go-cqrs-lite'` = 0 |
| V3 | A/B against pre-fix code (binary built at commit `6b80811f` via `git worktree`) | cmdguard 77→34; cqrs-htmx 26→26 unchanged; go-cqrs-lite submodule counts −1 phantom each, `cmd/cqrs-lint` 58→2; go-appkit 34→33 with corrected `via` member |
| V4 | The 2 surviving indirect rows are module-graph-true | hand-checked on disk: `go-credentials-store/go.mod:8` (cmdguard for its in-module CLI), `samber-linter/go.mod:8` (cmdguard CLI), `branching-flow/go.mod:25` (samber-linter dep) |
| V5 | `why` answers truthfully post-fix | `why 'KeyHolderAI->cmdguard'` → "no path" |
| V6 | No debug residue shipped | `rg PDG_DEBUG` clean; `reach_debug_test.go` absent from HEAD; `git status` clean |
| V7 | Lint clean on changed files | `golangci-lint run` in graph/discovery/types → 0 issues on my files (2 pre-existing `wrapcheck` findings in `dep.go:177`/`focus.go:18` — NOT mine, untouched) |

**NOT verified (and why):**

- **Graph-layer paths** (`serve`, `generate`, `diff`, built-graph BFS) — only the query fast-path
  (`ConsumersFromModules`) + discovery were made member-aware; the built GRAPH still contains the
  aggregate's merged-union edges. The empirical `why` result suggests low impact; structurally
  unaudited (b-item).
- **`nix flake check`** (repo's sanctioned gate) — not run; manual build+vet+test+race loop used
  instead (devShell tooling availability).
- **A dedicated regression test for the production bug shape** — the zero-dep-member trap
  (`metadata → record`, record having zero requires) that tests-green missed and real data caught
  has NO synthetic pin in the suite (b-item, top next-task).
- **Morning-session disk facts not re-read post-noon** (e.g. KeyHolderAI go.sum) — AND two
  file-read contradictions during this phase suggest **parallel sessions were actively mutating
  `~/projects` repos** (see d/e below); late reads were double-checked, early ones carry a
  time-of-check caveat.
- **CHANGELOG release cut** — fix sits in `[Unreleased]`; no tag (version authority is a pending
  maintainer decision per this repo's AGENTS.md).

## Self-review: what I forgot, what could be better (brutal, session-scoped)

**Forgot / process misses:**

1. **Tests green ≠ end-to-end done — I nearly shipped a half-fix.** After the suite went green my
   first end-to-end run still showed **38 phantom rows**: the zero-dep-member trap (`record` has
   no requires → absent from the provenance index → its keys fell through the aggregate-union
   fallback → `metadata → go-cqrs-lite(union) → cmdguard`). My own fixtures never modeled a
   zero-dep member being REQUIRED by another member. The original reproducing command is the
   only oracle; I ran it too late.
2. **Synthetic reasoning before real data.** I debugged the union-fallback leak through ~4 rounds
   of synthetic probing and grep-anchor fumbles before writing the decisive one-off probe against
   the REAL discovered workspace (`TestDebugMemberDeps` → `memberDeps=98, record MISSING` —
   solved it in one shot). Real-data probe should have been step one.
3. **Python string-patching for code edits** (debug instrumentation + removal) — first patch
   silently failed on imports, another used a regex; assert-guards saved me, but this pattern is
   exactly how silent no-ops slip in. The `edit` tool or throwaway test files are the honest way.
4. **A corrupted shell command shipped mid-session** (`gci write --skip-comment生成` — CJK
   artifact, stderr suppressed). Harmless only by luck. Command hygiene matters when `2>/dev/null`
   is involved.
5. **Two contradictory disk reads accepted without immediate re-check** (a `go.mod` that matched
   rg then "didn't exist" minutes later; samber-linter's cmdguard line found by grep but not an
   earlier rg). Root cause: **parallel sessions were mutating repos concurrently**. I flagged the
   pattern but kept building on reads without timestamp discipline.
6. **`why` CLI syntax guessed** (positional pair, not `from->to`) — wasted a round; read the
   command's contract first.
7. **TODO_LIST harvest still not done** — the 09:16 report's (f) list and this one's are
   entombed in timestamped files; docs-health HARVEST remains undone for both.

**Could be better:**

8. Prove the fix shape with a real-data probe BEFORE writing synthetic fixtures.
9. Pin every production bug shape with a synthetic test at fix time (the record trap has no pin).
10. Treat the original reproduction command as the definition of done, not the test suite.
11. Re-read any disk fact that will be cited when concurrency is known.
12. Run the repo's canonical gate (`nix flake check`) when feasible, not only the manual loop.

## a) FULLY DONE

1. Attribution fix shipped: `types.Module.MemberDeps` provenance (discovery) + proven member
   folds with identity rule (graph) — `who-uses` can no longer inherit repo-tooling deps onto
   library importers.
2. Zero-dep-member trap fixed (store ALL members incl. empty) + root-caused via real-data probe.
3. Regression suite: 4 new tests (reachability, sibling-member closure, unknown-provenance
   refusal, indirect-pin trace via reaching member) + fixture updates; fixtures now carry
   `MemberDeps`; identity-case test doubles as the no-provenance identity pin.
4. A/B verification vs pre-fix binary across 5 targets (V3) — 45 phantom rows gone, zero intended
   folds lost.
5. Surviving rows hand-verified true against disk manifests (V4).
6. `why KeyHolderAI->cmdguard` → "no path" (V5).
7. Docs: project-dependency-graph CHANGELOG `[Unreleased] Fixed` (with incident numbers),
   README ×2, AGENTS.md incident note; go-cqrs-lite 09:16 status report addendum with the full
   A/B table.
8. Concurrency-aware handling of the foreign flake.nix edit: investigated, NOT touched, flagged
   (never revert/rebase others' in-flight work).
9. Debug instrumentation fully removed (V6); worktree cleaned up.

## b) PARTIALLY DONE

~~1. **Zero-dep-member regression pin** — the exact metadata→record shape has no dedicated~~ routed — pdg member-fold follow-ups TODO row (names the metadata→record pin, open)
   synthetic test (behavior is fixed and incidentally covered; the pin is missing).
~~2. **Graph-layer bleed audit** — `graph/build.go` edges still carry aggregate-union deps;~~ routed — pdg member-fold follow-ups TODO row (serve/diff/generate/why, open)
   member-aware paths for `serve`/`diff`/`generate`/`why` unaudited (empirical why-check only).
~~3. **Commit-policy remediation** — daemon swept a FOREIGN flake.nix edit (cmdguard pin~~ routed — pdg member-fold follow-ups TODO row (§b3 foreign-pin commit flagged; reword only when no parallel session)
   tag→SHA `56fd69b4`, likely a parallel session) into heuristic commit `05e5b1ae` together with
   my source changes; that commit's body violates the repo's detailed-body policy for flake.nix
   commits. Flagged; no rebase (concurrency detected — rewriting history under a parallel session
   is how work gets destroyed).
~~4. **Concurrency anomalies** — go-design-smells/branching-flow go.mod appearance/disappearance +~~ routed — pdg member-fold follow-ups TODO row (§b4 bounded, not chased)
   samber-linter cmdguard line found/lost between reads. Noticed, bounded (late reads double-
   checked), not chased to the owning session.
~~5. **`nix flake check`** — manual loop run instead; sanctioned gate not executed.~~ routed — pdg member-fold follow-ups TODO row (open)
~~6. **TODO_LIST harvest** — neither status report's (f) has been harvested.~~ done — M31/F194 harvest DONE 2026-11-13 (both reports)
~~7. **go.work stale sibling entries** in project-dependency-graph (`../go-cqrs-lite/catalog` —~~ routed — pdg member-fold follow-ups TODO row (open)
   pre-restructure paths) — identified as warning-flood source, not fixed.
~~8. **Staging-repo hygiene** from the morning report (crm-stage-t13b, cv-stage-crmfix,~~ routed — TODO_LIST:147 (staging/fork owner row, open)
   erraudit-src, ci-siblings fork) — still broken, untouched.

## c) NOT STARTED

~~1. Dedicated zero-dep-member regression test (item b1).~~ routed — pdg member-fold follow-ups (same as b1)
~~2. Member-aware graph layer (`why`/`serve`/`diff` paths crossing aggregates).~~ routed — pdg member-fold follow-ups (b2/c2, open)
~~3. Release/versioning of project-dependency-graph carrying this fix (version-authority decision~~ routed — pdg member-fold follow-ups (version-authority decision + release, open)
   blocks it — tags end at v0.2.0, CHANGELOG says 0.7.0-dev).
~~4. Post-fix fleet re-census / re-run of adoption reports that cited the inflated numbers.~~ routed — TODO_LIST Post-fix census re-run row (12-41 f43/f44, open, M54)
~~5. `--attribution=strict|repo` flag, `viaKind` field in JSON/SARIF outputs (idea-tier from the~~ routed — pdg backlog (routed not minted: idea tier f5)
   morning list).
~~6. Fixture/golden pinning of the 34-consumer cmdguard tree as a named regression fixture.~~ routed — pdg member-fold follow-ups (f2/f6, open)
~~7. Verification that go-cqrs-lite's own `census-fleet.sh` numbers were NEVER affected (it uses~~ routed — TODO_LIST Post-fix census re-run row + TODO_LIST:144 census-vs-who-uses note
   module-graph tooling directly — presumed independent of this tool, unverified).
~~8. Re-verification of morning disk facts (KeyHolderAI go.sum etc.) under the concurrency caveat.~~ done — declined-with-reason in harvest ledger (sampling class proven by A/B + disk hand-checks)

## d) TOTALLY FUCKED UP

**Nothing shipped broken** — final state is verified green end-to-end with the original
reproducing command, and nothing was destroyed (no reverts, no history rewrites, foreign work
preserved). The honest near-miss class, stated plainly: **I declared an intermediate state
fixed on the strength of a green test suite while the original bug still reproduced at 38
phantom rows.** One more end-to-end step and I would have reported false success in chat. That
is precisely the failure class this fleet's own docs codify against ("release verification must
exercise real behavior, never just resolution/build") and I walked into it anyway. The corrupted
`gci` command and the unchased file-read contradictions are the same family: verification
shortcuts under momentum.

## e) WHAT WE SHOULD IMPROVE

~~1. **Definition of done = original repro command, every time** — green tests are necessary, not~~ routed — process lesson codified: docs/agents/gotchas-testing.md "Direct sweeps are NOT the gate" row + release-verification rule (AGENTS)
   sufficient.
2. **Real-data probes first** when a bug survives synthetic reasoning; the one-off discovery
   probe outperformed four rounds of synthetic analysis.
~~3. **Regression-pin every production bug shape at fix time** (or immediately after, same~~ routed — codified: AGENTS "New goldens must be mutation-tested" + M37/F227 executed the class
   session).
4. **No python-string-patching of code**; use the edit tool, or commit-flavored throwaway tests.
~~5. **Concurrency discipline**: when parallel sessions are known-active, re-read every fact before~~ routed — codified: global AGENTS safety rules (never revert others' work; re-read before citing)
   citing it and never rewrite shared history.
6. **Prefer the repo's canonical gate** (`nix flake check`) over ad-hoc loops when available.
~~7. **Shell hygiene**: no `2>/dev/null` on commands whose failure matters; no pasted garbage flags.~~ routed — codified: A7 SIGPIPE lesson row (TODO_LIST:337) + piped-grep gotcha
~~8. **Harvest discipline**: (f) lists go to TODO_LIST at session end, not into report files.~~ done — M31/F194 DONE 2026-11-13

## f) Up to 50 things to get done next (BRAINSTORM — HARVEST must route with rigor)

**Immediate hardening (this fix):**
~~1. Add the dedicated zero-dep-member regression test (mid requires `record`-shaped empty member; assert no fold to aggregate) — closes b1/gap from the incident.~~ routed — pdg member-fold follow-ups (f1)
~~2. Pin the post-fix cmdguard tree (34 consumers) as a named regression fixture in the repo.~~ routed — pdg member-fold follow-ups (f2)
~~3. Audit `graph/build.go` + BFS (`why`/`serve`/`diff`) for aggregate-union path bleed; make paths member-aware or document the boundary.~~ routed — pdg member-fold follow-ups (f3)
~~4. Add a `viaKind` (module-edge | member-fold | indirect-pin) field to who-uses JSON/SARIF.~~ routed — pdg follow-ups + backlog
~~5. Add `--attribution=strict|repo` mode (strict = module-graph edges only, no identity folds).~~ routed — pdg backlog (idea tier)
~~6. Confirm `census-fleet.sh` (go-cqrs-lite) independence from this tool's attribution; re-run it and compare with who-uses numbers post-fix.~~ routed — TODO_LIST census re-run row (M54)
~~7. Re-run any adoption/delete-at-zero reports that cited pre-fix consumer counts; annotate the delta.~~ routed — TODO_LIST census re-run row ("verify docs claims citing fleet consumer counts")
~~8. Run `nix flake check` on project-dependency-graph (sanctioned gate, closes b5).~~ routed — pdg member-fold follow-ups (f8)
~~9. Re-verify morning disk facts under the concurrency caveat (KeyHolderAI go.sum, cqrs-htmx go.mods, go-appkit).~~ done — declined-with-reason (harvest ledger)
~~10. Resolve the flake.nix foreign-pin commit `05e5b1ae`: confirm ownership, then reword the body (interactive rebase ONLY when no parallel session is active).~~ routed — pdg member-fold follow-ups (f10; owner confirm before release)
~~11. Fix `project-dependency-graph/go.work` stale sibling entries (`../go-cqrs-lite/catalog` etc.) to post-restructure paths; kills the discovery warning flood.~~ routed — pdg member-fold follow-ups (f11)
~~12. Quiet the ci-siblings fork parse warnings by aligning skip dirs with census exclusions.~~ routed — pdg member-fold follow-ups (f12) + TODO_LIST:147
~~13. Add renderer golden coverage for aggregates carrying `MemberDeps` (JSON shape stability).~~ routed — pdg member-fold follow-ups (f13)
~~14. Add a doctor/summary line counting parse-error repos with names (they warn; nothing summarizes).~~ routed — pdg backlog (idea tier)
~~15. Self-test the hidden-indirect accounting (32+2=34) against `--direct-only` flips.~~ routed — pdg member-fold follow-ups (f15)

**Release & versioning:**
~~16. Decide version authority for project-dependency-graph (tags end v0.2.0 vs CHANGELOG 0.7.0-dev) — maintainer decision already pending.~~ routed — pdg member-fold follow-ups (decision, open)
~~17. Cut the release carrying this fix once 16 is decided; CHANGELOG `[Unreleased]` is ready.~~ routed — pdg member-fold follow-ups (blocked on 16)
~~18. Verify cmdguard pin (`56fd69b4`) vs `v5.0.0` tag intent with the cmdguard repo state before releasing.~~ routed — pdg member-fold follow-ups (rides 16; = g/Q1)
~~19. Tag-wave mechanics: if go-cqrs-lite modules consume the fixed tool version anywhere, plan the consumer bump wave.~~ routed — pdg member-fold follow-ups (f19)
~~20. Write the incident as a short ADR/postmortem in project-dependency-graph docs (77→34, root cause, semantics chosen) — changelog is not a postmortem.~~ routed — pdg member-fold follow-ups (f20)

**Fleet hygiene (carried from morning, still open):**
~~21. crm-stage-t13b: broken go.mod (cmdguard v5 path @ v4.2.1) — upgrade or trash.~~ routed — TODO_LIST:147
~~22. cv-stage-crmfix: broken go.mod (@ v4.1.0) — same decision.~~ routed — TODO_LIST:147
~~23. eventcatalog-hub/work/erraudit-src: broken go.mod (@ v4.2.0) — refresh or remove.~~ routed — TODO_LIST:147
~~24. ci-siblings/go-cqrs-lite fork: refresh-vs-delete decision (5 cmd modules on cmdguard v4 requires).~~ routed — TODO_LIST:147
25. Identify which parallel session(s) mutated repos mid-investigation (go-design-smells/branching-flow, samber-linter) — coordinate, don't collide.
~~26. Harvest BOTH status reports' (f) lists into TODO_LIST.md (docs-health HARVEST) — two sessions, zero harvests so far.~~ done — M31/F194 DONE 2026-11-13

**Tests & quality:**
~~27. Table-driven unit tests for `memberReaches` itself (cycle shapes, memo reuse, ancestor-fold fallback) — currently only exercised via `ConsumersFromModules`.~~ routed — pdg backlog (routed not minted: f26-f40 tier)
~~28. Property test: fold set ⊆ union fallback set (the fix must never HIDE a real edge; union is the conservative superset).~~ routed — pdg backlog (f26-f40 tier)
~~29. Parity test: `ConsumersFromModules` vs `Graph.Consumers` for member-keyed targets (the two query paths can drift).~~ routed — pdg backlog (f26-f40 tier)
~~30. Benchmark: `membersReachingTarget` cost on the real 239-module fleet (called per query; memoized but measure).~~ routed — pdg backlog (f26-f40 tier)
~~31. Fuzz `memberReaches` dispatch (cyclic MemberDeps unions across forked workspaces).~~ routed — pdg backlog (f26-f40 tier)
~~32. Cover `buildMemberDepsIndex` fork-collision union behavior with a two-aggregate-same-member fixture.~~ routed — pdg backlog (f26-f40 tier)

**Tool UX (idea bucket):**
~~33. `--via-detail`: full BFS path for attribution debugging.~~ routed — pdg backlog (idea tier)
~~34. Label aggregate roots in tree output (`go-cqrs-lite (go.work aggregate, 104 modules)`).~~ routed — pdg backlog (idea tier)
~~35. Warn when >N% of a target's consumers are indirect-via-fold.~~ routed — pdg backlog (idea tier)
~~36. Reconcile ADR-0152's fleet edge counts with post-fix attribution (pick one canonical source).~~ routed — pdg follow-ups (version authority) + TODO_LIST:144
~~37. Export who-uses JSON for the docserver fleet pages; single canonical consumer number.~~ routed — pdg backlog (idea tier)
~~38. Split app-tooling vs library edges in aggregates via `hasMain` (already tracked).~~ routed — pdg backlog (idea tier)
~~39. Re-evaluate the aggregate-union fallback: keep for provenance-less aggregates only (current) vs strict-stop (more precise, hides edges for non-go.work discoveries).~~ routed — pdg backlog (idea tier)
~~40. Extend the same provenance idea to Node/Rust/Python workspace discoverers if they model aggregates.~~ routed — pdg backlog (idea tier)

**go-cqrs-lite side (light, post-ruling):**
~~41. One-line AGENTS.md pointer: consumer-count questions → project-dependency-graph semantics (proven folds), not census numbers — avoids the next session re-deriving.~~ routed — TODO_LIST:144 (fold-artifact docs pack)
~~42. Annotate `docs/agents/v5-fork-and-delete-playbook.md`: delete-at-zero triggers must cite census (strict), never who-uses.~~ routed — TODO_LIST:144
~~43. Re-run `bash scripts/census-fleet.sh --zero-consumers` post-fix and archive the evidence JSON with the fixed attribution era noted.~~ routed — TODO_LIST census re-run row (M54)
~~44. Verify module-map/docs claims that cited fleet consumer counts against post-fix numbers.~~ routed — TODO_LIST census re-run row
~~45. Confirm no go-cqrs-lite module EVER required cmdguard historically (`git log -S cmdguard -- '**/go.mod'`) — closes the morning question properly.~~ routed — TODO_LIST:144
~~46. Sample 5 more apps' go.sum files for cmdguard absence (morning item, still open).~~ done — declined-with-reason (harvest ledger)
~~47. Full `go mod graph | grep cmdguard` on one rich app (InboxClean) as final proof.~~ done — declined-with-reason (harvest ledger)
~~48. Decide the staging repos' fate (21–24) with one sweep command + trash.~~ routed — TODO_LIST:147
~~49. Document in go-cqrs-lite that repo CLIs standardize on cmdguard v5 (one line; verify against cmdguard skill first).~~ routed — TODO_LIST:144
~~50. Close the loop: after 1–3 land, re-run the full A/B suite and pin the numbers in the postmortem doc (20).~~ routed — pdg member-fold follow-ups (rides postmortem f20)

## g) Questions I can NOT figure out myself

**Q1 — The foreign flake.nix pin:** commit `05e5b1ae` changed the cmdguard input from tag
`v5.0.0` to SHA `56fd69b4` — was that you or a parallel session's deliberate pin, and is
`v5.0.0` still the canonical ref project-dependency-graph should release against? (I preserved
it; releasing on a wrong pin would ship the wrong cmdguard.)

**Q2 — Release line:** with version authority pending (tags end v0.2.0, CHANGELOG says
0.7.0-dev), which line carries this fix — do you want a v0.2.1 patch cut now, or does it ride
the 0.7.0 line? (Blocks items 16–19; I will not invent a version scheme.)

**Q3 — Graph-layer scope:** do you want the built-GRAPH paths (`serve`, `diff`, `generate`, and
`why` beyond the empirical check) made member-aware too — a deeper change to `graph/build.go` —
or is accurate `who-uses` (the query surface you actually use) sufficient for now?

---

*Report per status-report skill (`.md` default). Not hand-committed (harness rule); auto-commit
daemon absorbs it. Section (f) is HARVEST input — both this and the 09:16 report remain
unharvested (own miss, listed in e8).*
