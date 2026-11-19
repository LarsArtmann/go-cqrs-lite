# Archive ritual part 1 - triage of the docs/status corpus (2026-11-19)

**Pass:** 16th docs-health annotate/archive pass (part 1: classify; part 2 commits the
`git mv` execution + digest row the same day).

**Method:** mechanical, whole-corpus - every top-level `docs/status/2026-*.md`
report (195 files) classified by (a) date relative to the live-index window and
(b) open TODO checkbox count (`^- [ ]`), per the docs-health doctrine:
annotate/keep/pool. Narrative open-item text is NOT counted - only unchecked
checkbox lines are pool evidence.

**Verdicts:**

- **KEEP (53):** dated >= 2026-11-03, covered by the live reports index in
  `README.md` (2026-11-03 harvest window through today). Remain in place.
- **ANNOTATE + ARCHIVE (141):** dated <= 2026-11-02 with ZERO unchecked TODO
  boxes - resolved snapshots; part 2 bulk-annotates (dated resolved banner) and
  `git mv`s them to `docs/status/archived/`.
- **POOL (1):** carries unchecked TODO boxes; its items must be harvested into
  `TODO_LIST.md` (or ruled) before it can ever archive. NOT archived in part 2.

## KEEP (53)

| Report | Date | Open boxes |
| --- | --- | --- |
| [`2026-11-03_16-22_todo-read-session-self-review-and-status.md`](2026-11-03_16-22_todo-read-session-self-review-and-status.md) | 2026-11-03 | — |
| [`2026-11-03_16-55_cqrs-lint-c044-branded-string-hardening.md`](2026-11-03_16-55_cqrs-lint-c044-branded-string-hardening.md) | 2026-11-03 | — |
| [`2026-11-03_17-31_readme-agents-superb-session.md`](2026-11-03_17-31_readme-agents-superb-session.md) | 2026-11-03 | — |
| [`2026-11-03_18-42_system-promotion-out-of-experimental.md`](2026-11-03_18-42_system-promotion-out-of-experimental.md) | 2026-11-03 | — |
| [`2026-11-03_19-09_rawquery-warning-e023-unreleased-finding-doc-truth-fixes.md`](2026-11-03_19-09_rawquery-warning-e023-unreleased-finding-doc-truth-fixes.md) | 2026-11-03 | — |
| [`2026-11-03_19-41_todo-list-done-row-sweep-changelog-sync.md`](2026-11-03_19-41_todo-list-done-row-sweep-changelog-sync.md) | 2026-11-03 | — |
| [`2026-11-03_21-15_tier-1pct-red-to-green-execution.md`](2026-11-03_21-15_tier-1pct-red-to-green-execution.md) | 2026-11-03 | — |
| [`2026-11-03_22-15_pareto-t01-trust-floor-execution.md`](2026-11-03_22-15_pareto-t01-trust-floor-execution.md) | 2026-11-03 | — |
| [`2026-11-03_22-24_cqrs-lint-f032-catalog-mandate-t009-test-framework-map.md`](2026-11-03_22-24_cqrs-lint-f032-catalog-mandate-t009-test-framework-map.md) | 2026-11-03 | — |
| [`2026-11-04_02-18_pareto-t05-t09-ci-triage-continuation.md`](2026-11-04_02-18_pareto-t05-t09-ci-triage-continuation.md) | 2026-11-04 | — |
| [`2026-11-04_04-50_gosec-t009-t06-t08-session.md`](2026-11-04_04-50_gosec-t009-t06-t08-session.md) | 2026-11-04 | — |
| [`2026-11-04_04-56_gosec-t009-t06-t08-brutal-status.md`](2026-11-04_04-56_gosec-t009-t06-t08-brutal-status.md) | 2026-11-04 | — |
| [`2026-11-04_05-51_art-dupl-triage-dedup-consolidation.md`](2026-11-04_05-51_art-dupl-triage-dedup-consolidation.md) | 2026-11-04 | — |
| [`2026-11-04_23-17_open-issues-sweep-status-report.md`](2026-11-04_23-17_open-issues-sweep-status-report.md) | 2026-11-04 | — |
| [`2026-11-05_01-14_open-issues-sweep-completion-status.md`](2026-11-05_01-14_open-issues-sweep-completion-status.md) | 2026-11-05 | — |
| [`2026-11-05_06-39_post-sweep-plan-execution-interim-status.md`](2026-11-05_06-39_post-sweep-plan-execution-interim-status.md) | 2026-11-05 | — |
| [`2026-11-05_09-46_ci-repair-and-verify-receipts-status.md`](2026-11-05_09-46_ci-repair-and-verify-receipts-status.md) | 2026-11-05 | — |
| [`2026-11-05_18-10_sse-root-cause-ci-triage-verify-interrupted.md`](2026-11-05_18-10_sse-root-cause-ci-triage-verify-interrupted.md) | 2026-11-05 | — |
| [`2026-11-05_18-16_fleet-audit-followup-plan-one-week-review.md`](2026-11-05_18-16_fleet-audit-followup-plan-one-week-review.md) | 2026-11-05 | — |
| [`2026-11-05_19-08_resume-watcher-armed-prep-complete.md`](2026-11-05_19-08_resume-watcher-armed-prep-complete.md) | 2026-11-05 | — |
| [`2026-11-05_21-45_issue-114-landed-all-green-verify-armed.md`](2026-11-05_21-45_issue-114-landed-all-green-verify-armed.md) | 2026-11-05 | — |
| [`2026-11-05_23-52_verify-fast-fail-adr-index-split-brain.md`](2026-11-05_23-52_verify-fast-fail-adr-index-split-brain.md) | 2026-11-05 | — |
| [`2026-11-06_10-54_issue-sweep-116-115-114-verify-sentinels-prefetch-wiring-per-group-caps.md`](2026-11-06_10-54_issue-sweep-116-115-114-verify-sentinels-prefetch-wiring-per-group-caps.md) | 2026-11-06 | — |
| [`2026-11-06_18-15_post-sweep-pareto-execution-complete.md`](2026-11-06_18-15_post-sweep-pareto-execution-complete.md) | 2026-11-06 | — |
| [`2026-11-06_21-28_post-sweep-pareto-final-status.md`](2026-11-06_21-28_post-sweep-pareto-final-status.md) | 2026-11-06 | — |
| [`2026-11-07_02-01_post-park-ci-stabilization-eight-defect-classes.md`](2026-11-07_02-01_post-park-ci-stabilization-eight-defect-classes.md) | 2026-11-07 | — |
| [`2026-11-07_07-27_datastar-client-side-research.md`](2026-11-07_07-27_datastar-client-side-research.md) | 2026-11-07 | — |
| [`2026-11-08_08-52_brutal-audit-ci-fuzz-doc-drift-status.md`](2026-11-08_08-52_brutal-audit-ci-fuzz-doc-drift-status.md) | 2026-11-08 | — |
| [`2026-11-08_16-00_brutal-episode7-oteli-injection-gate-repairs-status.md`](2026-11-08_16-00_brutal-episode7-oteli-injection-gate-repairs-status.md) | 2026-11-08 | — |
| [`2026-11-08_16-06_standard-topology-nixos-cue-adr0165-status.md`](2026-11-08_16-06_standard-topology-nixos-cue-adr0165-status.md) | 2026-11-08 | — |
| [`2026-11-08_16-35_irohengine-testing-ladder-dnsblockd-fit-session.md`](2026-11-08_16-35_irohengine-testing-ladder-dnsblockd-fit-session.md) | 2026-11-08 | — |
| [`2026-11-08_18-30_standard-topology-completion-arc-status.md`](2026-11-08_18-30_standard-topology-completion-arc-status.md) | 2026-11-08 | — |
| [`2026-11-08_21-53_adr0165-completion-final-status.md`](2026-11-08_21-53_adr0165-completion-final-status.md) | 2026-11-08 | — |
| [`2026-11-11_09-16_cmdguard-downstream-attribution-forensics-status.md`](2026-11-11_09-16_cmdguard-downstream-attribution-forensics-status.md) | 2026-11-11 | — |
| [`2026-11-11_12-41_who-uses-member-fold-fix-status.md`](2026-11-11_12-41_who-uses-member-fold-fix-status.md) | 2026-11-11 | — |
| [`2026-11-11_16-42_issue-128-checkpoint-dispatch-fix-and-gate-revival.md`](2026-11-11_16-42_issue-128-checkpoint-dispatch-fix-and-gate-revival.md) | 2026-11-11 | — |
| [`2026-11-11_20-24_nine-issue-fix-wave.md`](2026-11-11_20-24_nine-issue-fix-wave.md) | 2026-11-11 | — |
| [`2026-11-13_14-54_crush-session-library-deep-dive-gfl-integration.md`](2026-11-13_14-54_crush-session-library-deep-dive-gfl-integration.md) | 2026-11-13 | — |
| [`2026-11-13_20-22_issues-130-131-fold-rejection-cell-versioning.md`](2026-11-13_20-22_issues-130-131-fold-rejection-cell-versioning.md) | 2026-11-13 | — |
| [`2026-11-14_01-57_issues-130-131-closeout-gate-repairs-verify-riding.md`](2026-11-14_01-57_issues-130-131-closeout-gate-repairs-verify-riding.md) | 2026-11-14 | — |
| [`2026-11-14_01-58_w0-truth-green-execution-harvest-redgates-origin-triage.md`](2026-11-14_01-58_w0-truth-green-execution-harvest-redgates-origin-triage.md) | 2026-11-14 | — |
| [`2026-11-14_05-36_w0-critical-path-chain-storms-m35-staged-release-tooling-fixes.md`](2026-11-14_05-36_w0-critical-path-chain-storms-m35-staged-release-tooling-fixes.md) | 2026-11-14 | — |
| [`2026-11-14_07-21_w0-continuation-m36-m38-complete-m39-six-gates-chain-riding.md`](2026-11-14_07-21_w0-continuation-m36-m38-complete-m39-six-gates-chain-riding.md) | 2026-11-14 | — |
| [`2026-11-14_18-46_w0-chain-died-on-v003-coverage-debt-m36-m39-complete.md`](2026-11-14_18-46_w0-chain-died-on-v003-coverage-debt-m36-m39-complete.md) | 2026-11-14 | — |
| [`2026-11-15_08-17_w1-m39-residue-cleared-m40-two-thirds-chain5-armed.md`](2026-11-15_08-17_w1-m39-residue-cleared-m40-two-thirds-chain5-armed.md) | 2026-11-15 | — |
| [`2026-11-15_11-51_w2-m40-m52-done-chain5-died-on-receipt-hygiene.md`](2026-11-15_11-51_w2-m40-m52-done-chain5-died-on-receipt-hygiene.md) | 2026-11-15 | — |
| [`2026-11-15_13-51_w3-t27-promotion-adr0167-chain6-riding.md`](2026-11-15_13-51_w3-t27-promotion-adr0167-chain6-riding.md) | 2026-11-15 | — |
| [`2026-11-17_12-26_idempotency-module-usage-review-and-doc-reconciliation.md`](2026-11-17_12-26_idempotency-module-usage-review-and-doc-reconciliation.md) | 2026-11-17 | — |
| [`2026-11-17_22-38_idempotency-debt-review-cqrs-lint-truth-fixes-and-harvest.md`](2026-11-17_22-38_idempotency-debt-review-cqrs-lint-truth-fixes-and-harvest.md) | 2026-11-17 | — |
| [`2026-11-18_07-02_linter-truth-follow-through-and-fleet-survey-persisted.md`](2026-11-18_07-02_linter-truth-follow-through-and-fleet-survey-persisted.md) | 2026-11-18 | — |
| [`2026-11-18_07-05_postmortem-cqrs-lint-truth-session.md`](2026-11-18_07-05_postmortem-cqrs-lint-truth-session.md) | 2026-11-18 | — |
| [`2026-11-18_08-27_f030-registry-unification-and-the-missing-verify-receipt.md`](2026-11-18_08-27_f030-registry-unification-and-the-missing-verify-receipt.md) | 2026-11-18 | — |
| [`2026-11-19_20-35_data-model-review-session-postmortem.md`](2026-11-19_20-35_data-model-review-session-postmortem.md) | 2026-11-19 | — |

## ANNOTATE + ARCHIVE (141)

| Report | Date | Open boxes |
| --- | --- | --- |
| [`docs/status/archived/2026-09-13_15-55_event-query-model-not-shipped-vs-reality.md`](archived/2026-09-13_15-55_event-query-model-not-shipped-vs-reality.md) | 2026-09-13 | — |
| [`docs/status/archived/2026-09-13_17-40_event-query-model-t02-verification-notes.md`](archived/2026-09-13_17-40_event-query-model-t02-verification-notes.md) | 2026-09-13 | — |
| [`docs/status/archived/2026-09-17_fp-sweep-baseline.md`](archived/2026-09-17_fp-sweep-baseline.md) | 2026-09-17 | — |
| [`docs/status/archived/2026-09-23_eventcatalog-agent-changelog-crash-issue-draft.md`](archived/2026-09-23_eventcatalog-agent-changelog-crash-issue-draft.md) | 2026-09-23 | — |
| [`docs/status/archived/2026-10-01_04-30_config-war-recurrence-killed-ddl-dedup-absorbed-breakages-m24-enricher.md`](archived/2026-10-01_04-30_config-war-recurrence-killed-ddl-dedup-absorbed-breakages-m24-enricher.md) | 2026-10-01 | — |
| [`docs/status/archived/2026-10-01_05-33_templ-components-audit-and-docserver-utilization-fixes.md`](archived/2026-10-01_05-33_templ-components-audit-and-docserver-utilization-fixes.md) | 2026-10-01 | — |
| [`docs/status/archived/2026-10-01_05-54_catalog-eventcatalog-integration-research-and-render-gate-verification.md`](archived/2026-10-01_05-54_catalog-eventcatalog-integration-research-and-render-gate-verification.md) | 2026-10-01 | — |
| [`docs/status/archived/2026-10-01_06-15_embedded-eventcatalog-static-server-shipped.md`](archived/2026-10-01_06-15_embedded-eventcatalog-static-server-shipped.md) | 2026-10-01 | — |
| [`docs/status/archived/2026-10-01_06-41_bdd-suite-embedded-eventcatalog-server-plus-config-war-8-found.md`](archived/2026-10-01_06-41_bdd-suite-embedded-eventcatalog-server-plus-config-war-8-found.md) | 2026-10-01 | — |
| [`docs/status/archived/2026-10-01_09-56_c043-shipped-mesh-gate-demo-preflight-four-reds-absorbed-m04-still-load-blocked.md`](archived/2026-10-01_09-56_c043-shipped-mesh-gate-demo-preflight-four-reds-absorbed-m04-still-load-blocked.md) | 2026-10-01 | — |
| [`docs/status/archived/2026-10-01_14-50_config-war-8-killed-sigpipe-audit-complete-fp-sweep-turso-pin-leg-m04-gate-green-verify-refused-on-rebound.md`](archived/2026-10-01_14-50_config-war-8-killed-sigpipe-audit-complete-fp-sweep-turso-pin-leg-m04-gate-green-verify-refused-on-rebound.md) | 2026-10-01 | — |
| [`docs/status/archived/2026-10-01_23-10_config-war-9-absorbed-linter-trust-fixes-loadbyeventid-shipped-chain-died-loud.md`](archived/2026-10-01_23-10_config-war-9-absorbed-linter-trust-fixes-loadbyeventid-shipped-chain-died-loud.md) | 2026-10-01 | — |
| [`docs/status/archived/2026-10-03_02-12_cqrs-lint-store-detection-explained-stale-untracked-binary.md`](archived/2026-10-03_02-12_cqrs-lint-store-detection-explained-stale-untracked-binary.md) | 2026-10-03 | — |
| [`docs/status/archived/2026-10-03_03-44_multi-store-detection-design-proposed-caveat-gaps-concurrent-edits.md`](archived/2026-10-03_03-44_multi-store-detection-design-proposed-caveat-gaps-concurrent-edits.md) | 2026-10-03 | — |
| [`docs/status/archived/2026-10-03_04-25_cqrs-lint-subcommand-consistency.md`](archived/2026-10-03_04-25_cqrs-lint-subcommand-consistency.md) | 2026-10-03 | — |
| [`docs/status/archived/2026-10-03_05-58_cqrs-lint-cli-consistency-pareto-execution.md`](archived/2026-10-03_05-58_cqrs-lint-cli-consistency-pareto-execution.md) | 2026-10-03 | — |
| [`docs/status/archived/2026-10-03_06-00_cli-consistency-closeout-self-review.md`](archived/2026-10-03_06-00_cli-consistency-closeout-self-review.md) | 2026-10-03 | — |
| [`docs/status/archived/2026-10-03_06-14_multi-store-detection-shipped-jsonv2-guess-cost-environment-archaeology.md`](archived/2026-10-03_06-14_multi-store-detection-shipped-jsonv2-guess-cost-environment-archaeology.md) | 2026-10-03 | — |
| [`docs/status/archived/2026-10-03_06-15_nsfw-feedback-triage-pushdown-utilization.md`](archived/2026-10-03_06-15_nsfw-feedback-triage-pushdown-utilization.md) | 2026-10-03 | — |
| [`docs/status/archived/2026-10-03_07-45_post-train-completion-bench-format-fix-config-repair.md`](archived/2026-10-03_07-45_post-train-completion-bench-format-fix-config-repair.md) | 2026-10-03 | — |
| [`docs/status/archived/2026-10-04_02-37_pareto-plan-and-wave1-2-execution-session.md`](archived/2026-10-04_02-37_pareto-plan-and-wave1-2-execution-session.md) | 2026-10-04 | — |
| [`docs/status/archived/2026-10-04_03-25_wave2-t09-t10-t12-execution.md`](archived/2026-10-04_03-25_wave2-t09-t10-t12-execution.md) | 2026-10-04 | — |
| [`docs/status/archived/2026-10-04_05-05_asyncapi-request-reply-leverage-status.md`](archived/2026-10-04_05-05_asyncapi-request-reply-leverage-status.md) | 2026-10-04 | — |
| [`docs/status/archived/2026-10-04_06-27_signing-v4.3.3-encoding-fix-issue-52.md`](archived/2026-10-04_06-27_signing-v4.3.3-encoding-fix-issue-52.md) | 2026-10-04 | — |
| [`docs/status/archived/2026-10-08_13-11_branch-pr-issue-review-and-ci-red-campaign.md`](archived/2026-10-08_13-11_branch-pr-issue-review-and-ci-red-campaign.md) | 2026-10-08 | — |
| [`docs/status/archived/2026-10-08_14-27_wire-form-leak-fix-campaign.md`](archived/2026-10-08_14-27_wire-form-leak-fix-campaign.md) | 2026-10-08 | — |
| [`docs/status/archived/2026-10-08_16-35_v5-goal-execution-w0-shipped-config-war-repaired.md`](archived/2026-10-08_16-35_v5-goal-execution-w0-shipped-config-war-repaired.md) | 2026-10-08 | — |
| [`docs/status/archived/2026-10-08_18-30_turso-bump-dedup-cleared-stack-seam-shipped-bridge-tagged-unpushed.md`](archived/2026-10-08_18-30_turso-bump-dedup-cleared-stack-seam-shipped-bridge-tagged-unpushed.md) | 2026-10-08 | — |
| [`docs/status/archived/2026-10-08_19-15_w1-t06-closed-config-war-7-repaired-t07-core-half-tested.md`](archived/2026-10-08_19-15_w1-t06-closed-config-war-7-repaired-t07-core-half-tested.md) | 2026-10-08 | — |
| [`docs/status/archived/2026-10-08_20-00_w1-t07-sqlite-compound-cursor-landed-legacy-loop-fixed.md`](archived/2026-10-08_20-00_w1-t07-sqlite-compound-cursor-landed-legacy-loop-fixed.md) | 2026-10-08 | — |
| [`docs/status/archived/2026-10-08_20-02_humanize-linter-findings-resolved.md`](archived/2026-10-08_20-02_humanize-linter-findings-resolved.md) | 2026-10-08 | — |
| [`docs/status/archived/2026-10-08_20-23_go-modules-audit-external-usage-and-self-review.md`](archived/2026-10-08_20-23_go-modules-audit-external-usage-and-self-review.md) | 2026-10-08 | — |
| [`docs/status/archived/2026-10-08_20-38_w1-t07-kv-unification-duckdb-landed.md`](archived/2026-10-08_20-38_w1-t07-kv-unification-duckdb-landed.md) | 2026-10-08 | — |
| [`docs/status/archived/2026-10-08_21-03_w1-t07-pg-mysql-landed-gates-red.md`](archived/2026-10-08_21-03_w1-t07-pg-mysql-landed-gates-red.md) | 2026-10-08 | — |
| [`docs/status/archived/2026-10-08_21-09_w1-t07-dup-gate-cleared-preflight-7of9.md`](archived/2026-10-08_21-09_w1-t07-dup-gate-cleared-preflight-7of9.md) | 2026-10-08 | — |
| [`docs/status/archived/2026-10-09_00-32_w1-t03-verify-load-refusals-t07-fully-verified.md`](archived/2026-10-09_00-32_w1-t03-verify-load-refusals-t07-fully-verified.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_02-41_cqrs-lint-purpose-recalibration-session-review.md`](archived/2026-10-09_02-41_cqrs-lint-purpose-recalibration-session-review.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_02-46_axon5-bdd-testing-analysis-session-review.md`](archived/2026-10-09_02-46_axon5-bdd-testing-analysis-session-review.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_02-47_wal-extraction-design-exploration-session-review.md`](archived/2026-10-09_02-47_wal-extraction-design-exploration-session-review.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_05-23_scorecard-ratchet-execution-session-review.md`](archived/2026-10-09_05-23_scorecard-ratchet-execution-session-review.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_06-16_t03-verify-three-deep-runs-systemscenario-blocked.md`](archived/2026-10-09_06-16_t03-verify-three-deep-runs-systemscenario-blocked.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_14-40_systemscenario-bdd-harness-full-execution-status.md`](archived/2026-10-09_14-40_systemscenario-bdd-harness-full-execution-status.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_16-08_v5-declarative-schema-evolution-proposal-session.md`](archived/2026-10-09_16-08_v5-declarative-schema-evolution-proposal-session.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_17-32_scorecard-followup-wave-session-review.md`](archived/2026-10-09_17-32_scorecard-followup-wave-session-review.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_17-57_v5-schema-evolution-t2-t1-shipped.md`](archived/2026-10-09_17-57_v5-schema-evolution-t2-t1-shipped.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_20-05_t03-chain-v3-armed-self-review-sibling-wave.md`](archived/2026-10-09_20-05_t03-chain-v3-armed-self-review-sibling-wave.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_20-17_branching-flow-triage-session-review.md`](archived/2026-10-09_20-17_branching-flow-triage-session-review.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_20-45_scorecard-probes-multi-module-fix-session-review.md`](archived/2026-10-09_20-45_scorecard-probes-multi-module-fix-session-review.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-09_22-10_t03-chain-v4-armed-durable-preflight-wired.md`](archived/2026-10-09_22-10_t03-chain-v4-armed-durable-preflight-wired.md) | 2026-10-09 | — |
| [`docs/status/archived/2026-10-10_03-40_adoption-wave-t01-t13-execution-status.md`](archived/2026-10-10_03-40_adoption-wave-t01-t13-execution-status.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_04-01_tag-wave-4of5-live-systemscenario-blocked.md`](archived/2026-10-10_04-01_tag-wave-4of5-live-systemscenario-blocked.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_05-13_superfly-ltx-deep-review-session.md`](archived/2026-10-10_05-13_superfly-ltx-deep-review-session.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_05-48_graph-native-docs-audit-allengines-consult-session.md`](archived/2026-10-10_05-48_graph-native-docs-audit-allengines-consult-session.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_05-54_bdd-adoption-wave-t14-t19-tag-cut-migrations-hardening.md`](archived/2026-10-10_05-54_bdd-adoption-wave-t14-t19-tag-cut-migrations-hardening.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_07-27_bdd-adoption-wave-t20-complete-t21-t22-done-t23-research.md`](archived/2026-10-10_07-27_bdd-adoption-wave-t20-complete-t21-t22-done-t23-research.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_07-37_graph-native-wave-execution-status.md`](archived/2026-10-10_07-37_graph-native-wave-execution-status.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_08-43_bdd-adoption-wave-t23-t25-complete-t26-partial-watermill-guard-shipped.md`](archived/2026-10-10_08-43_bdd-adoption-wave-t23-t25-complete-t26-partial-watermill-guard-shipped.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_08-45_v5-schema-plan-phase0-done-m7-inflight.md`](archived/2026-10-10_08-45_v5-schema-plan-phase0-done-m7-inflight.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_09-05_graph-native-wave-closure-brutal-self-review.md`](archived/2026-10-10_09-05_graph-native-wave-closure-brutal-self-review.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_09-49_v5-schema-plan-m7-m8-done-m9-research.md`](archived/2026-10-10_09-49_v5-schema-plan-m7-m8-done-m9-research.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_11-51_bdd-adoption-wave-t26-complete-t27-verify-load-blocked.md`](archived/2026-10-10_11-51_bdd-adoption-wave-t26-complete-t27-verify-load-blocked.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_12-34_dedup-campaign-templ-refactor-baseline-surgery.md`](archived/2026-10-10_12-34_dedup-campaign-templ-refactor-baseline-surgery.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_13-50_branching-flow-followup-m4-m5-lockstep-lint-config-recovery.md`](archived/2026-10-10_13-50_branching-flow-followup-m4-m5-lockstep-lint-config-recovery.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_17-02_post-review-defaults-harvest-capability-gate-adr-gate.md`](archived/2026-10-10_17-02_post-review-defaults-harvest-capability-gate-adr-gate.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_21-10_post-session-brutal-self-review-harvest-and-gates.md`](archived/2026-10-10_21-10_post-session-brutal-self-review-harvest-and-gates.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_23-46_v5-schema-plan-evening-closed-m9-m26-self-review.md`](archived/2026-10-10_23-46_v5-schema-plan-evening-closed-m9-m26-self-review.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-10_23-47_wal-journal-exploration-follow-up-execution-session-review.md`](archived/2026-10-10_23-47_wal-journal-exploration-follow-up-execution-session-review.md) | 2026-10-10 | — |
| [`docs/status/archived/2026-10-11_03-21_branch-consolidation-buildflow-stabilization-self-review.md`](archived/2026-10-11_03-21_branch-consolidation-buildflow-stabilization-self-review.md) | 2026-10-11 | — |
| [`docs/status/archived/2026-10-11_03-21_branching-flow-triage-dedup-generics.md`](archived/2026-10-11_03-21_branching-flow-triage-dedup-generics.md) | 2026-10-11 | — |
| [`docs/status/archived/2026-10-11_03-57_strong-id-fix-wave-issue-10.md`](archived/2026-10-11_03-57_strong-id-fix-wave-issue-10.md) | 2026-10-11 | — |
| [`docs/status/archived/2026-10-11_07-28_idempotency-placement-metaengine-dedup-investigation-status.md`](archived/2026-10-11_07-28_idempotency-placement-metaengine-dedup-investigation-status.md) | 2026-10-11 | — |
| [`docs/status/archived/2026-10-11_19-27_why-build-was-broken-cqrs-lint-triage.md`](archived/2026-10-11_19-27_why-build-was-broken-cqrs-lint-triage.md) | 2026-10-11 | — |
| [`docs/status/archived/2026-10-11_20-12_docs-health-13th-pass-2026-0-mandate-21-archived-index-rot-killed.md`](archived/2026-10-11_20-12_docs-health-13th-pass-2026-0-mandate-21-archived-index-rot-killed.md) | 2026-10-11 | — |
| [`docs/status/archived/2026-10-11_23-23_SUPERB-trust-green-to-reentrancy-wave-execution.md`](archived/2026-10-11_23-23_SUPERB-trust-green-to-reentrancy-wave-execution.md) | 2026-10-11 | — |
| [`docs/status/archived/2026-10-12_01-18_superb-p015-linter-session-status.md`](archived/2026-10-12_01-18_superb-p015-linter-session-status.md) | 2026-10-12 | — |
| [`docs/status/archived/2026-10-12_02-56_SUPERB-reentrancy-wave-closed-and-m06-bug-trio.md`](archived/2026-10-12_02-56_SUPERB-reentrancy-wave-closed-and-m06-bug-trio.md) | 2026-10-12 | — |
| [`docs/status/archived/2026-10-12_02-56_p015-validation-session-status.md`](archived/2026-10-12_02-56_p015-validation-session-status.md) | 2026-10-12 | — |
| [`docs/status/archived/2026-10-12_07-14_trust-wave-II-shipped-and-bug-backlog-zeroed.md`](archived/2026-10-12_07-14_trust-wave-II-shipped-and-bug-backlog-zeroed.md) | 2026-10-12 | — |
| [`docs/status/archived/2026-10-12_10-34_m14-harness-depth-shipped-and-buildflow-collateral-cleaned.md`](archived/2026-10-12_10-34_m14-harness-depth-shipped-and-buildflow-collateral-cleaned.md) | 2026-10-12 | — |
| [`docs/status/archived/2026-10-12_12-26_n05-fleet-roll-partial-doctor-audit-bug-found.md`](archived/2026-10-12_12-26_n05-fleet-roll-partial-doctor-audit-bug-found.md) | 2026-10-12 | — |
| [`docs/status/archived/2026-10-12_13-30_p015-validation-review-and-gaps-status.md`](archived/2026-10-12_13-30_p015-validation-review-and-gaps-status.md) | 2026-10-12 | — |
| [`docs/status/archived/2026-10-12_13-52_idempotency-deprecation-and-dedupstore-docs-wave-status.md`](archived/2026-10-12_13-52_idempotency-deprecation-and-dedupstore-docs-wave-status.md) | 2026-10-12 | — |
| [`docs/status/archived/2026-10-12_17-15_n05-doctor-audit-fix-m20-verify-blockers.md`](archived/2026-10-12_17-15_n05-doctor-audit-fix-m20-verify-blockers.md) | 2026-10-12 | — |
| [`docs/status/archived/2026-10-13_18-14_issue-67-69-fix-wave-status.md`](archived/2026-10-13_18-14_issue-67-69-fix-wave-status.md) | 2026-10-13 | — |
| [`docs/status/archived/2026-10-13_19-26_green-ci-wave-execution-status.md`](archived/2026-10-13_19-26_green-ci-wave-execution-status.md) | 2026-10-13 | — |
| [`docs/status/archived/2026-10-13_22-56_issue-70-eventwithid-bare-id-fix-status.md`](archived/2026-10-13_22-56_issue-70-eventwithid-bare-id-fix-status.md) | 2026-10-13 | — |
| [`docs/status/archived/2026-10-14_00-40_green-ci-wave-m10-m20-ci-un-red-sweep.md`](archived/2026-10-14_00-40_green-ci-wave-m10-m20-ci-un-red-sweep.md) | 2026-10-14 | — |
| [`docs/status/archived/2026-10-14_05-30_green-ci-wave-ci-un-red-completion.md`](archived/2026-10-14_05-30_green-ci-wave-ci-un-red-completion.md) | 2026-10-14 | — |
| [`docs/status/archived/2026-10-14_17-19_ci-un-red-turso-race-catchup-doublefold-config-restore.md`](archived/2026-10-14_17-19_ci-un-red-turso-race-catchup-doublefold-config-restore.md) | 2026-10-14 | — |
| [`docs/status/archived/2026-10-14_22-05_referee-round2-tree-guard-four-reds-coverage-allowance.md`](archived/2026-10-14_22-05_referee-round2-tree-guard-four-reds-coverage-allowance.md) | 2026-10-14 | — |
| [`docs/status/archived/2026-10-15_03-01_query-http-binding-doc-session.md`](archived/2026-10-15_03-01_query-http-binding-doc-session.md) | 2026-10-15 | — |
| [`docs/status/archived/2026-10-15_18-02_eventcatalog-v5-licence-impact-analysis.md`](archived/2026-10-15_18-02_eventcatalog-v5-licence-impact-analysis.md) | 2026-10-15 | — |
| [`docs/status/archived/2026-10-15_20-18_eventcatalog-v5-bump-execution-session.md`](archived/2026-10-15_20-18_eventcatalog-v5-bump-execution-session.md) | 2026-10-15 | — |
| [`docs/status/archived/2026-10-16_03-55_v4-to-v5-migration-critical-path-and-m23-delivery-mark-fork.md`](archived/2026-10-16_03-55_v4-to-v5-migration-critical-path-and-m23-delivery-mark-fork.md) | 2026-10-16 | — |
| [`docs/status/archived/2026-10-16_17-24_codec-ghost-module-misdiagnosis-and-resolution.md`](archived/2026-10-16_17-24_codec-ghost-module-misdiagnosis-and-resolution.md) | 2026-10-16 | — |
| [`docs/status/archived/2026-10-17_08-23_testfixtures-research-session-review.md`](archived/2026-10-17_08-23_testfixtures-research-session-review.md) | 2026-10-17 | — |
| [`docs/status/archived/2026-10-17_08-56_architecture-reference-html-export-session.md`](archived/2026-10-17_08-56_architecture-reference-html-export-session.md) | 2026-10-17 | — |
| [`docs/status/archived/2026-10-17_08-56_delete-at-zero-first-application-and-self-review.md`](archived/2026-10-17_08-56_delete-at-zero-first-application-and-self-review.md) | 2026-10-17 | — |
| [`docs/status/archived/2026-10-17_19-36_streammarker-display-form-db-leak-fix-status.md`](archived/2026-10-17_19-36_streammarker-display-form-db-leak-fix-status.md) | 2026-10-17 | — |
| [`docs/status/archived/2026-10-17_19-57_streammarker-fix-round2-self-review-and-solutions-status.md`](archived/2026-10-17_19-57_streammarker-fix-round2-self-review-and-solutions-status.md) | 2026-10-17 | — |
| [`docs/status/archived/2026-10-17_20-13_bdd-harness-depth-specs-execution-and-self-review.md`](archived/2026-10-17_20-13_bdd-harness-depth-specs-execution-and-self-review.md) | 2026-10-17 | — |
| [`docs/status/archived/2026-10-17_23-15_architecture-reference-completion-session.md`](archived/2026-10-17_23-15_architecture-reference-completion-session.md) | 2026-10-17 | — |
| [`docs/status/archived/2026-10-17_23-16_bdd-followups-plan-v2-execution-status.md`](archived/2026-10-17_23-16_bdd-followups-plan-v2-execution-status.md) | 2026-10-17 | — |
| [`docs/status/archived/2026-10-17_23-18_disk-truth-plan-execution-wave-I-delivery-status.md`](archived/2026-10-17_23-18_disk-truth-plan-execution-wave-I-delivery-status.md) | 2026-10-17 | — |
| [`docs/status/archived/2026-10-17_23-39_bdd-followups-closeout-brutal-self-review.md`](archived/2026-10-17_23-39_bdd-followups-closeout-brutal-self-review.md) | 2026-10-17 | — |
| [`docs/status/archived/2026-10-23_09-28_open-issues-remediation-comprehensive-status.md`](archived/2026-10-23_09-28_open-issues-remediation-comprehensive-status.md) | 2026-10-23 | — |
| [`docs/status/archived/2026-10-23_12-00_open-issues-execution-and-gate-status.md`](archived/2026-10-23_12-00_open-issues-execution-and-gate-status.md) | 2026-10-23 | — |
| [`docs/status/archived/2026-10-23_14-35_open-issues-remediation-final-closeout.md`](archived/2026-10-23_14-35_open-issues-remediation-final-closeout.md) | 2026-10-23 | — |
| [`docs/status/archived/2026-10-23_15-20_v4-to-v5-phase0-execution-and-gate-closure.md`](archived/2026-10-23_15-20_v4-to-v5-phase0-execution-and-gate-closure.md) | 2026-10-23 | — |
| [`docs/status/archived/2026-10-23_16-09_ten-issues-closed-benchkit-race-fix-and-unpushed-delivery.md`](archived/2026-10-23_16-09_ten-issues-closed-benchkit-race-fix-and-unpushed-delivery.md) | 2026-10-23 | — |
| [`docs/status/archived/2026-10-23_17-08_disk-truth-tail-session-status.md`](archived/2026-10-23_17-08_disk-truth-tail-session-status.md) | 2026-10-23 | — |
| [`docs/status/archived/2026-10-23_17-31_architecture-completion-a10-a27-session.md`](archived/2026-10-23_17-31_architecture-completion-a10-a27-session.md) | 2026-10-23 | — |
| [`docs/status/archived/2026-10-24_05-27_metaengine-codec-boundary-repair-status.md`](archived/2026-10-24_05-27_metaengine-codec-boundary-repair-status.md) | 2026-10-24 | — |
| [`docs/status/archived/2026-10-24_07-00_disk-truth-tail-II-wave-cut-status.md`](archived/2026-10-24_07-00_disk-truth-tail-II-wave-cut-status.md) | 2026-10-24 | — |
| [`docs/status/archived/2026-10-24_09-40_disk-truth-tail-II-brutal-status.md`](archived/2026-10-24_09-40_disk-truth-tail-II-brutal-status.md) | 2026-10-24 | — |
| [`docs/status/archived/2026-10-24_12-00_metaengine-codec-boundary-wave4-and-t16-spike-status.md`](archived/2026-10-24_12-00_metaengine-codec-boundary-wave4-and-t16-spike-status.md) | 2026-10-24 | — |
| [`docs/status/archived/2026-10-24_12-44_verification-debt-m-plan-session.md`](archived/2026-10-24_12-44_verification-debt-m-plan-session.md) | 2026-10-24 | — |
| [`docs/status/archived/2026-10-24_16-56_extraction-candidates-session-status.md`](archived/2026-10-24_16-56_extraction-candidates-session-status.md) | 2026-10-24 | — |
| [`docs/status/archived/2026-10-24_17-40_benchkit-extraction-analysis-session-status.md`](archived/2026-10-24_17-40_benchkit-extraction-analysis-session-status.md) | 2026-10-24 | — |
| [`docs/status/archived/2026-10-24_18-42_io-directive-execution-status.md`](archived/2026-10-24_18-42_io-directive-execution-status.md) | 2026-10-24 | — |
| [`docs/status/archived/2026-10-24_19-29_consumer-fleet-audit-follow-up-status.md`](archived/2026-10-24_19-29_consumer-fleet-audit-follow-up-status.md) | 2026-10-24 | — |
| [`docs/status/archived/2026-10-25_03-14_architecture-review-session.md`](archived/2026-10-25_03-14_architecture-review-session.md) | 2026-10-25 | — |
| [`docs/status/archived/2026-10-25_03-14_p0-execution-red-gates-closed-wave1-shipped.md`](archived/2026-10-25_03-14_p0-execution-red-gates-closed-wave1-shipped.md) | 2026-10-25 | — |
| [`docs/status/archived/2026-10-25_08-52_issues-84-85-eventtest-setters-scenario-versioned-decide.md`](archived/2026-10-25_08-52_issues-84-85-eventtest-setters-scenario-versioned-decide.md) | 2026-10-25 | — |
| [`docs/status/archived/2026-10-25_10-46_value-integrity-wave1-byte-pins-and-parallel-lsm-closure-status.md`](archived/2026-10-25_10-46_value-integrity-wave1-byte-pins-and-parallel-lsm-closure-status.md) | 2026-10-25 | — |
| [`docs/status/archived/2026-10-25_14-06_value-integrity-wave1-completion-proofs-gates-and-foreign-compile-repair.md`](archived/2026-10-25_14-06_value-integrity-wave1-completion-proofs-gates-and-foreign-compile-repair.md) | 2026-10-25 | — |
| [`docs/status/archived/2026-10-25_15-12_trust-repair-and-receipt-wave-execution.md`](archived/2026-10-25_15-12_trust-repair-and-receipt-wave-execution.md) | 2026-10-25 | — |
| [`docs/status/archived/2026-10-25_15-13_engine-coverage-and-vector-substrate-research-status.md`](archived/2026-10-25_15-13_engine-coverage-and-vector-substrate-research-status.md) | 2026-10-25 | — |
| [`docs/status/archived/2026-10-25_15-14_memgraph-engine-evaluation-status.md`](archived/2026-10-25_15-14_memgraph-engine-evaluation-status.md) | 2026-10-25 | — |
| [`docs/status/archived/2026-10-25_15-24_trust-repair-wave-comprehensive-status.md`](archived/2026-10-25_15-24_trust-repair-wave-comprehensive-status.md) | 2026-10-25 | — |
| [`docs/status/archived/2026-10-25_16-07_turso-ivm-rewrite-source-verified-avg-int-division-fix.md`](archived/2026-10-25_16-07_turso-ivm-rewrite-source-verified-avg-int-division-fix.md) | 2026-10-25 | — |
| [`docs/status/archived/2026-10-26_11-18_catalog-do-auditlog-bridge-analysis.md`](archived/2026-10-26_11-18_catalog-do-auditlog-bridge-analysis.md) | 2026-10-26 | — |
| [`docs/status/archived/2026-10-26_21-41_architecture-review-gates-execution.md`](archived/2026-10-26_21-41_architecture-review-gates-execution.md) | 2026-10-26 | — |
| [`docs/status/archived/2026-10-26_23-23_systemcatalog-derivation-execution-status.md`](archived/2026-10-26_23-23_systemcatalog-derivation-execution-status.md) | 2026-10-26 | — |
| [`docs/status/archived/2026-10-26_23-50_testing-improvements-execution-status.md`](archived/2026-10-26_23-50_testing-improvements-execution-status.md) | 2026-10-26 | — |
| [`docs/status/archived/2026-10-27_00-10_testing-gates-execution-and-forgotten-verifications.md`](archived/2026-10-27_00-10_testing-gates-execution-and-forgotten-verifications.md) | 2026-10-27 | — |
| [`docs/status/archived/2026-10-27_04-20_testing-gates-second-pass-proofs-and-defects.md`](archived/2026-10-27_04-20_testing-gates-second-pass-proofs-and-defects.md) | 2026-10-27 | — |
| [`docs/status/archived/2026-10-27_09-45_testing-improvements-session-retrospective.md`](archived/2026-10-27_09-45_testing-improvements-session-retrospective.md) | 2026-10-27 | — |
| [`docs/status/archived/2026-10-27_19-19_erraudit-full-repo-sweep-and-max-audit-status.md`](archived/2026-10-27_19-19_erraudit-full-repo-sweep-and-max-audit-status.md) | 2026-10-27 | — |
| [`docs/status/archived/2026-11-02_11-37_catalog-cue-g0-nogo-and-compat-fallback.md`](archived/2026-11-02_11-37_catalog-cue-g0-nogo-and-compat-fallback.md) | 2026-11-02 | — |
| [`docs/status/archived/2026-11-02_14-47_catalog-compat-cue-reverification-and-lint-debt.md`](archived/2026-11-02_14-47_catalog-compat-cue-reverification-and-lint-debt.md) | 2026-11-02 | — |

## POOL (1)

| Report | Date | Open boxes |
| --- | --- | --- |
| [`2026-10-23_17-20_metaengine-go-codec-tier-minus-one-gap-status.md boxes=13`](2026-10-23_17-20_metaengine-go-codec-tier-minus-one-gap-status.md boxes=13) | 2026-10-23 | 13 unchecked TODO boxes (harvest candidates) |

## What part 2 does

1. Prepend to each of the 141 archive-verdict reports a dated bulk-pass banner.
2. `git mv` each into `docs/status/archived/`.
3. Land the `digest 2026-11` row in `docs/status/README.md`.
