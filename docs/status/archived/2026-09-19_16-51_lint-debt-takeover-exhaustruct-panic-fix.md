# Status Report: lint-debt takeover — exhaustruct panic root-caused and killed, queue/conformance clean, claiming in flight

**Date:** 2026-09-19 16:51 · **Session:** continuation of the substrate-tail execution, took over the lint finish line from the quiet 15:34 session

> **RESOLVED-BY-ROUTING (2026-09-19 docs-health 8th pass):** struck items above = verified shipped (CHANGELOG 2026-09-19 ADR-0142/ADR-0143 entries; TODO_LIST `[x]` rows). Open remainder tracked in TODO_LIST "Metaengine Universal Storage Substrate": T18b load-sweep + benchmark re-baseline (quiet-window gated), T19–T21 (v5-gated), tag waves, claim-metrics parity owner decision. ARCHIVED.
**Trigger:** owner authorization "keep going until everything works". Context: prior reports `2026-09-19_15-11` (substrate tail self-review), `2026-09-19_15-34` (ADR-0143/jsonv2-sweep session).

## a) Fully done (verified this session)

1. **Repo state re-verified before touching anything** — the 15:34 session (lint cleanup + ADR-0143 + jsonv2 sweep) went quiet ~20 min prior; its partial work was uncommitted but compilable (all 9 dirty modules built). Its final report at 15:34 handed off an enumerated lint-debt list (f/1–f/2) as its own top next task — I took it over per the owner's blanket authorization. False alarm avoided: `metaengine/memory_reset.go` untracked + `memory_engine.go` modified are BOTH uncommitted (working tree and HEAD each internally consistent; the daemon later absorbed both).
2. **Real lint baseline established: 47 findings + 6 PANICKING modules** — `nix run .#lint` showed findings in 21 modules, but stack/{sqlite,duckdb,postgres,mysql} and storage/{pebble,bbolt} fail with ZERO printed findings: exhaustruct_v5 **panics** (`makeslice: cap out of range`, upstream `skippedNamed` bug, v5.0.3). The 15:34 report only knew sqlite+mysql; duckdb/postgres/pebble/bbolt were hidden failures.
3. **Panic root-caused to the literal shape**: keyed literals setting PROMOTED fields of embedded structs (all four stack presets' `defaultConfig()` over `sqlopt.DSNConfig`/`PragmaConfig`/`duckdbConfig`; pebble constructors over `storeBase`) AND unkeyed embedded literal elements (bbolt `&CommandStore{storeBase{...}}`). Confirmed empirically: ignore-patterns do NOT prevent the panic (it fires during analysis, before output filtering) — the config route is dead for the panic itself; only rewriting the literals works.
4. **All 6 panic modules killed and verified** (lint clean + build + tests where run: bbolt `ok 0.082s`, duckdb `ok 0.525s`):
   - storage/bbolt: 5 constructor literals → `storeBase: storeBase{db:…, logger:…}` keyed form.
   - storage/pebble: 5 constructor literals → same explicit-embedded form.
   - stack/{sqlite,postgres,mysql}: `defaultConfig()` → explicit `DSNConfig:`/`PragmaConfig:` keyed literals (all fields already had values — zero-behavior-change, strictly more explicit defaults).
   - stack/duckdb: same, PLUS its second embedded `duckdbConfig` (first pass missed it → still panicked → caught by per-module re-verify and fixed).
5. **.golangci.yml** (3 changes, each with dated rationale comments):
   - exhaustruct_v5 ignore-patterns += `metaengine/v4.EngineProfile` (partial-by-design: `ApplyCalibration` + live probing fill cost fields at runtime) and the two unexported `storeBase` scaffolding types (suppresses missing-`shards`/`batch` noise on inner literals).
   - `gochecknoinits` exclusion widened `metaengine/(.*/)?register\.go$` → `(metaengine|queue)/(.*/)?register\.go$` (AGENTS contract 19 pattern, mirrors the metaengine precedent instead of 3× nolint).
   - New exclusion rule (text-scoped) for modernize **embedlit**: it demands the exact unkeyed embedded-literal form that panics exhaustruct — the two analyzers directly contradict each other; we keep the explicit keyed form.
6. **queue/{sqlite,postgres,mysql} engine/register/open/ddl findings fixed** (13 of the 47): embedded fields moved above named fields ×3; err113 → per-module `errDSNRequired` sentinels ×3; sqlite `//nolint:contextcheck // constructor takes no ctx` (pgengine's established pattern); postgres mnd 5000/500 → `postgresNsPerOp`/`postgresNetworkRTT` consts (mirrors mysql's existing const block); mysql open.go `maxOpenConns = 8` named const with rationale + explicit `ownsDB: false` in OpenDB; ddl.go `//nolint:gochecknoglobals // DDL is immutable compile-time package data`.
7. **queue/conformance fully fixed** (5 of the 47): facttx.go `errObservationAbandoned` package sentinel + wrapcheck fix (`fmt.Errorf("append %s: %w", …)`); deps.go varnamelen `a`→first/second/third renames; lifecycle.go wastedassign — dead `c := e.claim(...)` → `_ = e.claim(...)` with a comment (the claim exists for the Running side effect) and `short`/`_ = short` collapsed to `_ = e.enqueue(...)`; journal.go gocyclo-21 `pinWatermark` → `read(consumer, wantSeq, wantExists)` closure, dropping all four triple-condition blocks.

## b) Partially done

1. **claiming module (6 findings) — planned, edit call was interrupted by this report request.** Evidence gathered: `claiming.go:54` documents "DialectDuckDB uses $N placeholders and native TIMESTAMP columns"; `rich.go:314` already has the `case DialectPostgres, DialectDuckDB:` precedent. Pending edits: (i) migrate.go:16 `EnsureLeaseColumn` → `case DialectPostgres, DialectDuckDB:` sharing the IF-NOT-EXISTS branch (DuckDB supports ADD COLUMN IF NOT EXISTS + TIMESTAMPTZ); (ii) rich.go:277 `RenewOwnedStmt` + stmt.go:81 `RenewStmt` → merge DuckDB into the Postgres `$N`cases; (iii) rich.go:234 mnd`len(ids)+3`→ named const; (iv) itoa's bare`10`s →`decimalBase` const.
2. **Lint finish line**: ~22 of 47 findings remain after this session's fixes (claiming 6, metaengine embedded-reorder 9, otelobserver SA1019 1, types.go godoclint 1, adttest maintidx/tparallel/revive 5, claimkit bench sqlclosecheck 1 [my own file], cqrs-lint loader.go gochecknoglobals 1). None re-verified by a full `#lint` run yet — only per-module verifications.

## c) Not started (this session)

- Full `#lint` re-run to zero + module tests for every touched module (queue engines' tests, stack presets' tests, pebble/bbolt tests beyond the two run).
- `#verify` end-to-end; `#verify-ci`; T18b load-sweep + benchmark baseline regen; cqrs-lint on example/taskmanager.
- Repairs: Config.HTTPAddr dead config in example/taskmanager; gotchas-testing.md lessons (WAL-race, journal-survival test hygiene).
- Quick gates re-run: #check-duplication (my literal rewrites + conformance refactor could shift clones), #check-file-size, api-stability golden (no exports changed — expected no-op), doc-check.
- CHANGELOG fold of the lint/panic cleanup; TODO_LIST final state.

## d) Totally fucked up (honest)

1. **Introduced-and-fixed bug**: queue/mysql/register.go — my multiedit dropped the `errors` import while adding a sentinel that uses `errors.New`; caught within one tool call and fixed. A build would have caught it; the lesson is to run the build after EVERY batch, not after "a few" batches.
2. **Silent no-op config attempt**: `modernize.excludes: [embedlit]` was accepted by golangci without error but suppressed nothing; burned a verify cycle before switching to a text-scoped exclusion rule (which works).
3. **Multiedit ambiguity failure**: one of three lifecycle.go edits silently failed ("Applied 2 of 3") because the prologue text appears in multiple scenario tests; caught only because I re-Viewed the file instead of trusting the summary line.
4. **Pipeline masking, twice, in the panic investigation**: (i) my first bisect used v1-style `--disable-all` (unknown flag in v2) — every run errored instantly and my grep showed "no panic"; (ii) all standalone golangci runs were dying at context loading because nixpkgs golangci-lint bundles go 1.26.7 (GOTOOLCHAIN=local) which cannot load go.work 1.27.1 — output looked like "clean". Correct invocation (worked): `nix shell nixpkgs#go_1_27 nixpkgs#golangci-lint nixpkgs#gcc` with the cache-env chain. Every "clean" verdict before that was garbage.
5. First duckdb fix missed the second embedded struct (`duckdbConfig`) — one extra panic/repro cycle that a grep for ALL embedded fields in the module would have prevented.

## e) What we should improve

- **Per-module lint needs a proper entry point**: the flake `#lint` runs all 94 modules (~4 min); standalone golangci needs the exact nix-shell trio above or it silently no-ops on the toolchain. A `#lint-module <mod>` flake app (or a scripts/lint-module.sh wrapper) would have saved this session ~6 wasted tool calls and removes a standing trap for every future session.
- **exhaustruct v5.0.3 upstream bug is now load-bearing**: the repo carries 2 ignore-patterns + 1 exclusion rule + ~19 literal rewrites because of it. Every future embedded-struct literal is a latent panic until upstream fixes `skippedNamed`. Candidate for an upstream issue (minimal repro: struct with embedded struct + keyed literal using promoted fields).
- **The embedlit↔exhaustruct contradiction** (one analyzer demands the exact shape the other crashes on) is documented in .golangci.yml with a dated "revisit when exhaustruct > v5.0.3" note — hold future sessions to that.
- **Verify after EVERY batch**: the import-drop and the multiedit-miss both would have been caught by an immediate `go build ./...` in the touched module; I had been batching verifications.
- **Concurrent-session coordination is now 3-way**: besides me, a benchkit/cqrs-bench variation workstream left uncommitted work, and a NEW goal-shaped-app + skill-refs + dogfooding-review session became active around 16:22 (its edits are in the tree). Never assume "the" concurrent session — check `git status` ownership per file before every gate run.

## f) Next tasks (prioritized)

~~1. Apply the claiming fixes exactly as specified in b/1 (cases + consts) — the edit was mid-flight when interrupted.~~ done 2026-09-19 — 18:05 §a1
~~2. metaengine embedded-field reorder ×9: badgerengine/engine.go:65, bboltengine/engine.go:103, duckdbengine/engine.go:64, mysqlengine/engine.go:64, pebbleengine/engine.go:109, pgengine/engine.go:82, claimkit/claims.go:60, claimkit/claims_test.go:24, claimkit/dedup.go:24.~~ done 2026-09-19 — 18:05 §a2
~~3. metaengine/otelobserver/observer_test.go:202: deprecated `Emit` → `Value.String`.~~ done 2026-09-19 — 18:05 §a3
~~4. metaengine/types.go:130: godoc should start with "IsDegraded".~~ done 2026-09-19 — 18:05 §a4
~~5. metaengine/adttest/claim_conformance.go:34: maintidx nolint (conformance matrix function, deliberate).~~ done 2026-09-19 — 18:05 §a5
~~6. adttest claim_conformance_test.go:44+83: tparallel — add `t.Parallel()` to subtests or nolint with reason (check whether hosts share state first).~~ done 2026-09-19 — fixed
~~7. adttest temporal_conformance.go:100+164: revive unused params `t1`, `vw` → `_`.~~ done 2026-09-19 — fixed
~~8. metaengine/claimkit/bench_test.go:165: sqlclosecheck — proper `defer rows.Close()` (my own file from T18a).~~ done 2026-09-19 — real fix
~~9. cmd/cqrs-lint/pkg/analyzer/loader.go:17: `//nolint:gochecknoglobals` on `loadMu` (mutex guarding packages.Load, mirrors queue/mysql ddl precedent).~~ done 2026-09-19 — 18:05 §a7
~~10. Full `nix run .#lint` → expect 0/94; any stragglers fixed on sight.~~ done 2026-09-19 — 18:05 §a9: 88 modules 0
~~11. Module tests for every touched module: queue/{sqlite,postgres,mysql,conformance}, stack/{sqlite,duckdb,postgres,mysql}, storage/{pebble,bbolt}, claiming, metaengine, metaengine/claimkit.~~ done 2026-09-19 — 18:05 §a10 incl. SOAK_SKIP_BOLT rerun
~~12. `nix run .#check-duplication` — the bbolt/pebble literal rewrites + conformance refactor may have shifted clone groups (annotations may need re-placement).~~ done 2026-09-19 — 18:05
~~13. `nix run .#check-file-size` — journal.go grew ~8 lines (closure); confirm under 350.~~ done 2026-09-19 — gate green
~~14. api-stability golden: `cd cmd/api-stability && GOWORK=off go run . --update` — expect no-op (no export changes); run to confirm.~~ done 2026-09-19 — zero drift
~~15. `#check-error-taxonomy` (new sentinel `errObservationAbandoned` + 3× `errDSNRequired` may need codes? they are plain errors.New — check whether the gate cares).~~ done 2026-09-19 — 525 green
~~16. cqrs-lint `TestExamples_AreV5Clean` on example/taskmanager (first honest lint of the T22 workqueue code; goldens were re-pinned by the 15:34 session — verify, don't trust).~~ done 2026-09-19 — green
17. `nix run .#verify` END-TO-END — lint phase should now pass; race phase re-validates ADR-0143 on the current tree.
~~18. Repairs: wire `Config.HTTPAddr` through `Run()` in example/taskmanager (or delete the field).~~ done 2026-09-19 — 18:05 §a12
~~19. gotchas-testing.md: add the two lessons (two-sqlite-pools WAL-conversion race; run-unique collections/keys for reset tests on shared DBs).~~ done 2026-09-19 — 18:05 §a12
20. `nix run .#verify-ci` per-module matrix.
21. T18b: `nix run .#load-sweep` + `./scripts/benchmark-regression.sh --save benchmarks/benchmark-baseline.txt` — ONLY on a quiet window (load was **70–85** at 16:51; needs <~10).
~~22. CHANGELOG fold: lint-debt + panic-fix section (mention the 6 unlintable modules and the config workarounds with their dated comments).~~ done 2026-09-19 — 18:05 §a13
~~23. TODO_LIST: mark lint/verify items; T18b still open; T19–T21 remain v5-gated.~~ done 2026-09-19 — TODO marks updated
~~24. Doc sync: `.golangci.yml` behavior notes → AGENTS.md internal contracts or gotchas-tooling-build.md (the exhaustruct panic shape + the nix-shell lint invocation trap).~~ done 2026-09-19 — 18:05 §a12
~~25. Consider `#lint-module` flake app (e/1) — small, prevents a recurring trap.~~ **Won't implement — #lint-module already exists (18:05 §d6).**
26. Assess exhaustruct upstream filing (see g/2).
~~27. Re-check the goal-shaped-app session's files before any repo-wide `nix fmt` (never reformat a live session's files).~~ done 2026-09-19 — moot
28. After tree stabilizes: tag-wave assessment for claiming/v4.0.0 + queue family (owner-gated regardless, g/3).

## g) Questions for the owner

1. **Commit strategy (carried over, now urgent)**: the daemon is absorbing this session's ~25-file lint/panic cleanup into `chore: auto-commit` history, interleaved with two other sessions' work. Want authored, per-task commits for the remaining work (I would commit at each boundary — claiming fixes, metaengine batch, gates-green), or keep letting the daemon absorb everything?
2. **Upstream filing for exhaustruct v5.0.3**: I have a minimal repro (embedded struct + keyed promoted-field literal → `makeslice: cap out of range` in `skippedNamed`). File it in your voice (verify-before-filing + github-voice), or track TODO-only until a fixed release? Same question applies to the 15:34 session's go/types+x/tools race.
3. **Tag-wave timing (carried over)**: `claiming/v4.0.0` + queue family — once #verify is green, tag now, or hold for a coordinated release train? (Tags publish on push; also strips the example's new sibling replaces.)

**Bottom line:** the "76-finding lint debt" was actually 47 findings PLUS six modules the linter could not even run on. The panic class is root-caused, dead, and its workarounds are documented in-config; queue/* and queue/conformance are fully fixed and verified per-module; claiming is one interrupted edit away from done. The finish line (lint 0 → verify → ci → T18b on a quiet box) is mechanical from here.
