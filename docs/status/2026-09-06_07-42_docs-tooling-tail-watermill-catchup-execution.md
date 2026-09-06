# Status Report — Docs/Tooling Tail Execution: Watermill CatchUp Hardening + README Honesty Sweep

**Date:** 2026-09-06 07:42 CEST
**Session scope:** Executed the 8-item "Docs / Tooling tail (harvested 2026-09-06)" block from `TODO_LIST.md` — watermill CatchUp regression tests, catch-up throughput benchmark + watermark documentation, readme-quickstart modernization, metaengine comment rot, README unverified-claims audit, example/docs README audits, `storage.EventSchema` symmetry, `watermill.WithBackend` arity verification.
**Parallel session observed:** another session committed work on `cmd/cqrs-lint` (doctor JSON), `storage/pg_integration_test.go`, `storage/snapshot_migration.go`, `metaengine/duckdbengine`, and a CHANGELOG T18 entry mid-session. Its landmines are flagged in (d); I did not touch its files.

---

## a) FULLY DONE

All items below are committed (auto-commit daemon waves `502222e30`, `b4c3d4c03`, plus the final report commit) and gated.

| # | Item | Evidence | Files / scope |
| - | ---- | -------- | ------------- |
| A1 | **CatchUp Close-while-blocked regression tests** — two tests pin that `Close()` returns promptly and the subscription channel closes (a) while replay is blocked on a full 256-slot output buffer (600 seeded events, no consumer) and (b) while parked in `awaitAck` (checkpoint verified untouched — at-least-once holds). | `GOWORK=off go test -race -count=3` green; full watermill suite green | `watermill/catchup_lifecycle_test.go` (new, 293 lines incl. helper `closePromptly`/`drainUntilClosed`) |
| A2 | **Double-Subscribe regression test** — two `Subscribe` calls on one topic each replay the full journal into their own channel; both close on `Close()`; post-Close `Subscribe` is rejected. Determinism achieved via `readBarrierJournal` (test waits until BOTH subscriptions did their checkpoint-load `ReadFrom` before any ack advances the shared checkpoint). | Same run green (and under `-race`) | `watermill/catchup_lifecycle_test.go` |
| A3 | **Catch-up throughput benchmark** — `BenchmarkCatchUp_ReplayThroughput`: end-to-end serialized pipeline (journal read → eventToMessage → channel hop → ack → checkpoint save). Measured 3.6-6.2 µs/event (~160-280K events/s) at ambient load avg 37-59 on 32 cores. Ack-window pipelining deliberately deferred; the documented remedy trigger is a 10× degradation. | Ran with `-benchtime=3x/5x -count=3`; numbers + load recorded in the benchmark doc comment | `watermill/benchmark_test.go` |
| A4 | **Watermark ULID-skew caveat documented** — new "Watermark ordering assumption" section on the `CatchUpSubscriber` doc comment: cross-process clock skew can mint an event whose ID sorts below the watermark; it is suppressed live and stays missing until the next restart, where replay from the checkpoint recovers it. Staleness window bounded by producer skew; self-healing on restart. | Code doc; builds; doc-check unaffected | `watermill/catchup_subscriber.go` |
| A5 | **readme-quickstart modernized** — deprecated `repo.Execute`/`repo.Load` pair forms (main.go:51,65) migrated to `ExecuteRef`/`LoadRef` with `id.NewStreamRef`; README fence updated to match. | Example test suite green (`ok ... 0.003s`); the two gopls deprecation hints are gone | `example/readme-quickstart/main.go`, `README.md` |
| A6 | **metaengine/dsl.go comment rot fixed** — `LogPlan` comment referenced nonexistent `PlanFromSQLite`; now references `Plan` / `PlanFromMemory`. The real SQLite one-shot remains `sqliteengine.PlanFromDSN` (decision: comment fix, not new API — golden + dep-budget cost not justified by demand). | `go build` green; lint green | `metaengine/dsl.go` |
| A7 | **`storage.EventSchema` symmetry completed via re-export (the "add" option)** — `eventstore.EventSchema()` / `eventstore.SQLiteEventSchema()` added next to their snapshot/checkpoint siblings; aliased in `storage` (`eventstore_aliases.go`); `storage/README.md` DDL section now lists all six; api-stability golden regenerated (+7 lines); CHANGELOG `[Unreleased]` entry added. | storage full suite green; `check-changelog-symbols.sh` verifies all 4 new citations against the golden | `storage/eventstore/event_store.go`, `storage/eventstore_aliases.go`, `storage/README.md`, `docs/api_surface.txt`, `CHANGELOG.md` |
| A8 | **`WithBackend`/`WithCommandBackend` arity verified** — both are 3-arg `(message.Publisher, message.Subscriber, io.Closer)` (`event_bus_options.go:26`, `command_bus_options.go:26`); `watermill/README.md` example + `TestRedisStreamRoundtrip` are correct (`*redis.Client` satisfies `io.Closer`). Two live docs showed a non-compiling 2-arg call — fixed. | Signature read from source; grep sweep of all non-archived `.md` | `docs/DOMAIN_LANGUAGE.md:307`, `docs/architecture-understanding/2026-07-23_book-insights-detailed-answers.md:43` |
| A9 | **Root README claims corrected** — event "3 dependencies" → "9 module deps — only 3 third-party" (verified via `go list -deps`: third-party = fxamacker/cbor, oklog/ulid, x448/float16; 6 first-party larsartmann); coverage claims re-floored to `scripts/check-coverage.sh` baselines (event 90%, decider 96%, id 86%, dispatcher 87%; range 82–97% → 86–96%). Claims now round DOWN from baselines only. | `go list` output; baseline map read from script | `README.md` (lines 17/28/38/212 region) |
| A10 | **"82 modules" exact-phrase sweep** — README already used "80+" (true: 82 go.mod files); the one live hardcoded "82 go.mod files today" (`docs/migration/V5-OUTLINE.md`) made rot-resistant ("80+"). CHANGELOG mentions are dated history — untouched. | grep sweep of live docs | `docs/migration/V5-OUTLINE.md` |
| A11 | **README-claims meta-tests added** — `TestREADMEClaim_EventThirdPartyDeps` (shells `go list` in the event module, asserts 3 third-party + README phrasing), `TestREADMEClaim_ModuleCountFloor` (walks go.mod count ≥ 80 + "80+" claim present), `TestREADMEClaim_CoverageFloors` (parses check-coverage.sh baselines; README claims may round down, never up). | 3/3 green in the full api-stability suite | `cmd/api-stability/readme_claims_test.go` (new) |
| A12 | **`.github/` templates created** — `PULL_REQUEST_TEMPLATE.md` (with the repo's per-task gate checklist), `ISSUE_TEMPLATE/bug_report.md`, `ISSUE_TEMPLATE/feature_request.md`, `ISSUE_TEMPLATE/config.yml` (blank issues disabled, SKILL.md as contact). | Files exist; no validation gate applies | `.github/` |
| A13 | **taskmanager README rewritten (6 stale sections)** — it described the PRE-modernization architecture: claimed `decider.Repository.Execute` (code: `system.Execute` Op), `stack.Materialize` KV read model (code: metaengine Map ADT + `projectionadapter`), `CatchUpSubscriber` ordered delivery (code: `projectionhost`), metadata tombstone (code: `task.deleted` domain event, ADR-0114), `id.AggregateID` (code: `TaskID = id.StreamID`), `sqlite.New(...)` in setup.go (code: `system.EngineConfig{Driver: "sqlite"}` in `DeploymentConfig`), 10 commands (11 `system.RegisterCommand`), 11-file structure (17 files). Feature table, architecture diagram, file structure, design-decisions, deployer-first, and swapping-databases sections now match the code (11 event types ✓, 11 commands ✓ verified by grep). | Every claim re-verified against source before writing | `example/taskmanager/README.md` |
| A14 | **getting-started README title honest** — "in 140 Lines" → "in a Single File" (actual: 251 lines; a hardcoded line count rots, so the claim is now rot-proof). Output claim (`value=10`) verified against `main.go:244`. | `wc -l` + grep | `example/getting-started/README.md` |
| A15 | **docs/README.md counts fixed** — "(68 modules)" → "(80+ modules)"; "109 ADRs" → "130+ ADRs" (actual count: 134); "Minimal 80-line tour" → "Minimal single-file tour"; missing 4th example row added (metaengine-quickstart). planning/status/benchmarks/adr READMEs audited — accurate (snapshot-honesty framing correct, benchmarks dated 2026-06-03 with hardware recorded). | `ls docs/adr/*.md \| wc -l` = 134; doc-check green | `docs/README.md` |
| A16 | **TODO_LIST.md harvested block closed out** — all 8 items marked `[x]` with DONE annotations carrying evidence and file pointers; the one manual remainder (social preview/homepage) explicitly carved out. | Section re-read immediately before edit (shared-ledger protocol) | `TODO_LIST.md` |
| A17 | **Gates run this session** — per-module full test suites: watermill ✓, storage ✓, cmd/api-stability ✓, example/readme-quickstart ✓; lint green on watermill, metaengine, cmd/api-stability, example/readme-quickstart (storage lint: only the PARALLEL session's `snapshot_migration.go` fails — not my file); canonical doc-check: 956 references valid across 42 packages; changelog symbol gate: my 4 new symbols verified (1 remaining fiction is the parallel session's — see D3); `-race -count=3` on the new timing-sensitive tests. | Exit codes captured pipe-free per AGENTS.md protocol | — |
| A18 | **storage `GOWORK=off` go.sum gap fixed** — `modernc.org/libc v1.75.7` was missing its `/go.mod` hash (workspace mode had hidden it); targeted `go mod download` per the toolchain's own suggestion; +1 line, storage standalone builds green again. | `GOWORK=off go test` flipped FAIL → ok | `storage/go.sum` |

---

## b) PARTIALLY DONE

| # | Item | What works | What remains | Blocker | Effort |
| - | ---- | ---------- | ------------ | ------- | ------ |
| B1 | **README unverified-claims item** | Everything CLI-doable is done (claims fixed, meta-tests added, templates created, 82-sweep done) | Social preview image + homepage URL are GitHub **settings-UI** fields — not settable from the CLI. Need either your manual action or a generated asset (HyperFrames render) + your paste | Repo admin UI access / branding decision | S (manual) |
| B2 | **example/ README audit (said "4 files")** | Audited all 3 READMEs that exist (taskmanager, getting-started, readme-quickstart) | The 4th example (`metaengine-quickstart`) has **no README at all** — docs/README.md now links to its directory, but the copy-paste surface is missing its own page | None — authoring work | M |
| B3 | **Final GREEN claim discipline** | Per-task gates all green for this session's diff (tests, lint, doc-check, symbol check, golden) | The session-level `nix run .#verify` (or `#verify-fast`) was NOT run — per-module greens are not the repo gate, and the AGENTS.md "stale GREEN" rule wants verify-fast before any GREEN banner. The report therefore claims per-module green only | None — ~minutes of runtime; must run EXCLUSIVELY (no concurrent heavy jobs) | S |
| B4 | **Benchmark baseline quality** | Measured numbers recorded WITH ambient load (37-59) as the AGENTS.md protocol requires; gate framed as order-of-magnitude | An idle-machine re-run would tighten 3.6-6.2 µs to a real ceiling; machine had 31-33 users all session | Ambient load, not code | S (timing) |

---

## c) NOT STARTED

Planned-adjacent work this session surfaced but did not begin. None of it was in the harvested 8-item scope.

| # | Item | Why not started | Still wanted? |
| - | ---- | --------------- | ------------- |
| C1 | **Catalog tail items** (same TODO_LIST neighborhood): golden-test the flattened eventcatalog exporter output + embedded-flattening fallout check in `cmd/cqrs-gen` + `catalog/eventcatalog`; browser-validate CSP against embedded Scalar/AsyncAPI bundles; automate the EventCatalog render validation | Outside this session's harvest block | Yes — untouched TODO_LIST items |
| C2 | **`awaitAck` lying log line** — on `Close()`, `awaitAck` returns false and `replayPhase` logs `ERROR ... "consumer nacked replay event"` even though the consumer never nacked (it was a Close). Noticed during test writing (visible in test output), deliberately deferred to keep the diff on one variable, then FAILED to record it anywhere. Logged here now. | Scope discipline (correct call), follow-through lapse (no TODO entry until now) | Yes — small truthfulness fix |
| C3 | **Skill-reference propagation of the skew caveat** — the watermark caveat now lives in the Go doc comment; `.agents/skills/go-cqrs-lite/references/advanced.md` §CatchUp likely documents the watermark dedup without the skew caveat. doc-check passes (it checks symbol existence, not contract completeness), so nothing forces this | Not in scope; doc-check gave no signal | Yes — consumers read advanced.md, not doc comments |
| C4 | **FEATURES.md update** for the new public re-exports (`storage.EventSchema`/`SQLiteEventSchema`) — API additions should touch the feature inventory, not just CHANGELOG + golden | Not in my write-reflex; process gap noted in (e) | Yes |
| C5 | **Ack-window pipelining** for CatchUpSubscriber | Deliberately deferred — throughput ceiling is ~160-280K ev/s in-memory; trigger documented (10× degradation) | On hold by design |
| C6 | **Idle benchmark re-run + docs/benchmarks/ entry** | Ambient load never dropped below ~35 this session | Nice-to-have |

---

## d) TOTALLY FUCKED UP

Nothing currently broken in this session's delivered work — but radical honesty requires naming what went wrong, even when caught in-session.

| # | What happened | Severity | Root cause | Mitigation / status |
| - | ------------- | -------- | ---------- | ------------------- |
| D1 | **I wrote a fabricated number into source.** The benchmark comment initially claimed "2026-09-06 baseline (idle machine): ~2-4 µs/event" BEFORE any measurement — a predicted number dressed as a baseline, on a machine that was never idle. Caught during the measurement pass and corrected to the real measured range (3.6-6.2 µs with load recorded). | Was: doc-lie in committed source. Now fixed. | `verify-external-claims` discipline applied to MY OWN future numbers, not just external claims. Writing the comment felt like finishing the task; it was manufacturing evidence. | Fixed in-session; process rule added in (e)-2 |
| D2 | **Trusted a pipe's exit code** — first storage test run printed `STORAGE_EXIT=0` while the log tail showed `FAIL [setup failed]` (the `0` was `tail`'s exit). This is EXACTLY the documented "exit codes after pipes lie" trap from AGENTS.md, and I walked into it on the first gate run. | Caught immediately (FAIL visible in output); no false GREEN claimed | Ran gates as `cmd > log; echo $?` only after the miss | Re-ran correctly; root cause of the FAIL itself was a pre-existing go.sum gap (fixed, A18) |
| D3 | **Parallel session's live landmines (flagged, NOT touched)** — (a) `sqlpkg.DeleteByStream` is cited in CHANGELOG `[Unreleased]` but absent from the api golden — `check-changelog-symbols.sh` exits 1 repo-wide until their golden regen lands; (b) `storage/snapshot_migration.go` has 4 lint failures (errcheck ×2, gofumpt, modernize) — in their in-flight file. My changelog citations pass; my modules lint clean. | Repo-wide symbol gate is RED until their wave finishes | Concurrent-session work in flight; their files, their fix | Recorded here so the next session doesn't misattribute the red gate to this work |
| D4 | **Shared-ledger collision (handled)** — `CHANGELOG.md` was modified by the parallel session between my read and edit (mod-time trip fired). Re-read, then inserted via a stable anchor without touching their entry. No damage; protocol worked as designed. | None | Inherent to concurrent sessions | None needed |

---

## e) WHAT WE SHOULD IMPROVE

1. **Numbers enter comments only after measurement.** D1 happened because writing the "explaining" comment was part of drafting the benchmark. Rule: benchmark doc comments are written AFTER the first real run, with load recorded, or left with a placeholder TODO. (Generalizes `verify-external-claims` to self-authored baselines.)
2. **Gate capture format from line one.** Every gate should be `cmd > /tmp/x.log 2>&1; echo $?` from the FIRST run, never `cmd | tail` + `$?`. D2 cost a re-run and would have cost a false GREEN in a noisier output.
3. **Session-final `#verify-fast`, exclusively.** Per-task gates are necessary but the session GREEN claim should end with one repo-gate run (B3). Cost: minutes. Benefit: no per-module-vs-repo-gate gap.
4. **Check benchmark auto-discovery before adding benchmarks.** `scripts/benchmark-regression.sh` gates median ns/op with a 25% threshold in CI — whether it auto-discovers `BenchmarkCatchUp_ReplayThroughput` (timing-sensitive, load-sensitive) was NOT verified this session. If it does, the new benchmark may flake CI. Item F2.
5. **Run `check-duplication` after adding test files.** The new lifecycle tests share seed-pattern shape with the existing catchup tests; `art-dupl` may see clone groups. Not run this session.
6. **Consumer-facing contracts belong in skill references, not just doc comments.** The skew caveat (A4) is invisible to consumers reading `advanced.md`. When a doc comment defines behavior a consumer must know, update the reference in the same session (C3).
7. **API-addition checklist should include FEATURES.md.** Golden regen + CHANGELOG are reflexes; FEATURES.md inventory is not (C4). Add it to the "Change an Exported Symbol" procedure in AGENTS.md.
8. **doc-check's blind spot is call arity** — it verifies symbol existence, not signatures. The WithBackend item (A8) was exactly this class and had to be done by hand. Teaching doc-check (or a meta-test) to verify arity of known signatures in fences would kill the whole class. Item F12.
9. **Test determinism hooks beat sleeps.** The `readBarrierJournal` made the double-subscribe test deterministic; `CloseWhileBlockedOnFullBuffer` still leans on a 100ms sleep (tolerant assertions make it safe, but a blocking-journal hook would make it airtight). Item F11.
10. **TODO_LIST DONE-annotations with evidence pointers worked well** — keep the format (what + where + how verified), it made this report writable from memory.

---

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

Brainstorm ranked by impact; HARVEST should apply routing rigor (most Medium/Low items are TODO_LIST-grade, a few are ROADMAP fuel — flagged).

| # | Task | Impact | Effort | Category |
| - | ---- | ------ | ------ | -------- |
| 1 | Run `nix run .#verify-fast` exclusively and convert this session's per-module greens into a repo-gate GREEN | High | S | Quality |
| 2 | Check whether `scripts/benchmark-regression.sh` auto-discovers `BenchmarkCatchUp_ReplayThroughput`; pin/exclude it so the new load-sensitive benchmark can't flake the CI regression gate | High | S | Quality |
| 3 | Fix `awaitAck`/`replayPhase` truthfulness: distinguish Close from Nack so Close doesn't log `ERROR "consumer nacked replay event"` (C2) | Medium | S | Bug |
| 4 | Propagate the watermark ULID-skew caveat into `advanced.md` §CatchUp + run doc-check (C3) | Medium | S | Documentation |
| 5 | Resolve the parallel session's `sqlpkg.DeleteByStream` changelog fiction once their wave lands — re-run `check-changelog-symbols.sh` to repo-green (D3a) | High | S | Bug |
| 6 | Fix `storage/snapshot_migration.go` lint (errcheck ×2, gofumpt, modernize) after the parallel session's file settles (D3b) | Medium | S | Quality |
| 7 | Storage tag wave: `EventSchema`/`SQLiteEventSchema` are new API — include in the next `storage/v4` tag with the consumer pin sweep and GOWORK=off build matrix | High | M | Release |
| 8 | Example v5-policy audit (Q3 policy): verify taskmanager + metaengine-quickstart use no v5-removed APIs (getting-started/readme-quickstart verified this session; those two not) | High | M | Quality |
| 9 | Author `example/metaengine-quickstart/README.md` (B2) | Medium | M | Documentation |
| 10 | GOWORK=off go.sum health sweep: script a per-module `go build ./...` matrix to catch the `modernc.org/libc` gap class repo-wide before CI does (A18 class) | High | L | Tooling |
| 11 | Make `CloseWhileBlockedOnFullBuffer` deterministic: blocking-journal hook instead of the 100ms sleep (tolerant assertions currently keep it safe) | Low | S | Quality |
| 12 | Teach doc-check (or a meta-test) arity verification for known signatures in markdown fences — kills the WithBackend class mechanically | High | M | Tooling |
| 13 | Update FEATURES.md with the new `storage.EventSchema`/`SQLiteEventSchema` re-exports (C4) | Medium | S | Documentation |
| 14 | Idle-machine re-run of the catch-up benchmark; update the comment + record in `docs/benchmarks/` with hardware + load (C6, B4) | Low | S | Quality |
| 15 | Set social preview image + homepage URL in GitHub repo settings (B1) | Medium | S | Documentation |
| 16 | Generate a social preview asset (HyperFrames render of the pipeline diagram) so F15 has something to upload | Low | M | Feature |
| 17 | Add a broker-backed catch-up throughput variant via `ephemeral-redis.sh` — real-broker ceiling informs the pipelining decision with data | Medium | M | Feature |
| 18 | Property test: restart-recovery after a skewed-event suppression (checkpoint behind suppressed event ⇒ replay re-delivers) — pins the documented self-healing claim | Medium | M | Quality |
| 19 | Consider checkpoint-vs-watermark hybrid suppression (`suppress ≤ max(checkpoint, watermark)`) to close the skew staleness window by design — needs design review; ROADMAP fuel | High | L | Feature |
| 20 | Generalize `readBarrierJournal`/`drainUntilClosed` into `eventtest` shared primitives for future subscriber tests | Low | S | Cleanup |
| 21 | Extend `readme_claims_test` to pin docs/README.md counts too (ADR count band, module count band — the "68"/"109" rot class) | Low | S | Quality |
| 22 | Extend coverage claims in README to query + command once their baselines are stable | Low | S | Documentation |
| 23 | Record the watermill benchmark baseline + ambient-load caveat in AGENTS.md gotchas (benchmark section) | Low | S | Documentation |
| 24 | Add `-shuffle=on` to the ephemeral-pg/mysql/dgraph integration app invocations at the next tag wave (pre-existing AGENTS.md verdict, still unrolled) | Medium | S | Quality |
| 25 | Evaluate a routine watermill module tag for the doc-comment + test hardening (no API change; batch with the next wave) | Low | S | Release |
| 26 | Sweep per-module READMEs (watermill/storage/transport/*/kv/…) for the numeric-rot class found in docs/README.md | Medium | M | Documentation |
| 27 | Make example README fences drift-proof: embed from `main.go` via a doc-compile test so README code can't silently diverge (readme-quickstart class) | Medium | M | Tooling |
| 28 | Add `TestEveryExampleHasREADME` meta-test (would have caught B2 mechanically) | Low | S | Quality |
| 29 | storage/README DDL section: mention the DuckDB dialect DDL too (`DuckDBDialect.EventSchema()` exists) | Low | S | Documentation |
| 30 | Decide `metaengine.PlanFromSQLite(dsn, ...)`: comment now references only real helpers; the convenience API stays declined unless a consumer asks (decision default: no) | Low | S | Feature |
| 31 | Add a SECURITY.md (proprietary repo — private disclosure path) | Low | S | Documentation |
| 32 | Dependabot/renovate decision for Nix + 82-module Go graph (or document why not) | Low | S | Tooling |
| 33 | Link the new `.github` templates from CONTRIBUTING.md's PR flow section | Low | S | Documentation |
| 34 | Run `nix run .#check-duplication` over the new lifecycle tests; annotate or extract any flagged groups | Low | S | Quality |
| 35 | `readme_claims_test`: guard the `go list` exec with a timeout so a cold CI cache can't stall the suite | Low | S | Quality |
| 36 | Add a watermill test asserting Close does NOT emit the "consumer nacked" log after F3 lands | Low | S | Quality |
| 37 | scripts/check-coverage.sh: add a README-sync warn mode (parse README claims, warn on >1% drift) — formalizes what `readme_claims_test` asserts in tests | Low | S | Tooling |
| 38 | docs/README.md: link check-coverage.sh as the coverage source-of-truth next to the claims | Low | S | Documentation |
| 39 | README "Key modules" table: one-liner accuracy sweep against current APIs (spot-checks passed this session; full sweep not done) | Low | M | Documentation |
| 40 | Watermill README: link the new benchmark + regression tests as verification evidence in the CatchUp section | Low | S | Documentation |
| 41 | ROADMAP-route: F19 (hybrid suppression), F12 (doc-check arity), F10 (go.sum sweep) if not TODO_LIST-grade after HARVEST review | Low | S | Documentation |
| 42 | Annotate/archive this report at the next docs-health pass (point-in-time, will rot) | Low | S | Documentation |
| 43 | `advanced.md`: surface the pipelining-deferral threshold + benchmark ceiling next to the CatchUp recipe (pairs with F4) | Low | S | Documentation |
| 44 | TODO_LIST: add the awaited `awaitAck` fix + the benchmark-discovery check so they don't live only in this report | Medium | S | Documentation |
| 45 | Verify the parallel session's T18 snapshot-tag wave didn't change `storage` test expectations this session touched (coordinated re-run of storage suite after their landing) | Medium | S | Quality |
| 46 | Consider `Suppress`-log dedup in catchup: repeated per-message errors during Close flood logs (observed in benchmark output) | Low | S | Quality |
| 47 | Add `docs/status/README.md` pointer convention check: ensure new reports land unarchived with the harvest note | Low | S | Documentation |
| 48 |getting-started README: swap-to-sqlite claim says "one EngineConfig line" — add the blank-import caveat visible in the code fence (partially there; tighten) | Low | S | Documentation |
| 49 | Bench: add allocs/event sub-metric to the catch-up benchmark for regression visibility (currently only ns/event) | Low | S | Quality |
| 50 | Close the loop: run docs-health HARVEST on this report's (f) section into TODO_LIST/ROADMAP | Medium | S | Documentation |

---

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Social preview + homepage (B1/F15/F16):** These are GitHub settings-UI fields only you can set. Do you want me to generate a social preview image asset (e.g., a HyperFrames render of the pipeline diagram) for you to upload, and which URL should the repo's homepage field point to (lars.software project page, docs site, or none)?

2. **Catch-up throughput policy (C5/F17/F19):** I documented "10× degradation ⇒ revisit ack-window pipelining" as the trigger. Is that the policy you want, or do you want a hard SLO for `CatchUpSubscriber` (e.g., ≥100K events/s) — optionally enforced as a CI bench gate — and should the hybrid checkpoint-vs-watermark suppression rule (F19) go on the ROADMAP for design review?

3. **metaengine-quickstart (B2/F9):** The harvested item said "4 example README files" but only 3 examples have READMEs. Should `metaengine-quickstart` get a full README (I'll author it from its three demo files), or is the docs/README.md directory link sufficient — i.e., was "4" ever accurate, or should the meta-test enforce "every example has a README" going forward?

---

**HARVEST note:** Section (f) is the primary input for `docs-health` HARVEST into `TODO_LIST.md`/`ROADMAP.md`. Items F3/F4/F9/F13/F44 are already partially reflected in TODO_LIST DONE-annotations' leftovers; the rest need routing.

**Format override flag:** the status-report skill's canonical output is a styled HTML dashboard; the user explicitly requested `.md`, so this report is Markdown. Not propagated back into the skill.
