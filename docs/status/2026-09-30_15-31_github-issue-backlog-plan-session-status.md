# Session Status: GitHub Issue Backlog Review + Pareto Plan

**Date:** 2026-09-30 15:31
**Scope:** THIS SESSION ONLY — plan-driven, zero production code touched. The session
reviewed all 10 open GitHub issues, verified each against master HEAD (`004298c1a`),
and produced an execution plan. No fixes shipped, no issues closed, no tags cut.

**Companion artifacts (all produced this session):**

- `docs/planning/2026-09-30_14-28_SUPERB-github-issue-backlog-pareto-plan.html` (95 KB, self-contained, inline SVG)
- `docs/planning/2026-09-30_14-28_github-issue-plan.d2` + `.svg` (execution graph)
- `TODO_LIST.md` → new section "GitHub issue backlog (2026-09-30 plan)" + section-index entry

---

## a) FULLY DONE

| # | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                | Receipt                                                                                        |
| - | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| 1 | All **10 open issues** fetched and read in full (bodies + comments; all 10 have **zero comments** — no hidden contributor discussion to miss)                                                                                                                                                                                                                                                                                                                       | `gh issue list` → #43 #42 #36 #35 #32 #28 #27 #26 #25 #21                                      |
| 2 | **Per-issue verification against master HEAD** — this caught two stale claims before they polluted the plan: **#25 already resolved** (tag `metaengine/projectionadapter/v4.5.0` exists on remote, `8c87c48a6`) and **#21 half-fixed** (typed `Causation` written/parsed on master via `writeCausation`/`parseCausation` in `watermill/protocol.go`, but absent from released v4.6.2; scalar `CorrelationID`/`CausationID` still never written by `eventToMessage`) | `git ls-remote`, `git show v4.6.2:watermill/protocol.go` (0 hits), `awk` over `eventToMessage` |
| 3 | Verified remaining scope for #26 (no `retract` in `stack/postgres/go.mod`; latest v4.4.1), #42 (guard only at `cmd/cqrs-lint/run.go:326`, absent in `toolspec.go`), #43 (`d003_d005.go:279` still `docVersion := versions[0]`), #32 (no `LoadByEventID` in `system/adapter_event.go`), #35 (only `ActorEnricher` + `CommandCausalityEnricher` exist), #36 (4 root stack files import metaengine), #27 (no manifest; only `scripts/verify-versions.sh`)              | greps/`sed` outputs, all in-session                                                            |
| 4 | **TODO_LIST.md cross-check**: none of the 10 issues were tracked anywhere in TODO_LIST.md / docs/agents/ / skill references before this session                                                                                                                                                                                                                                                                                                                     | grep for `#21`–`#43` → 0 hits                                                                  |
| 5 | **Pareto plan produced**: 4 waves, 27 comprehensive tasks (30–100 min), 88 fine tasks (≤15 min), ~21h estimate, D2 execution graph rendered to SVG and inlined                                                                                                                                                                                                                                                                                                      | HTML report + `.d2`/`.svg` files exist, HTML tags balance-validated                            |
| 6 | **TODO_LIST.md updated** with the issue backlog as the living source (plan = snapshot), per repo convention and the pareto-planning skill's "add new tasks" rule                                                                                                                                                                                                                                                                                                    | new section + `[GitHub issue backlog]` index entry                                             |
| 7 | Skill discipline: `pareto-planning` loaded and followed end-to-end (Pareto tiers, both table views, HTML+D2 artifact, date-from-CLI, TODO_LIST sync); `status-report`/`brutal-self-review` loaded for this report                                                                                                                                                                                                                                                   | this report                                                                                    |
| 8 | Honest scoping: **no code was changed** — the user asked for a plan, and the session resisted drifting into fixes (with one defensible exception, see b-1)                                                                                                                                                                                                                                                                                                          | `git log` shows only docs                                                                      |

## b) PARTIALLY DONE

1. **#25 could have been closed in-session.** Verification was complete; the remaining work (comment with receipt + close) is ~5 minutes and fully unblocked. I deferred it to plan task T01 instead of just doing it. Defensible under "user asked for a plan", but it was free value left on the table.
2. **The plan's evidence base is only ~70% source-grounded.** Verified against master: #21, #25, #26, #42, #43 (line-level), #32 (absence), #35, #36, #27. Still sourced from ISSUE TEXT ONLY (never checked on master): (a) `SQLEventStore.LoadByEventID` at `storage/eventstore/event_store_by_id.go:23` — the #32 plan assumes this capability exists today; (b) `cmd/cqrs-upgrade`'s actual capabilities vs #28's ask (existence taken from TODO_LIST's 2026-09-07 mention); (c) whether stack/postgres v4.2.0 is STILL broken today (issue reproduced it 2026-09-10; retraction is correct policy either way, but the plan asserts brokenness as fact); (d) the actual parser internals of `d003_d005.go:230-290` — the "positional attachment window" design for #43 is plausible but was designed without reading the collection code.
3. **Receipt discipline in the new TODO_LIST section is partial.** One claim carries a dated receipt (`8c87c48a6, verified 2026-09-30`); the "typed Causation fixed on master but UNRELEASED" claim lacks its file:line/`git show v4.6.2` receipt — violating the section's own Legend convention (2026-09-28 §6.9).
4. **#21 gap analysis is incomplete on one point:** I never confirmed whether master's `buildMetadata` already PREFERS typed `Causation` over the scalar when both exist. Plan task F010 assumes "make typed win" — it might already. One `awk`/`sed` away from certainty; not done.
5. **Estimate consistency between the two tables** (27-task vs 88-task): not cross-checked. Known drift found while writing this report: T07 = 90m but its fine tasks (F017–F020) sum to 60m — a real release wave (verify + tag + post-push proxy checks) realistically exceeds both.

## c) NOT STARTED

1. **All 27 plan tasks / all 88 fine tasks** — zero fixes coded, zero issues closed, zero tags, zero golden regens, zero release mechanics. The repo's issue backlog is untouched since the plan was written.
2. **PR-overlap check** — `gh pr list` was never run. Three issues (#21, #42, #43) end with "happy to send a PR"; if the reporter (or anyone) already opened PRs, the plan double-tracks work. Real omission.
3. **Repo gates over the new artifacts** — `doc-check`, `check-md-go`, README link gates were never run against the new plan HTML / TODO_LIST edits. Almost certainly green (no `go` fences, planning/ is outside doc-check's default scan), but unproven.
4. **TODO_LIST header blockquote update** — the header still names the 2026-09-28 publish-integrity plan as the "Current plan". The 2026-09-30 plan is referenced only inside the new section. Two competing "current" pointers = a small doc split brain I created and did not fix.
5. **HARVEST** — this report's section (f) is input for `docs-health` → HARVEST into TODO_LIST/ROADMAP. Explicitly deferred per the user's "THEN WAIT FOR INSTRUCTIONS".

## d) TOTALLY FUCKED UP

Nothing code-level is broken — **no production code was touched this session**, so the blast radius is docs + planning artifacts only. Within that scope, three things are genuinely fucked up:

1. **The auto-commit daemon committed a garbage intermediate of the plan HTML.** I assembled the report in two disk writes (seed 827-line CSS shell → append body). The daemon committed the CSS-only shell between the writes — history now contains a broken half-file commit of the plan. Under this repo's daemon regime, multi-write file assembly is a hazard; it should have been ONE `write` call with the full content (SVG inlined in-memory).
2. **File-permission inconsistency:** `2026-09-30_14-28_github-issue-plan.svg` landed `-rw-------` (600) while sibling planning artifacts are 644. Any non-owner consumer (docserver, CI artifact step, another user) gets EACCES. Trivial but sloppy — should be `chmod 644`.
3. **The plan contains an internal contradiction I shipped without noticing:** the D2 graph serializes W1 → W2 with a text label admitting "can overlap", and the estimate drift (T07: 90m vs 60m fine-sum) contradicts itself one section later. A plan whose own numbers disagree undercuts the trust it asks for.

## e) WHAT WE SHOULD IMPROVE

1. **Verify external claims against HEAD before planning — mechanically, not ad hoc.** 2 of 10 issues were stale (#25 resolved, #21 half-fixed). The `verify-external-claims` skill exists for exactly this and was NOT loaded this session. Every issue claim that plan tasks depend on (existence of storage API, tool capabilities, "still broken" assertions) should get a source-grounding pass before it becomes a task.
2. **Check PR overlap before planning fixes** (`gh pr list --search linked:issue`). Ten minutes would have caught duplicate work; it was not spent.
3. **Single-write file assembly** under the auto-commit daemon — assemble full content in memory, write once. Prevents daemon-absorbed intermediate commits.
4. **Cross-check estimates between granularity levels** (comprehensive table vs fine table) before publishing a plan. Arithmetic drift between sections is a credibility bug.
5. **Visually verify HTML artifacts** — I validated tag balance, not rendering (does the d2 SVG actually display at reasonable width? do the tables collapse?). Open the file once before declaring done.
6. **Run the repo's own doc gates over self-produced docs** (`doc-check`, `check-md-go`). The repo's culture is "gates over claims"; my own artifacts skipped that.
7. **Apply the repo's receipt convention to my own TODO_LIST additions** — dated receipts on every verify-class claim, not just the convenient one.
8. **Do free closes immediately when verification completes** — "user asked for a plan" doesn't require leaving a 5-minute unblocked issue close (#25) on the table.
9. **Load `brutal-self-review` DURING planning sessions, not after** — the questions in this report (stale claims, PR overlap, split brains) would have sharpened the plan itself had they run before publication.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

Priority-sorted; items 1–27 are the shipped plan (detail in the HTML report), 28–50 are session-artifact and process debt.

| #  | Task                                                                                                                                                    | Size |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ---- |
| 1  | T01: verify `OccurredAt` in projectionadapter v4.5.0 tag, comment receipt, close #25                                                                    | XS   |
| 2  | T02: add `retract [v4.2.0, v4.2.0]` to stack/postgres/go.mod (#26)                                                                                      | S    |
| 3  | T03: `eventToMessage` writes scalar `correlation_id`/`causation_id` (#21)                                                                               | M    |
| 4  | T04: buildMetadata typed-vs-scalar precedence + round-trip table test (#21)                                                                             | M    |
| 5  | T05: backward-compat tests — pre-fix messages still decode (#21)                                                                                        | M    |
| 6  | T06: CHANGELOG [Unreleased] entries (watermill + stack/postgres)                                                                                        | S    |
| 7  | T07: release wave — `#verify` green → tag watermill v4.7.0 + stack/postgres v4.4.2                                                                      | L    |
| 8  | T08: close #21 + #26 with receipts                                                                                                                      | S    |
| 9  | T09: port none-import guard into `toolspec.detect` (#42)                                                                                                | S    |
| 10 | T10: fixture test + negative control + close #42                                                                                                        | S    |
| 11 | T11: D005 positional-attachment rule (#43)                                                                                                              | S    |
| 12 | T12: D005 FP-sentence tests, stale detection stays green                                                                                                | S    |
| 13 | T13: cqrs-lint verify + close #43                                                                                                                       | S    |
| 14 | T14: `EventByIDBackend` capability + `EventAdapter.LoadByEventID` (#32)                                                                                 | M    |
| 15 | T15: NotImplemented fallback + both-path adapter tests (#32)                                                                                            | M    |
| 16 | T16: system verify + api golden + docs + close #32                                                                                                      | S    |
| 17 | T17: typed context keys + `RequestContextEnricher` (#35)                                                                                                | M    |
| 18 | T18: enricher tests (full/partial/empty context)                                                                                                        | M    |
| 19 | T19: core.md recipe + CHANGELOG + golden + close #35                                                                                                    | S    |
| 20 | T20: versions.json manifest generator + committed manifest (#27)                                                                                        | M    |
| 21 | T21: tag-push CI wiring + README compatibility matrix (#27)                                                                                             | M    |
| 22 | T22: consumer jq examples + close #27                                                                                                                   | S    |
| 23 | T23: docs/consumer-upgrades.md — sweep pattern + phantom-require failure mode (#28)                                                                     | M    |
| 24 | T24: bless cmd/cqrs-upgrade in docs + close #28                                                                                                         | S    |
| 25 | T25: create stack/metaengine module + deprecated root forwarders (#36)                                                                                  | L    |
| 26 | T26: new-module gate sweep (go.work, api-stability, layers, cqrs-lint catalog) (#36)                                                                    | L    |
| 27 | T27: hermetic metaengine-free probe + docs + close #36                                                                                                  | M    |
| 28 | `chmod 644` the plan SVG (permission inconsistency)                                                                                                     | XS   |
| 29 | Fix plan estimate drift: T07 90m vs fine-sum 60m; re-sum all waves                                                                                      | XS   |
| 30 | Update TODO_LIST header blockquote "Current plan" pointer to include the 2026-09-30 plan (kill the two-pointer split brain)                             | XS   |
| 31 | `gh pr list` — check for existing PRs linked to #21/#42/#43 before executing W1/W2                                                                      | XS   |
| 32 | Verify `SQLEventStore.LoadByEventID` still exists at the claimed path (plan's #32 evidence)                                                             | XS   |
| 33 | Verify cmd/cqrs-upgrade's actual capabilities vs #28's ask before writing the docs                                                                      | S    |
| 34 | Re-run the isolated v4.2.0 breakage probe to confirm "still broken" before retracting                                                                   | S    |
| 35 | Read `d003_d005.go:230-290` and ground the D005 rule design in the real parser                                                                          | S    |
| 36 | Confirm whether master's buildMetadata already prefers typed Causation over scalar (closes plan uncertainty)                                            | XS   |
| 37 | Visual render check of the plan HTML (SVG displays, tables collapse on mobile)                                                                          | XS   |
| 38 | Run `doc-check` + `check-md-go` + README gates over the new artifacts                                                                                   | S    |
| 39 | Add file:line receipts to every claim in the new TODO_LIST backlog section (§6.9 convention)                                                            | XS   |
| 40 | Reconcile D2 graph topology with the stated W1∥W2 overlap (or serialize the text)                                                                       | XS   |
| 41 | Execution note: run `load-sweep` before `#verify` in T07 if timing paths are touched                                                                    | XS   |
| 42 | Decide whether #27's manifest updates ride the existing nightly/CI infra vs a new workflow                                                              | S    |
| 43 | Make the #36 hermetic probe a permanent check script (not a one-off), if owner ratifies the split                                                       | S    |
| 44 | After watermill v4.7.0: update `.agents/skills/watermill` + modules.md watermill row (wire protocol no longer drops causation)                          | S    |
| 45 | Record session lesson: "2/10 issues stale → verify claims against HEAD before planning" (candidate for crush-config `references/lessons.md`, by commit) | XS   |
| 46 | AGENTS.md note: GitHub-issue backlog now mirrors into TODO_LIST §GitHub issue backlog (workflow memory)                                                 | XS   |
| 47 | Decide contributor-PR policy for the three "happy to send a PR" issues (#21 #42 #43): in-repo vs guided external PR                                     | XS   |
| 48 | Run `docs-health` HARVEST on this report's (f) items → TODO_LIST/ROADMAP routing                                                                        | S    |
| 49 | After W1 ships: sweep cqrs-htmx/go-localsync-adjacent references in docs that document the watermill causation limitation                               | S    |
| 50 | Owner-gated: #27 ask-3 stale-module CI annotation (already recorded [BLOCKED] in TODO_LIST)                                                             | S    |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Release-train policy for W1:** should the #26 retraction + #21 watermill fix ship as their own immediate two-tag wave (my plan), or batch into the already-queued release-train tail sitting in `[Unreleased]` (queue/mysql pair, metaengine wave, cqrs-lint typed-info tier)? One tag costs a full `#verify` run either way — your call on wave economics vs consumer-unblock urgency.
2. **#36 design ruling:** the issue offers two shapes — module split (my pick: `stack/metaengine`, deprecated root forwarders until v5) vs keeping the API and hiding `Plan`/`Store` behind stack-defined interfaces (zero surface change, but keeps the transitive require). Confirm the split + deprecation window, or rule for the interface alternative?
3. **External-PR policy:** #21, #42, #43 all close with "happy to send a PR". Do we implement in-repo (my plan's assumption) or reply "yes, please" and shepherd external PRs (review latency, but builds contributor relationship)? This decides whether T03–T13 are our commits or review cycles.

---

_Format note: user explicitly requested `.md` for this report — the status-report skill's HTML default was overridden per its own escape clause. Git: nothing committed manually (harness forbids commits without explicit request); the auto-commit daemon absorbed all artifacts, including an intermediate CSS-only shell of the plan HTML (see d-1)._
