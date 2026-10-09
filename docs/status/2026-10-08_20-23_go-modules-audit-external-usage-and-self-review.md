# Go Modules Audit — Status & Self-Review

**Date:** 2026-10-08 20:23 CEST
**Session scope:** Analysis-only session triggered by _"I feel like we are doing something wrong with the way we use go modules??"_ — repo investigation, verdict, analysis of the user-supplied `who-uses` consumer scan (202 modules in `~/projects`), external-usage verification, and self-review. **Zero code changes were made this session.** This file is the session's only durable artifact.

---

## Headline results — every claim verification-graded

| # | Session claim                                                                         | Verdict                    | Evidence                                                                                                                                                                                       |
| - | ------------------------------------------------------------------------------------- | -------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | "Repo is private → the `~/projects` scan IS the entire possible market"               | **WRONG — corrected**      | `gh api repos/LarsArtmann/go-cqrs-lite`: `private:false, visibility:public`, 1 star, 0 forks, 0 watchers                                                                                       |
| 2 | "metaengine/benchkit go-humanize drift; GOWORK=off standalone build broken RIGHT NOW" | **WRONG — false alarm**    | `go-humanize` IS required (`metaengine/go.mod:6`, `benchkit/go.mod:6`); `GOWORK=off go build ./...` green in both; the 18 LSP diagnostics are gopls workspace-mode noise (a documented gotcha) |
| 3 | External consumers?                                                                   | **Verified: zero known**   | pkg.go.dev `event/v4`: **Imported by: 0**; 1 star / 0 forks / 0 watchers; proxy lists fully populated (indexer-driven)                                                                         |
| 4 | Why does pkg.go.dev hide all documentation?                                           | **License: PROPRIETARY**   | `LICENSE` first line "PROPRIETARY LICENSE", added 2026-03-15; pkg.go.dev reports UNKNOWN/redistributable-unchecked at every tagged version → docs hidden **by policy, permanently**            |
| 5 | Root train "." = v4.0.0                                                               | **Phantom tag**            | proxy `@v/list` for the suffix-less root path serves only v0.1.0–v1.7.1; a v4 tag on a `/vN`-less path is proxy-invisible (the repo's own issue-#20 class)                                     |
| 6 | 101 `go.mod` files                                                                    | Current truth              | `find . -name go.mod -not -path './vendor/*' \| wc -l` = 101; AGENTS.md still says 98 (stale)                                                                                                  |
| 7 | 76/101 modules are single-package                                                     | Counted, not tier-caveated | includes legitimately-single-package driver modules (the count is right, the framing was incomplete)                                                                                           |
| 8 | 52 trains consumed, 49 live trains + root with zero consumers (in `~/projects`)       | Parsed from who-uses log   | Appendices A/B                                                                                                                                                                                 |

**Net market picture (corrected):** the repo is _source-available but proprietary_ — publicly fetchable, legally unusable by third parties. Combined with zero known importers, the `~/projects` universe (42 Lars projects + cqrs-htmx/go-appkit/buildflow/go-taskqueue intermediaries) is the complete real consumer population **today**; an external consumer could only exist after relicensing. The consolidation argument from earlier in the session survives the public-repo correction at full strength — for a different reason than originally stated.

---

## Session timeline

1. Loaded `go-modularize` + `go-cqrs-lite` skills; investigated repo state (go.work 99 members, root module = one `doc.go`, `versions.json`, `batch-release.sh`/`tag-release.sh`/`pin-sweep.sh` + ~6 check gates ≈ 1,839 LOC of module-management scripts, gotchas docs).
2. Delivered verdict: boundaries mostly right; **versioning model** (independent semver per train, everything tagged, lockstep waves of 90+ tags) and **granularity** (module-per-package) are the problem. Options: A lockstep versions / B consolidate at v5 / C stop tagging non-consumers.
3. User supplied `who-uses github.com/larsartmann/go-cqrs-lite --dir ~/projects` output (202 modules discovered, 52 trains matched).
4. Parsed consumer data (two buggy parses before a correct one), delivered six data-driven learnings: zero-consumer zoo, inverted import surface (darlings event/id/command/decider/middleware/query vs system=13), measured merge set, stranded retired pins, one-engine reality (sqlite+memory), unused version independence.
5. User challenged repo-privacy premise → verification round: corrected claims 1–2, measured external usage, found the proprietary-license docs block and the phantom root tag.

---

## a) FULLY DONE

- Evidence-based diagnosis of the multi-module model with concrete cost inventory (92/90-tag waves, 66-module pin sweeps, lockstep releases defeating per-train semver, version soup v0–v4 across siblings, graveyard manifest, internal tooling on release trains).
- Full consumer-usage matrix: 52 trains × (total, direct) — Appendix A.
- Zero-consumer list: 49 live trains + root — Appendix B.
- Merge-set identification with **measured** migration cost (trains with direct ≤ 3 consumers: dispatcher 41/1, metadata 41/3, dedup 34/1, commandlifecycle 31/0, projections 31/1, claiming 30/0, kv 33/3, scheduling 33/3, listing 31/4, encryption 26/3, signing 27/3, prometheus 26/5, stack 27/5, pgtestcontainer 26/0, eventtest 26/5).
- Stranded-pin inventory on retired trains: codec v4.4.0 → 29 consumers (12 direct), idempotency → 9 (7 direct), retry → 1, flightrecorder → 1.
- Import-surface inversion finding: ADR-0123 says "system + metaengine only", the fleet imports event(35)/id(36)/command(29)/decider(28)/middleware(23)/query(21)/storage(18) directly; cqrs-htmx + go-appkit are the de-facto composition layer.
- External-usage verification: public repo, proprietary license, Imported-by 0, proxy populated by indexers, phantom root tag.
- Options A/B/C with data-driven v5 shape (≈ 1 core + ~10 driver/tool trains; cqrs-lint the only tool with a cross-repo consumer: buildflow + gomend).

## b) PARTIALLY DONE

- **v5 proposal:** direction and shape given; no ADR, no doc, no go-modularize Phase 1–5 migration matrix (offered, not written).
- **Stranded pins:** counted, not root-caused (which of the 12 direct codec consumers are leaf apps vs cqrs-htmx/go-appkit) and not swept.
- ~~**External usage:** flagship trains checked (root/event/system/metaengine/id); the other 47 consumed trains not individually checked against pkg.go.dev/proxy.~~ done 2026-10-09: all 52 swept, zero non-LarsArtmann importers — `docs/evidence/consumer-usage-2026-10-09.md` (note: id shows 48 public importers — all intra-repo + first-party repos; "Imported-by: 0" was shorthand, not literal for every train).
- ~~**Module-count drift:** 101 vs AGENTS.md's 98 found; not reconciled (canonical-facts gate derives from the same `find` and would also report 101 — the AGENTS.md text is stale, delta = 3 modules since last census).~~ done 2026-10-09: AGENTS.md re-counted (101 go.mod / 97 rowed modules), canonical-facts gate green.
- ~~**gopls noise classification:** false alarm identified and explained; not yet added to the LSP-noise gotcha doc.~~ done 2026-10-09: `docs/agents/gotchas-tooling-build.md` (go-humanize bullet beside the stdversion/tidy entries).
- **cqrs-htmx/go-appkit centrality:** asserted from eyeballing the dependency tree ("nearly all indirection flows via them"); the "via" edges were never mechanically counted.

## c) NOT STARTED

- ADR: v5 module topology + lockstep-versioning decision (blocked on Q1–Q3 below).
- go-modularize Phase 1–5 full per-module scoring (cohesion/coupling/independence/depth/payoff) → migration matrix.
- Fleet sweep: stranded codec/idempotency/retry/flightrecorder pins → external repos.
- Untagging policy + tag-on-first-consumer policy for drivers and internal tooling.
- `cqrs-upgrade` extension as the v4→v5 import-rewrite codemod.
- Nightly `who-uses` consumer gate feeding the FEATURES maturity matrix.
- `versions.json` live/retired split.
- Relicensing decision (or explicit confirmation of proprietary-by-design).

## d) TOTALLY FUCKED UP — honest

1. **Stated "the repo is private" as fact.** It was an inference from the devShell's `GOPRIVATE=github.com/larsartmann/*` family-wide setting (which exists because _other_ repos in the family are private). The documented one-line audit (`gh api repos/... --jq .private`) was available and was skipped. An entire learning ("zero _possible_ consumers, forever") was built on the false premise. User corrected; verified public within the same session.
2. **Invented a live build break.** Trusted 18 gopls diagnostics as truth ("go-humanize is not in your go.mod"), plus a `grep | head`-truncated module list, and declared the metaengine standalone build broken _without ever running the build_. Both go.mods require the dependency (line 6, both modules); `GOWORK=off` builds are green; the imports were committed same-day (auto-commit `24b14f485`, 2026-10-08), which explains the gopls lag. The "live specimen" was noise that flattered the thesis — **confirmation bias, not verification.**
3. **Two buggy parses before a correct one.** First awk emitted one row (state machine bug); second kept stale totals when the regex missed the singular "(1 consumer)" — queue/flightrecorder/retry/storage-turso/pebbleengine/scheduling-engine totals were wrong in intermediate tool output (the user-facing table came from the corrected third parse). Lesson: anchor-check a parser against rows whose truth is already known before trusting any of it.
4. Minor: `cmd 2>&1 | head; echo $?` captured `head`'s exit code, not the command's (empty output was the real signal); the "76 single-package" table row lacked the drivers-are-legit caveat; the 101-vs-98 drift was noticed and silently dropped instead of reported.

**The pattern across 1–2 — stated inference as fact when a one-command verification existed, twice in one session — is the session's core failure.** The remedy is procedural, not personal (see e).

## e) WHAT WE SHOULD IMPROVE (process)

- **Verification bar:** any externally-checkable claim gets its one-liner _before_ it is stated (`gh api`, `GOWORK=off go build`, un-truncated `grep`). No exceptions for claims that fit the narrative — especially those.
- **gopls diagnostics are noise until a GOWORK=off build agrees** (already a documented gotcha; this session is a fresh incident to cite).
- **Anchor-validate ad-hoc parsers** against known rows; prefer `jq` for structured data.
- **Persist analysis artifacts in-repo when the analysis concludes**, not only in chat (this report, incl. appendices, is that artifact).
- **Reconcile canonical numbers on sight** (98→101; the canonical-facts script exists precisely for this number-rot class).
- **Record durable corrections in memory:** "go-cqrs-lite is PUBLIC (verified 2026-10-08) with a PROPRIETARY LICENSE; GOPRIVATE is family-wide and must never be used to infer this repo's privacy."

## f) Next things (prioritized)

**P0 — evidence closure & memory (minutes–hours)**

1. AGENTS.md: record public+proprietary fact and the GOPRIVATE-no-inference rule (memory rule, blocks repeat of failure 1).
2. Sweep all 52 consumed trains through pkg.go.dev/proxy (`Imported by`, `@v/list` vs latest tag) → `docs/evidence/consumer-usage-2026-10-08.md`; close the "47 unchecked trains" gap.
3. Fix AGENTS.md module count 98→101 (or run the canonical-facts update path); identify the 3-module delta.
4. Add this session's gopls false-alarm to the LSP-noise gotcha.
5. Root train: decide phantom-tag disposition (untag-going-forward vs leave; `v4.0.0` is proxy-invisible — note it in `check-versions-manifest.sh` README section or versions.json annotations).

**P1 — decisions (need Q1–Q3)**
6. Relicensing decision: stay proprietary-by-design (then consolidation logic is unconstrained) vs relicense for adoption (then license wave + pkg.go.dev docs visibility + stable surface become P0).
7. ADR: v5 module topology — core merge set (all §b merge-list trains + zero-consumer non-drivers), driver tier, tooling.
8. ADR: lockstep single version for surviving trains (option A) vs per-train semver; wave tooling changes that follow from it.
9. Driver policy: tag-on-first-consumer; untag the 10 unused engines until adopted.
10. Untag/retire internal-only trains: root, `cmd/{api-stability,cqrs-bench,cqrs-gen,cqrs-upgrade,doc-check}`, examples, test-infra, benches (keep tagging only `cmd/cqrs-lint` — buildflow+gomend consume it).
11. Ghost-system verdicts: `graph`, `deriver`, `transport/grpc`, `transport/http`, `otel/otlp`, `idempotency/kvstore`, `queue/mysql`, `storage/{pebble,backuptest}` — integrate (find/winning a consumer) or kill at v5; zero consumers + zero integration = ghost systems by definition.
12. cqrs-htmx + go-appkit: formalize as first-class consumers; lockstep co-release decision.

**P2 — execution (post-decision)**
13. go-modularize Phases 1–5: per-module scoring → migration matrix doc.
14. Core merge waves in dependency order; per wave: api-stability golden regen + check-module-layers/dep-budget re-baseline + GOWORK=off per-module test matrix.
15. Extend `cmd/cqrs-upgrade` into the v4→v5 import-rewrite codemod; dogfood across the fleet.
16. Fleet sweep: codec→go-codec (12 direct consumers), idempotency→go-idempotency (7), retry, flightrecorder.
17. Consolidate test suites: systemtest + system/integration + integration → one untagged internal module.
18. Consolidate benches: benchkit + stack/bench + metaengine/bench → one.
19. versions.json: split live/retired; README manifest marks retired trains.
20. eventtest v0.4.0: align to the v4 family or fold into core at v5.
21. batch-release: auto-generate wave manifest from changed-set + transitive dependents (kills the manual triple list).
22. pin-sweep: run per-module _tests_, not build-only (the 2026-10-06 red-suites lesson — still open in AGENTS.md).

**P3 — hygiene / monitoring**
23. Periodic `who-uses` gate feeding FEATURES maturity matrix (consumer counts per train; zero-consumer engines marked experimental-by-evidence).
24. Per-train proxy-freshness check (latest tag vs proxy `@latest`) — catches indexer lag and phantom classes mechanically.
25. deps.dev/pkg.go.dev watch: alert if any external dependent ever appears (would reopen the versioning calculus).
26. Refresh skill references (`modules.md`, `core.md`) after the v5 decision lands.
27. CHANGELOG: open the v5 planning section.
28. Root module disposition: doc-only placeholder (untagged) vs home of the v5 core.
29. README honesty: "used by" section reflecting the real consumer population (the fleet), or remove claims that imply broader use.
30. If relicensing ever happens: license-note tag wave across live trains to un-hide pkg.go.dev docs.

## g) Questions I cannot answer myself

1. **Is external adoption ever a goal?** The repo is public but PROPRIETARY (LICENSE 2026-03-15; pkg.go.dev permanently hides docs; zero importers; 1 star). If source-available-by-design: consolidation is unconstrained and P1 item 6 resolves "stay proprietary". If adoption is desired: relicensing is the single highest-leverage change, ahead of any module restructuring. I cannot know the intent.
   > **ANSWERED 2026-10-08 (chat, same day): "first only my projects"** — external adoption is not a current goal; proprietary license is intentional; consolidation is unconstrained. Recorded in AGENTS.md ("Consumer scope"). Relicensing wave dropped from the backlog (P1 #6 resolved: stay proprietary; P3 #30 parked behind a future intent change).
2. **v5 migration appetite:** when the darlings' import paths change (event/id/command/decider/middleware/query/storage into a core module), do you want a hard one-shot fleet migration (~40 projects, codemod-driven, v4 dies) or a v4/v5 dual-support transition period? This decides whether option B is cheap or expensive — and I cannot pick your risk tolerance for 40 production projects.
   > **ANSWERED 2026-10-08 (structured question): DUAL-SUPPORT** — v4 keeps tagging fixes, v5 lands alongside, apps migrate opportunistically; intermediaries (cqrs-htmx, go-appkit) migrate first.
3. **cqrs-htmx + go-appkit boundary:** are they in-scope companions of go-cqrs-lite (co-designed, lockstep-released, their imports shape the v5 core) or independent downstream projects to be treated like any external consumer? They are the de-facto composition layer — but their ownership/release coupling to this repo is a business call, not a code fact.
   > **ANSWERED 2026-10-08 (after consumed-surface analysis, structured question): BOTH IN-SCOPE COMPANIONS.** Analysis: go-appkit = thin system wrapper (8 production files centered on system/v4; 24-train require list is mostly transitive surface); cqrs-htmx = full-surface chassis (~30 trains, deep imports in root/adminui/dashboardui/datastar/e2e + own examples; dashboardui/system_bridge.go already builds on system). They co-release in the same waves; v5 core shaped around their surface; codemod targets them first.

---

## Appendix A — 52 consumed trains (from who-uses, corrected parse; format total/direct)

| Train                        | Total | Direct |   | Train                                  | Total | Direct |
| ---------------------------- | ----- | ------ | - | -------------------------------------- | ----- | ------ |
| id                           | 41    | 36     |   | system                                 | 31    | 13     |
| event                        | 41    | 35     |   | commandlifecycle/projections           | 31    | 1      |
| dispatcher                   | 41    | 1      |   | commandlifecycle                       | 31    | 0      |
| metadata                     | 41    | 3      |   | listing                                | 31    | 4      |
| record                       | 41    | 12     |   | claiming                               | 30    | 0      |
| command                      | 40    | 29     |   | codec (retired)                        | 29    | 12     |
| query                        | 39    | 21     |   | storage/memory                         | 29    | 15     |
| otel                         | 38    | 16     |   | scenario                               | 27    | 11     |
| projection                   | 37    | 11     |   | signing                                | 27    | 3      |
| snapshot                     | 37    | 15     |   | stack                                  | 27    | 5      |
| decider                      | 36    | 28     |   | catalog                                | 26    | 10     |
| watermill                    | 35    | 16     |   | encryption                             | 26    | 3      |
| dedup                        | 34    | 1      |   | eventtest (v0.4.0)                     | 26    | 5      |
| kv                           | 33    | 3      |   | prometheus                             | 26    | 5      |
| metaengine                   | 33    | 15     |   | testutil/pgtestcontainer               | 26    | 0      |
| scheduling                   | 33    | 3      |   | idempotency (retired)                  | 9     | 7      |
| metaengine/projectionadapter | 32    | 9      |   | schema                                 | 8     | 6      |
| metaengine/sqliteengine      | 32    | 15     |   | idempotency/sqlstore                   | 4     | 4      |
| projectionhost               | 32    | 11     |   | storage/bbolt                          | 4     | 1      |
| middleware                   | 31    | 23     |   | stack/sqlite                           | 3     | 3      |
| storage                      | 31    | 18     |   | cmd/cqrs-lint                          | 2     | 1      |
|                              |       |        |   | testutil                               | 2     | 1      |
|                              |       |        |   | flightrecorder (retired)               | 1     | 0      |
|                              |       |        |   | metaengine/pebbleengine                | 1     | 1      |
|                              |       |        |   | queue, queue/sqlite, queue/postgres    | 1 ea  | 0      |
|                              |       |        |   | retry (retired)                        | 1     | 0      |
|                              |       |        |   | scheduling/engine, scheduling/sqlstore | 1 ea  | 1      |
|                              |       |        |   | storage/turso                          | 1     | 1      |

## Appendix B — 49 live trains with zero consumers (+ root)

benchkit, cmd/api-stability, cmd/cqrs-bench, cmd/cqrs-gen, cmd/cqrs-upgrade, cmd/doc-check, deriver, example/getting-started, example/goal-shaped-app, example/mesh-demo, example/metaengine-quickstart, example/readme-quickstart, example/scheduler-otel-status, example/taskmanager, graph, idempotency/kvstore, integration, metaengine/badgerengine, metaengine/bboltengine, metaengine/bench, metaengine/bigtableengine, metaengine/dgraphengine, metaengine/duckdbengine, metaengine/graphadapter, metaengine/irohengine, metaengine/irohengine/loopback, metaengine/irohengine/quic, metaengine/mysqlengine, metaengine/otelobserver, metaengine/pgengine, metaengine/tursoengine, otel/otlp, queue/mysql, stack/bbolt, stack/bench, stack/duckdb, stack/memory, stack/metaengine, stack/mysql, stack/pebble, stack/postgres, stack/turso, storage/backuptest, storage/pebble, system/integration, systemtest, testutil/mysqltestcontainer, transport/grpc, transport/http — plus the root module (doc.go only; its "v4.0.0" tag is proxy-invisible).

## Appendix C — brutal self-review answers (compressed)

- **Forgotten?** External verification and license check belonged in the first message; artifacts belonged in the repo at each conclusion.
- **Something stupid we do anyway?** 101 release trains, 1,839 LOC of compensating release/gate scripts, and lockstep waves — for a market of one proprietary fleet. (The user's original instinct was correct.)
- **Better?** Verify-before-claim; anchor parsers; treat gopls as noise; mechanical proof over diagnostics.
- **Did I lie?** Two false claims presented as fact (private repo; broken build). Not lies — unverified inferences — but indistinguishable in effect. Both corrected the same session, with commands cited.
- **Less stupid?** Consolidate (≈1 core + driver tier), lockstep the survivors, untag internals — removes the machinery that generates the failure classes the gotchas document.
- **Ghost systems?** 49 zero-consumer trains; especially graph, deriver, transport/*, otel/otlp, 10 engines. Per the rule: integrate (win a consumer) or kill; capability-without-customer is not a product.
- **Scope creep?** None — the session stayed on the module question; this report is the boundary.
- **Removed something useful?** Nothing (analysis-only session).
- **Split brains?** versions.json tag-truth vs tree-truth; AGENTS.md 98 vs reality 101; codec/idempotency/retry/flightrecorder dual homes (frozen in-repo tags vs external repos); eventtest v3 dead train vs v4.
- **Tests?** None run in anger; the single build verification invalidated my own claim — which is the argument for mechanical proof.

---

_Point-in-time snapshot. Stale by design; annotate, never rewrite (docs-health ANNOTATE mode)._
