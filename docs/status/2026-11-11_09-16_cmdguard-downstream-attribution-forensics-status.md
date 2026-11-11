# Status: cmdguard Downstream Attribution Forensics (who-uses artifact investigation)

**Date:** 2026-11-11 09:16 CET
**Session scope:** One question from Lars: *"Why does go-cqrs-lite add cmdguard to downstream apps?"*, asked against a pasted `project-dependency-graph who-uses cmdguard` tree (77 consumers, many `[indirect] via go-cqrs-lite/command@v4.14.1`-style rows).
**Repos touched:** read-only investigation in `go-cqrs-lite` + `project-dependency-graph` + downstream app manifests. **No code changed in any repo.** This report is the only file written.

## Scope line — what this report VERIFIED vs did NOT verify

**Verified (command/artifact for every claim):**

| # | Claim | Evidence |
| - | ----- | -------- |
| V1 | Only 5 go-cqrs-lite modules require cmdguard, all `cmd/*` CLI tooling | `rg -l 'larsartmann/cmdguard' --glob 'go.mod'` → cmd/api-stability, cmd/cqrs-bench, cmd/cqrs-gen, cmd/cqrs-lint, cmd/doc-check (all `v5.0.0`) |
| V2 | No LIBRARY module requires cmdguard — not even at the published tags the tree cites | Downloaded proxy `.mod` files for `command/v4@v4.14.1`, `event/v4@v4.15.2`, `catalog/v4@v4.12.0`, `decider/v4@v4.8.0`, `claiming/v4@v4.0.2` — zero cmdguard lines in every one |
| V3 | Downstream apps do not inherit cmdguard | `KeyHolderAI/go.mod` + `KeyHolderAI/go.sum`: no cmdguard at all; `cqrs-htmx` go.mods: none; `go-appkit/go.mod`: none |
| V4 | The 5 cmd modules are go.work members | `grep 'cmd/(api-stability\|cqrs-bench\|cqrs-gen\|cqrs-lint\|doc-check)' go.work` — all present in the `use` block |
| V5 | Tool mechanism: workspace aggregate merge | `project-dependency-graph/discovery/workspace.go:53-112` (`buildWorkspaceModule`) + `:121-190` (`collectWorkspaceModules`) merge ALL internal member go.mod requires into ONE aggregate node per go.work repo |
| V6 | Tool mechanism: member fold + via label | `project-dependency-graph/graph/consumers.go:100-160` (`ConsumersFromModules` BFS), `:356-380` (`buildMemberEdgeIndex` re-indexes consumer-of-member edges under the owning aggregate), `:143-149` (those consumers are stamped `Direct:false` with the member key as `ViaKey`) |
| V7 | Reproduction matches the pasted tree exactly | `cd ~/projects/project-dependency-graph && go run . who-uses cmdguard --direct-only=false` → 32 direct + "45 hidden indirect" = 77; KeyHolderAI renders as `@ v5.0.0 [indirect] via go-cqrs-lite/command@v4.14.1`, identical to the paste |

**NOT verified (and why):**

- The non-go-cqrs-lite `via` rows (`go-appkit@v0.9.0`, `samber-linter@v0.4.0`, `buildflow/constants`, `universal-workflow@v0.0.4`) were explained by the SAME mechanism but not individually audited per-repo (mechanism-inferred, marked b-item below).
- Whether any go-cqrs-lite module required cmdguard at some HISTORICAL tag (`git log -S cmdguard`) — not run; V2 covers only the five tags cited in the tree.
- The 45 indirect consumers beyond the sampled three apps — no fleet-wide go.sum sweep.
- `project-dependency-graph`'s `why`, `serve`, `daemon`, `orphans`, `diff` subcommands — not audited for the same/opposite fold semantics.
- No AGENTS.md/gotchas/TODO_LIST write happened yet (this report is the first durable artifact).

## Executive answer (the finding)

**go-cqrs-lite does NOT add cmdguard to downstream apps.** The consumer tree is a projection artifact of `project-dependency-graph`'s workspace-aggregate model:

1. go-cqrs-lite's `cmd/*` CLI tooling (api-stability, cqrs-lint, doc-check, cqrs-bench, cqrs-gen) is built with cmdguard v5 — the fleet-standard CLI framework. These five go.mods carry the only cmdguard requires in the repo (V1).
2. The tool models a go.work repo as ONE aggregate node and merges every member module's requires into it (V5). The aggregate "go-cqrs-lite" therefore shows a DIRECT cmdguard edge → the `go-cqrs-lite @ v5.0.0 [direct]` row.
3. Reverse BFS folds every importer of ANY go-cqrs-lite sub-module under that aggregate and stamps the sub-module the app imports as the `via` label (V6) → `[indirect] via go-cqrs-lite/command@v4.14.1` means "this app imports go-cqrs-lite/command", NOT "command depends on cmdguard". The fold is intentional design (its own comment cites the `cqrs-htmx/setup` visibility case) — the cmd/* side effect is unintended bleed.
4. Consequence: `go build` / `go get` in KeyHolderAI (or any library consumer) never downloads cmdguard. Verified in go.sum (V3).

Same artifact class plausibly explains `via go-appkit` (go-appkit is a go-cqrs-lite wrapper whose own tooling/aggregate reaches cmdguard), `via samber-linter`, `via buildflow/constants`, `via universal-workflow` — inferred, not audited (see b/c).

## Self-review: what I forgot, what could be better (brutal, session-scoped)

Per Lars's instruction this is scoped to THIS session's run, not a codebase-wide brutal-self-review.

**Forgot / process misses:**

1. **No durable memory write.** The global AGENTS.md memory protocol says significant findings get written to project memory IMMEDIATELY. The who-uses fold artifact is a textbook gotcha and I wrote nothing into `docs/agents/gotchas-*` or AGENTS.md before being asked for a report. This report is the first durable artifact — the lesson nearly got entombed in chat scrollback.
2. **No todo list on turn 1.** The investigation only got a todo list after Lars forced step-by-step mode. It was a 4-step forensic task from the first message.
3. **Skill loaded only partially.** I viewed the cmdguard SKILL.md's first 100 lines before tooling off; the go-cqrs-lite project skill (which arguably also triggered) was never opened. No harm materialized (the task was dependency forensics, not cmdguard API usage), but the discipline slipped.
4. ** Didn't use the tool's own `why` subcommand** — source-tracing `who_uses.go`/`consumers.go` was slower than asking `project-dependency-graph why cmdguard KeyHolderAI` for a native path first.
5. **My own grep duplicated output lines** in the final verification (grepping both `/direct]/p` and `/via /p`), which I had to explain away as "my artifact" — sloppy evidence presentation.
6. **First reply's one-liner** ("cmdguard shows up in cmd/* tooling") was correct but delivered before the `via` mechanism was understood; a reader could have taken it as the whole story two hours before it was proven.

**Could be better:**

7. Batch the evidence rounds (published .mod fetch + tool source read + downstream manifests) into fewer, parallel rounds.
8. Verify ALL `via` families with equal rigor, or label confidence (verified vs inferred) inline in the first answer — I only did that in the report.
9. Present a one-glance evidence table in the chat answer, not just the report.

## a) FULLY DONE

1. Root cause of the 77-consumer cmdguard tree — identified, code-traced, and reproduced (V1–V7).
2. Premise correction: no cmdguard dependency reaches any downstream app's build (V2, V3).
3. Mechanism documentation with exact `file:line` refs in project-dependency-graph (V5, V6).
4. End-to-end reproduction matching the pasted output, including the 32+45=77 split (V7).
5. Session-scoped self-review (above) and this report.

## b) PARTIALLY DONE

1. Secondary `via` family verification (go-appkit, samber-linter, buildflow/constants, universal-workflow rows): mechanism-inferred only; no per-repo aggregate/go.mod audit.
2. Durability of the finding: nothing yet in go-cqrs-lite gotchas docs, project-dependency-graph docs, or any TODO_LIST — this report is step one, HARVEST pending.
3. `ci-siblings/go-cqrs-lite` fork parse warnings (stale `cmdguard/v5` requires at `v4.1.0` versions): observed in the reproduction run's logs, not triaged — almost certainly expected pin-snapshot state (census-fleet excludes forks by design) but unconfirmed.
4. Indirect consumers' `@ v5.0.0` version backfill: mechanism skimmed (`fillResolvedVersions` → `resolvedTargetVersion`, consumers.go:162-238), not deeply audited.
5. Three unrelated staging repos surfaced with REAL broken go.mods during reproduction (`crm-stage-t13b`, `cv-stage-crmfix`, `erraudit-src` — all `require cmdguard/v5` at v4 versions → hard parse errors): noticed, not fixed (out of scope per instruction).

## c) NOT STARTED

1. Any tool-side change in project-dependency-graph (classification, annotation, flags, tests) — zero lines written.
2. go-cqrs-lite documentation of the artifact (gotchas doc + AGENTS.md cross-link).
3. Fleet-wide audit for the same artifact on other target modules (e.g. `who-uses go-output`, `who-uses go-error-family`).
4. TODO_LIST harvest of this report's section (f).
5. `git log -S cmdguard` historical sweep proving no module EVER required it.
6. Deeper go mod graph spot-check on an app that also imports cqrs-htmx adminui (to rule out any REAL transitive cmdguard path in richer apps).
7. Cleanup decisions for the three broken staging repos and the ci-siblings fork.

## d) TOTALLY FUCKED UP

**Nothing.** The analysis was correct on the first pass and survived reproduction, published-tag verification, and downstream-manifest checks. The nearest misses (unrecorded memory, partial skill load, no turn-1 todos) are process debt, not wrong results — they live in (e). The one thing that WOULD have been a fuckup: presenting the secondary `via` families as verified when they are inferred — that trap was dodged by the scope-line discipline in this report.

## e) WHAT WE SHOULD IMPROVE

1. **Memory latency:** significant forensic findings get written to the right project doc in the same session, not on request (global AGENTS.md rule — violated here).
2. **Todos from message one** on any multi-step investigation.
3. **Full skill load before first tool call** even when the match feels partial.
4. **Use the tool's own query surface first** (`why`) before reading its source.
5. **Confidence labeling as default** in complex findings: every claim marked verified / inferred / observed in the chat answer itself.
6. **Evidence batching** to cut round-trips (this took ~6 rounds; 3 would have sufficed).
7. **Project-dependency-graph UX:** its current output cannot distinguish "real module edge" from "repo-tooling fold" — that ambiguity caused this whole investigation and will cause the next one.

## f) Up to 50 things to get done next (BRAINSTORM — not a commitment list; most are ROADMAP fuel, HARVEST must route with rigor)

**Project-dependency-graph (the tool — highest leverage):**

1. Decide the policy question (see question Q1): is repo-level attribution intended, or should library importers not inherit repo-tooling deps?
2. Classify workspace-member requires: separate "repo tooling" (cmd/* dirs) from "library" requires during aggregate merge (`discovery/workspace.go`).
3. Add a `--attribution=strict|repo` mode to `who-uses` (strict = real module-graph edges only).
4. Annotate folded rows: e.g. `[indirect] via go-cqrs-lite/command@v4.14.1 (repo-aggregate fold)` so the artifact is self-describing.
5. Regression test: synthetic workspace with cmd/* tooling + a consumer app; pin the rendered tree + JSON shape.
6. Golden-pin the current go-cqrs-lite/cmdguard 77-consumer shape as testdata before any change.
7. Extend `why` to print aggregate-fold hops explicitly (audit whether it already does — unverified this session).
8. Emit fold provenance in JSON/SARIF outputs so scripts can filter (`viaKind: module-edge | member-fold`).
9. Document the aggregate fold + its cmd/* side effect in project-dependency-graph README/CONTEXT.md (the fold is intended for cqrs-htmx/setup visibility; the bleed is undocumented).
10. Quiet the known fork noise: `ci-siblings/*` go.mod parse warnings → skip dirs aligned with census-fleet exclusion rules.
11. Verify the renderer never duplicates consumer rows (my duplicated grep lines were mine, but add a dedupe assertion to be safe).
12. Self-test the hidden-indirect accounting (`77 = 32 direct + 45 hidden`) against `--direct-only` flag flips.
13. Audit sibling subcommands for fold semantics: `why`, `serve`, `daemon`, `orphans`, `diff` (all unexamined this session).
14. Consider `hasMain` (already tracked in workspace.go:95-98) to split "app tooling" from "lib" edges in aggregates.
15. Idea: `--exclude-tooling-dirs` heuristic flag for aggregate discovery.
16. Idea: warn when >N% of a target's consumers are indirect-via-fold (artifact smell detector).
17. Idea: module-per-member discovery mode (each go.work member its own node; aggregate as container) — bigger redesign, evidence-gate it.

**go-cqrs-lite (documentation + process):**

18. Add a gotcha to `docs/agents/gotchas-tooling-build.md`: who-uses consumer counts are REPO-folded; `via go-cqrs-lite/<module>` ≠ module edge.
19. Cross-link that gotcha from AGENTS.md Quick Reference (next to the fleet census row) so future sessions don't re-derive it.
20. Note in `census-fleet.sh` docs that census (strict, fork-excluding) and who-uses (folded) answer DIFFERENT questions — numbers will disagree by design.
21. Guard the delete-at-zero trigger: the v5 fork-and-delete playbook cites census numbers; add one line warning that who-uses numbers are inflated by folds.
22. Record the one-line policy fact: go-cqrs-lite repo CLIs standardize on cmdguard v5 (fleet CLI framework) — that IS why the repo is a direct cmdguard consumer.
23. Run `git log -S cmdguard -- '**/go.mod'` to prove no historical library-module require existed (closes the historical-debt question).
24. Harvest section (f) into TODO_LIST.md / ROADMAP.md (docs-health HARVEST) instead of entombing it here.
25. Audit other core modules for the same false-edge shape (`who-uses go-output`, `go-error-family`, `go-idempotency`) and record a known-artifact list.

**Verification debt from this session:**

26. Audit go-appkit's aggregate: does its go.work/tooling reach cmdguard? (explains its own `[indirect]` row + the `via go-appkit` rows).
27. Audit samber-linter, buildflow/constants, universal-workflow rows with the same V1–V3 method.
28. Sample 5 more of the 45 indirect consumers' go.sum files to confirm zero real cmdguard pulls fleet-wide.
29. Full `go mod graph | grep cmdguard` on ONE rich app (KeyHolderAI or InboxClean) as the strongest end-to-end proof.
30. Confirm whether cqrs-htmx adminui/usermgmt CLIs might pull cmdguard via a REAL transitive path (the one scenario where an app could legitimately see cmdguard).

**Staging/fork hygiene (surfaced by the reproduction run):**

31. `crm-stage-t13b`: broken go.mod (cmdguard v5 path at v4.2.1) — upgrade or trash the staging repo.
32. `cv-stage-crmfix`: broken go.mod (cmdguard v5 path at v4.1.0) — same decision.
33. `eventcatalog-hub/work/erraudit-src`: broken go.mod (v4.2.0) — refresh or remove.
34. `ci-siblings/go-cqrs-lite`: decide refresh-vs-delete for the stale fork (5 cmd modules on cmdguard v4 requires).

**Process improvements (from self-review):**

35. Make "verified vs inferred" labeling standard in chat answers, not just reports.
36. Turn-1 todo lists for forensic tasks.
37. Full SKILL.md reads before first tool call.
38. Prefer the tool's query surface (`why`) over source-tracing first.
39. Batch independent evidence commands into single rounds.
40. Immediate AGENTS.md/gotchas writes on significant discovery.

**Idea bucket (ROADMAP fuel, low priority):**

41. who-uses `--via-detail`: full BFS path (aggregate chains) for debugging attribution.
42. Label aggregate roots in the tree: `go-cqrs-lite (go.work aggregate, 104 modules)`.
43. Fold-percentage metric per target in `arch-metrics`-style output for the fleet.
44. Export who-uses JSON for the docserver/website fleet pages (single canonical consumer number per module).
45. Reconcile ADR-0152's edge counts with whichever attribution mode becomes canonical (topology ADRs cite fleet edges — pick ONE source of truth).
46. Pin the post-fix cmdguard tree as a named regression fixture with a friendly name ("the 77-consumer incident").
47. Evaluate whether `who-uses` should default to strict attribution with `--repo-folds` as the opt-in (inverse of item 3).
48. Add a `doctor` check in project-dependency-graph that counts parse-error repos and names them (it warns; a summary line would help).
49. Consider a `--json` field `hiddenIndirect` already present? — verify and document (unaudited).
50. Write the incident up as ONE feature request in project-dependency-graph rather than N scattered fixes; link this report.

## g) Questions I can NOT figure out myself

**Q1 — Policy:** Should `who-uses` attribution stay repo-level (a go.work repo's tooling deps count against every library importer — current behavior), or should strict module-level attribution become the default? This decides whether items 2–5 are a fix or a documentation task. I cannot infer the intended semantics from the code: the fold is deliberately built (`buildMemberEdgeIndex` comments) but its cmd/* bleed is almost certainly accidental.

**Q2 — Durable home:** Where should the lesson live — go-cqrs-lite `gotchas-tooling-build.md` + AGENTS.md cross-link (items 18–19), project-dependency-graph's own docs (item 9), or both? And should I HARVEST section (f) into TODO_LIST.md now (item 24) or leave that to the docs-health pass?

**Q3 — Intent/scope:** Was the underlying worry (a) dependency hygiene ("do apps accidentally depend on cmdguard?" — answer: no, proven) or (b) adoption-metric trust ("blast-radius numbers look inflated")? If (b), do you want the fleet-wide false-edge audit (items 25–29) as the next session, or was cmdguard a one-off?

---

*Report generated per status-report skill (`.md` default per docs/decisions/2026-10-27). Not committed by hand — auto-commit daemon absorbs it (harness rule: no commit without explicit request).*

---

## Addendum (2026-11-11 ~12:40) — ruling executed: fix landed in project-dependency-graph

Lars ruled: *"who-uses is a fucking ~/projects/project-dependency-graph/ responsibility"* — the
attribution behavior is the tool's to own, not a go-cqrs-lite documentation problem. The fix was
implemented in project-dependency-graph the same session (Q1/Q2 resolved by the ruling):

**Semantics shipped:** go.work aggregates retain per-member requires (`types.Module.MemberDeps`,
new field, built in `discovery/workspace.go`). `graph.ConsumersFromModules` folds a member
consumer onto a target only when that member's OWN dependency closure reaches the target
(proven fold); when the target IS the aggregate, folds stay identity-based (importing any member
consumes that repo — the `cqrs-htmx/setup` visibility case). Members without provenance do not
fold; zero-dep members are stored so "proven empty" ≠ "no provenance" (the record-module trap:
a zero-dep member missing from the index pushed its keys through the aggregate-union fallback and
resurrected the bleed mid-implementation — caught by tracing, covered by the new tests).

**Verified results (A/B against a binary built at the pre-fix commit `6b80811f`):**

| Query | Before | After | Removed |
| ----- | ------ | ----- | ------- |
| `who-uses cmdguard` | 77 consumers (45 phantom `via go-cqrs-lite/*`) | 34 (32 direct + 2 true indirect) | all 45 fabrications |
| `who-uses cqrs-htmx` | 26 | 26 (unchanged — intended folds intact) | 0 |
| `who-uses go-cqrs-lite` (submodule expansion) | inflated per-module counts | −1 phantom per library module (dnsblockd-style rows); `cmd/cqrs-lint` 58 → 2 real | fabrications |
| `who-uses go-appkit` | 34 | 33; `via` now names the provable member (`setup`), not an arbitrary fold edge (`identity-model`) | 1 |

The 2 surviving cmdguard indirect rows are module-graph-true, hand-verified against disk:
`bank-sync → go-credentials-store` (gcs go.mod:8 requires cmdguard for its in-module CLI) and
`branching-flow → samber-linter` (samber-linter go.mod:8 = cmdguard CLI). `why KeyHolderAI->cmdguard`
now correctly answers "no path". All 6 modules build + race-test green; graph/discovery/types lint
clean (2 pre-existing `wrapcheck` findings in `dep.go`/`focus.go` — untouched, not this change).

**Residual notes:** (1) heuristic commit `05e5b1ae` swept a FOREIGN flake.nix edit (cmdguard input
pin tag→SHA `56fd69b4`, likely a parallel session) together with this session's source changes —
flagged for the maintainer; no history rewritten because of the detected concurrency.
(2) `project-dependency-graph/go.work` carries stale sibling `use` entries
(`../go-cqrs-lite/catalog` — pre-restructure paths) that flood discovery with skip warnings;
harmless noise, cleanup candidate. (3) Full change record: CHANGELOG `[Unreleased] Fixed` +
README + AGENTS.md in project-dependency-graph; 4 new regression tests in
`graph/consumers_from_modules_test.go` pin the incident shape.
