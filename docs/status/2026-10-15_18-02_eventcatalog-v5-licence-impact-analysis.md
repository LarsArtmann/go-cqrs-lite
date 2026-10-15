# Status Report: EventCatalog v5 Licence-Impact Analysis Session

- **Date:** 2026-10-15 18:02 CEST
- **Session scope:** Impact analysis of the EventCatalog v5 announcement (2026-09-24) on the `catalog/` module; durable doc recording; upgrade-gain analysis. No code changes.
- **Trigger:** User question: "Does this change anything for us: eventcatalog.dev/blog/eventcatalog-v5 (i.e. the catalog/ sub module)?" plus follow-up "If we upgrade, what do we gain?"
- **Scope discipline:** Per operator instruction, this report covers ONLY this session's run and what it noticed in passing. Parallel-session work is inventoried only where it was directly observed (compiler diagnostics, git status).

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| A1 | **Upstream v5 impact analysis completed and answered.** Fetched the v5 announcement, determined: v5 = BSL 1.1 (free only for orgs <$10M revenue AND <$10M raised, self-certified); v4 stays MIT but EOL ~2027-03-24 (6 months of security-only fixes post-v5 release); our `^4.6.3` caret never jumps majors, so **nothing breaks today** and there is no accidental BSL exposure. | Blog (primary source) + inspection of `catalog/AGENTS.md`, `catalog/eventcatalog/exporter.go:19`, `writer.go:134,222`, `exporter_customdoc.go:20`, grep sweep of `@eventcatalog` refs, `TODO_LIST.md` data-mesh section |
| A2 | **Durable record written: `catalog/AGENTS.md` (+12 lines).** New bullet "Upstream v5 licence shift (announced 2026-09-24)" placed directly under the pin-policy bullet, covering: licence terms, v4 EOL date, permanence of the 4.6.3 workarounds, bump = licence decision + format re-verification, and the v5 enterprise-feature fold-in. | `git diff HEAD -- catalog/AGENTS.md` (+12, verified in diff review; wording error "above"→"below" caught and fixed on sight) |
| A3 | **Decision row filed in `TODO_LIST.md` (+22 lines).** New section "EventCatalog upstream v5 licence shift (2026-10-15)" with an open checkbox "Decide the `@eventcatalog/core` pin strategy before v4 EOL" spelling out both options (stay frozen on 4.6.3 vs bump to v5), the licence dimension, and the March 2027 deadline; section-index entry added and anchor slug verified against the heading. | `git diff HEAD -- TODO_LIST.md` (+22; heading `#eventcatalog-upstream-v5-licence-shift-2026-10-15` matches index link) |
| A4 | **Upgrade-gain analysis delivered**, mapped to concrete emitter surfaces rather than marketing claims: custom-docs routing unlock (we already emit `docs/<slug>/index.mdx`), agent-changelog unlock (guarded off at `writer.go:222` due to the 4.6.3 crash), 3-level diagrams, team swimlanes riding our shipped `Flow.Owners`, v5 MCP server vs our hand-rolled `llms.txt`/`schemas.txt` writers, federation vs the eventcatalog-hub machinery, EOL risk avoidance. Unaffected surfaces (Go `docserver`, `StaticServer`) called out explicitly. | Session answer #2; each claim cites a repo file:line |
| A5 | **Edits verified at the proportionate level.** Diff-reviewed both files: 34 insertions total, zero go-code fences added (md-go gate exposure = none), exact-match anchors. No Go files touched, so no build/test surface changed. | `git diff HEAD` + fence count check (0) |

**Verdict:** the session's actual deliverable — "does this change anything, and what would we gain" — is complete, evidence-backed, and durably recorded. It was an analysis session; A1–A5 is the whole deliverable, and it is done.

## b) PARTIALLY DONE

| # | Item | What works | What remains | Blocker | Effort |
|---|------|-----------|--------------|---------|--------|
| B1 | **Pin-strategy decision (stay 4.6.3 vs bump v5)** | Fully framed: both options costed, recorded as an actionable TODO_LIST row with deadline (before 2027-03-24) | Undecided — this is a decision, not implementation; the licence leg is owner-only knowledge | Owner's Free-Use qualification answer (see Q1) | S (decision) / M (bump if chosen) |
| B2 | **v5-gain claims that hinge on upstream behavior** | Clearly flagged as *needs bump-time verification* rather than asserted: does v5 fix the agent-changelog crash? do the 3 sidebar WARNs persist? does v5 route standalone custom docs? | All three upstream verifications unstarted (deliberately out of session scope: no bump decision existed yet) | None — but doing them before the decision would sharpen it | M total |
| B3 | **Gate coverage of the doc edits** | Proportionate verification done: diff review, fence count, anchor check. The edited files (`catalog/AGENTS.md`, `TODO_LIST.md`) are not in `cmd/doc-check`'s default scan set and contain no go fences, so no gate *should* fire | The full `doc-check` invocation was not run (would also not scan these two files by default); TODO_LIST/catalog-AGENTS have no dedicated link gate I ran | None — running `nix run .#check-md-go` + `doc-check` once as belt-and-braces is cheap | XS |

## c) NOT STARTED

All of these are *follow-on* work the analysis surfaced but that was explicitly out of scope ("don't research unrelated stuff") and/or blocked on the pin decision:

| # | Item | Why not started | Priority |
|---|------|-----------------|----------|
| C1 | Upstream verification of the three flagged v5 behaviors (changelog crash, sidebar WARNs, custom-docs routing) via v5 release notes / fixture render | Awaits decision context; verifying before deciding would improve it, though | High |
| C2 | Free-use qualification assessment | Owner-only data (revenue/funding); cannot be researched from the repo | Critical (unblocks everything) |
| C3 | The bump itself: v5 zod-schema format re-verification, React 19 clean-install drill, golden regen (`package.snap`, `eventcatalog-config.snap`), guard removal (`shouldEnableChangelog`, ec-fixture changelog profile), README pin-reference update (`catalog/README.md:338`) | Blocked on B1 decision | High (after decision) |
| C4 | v5 MCP server vs `writer_llms.go`/`writer_schemas_txt.go` consolidation evaluation | Post-decision feature work | Medium |
| C5 | v5 federation vs eventcatalog-hub (merge gate, governance lint, stale-source probe) consolidation evaluation | Post-decision; also depends on Q3 (hub's strategic role) | Medium |
| C6 | Caret (`^4.6.3`) vs exact-pin policy review | Surfaced during analysis (caret silently drifts downstream installs across 4.x minors); never decided | Low |
| C7 | Upstream-bug tracking home for our two known 4.6.3 bugs (currently prose-only in `catalog/AGENTS.md`; no issue links, nothing tells us if upstream fixes them) | New idea from this session's retrospective | Medium |

## d) TOTALLY FUCKED UP

Radical honesty, session-scoped:

1. **`example/taskmanager` is compile-broken in the working tree — 4 errors.** `query_http_test.go:159` `systemRegisterQuery` undefined; `:199` `io` undefined; `:203` `idStreamID` undefined; `:206` `parseIDString` undefined; plus 3 `go.mod` tidy warnings (go-etag/server, httputil, httputil/server_timing should be direct). **Not this session's breakage** — it is uncommitted WIP from the parallel 03-01 query-http-binding session (which left its own untracked status report at `docs/status/2026-10-15_03-01_query-http-binding-doc-session.md`). Severity: blocks that module's build/tests, and its gopls diagnostics pollute every tool call in any session touching the repo. Workaround: none needed for `catalog/` work (separate module), but the tree is not green. **Not touched by this session, per the never-revert-foreign-changes rule — it needs its owning session (or you) to finish or stash it.**
2. **I introduced a factual-direction error in my own doc edit and caught it only during report prep.** The new `catalog/AGENTS.md` bullet said the 4.6.3 bugs are "worked around **above**" — they are documented **below** the insert. Trivial (two words), fixed on sight during this report (owner's standing on-sight-fix permission), but it shipped in the first answer's edit and would have misled a skimming reader about where the bug inventory lives. Root cause: wrote from memory of the section instead of re-checking line order. Mitigation going forward: the diff review step (which caught it) should precede the answer, not the status report.
3. **No gate run at all on doc-only edits.** Defensible (markdown-only, no fences, files outside the doc-check default set) but strictly speaking the "test after changes" discipline was satisfied by eyeball, not by gate. Listed here rather than hidden in (e) because "defensible skip" and "verified" are different claims.

Nothing else broke in this session: no code was changed, no tests exist to re-run, and the auto-commit daemon has not yet absorbed the two doc edits (they are verified present in the working tree via `git diff HEAD`).

## e) WHAT WE SHOULD IMPROVE

| # | Improvement | Impact | Concrete fix |
|---|-------------|--------|--------------|
| E1 | **Our pinned upstream bugs have no tripwire.** The agent-changelog crash and 3 sidebar WARNs live as prose in `catalog/AGENTS.md`, tied to "core 4.6.3". If upstream fixes them, nothing in the repo notices — the guard (`shouldEnableChangelog`) and its rationale silently become obsolete debt, and the "drop when fixed" instruction requires an archaeology session to rediscover. | Every future bump pays rediscovery cost; stale guards linger | Add an upstream-bugs checklist (bug, symptom, guard location, "verify at bump" items) to `catalog/AGENTS.md` or a small `docs/agents/upstream-bugs.md`, and check it as step 1 of any pin bump |
| E2 | **`^` caret range means silent downstream drift.** Generated `package.json` pins `^4.6.3`, so consumer `npm install` floats across future 4.x minors — reproducibility depends on upstream 4.x staying behavior-stable, which is exactly what an EOL'd line stops guaranteeing loudly. | A future 4.x release could change rendering under us with zero signal | Decide caret-vs-exact at pin-decision time; if staying on v4, exact-pin (`4.6.3`) maximizes reproducibility on a frozen line |
| E3 | **Verification-before-claim ordering.** The wording slip in (d2) happened because I answered, edited, and only re-read the diff during the *next* turn's report. | Small, but it's the same class as the docs-health publish-integrity lessons already on file | Run the diff review immediately after any doc edit, before the closing message of that turn |
| E4 | **`.md` status-report demand keeps recurring.** The skill's canonical format is HTML with a recorded dispatch-report `.md` exception; this interactive session produced a second explicit `.md`-path demand. | Recurring format friction | Owner call: either widen the recorded exception (interactive deep-dive answers that get filed) or keep HTML canonical and I keep flagging the override |
| E5 | **Parallel-session breakage is visible to everyone but owned by no one.** The taskmanager compile errors surface in gopls diagnostics on every tool call repo-wide. | Noise + risk a later session mistakes foreign WIP for rot | The owning session should finish/commit/stash before ending; if a session must end red, its status report should say so loudly (the 03-01 report exists but is untracked) |

## f) NEXT TASKS (up to 50; ranked, HARVEST-ready)

Impact-ranked. E = Effort (XS <10min, S <30min, M 30min–2h, L >2h). C = Category.

| # | Task | Impact | E | C |
|---|------|--------|---|---|
| 1 | Answer the Free-Use qualification question (Q1) — the single fact that decides stay-vs-bump | Critical | XS | Decision |
| 2 | Decide pin strategy before 2027-03-24 (TODO_LIST row filed) | Critical | S | Decision |
| 3 | Fix/finish/stash `example/taskmanager` compile errors (owned by the 03-01 session — coordinate, don't clobber) | Critical | M | Bug |
| 4 | Verify upstream: does v5 fix the agent-changelog crash? (release notes + one ec-fixture render) | High | M | Verification |
| 5 | Verify upstream: does v5 route standalone custom-docs pages? (we already emit them) | High | S | Verification |
| 6 | If bump: format-contract re-verification against v5 zod schemas (pointer-object lists, `x-*` extension properties, composite IDs, channel message pointers, changelog sidecars, flow node kinds) | High | L | Verification |
| 7 | If bump: React 19 clean-install drill in ec-fixture (rm `node_modules` `.astro` lockfile → install → build) | High | M | Verification |
| 8 | If bump: regen goldens (`package.snap`, `eventcatalog-config.snap`) via scoped UPDATE_SNAPS + `git restore` of cross-deleted snaps | High | S | Quality |
| 9 | If bump: re-evaluate `shouldEnableChangelog` guard + retire ec-fixture changelog profile if crash fixed | High | S | Cleanup |
| 10 | If staying: exact-pin `4.6.3` (drop caret) and record frozen-line rationale with revisit date | High | S | Decision |
| 11 | Evaluate v5 MCP server vs `writer_llms.go`/`writer_schemas_txt.go` (replace / complement / keep) | High | M | Feature |
| 12 | Evaluate v5 federation vs eventcatalog-hub merge-gate/lint/probe stack (depends on Q3) | High | M | Feature |
| 13 | Verify upstream: do the 3 sidebar changelog WARNs persist in v5 (render-gate tolerance check) | Medium | S | Verification |
| 14 | Confirm CI licence-key behavior if bumping: what does `check-eventcatalog`'s npm-install+build do under v5's 60-day evaluation (blog says builds never break — verify) | High | S | Verification |
| 15 | Update `catalog/README.md:338` pin prose (`^4.6.3` reference) whenever the pin moves | Medium | XS | Docs |
| 16 | Grep sweep for other hardcoded core-version mentions across `catalog/` (builders, docserver, docstrings) before any bump | Medium | S | Docs |
| 17 | Build the upstream-bugs tripwire checklist (E1) | Medium | S | Docs |
| 18 | Run full `doc-check` + `check-md-go` once over the standard set as belt-and-braces after today's AGENTS/TODO edits | Medium | S | Verification |
| 19 | File upstream issues (after verify-before-filing) or a watch note for the two known 4.6.3 bugs | Medium | M | Quality |
| 20 | Decide MCP/`llms.txt` overlap: compare our `llms.snap` output shape vs upstream v5's llms.txt contract | Medium | M | Feature |
| 21 | Evaluate 3-level diagram drill-down vs `catalog/d2/` exporter + docserver `d2view.templ` (replace/complement) | Medium | M | Feature |
| 22 | Verify team-swimlane flows render from our `Flow.Owners` data without exporter changes | Medium | S | Verification |
| 23 | Trial EventCatalog Studio v2 against the mesh-demo catalog fixture | Low | S | Feature |
| 24 | Decide SSO relevance for first-party doc deployments (likely none — record the "why not") | Low | XS | Decision |
| 25 | Add the 2027-03-24 v4-EOL date to ROADMAP as a dated revisit trigger | Medium | XS | Docs |
| 26 | If bump: update `WithPlainRefIDs` guidance after checking v5 linter ID rules | Low | S | Docs |
| 27 | Confirm auto-commit daemon absorbed the two doc edits at next session start (`git log`) | Low | XS | Hygiene |
| 28 | Track the untracked `docs/status/2026-10-15_03-01_query-http-binding-doc-session.md` (git add via its owning session) | Low | XS | Hygiene |
| 29 | taskmanager `go.mod` tidy warnings ×3 (go-etag/server, httputil, httputil/server_timing → direct) — owning session | Low | XS | Cleanup |
| 30 | After any pin movement: CHANGELOG `[Unreleased]` row (exporter surface is api-stability-tracked; docs-only today = correctly no row, but a bump is user-visible) | Medium | XS | Docs |
| 31 | Decision record: file a dated receipt sub-bullet on the TODO_LIST row once the decision lands (receipt convention) | Medium | XS | Docs |
| 32 | Decide whether `cmd/ec-fixture` needs a v5 profile (fixture covers default+changelog today) | Low | S | Quality |

Items 1–2 unblock the rest; 3 is the only red item in the tree but belongs to another session. **HARVEST note (docs-health):** items 1–16 and 25 belong in `TODO_LIST.md` (the pin row already filed covers 2, 6–10, 15–16 conditionally); 17–24 are ROADMAP-fuel pending the decision; 3, 28, 29 route to their owning session, not the backlog.

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Does the owning org qualify for EventCatalog v5 Free Use — i.e. under $10M annual revenue AND under $10M raised?** I tried: the blog, the licensing FAQ link, and every repo doc — the qualification is self-certified financial data that exists nowhere I can read. This single yes/no decides the whole stay-vs-bump strategy (Q: TODO_LIST row, task 1–2).
2. **Is the Node-rendered EventCatalog site path actually load-bearing in production for the first-party fleet, or is the Go `docserver` the real consumption surface?** I tried: grep of consumer wiring in this repo, `catalog/README.md`, the data-mesh section — I can see the exporter→npm-build→`StaticServer` pipeline exists, but not which first-party consumers deploy it. Answer determines whether v4 EOL is a hard deadline or a slow drip.
3. **Is `eventcatalog-hub` a strategic investment to keep evolving, or scaffold to retire if v5's included federation covers the merge/governance/stale-source needs?** I tried: the TODO_LIST data-mesh section (hub-side work inventory) — but the hub repo's purpose and your appetite for it live outside this repo. Answer re-weights the federation "gain" from nice-to-have to potentially repo-retiring.

---

*Point-in-time snapshot. Section (f) is docs-health HARVEST input — do not let it entomb here. Report per operator demand written as `.md` (skill default is HTML-styled dashboard; operator's explicit path demand wins, override flagged in the closing message).*

---

## Addendum (2026-10-15 19:12): cross-repo reconciliation with eventcatalog-hub

Parallel same-day session in `~/projects/eventcatalog-hub` published a core-5.0.1
adoption audit (`docs/research/2026-10-15_eventcatalog-deep-dive.html`, 18:09)
and a v5 leverage Pareto plan
(`docs/planning/2026-10-15_18-22_pareto-eventcatalog-v5-leverage-plan.md`, 18:22).
They answer this report's section (g) and correct parts of its analysis:

- **Q1 ANSWERED:** LarsArtmann qualifies for EventCatalog v5 Free Use
  (individual, self-certified) — recorded as a hard fact in the hub audit.
  The licence blocker on the pin decision is gone; it is now a timing +
  re-verification + cross-repo-coordination decision. TODO_LIST row updated
  with a dated receipt.
- **Q2 LARGELY ANSWERED:** the hub IS the production consumer — it serves
  catalog.home.lan (Gatus-checked) and builds our v4-format exported trees
  under core `^5.0.1` with link-validation as CI gate. Empirical de-risk:
  v5 parses our current exporter output.
- **Q3 ANSWERED (in motion):** the hub is strategic and has scheduled the
  federation bake-off (T5, inside its eval window ending ~2026-12-10) that
  formally decides keep-merge.py-vs-upstream.
- **Gain-list corrections:** the "v5 MCP server" gain is NOT bump-gated — the
  built-in server needs SSR (verified NO-GO for static serving) and the
  standalone `@eventcatalog/mcp-server` consumes the `llms.txt` we already
  emit. The custom-docs routing gain is CONFIRMED live (`/docs/custom/*` in
  the hub deployment).
- **Split-brains found and filed:** hub's "blocker is exporter support" for
  channels/entities is wrong (both are first-class here — `catalog.Channel`,
  `catalog.Entity`, `catalog.Agent`; only SYSTEMS lack a config type), and its
  AsyncAPI "exporter gap" overlooks `catalog/asyncapi/`. Recorded as
  responded rows in TODO_LIST § "Upstream asks from eventcatalog-hub
  (harvested 2026-10-15)".

No code changed in this addendum; edits: TODO_LIST.md (pin-row receipt + new
upstream-asks section + index entry), catalog/AGENTS.md (licence bullet
updated + new hub-consumer bullet).
