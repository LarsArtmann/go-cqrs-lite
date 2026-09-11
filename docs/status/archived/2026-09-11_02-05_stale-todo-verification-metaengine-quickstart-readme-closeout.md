# Status: stale-TODO verification — metaengine-quickstart README closeout

> **RESOLVED (docs-health pass 2026-09-11):** **Superseded — archived by the docs-health pass 2026-09-11.** The §b1 stale-TODO-sweep class it proved was EXECUTED repo-wide by this pass (42 completed `[x]` TODO_LIST rows swept per the file's own header policy + the docs-health skill). §b3 skill-vs-repo TODO convention: resolved in favor of the skill + file header (delete done items) — this pass is the precedent. §f items routed: quickstart smoke test → TODO_LIST.
> Open work lives in [`TODO_LIST.md`](../../TODO_LIST.md); shipped surface in [CHANGELOG.md](../../CHANGELOG.md) `[Unreleased]`.


**Session window:** 2026-09-11 ~01:57–02:05 CEST (single task)
**Task:** `TODO_LIST.md` item — "`example/metaengine-quickstart/README.md` does not
exist — author it … Consider a `TestEveryExampleHasREADME` meta-test"
(source 07-42 §b2/§f28, Effort M)
**Outcome:** Task was **already shipped 2026-09-09** (auto-commit `6bb82f5b`) but
never closed off. This session verified every claim end-to-end and closed the
paperwork — README item **and** the adjacent v5-audit item, plus an inline
annotation of today's 01:38 report that had re-listed the stale TODO.

---

## Verification evidence (all gathered this session, not trusted from docs)

| # | Claim | Independent check | Result |
| --- | --- | --- | --- |
| V1 | README exists, covers the four demo sections | Cross-checked all 4 sections against code: `AutoCRUDByConvention` maps (main.go:87), `UserFollowed`→`metaengine.Edge` fold (graph_demo.go:43), `DocEmbedded`→`metaengine.Embedding` (vector_demo.go:33), koanf/`cqrs.yaml` boot (configfile_demo.go:25) | ✅ 4/4 accurate |
| V2 | Example works | `go run -tags "goexperiment.jsonv2" .` in the example dir | ✅ all 4 sections green (map CRUD + delete not-found; graph 1–2 hop reachability; k-NN distances; config-file boot) |
| V3 | Meta-test exists and passes | `TestEveryExampleHasREADME` (cmd/api-stability/examples_readme_test.go:13), run twice | ✅ PASS both runs |
| V4 | Examples v5-clean (adjacent TODO item, claimed in CHANGELOG) | Re-ran `cqrs-upgrade -dry-run -strict -no-build` on **both** `example/metaengine-quickstart` and `example/taskmanager` | ✅ "no v5-removed API usage detected", exit 0, all pins up-to-date |
| V5 | CHANGELOG already documents both artifacts | CHANGELOG.md:221-224 (from the same `6bb82f5b` docs-truth batch) | ✅ consistent |

**Files changed by this session (2):**

- `TODO_LIST.md` — item at :498 (README) closed `- [x] DONE` with re-verification
  stamp; item at :504 (v5 audit) closed after independent re-run (bonus closure,
  not in the assigned task).
- `docs/status/2026-09-11_01-38_todo-batch-verification-and-gates_status.md` —
  row 18 annotated inline (`~~…~~ VERIFIED …`) in the same style as the
  `bf4715bad` annotations from 01:51 today, so future HARVEST passes will not
  re-add the stale item.

---

## a) FULLY DONE

1. **Stale-TODO detection before any writing.** Grep ran before any authoring;
   discovered README + meta-test already exist. No duplicate work, no doc
   overwrite. This is the session's core win: the failure mode it prevents
   (re-authoring a shipped README) is the exact "status reports are
   point-in-time" lesson from memory, applied correctly.
2. **Full claim-by-claim verification of the README** against the four demo
   files (V1) — the README is not just present, it is *accurate*.
3. **End-to-end example run** (V2) — copy-paste surface proven runnable, not
   just compiling.
4. **Meta-test run** (V3) — the class tripwire works today.
5. **Independent v5-policy re-audit of both examples** (V4) — the CHANGELOG's
   0-findings claim for 2026-09-09 re-confirmed on 2026-09-11 with a fresh run.
6. **TODO_LIST closure ×2** with dated evidence in the repo's established
   `- [x] … DONE <date> (evidence)` convention (15 such rows precede these).
7. **Inline annotation of the 01:38 report row 18** — non-destructive,
   style-matched to the same-day `bf4715bad` precedent.
8. **Skill-guided process** — docs-health loaded before any TODO/report edits;
   VERIFY mode steps followed (claims checked against code, quality gate run
   on the touched surface).

## b) PARTIALLY DONE

1. **The "stale-TODO sweep" class (07-42 §f34) is proven but not executed.**
   This session proved the class is real — a TODO open for 2 days whose work
   shipped, *and* a same-day (01:38) report that re-harvested it without
   verification. Only this one item was swept; the repo-wide sweep of "deferred
   items whose TODO state may now be resolvable" has not run.
2. **Verification was targeted, not full-gate.** The meta-test was run
   directly; `nix run .#verify-fast` / doc-check were not run. Justified for a
   2-file markdown change (no API surface touched), but the session's evidence
   is surface-scoped by design.
3. **Skill-vs-repo TODO convention divergence noticed, not resolved.** The
   docs-health skill mandates "delete done TODO items"; the repo keeps
   `- [x] DONE` rows with evidence. I followed repo precedent (correct per
   "follow existing patterns") but did not reconcile the skill text — see
   question Q2.
4. **This report's section (f) is written but not yet HARVESTed** into
   TODO_LIST/ROADMAP — per instruction, waiting for user go before routing.

## c) NOT STARTED (noticed this session, deliberately untouched)

1. **Foreign working-tree changes:** `metaengine/adttest/conformance.go` (+11)
   and `metaengine/demote.go` (+11/−2) — substantive shadow-replay work
   ("replayToShadow honors each event's recorded Record context … OnRecord
   folds see the original StreamID/Version instead of a synthesized
   Type-only record"). Not authored this session; left untouched per the
   never-revert-unauthored-changes rule. Uncommitted, un-owned, un-audited.
2. **Six LSP `stdversion` warnings on the example**
   (main.go:118,122,126,139,153,185 — "`json.Unmarshal/Marshal` requires
   go1.27 or later (file is go1.26)"). Almost certainly gopls not seeing the
   `goexperiment.jsonv2` build tag rather than real drift (the tagged build
   passes; the example runs), but nobody has confirmed that or silenced it via
   gopls env config.
3. **Pre-existing LSP backlog** (also in 01:38 report rows 26/27):
   `integration/go.mod:131` genproto tidy warning;
   `cmd/api-stability/pin_drift_test.go:148` unused parameter `root`;
   `vector_demo.go:61` infertypeargs hint.
4. **HARVEST of this report** — waiting for explicit go.
5. **Blocked items seen in passing, untouched:** macOS verification of
   ephemeral PG; `integration-mysql-nspawn` (needs root); dgraph `-shuffle=on`
   evaluation (TODO_LIST neighbors of the closed items).

## d) TOTALLY FUCKED UP

Nothing in this session's own work broke. Honest self-critique instead:

1. **The first tool invocation of the v5 audit was wrong** — ran
   `cqrs-upgrade` via its module path from the example dirs with GOWORK=off
   (package not resolvable), wasting one round trip before re-running from the
   tool's own module dir. Small, but the correct invocation should have been
   obvious from the `-workspace` flag help.
2. **Adjacent-item closure was luck-adjacent.** The v5-audit TODO closure only
   happened because the CHANGELOG passage for the README happened to mention
   both. A strictly-scoped reading of the task would have left a
   verified-done item open. A closing pass over the source report's items
   (07-42 §f8) — rather than stumbling on the CHANGELOG sentence — would have
   been the systematic way to catch it.
3. **Skill conflict handled silently.** docs-health says delete done TODOs;
   I kept them per repo precedent and did not surface the conflict until this
   report (see Q2). Flagging at decision time would have been better.
4. **Systemic (not this session's) fuckup, re-confirmed:** the 01:38 report —
   written *today* — re-listed the README item as open without checking the
   code, 33 hours after the work shipped. Two consecutive sessions' doc
   processes (07-42 authored the stale TODO; 01:38 propagated it) missed that
   a grep would have closed. That is the Harvest-verification gap, now proven
   twice.

## e) WHAT WE SHOULD IMPROVE

1. **HARVEST step 3 must actually run** ("verify against code — grep before
   adding"). Both the stale TODO (07-42) and its re-harvest (01:38) skipped
   it. Concrete rule of thumb: any item phrased as "X does not exist" must be
   answered with `ls`/`glob` before it enters TODO_LIST.
2. **Cheap mechanical stale-TODO tripwire.** A script (or doc-check mode)
   that greps CHANGELOG `[Unreleased]` claims and matches them against open
   TODO_LIST items by module/symbol overlap, flagging "claimed done but still
   open" pairs for verification. This session's pair would have been caught
   by the crudest possible version of this.
3. **Reconcile the TODO-closure convention** between docs-health skill text
   and repo practice (see Q2) — otherwise every future closure is a judgment
   call.
4. **gopls env for the jsonv2 tag.** Configure the LSP (GOFLAGS/
   buildTags in gopls settings, flake devShell) so the 6 stdversion warnings
   on the example disappear or are confirmed as real drift. Either outcome is
   better than daily noise.
5. **Tool-invocation notes for `cqrs-upgrade`** — the working invocation
   (`cd cmd/cqrs-upgrade && GOWORK=off go run … . -dry-run -strict -no-build
   <target-dir>`) belongs in `docs/agents/gowork-modes.md` or the testing
   gotchas so the next session doesn't re-derive it.
6. **Scope-plus-one habit.** When closing an item whose completion evidence
   is a shared doc passage (CHANGELOG/commit), check that passage's *other*
   claims for sibling TODOs — this session's bonus closure is the template.

## f) Up to 50 things to get done next (ranked by impact; provenance marked)

_A brainstorm, not a commitment list — most items below the top few are
ROADMAP/TODO_LIST fuel pending user go. "[38]" = row in today's 01:38 report
§f (already read this session); "[TL]" = TODO_LIST; "[S1]" = observed this
session._

**Directly enabled / discovered this session**

1. [S1] **Repo-wide stale-TODO sweep** (07-42 §f34, elevated): verify every
   open TODO whose work may have shipped since its authoring — this session
   found 2 items in one batch; the class is proven.
2. [S1] **Build the stale-TODO tripwire** (see e2) — CHANGELOG-claims vs
   open-TODOs cross-check, wire into `#verify` or doc-check.
3. [S1] **Decide ownership of the foreign `metaengine` working-tree changes**
   (conformance.go / demote.go shadow-replay Record context) — commit, audit,
   or discard is not my call (see Q1).
4. [S1] **gopls jsonv2 build-tag env** — kill or confirm the 6 stdversion
   warnings on `example/metaengine-quickstart`.
5. [S1] **Document the working `cqrs-upgrade` invocation** in
   docs/agents gotchas.
6. [S1] **HARVEST this report** into TODO_LIST/ROADMAP on user go.
7. [S1] **Reconcile docs-health TODO-closure rule with repo practice** (Q2).
8. [S1] **Push/commit policy for agent-written reports** (Q3; related to the
   open tag-push question in the 01:47 report).

**Carried from the 01:38 report (read this session, unverified since)**

9. [38·r39] Run `pin-sweep.sh --check` (post-release census never confirmed).
10. [38·r40] One integration-tag lint run via the nix binary (retire
    version-drift doubt).
11. [38·r13] "Days-since-green" metric + nightly all-green sentinel.
12. [38·r14] Fix `check-coverage.sh` nix wrapper running without cache env
    (vacuous 0.0% drift).
13. [38·r20] sqliteengine `EngineResetter` (ADR-0136 follow-up; memory engine
    is the only one today).
14. [38·r21] Fold-write failover for health-quarantined engines (ADR-0137
    known gap: reads reroute, writes fail loudly).
15. [38·r22] **User decision:** dgraph one-RPCheduler flip scope.
16. [38·r23] **User decision:** MariaDB :33061 container retention.
17. [38·r17] >350-line production files (~54) split program (needs gate-policy
    decision first).
18. [38·r36] Verify the rename tripwire also fires under `-race`.
19. [38·r38] CHANGELOG↔tripwire consistency meta-test (17 renamed codes).
20. [38·r24] Table-driven rename-tripwire harness for future renames.
21. [38·r30] Decide `_test.go` exclusions for gosec/wsl_v5: policy or debt
    (scheduling/sqlstore pilot).
22. [38·r15] CV consumer bump, operator-gated (8 modules behind latest).
23. [38·r16] actionlint CI step + shellcheck for `scripts/`.
24. [38·r19] templ tripwire script (parse `_templ.go` FileName metadata).
25. [38·r25] Document/adopt `$PIPESTATUS` rule in memory + gowork-modes.md.
26. [38·r26] `integration/go.mod` genproto tidy warning cleanup.
27. [38·r27] `pin_drift_test.go:148` unused parameter cleanup.
28. [38·r28] AGENTS.md note: ad-hoc `go mod download` go.sum hashes — commit
    on sight.
29. [38·r29] Daemon heuristic: skip message-flattening when diff contains new
    test files.
30. [38·r31] golangci-lint version pin note (PATH vs nix binary).
31. [38·r32] Dirty-tree guard messaging: mention `//art-dupl:accept` workflow.
32. [38·r33] CHANGELOG entry for the go.sum repair class if releases are cut
    from this state.
33. [38·r35] `claiming_mysql.go` IN-list builder comment cross-ref to nolint
    rationale.
34. [38·r3] `testdata/` mutation fixture + scanner self-assert (aggregate
    tripwire).
35. [38·r4] `lint-module` optional build-tag argument.
36. [38·r5] CI leg: integration-tag lint for modules shipping
    `*_integration_test.go`.
37. [38·r6] `#verify-ci`: per-module `go mod download` + no-diff assertion.
38. [38·r7/r8/r9] Docs follow-ups from 01:38 (route §f; annotate orphaned
    `cec9248da` report; close stale GOWORK-table TODO).
39. [38·r34] Sweep other status reports for "deferred, owners landed since"
    items (same class as item 1 — do after the sweep methodology exists).
40. [38·r11/r12] v5-sweep census entries (Pebble slog keys; consumer grep for
    old `aggregate_*` strings outside the repo).

**Seen in passing (TODO_LIST neighbors, unstarted)**

41. [TL] macOS verification of ephemeral PG (BLOCKED: needs CI runner leg).
42. [TL] `integration-mysql-nspawn` full run (BLOCKED: needs root).
43. [TL] Evaluate `-shuffle=on` for the dgraph suite specifically.

_(43 items; the remaining slots deliberately left empty rather than padded.)_

## g) Questions I can NOT figure out myself

- **Q1 — Foreign `metaengine` changes:** `metaengine/adttest/conformance.go`
  and `metaengine/demote.go` carry substantive uncommitted work (shadow-replay
  honoring recorded Record context) that nobody this session authored. Keep
  and hand to its owner as WIP, or is it abandoned output I should flag for
  review-and-discard? I cannot know whose it is or whether it is mid-edit.
- **Q2 — TODO closure convention:** repo practice keeps `- [x] DONE` rows in
  TODO_LIST (15 and growing); the docs-health skill mandates deleting done
  items. Which is canonical going forward? It changes how every future
  closure looks, and one of the two should be updated to match.
- **Q3 — Agent commit policy for reports:** my rules forbid commits without an
  explicit "commit", but the status-report skill's flow includes committing
  the report. I relied on the auto-commit daemon this time. Should
  agent-written status reports be (a) daemon-committed (current behavior),
  (b) explicitly committed by the agent, or (c) left uncommitted for manual
  review?

---

*Point-in-time snapshot. Section (f) is HARVEST input for docs-health →
TODO_LIST/ROADMAP once the user says go. HTML default overridden to `.md` per
explicit user instruction for this report.*
