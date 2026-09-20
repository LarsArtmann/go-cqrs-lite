# SUPERB Plan: Publish & Prove — Release the Unpublished Surface + Close the Trust Floor

**Date:** 2026-09-19 22:34 CEST
**Trigger:** owner directive (paste_1): full Pareto breakdown (1%→51%, 4%→64%, 20%→80%, +20%→100%), ALL 120 open TODO_LIST rows planned at 30–100min tasks, top tier micro-broken to ≤12min, execution graph, then commit + push.
**Predecessors:** [`2026-09-19_18-15_tag-wave-release-prep-ci-triage`](../status/archived/2026-09-19_18-15_tag-wave-release-prep-ci-triage-concurrent-session.md) (90-tag/6-batch plan, dry-run validated, 0 cut) · [`2026-09-19_20-08_docs-health-eighth-pass-full-audit`](../status/archived/2026-09-19_20-08_docs-health-eighth-pass-full-audit.md) (docs floor: 815 annotations, 70 files archived, gates re-greened 22:35).
**Scope statement (the honest headline):** the library's code is DONE through the substrate arc (ADR-0141/0142/0143, queue M4, Go 1.27, lint-zero) — but **0 of ~90 pending tags are published** and the composed verification floor has no recorded green since 2026-09-09. This is a LIBRARY: the product IS the published, trusted surface. Publishing + proving IS the Pareto.

## §0 Current-state truth pass (verified 2026-09-19 22:35)

- `go.mod` count 95; working tree clean (daemon-absorbed); `check-doc-links` **0 broken / 789 targets**; `cmd/doc-check` **1,195 refs valid / 49 packages** (3 known ambiguous-alias advisories = open TODO row).
- Lint debt zero (88 modules, 2026-09-19). Module tests green through the 09-19 waves; composed `#verify` exit-code residue at 18:15 unrecorded since.
- Unpublished on master: ADR-0142 substrate + claimkit, queue family (4 modules), ADR-0141 temporal (3 engines), Go 1.27.1 cutover + jsonv2 graduation, ADR-0143 fix, benchkit rigor tail, goal-shaped-app, CI hardening — **all invisible to consumers**.
- Docs floor: TODO_LIST 120 open rows (incl. 8th-pass harvest), all archived reports annotated; the 3 zero-marker archived reports are banner-covered (owner ratification pending).

## §1 The Pareto ladder

| Tier                 | What                                                                                                                                                                                                                                                                                                                                                                            | Why it is THE tier                                                                                                                                                                       | Deliverable                                                            |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| **1% → 51%**         | **Cut + push the ~90-tag wave** (W0)                                                                                                                                                                                                                                                                                                                                            | Every consumer-visible capability since 2026-09-08 rides it; ~15 TODO rows die the moment tags exist; go-taskqueue/PapDashboard/CV/go-graph-rag consumers are blocked on published paths | 90 tags on the proxy, pin-sweep green, 90 GitHub Releases              |
| **4% → 64%**         | **Prove it** (W1): T18b load-sweep + bench-baseline regen under Go 1.27; composed `#verify` + `#verify-ci` GREEN recorded; `verify-docs.sh` e2e; api-stability README-claims test ownership verified                                                                                                                                                                            | A release you cannot prove green is a liability; S03 acceptance is the repo's standing trust floor; perf claims under 1.27 are currently unanchored                                      | S03 record (date/commit/durations), re-pinned `benchmark-baseline.txt` |
| **20% → 80%**        | **Unblock + harden** (W2): CI billing fix + first nightly run; the two meta-scripts (annotation gate, canonical-fact gate); the two go-graph-rag correctness fixes; dogfooding Tier-0 ruling + close-idiom sweeps; quiet-window verify tooling; camelCase pushdown P0 fix; ctx-tx port to pg/mysql/duckdb; README/benchkit/queue polish slices; FP-sweep refresh; goal-app tail | Each kills a standing failure class or a consumer-blocking bug; all are S/M and independent of the wave                                                                                  | ~25 TODO rows closed, 2 new standing gates                             |
| **other 20% → 100%** | **The long tail** (W3): v5 train rulings + deletions staging, ADR-0139 rulings, censuses, ADR-0141 test tails, md-go-validator gate, E-item batches, matview v2, upstream filings, owner-decision bundle                                                                                                                                                                        | Necessary for v5 and hygiene, but zero immediate consumer value; deliberately sequenced last                                                                                             | v5-ready backlog, clean docs census                                    |

## §2 Task tables (ALL 120 open TODO_LIST rows mapped; 30–100min each; sorted by impact/effort/customer-value)

### Wave 0 — PUBLISH (the 1% → 51%)

| ID  | Task (source TODO row(s))                                                                                                                                                                                     | Impact | Effort  | Customer value                  |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------- | ------------------------------- |
| T01 | Quiet-window gate check: `calibration-gate.sh` pass → run composed `nix run .#verify`; triage any red to zero (W4-residual, composed-verify rows)                                                             | 🔥     | 100m    | Trust floor for the wave        |
| T02 | Batch B0: tag `metaengine` + core prereqs (`record`, `dedup` pins) per 18-15 ordering; strip their sibling replaces; proxy probe                                                                              | 🔥     | 100m    | Unblocks every engine tag       |
| T03 | Batches B1–B2: engines + `claiming/v4.0.0` FIRST then `queue`/`queue/{sqlite,postgres,mysql}`; otel `DBSystem` re-tag precondition; replace-strips; `tag-release.sh --audit` per batch                        | 🔥     | 100m ×3 | Substrate + queue live          |
| T04 | Batches B3–B4: `storage`, `snapshot`, `encryption`, `watermill/v4.7.0` (issue-#21 typed causation — go-localsync workaround dies), `scheduling/sqlstore`, `system/v4.8` (coeffect gate + matview)             | 🔥     | 100m ×2 | Core surface current            |
| T05 | Batches B5–B6: `cmd/cqrs-lint/v4.10.2+`, `cmd/cqrs-bench`, `benchkit`, `catalog`, `commandlifecycle/projections` (`CommandsByActor`), reconciliation-wave untagged surfaces, goal-shaped-app first-tag policy | 🔥     | 100m ×2 | Tooling + docs surface          |
| T06 | `--smoke-all` across wave binaries; `scripts/smoke-probes.txt` per-CLI probes (kills smoke-probes TODO half)                                                                                                  | 🔥     | 60m     | Release integrity               |
| T07 | `pin-sweep.sh --check --remote` post-wave + storage/eventstore pin evidence (standing row)                                                                                                                    | 🔥     | 45m     | Coherent consumer pins          |
| T08 | `create-github-releases.sh` for all new tags (only storage/v4.7.1 ever had one)                                                                                                                               | 🔥     | 60m     | Discoverability                 |
| T09 | Post-wave: consolidate ~49 consumer indirect-dep refs; CV consumer bump (operator-gated); consumer propagation handoff note                                                                                   | 🔥     | 100m    | Consumers actually upgrade      |
| T10 | `check-retracts-shipped.sh` + clean-dir `go list -m @latest` acceptance (release hygiene pair)                                                                                                                | Med    | 60m     | No inert retracts               |
| T11 | `tag-release.sh --audit --baseline` known-violations mode (24 historical) + CI leg                                                                                                                            | Med    | 90m     | Audit gates NEW violations only |
| T12 | Ratify iroh P99 50→150ms judgment call (owner) + dead-path example-module decisions (owner)                                                                                                                   | Med    | 30m     | Closes 2 BLOCKED rows           |

### Wave 1 — PROVE (the 4% → 64%)

| ID  | Task                                                                                                                  | Impact | Effort | Customer value                 |
| --- | --------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------------------------ |
| T13 | **T18b**: `#load-sweep` over timing paths (Latency/Timer/Deadline under soakers) in a quiet window                    | 🔥     | 100m   | Perf truth under 1.27          |
| T14 | **T18b**: `benchmark-regression.sh --save` re-pin with titled provenance header (incl. new claimkit + sqlite entries) | 🔥     | 60m    | Anchored baseline              |
| T15 | `#verify-ci` GOWORK=off matrix green record + `verify-docs.sh` first e2e run (S03 acceptance record into TODO)        | 🔥     | 100m   | The trust floor, recorded      |
| T16 | Verify `cmd/api-stability` incl. `readme_claims_test.go` against current README (foreign change, unowned) + TestEvery | 🔥     | 45m    | No surprise red at next golden |
| T17 | SearchQuery count=5 quiet re-run + dgraph constants re-anchor (calibration campaign (c)+(d))                          | Med    | 100m   | Honest cost model              |

### Wave 2 — UNBLOCK + HARDEN (the 20% → 80%)

| ID  | Task                                                                                                                                                             | Impact | Effort                   | Customer value                     |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------------------------ | ---------------------------------- |
| T18 | Fix GitHub Actions billing (user action) → re-run self-lint CI leg + first nightly-gates run (arms ~10 rows)                                                     | 🔥     | 30m + wait               | CI trustworthy again               |
| T19 | `scripts/check-doc-annotations.sh` (marker-OR-banner gate, live-report strike check) + wire into nightly                                                         | 🔥     | 90m                      | Kills recurring hand-rolled audits |
| T20 | Canonical-fact gate: go.mod count / recipes count / module-map rows derived from repo (3rd consumer of the pattern)                                              | 🔥     | 90m                      | Ends number-rot class              |
| T21 | go-graph-rag #3: fail-closed `EventAdapter.Save` racy fallback (+ test)                                                                                          | 🔥     | 60m                      | Correctness for 3rd-party engines  |
| T22 | go-graph-rag #5: experimental stamps in engine/module `doc.go`s (mechanical sweep)                                                                               | Med    | 60m                      | pkg.go.dev honesty                 |
| T23 | Dogfooding Tier-0 close-helper ruling (owner/ADR) → sweep queue/postgres, storage/pebble, scheduling/sqlstore, projectionhost dlq, stack/run_projections, kv/cmd | 🔥     | 30m ruling + 100m sweeps | One close idiom repo-wide          |
| T24 | Extract scan/paginate helpers (dogfooding finding 4)                                                                                                             | Med    | 100m                     | DRY hot paths                      |
| T25 | Retry-idiom reconciliation audit (middleware/retry vs go-retry, projectionhost, replicator, dgraph)                                                              | Med    | 100m                     | Consistent resilience              |
| T26 | `wait-for-quiet.sh` + `#verify` parallelism cap + golangci cache mount (3-session ask)                                                                           | 🔥     | 90m                      | Verify stops burning attempts      |
| T27 | **camelCase planned-table pushdown fix** (P0 bug: Filterable camelCase fields silently NULL; CRM workaround deletion; pushdown round-trip parity test)           | 🔥     | 100m                     | Silent-wrong-results class dies    |
| T28 | Port ctx-scoped tx (marker-carrying ctx) to pgengine, mysqlengine, duckdbengine + isolation tests each (same `activeTx` leak class)                              | 🔥     | 100m ×3                  | Dirty-read class dies              |
| T29 | Queue M4 polish slice: README MySQL quickstart + conformance doc.go 3-engine list + PG `-race -count=2` leg                                                      | Med    | 60m                      | Queue docs honest                  |
| T30 | Queue M4: MySQL deadlock backoff+jitter, pool options, MYSQL_TEST_DSN nix fold, testcontainer                                                                    | Med    | 100m                     | MySQL production-ready             |
| T31 | cqrs-lint FP-sweep harness: stderr surfacing + corrected 12-repo re-run + crush-daily outlier                                                                    | Med    | 100m                     | Honest FP rates                    |
| T32 | Benchkit polish slice 1: render Min, `--strict` NOISY fail, list-phases mapping, Load1 env row                                                                   | Med    | 90m                      | Output honesty                     |
| T33 | Benchkit slice 2: CSV variation columns + sweep CoV column + per-repeat progress                                                                                 | Med    | 90m                      | Variation visible                  |
| T34 | Benchkit slice 3: RunSuite testing.B variant, metric-name constants, stale-baseline re-pin protocol, recipes/FAQ docs                                            | Med    | 100m                     | SDK completeness                   |
| T35 | README tail slice 1: doc-check repoRoot regression test + gotcha note + CHANGELOG Fixed entry                                                                    | Med    | 60m                      | Gate self-trust                    |
| T36 | README tail slice 2: READMEs into doc-check gate + `check-readme-links.sh` + deprecated-symbol grep gate                                                         | Med    | 100m                     | README drift gated                 |
| T37 | README tail slice 3: deep-read the six big READMEs (catalog, graph, stack, storage/view, watermill, otel, prometheus) + quick-start drift guards                 | Med    | 100m ×2                  | Docs truth                         |
| T38 | goal-shaped-app tail: README fence compile-gating, AGENTS module-procedure extension, real postgres e2e leg, ns caveat, cqrs-lint probe, AsyncAPI demo           | Med    | 100m ×2                  | The Goal demo airtight             |
| T39 | Turso grouped-view upstream filing (draft ready; user approval) + defect-A onset bisect for the envelope                                                         | Med    | 30m + 100m               | Upstream fix motion                |
| T40 | Turso mechanical grouped-spec guard (`AllowGroupedViews` decision + implement)                                                                                   | Med    | 90m                      | Fail-closed matviews               |
| T41 | Watermill skill tail: NATS JetStream roundtrip leg + `references/advanced.md` + trigger evals + cross-links                                                      | Med    | 100m ×2                  | Skill completeness                 |
| T42 | `scheduling/engine` README + claim-metrics parity decision (owner)                                                                                               | Low    | 60m                      | Substrate docs tail                |
| T43 | Doctor `--- Refused ADTs ---` + ExplainPlan refused-engine diagnostic + audit rule 4 rendering polish                                                            | Med    | 90m                      | Operator visibility                |
| T44 | `.golangci.yml` ownership guard + load-threshold guard + restore-depguard `--self-test` (config-corruption class, final nails)                                   | Med    | 90m                      | Self-heal complete                 |
| T45 | asyncapi-react CSP decision (page-scoped relaxation vs bundle swap; owner) + implementation                                                                      | Med    | 30m ruling + 90m         | Docs UI interactive                |
| T46 | Pre-commit tree-wide gates → staged-scoped variants where cheap (workspace-build stays tree-wide)                                                                | Low    | 60m                      | Multi-writer safety                |
| T47 | module-map census (73→95) + FEATURES matrix census + last-verified stamps                                                                                        | Med    | 100m                     | Docs census closed                 |
| T48 | TODO_LIST `[x]` sweep (delete ~15 done rows per header policy)                                                                                                   | Low    | 30m                      | TODO hygiene                       |
| T49 | quic/loopback const split brain + dedup parity test; scan/paginate follow-ups                                                                                    | Low    | 60m                      | Twin-DRY                           |

### Wave 3 — THE OTHER 20% → 100% (v5 train staging + long tail)

| ID  | Task                                                                                                                                                                                                                                                    | Impact                  | Effort           |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------- | ---------------- |
| T50 | Owner Direction Ruling G-T01 (Goal "declare ONLY" post-Infer) + ADR-0141 amendment + AGENTS Goal sentence                                                                                                                                               | 🔥 (gates Goal closure) | 30m ruling + 60m |
| T51 | Zenoh go/no-go ruling (ROADMAP OQ#2; gates 26 parked items)                                                                                                                                                                                             | Med                     | 30m              |
| T52 | 350-line policy ratification (ratchet-as-policy vs split waves vs harness exemptions) → then first split wave if ruled                                                                                                                                  | Med                     | 30m + XL         |
| T53 | ADR-0139 four open questions rulings → implementation wave (DriverConfig.Encryption + KeyProvider)                                                                                                                                                      | Med                     | 30m + 100m       |
| T54 | v5 deletion staging: tombstone API prereqs (listing type-driven status, taskmanager off OnTombstone), sweep §4 remainder (watermill keys, SQL columns, benchkit key, bbolt tags + wire-key table doc)                                                   | Med                     | 100m ×3          |
| T55 | v5 E-item batch 1 (E1 Encoding, E7 RetryConfig, E8 Kind enum, E11 Encode error, E13 phantom param, E15 middleware sigs)                                                                                                                                 | Med                     | 100m ×2          |
| T56 | v5 E-item batch 2 (E3 bbolt errorfamily, E6 Option merge, E9 turso nil panics, E10 ShutdownDependency, E14 ownership)                                                                                                                                   | Med                     | 100m             |
| T57 | T18 snapshot-migration verification tail (live MySQL/DuckDB, mixed-state, failure-path, idempotency, property)                                                                                                                                          | Med                     | 100m ×2          |
| T58 | V5-MIGRATION-GUIDE expansion (before/after per tier, envelope note, operator snippets)                                                                                                                                                                  | Med                     | 100m             |
| T59 | Auto-projection completion (Evolution-fold inheritance audit, tombstone auto-fold, planned-table auto-backfill) — G-T09..12                                                                                                                             | Med                     | 100m ×2          |
| T60 | ADTSet on pg/mysql (meta_set DDL + SetContains parity) or Doctor-loud degraded — G-T13                                                                                                                                                                  | Med                     | 100m             |
| T61 | AggregateOn(fn,column,group) planner-seam one-pager → routing v1 scalar-covered (Turso O(1))                                                                                                                                                            | Med                     | 60m + 100m       |
| T62 | CV design rows: first-class lease/single-writer story (ADR + impl), FilterContains/Prefix, Forever adapters (upstream-gated), benchkit parity gate, tuned-tier benchmark                                                                                | Med                     | 100m ×4          |
| T63 | md-go-validator real gate (baseline + flake app + CI packaging + P2/P3 sweeps + P4 policy)                                                                                                                                                              | Low                     | 100m ×2          |
| T64 | ADR-0141 test tails: property-based version chains, sqlite restart soak, bigtable restart + soak + calibration stubs, MapUpdateAt decision, MaxAge trim, cross-engine fuzz, adapter stamp test, engine README gap notes, Pebble/bbolt scope             | Low                     | 30–100m ×9       |
| T65 | cqrs-lint loose-heuristic batch (import-scope substring gates, B018, A015/A016/A017/A019, F006/F009/F010, V002/V003/V006)                                                                                                                               | Low                     | 100m ×3          |
| T66 | cqrs-upgrade residual holes (NoPins scan, schemaVersion field, E2E fixture)                                                                                                                                                                             | Low                     | 60m              |
| T67 | Metaengine misc: dgraph one-RPC scope benches, CapabilityGaps→Doctor, RenewLease design, conformance mysql-nspawn half, turso/badger contention-retry review, ephemeral passthrough unification, shuffle composites eval (OQ#9), macOS ephemeral-PG leg | Low                     | 30–100m ×8       |
| T68 | session-log boundary + Doctor-JSON + Q3 severity + daemon Q2 + F040 branch protection + ERRAUDIT_PAT (owner bundle — most are 12-min user actions)                                                                                                      | Med                     | 30m each         |
| T69 | Upstream filings: exhaustruct_v5 panic, go/types+x/tools race, turso-go DriverContext/BYOK/DSN (verify-before-filing first)                                                                                                                             | Low                     | 30m each         |
| T70 | T23 skill-maintenance pass (docs/reviews↔brainstorming divergence; read-prior-reports steps)                                                                                                                                                            | Low                     | 60m              |
| T71 | MariaDB VECTOR claim source-verify (ROADMAP UNVERIFIED flag)                                                                                                                                                                                            | Low                     | 30m              |
| T72 | FEATURES Goal-story flip (G-T25) once gates A–D earned                                                                                                                                                                                                  | Med                     | 60m              |
| T73 | Cut v5.0.0 (after W3 rulings + deletions; full verify + guide + examples)                                                                                                                                                                               | 🔥 (terminal)           | 100m ×3          |

**Coverage check:** every open TODO_LIST row maps to T01–T73 (observe-only CatchUp row intentionally unmapped — it is a watch item, not a task; Declined section excluded by definition).

## §3 Micro-breakdown (≤12min each) — Wave 0 + Wave 1 top tier

| #   | Micro-task                                                                                                 | Parent  | Est            |
| --- | ---------------------------------------------------------------------------------------------------------- | ------- | -------------- |
| M01 | `scripts/calibration-gate.sh` load probe; abort-loud if >5                                                 | T01     | 5m             |
| M02 | Kick `nix run .#verify`; capture per-phase log to /tmp/verify-$(date).log                                  | T01     | 10m            |
| M03 | Triage matrix: any red → classify real vs load-transient (template from 15-09)                             | T01     | 12m ×4         |
| M04 | Fix-real loop per red finding (one commit each)                                                            | T01     | 12m ×N         |
| M05 | Record S03: date+commit+durations into TODO_LIST composed-verify row                                       | T01/T15 | 5m             |
| M06 | Pre-tag checklist walk (CONTRIBUTING) against 18-15 manifest; tick boxes                                   | T02     | 12m            |
| M07 | `tag-release.sh claiming` dry-run → real `claiming/v4.0.0`; strip sqlstore replace; tidy; standalone build | T03     | 12m ×3         |
| M08 | Per-batch: `tag-release.sh <modules>` → proxy probe (`go list -m @v4.x`) → smoke                           | T03–T05 | 12m ×6 batches |
| M09 | Per-batch: `--audit` + pin-sweep delta + replace-strip verification grep                                   | T03–T05 | 10m ×6         |
| M10 | `--smoke-all` run; record binary outputs                                                                   | T06     | 12m            |
| M11 | Write `scripts/smoke-probes.txt` (per-CLI exit semantics) + Test-5 skip-path case                          | T06     | 12m            |
| M12 | `create-github-releases.sh <tags>` batched 20/run; verify one Release renders                              | T08     | 12m ×4         |
| M13 | `pin-sweep.sh --check --remote`; storage/eventstore pin evidence into TODO row                             | T07     | 12m            |
| M14 | Soaker setup: build CPU soakers; confirm load-storm absent (load <10)                                      | T13     | 10m            |
| M15 | `nix run .#load-sweep` leg 1 (Latency tests); capture                                                      | T13     | 12m ×3 legs    |
| M16 | `./scripts/benchmark-regression.sh --save` with provenance header; diff vs old baseline; commit            | T14     | 12m            |
| M17 | `#verify-ci` run; poll matrix; record green evidence                                                       | T15     | 12m ×2         |
| M18 | `verify-docs.sh` first e2e; fix tripwire findings if any                                                   | T15     | 12m            |
| M19 | `cd cmd/api-stability && GOWORK=off go test ./...` (readme_claims ownership check); note verdict in TODO   | T16     | 8m             |
| M20 | Regenerate api golden if drift; TestEvery                                                                  | T16     | 12m            |
| M21 | Post-wave CHANGELOG release-note polish per batch (dates ↔ tags)                                           | T03–T05 | 12m ×3         |
| M22 | Consumer propagation handoff note (CV + go-taskqueue + PapDashboard pins)                                  | T09     | 12m            |

## §4 Execution graph

```mermaid
flowchart TD
    subgraph W0["W0 PUBLISH — the 1% → 51%"]
        T01[T01 quiet-window #verify green] --> T02[T02 B0 core prereqs]
        T02 --> T03[T03 B1-B2 engines + claiming + queue family]
        T03 --> T04[T04 B3-B4 storage/watermill/system]
        T04 --> T05[T05 B5-B6 tooling + docs + goal-app]
        T05 --> T06[T06 --smoke-all + probes]
        T06 --> T07[T07 pin-sweep --remote]
        T07 --> T08[T08 GitHub Releases]
        T08 --> T09[T09 consumer propagation]
        T02 --> T10[T10 retracts gate]
        T05 --> T11[T11 audit baseline mode]
    end
    subgraph W1["W1 PROVE — the 4% → 64%"]
        T01 --> T13[T13 load-sweep]
        T13 --> T14[T14 baseline re-pin]
        T01 --> T15[T15 verify-ci + verify-docs + S03]
        T15 --> T16[T16 api-stability ownership check]
        T14 --> T17[T17 calibration campaign]
    end
    subgraph W2["W2 UNBLOCK+HARDEN — 20% → 80%"]
        T18[T18 billing + nightly] -.user.-> W2
        T19[T19 annotation gate] --> T20[T20 canonical-fact gate]
        T21[T21 EventAdapter fail-closed]
        T22[T22 experimental stamps]
        T23[T23 Tier-0 ruling -> sweeps]
        T26[T26 wait-for-quiet + -p cap]
        T27[T27 camelCase pushdown fix]
        T28[T28 ctx-tx port x3]
        T29[T29-T31 queue/lint/benchkit slices]
        T35[T35-T38 README/benchkit/goal-app slices]
        T39[T39-T41 turso/watermill]
    end
    subgraph W3["W3 OTHER 20% → 100%"]
        T50[T50 G-T01 ruling] --> T59[T59 auto-projection completion]
        T51[T51 Zenoh ruling]
        T52[T52 350-line ruling]
        T53[T53 ADR-0139 rulings] --> T54[T54 v5 deletion staging]
        T54 --> T55[T55-T56 E-items] --> T57[T57 migration tail] --> T58[T58 guide] --> T73[T73 cut v5.0.0]
        T72[T72 FEATURES Goal flip] --> T73
    end
    W0 --> W1 --> W2 --> W3
    T09 -.consumers unblocked.-> T62[T62 CV design rows]
    T05 -.tags exist.-> T38[T38 goal-app post-tag items]
```

## §5 Gates + acceptance

- **W0 done =** proxy serves all ~90 tags; `pin-sweep --check --remote` green; `--smoke-all` green; 90 GitHub Releases; zero local `=>` replaces except documented `storage/go.mod` pair (stripped in-wave per plan); `check-retracts-shipped` green.
- **W1 done =** S03 row filled (date/commit/durations); `benchmark-baseline.txt` carries 1.27 + claimkit provenance; api-stability TestEvery green.
- **W2 done =** the two new gate scripts wired into nightly and mutation-tested; go-graph-rag #3/#5 shipped; dogfooding sweeps done or Tier-0-blocked with banner; pushdown parity test green; 3 engine isolation tests green.
- **W3 done =** owner rulings recorded; v5 staging checklist green; v5.0.0 cut procedure queued.
- Every wave ends: `nix fmt` scoped, `cmd/doc-check` zero-warning, `check-changelog-symbols` green, api golden regenerated in-change, CHANGELOG `[Unreleased]` entries per consumer-visible change, blast-radius consumer-module tests run.

## §6 Verschlimmbesserung guardrails

1. **No code refactors ride the release wave** — W0 tags what IS on master; fixes queue behind it.
2. **No golden/baseline re-pin without a quiet-window load reading** (the calibration-gate abort is the rule, not a suggestion).
3. **No edits to files another session owns mid-flight** (readme_claims_test.go class) — verify, don't touch.
4. Tag ordering is load-bearing (otel `DBSystem` before storage; claiming before queue; metaengine pins before engines) — follow CONTRIBUTING pre-tag checklist mechanically.
5. If a batch fails mid-wave: STOP, fix forward on master, re-audit; never force a tag over a red gate; never `git reset`.
6. The plan is not a license to grow scope: a task that exceeds its estimate by >2× gets split or parked, never silently absorbed.

## §7 Embedded owner decisions (12-minute user actions that unblock >20 tasks)

1. **Authorize the wave** (W0) and its green bar: quiet-window `#verify` vs verify-fast+module-green (18-15 proposal).
2. **G-T01 Direction Ruling** — the Goal's "declare ONLY" meaning (T50).
3. **Zenoh go/no-go** (T51) · **350-line ratification** (T52) · **ADR-0139 questions** (T53) · **billing fix** (T18) · **Turso upstream filing approval** (T39) · **pass-scoped annotation-gate ratification** (8th pass §g1).
