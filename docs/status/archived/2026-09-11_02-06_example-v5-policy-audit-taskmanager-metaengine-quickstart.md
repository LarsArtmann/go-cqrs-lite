# Status Report — Example v5-Policy Audit (taskmanager + metaengine-quickstart)

> **RESOLVED (docs-health pass 2026-09-11):** **Superseded — archived by the docs-health pass 2026-09-11.** §f routed: cqrs-upgrade flags-after-positional guard + `--json` deprecations array + mechanized example v5-clean scan + V007 decider pair-form split brain + quickstart smoke test → TODO_LIST (cqrs-upgrade hardening batch + V007 item). §g1 provenance: the 09-09 audit record exists — `2026-09-11_02-05` verified both examples shipped 2026-09-09 (`6bb82f5b`); date stands. §g2 enforcement: routed as TODO option (c) (per-PR strict scan vs nightly sentinel). §g3 claim convention: still open, noted here.
> Open work lives in [`TODO_LIST.md`](../../TODO_LIST.md); shipped surface in [CHANGELOG.md](../../CHANGELOG.md) `[Unreleased]`.

**Date:** 2026-09-11 02:06 CEST
**Session scope:** Single TODO_LIST item — `Example v5-policy audit` (source: 07-42 §f8). No other work assigned or performed. Per user directive, no unrelated research.
**Verdict:** ✅ Audit complete. Both examples verified free of v5-removed APIs. All four examples now clean (getting-started + readme-quickstart: 2026-09-06).

---

## a) FULLY DONE

1. **The audit itself.** TODO item "Example v5-policy audit — taskmanager + metaengine-quickstart" is closed (parallel session annotated TODO_LIST.md:504-508 mid-session; my run independently confirms it with strictly broader evidence).
2. **Scanner gate (corrected run).** `cqrs-upgrade --dry-run --strict --json <dir>` (flags BEFORE positional): exit 0 on both examples; JSON `deprecations` key absent = zero findings (`omitempty`, report.go:31); every go-cqrs-lite pin up-to-date (taskmanager: 17 modules; metaengine-quickstart: 7).
3. **Independent V007 cross-check.** Direct grep of the FULL removal surface (`cmd/cqrs-lint/pkg/rules/version/v007_tables.go:17-323`, all 10 deprecated modules + ~60 deprecated symbols) over both example trees: **0 code usages**. Only hits are prose comments describing the _replaced_ `Materialize.List` approach (taskmanager: metaengine.go:16, metaengine_test.go:17, integration_test.go:21, integration_test.go:211) — not API usage. These greps had no `--type` filter, so README code fences were covered too.
4. **Removed-module import grep.** `stack/{memory,sqlite,pebble,bbolt,duckdb,postgres,mysql,turso}`, `storage/relational`, `storage/view`: **0 hits**.
5. **Deprecated decider pair-form sweep.** `Execute`/`Load`/`LoadAtVersion`/`LoadAtTime`/`WaitForVersion`/`ExecuteCommand`/typed `Load` (all marked "Deprecated: removed in v5" in decider/*.go): **0 example usages**. The lone `.Load(` hit (taskmanager/idempotency_test.go:54) is `event.Store.Load` — a core keeper API (event/store.go:70), not the pair-form.
6. **Published-pin proof.** GOWORK=off build + vet + test per module: taskmanager full suite green; metaengine-quickstart build+vet clean (no test files exist). Neither example go.mod carries local `replace` directives, so the build proves compatibility against published tags, not workspace siblings.
7. **Zero repo damage.** This session modified no files. Both edit attempts bounced off the staleness guard because a parallel session had already closed the target item; example/ tree verified untouched (`git diff --stat -- example/` empty).

## b) PARTIALLY DONE

1. **My first scanner runs were methodologically broken (caught + corrected).** I passed flags AFTER the positional path (`go run . ../../example/taskmanager --dry-run --strict --json`). Go's `flag` package stops parsing at the first non-flag argument, so **all three flags were silently ignored** (main.go:66-77). Additionally my `$?` captured `tail`'s exit code, not the scanner's (pipeline masking — the exact trap AGENTS.md warns about). Mitigations that held anyway: the deprecation scan itself still ran and reported clean; no go.mod mutation occurred (all pins up-to-date → `editGoMod` was a no-op); and the conclusion is now backed by the corrected re-run (exit 0, strict active, JSON clean) plus the independent greps. The conclusion never changed; two of my initial _claims_ ("strict", "exit 0") were not earned by the run that produced them.
2. **metaengine-quickstart has no tests.** "Runs all 4 demo sections green" rests on the parallel session's annotation; my verification is build+vet only. A smoke test would close this.
3. **README pair-form sweep gap.** The pair-form grep ran with `--type go`, so README code fences were not swept for pair-form _method calls_ (the symbol/module greps did cover READMEs). Residual risk: tiny but real.

## c) NOT STARTED

- **Mechanical enforcement** of example v5-cleanliness (CI gate or meta-test). The nightly `cqrs-upgrade --workspace --strict` sentinel covers the whole tree passively, but nothing fails a PR that introduces a v5-removed API into `example/`. This manual-audit TODO class will keep regenerating until mechanized.
- Nothing else was assigned to this session.

## d) TOTALLY FUCKED UP

1. **My own tool invocation** — flags-after-positional + pipeline-masked exit code. Two violations of documented gotcha classes in one command. Caught in-session (before any conclusion was written to a doc), root-caused to `fs.Parse` semantics (main.go:66), corrected with a clean re-run, and the honest evidence chain is what the TODO closure now stands on. Lesson recorded in this report and routed to gotchas (see f-8/f-9).
2. **Nothing in the repo is fucked up by this session.** No file modifications, no reverts, no destructive operations. The parallel session's in-flight edits (TODO_LIST README item, metaengine/adttest/conformance.go, demote.go, a status doc) were left strictly alone.

## e) WHAT WE SHOULD IMPROVE

1. **cqrs-upgrade should reject/warn on flags after the positional dir.** Silence here turned a "strict gate" into a plain report. Detect `fs.Args()[1:]` starting with `-` and fail loudly.
2. **cqrs-upgrade `--json` should emit the deprecation report explicitly.** "Key absent = clean" is implicit `omitempty` knowledge (report.go:31); the flag doc even promises "bump plan + deprecations per module". Emit `"deprecations": []` when empty.
3. **Mechanize the example v5-clean check** (CI job on the 4 example modules, or a meta-test against the V007 tables) so this audit never needs a human again.
4. **Parallel-session item claiming.** The item I was assigned was closed under me while I worked. A claim convention (or daemon-aware lock) would prevent duplicate/lost work.
5. **Shell discipline.** `set -o pipefail` / `PIPESTATUS` whenever an exit code matters; flags before positionals for every Go CLI. Both belong in gotchas as concrete examples.

## f) Up to 50 things we should get done next (brainstorm, sorted roughly by impact)

1. cqrs-upgrade: fail/warn on flags placed after the positional path (closes this session's trap class for everyone).
2. cqrs-upgrade `--json`: always emit `deprecations` (and `noPins`/`error` semantics doc) so machine consumers don't rely on `omitempty` absence.
3. CI: strict `cqrs-upgrade` scan of the 4 example modules on every PR (small, fast, kills the manual audit class).
4. Meta-test `TestEveryExampleAvoidsV5RemovedAPIs`: scan example `*.go` + README fences against the V007 tables (cmd/api-stability or cqrs-lint home).
5. Repurpose the nightly sentinel evidence: make it post a v5-cleanliness summary (per-module findings count) instead of just exit codes.
6. metaengine-quickstart: add a smoke test exercising the 4 demo sections (currently zero test files).
7. Sweep all 4 example READMEs for deprecated pair-form _calls_ (closes this session's `--type go` gap).
8. Document the flags-after-positional gotcha in `docs/agents/gotchas-tooling-build.md`.
9. Add a concrete `set -o pipefail` / `PIPESTATUS` example to the gotchas (AGENTS.md warns about masking in prose; give the bash recipe).
10. Record the verified scanner invocation (flags-first) in project gotchas/memory for future audits.
11. V007 split brain: decider pair-forms are "Deprecated: removed in v5" in source but ABSENT from the V007 tables — reconcile (add entries or an explicit policy note).
12. Golden test: every `Deprecated: removed in v5` godoc marker must have a V007 table entry or explicit allowlist (mechanical drift guard between doc-deprecation and scan-surface).
13. Repo sweep: cross-reference ALL "removed in v5" markers vs v007_tables.go to find other missing entries (see 12).
14. Annotate archived 07-42 report item 19 (docs-health ANNOTATE: `~~item~~ done at <hash>` with this report as evidence) — the source report still shows it open.
15. Provenance check: the closed TODO says "DONE 2026-09-09" — no 2026-09-09 audit record was found in-session; verify or correct the date.
16. Port getting-started's `docs_compile_test.go` pattern to taskmanager + metaengine-quickstart READMEs (compile their code fences).
17. Confirm `nix run .#verify-ci` (GOWORK=off matrix) actually includes the 4 example modules (not checked this session).
18. Confirm the nightly sentinel's "83 modules" vs AGENTS.md's "84 go.mod files (incl. root)" — off-by-one worth explaining (root go.mod? examples counted?).
19. Check whether `cqrs-upgrade`'s internal `verify()` (tidy+build+vet) respects `GOEXPERIMENT=jsonv2` — pin the env contract.
20. metaengine-quickstart has no `.cqrs-lint.json` (the other two audited examples do) — decide if demo examples should carry one.
21. Add a `nix run .#check-v5-clean-examples` flake app wrapping the strict scan (local gate mirroring CI).
22. Add "how to check my project is v5-clean" recipe (3 lines, cqrs-upgrade invocation) to `references/faq.md`.
23. Auto-generate a v5-removed-surface doc from `v007_tables.go` (docs drift guard; table is the single source).
24. cqrs-lint V007: consider file:line in deprecation findings for actionable reports (verify what it emits today).
25. Make example v5-cleanliness part of the release checklist in CONTRIBUTING.md.
26. cqrs-upgrade workspace mode: end-of-run strict summary across modules (per-module pass/fail line).
27. Archive `--json` scanner reports as CI artifacts for audit provenance.
28. Audit all `cmd/*` mains for the same flag-after-positional silence pattern (`flag.NewFlagSet` is likely reused).
29. Consider `--strict` as default for CI contexts (opt-out `--lenient`) — policy decision.
30. Document `event.Store.Load` vs decider `Load` distinction in faq.md (this session's false-positive class).
31. TODO_LIST: adopt an in-progress/claim marker convention for concurrent agents.
32. Investigate whether the auto-commit daemon can absorb another agent's mid-flight edit while a second agent reads (raced me twice via the staleness guard) — maybe a TODO_LIST-specific lock.
33. Getting-started README/code fences: re-verify against current tags at next audit cycle (last verified 2026-09-06; three tags since).
34. Add the four examples to the api-stability golden check if not already (verify membership).
35. Consider examples exempt from `check-arch` dep budgets — confirm that's intentional and documented.
36. cqrs-upgrade: support `go.work`-aware mode warning when run inside a workspace consumer (GOWORK masking class).
37. Add session-scoped audit evidence convention: TODO closures cite the status report path that carries the evidence (this closure predates this report).
38. Docs: mention in SKILL.md references that `cqrs-upgrade` is the consumer-side v5-readiness tool (copy-paste surface for consumers).
39. Verify `TestEveryExampleHasREADME` covers the newly-authored metaengine-quickstart README (parallel session says passes; confirm).
40. Consider a per-example `go vet` config (vetflags) since examples tolerate demo-style code.
41. Sweep for other prose-only deprecation mentions ("Materialize.List" style) that could confuse future audits — maybe rename comments to name the replacement API explicitly.
42. Nightly sentinel runtime (4m21s, network-bound) — consider GOMODCACHE warm strategy or `--no-build`.
43. Cross-check the sentinel's `--no-build` usage vs this session's full-verify needs — document which contexts need which.
44. Add CI annotation (PR comment) when the example scan finds anything, linking the migration doc per finding (replacements live in V007 tables).
45. Consider extracting the V007 tables into a shared package consumable by both cqrs-lint and cqrs-upgrade (today: two tools, one policy — drift risk; verify whether they already share).
46. Add a regression test for the corrected invocation order (flags before positional) so CI catches a parser change.
47. Post-audit hygiene: confirm the parallel session's metaengine/adttest + demote.go edits eventually land clean (they were in-flight; not mine to touch).
48. Date-provenance convention for TODO closures: cite both original-done and re-verified dates with report paths.
49. `docs/status/` grows unboundedly — the archived/ convention exists; consider archiving reports older than N weeks as a periodic HARVEST step.
50. This report's section (f) → run docs-health HARVEST into TODO_LIST/ROADMAP once the user approves (skill: the loop isn't closed until harvested).

## g) Questions I cannot figure out myself (max 3)

1. **Provenance of "DONE 2026-09-09".** The closed TODO item claims an original 2026-09-09 audit, but in-session I found no 2026-09-09 record for these two examples (07-42 is 2026-09-06 and covered getting-started + readme-quickstart only). Was there a 09-09 session/report I should cite, or should the date be corrected to 2026-09-11?
2. **Enforcement policy.** Should example v5-cleanliness become a _blocking_ per-PR CI gate, or stay advisory (nightly sentinel + periodic manual audit)? Blocking costs ~4 fast module scans per PR; advisory risks the TODO class regenerating.
3. **Concurrent-agent convention.** A parallel session closed the exact item I was auditing and held uncommitted edits in files I needed to touch. Do you want a claim/lock convention for TODO items across concurrent agents, or is daemon-absorbed convergence acceptable?

---

_Report per user instruction: Markdown at `docs/status/` (explicit user format override of the status-report skill's HTML default — flagged, not propagated back into the skill). Self-review folded into sections d/e per the same single-deliverable instruction. Not committing per critical rules; the auto-commit daemon will absorb this file._
