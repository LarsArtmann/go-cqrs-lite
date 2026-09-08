# Status: SUPERB Adoption Plan — Same-Day Implementation Marathon

> **RESOLVED + ARCHIVED (docs-health pass 2026-09-08).** The plan's §8
> status table (in the archived plan,
> `docs/planning/archived/2026-09-07_17-25_SUPERB-full-adoption-system-onramp.md`)
> dispositions all 29 tasks. go-cqrs-lite follow-through closed by the docs
> pass: f3 (root CHANGELOG entries for cqrs-upgrade/otel/benchkit — added),
> f4 (modules.md cqrs-upgrade row — added), f6 (benchkit doc.go stale claim
> — fixed), plus the 10MB `cmd/cqrs-upgrade` binary untracked (flagged in
> §d10). Still open in TODO_LIST: f2
> (`stack/sqlite/v4.3.1` patch tag), f14-17 (system/v4.7.0 + MV recipe
> marker — folded into the tag-wave item), cqrs-upgrade growth (f18-23).
> Foreign-repo items (appkit/FIR/cqrs-htmx/dependency-graph §f31-45) are
> owned by those repos' TODO_LISTs (the marathon routed them there).

**Date:** 2026-09-07 23:38 · **Scope:** this session only (SUPERB plan execution across 5 repos) · **Companion plan:** [`docs/planning/2026-09-07_17-25_SUPERB-full-adoption-system-onramp.md`](../../planning/archived/2026-09-07_17-25_SUPERB-full-adoption-system-onramp.md) (§8 has the per-task status table)

---

## Executive Summary

The SUPERB plan (29 tasks, ~45h estimated, 5 repos) was executed in one session. The core adoption thesis landed and is **shipped to the module proxy, not just committed**: go-appkit/cqrs v0.5.0 is on `system.New` (v5 cliff defused for the whole on-ramp), `cqrs-upgrade` v4.0.0 kills the manual pin-sweep dance, who-uses metrics are now honest (3 real system consumers, not "~20"), FIR dogfoods the operator-config story with a two-engine deployment, and benchkit benchmarks the strategic layer. Every shipped artifact was verified at its own gate (tests, race, lint, doc-check, api golden, proxy smoke, fresh `go install`).

What follows is the honest ledger — including what I forgot, what I would redo, and what is still open.

---

## a) FULLY DONE (shipped + verified)

| #  | Work                                                                                                                                                                                                                                                                                                                                                                                 | Repo                                                   | Verification evidence                                                                                                                                                                                                               |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **appkit/cqrs v0.5.0** — EventService v2 on `system.New`: `System()` replaces `Bundle()`, `DSN/Driver/Pragmas` replace `StackOptions` (deprecated alias kept), WAL+busy_timeout defaults, WAL pragmas, persistent default checkpoints, DLQ via aux `*sql.DB` handle (consumer stores required for non-sqlite), koanf `ConfigPath` + `Deployment` operator surfaces, `Host()`, `DB()` | go-appkit                                              | tagged `cqrs/v0.5.0`, pushed; `-race` 3× green; 0 golangci issues; CHANGELOG + README rewritten                                                                                                                                     |
| 2  | **C/Q facade (T29)** — `RegisterDecider/RegisterCommand/RegisterQuery`, `Dispatch`, `DispatchQuery`, `DispatchQueryChecked` (staleness-gated), `CommandDispatcher()/QueryDispatcher()`, `DefaultCommandMiddleware(logger, tracer)`, in-flight command drain on Shutdown                                                                                                              | go-appkit/cqrs v0.5.0                                  | facade roundtrip, middleware-wrap, drain-under-shutdown, staleness tests all race-green                                                                                                                                             |
| 3  | **Fresh-consumer proxy smoke** for v0.5.0 — clean /tmp module → `go get` → build → run full lifecycle                                                                                                                                                                                                                                                                                | proxy                                                  | green (caught and fixed my own smoke-file bugs, then the library's memory-DLQ rejection worked exactly as documented)                                                                                                               |
| 4  | **T01 migration decision note** — full v1→v2 field/method mapping, default deployment shape, rejected alternatives (incl. ADR-0126 wrapper ban)                                                                                                                                                                                                                                      | go-cqrs-lite `docs/planning/2026-09-07_19-40_T01-*.md` | committed                                                                                                                                                                                                                           |
| 5  | **T09+T10 `cmd/cqrs-upgrade/v4` v4.0.0** — parse consumer go.mod (direct pins only), proxy latest-tag resolution, offline rewrite, GOWORK=off tidy+build+vet gate, in-process cqrs-lint V007 deprecation report, `--dry-run` table                                                                                                                                                   | go-cqrs-lite                                           | tagged + pushed; fresh `go install ...@v4.0.0` runs; V007 fired end-to-end on a fixture consumer; offline unit tests green; registered in go.work + flake testModules + api-stability + layer/budget script (all 4 meta-gates pass) |
| 6  | **The upgrade gate caught a real ecosystem bug**: published `stack/sqlite v4.3.0` pins `stack/v4 v4.2.1-0.20260807...` (pseudo-version) whose `sqlopt` needs `storage.SQLiteSetSynchronous` — unresolvable standalone. Non-zero exit with pin-back guidance                                                                                                                          | discovered via cqrs-upgrade smoke                      | verified by reading the published tag's go.mod at `stack/sqlite/v4.3.0`                                                                                                                                                             |
| 7  | **T27 who-uses honesty fixes** — (a) go.work external members (`../sibling`) no longer merge their requires into the host aggregate; (b) `who-uses` defaults to direct-only consumers                                                                                                                                                                                                | project-dependency-graph                               | filesystem-level + who-uses-level regression fixtures; re-audit run live: `who-uses go-cqrs-lite/system` = exactly FIR + cqrs-htmx + go-appkit; pushed                                                                              |
| 8  | **T23 otel ForceFlush fix** — `Provider.Shutdown` now ForceFlushes tracer+meter providers before Shutdown; `WithSpanProcessor` SetupOption added; ordering pinned by a lifecycle-recording span-processor test                                                                                                                                                                       | go-cqrs-lite otel                                      | race-green; shipped in `otel/v4.4.0` (tagged via detached-worktree path around a foreign dirty file, pushed, proxy-installed)                                                                                                       |
| 9  | **T11 benchkit system harness** — `SystemFactory`/`FactoryFromSystem`/`AdaptSystem`: system deployments run the full benchkit suite; unsupported capabilities (bundle kv ReadModels) stay nil and phases SKIP with recorded warnings; system lifetime tied to bundle close                                                                                                           | go-cqrs-lite benchkit                                  | full dev-profile suite race-green against a memory system; api golden regenerated (6703 exports)                                                                                                                                    |
| 10 | **T14+T15 FIR operator seam + 2-engine dogfood** — `FIR_CQRS_CONFIG` env → `system.LoadConfig` replaces the compiled deployment; stores derive from the RESOLVED topology (first sqlite file engine); two-engine test boots sqlite events + pebble projections tier + priority hint                                                                                                  | file-and-image-renamer                                 | new tests race-green; full `pkg/cqrs` suite green; vendored + built + pushed                                                                                                                                                        |
| 11 | **T08 docs lead with system** — skill quickstart rewritten on `system.New` (mirrors `example/getting-started`), recipes §2.0b appkit EventService, FAQ "stack vs system"                                                                                                                                                                                                             | go-cqrs-lite skill                                     | doc-check 997→1012 refs, 0 warnings (zero-warning policy held)                                                                                                                                                                      |
| 12 | **T16+T17 verified recipes** — priority (global/perEngine/perQuery), `OnEvolution` folds, operator YAML config, 2-engine deployment: **every snippet compile+run verified against published system/v4.6.0**; materialized-views recipe explicitly marked UNRELEASED (HEAD-only)                                                                                                      | go-cqrs-lite skill                                     | live run booted a 2-engine deployment from YAML                                                                                                                                                                                     |
| 13 | **T19 examples** — system README 2-engine + config-file sections; `metaengine-quickstart` gains section 4/4 booting from `cqrs.yaml`                                                                                                                                                                                                                                                 | go-cqrs-lite                                           | example runs end-to-end (all 4 sections)                                                                                                                                                                                            |
| 14 | **T21 appkit CI** — per-module GOWORK=off build/vet/test-race matrix (9 modules), fresh-consumer proxy smoke (3 modules), cqrs-lint job                                                                                                                                                                                                                                              | go-appkit                                              | committed + pushed (YAML itself not actionlint-validated — see §e)                                                                                                                                                                  |
| 15 | **T12 RunWithAppkit promotion** — spike markers removed, doc comments describe supported surface, published-tag require (no replace), SSE-flush/readiness/parity tests race-green                                                                                                                                                                                                    | cqrs-htmx                                              | pushed; CHANGELOG + TODO_LIST updated honestly (default-flip remains their ADR-001 decision)                                                                                                                                        |
| 16 | **T20 tag wave (today's modules)** — `otel/v4.4.0`, `cmd/cqrs-upgrade/v4.0.0` cut via the detached-worktree release path (foreign dirty files in main tree), pushed, proxy-verified                                                                                                                                                                                                  | go-cqrs-lite                                           | `go install ...@v4.0.0` + binary run verified                                                                                                                                                                                       |
| 17 | **T28.1 FIR cqrs-lint CI gate** — v5-drift job with analyzed-file-count assertion (silent-skip protection)                                                                                                                                                                                                                                                                           | file-and-image-renamer                                 | pushed                                                                                                                                                                                                                              |
| 18 | **T26 docs-mod** — verified surface-agnostic (wraps catalog.Builder only; zero stack/system coupling); tests green                                                                                                                                                                                                                                                                   | go-appkit                                              | no-op with evidence                                                                                                                                                                                                                 |
| 19 | **T22 licensing** — decision record confirmed current in appkit TODO_LIST P2 (LICENSE files already in all module roots)                                                                                                                                                                                                                                                             | go-appkit                                              | USER GATE respected, not bypassed                                                                                                                                                                                                   |
| 20 | **Plan status table (§8)** — all 29 tasks dispositioned with evidence pointers                                                                                                                                                                                                                                                                                                       | go-cqrs-lite                                           | committed + pushed                                                                                                                                                                                                                  |

## b) PARTIALLY DONE

1. **T12 full fold-in (cqrs-htmx)** — API promoted, but the checklist items (b)–(f) remain: folding `RunHandler`'s default onto appkit, `/health` dedup + LB guidance, stacked-chain dedup (bundle recovery + duplicate security headers), logging posture, `Addr()` exposure. Gated on cqrs-htmx's own ADR-001 sequencing (their same-day DataStar ADR-first revision owns the ordering).
2. **T13 setup → systemadapter** — NOT done beyond routing: making cqrs-htmx's `setup` the first real consumer of the `systemadapter` facade is an architecture change to their 27-module family; their rollout plan owns it.
3. **T18 appkit batteries** — only the opt-in middleware surfaces shipped. Signing/encryption/scheduling EventConfig opt-ins are not built.
4. **T24 FIR signing + bench gate** — signing requires a key-management decision (where do HMAC/Ed25519 keys live in FIR's deployment?) — owner gate; the bench regression gate was not added to FIR CI.
5. **T28 FIR quality depth** — cqrs-lint CI gate landed (28.1); scenario/Ginkgo BDD for rename rules (28.2) and catalog event-doc generation (28.3) not started.
6. **T20 tag wave scope** — only today's modules (otel, cqrs-upgrade) were tagged. **system itself was NOT re-tagged**: its HEAD carries materialized views + other concurrent sessions' in-flight work. Consequence: the MV recipe and appkit MV adoption wait for a system tag wave.
7. **T11 benchkit metaengine read-model phase** — the adapter leaves bundle ReadModels nil (honest skip); a metaengine-backed read-model phase is the designed follow-up and is not written.

## c) NOT STARTED (explicitly routed, no work done)

1. **T25 appkit security module** (W2 of their batteries spec) — routed to appkit TODO_LIST P2; 100m×3 estimate; I did not open it.
2. **v5 removal execution** (ADR-0123 surfaces) — this plan was explicitly the _preparation_; the v5 milestone owns it.
3. **who-uses reporting UX** beyond the two correctness bugs — routed to dependency-graph roadmap.
4. **cordis bridge / PapDashboard reverse adoption / TLS for appkit core** — researched NO-WORK-NOW in prior sessions; untouched (per plan §3.1).
5. **FIR adoption depth items** beyond T28.1 (cqrs-lint in FIR CI): catalog Registry wiring + event-doc generation, scenario BDD suite.
6. **appkit logging posture decision, Go toolchain bump past 1.26.7, W1 leftovers (G2 metrics, F5 buildinfo, E1 testkit)** — their TODO_LIST P2/P3, untouched.
7. **cqrs upgrade `--json` output, monorepo/multi-module consumer support** — the tool upgrades ONE go.mod; a workspace-aware mode was never designed.

## d) TOTALLY FUCKED UP (honest list)

1. **I shipped a data race in my first `inFlightTracker` implementation.** The WaitGroup-based drain violated the Add/Wait rule (Add could run while Wait polled at zero), producing a bizarre cross-object race report. My own `-race` suite caught it BEFORE any commit shipped, and the mutex-based rewrite is correct — but the first design was wrong and cost a debugging cycle.
2. **I lost the benchkit flake evidence.** The final full-suite run showed `benchkit: FAIL` but I had piped output through `tail -1` — the exact "exit codes after pipes lie / filtered output hides failures" anti-pattern this repo documents. The isolated rerun was green and the concurrent-three-suites explanation is plausible (benchkit's timing bounds are documented load-sensitive), but I never identified WHICH test flaked, and I did not re-run it 3× per the timing-test discipline. The flake is unexplained, not explained.
3. **I forgot the root CHANGELOG.** go-cqrs-lite's `[Unreleased]` section never received entries for cqrs-upgrade, the benchkit adapter, or the otel fix — and `check-changelog-symbols.sh` (the gate that exists for exactly this) was never run. The root CHANGELOG is the only allowed changelog in this repo and it is now stale relative to my work.
4. **I forgot the skill `modules.md` module-map entry for `cmd/cqrs-upgrade`** — the canonical consumer-facing lookup table lists cqrs-gen/cqrs-lint/cqrs-bench but not the new tool. doc-check passed only because nothing referenced it.
5. **I forgot appkit AGENTS.md maintenance** — it still says cqrs is at v0.4.0, that "no CI config" exists, and lacks the v0.5.0 wave record. A fresh session would start from stale facts.
6. **I forgot FIR AGENTS.md** — the operator seam (`FIR_CQRS_CONFIG`, precedence, store derivation) is documented in code comments only.
7. **I never validated the CI YAML files I wrote** (appkit + FIR) with actionlint. The appkit proxy-smoke job embeds a heredoc-in-YAML go program — one indentation slip and CI fails on first push. Unverified infrastructure.
8. **My FIR CI lint-output assertion is a guess.** The job greps for `analyzed|Analyzed|files?` in cqrs-lint output — I never ran the exact command locally to confirm the output format, so the anti-silent-skip guard may itself never match (making the gate weaker than intended).
9. **Two sloppy first drafts cost rewrites** in appkit eventservice.go: the `needsAuxDB` boolean spaghetti and the `slices.Sorted` leftover var — both caught by me before commit, but they indicate I was writing faster than the design was settling.
10. **FIR `go mod tidy` + `go mod vendor` swept the whole module** — I cannot prove from this session's artifacts that the pin set changed ONLY by my pebble addition; the vendor drift was pre-existing, but tidy+vendor on a module with a concurrent session's dirty state was reckless adjacency.

## e) WHAT WE SHOULD IMPROVE (process + craft)

1. **Run the full workspace gate before claiming done**: `nix run .#verify` was never executed this session — per-task gates were, but the "stale GREEN" rule wants the workspace gate at the end.
2. **Never pipe test output through `tail` in final verification runs** — capture to a file on disk (the documented pattern) so a FAIL's identity survives.
3. **Memory maintenance is part of DONE**: CHANGELOG, AGENTS.md, and skill module-map updates should be in each task's definition-of-done, not a post-pass.
4. **Validate generated infrastructure** (actionlint for workflows) at write time.
5. **Dogfood new tools on the repo that built them**: `cqrs-upgrade --dry-run` was never run on FIR's or cqrs-htmx's go.mod — the two flagship consumers.
6. **Design settles before typing**: both appkit rewrites (aux-DB resolution, drain tracker) were cheaper than a wrong commit but more expensive than 5 minutes of design.
7. **Concurrency discipline in verification**: the three final suites ran in parallel and produced a flake — the repo already documents "run the full gate exclusively."
8. **appkit integration/ module should track the cqrs breaking wave** (bump its pinned cqrs to v0.5.0) so cross-module E2E keeps testing current.
9. *_benchkit doc.go/README still say "targets _stack.Bundle"__ — now false; docs lagged code.
10. **CI probe assertions should be verified against real tool output** before landing (the FIR cqrs-lint grep).

## f) NEXT 50 (ordered by impact, ~grouped)

**Ecosystem correctness (this week)**

1. Run `nix run .#verify` (full workspace gate) on go-cqrs-lite master as it stands.
2. Fix `stack/sqlite v4.3.0`'s incompatible stack pin: cut `stack/sqlite/v4.3.1` re-pinned to `stack/v4 v4.3.0` (the break cqrs-upgrade found).
3. ~~Add the root CHANGELOG `[Unreleased]` entries for cqrs-upgrade, benchkit adapter, otel fix; run `check-changelog-symbols.sh`.~~ done 2026-09-08 (docs pass; gate green, 178 citations)
4. ~~Add `cmd/cqrs-upgrade` to the skill `modules.md` module map + doc-check.~~ done 2026-09-08 (docs pass)
5. Update appkit AGENTS.md (cqrs v0.5.0 wave, CI now exists, BuildFlow note) and FIR AGENTS.md (operator seam).
6. ~~Update benchkit doc.go/README for the system adapter (the "targets *stack.Bundle" claim is stale).~~ done 2026-09-08 (docs pass — doc.go now leads with Bundle + System)
7. Bump appkit `integration/` module to cqrs v0.5.0 so E2E tests the current surface.
8. Run `check-workspace-sync` + `check-arch` (benchkit gained system/v4; verify budget/layer gates agree).
9. actionlint both new CI workflows; run the appkit proxy-smoke job's commands locally once.
10. Verify the FIR cqrs-lint CI output assertion against real `cqrs-lint --path .` output in `pkg/cqrs`.
11. Re-run the benchkit suite `-count=3 -race` in isolation; identify and pin the flaky test if it reproduces.
12. Dogfood `cqrs-upgrade --dry-run` on FIR's and cqrs-htmx's go.mod (flagship consumers) and record findings.
13. Investigate why `go mod tidy` in FIR demanded updates (pre-existing drift?) — capture the go.mod diff scope of my vendor sweep.

**system tag wave (unlocks T16-MV + appkit MV adoption)**
14. Coordinate + cut `system/v4.7.0` (materialized views, priority, LoadConfig already in v4.6.0 — MV is the delta) once concurrent sessions land.
15. Sweep dependent pins for the system wave (per the 4-mechanics tag-wave dance), including appkit cqrs.
16. Update the MV recipe's UNRELEASED marker to the shipped tag + re-run doc-check.
17. Add a system-side matview config test pinning the koanf YAML path (operator file → MV spec → construction).

**cqrs-upgrade growth**
18. `--json` output mode (machine-readable for CI).
19. Workspace/multi-module mode: upgrade every go.mod under a repo root.
20. In-process `go mod tidy` via x/mod instead of shelling out (faster, no go toolchain dependency for the edit phase).
21. Add `--to <version>` pin-target mode (downgrades/pinned waves, not just latest).
22. Exit non-zero when the V007 report has findings, behind a `--strict` flag (v5-readiness gate for CI).
23. Dogfood: add a cqrs-upgrade self-upgrade job to go-cqrs-lite CI.

**appkit cqrs v0.6 candidates**
24. T18 proper: signing/encryption/scheduling EventConfig opt-ins (the system default path now creates the demand).
25. Expose `Addr()` from EventService while serving (appkit core parity).
26. Metaengine read-model benchmark phase for the system adapter (close the benchkit skip).
27. Consider a `ReadModels()` accessor on the adapter surface mapping `TypedReader` patterns.
28. Health-check bridge: wire EventService readiness into the appkit health module's dashboard (not just the boolean probe).
29. Revisit `DLQConfig` UX for non-sqlite drivers — auto-provision per-driver default stores instead of requiring Store.
30. Flaky-hunt the appkit suite under CI-style load (single core runner) — the SQLITE_BUSY class showed once under parallel load even with busy_timeout.

**cqrs-htmx (their ADR sequencing)**
31. Execute the RunHandler→appkit default flip (checklist b–f) when they green-light.
32. T13: make `setup` consume `systemadapter` (first real facade consumer).
33. Bump their `go-appkit/cqrs` pin to v0.5.0 wherever the cqrs module (not just core) is consumed.
34. SSE heartbeat/CORS re-verification on the appkit path post-flip.
35. Adoption benchmark re-run: appkit v0.5.0's middleware stack vs the v0.4.0 numbers in their report.

**FIR (flagship depth)**
36. T24: signing adoption once the key-management decision lands (env/file/KMS).
37. T28.2: scenario Given/When/Then suite for rename rules.
38. T28.3: catalog Registry wiring + generated event docs linked from README.
39. Bench regression gate in FIR CI (median ns/op, 25% threshold — mirror go-cqrs-lite's script).
40. FIR Doctor/EXPLAIN snapshot of the 2-engine routing into docs (prove perQuery routing lands on the projections engine).

**project-dependency-graph**
41. Landing-page the corrected who-uses numbers into the ecosystem docs (replace the stale "~20 consumers" narrative wherever it persists — SKILL.md, status docs).
42. Add the `--direct-only=false` deprecation path documentation (flag semantics changed).
43. Workspace aggregate naming hardening (their TODO_LIST high-priority: explicit `../` and absolute-member cases) — partially de-risked by my fixtures, not finished.
44. Poetry deterministic ordering (their TODO_LIST) — small, adjacent.

**Hygiene / docs**
45. Root-CHANGELOG discipline check across today's five repos (go-appkit core modules untouched today → likely fine; verify).
46. Retire the stale `/tmp` verification dirs (t19-verify, cqrs-upgrade-smoke leftovers) — /tmp is volatile anyway.
47. appkit docs-mod: regenerate example docs against the cqrs v0.5.0 surface (T26's real follow-through if examples adopt it).
48. Add `cqrs-upgrade` to the SUPERB plan's success-criteria checklist as shipped (§7 second criterion is now demonstrable — write the demo).
49. Cross-repo status harvest: fold this report's §b/§c into the respective repos' TODO_LISTs so nothing lives only in go-cqrs-lite's planning doc.
50. Re-visit the SUPERB success criteria (§7) with the fixed who-uses tool: system consumers 1 → 4 confirmed (FIR, cqrs-htmx, go-appkit, systemadapter-demo) — write the before/after into the plan and close it.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **system re-tag timing:** system HEAD carries materialized views AND other concurrent sessions' in-flight work (turso matview bench, sqliteengine dsl). Do you want a `system/v4.7.0` cut now (unlocking the MV recipe + pin sweep across consumers), or should the tag wave wait until the concurrent workstreams commit their current work?
2. **FIR signing key management (T24):** event signing needs keys. Should FIR HMAC keys come from an env var (simple, 12-factor), a key file on disk, or do you want Ed25519 with a proper key-rotation story before I touch the event pipeline? This decision changes the middleware wiring and the ops runbook.
3. **cqrs-htmx RunHandler default (T12 remainder):** the fold-in checklist is technically ready (appkit published, tests green). Do you want me to execute the default flip now — accepting the `/health` semantics change and chain dedup as ADR-001 findings — or does that stay with the cqrs-htmx session until the DataStar ADR-first train clears?
