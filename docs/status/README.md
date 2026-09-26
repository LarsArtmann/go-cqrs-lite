# Status Reports — Historical Snapshots

> **⚠️ These reports are point-in-time snapshots, not living documents.**

Each file captures the project status at a specific timestamp. They are
preserved for audit trail and progress tracking.

## Live reports index (2026-09-22)

| Report                                                                                                              | What it captured                                                               |
| ------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| [2026-09-13 event-query-model not-shipped vs reality](2026-09-13_15-55_event-query-model-not-shipped-vs-reality.md) | KEEP-LIVE evidence: planned event-query surface vs shipped truth               |
| [2026-09-13 T02 verification notes](2026-09-13_17-40_event-query-model-t02-verification-notes.md)                   | KEEP-LIVE evidence: file:line verification behind the reconciliation           |
| [2026-09-17 cqrs-lint FP sweep baseline](2026-09-17_fp-sweep-baseline.md)                                           | KEEP-LIVE baseline: sweep numbers feeding the next lint refresh                |
| [2026-09-20 T18b load-sweep + baseline re-pin](2026-09-20_22-02_t18b-load-sweep-bench-baseline-repin.md)            | T18b #1: perf claims re-anchored under Go 1.27 (arc OPEN — chain armed)        |
| [2026-09-21 T18b gate hardening watch](2026-09-21_10-30_t18b-gate-hardening-verification-watch.md)                  | T18b #2: hardening + rulings implementation watch                              |
| [2026-09-21 T18b promotion + matview lottery](2026-09-21_12-38_t18b-promotion-gate-fix-proven-matview-lottery.md)   | T18b #3: promotion gate fix proven; flake quantified                           |
| [2026-09-21 T18b GOTOOLCHAIN incident](2026-09-21_14-18_t18b-gotochain-incident-armed-closure.md)                   | T18b #4: toolchain incident + guarded re-arm                                   |
| [2026-09-21 T18b rulings encoded](2026-09-21_14-52_t18b-all-rulings-encoded-chain-armed.md)                         | T18b #5: all rulings encoded; chain armed; addenda through 16:10               |
| [2026-09-21 two-session consolidated status](2026-09-21_14-25_two-session-consolidated-status.html)                 | cross-session HTML snapshot (HTML exempt from archiving per standing rule)     |
| [2026-09-21 T18b holding under extreme load](2026-09-21_16-37_t18b-holding-under-extreme-load.md)                   | T18b #6: chain intact at load 863; re-arm one-liner inside                     |
| [2026-09-22 Goal-closure G-T09..T14 wave](2026-09-22_01-19_goal-closure-gt09-t14-polish-tail-wave.md)               | G-T09..T14 + G-T25 gate check + goal-shaped-app tail                           |
| [2026-09-22 11th docs-health pass](2026-09-22_01-20_docs-health-eleventh-pass-full-audit.md)                        | full audit + self-review; M23/go-graph-rag/W0 cluster archived                 |
| [2026-09-22 unblock-machine root cause](2026-09-22_02-30_unblock-machine-root-cause-session.md)                     | go-directive root cause (BuildFlow auto-configure) + Tier-1 fixes              |
| [2026-09-22 full-execution T01–T17](2026-09-22_03-31_full-execution-t01-t17-disjoint-waves.md)                      | Pareto plan T01–T17 disjoint-wave execution                                    |
| [2026-09-22 execution review + verify RED](2026-09-22_11-32_full-execution-review-verify-red-honest-ledger.md)      | T01–T17 review; T04 verify RED; honest ledger                                  |
| [2026-09-22 unblock-session self-review](2026-09-22_11-32_unblock-session-full-self-review-status.md)               | unblock-the-machine full self-review + state                                   |
| [2026-09-22 v5-train T26 paste](2026-09-22_12-58_v5-train-execution-t26-paste.md)                                   | T26 paste items; preflight GREEN; verify armed                                 |
| [2026-09-22 TODO_LIST consistency audit](2026-09-22_14-23_todo-list-consistency-audit-and-fix.md)                   | TODO_LIST vs reports/archives consistency fix                                  |
| [2026-09-22 metaengine scan-family dedup](2026-09-22_23-49_metaengine-scan-family-dedup.md)                         | scan/vector/planned/filter/GraphBFS core extraction                            |
| [2026-09-23 t4 campaign mid-flight](2026-09-23_01-38_t4-campaign-mid-flight-status.md)                              | t4 clone-elimination in-flight status                                          |
| [2026-09-23 EventCatalog overhaul](2026-09-23_04-36_eventcatalog-integration-overhaul.md)                           | EventCatalog integration overhaul                                              |
| [2026-09-23 t4 campaign completion](2026-09-23_05-03_t4-clone-campaign-completion-status.md)                        | t4 clone-elimination completion + self-review                                  |
| [2026-09-23 EventCatalog closeout](2026-09-23_05-14_eventcatalog-verification-closeout-upstream-draft.md)           | verification re-runs + upstream draft                                          |
| [2026-09-23 data-mesh conformance](2026-09-23_18-09_data-mesh-conformance-assessment-session.md)                    | data-mesh conformance assessment                                               |
| [2026-09-23 EventCatalog agent-changelog crash draft](2026-09-23_eventcatalog-agent-changelog-crash-issue-draft.md) | upstream issue draft: agent-changelog crash                                    |
| [2026-09-24 data-mesh execution](2026-09-24_12-26_data-mesh-pareto-execution-session.md)                            | data-mesh Pareto execution T01–T27                                             |
| [2026-09-24 data-mesh completion](2026-09-24_13-32_data-mesh-completion-session.md)                                 | T22–T27 tail + §f queue harvest                                                |
| [2026-09-24 hub-phase + final gate](2026-09-24_18-14_hub-phase-and-final-gate-session.md)                           | hub phase completion + final gate                                              |
| [2026-09-26 TODO-list full-execution session](2026-09-26_16-44_todo-list-full-execution-session.md)                 | whole-TODO execution: ~29 items done, 3 bugs fixed, honest skip/blocker ledger |

**Row-ownership convention (2026-09-22):** for multi-report sessions, the
latest report owns its index row; predecessors are marked superseded in the
row they keep, and fully-superseded reports are archived. New status docs
carrying pseudo-Go fences get `// skip-validate` at WRITE time (the
`#check-md-go` gate fails otherwise — see `scripts/check-md-go.sh`).

**Fully-resolved reports live in [`archived/`](archived/)** (consolidated
2026-08-29 from the older `archive/` + `archived/` split). A report is moved
there once every item it raised is verified resolved, tracked in
[TODO_LIST.md](../../TODO_LIST.md), or superseded — the 2026-08-29 docs-health
audit classified and annotated ~450 August reports and archived ~380 of them,
with 28 stale claims corrected inline. The July archive pass (2026-08-29,
same session) moved all 2026-07 status (240 files) and planning (52 files)
snapshots to `archived/` — July work is shipped or superseded by the August
waves; inbound references from active docs were repointed.

**2026-09-22 (11th docs-health pass):** archived 10 files (the M23 pair, both
go-graph-rag 23-24 reports, the W0 guard-wave report, the 10th-pass report
itself, two SUPERB plans [excellence + publish-and-prove], the go-graph-rag
feedback doc, the md-go-validator review — each bannered + inline-struck),
harvested their §f tails into TODO_LIST (md-go open tail, go-graph-rag follow-ups,
release-train tail, W0 verification tail, docs-health hygiene + the never-harvested
16-37 bench-gate items), deleted every completed receipt row from TODO_LIST
(1,456 → 1,302 lines; evidence lives in CHANGELOG + archived reports), fixed
stale dev-replace comments (system, scheduling/sqlstore), corrected the flake.nix
"all six carry suites" lie, adopted the fluent `.On` chain in the goal-shaped-app
README, annotated + archived the md-go review, and recorded the THIRD go-directive
downgrade incident (140-file wave 23:33 — go.work drift gate row). Live set: the
open T18b arc (6 reports) + three KEEP-LIVE evidence docs. Doc gates re-run green.

**2026-09-21 (10th docs-health pass):** harvested the 2026-09-20/21 cluster
(queue M4/T20, dogfooding pair, 16:39/16:43 close-outs, W3 owner bundle, 9th-pass
audit itself, the M20 wave pair, and the benchkit + T23 skill-maintenance reports —
11 files archived, each with a RESOLVED-BY-ROUTING banner + targeted inline
strikes) and rebuilt TODO_LIST from their forward sections (benchkit test tails,
queue verification tails, T18b chain-hardening + canonical record, go-env.sh
helper, M20 design-ratification follow-ups, M13 stamps). The live set is now the
open T18b arc (6 reports, chain armed), the M10/M23 remainders, and three
KEEP-LIVE evidence docs. Inbound links repointed; doc gates re-run green.
See [TODO_LIST.md](../../TODO_LIST.md).

**2026-09-06 passes (two):** the morning pass archived 83 files (25 status
reports 08-27→09-06, 8 plans + 16 artifacts, 5 reviews, 23 feedback) and
rebuilt TODO_LIST (817→452 lines). The evening pass harvested + inline-
annotated all ten 2026-09-06 session reports (02:40→15:09) plus the cqrs-lint
pareto plan (79 table rows struck), archived them (11 files), added the
missing CHANGELOG `[Unreleased]` wave entries, and extended FEATURES/ROADMAP.

**2026-09-08 pass:** harvested + inline-annotated all thirteen 2026-09-07/08
status reports plus the SUPERB adoption plan + T01 migration note (15 files
archived), rebuilt TODO_LIST (677→~570 lines, zero completed items), added
the missing CHANGELOG entries for the tagged-but-undocumented `otel/v4.4.0` +
`cmd/cqrs-upgrade/v4.0.0` + benchkit system harness, fixed FEATURES (stale
"MySQL claiming rejected" row, missing cqrs-upgrade/matview/turso-encryption
rows, cqrs-bench `/v4` path), README (8 presets incl. bbolt), ROADMAP
([Unreleased] history refreshed through the 09-08 waves), and the skill
references (modules.md cqrs-upgrade + tursoengine rows + a corrupted
mysqlengine row, readmodels.md matview section). New reports land here
unarchived; the next docs-health pass harvests their forward-looking
sections into TODO_LIST/ROADMAP, then archives them.

**2026-09-20 09:40 (publish-and-prove execution):** the SUPERB publish-and-prove
plan executed post-train — the parallel session's 92-tag wave (21:08–21:45) was
verified end-to-end (1314 tags local==remote, proxy @latest, 92 GitHub Releases,
pin-sweep --remote, retracts, audit-baseline 0 NEW, smoke 91/92), post-wave
residue repaired (13 go.sum completions, typedfixture re-pin, scheduling/engine
gate registration, 4 committed binaries untracked), and the W2 code wave landed:
quiet-window gate tooling, canonical-facts + doc-annotations gates wired into
nightly, the camelCase pushdown P0 fix, ctx-scoped transactions across
pg/mysql/duckdb engines, experimental doc.go stamps, the module-map census to
all 95, and the TODO [x] sweep. Composed verify attempt 4 stays queued behind
the host-load gate.

**2026-09-20 16:39 (verify-green close-out):** the composed `#verify` went
GREEN (attempt 10, 14:53–15:04, all 19 phases — S03 recorded in TODO_LIST),
four attempts after the 10:31 concurrent-corruption repair. The path cleared
every dark W2-wave tail: lint godoclint debt (24 findings), the
tx-isolation/graph-SQL clone groups (consolidated into
`adttest.AssertTxIsolationFromForeignContext` + dialect annotations),
templ codegen drift (regenerated with the pinned CLI), plus a genuine
`GracefulClose` pre-cancelled-context select race (fixed + race-stressed).
Shipped alongside: the README deprecation-honesty gate system
(`check-readme-links.sh`/`check-readme-deprecated.sh`, nightly-wired, baseline
0) with the full ADR-0123 sweep (10 banners, 13 README migrations),
`scheduling/engine.ErrEngineNotDueClaimer`, `example/taskmanager v0.2.1`
(proxy-verified `--help` fix — the v4.x git tags are proxy-invisible for the
suffix-less module; v0 is the examples' line), T13 load-sweep GREEN, T15
verify-ci GREEN. T14 bench-baseline supersede deferred to the next quiet
window (fleet ran 20–139 load all afternoon; protocol forbids a tainted
capture). See
[`2026-09-20_16-39_composed-verify-green-s03-w2-tails-cleared.md`](archived/2026-09-20_16-39_composed-verify-green-s03-w2-tails-cleared.md)
and the mid-session reports (10:56, 13:21, 11:36 owner bundle).

**2026-09-08 23:12 (Pareto execution):** the SUPERB plan's Wave 0 shipped —
60 tags pushed (54-module release train + iroh trio + stack/sqlite v4.3.1 +
first proxy-visible example tags), 59 GitHub Releases, 63-module pin-sweep,
release.yml un-red — and the first composed local `#verify` GREEN of the
09-06→09-08 surface. Wave 1 landed P08/P09/P10/P11-core/P12 (SARIF
determinism, matview pins, DSN auth_token leak fix, `--typed-info` +
F090(b), IsQualifierFor sweep) and P13/P15/P16 (AGENTS indexed-split 92→28
KB, GOWORK table, recipes §2.30-2.31). See
[`2026-09-08_23-12_release-train-composed-green.md`](archived/2026-09-08_23-12_release-train-composed-green.md).

**2026-09-09 01:54 (Pareto continuation):** Wave 1/2 remainder — CI re-triage
(WASM-leg codec fix, dogfood/check-csp/lint-scripts jobs, nightly sentinel),
P07 benchkit skipped-run root-cause fix + load scaling, P14 cqrs-upgrade
growth (`--strict`/`--json`/`--to`/`--workspace`), P17 shellcheck zero +
pre-commit cheap gates, P18 coverage gate green, and the P19 first split
wave (storage/sql dialects, cqrs-lint helpers, metaengine typed_reader
1127→6 files). See
[`2026-09-09_01-54_pareto-w1-w2-continuation.md`](archived/2026-09-09_01-54_pareto-w1-w2-continuation.md).

**2026-09-09 04:10 (Pareto W2/W3 execution):** P19 tail through the feasible
P27 chunks all landed — P014 ApplyLayout rule (detection pair corrected from
the T23 addendum), encryption docs+goldens+symmetry, repo hygiene (incl. a
real test data-race fix and the watermill Close≠Nack fix), docs-truth batch,
the v5 wire-key renames with dual-read/dual-write windows + WIRE-FORMAT-KEYS,
migration concurrency hardening (two verify-caught flaws) live-verified on
MariaDB+DuckDB, and release tooling (--smoke, retract v4.8.0, buildinfo
version). Final exclusive `#verify` GREEN after 4 rounds; master synced
(`458eeaac`). Includes the honest fuckup ledger and the next-50 list. See
[`2026-09-09_04-10_pareto-w2-w3-execution-green-verify.md`](archived/2026-09-09_04-10_pareto-w2-w3-execution-green-verify.md).

**2026-09-09..11 sessions (16 reports, archived):** the Pareto tail (cqrs-upgrade
growth + nightly dogfood, watermill issue-#21 typed causation, issue-#20
closeout: `cmd/cqrs-bench/v0.1.1` stub + `cmd/cqrs-lint/v4.10.1` retract-carrier

- `metaengine/badgerengine/v4.2.1` data-loss retracts + tag-release `--audit`/
  `--smoke`), the Cordis 27-task execution (ADR-0136 reset ladder, coeffect gate,
  E018, ADR-0137 engine deactivation, equivalence tooling — all shipped
  2026-09-10), and the 2026-09-11 batch day (dgraph `-shuffle=on` rollout +
  contention fix, turso matview pre.10 re-verification + bench-gate extension,
  M27.16 micro-batch incl. the Demote record-context bug fix + ClaimMetrics
  surfacing, cqrs-lint F091 Tier-3 + T13–T19 audits, go.sum sweep + CI triage).
  See each report under [`archived/`](archived/).

**2026-09-11 pass (5th docs-health audit):** harvested the unharvested
forward items of all 21 active status reports into TODO_LIST (~20 new routed
items: encoded-apply conformance sweep, watermill v4.7.0 tag-wave manifest
entry, calibration provenance, `check-retracts-shipped.sh`, integration-tag
lint gate, error-taxonomy drift gate, and more), deleted 42 completed `[x]`
TODO rows per the file's own header policy + the docs-health skill (evidence
lives in CHANGELOG `[Unreleased]` + these archived reports), closed the stale
GOWORK-decision-table TODO (table shipped 2026-09-08 as P15/P16), fixed
FEATURES (6 missing rows: EngineResetter/ADR-0137 health, coeffect gate,
ClaimMetricsSnapshot, watermill typed causation, scenario equivalence, E018
rule count 204→206), fixed ROADMAP (84 `go.mod` count, Open Question 11),
annotated + archived 21 status reports + 3 planning docs (cordis plan, pareto
plan, t23 design passes — all inline-struck with resolution markers), repointed
5 inbound references, and DECIDED the four-passes-carried exemption rule:
generated HTML dashboards and raw bench `.txt` outputs are inventoried by
title, never annotated. `docs/status/` again holds zero unarchived reports.

**2026-09-11 pass (6th docs-health audit):** harvested + inline-annotated +
archived all eight batch-day reports (04:35→05:51: the 5th-audit report,
cqrs-lint cheap-fix tail, turso IVM repro, code-quality quintet, release-tooling
hardening, encoded-apply sweep, Cordis reset wave, publish/reset wave 0/1) —
64 §f rows struck with verified evidence (every strike grep/tag/gate-proven;
prepared verdicts for 05-34 §f19/§f20 and 05-40 §f07 were OVERTURNED by
verification and left open), 8 RESOLVED-BY-ROUTING banners, open rows routed to
TODO_LIST (~20 new rows, 10 stale rows deleted, 0 `[x]`/90 open) + ROADMAP
(13 raw ideas, OQ 10 answered) + FEATURES/CHANGELOG/AGENTS rebuilds. Also
archived `docs/adr/archived/2026-08-17_system-v4-review-proposals.md` (all 8
proposals verified resolved; dir created) and the 2026-07-23 analytics feedback
pair; the SUPERB plan stays LIVE with ✅12/◐8/open-10 wave markers. Known-open:
`#check-duplication` RED (5 new `reset*.go` clone groups — annotate-vs-re-pin
decision pending). The pass's own reports: midflight
[`06-33`](archived/2026-09-11_06-33_docs-health-sixth-pass-midflight.md)
(archived, superseded) and the completion + self-review
[`2026-09-11_14-10_docs-health-sixth-pass-completion-self-review.md`](archived/2026-09-11_14-10_docs-health-sixth-pass-completion-self-review.md)
(archived by the 8th pass, 2026-09-19).

**2026-09-11 batch-day reports (8 files, all archived; per-file index added
2026-09-13 — this README is the only map of the ~1150-file archive):**

1. [`04-35 docs-health 5th pass — full audit`](archived/2026-09-11_04-35_docs-health-fifth-pass-full-audit.md)
2. [`05-12 cqrs-lint cheap-fix tail`](archived/2026-09-11_05-12_cqrs-lint-cheap-fix-tail-status.md)
3. [`05-21 turso IVM repro suite + version citation`](archived/2026-09-11_05-21_turso-ivm-repro-suite-and-version-citation-session.md)
4. [`05-26 code-quality quintet`](archived/2026-09-11_05-26_code-quality-quintet-complete.md)
5. [`05-34 release-tooling hardening + dogfood sentinel`](archived/2026-09-11_05-34_release-tooling-hardening-and-dogfood-sentinel-first-run.md)
6. [`05-38 encoded-apply fix + conformance sweep + ClaimMetrics + calibration protocol`](archived/2026-09-11_05-38_encoded-apply-fix-conformance-sweep-claimmetrics-calibration-protocol.md)
7. [`05-40 Cordis follow-up wave — reset + failover`](archived/2026-09-11_05-40_cordis-followup-wave-reset-failover.md)
8. [`05-51 publish/reset + v5-train execution wave 0/1`](archived/2026-09-11_05-51_publish-reset-v5-train-execution-wave0-1.md)

**2026-09-13 (quick-win batch, partial index debt):** reconstructed the
orphaned `cec9248da` work record (tripwire + fix.go dedup + PG claiming test
race fix — re-verified, provenance gap closed):
[`archived/2026-09-13_08-42_reconstructed-cec9248da-work-record.md`](archived/2026-09-13_08-42_reconstructed-cec9248da-work-record.md)
(archived 2026-09-16 — no forward-looking items were lost).

**2026-09-15 (quick-win tail batch — calibration-gate self-test + golden,
recipes compile harness):** shipped the three "quick-win batch follow-ups
(2026-09-13)" items: `calibration-gate.sh --self-test` (8-check
fault-injection suite, `CALIB_GATE_LOADAVG_FILE` fixture hook), the
`scripts/testdata/calibration-gate-fail-message.golden` pin
(mutation-tested), and the `cmd/doc-check` recipes.md snippet compile
harness (77/77 blocks classified, coverage ratchet; caught 9 real doc lies
— all fixed in recipes.md + `stack/options.go`). Full a)–g) breakdown:
[`archived/2026-09-15_15-21_quick-win-tail-gates-self-review.md`](archived/2026-09-15_15-21_quick-win-tail-gates-self-review.md)
(archived 2026-09-16 — superseded by the 18:19 completion pass).

**2026-09-15 18:19 (tail-gate COMPLETION pass):** harness driven to zero
failing packages (18 → 0; 12 doc fences fixed incl. the not-legal-Go
`Plan` variadic-spread), full verification sweep green, docs reconciled,
3 authored commits. Honest d)-section: stash-on-shared-tree gamble,
`rm .git/index.lock`, three `--no-verify` commits, stale-queue reuse.
Full a)–g) breakdown:
[`2026-09-15_18-19_quick-win-tail-gates-completion-self-review.md`](archived/2026-09-15_18-19_quick-win-tail-gates-completion-self-review.md)
(archived by the 8th pass, 2026-09-19; §f tails harvested).

**2026-09-16 (7th docs-health pass) — the 2026-09-13..16 wave processed:**
harvested the forward-looking sections of the 2026-09-13..16 reports into
TODO_LIST (queue-family status refresh + tag rows, benchkit statistical-rigor
tail, vector-search verification tail, watermill sibling skill,
md-go-validator CI integration, CI hook/nightly-gate/config-corruption rows)
and ROADMAP (zenoh go/no-go = Open Question #2; `[Unreleased]` release
history refreshed through 09-16; MariaDB vector claim labeled UNVERIFIED);
closed the stale TODO rows the 09-13..16 sessions had shipped past
(`command.rejected` T17, `db.system` dialect spans, CatchUpEngine race,
Doctor entry-point counters, sqlstore hardening tails); verified the recipes
compile gate live (`TestRecipesCompile` → ok). Eleven reports bannered
RESOLVED-BY-ROUTING / STATUS with inline §f strikes; four archived
(`08-42` work record, `08-52` quick-win batch, `15-21` quick-win tail,
`19-57` zenoh clarification — all superseded or fully routed). Living-doc
fixes: FEATURES lifecycle row (+`command.rejected`/`RejectionLog`) + queue
matrix rows + benchkit 151-test count; modules.md 5→6 lifecycle event types;
AGENTS.md 88→91 go.mod + queue family in Tier 4. Reports of this wave that
stay active (open tails, now bannered): the four 09-16 reports (`02-09`
benchkit rigor, `08-04` metaengine follow-ups, `08-05` CI infrastructure,
`09-35` queue-green recovery), `18-32` vector search, `19-27` zenoh,
`19-45` watermill skill, `15-22` rejection follow-ups, `18-19` completion,
and the 09-13 audit wave (`08-47` skill-docs audit, `09-17` backlog,
`12-21` md-go-validator) — see each file's dated banner for what shipped
vs what remains.

**2026-09-16 13:05 (navigation-hardening execution):** all five 09-13
skill-docs TODO items shipped — anchor + § cross-ref validation and a call
arity spot-check now live in `cmd/doc-check` (GitHub-exact slugger,
duplicate-number gate, precision filters, unit-tested; CI-gated at
skill scope), v5-deprecation story consolidated into the faq.md canonical
list with 6 pointers, `example/metaengine-quickstart` linked from both
READMEs, `metaengine.Infer`/`InferFromNamedEvents` deprecated for v5.
Gate yield: 3 real anchor defects fixed. Honest d)-section: the new gate's
silent-skip design hid a defect this session itself introduced (caught by
the layered bash gate); `#verify` deferred a 3rd consecutive session.
Full a)–g) breakdown:
[`2026-09-16_13-05_skill-docs-navigation-hardening-execution.md`](archived/2026-09-16_13-05_skill-docs-navigation-hardening-execution.md)
(archived by the 8th pass, 2026-09-19; §f polished-tail items remain
open in the archived file as the historical record).

**2026-09-19 (8th docs-health pass):** processed the entire 2026-09-13..19
accumulation — 57 session reports read in full, **815+ forward items resolved
inline** (`~~struck~~ done <date> — <evidence>`; evidence = TODO_LIST `[x]` rows

- CHANGELOG `[Unreleased]` dated entries; agent line-drift in one file caught by
  atomic-write validation before any wrong strike), 62 RESOLVED-BY-ROUTING
  banners, **60 session reports archived** (only the two KEEP-LIVE evidence docs
  — `2026-09-13_15-55`, `2026-09-13_17-40` — plus the fp-sweep baseline stay
  active), 9 planning docs annotated + archived (OTEL, 16-01 reconciliation, 18-41
  close-out, durable-work-queue, T16/T17/T18 memos, queue-dedup-seam,
  go-finding-v1.10), 3 planning docs refreshed with dated addenda (publish-reset
  v5 train, pareto-v2, cqrs-to-the-max), 1 review archived (event-command
  duplication), go-graph-rag feedback triaged new/→reviewed/ with routing banner.
  Living docs: TODO_LIST header ledger + 9 harvested rows (README deep-read tail,
  docs censuses, benchkit CLI polish, goal-shaped-app tail, queue M4 polish +
  PapDashboard T20, quiet-window verify tooling, dogfooding follow-ups section,
  go-graph-rag follow-ups, cqrs-lint FP-sweep refresh); README 80+→90+ modules;
  ROADMAP 84→95 go.mod + [Unreleased] history extended through 09-19; AGENTS
  recipes 77→80, module-map 90→95. All inbound references to moved files
  repointed (0 stale). Pass report:
  [`2026-09-19_20-08_docs-health-eighth-pass-full-audit.md`](archived/2026-09-19_20-08_docs-health-eighth-pass-full-audit.md) (now archived — the 9th pass closed its follow-through).

**2026-09-20 (9th docs-health pass):** closed the 09-19/20 arc — 8 session
reports (the 8th-pass audit itself, verify-green/slirp-war, the 00-19 verify
tail, 09-40 publish-and-prove, 10-24 verify-live, 10-25 tag-wave execution,
10-56 corruption repair, 13-21 README-honesty) + 5 planning docs (15-37
verify-green/tag-wave/CRM-ports, command-side depth, the vector-at-scale
spike, publish-reset v5 train, pareto-v2) + 2 reviews (command-side depth —
items 1–2 struck `done 2026-09-13` inline; event-module split re-review) +
4 research docs (turso-8257 POSTED draft, systemd-timer feasibility,
go-taskqueue semantic diff, benchkit tool design — its stale "Phases 6/7
remain" line corrected inline) annotated with resolution banners and
archived. Every banner cites where the work shipped (the 92-tag train, the
S03 composed-verify GREEN, the T19/T20 doc gates) or where the remainder
lives. HARVEST: the 10-25 report's 50-item §f (explicitly marked "harvest
fuel" and never harvested) plus the close-out tails landed as TODO_LIST's
"92-tag release-train tail" + consolidated "Owner decisions — W3 bundle"
sections; stale rows closed (W4 gates + release train, the Go 1.27 wave
section, quiet-window tooling, the module-map census, the cordis
release-train note, a dangling experimental-stamps strike, an empty
load-ordering section). Inbound references repointed (ROADMAP ×3, TODO_LIST,
CHANGELOG ×3, this README). Still active in `docs/status/`: the fp-sweep
baseline, the two KEEP-LIVE evidence docs, the W3 owner bundle
(`2026-09-20_11-36_owner-bundle-w3.md`), and the 16:39/16:43 close-out pair
(live §f backlog).

## What this means

- **Claims of "broken" or "failing" may be resolved.** The codebase evolves
  rapidly. Many items flagged as broken in older reports are fixed in later
  reports or in the current codebase.
- **Module references may be outdated.** Several modules were renamed, merged,
  or deleted between reports (e.g., `readmodel/`, `projection/`, `memory/bus.go`).
- **Coverage numbers and export counts are frozen in time** and will not match
  the current state.

## How to use these reports

1. Read the **newest** report first for the most accurate picture.
2. For the current ground truth, run: `go build ./... && go test ./... -count=1`
3. For current tasks, see [`TODO_LIST.md`](../../TODO_LIST.md).
4. For current features, see [`FEATURES.md`](../../FEATURES.md).

## Quick verification commands

```bash
go build ./...                    # Build health
go test ./... -count=1            # Test health
go vet ./...                      # Static analysis
find . -name "*.go" -not -name "*_test.go" -exec wc -l {} + | sort -rn | head  # Largest files
```

## Link hygiene

Relative markdown links across living docs are checked by
[`scripts/check-doc-links.sh`](../../scripts/check-doc-links.sh) (resolves
symlinked docs like `SKILL.md`, skips fenced code and archived history).
Run it after any doc move; CI-truth for doc references remains `cmd/doc-check`.
