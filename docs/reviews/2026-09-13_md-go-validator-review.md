# Audit: `md-go-validator` over the full repo

> **Date:** 2026-09-13
> **Kind:** Point-in-time tooling audit (no code changes)
> **Command:** `md-go-validator . -v` (system-installed binary, exit 1 = failures found)
> **Repo state:** `master` @ 37ea7a7cf (auto-commit daemon active; working-tree changes in
> `scripts/batch-release.sh` + 3 module READMEs are pre-existing and unrelated)
> **Raw output:** `/tmp/mdgov-full.txt` (verbose), `/tmp/mdgov.json` (machine-readable, this session)

---

## Verdict

**Zero true positives. Not a single failing block is code that was intended to compile.**

Every one of the 172 failures is one of: intentional pseudo-code (ellipsis placeholders,
API sketches, untyped-parameter interface drawings), a deliberately-partial excerpt of a
real file, a struct/case fragment shown out of context, or a fence-tagged-as-`go` block
whose content is actually JSON / `go.mod` / `go.work`. The docs are fine; the validator is
correctly strict; the remedy is annotations and fence tags, not rewrites.

## Headline numbers

| Metric                          | Value                          |
| ------------------------------- | ------------------------------ |
| Markdown files                  | 1,829                          |
| Go blocks validated             | 1,632                          |
| Valid                           | 1,449 (88.8%)                  |
| Auto-skipped (tool)             | 11                             |
| **Errors**                      | **172 (10.5%)**                |
| Files with errors               | 100                            |
| `// skip-validate` uses in repo | 0 (convention not yet adopted) |

All 172 errors are `go` syntax errors. No other language class reported failures.

## Failure taxonomy

| Class                                                 | Count | Typical shape                                                                                                                             |
| ----------------------------------------------------- | ----- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Ellipsis placeholder `...` as a real operand/argument | 40    | `grpc.NewClient(addr, ...)`, `Engines: ...`                                                                                               |
| Illustrative fragment / API sketch                    | 101   | case bodies, bare struct fields, signature sketches (`LoadToVersion(ctx, ..., maxVersion)`), suppression-comment demos, arrow annotations |
| Mixed package-level decls + statements                | 20    | `import (...)` + statements + funcs in one "usage" block (README/COOKBOOK recipe style)                                                   |
| Truncated excerpt (EOF / unterminated raw string)     | 11    | one-line signature quotes, cut-off DQL fragment                                                                                           |

Notable members of the fragment class that deserve special handling:

- **Wrong fence language (content is valid, just not Go) — 5 spots:**
  JSON shown in a ``go fence (`docs/feedback/archived/2026-07-05_cross-consumer-integration-gaps.md:79`);
  `go.mod` in``go (`docs/MIGRATION_v1.md:51`);
  `go.work` in ```go (`docs/planning/archived/2026-04-23_MULTI_MODULE_MONOREPO_PLAN.md:160`,
  `docs/planning/archived/2026-04-30_SAMBER_RO_PROJECTION_INTEGRATION.md:146,433`).
- **Special tokens:** Unicode ellipsis `…` as an ILLEGAL Go token
  (`docs/V5-MIGRATION-GUIDE.md:50`); `←` annotation arrows in go fences
  (`docs/feedback/archived/2026-08-02_cqrs-htmx_cqrs-lint-feedback-round-2.md:167`).
- **Forward-looking sketch the validator is right to reject today:** methods cannot have
  their own type parameters in current Go — `func (s *Store) Get[I, V any](...)` in
  [`docs/adr/0081-metaengine-runtime-casts.md:59`](adr/0081-metaengine-runtime-casts.md)
  is a Go-1.27-era sketch, correctly flagged as not-valid-today.

## Where the failures live

| Bucket                                                                                                                                | Errors  | Files                                                                                                                                   |
| ------------------------------------------------------------------------------------------------------------------------------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| Consumer-facing module docs (READMEs, CONTRIBUTING)                                                                                   | 9       | 7 (`cmd/cqrs-lint/CONTRIBUTING.md` 2, `metaengine/README.md` 2, + 1 each: event, metaengine/COOKBOOK, scenario, system, transport/grpc) |
| Top-level guides (design/, migration/, MIGRATION_v1, V5-MIGRATION-GUIDE, turso-indexing-guidance, projection-tiers, art-dupl report)  | 21      | 7                                                                                                                                       |
| ADRs (0081, 0082, 0122)                                                                                                               | 4       | 3                                                                                                                                       |
| Active planning docs (incl. today's `2026-09-13_11-45_SUPERB-command-side-depth.md`)                                                  | 22      | 13                                                                                                                                      |
| Active research (domain-linter-research, command-event-causality)                                                                     | 8       | 2                                                                                                                                       |
| **Archived / historical** (status/, feedback/, planning/archived, research/archive, quality/archive, brainstorming/, modularization/) | **108** | 68                                                                                                                                      |

Top hotspots:

| File                                                                  | Errors                                                     |
| --------------------------------------------------------------------- | ---------------------------------------------------------- |
| `docs/design/v5-consumer-api.md`                                      | 14 (fluent pseudo-API: `.On(...)`, `...`, fragment chains) |
| `docs/research/domain-linter-research.md`                             | 6                                                          |
| `docs/research/archive/2026-05-20_TIME_TRAVEL_CAPABILITIES_REPORT.md` | 6 (method-signature sketches)                              |
| `docs/feedback/archived/2026-07-17_sec_cqrs-lint-feedback.md`         | 6                                                          |
| `docs/research/archive/2026-05-20_TIME_TRAVEL_INDUSTRY_SURVEY.md`     | 5                                                          |

## Sanity checks performed (is anything actually broken?)

- **All 11 truncated excerpts inspected in place** — each quotes a deliberate fragment of a
  real file (e.g. `docs/status/archived/2026-08-10_14-17_phase2-graphbackend-delete-bus-driver-registry-removal.md:114`
  quotes the single stale `func TestGraphBackend(t *testing.T) { // line 130` line;
  `docs/status/archived/2026-08-08_22-48_dgraph-todo-...md:172` shows only the vulnerable
  prefix of the fixed DQL string). Intentional excerpts, not authoring slips.
- **Consumer-facing README/COOKBOOK blocks** (`metaengine/README.md:237,649`,
  `metaengine/COOKBOOK.md:91`, `system/README.md:324`, `event/README.md:38`,
  `transport/grpc/README.md:50`, `scenario/README.md:27`, `cmd/cqrs-lint/CONTRIBUTING.md:28,144`)
  — all are fragment-style recipes (leading type decl + trailing statements) or ellipsis
  placeholders. None promises a compilable program. Worst cosmetic offender:
  `docs/V5-MIGRATION-GUIDE.md:50` uses a literal `…` character where `...` was meant.
- **No block failed that would pass under the repo's own "copy-paste recipe" standard**
  used by `references/*.md` (those files validate clean).

## Recommendations (Pareto order)

1. **P1 — fix the 5 wrong-fence spots** (cheapest wins; content is valid for its real
   language): JSON → ``json, `go.mod`/`go.work` →``text. After this, zero failures
   remain where the fence lies about the language.
   **APPLIED 2026-09-13:** all 5 fences fixed (plus the sibling go.mod "Before" fence in
   `docs/MIGRATION_v1.md` that only passed by tool accident — both go.mod blocks in that
   file now use ``text for consistency). Files touched:
   `docs/feedback/archived/2026-07-05_cross-consumer-integration-gaps.md` (JSON comment
   moved into prose +``json),
   `docs/MIGRATION_v1.md`, `docs/planning/archived/2026-04-23_MULTI_MODULE_MONOREPO_PLAN.md`,
   `docs/planning/archived/2026-04-30_SAMBER_RO_PROJECTION_INTEGRATION.md` (×2).
   Re-run: **167 errors** (was 172); all 4 files now 0 errors / 31 blocks.
2. **P2 — consumer-facing docs (9 errors, 7 files):** add `// skip-validate` as the first
   line inside the failing block (the tool's own advice) — keeps syntax highlighting while
   silencing intentional pseudo-code.
3. **P3 — active guides / ADRs / planning / research (55 errors):** same skip-validate
   treatment, only where the fragment is load-bearing for the prose.
4. **P4 — archived/historical (108 errors):** do NOT edit history. Use the tool's
   `--baseline` (file:line list, exactly like `scripts/file-size-baseline.txt` and
   `.art-dupl-baseline.json`) so only _new_ failures fail.
5. **Wire it into CI:** no `.md-go-validator.yaml` and no baseline exist yet
   (`md-go-validator --init` creates the former). Add a flake app `check-md-go` running
   `md-go-validator . --baseline scripts/md-go-baseline.txt -q`, mirroring
   `check-file-size` / `check-duplication`.
6. **Adopt the annotation convention explicitly** — the 11 tool-auto-skipped blocks today
   come from built-in heuristics, not repo annotations; making `// skip-validate` the
   documented convention keeps the gate honest going forward.

## Appendix: full failure inventory (172)

Generated from `/tmp/mdgov.json` (format: `file:line (block N) — position: message`).

- `cmd/cqrs-lint/CONTRIBUTING.md:28` (block 1) — 5:19: Go syntax error: expected 1 expression
- `cmd/cqrs-lint/CONTRIBUTING.md:144` (block 5) — 10:7: Go syntax error: expected '(', found isLocalOnly
- `docs/MIGRATION_v1.md:51` (block 3) — 5:10: Go syntax error: expected ';', found github
- `docs/V5-MIGRATION-GUIDE.md:50` (block 1) — 17:43: Go syntax error: expected operand, found 'ILLEGAL'
- `docs/adr/0081-metaengine-runtime-casts.md:59` (block 1) — 4:27: Go syntax error: missing ',' in type argument list
- `docs/adr/0081-metaengine-runtime-casts.md:71` (block 2) — 5:6: Go syntax error: expected '}', found 'type' — snippet mixes package-level declarations with function-body statements
- `docs/adr/0082-metaengine-store-redesign-analysis.md:203` (block 5) — 10:31: Go syntax error: expected operand, found '...'
- `docs/adr/0122-withclock-injectable-time.md:38` (block 2) — 6:50: Go syntax error: expected operand, found '...'
- `docs/art-dupl-improvement-report.md:70` (block 4) — 5:52: Go syntax error: expected type, found ')'
- `docs/brainstorming/archive/2026-06-21_07-21_API-SURFACE-REDUCTION-IDEAS.md:188` (block 5) — 5:12: Go syntax error: expected '{', found Algorithmer
- `docs/feedback/archived/2026-07-05_cross-consumer-integration-gaps.md:79` (block 2) — 5:11: Go syntax error: illegal label declaration
- `docs/feedback/archived/2026-07-10_DiscordSync_leverage_review.md:197` (block 6) — 4:60: Go syntax error: expected operand, found '...'
- `docs/feedback/archived/2026-07-10_DiscordSync_leverage_review.md:273` (block 7) — 7:2: Go syntax error: expected ')', found 'EOF'
- `docs/feedback/archived/2026-07-12_file-and-image-renamer-adoption-blockers.md:76` (block 2) — 14:2: Go syntax error: expected ';', found 'EOF'
- `docs/feedback/archived/2026-07-12_DiscordSync_view-store-as-projection-target.md:61` (block 2) — 5:2: Go syntax error: expected '}', found 'case'
- `docs/feedback/archived/2026-07-17_Cyberdom_cqrs-lint-feedback.md:77` (block 2) — 5:15: Go syntax error: expected ';', found int
- `docs/feedback/archived/2026-07-17_Cyberdom_cqrs-lint-feedback.md:214` (block 3) — 10:10: Go syntax error: expected ';', found string
- `docs/feedback/archived/2026-07-17_Cyberdom_cqrs-lint-feedback.md:231` (block 4) — 5:11: Go syntax error: illegal label declaration
- `docs/feedback/archived/2026-07-17_SwettySwipper_cqrs-lint-feedback.md:406` (block 6) — 5:7: Go syntax error: expected '(', found RegisterHandlers
- `docs/feedback/archived/2026-07-17_bank-sync_encryption-key-management-standardization.md:55` (block 2) — 7:16: Go syntax error: expected ';', found string
- `docs/feedback/archived/2026-07-17_deployment-profiles-design-proposal.md:209` (block 4) — 6:2: Go syntax error: expected '}', found 'case'
- `docs/feedback/archived/2026-07-17_bank-sync_cqrs-lint-feedback.md:47` (block 1) — 8:42: Go syntax error: expected operand, found '...'
- `docs/feedback/archived/2026-07-17_bank-sync_cqrs-lint-feedback.md:302` (block 10) — 4:7: Go syntax error: expected '(', found unwrapSelector
- `docs/feedback/archived/2026-07-17_bank-sync_cqrs-lint-feedback.md:387` (block 11) — 5:7: Go syntax error: expected '(', found mustCommand
- `docs/feedback/archived/2026-07-17_bank-sync_cqrs-lint-feedback.md:429` (block 13) — 5:12: Go syntax error: expected 1 expression
- `docs/design/v5-consumer-api.md:70` (block 1) — 8:36: Go syntax error: expected type, found ')' — snippet mixes package-level declarations with function-body statements
- `docs/design/v5-consumer-api.md:332` (block 6) — 5:37: Go syntax error: expected ';', found '...'
- `docs/design/v5-consumer-api.md:354` (block 7) — 5:40: Go syntax error: expected ';', found '...'
- `docs/design/v5-consumer-api.md:367` (block 8) — 4:2: Go syntax error: expected 1 expression
- `docs/design/v5-consumer-api.md:409` (block 10) — 4:2: Go syntax error: expected statement, found '.'
- `docs/design/v5-consumer-api.md:476` (block 12) — 4:2: Go syntax error: expected 1 expression
- `docs/design/v5-consumer-api.md:514` (block 14) — 4:2: Go syntax error: expected 1 expression
- `docs/design/v5-consumer-api.md:585` (block 16) — 4:2: Go syntax error: expected 1 expression
- `docs/design/v5-consumer-api.md:648` (block 18) — 4:2: Go syntax error: expected 1 expression
- `docs/design/v5-consumer-api.md:708` (block 21) — 5:18: Go syntax error: expected operand, found '...'
- `docs/design/v5-consumer-api.md:754` (block 23) — 7:7: Go syntax error: expected '(', found Evolve — snippet mixes package-level declarations with function-body statements
- `docs/design/v5-consumer-api.md:771` (block 24) — 8:6: Go syntax error: expected statement, found '.'
- `docs/design/v5-consumer-api.md:891` (block 29) — 5:7: Go syntax error: expected '(', found buildProjections
- `docs/design/v5-consumer-api.md:1122` (block 31) — 8:1: Go syntax error: expected selector or type assertion, found '}'
- `docs/feedback/archived/2026-07-18_DiscordSync_order-by-and-index-gaps.md:47` (block 2) — 5:1: Go syntax error: expected operand, found 'case'
- `docs/feedback/archived/2026-07-20_browser-history_cqrs-lint-feedback.md:263` (block 7) — 4:70: Go syntax error: expected operand, found '...'
- `docs/feedback/archived/2026-08-02_Standup-Killer_cqrs-lint-feedback.md:241` (block 6) — 7:2: Go syntax error: expected ';', found 'EOF'
- `docs/feedback/archived/2026-08-02_cqrs-htmx_cqrs-lint-feedback-round-2.md:154` (block 2) — 7:2: Go syntax error: expected ';', found 'EOF'
- `docs/feedback/archived/2026-08-02_cqrs-htmx_cqrs-lint-feedback-round-2.md:167` (block 3) — 6:32: Go syntax error: expected '}', found 'ILLEGAL'
- `docs/feedback/archived/2026-07-17_sec_cqrs-lint-feedback.md:117` (block 3) — 4:2: Go syntax error: expected '}', found 'case'
- `docs/feedback/archived/2026-07-17_sec_cqrs-lint-feedback.md:139` (block 4) — 5:16: Go syntax error: expected ';', found float64
- `docs/feedback/archived/2026-07-17_sec_cqrs-lint-feedback.md:201` (block 7) — 4:7: Go syntax error: expected '(', found MustNewTestEvent
- `docs/feedback/archived/2026-07-17_sec_cqrs-lint-feedback.md:225` (block 9) — 4:10: Go syntax error: expected 1 expression
- `docs/feedback/archived/2026-07-17_sec_cqrs-lint-feedback.md:287` (block 13) — 10:2: Go syntax error: expected ';', found 'EOF'
- `docs/feedback/archived/2026-07-17_sec_cqrs-lint-feedback.md:459` (block 18) — 4:10: Go syntax error: expected ';', found string
- `docs/feedback/archived/2026-08-02_bank-sync_cqrs-lint-improvement-proposals.md:43` (block 1) — 5:2: Go syntax error: expected 1 expression
- `docs/feedback/archived/2026-08-02_bank-sync_cqrs-lint-improvement-proposals.md:57` (block 2) — 5:7: Go syntax error: expected '(', found CommandCausalityEnricher
- `docs/feedback/archived/2026-08-02_bank-sync_cqrs-lint-improvement-proposals.md:68` (block 3) — 12:1: Go syntax error: expected selector or type assertion, found '}'
- `docs/feedback/archived/2026-08-02_crush-daily_cqrs-lint-feedback.md:136` (block 4) — 5:2: Go syntax error: expected 1 expression
- `docs/feedback/archived/2026-08-02_crush-daily_cqrs-lint-feedback.md:201` (block 5) — 5:7: Go syntax error: expected '(', found FoldDailyReport
- `docs/feedback/archived/2026-08-02_crush-daily_cqrs-lint-feedback.md:210` (block 6) — 5:36: Go syntax error: missing ',' in argument list
- `docs/feedback/archived/2026-08-08_DiscordSync_read-write-census-and-metaengine-feedback.md:105` (block 1) — 11:56: Go syntax error: expected operand, found '...'
- `docs/feedback/archived/2026-08-05_KeyHolderAI_cqrs-lint-feedback.md:52` (block 1) — 4:2: Go syntax error: expected 1 expression
- `docs/feedback/archived/2026-08-04_cqrs-htmx_cqrs-lint-feedback-round2.md:218` (block 6) — 5:16: Go syntax error: expected 1 expression
- `docs/feedback/archived/2026-08-21_bank-sync_otel-v4-adoption.md:42` (block 2) — 5:46: Go syntax error: expected operand, found '...'
- `docs/feedback/archived/2026-08-21_bank-sync_go-retry-extraction.md:26` (block 1) — 11:81: Go syntax error: expected operand, found '...'
- `docs/feedback/reviewed/archived/2026-07-23_analytics-rollup-support-review.md:96` (block 2) — 4:16: Go syntax error: missing ',' in argument list
- `docs/feedback/archived/2026-08-13_file-renamer_drain-live-toctou-race.md:87` (block 3) — 4:35: Go syntax error: missing ',' in argument list
- `docs/feedback/reviewed/archived/2026-08-02_browser-history_cqrs-lint-feedback-round-2.md:38` (block 1) — 7:2: Go syntax error: expected ';', found 'EOF'
- `docs/feedback/reviewed/archived/2026-08-02_browser-history_cqrs-lint-feedback-round-2.md:47` (block 2) — 7:2: Go syntax error: expected ';', found 'EOF'
- `docs/feedback/reviewed/archived/2026-08-02_browser-history_cqrs-lint-feedback-round-2.md:250` (block 9) — 5:2: Go syntax error: expected 1 expression
- `docs/feedback/reviewed/archived/2026-08-02_browser-history_cqrs-lint-feedback-round-2.md:291` (block 12) — 7:1: Go syntax error: expected operand, found '}'
- `docs/feedback/reviewed/archived/2026-07-23_analytics-rollup-support.md:198` (block 6) — 13:84: Go syntax error: expected operand, found '...'
- `docs/feedback/reviewed/archived/2026-07-23_analytics-rollup-support.md:219` (block 7) — 11:20: Go syntax error: expected operand, found '...'
- `docs/modularization/PROJECT_GROUPS_IMPROVEMENTS.md:15` (block 1) — 5:2: Go syntax error: expected '}', found 'case'
- `docs/modularization/PROJECT_GROUPS_IMPROVEMENTS.md:23` (block 2) — 5:2: Go syntax error: expected '}', found 'case'
- `docs/migration/tombstone-to-domain-events.md:29` (block 1) — 17:1: Go syntax error: expected declaration, found status
- `docs/planning/2026-09-13_11-45_SUPERB-command-side-depth.md:44` (block 1) — 5:2: Go syntax error: expected statement, found 'package'
- `docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md:239` (block 2) — 10:41: Go syntax error: missing ',' in argument list
- `docs/planning/SAGA_DESIGN.md:46` (block 1) — 9:31: Go syntax error: expected operand, found ']' — snippet mixes package-level declarations with function-body statements
- `docs/planning/QUERY_HANDLER_GENERICS.md:172` (block 6) — 5:7: Go syntax error: expected '(', found handleGetUser — snippet mixes package-level declarations with function-body statements
- `docs/planning/archived/2026-04-23_MULTI_MODULE_MONOREPO_PLAN.md:160` (block 1) — 4:9: Go syntax error: expression in go must be function call
- `docs/planning/archived/2026-05-01_EXECUTION_PLAN.md:264` (block 9) — 4:15: Go syntax error: expected 1 expression
- `docs/planning/archived/2026-04-30_SAMBER_RO_PROJECTION_INTEGRATION.md:146` (block 3) — 6:6: Go syntax error: expected operand, found '.'
- `docs/planning/archived/2026-04-30_SAMBER_RO_PROJECTION_INTEGRATION.md:433` (block 15) — 6:6: Go syntax error: expected operand, found '.'
- `docs/planning/archived/2026-05-21_16-02_REFLECTION_AND_EXECUTION_PLAN.md:166` (block 1) — 5:31: Go syntax error: expected ';', found bool
- `docs/planning/archived/2026-05-19_19-25_CATALOG_ZERO_COST_API_AND_CORE_QUALITY.md:27` (block 1) — 6:29: Go syntax error: expected operand, found '...'
- `docs/planning/archived/2026-05-16_DO_MORE_WITH_LESS.md:220` (block 7) — 7:21: Go syntax error: expected type, found ')' — snippet mixes package-level declarations with function-body statements
- `docs/planning/archived/2026-05-21_LIBRARY_DESIGN_AUDIT_AND_CONSUMER_PAIN.md:261` (block 6) — 7:1: Go syntax error: expected declaration, found repo
- `docs/planning/archived/2026-05-21_LIBRARY_DESIGN_AUDIT_AND_CONSUMER_PAIN.md:524` (block 15) — 5:7: Go syntax error: expected '(', found NewEvent
- `docs/planning/archived/2026-07-06_ARCHITECTURE_LAYERS_RECONSIDERED.md:92` (block 1) — 5:7: Go syntax error: expected '(', found NewConflict
- `docs/planning/archived/2026-07-06_ARCHITECTURE_LAYERS_RECONSIDERED.md:114` (block 2) — 5:24: Go syntax error: expected selector or type assertion, found 'go'
- `docs/planning/archived/2026-07-23_extraction-analysis.md:72` (block 1) — 5:70: Go syntax error: expected operand, found '...' — snippet mixes package-level declarations with function-body statements
- `docs/planning/archived/2026-07-23_extraction-analysis.md:115` (block 2) — 5:70: Go syntax error: expected operand, found '...' — snippet mixes package-level declarations with function-body statements
- `docs/planning/archived/2026-07-23_extraction-analysis.md:145` (block 3) — 16:32: Go syntax error: expected type, found ')' — snippet mixes package-level declarations with function-body statements
- `docs/planning/archived/2026-07-23_extraction-analysis.md:170` (block 4) — 7:63: Go syntax error: expected type, found ')'
- `docs/planning/archived/2026-08-03_00-51_SUPERB-METAENGINE-REPLICATION-MODEL-CORRECTION.md:75` (block 1) — 14:17: Go syntax error: expected ';', found Replication
- `docs/planning/archived/2026-09-06_cqrs-lint-t23-design-passes.md:31` (block 1) — 5:16: Go syntax error: expected 1 expression
- `docs/planning/keep-apps-off-db-layer.md:98` (block 1) — 15:53: Go syntax error: expected operand, found '...'
- `docs/planning/keep-apps-off-db-layer.md:159` (block 3) — 8:56: Go syntax error: expected operand, found '...'
- `docs/planning/keep-apps-off-db-layer.md:226` (block 7) — 6:45: Go syntax error: expected operand, found '...'
- `docs/planning/event-query-model.md:182` (block 2) — 57:38: Go syntax error: expected operand, found '...' — snippet mixes package-level declarations with function-body statements
- `docs/planning/meta-engine-assumptions-and-query-planning.md:66` (block 1) — 4:2: Go syntax error: expected 1 expression
- `docs/planning/meta-engine-assumptions-and-query-planning.md:89` (block 2) — 4:2: Go syntax error: expected 1 expression
- `docs/planning/meta-engine-assumptions-and-query-planning.md:110` (block 3) — 4:2: Go syntax error: expected 1 expression
- `docs/planning/meta-engine-assumptions-and-query-planning.md:1021` (block 19) — 4:25: Go syntax error: illegal label declaration
- `docs/planning/meta-engine-design.md:570` (block 5) — 4:57: Go syntax error: expected operand, found '...'
- `docs/planning/meta-engine-design.md:842` (block 8) — 5:9: Go syntax error: expected ';', found ':='
- `docs/planning/meta-engine-superb-plan.md:370` (block 2) — 5:45: Go syntax error: expected operand, found '...'
- `docs/planning/meta-engine-project-definition.md:463` (block 7) — 6:41: Go syntax error: expected operand, found '...'
- `docs/planning/meta-engine-project-definition.md:477` (block 8) — 6:42: Go syntax error: expected operand, found '...'
- `docs/planning/meta-engine-project-definition.md:515` (block 10) — 9:2: Go syntax error: expected ';', found 'EOF'
- `docs/planning/nats-transport-design.md:150` (block 1) — 4:2: Go syntax error: expected statement, found 'import' — snippet mixes package-level declarations with function-body statements
- `docs/projection-tiers.md:84` (block 2) — 8:48: Go syntax error: expected operand, found '...'
- `docs/planning/storage-domain-separation.md:158` (block 2) — 8:56: Go syntax error: expected operand, found '...'
- `docs/planning/storage-domain-separation.md:292` (block 5) — 6:45: Go syntax error: expected operand, found '...'
- `docs/quality/archive/2026-06-13_06-43_BDD_REVIEW.md:58` (block 1) — 5:93: Go syntax error: expected operand, found '...'
- `docs/quality/archive/2026-06-16_PHANTOM_TYPE_TRIAGE.md:110` (block 4) — 4:7: Go syntax error: expected '(', found CommandAttrs
- `docs/quality/archive/2026-06-16_BRANCHING_FLOW_REVIEW.md:76` (block 3) — 7:29: Go syntax error: expected operand, found '...'
- `docs/quality/archive/2026-06-16_BRANCHING_FLOW_REVIEW.md:201` (block 7) — 4:31: Go syntax error: expected operand, found '...'
- `docs/quality/archive/2026-06-16_BRANCHING_FLOW_REVIEW.md:262` (block 9) — 6:2: Go syntax error: expected ';', found 'EOF'
- `docs/planning/metaengine-redesign.md:1075` (block 7) — 73:1: Go syntax error: expected declaration, found sys — snippet mixes package-level declarations with function-body statements
- `docs/research/archive/2026-05-20_TIME_TRAVEL_CAPABILITIES_REPORT.md:180` (block 3) — 6:10: Go syntax error: missing ',' in argument list
- `docs/research/archive/2026-05-20_TIME_TRAVEL_CAPABILITIES_REPORT.md:206` (block 4) — 6:10: Go syntax error: missing ',' in argument list
- `docs/research/archive/2026-05-20_TIME_TRAVEL_CAPABILITIES_REPORT.md:228` (block 5) — 5:20: Go syntax error: missing ',' in argument list
- `docs/research/archive/2026-05-20_TIME_TRAVEL_CAPABILITIES_REPORT.md:269` (block 7) — 6:10: Go syntax error: missing ',' in argument list
- `docs/research/archive/2026-05-20_TIME_TRAVEL_CAPABILITIES_REPORT.md:289` (block 8) — 5:18: Go syntax error: expected ';', found id
- `docs/research/archive/2026-05-20_TIME_TRAVEL_CAPABILITIES_REPORT.md:320` (block 10) — 11:22: Go syntax error: missing ',' in argument list — snippet mixes package-level declarations with function-body statements
- `docs/research/archive/2026-05-20_TIME_TRAVEL_INDUSTRY_SURVEY.md:706` (block 4) — 6:10: Go syntax error: missing ',' in argument list
- `docs/research/archive/2026-05-20_TIME_TRAVEL_INDUSTRY_SURVEY.md:743` (block 6) — 5:10: Go syntax error: missing ',' in argument list
- `docs/research/archive/2026-05-20_TIME_TRAVEL_INDUSTRY_SURVEY.md:758` (block 7) — 6:10: Go syntax error: missing ',' in argument list
- `docs/research/archive/2026-05-20_TIME_TRAVEL_INDUSTRY_SURVEY.md:773` (block 8) — 5:7: Go syntax error: expected '(', found replayContext
- `docs/research/archive/2026-05-20_TIME_TRAVEL_INDUSTRY_SURVEY.md:796` (block 9) — 5:10: Go syntax error: missing ',' in argument list
- `docs/research/archive/2026-05-28_SINK_SOURCE_SPLIT_AND_GENERIC_BOUNDARIES.md:134` (block 3) — 23:7: Go syntax error: expected '(', found NewRepositoryWithStore — snippet mixes package-level declarations with function-body statements
- `docs/research/archive/2026-05-28_STREAM_API_V2_PROPOSAL.md:286` (block 5) — 6:35: Go syntax error: missing ',' in argument list
- `docs/research/archive/2026-05-28_STREAM_API_V4_SELF_CRITIQUE.md:17` (block 1) — 6:14: Go syntax error: expected type, found ')' — snippet mixes package-level declarations with function-body statements
- `docs/research/archive/2026-06-11_INTERFACE_CONSOLIDATION_AND_TEST_DEDUP.md:62` (block 3) — 5:41: Go syntax error: expected operand, found ']'
- `docs/research/archive/2026-05-28_STREAM_API_V3_PROPOSAL.md:15` (block 1) — 5:6: Go syntax error: expected statement, found '.'
- `docs/research/command-event-causality.md:29` (block 1) — 4:41: Go syntax error: missing ',' in argument list
- `docs/research/command-event-causality.md:69` (block 2) — 5:19: Go syntax error: expected ']', found any
- `docs/research/archive/2026-05-28_STREAM_API_V4_PROPOSAL.md:102` (block 3) — 25:24: Go syntax error: expected type, found ')' — snippet mixes package-level declarations with function-body statements
- `docs/research/archive/2026-07-11_PARQUET_JOURNAL_DUCKDB_MATERIALIZATIONS.md:235` (block 4) — 9:25: Go syntax error: expected operand, found '...'
- `docs/status/archived/2026-04-29_22-17_SESSION10_ARCHITECTURE_IMPROVEMENTS.md:240` (block 3) — 4:27: Go syntax error: expected operand, found '...'
- `docs/status/archived/2026-05-03_07-05_SESSION_46_COMPREHENSIVE_STATUS.md:111` (block 1) — 6:7: Go syntax error: expected '(', found EveryNEvents
- `docs/status/archived/2026-05-07_08-32_SESSION_59_COMPREHENSIVE_STATUS.md:177` (block 1) — 4:16: Go syntax error: expected ';', found string
- `docs/status/archived/2026-05-19_19-57_SESSION_70_CATALOG_ZERO_COST_API.md:39` (block 1) — 5:33: Go syntax error: expected operand, found '...'
- `docs/research/domain-linter-research.md:394` (block 6) — 7:35: Go syntax error: expected ';', found string — snippet mixes package-level declarations with function-body statements
- `docs/research/domain-linter-research.md:943` (block 22) — 5:1: Go syntax error: expected operand, found 'default'
- `docs/research/domain-linter-research.md:1070` (block 27) — 5:27: Go syntax error: expected operand, found 'type'
- `docs/research/domain-linter-research.md:1223` (block 31) — 5:2: Go syntax error: expected '}', found 'case'
- `docs/research/domain-linter-research.md:2048` (block 41) — 5:14: Go syntax error: expected 1 expression
- `docs/research/domain-linter-research.md:2574` (block 55) — 11:1: Go syntax error: expected declaration, found registry
- `docs/status/archived/2026-05-28_06-24_SESSION_112C_EXECUTION_PLAN_STATUS.md:190` (block 1) — 5:65: Go syntax error: expected type, found ')'
- `docs/status/archived/2026-06-11_23-09_FEATURES-TODO-AUDIT-STATUS.md:107` (block 1) — 5:8: Go syntax error: expected 1 expression
- `docs/status/archived/2026-06-17_15-13_AI_ONBOARDING_DOCUMENTATION_OVERHAUL.md:208` (block 1) — 4:7: Go syntax error: expected '(', found AddSpanEvent
- `docs/status/archived/2026-06-17_15-13_AI_ONBOARDING_DOCUMENTATION_OVERHAUL.md:214` (block 2) — 4:11: Go syntax error: expected ';', found "github.com/larsartmann/go-cqrs-lite/otel/v2"
- `docs/status/archived/2026-07-16_20-47_cqrs-lint-dead-rules-duplicates-test-coverage.md:273` (block 1) — 5:42: Go syntax error: missing parameter name
- `docs/status/archived/2026-07-23_23-47_readme-creation-brutal-self-review.md:118` (block 3) — 4:66: Go syntax error: expected operand, found '...'
- `docs/status/archived/2026-07-23_23-47_readme-creation-brutal-self-review.md:126` (block 4) — 4:77: Go syntax error: expected operand, found '...'
- `docs/status/archived/2026-07-25_04-08_PARETO-EXECUTION-COMPLETION-STATUS.md:100` (block 1) — 8:19: Go syntax error: expected ';', found time — snippet mixes package-level declarations with function-body statements
- `docs/status/archived/2026-07-27_10-40_FIXUP-SESSION-VERIFICATION-HARDENING.md:66` (block 1) — 6:1: Go syntax error: expected operand, found '}'
- `docs/status/archived/2026-07-31_18-45_metaengine-first-class-integration-execution.md:147` (block 1) — 6:20: Go syntax error: expected operand, found '...'
- `docs/status/archived/2026-08-01_20-45_benchkit-metaengine-overhaul-completion.md:175` (block 1) — 10:2: Go syntax error: expected ')', found 'EOF'
- `docs/status/archived/2026-08-04_10-00_metaengine-redesign-fix-audit.md:203` (block 1) — 7:17: Go syntax error: expected operand, found '...'
- `docs/status/archived/2026-08-07_22-12_system-p1-hardening-scream-store-serialization-taskmanager-migration.md:114` (block 1) — 6:82: Go syntax error: expected operand, found '...'
- `docs/status/archived/2026-08-08_22-48_dgraph-todo-execution-calibration-security-adts.md:172` (block 1) — 4:7: Go syntax error: raw string literal not terminated
- `docs/status/archived/2026-08-10_14-17_phase2-graphbackend-delete-bus-driver-registry-removal.md:114` (block 1) — 4:7: Go syntax error: expected '(', found TestGraphBackend
- `docs/status/archived/2026-08-11_07-07_adr-0117-command-lifecycle.md:136` (block 1) — 4:17: Go syntax error: expected ';', found time
- `docs/turso-indexing-guidance.md:83` (block 2) — 4:2: Go syntax error: expected statement, found 'import' — snippet mixes package-level declarations with function-body statements
- `docs/turso-indexing-guidance.md:121` (block 3) — 11:7: Go syntax error: expected '(', found tursoSyncHealthChecker
- `event/README.md:38` (block 2) — 28:23: Go syntax error: expected type, found ')' — snippet mixes package-level declarations with function-body statements
- `metaengine/COOKBOOK.md:91` (block 4) — 20:1: Go syntax error: expected declaration, found adapter — snippet mixes package-level declarations with function-body statements
- `scenario/README.md:27` (block 2) — 5:77: Go syntax error: expected operand, found '...'
- `metaengine/README.md:237` (block 11) — 7:10: Go syntax error: expected ')', found metaengine
- `metaengine/README.md:649` (block 29) — 7:44: Go syntax error: missing ',' in argument list — snippet mixes package-level declarations with function-body statements
- `transport/grpc/README.md:50` (block 3) — 9:34: Go syntax error: expected operand, found '...'
- `system/README.md:324` (block 7) — 6:34: Go syntax error: missing ',' in argument list — snippet mixes package-level declarations with function-body statements
