# Publish-Integrity Pareto Plan — Execution Session 2 (continuation)

> **RESOLVED-BY-ROUTING (docs-health 12th pass, 2026-09-28):** Publish-integrity session 2 — M14/M19/M25/M15/M16 done (receipts in TODO rows: onset-matrix doc, issue #9391, quiet-campaign.sh + dgraph-calibration-leg.sh); M5+M20 armed window-dependent (the TODO calibration row owns the campaign; harvest procedure in plan §8 addendum); M22 owner Q3 + benchkit/LICENSE are TODO rows (12th-pass harvest). The 'Not done / blocked' trio is fully tracked.

**Written:** 2026-09-28 05:40 CEST
**Scope:** continuation of [`2026-09-28_02-22_publish-integrity-pareto-execution-session.md`](2026-09-28_02-22_publish-integrity-pareto-execution-session.md) — the remaining plan tasks after the report demand. Plan: [`docs/planning/2026-09-28_01-26_SUPERB-publish-integrity-pareto-plan.md`](../planning/2026-09-28_01-26_SUPERB-publish-integrity-pareto-plan.md).

## Done this session

- **M14 finished:** full 15-config bisect sweep ran (16.8 s, PASS) → matrix recorded in [`docs/benchmarks/2026-09-28_ivm-defect-a-onset-matrix.md`](../benchmarks/2026-09-28_ivm-defect-a-onset-matrix.md) + TODO receipt. Headlines: single-tx loads ALWAYS exact; ≤64 groups exact at any tx size; 2000 single-member groups exact — divergence needs (≥2nd tx) × (groups in 10²–10³ band), onset tx#2 for ≥500-row txs (tx#4/tx#33 at 100/10-row), loss PARTIAL (≈1–5 %), draft's 430.50 delta reproduces byte-for-byte.
- **M19 — upstream issue FILED: [tursodatabase/turso#9391](https://github.com/tursodatabase/turso/issues/9391)** (zombie-tx readback; verify-before-filing gates all passed; standalone repro re-verified same-day on v0.7.2 `@latest` AND v0.8.0-pre.13; github-voice check-draft 0 FAIL/0 WARN). NEW findings folded in: wall moved to 29k; on pre.13 the aborted chunk now PERSISTS after fresh reopen + a commit-error write still lands; on v0.7.2 fresh reopen dies with `API misuse: unknown database open flags`. The A+B standalone issue stays owner-gated (TODO row updated with the new numbers for its eventual body).
- **M25 shipped:** `scripts/quiet-campaign.sh` (batch orchestrator over `quiet-window-run.sh`; self-test 5/5 green ×2, mutation-verified) + `scripts/dgraph-calibration-leg.sh` (M20 leg: calibration-gate WITH provenance → both dgraph benches count=5). Plan §8 addendum documents invocation + the detached-campaign harvest procedure.
- **M5+M20 ARMED, window-dependent:** campaign detached (`nohup setsid`, deadline 10:56 CEST) under a sustained load storm (load ≈ 15–20 vs ceiling 5). Logs: `/tmp/quiet-campaign-session.log`. It fires unattended; harvest steps in the plan §8 addendum.
- **M23 (receipt):** the v5-removal census ALREADY EXISTS as gate machinery — `v007_tables.go` + bidirectional drift tests (green re-run). No census doc needed; do not hand-maintain what the gate derives.
- **M24 (receipt):** ADR-0142 §Decision 2 ↔ TODO row citation verified current; no drift.
- **M21 (receipt):** BLOCKED — ADC healthy (`lars@helpless.ai`), 9 accessible projects, ZERO Bigtable instances; creating one is billable owner-scope. Recorded in FEATURES:230.
- **M15 finished:** new rows — batch-release run-log, M13-tail gate-derived engine/ADT/driver counts, Legend receipt-convention, pkg.go.dev-license FAQ note; 3 claims spot-verified (engine/driver census via grep, modernc v1.59.0 has NO LoadExtension, matrix doc = verbatim harness output).
- **M16 green:** doc-check 1 218 refs / 0 warnings; readme-deprecated clean; readme-links 0 broken; changelog-symbols 42 green; md-go 1 463 blocks valid; check-duplication 0 new clones.
- **Lint/meta:** metaengine/system/tursoengine — my files zero findings (the golangci "not formatted" warnings are the documented repo-wide treefmt-vs-gci split; canonical `nix fmt` stable at 0 changed). api-stability TestEvery green; api golden unchanged (same-package splits move no exports).
- **File-size gate GREEN** (was 7 violations, CI-red): split `catalog/docserver/docserver.go` 351→220 (`serve.go`), `catalog/eventcatalog/frontmatter_convert.go` 364→206 (`frontmatter_convert_flow.go`), `catalog/cmd/ec-fixture/main.go` 356→292 (`fixture_flow.go`), `cmd/cqrs-lint/.../scanner_calls.go` 367→310 (`scanner_generic.go`), `.../lintutil/lintutil.go` 491→345 (`lintutil_imports.go`), `.../dsn_resolver.go` 460→318 (`dsn_evidence.go`), `scenario/dsl.go` 365→253 (`dsl_projection.go`). All module tests green after each split. `projectionhost/host.go` ratchet incident fixed format-stably (art-dupl directive on its own line above the `if`; golines@treefmt vs @golangci max-len conflict documented by the fix).

## Not done / blocked

~~- **M5+M20 completion** — window-dependent (see above); everything up to the measurement is shipped.~~ — routed — M5+M20 armed (TODO calibration row; harvest steps in plan §8); M22 is the TODO owner-Q3 row; owner mechanics rowed in Release/data-mesh sections

- **M22** — owner Q3 (report-artifact policy) still unanswered; gates only this.
- **Owner questions carried forward:** the stalled 6-tag wave; tursoengine v4.2.1 cut timing; benchkit/LICENSE "Unknown Author".

## Mistakes (honest)

1. Trashed the live campaign's log dir during /tmp cleanup (restored from `/tmp/.Trash-1000` — the campaign was still in its wait phase, zero data lost; lesson: check `ps` for live writers before /tmp sweeps).
2. First campaign launch passed a compound `&&` command through the wrapper — quoting mangled (killed instantly, replaced by the leg-script pattern + documented the constraint).
3. Two `head -N | cp` truncations clipped mid-function (closing brace lost, `foldOrFatal` dropped) — both caught by the compiler on the immediate build, both fixed before commit.
4. One bad `--log-dir` invocation in the first M5 launch (usage error, harmless).

## Session hygiene

Scratch trashed: `/tmp/goget-*`, `/tmp/metaengine-v4.14.0.zip`, `/tmp/turso-zombie` (repro lives inline in issue #9391), self-test campaign dirs (live dir kept).
