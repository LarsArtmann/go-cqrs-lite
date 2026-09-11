# Status Reports — Historical Snapshots

> **⚠️ These reports are point-in-time snapshots, not living documents.**

Each file captures the project status at a specific timestamp. They are
preserved for audit trail and progress tracking.

**Fully-resolved reports live in [`archived/`](archived/)** (consolidated
2026-08-29 from the older `archive/` + `archived/` split). A report is moved
there once every item it raised is verified resolved, tracked in
[TODO_LIST.md](../../TODO_LIST.md), or superseded — the 2026-08-29 docs-health
audit classified and annotated ~450 August reports and archived ~380 of them,
with 28 stale claims corrected inline. The July archive pass (2026-08-29,
same session) moved all 2026-07 status (240 files) and planning (52 files)
snapshots to `archived/` — July work is shipped or superseded by the August
waves; inbound references from active docs were repointed.

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
+ `metaengine/badgerengine/v4.2.1` data-loss retracts + tag-release `--audit`/
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
