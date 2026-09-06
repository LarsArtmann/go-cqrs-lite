# Status Report — cqrs-lint v5-Hardening Pareto Execution (T01–T24)

**Date:** 2026-09-06 02:40 CEST
**Scope:** execution of `docs/planning/2026-09-06_00-31_cqrs-lint-v5-hardening-pareto-plan.md`
(24 medium tasks T01–T24, 96 micro-tasks F001–F096), started from the
verified-green baseline `1039f92a4`.
**Baseline when this wave started:** `nix run .#verify` EXIT 0, tree clean at
`1039f92a4`, plan approved for execution with the three decision-gate defaults
(release now · keep V007 warning · examples suppressed-but-modernized).

---

## a) Fully done — verified

| Task                                                        | What shipped                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        | Evidence                                                                                                                                                                                                                                                                                                                                                            |
| ----------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **T02 — V007 drift meta-test**                              | Two-directional drift contract (`v007_drift_scan_test.go`, `v007_drift_test.go`): every repo `Deprecated: … v5` marker must be covered by the V007 tables or an explicit allowlist; every table entry must have a live marker. Building it caught and fixed a shipped bug: `stripVersionSuffix` only stripped a _trailing_ `/vN`, so `storage/v4/relational` and `storage/v4/view` — 2 of the 10 wholly-removed modules — could never fire. Expanded tables by 30 symbols (ParseType shims, `snapshot.SaveSnapshot`, `graph.Handler/ProjectionOption/WithSchema`, `listing.StatusMiddleware`, `storage/sql.KeysetPositionQuery`, storage-root re-exports of the view+relational tiers), removed 2 dead table entries that pointed at methods (`stack.RunProjections`, `metadata.EnsureCustom`), added v5/ADR citations to table-backed markers that lacked them, and gave the unmarked `storage/relational_aliases.go` re-exports ADR-0123 markers. | tests `TestV007_TablesCoverAllV5DeprecationMarkers`, `TestV007_TableEntriesHaveLiveMarkers`, `TestV007_FragmentSpaceMatchesDetector` + 5 new E2E detector tests, all pass; module suite green; commit `1e12d7e8d` (file attribution partially shredded by the auto-commit daemon into `5c246ea56`/`eae55e0f8`/`7711bf0b6` — content verified by suite, not by diff) |
| **T03 — getting-started modernized**                        | Example now teaches the v5 idiom: `system.New` (DomainConfig + DeploymentConfig), `system.RegisterDecider`/`RegisterCommand` → `system.Execute`, read model declared as metaengine folds (`OnRecordTyped` create+update), explicitly started projection host, reads via `metaengine.NewReader.Get`. Engine selection isolated in `buildSystem(dsn)`; SQLite driver blank-import documented as part of the deployment story. Test runs the full pipeline against a real SQLite DB (proves the one-line engine swap); `main()` uses memory. `docs/getting-started.md` snippets + `docs_compile_test.go` moved to `ExecuteRef`/`LoadRef`/`id.NewStreamID` (the pair forms die at v5). Example README rewritten. Policy recorded in AGENTS.md (Q3 default: examples stay self-lint exempt but v5-clean anyway).                                                                                                                                         | `go test` passes in workspace AND `GOWORK=off` standalone; `go run` prints `value=10`; `cqrs-lint` reports **0 findings** on the example; commit `dece8ccab` (+ daemon commits `f03ebaaa6`, `e3105d91b`, `278a3d41f`)                                                                                                                                               |
| **T01 — v4.9.0 released**                                   | Annotated tag `cmd/cqrs-lint/v4.9.0` cut via `scripts/tag-release.sh` (dry-run first), pushed to origin. Post-tag assertions: tagged `main.go` has quoted `const version = "4.9.0"` (the v4.8.0 poisoning class), `/v4` module path, zero local replaces. **Proxy-verified**: `go run …/cmd/cqrs-lint/v4@v4.9.0 version` → `cqrs-lint 4.9.0`, `rules` → **204 rules**. CHANGELOG release section stamped _before_ tagging (deviation from plan order, F011 done early so the tag carries it). Version constant in source bumped to keep `TestVersionMatchesLatestTag` green. No sibling consumers pin cmd/cqrs-lint (verified), so no pin sweep was needed.                                                                                                                                                                                                                                                                                         | tag on origin; clean `go run @v4.9.0` from a fresh dir through the module proxy; full module suite green post-bump                                                                                                                                                                                                                                                  |
| **T04 — severity/confidence meta-test**                     | `severity_meta_test.go` parses every `finding.NewBuilder` site in the rules tree and joins builder-declared severity/confidence against the catalog. Found **14 real split-brains**, all fixed with per-rule judgment: S008/S009 (asymmetric signing/encryption) warned while the catalog declared **error** — security findings now actually error; P006 (busy-poll) and B021 (fold-without-StrictApply) understated severity; A018/C003/C027 confidence aligned; catalog corrected where the builder was right (A017 → warning/high, B004 → medium, E007 → info/low). README severity cells updated in lockstep. Conditional detectors (B008/C008/S002/S006) and 50 helper-mediated rule families are explicitly allowlisted.                                                                                                                                                                                                                     | `TestRuleSeverityMatchesCatalog`, `TestCatalogRulesDeclareFindings` pass; full module suite green; commit `fbb10c0bb`                                                                                                                                                                                                                                               |
| **F043 — `rules --json`** (pulled forward from T10)         | Rule catalog machine-readable: stable field names (id, name, category, severity, confidence, autoFix, docUrl) for editor/tooling consumers. Verified: 204 entries, V007 metadata complete.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | CLI output checked; README documents it                                                                                                                                                                                                                                                                                                                             |
| **T05 — suppression parser tail**                           | Four blind spots closed: (1) multiple directives on one line — only the first parsed, later ones swallowed; (2) multi-space/tab between `//` and `cqrs-lint:` defeated block directives; (3) directives as literal text inside `/* */` were honored as real suppressions — block-comment interiors are now inert and within-line detection understands `/* */`; (4) stray `ignore-end` and never-closed `ignore-start` (suppresses to EOF!) were invisible — `--fail-on-stale-suppressions` now fails on both, with remediation hints. `parser_edge_test.go` (10 tests) added; CLI long-help + README document the syntax.                                                                                                                                                                                                                                                                                                                          | all suppression tests green; full module suite green; taskmanager lint output byte-identical to pre-change (no regressions); commit `ebfb5bc17`                                                                                                                                                                                                                     |
| **T06 — fix-provider unification**                          | `CanHandle` accepted only BeforeCode/AfterCode while `Edits` also knows Metadata old/new — a metadata-only fix source was unreachable. Both sources now accepted, with a round-trip test. Learning: go-finding validation forbids `FixStrategyDirect` without code data, so the pipeline's `HasFix()` gate stays code-change driven; the metadata path is defense-in-depth for direct callers.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | `TestCQRSFixProvider_MetadataOnlyRoundTrip` passes; `pkg/fix` suite green                                                                                                                                                                                                                                                                                           |
| **T07 — lintutil convergence**                              | (1) `lastSegment` handled only v2–v9 — a v10+ import resolved to package name "v10"; now v2–v99. (2) The Register/Handle denylist matched bare qualifier names, so a consumer's own package named `mux`/`http` was wrongly skipped — matching now resolves the qualifier to its import path first. (3) `QualifierToImportPath` returned the path of a **dot import for any qualifier** — dot imports bind no qualifier; the false-attribution branch is removed and the old behavior-pinning test updated. (4) Dead exports removed: `ImportQualifierMap` (zero callers) and `SelectorIdent` unexported.                                                                                                                                                                                                                                                                                                                                            | lintutil/api/boilerplate/consistency packages green; whole-module suite green; commit `0e74e2296`                                                                                                                                                                                                                                                                   |
| **Self-healing format guard** (T12 partial, pulled forward) | The auto-commit daemon's reformat sweep re-added `gci` to `formatters.enable` a **third** time. `check-formatters.sh` now self-heals: strips the gci entry in place and re-verifies, instead of only reporting (missing pinned formatters still fail). Verified: REPAIR path runs, exit 0, module lint 0 issues afterward.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | commit `0e74e2296`                                                                                                                                                                                                                                                                                                                                                  |

**Gates run during the wave (per-task discipline):** full cmd/cqrs-lint module
test suite (17 packages) green after every task; `golangci-lint run ./...` on
the module 0 issues; doc-check EXIT 0; changelog-symbols gate EXIT 0 (157
citations verified); example standalone `GOWORK=off` test green; CHANGELOG
`[Unreleased]` carries three new subsections (drift contract, severity
contract, suppression hardening).

---

## b) Partially done

1. ~~**T08 — dead links / RULES.md (F036–F037): investigation complete, artifact missing.**~~ done by follow-up session at `4e4b514d7` (RULES.md with 204 anchors, byte-freshness + DocURL meta-tests). Enumerated all 9 DocURL citations (`RULES.md#c001`, `#c008`, `#c010`, `#c017`, `#e016`, `#e017`, `#p012`, `#s002`, `#s010`) — every one points at a file that does not exist. The generator script for `cmd/cqrs-lint/RULES.md` was written but **failed on a Python quoting bug and produced no file**. Nothing committed for T08. TODO_LIST already carries the item.
2. ~~**F031 — `cqrs-lint --fix` E2E: attempted, exposed a real integration gap, not closed.**~~ RESOLVED at `e44da78fa` — root cause was C003 anchoring its Direct fix at the function declaration (the occurrence-safe provider rightly refused); NOT an upstream go-finding bug. Real-CLI `--fix` on the preserved fixture mutates exactly the targeted occurrence. Built a live consumer fixture that triggers C003 (verified the finding fires, `fixStrategy=direct` in JSON output). Running `--fix` performs **no file mutation** — and `--dry-run` shows no fix preview either. The provider layer is unit-proven correct (position-based matching, line-scoped fallback, no-edit-when-not-on-line), so the gap sits in the go-finding/pipeline@v1.6.0 fix-applier handoff (suspect: FixApplier rootDir vs absolute finding paths, or a silent compile-check rollback). I refused to fake a green test around it; documented here + TODO_LIST instead (see d/g).
3. ~~**T12 — ledger hygiene (F045–F046): F046's substance shipped early.**~~ done — F045 (double-`---` artifact) fixed at `d805c0af2`. The mid-gate tree-mutation prohibition and verify-log conventions were folded into the self-healing guard work and AGENTS notes; the TODO_LIST double-`---` artifact fix (F045) is **not done**.
4. **Daemon attribution integrity:** roughly 40% of this wave's file changes were committed by the auto-commit daemon under `chore: auto-commit N changed file(s)` messages (waves of 1–272 files), with my descriptive commits landing on the residual diffs. Nothing is lost — every task's final state is suite-verified — but per-file history attribution is shredded and bisecting a future regression to a _logical_ change will require the task commits' messages, not file diffs.

## c) Not started

- ~~**T09 — CI wiring (F038–F041):**~~ done 2026-09-06 — jobs wired in ci.yml (`cqrs-lint-self-lint`, `cqrs-lint-examples`, `check-lint-config`) + `V007-DEMO.md`; only F040 required-checks remains (owner decision).
- ~~**T10 — rule-ID gap documentation (F042):**~~ done at `d805c0af2` (README §Rule ID numbering gaps; F043 `rules --json` also shipped).
- ~~**T11 — V007 overhead measurement (F044):**~~ done at `5c4ebd0be` (`docs/benchmarks/2026-09-06_cqrs-lint-v007-walltime.md`).
- **T13–T19 — the seven rule-audit batches (F047–F069):** C001–C042, A001–A034, B001–B031, D001–D019, E001–E017, S/T/V/F families. The severity meta-test pre-verified 109 builder sites and the drift test pre-verified the V tables, but the per-rule logic/FP review has not begun.
- **T20 — scanner + feature_profile review (F070–F076).**
- **T21 — CLI subsystem review (F077–F084),** including the scorecard deprecated-module panel (F082) and health-policy test (F083).
- ~~**T22 — misc hardening (F085–F088):**~~ done — shuffle eval ADOPT + `-race -count=3` green (two sessions), ruletest alias helper + preset V007/F030 pins at `d341d95bd`.
- ~~**T23 — design passes (F089–F091):**~~ done at `d341d95bd` (`docs/planning/2026-09-06_cqrs-lint-t23-design-passes.md`); implementations remain tracked in TODO_LIST.
- ~~**T24 — sibling checks (F092–F093):**~~ done — links verified (session-3 log); docserver-CSS regenerate rule in CONTRIBUTING at `35851f652`.
- **G1/F094 and G2/F095 — the two full `nix run .#verify` gates:** not run in this session. Per-task module gates were run continuously instead; the full gate was deliberately deferred to avoid mutating the tree under a running gate — which turned out to be impossible anyway, see (d).
- ~~**F096 — plan check-off/annotation.**~~ done — execution logs + docs-health pass 2026-09-06 (79 rows struck inline).

## d) What's totally fucked up

1. **The `.golangci.yml` war with the auto-commit daemon is unwinnable manually — I lost it three times in one session.** The daemon's repo-wide reformat sweep re-added `gci` to `formatters.enable` at ~00:xx, again at ~01:xx, and again at ~02:23 (`082f49a2a`, a 1,680-line config rewrite that also stripped my explanatory NOTE). Each resurrection silently re-breaks the lint gate for hundreds of treefmt-clean files. The self-healing `check-formatters.sh` converts this from "manual chore + silent gate rot" to "auto-repair at next gate run", but the root loop is still: daemon re-adds → script strips → daemon re-adds. **The honest fix is upstream of the repo: the daemon's formatter needs to stop writing `gci` (or stop touching this file).** Until then, every status is potentially stale within minutes for this one line.
2. **The daemon also shredded commit attribution for ~40% of the wave** (see b.4). My commit discipline (commit before the daemon shreds) worked for message-level history but not file-level: several tasks' changes landed in daemon `chore:` commits minutes before my descriptive commit, so my commits carry only the residual diff (e.g. `0e74e2296` shows 2 files, not the ~10 its message describes). Anyone auditing this wave must read the task messages and trust the test runs, not `git show`.
3. **A 272-file dirty tree existed while I was working, twice** (`9a6dfb179` committed 272 files mid-session; another 272-file wave was dirty at report time). This violates the repo's own concurrent-session hygiene rule (never mutate the tree under a gate / never touch foreign dirty files) — not by my choice, and it means **G1 has not been run and cannot be meaningfully run until the tree stabilizes**: a full verify against a tree that a background process is rewriting produces unverifiable greens. The standing rule "never mutate the tree while a gate runs" is currently unenforceable.
4. **I shipped an inverted test assertion** (`TestFilter_SuppressionInsideBlockCommentIsInert` asserted the buggy outcome) in the T05 commit `ebfb5bc17`, and the suite caught it only when the next task's full run hit the suppression package. Fixed in `0e74e2296`. The miss happened because I ran the new tests in the same breath as writing them and read "FAIL" as "new test red → parser wrong" instead of re-reading the assertion polarity; the debug cycle that followed (standalone harness, then zz_debug_test.go) cost ~20 minutes.
5. **The RULES.md generator script failed on a Python quoting error** (nested quotes in a jq one-liner inside a heredoc) — T08 produced zero artifacts on its first attempt. Trivial bug, but it is the difference between "T08 done" and "T08 not done", and this report is the correction of that.
6. **`git add -A` habit risk:** because the daemon kept pre-committing my files, I used `git add -A` for two task commits. It worked here (tree contained only my + daemon-swept files), but `add -A` in a repo where a background process dirties the tree can silently commit unrelated mid-flight changes. Should revert to explicit paths.

## e) Improvements made (the "every change raises the bar" ledger)

- **Two shipped-bug classes found by the new meta-tests on day one:** the V007 mid-path `/vN` normalization bug (2 of 10 removed modules were undetectable — exactly the consumer-facing failure the rule exists to prevent) and 14 severity/confidence split-brains (including security rules that warned instead of erroring). Both test contracts now make these classes structurally impossible.
- **Rule quality:** V007's removal surface grew from 10 modules + 30 symbols to 10 modules + 59 symbols, all marker-backed and drift-enforced; two permanently-dead table entries removed.
- **Suppression parser:** four blind spots closed, all in the "consumer tried to silence a finding and silently failed" direction — the worst failure mode a linter can have.
- **Consumer-facing API:** `rules --json` (204 entries, stable fields) for editor integration.
- **Self-healing gate:** `check-formatters.sh` upgrades from tripwire to repair loop.
- **Example truthfulness:** the repo's front-door example no longer teaches the composition root that dies at v5, and its test proves the persistence story end-to-end instead of asserting a sleep-based race.
- **Docs:** three CHANGELOG subsections, README severity cells in lockstep (meta-tested next wave), suppression syntax documented in CLI help + README, getting-started README rewritten.

## f) Next items (50, in execution order)

**Immediate — close the wave (1–6)**

1. F036/F037: fix the RULES.md generator quoting bug; generate the stub with `<a id>` anchors for all 204 IDs; give V007's section its four ADR links; add a docurl_test case that every catalog DocURL anchor exists in RULES.md.
2. F044: V007 wall-time measurement (self-lint ± V007) → `docs/benchmarks/cqrs-lint-v007-overhead.md`.
3. F042: README rule-ID gap note (A028, A031, P002–P005, S004, D004 reserved/retired).
4. F045: fix TODO_LIST double-`---` artifact from the session-4 insertion.
5. G1/F094: full `nix run .#verify` — **only after the daemon's dirty-tree churn settles**; then commit/push.
6. F096 (incremental): annotate the plan file — T01–T07 done, F031 blocked-on-upstream, F011 reordered pre-tag.

**F031 root cause (7–10)**

7. Reproduce the C003 direct-fix non-application in a go-finding pipeline unit test (temp file + FixApplier) to isolate pipeline vs CLI.
8. If pipeline: check `FixApplier.rootDir` resolution against absolute finding paths (fixe2e fixture preserved at `/home/lars/projects/.gotmp/fixe2e`).
9. Fix upstream in go-finding or add a cqrs-lint-side pre-resolve (findings → absolute paths before pipeline).
10. Land the real F031 E2E: fixture → `--fix` → diff asserts the default-branch occurrence edited, first occurrence untouched.

**CI + proof (11–16)**

11. F038: CI self-lint job (exit-code policy: errors block, warnings report).
12. F039: examples lint matrix job (4 examples, zero-findings assertion).
13. F040: make `check-lint-config` a required check.
14. F041: capture V007 demo output → release notes snippet.
15. F085: `-shuffle=on` eval on the cqrs-lint suite; record verdict.
16. F086: `-race -count=3` on suppression + fix packages.

**Hardening tail (17–22)**

17. F087: ruletest alias-import fixture helper.
18. F088: preset V007/F030 on/off policy pins.
19. F082: scorecard deprecated-module-usage panel (F030/V007 data).
20. F083: health-policy test for V007 deduction behavior.
21. F028-followup: suppression docs audit in `cqrs-lint explain`.
22. F093: docserver-CSS drift root cause → dep-bump checklist in CONTRIBUTING.

**Rule audit batches (23–44, 90m each, only behind green G1)**

23. F047–F049: audit C001–C014 (severity/confidence already meta-verified; focus FP logic + emitted metadata).
24. F050–F053: audit C015–C042.
25. F054–F055: audit A001–A018 (incl. A017's dual-severity variant documentation).
26. F056–F058: audit A019–A034 + B001–B015.
27. F059–F061: audit B016–B031 + P001–P013.
28. F062–F065: audit D001–D019 + E001–E017.
29. F066: audit S001–S011 (S008/S009 severity change → verify consumer exit-code impact).
30. F067: audit T001–T008 + V001–V006.
31. F068–F069: audit F001–F030 (adoption family, helper-mediated — resolve the 50-entry allowlist into real verification).
32. F070–F072: scanner.go + scanner_calls/folds/resolve/adapters review.
33. F073–F074: feature_profile.go split-candidate review + feature_detect review.
34. F075–F076: loader/registry/upcaster + module_catalog review.
35. F077: doctor*.go review.
36. F078: health.go + score computation review.
37. F079: scorecard*.go review (4 files).
38. F080: output*.go + diagnostics review.
39. F081: explain/commands/config_loader/init/aggregate review.
40. F084: fix mechanical findings from the CLI subsystem reviews.

**Design + closure (41–50)**

41. F089: `v5-ready` preset design (severity escalation mechanism) — needs Q2 policy answer.
42. F090: dot-import detection design (V007 scope extension).
43. F091: typed-info (`packages.Types`) integration design for name-heuristic FPs (C008/C035 class).
44. F092: verify go-sse/cqrs-htmx replacement links in F030 findings.
45. F031-closure (see 7–10) if not resolved earlier.
46. G2/F095: final full verify gate.
47. F096: final plan check-off + deviation annotations.
48. Re-tag decision: if the severity tightening (S008/S009 now error) warrants it, cut v4.9.1 with a CHANGELOG "Changed" note (question 3 below).
49. Update AGENTS gotcha 18 to reference the self-healing guard instead of "remove gci to fix".
50. Consider a follow-up plan wave for the docs sweep the asrecord/MIGRATION_TO_STACK/PRESETS guides need once v5 nears (they teach the dying stack surface — accurate until the cut, then wrong).

## g) Questions I cannot answer myself (max 3)

1. ~~**`--fix` writes nothing despite `fixStrategy=direct` findings (F031).**~~ ANSWERED — not an upstream bug; see b2 above (`e44da78fa`). The gap is somewhere in go-finding/pipeline@v1.6.0's FixApplier (backup/rollback/compile-check path) or its path resolution against absolute finding paths — that repo is yours, and I don't know whether a compile-check gate is _supposed_ to silently skip writes in non-CI environments. Should I debug it upstream in go-finding (it's your lib, so a fix + tag there unblocks the real E2E here), or is there a documented flag/config I'm missing that makes the pipeline's fix stage a no-op by default?
2. **The auto-commit daemon keeps re-adding `gci` to `.golangci.yml` (3 incidents) and running repo-wide reformats that dirty 272 files mid-session.** My self-healing script now repairs the config at every gate run, but the churn makes "clean tree" — and therefore a trustworthy full `#verify` — impossible while it runs. Where does the daemon's formatter/sweep config live, and can `.golangci.yml` be excluded from its rewrite set (or its formatter pointed at a gci-free canonical)? Alternatively: should `check-formatters.sh`'s repair also `git add` the repaired file so the next daemon pass diffs against the healed state?
3. **Severity-tightening release policy:** S008/S009 now emit `error` (were `warning`), and the catalog/README alignment changed several rules' advertised severity/confidence. Consumers using `--min-severity error` or CI fail-on-error will see new failures after upgrading to ≥v4.9.0 — behaviorally correct, but it is a silent behavior change inside a minor release. Is severity-tightening acceptable in a minor (documented in CHANGELOG Added), or do you want severity changes gated behind a "Changed" section + a dedicated minor (v4.10.0) with the current tag annotated/superseded?

---

**Verification snapshot at report time:** `cmd/cqrs-lint` module suite green
(17 packages), module lint 0 issues, doc-check 0 warnings, changelog-symbols
0 violations, v4.9.0 proxy-verified (`cqrs-lint 4.9.0`, 204 rules), example
suite green standalone. Full `#verify` NOT run (blocked by live daemon churn —
see d.3). Known dirty tree at report time is daemon-owned (catalog/_,
system/_, .golangci.yml sweep), not session work.
