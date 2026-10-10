# BDD Adoption Wave — T20 Complete, T21+T22 Done, T23 Research Finished (Status Report)

**Session:** 2026-10-10, ~06:00–07:27 CEST (continues the 05:54 report)
**Plan:** `docs/planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md`
**Mode:** Full Execution, resuming at T20 implementation after owner GO.
**Numbering note:** this report uses the HANDOFF numbering (T21=property, T22=bench, T23=chaos+SSE, T24=lint, T25=codemod, T26=fleet+watermill, T27=final). The plan file's own table numbers them T21–T27 one row earlier for the same work; content is identical.

---

## a) FULLY DONE (this session)

### T20 — Harness presets (complete, all gates green)

- **`systemscenario/presets.go`** (NEW, committed): `Memory()` — memory primary (journal + projections) + dedicated memory `timers` engine (TimeAdvances works out of the box); `SQLite(t testing.TB)` — file-backed SQLite primary under `t.TempDir()` with `journal_mode=wal` + memory timers. File DSN deliberately (documented in code + README): engines own and close their `*sql.DB`; shared-cache in-memory DBs die with the last connection, so file DSNs keep reopen-style assertions honest. Blank import of `metaengine/sqliteengine/v4` registers the "sqlite" driver (database/sql pattern).
- **`systemscenario/go.mod`**: +`sqliteengine/v4 v4.5.2` direct (modernc.org/sqlite indirect through it).
- **`systemscenario/presets_test.go`** (NEW): 4 tests — Memory boot through the real Given/When/Then flow, SQLite boot, SQLite `ThenQueryFunc` projection assertion (Status=completed), Memory timers-engine resolution. The timers test bootstraps via `system.New` directly (the vacuous-pass guard correctly rejected a harness chain with no Then*).
- **Dogfooding:** `memoryDeployment()` fixture now returns `systemscenario.Memory()`; `timerDeployment()` likewise (its hand-rolled timers engine was redundant once Memory() included one).
- **Dep budget:** `DEP_BUDGET[systemscenario]` 7→8 in `scripts/check-module-layers.sh` with the zero-new-external-deps rationale (sqliteengine in-repo; modernc rides indirect; system's production go.mod already requires both). `nix run .#check-arch` GREEN.
- **Docs:** `systemscenario/README.md` gained a "Deployment presets" table; `recipes.md` §2.43 gained a preset paragraph + compile-fenced example, classified in `cmd/doc-check/recipes_catalog_meta2.go` (#3 entry: context/testing/system/systemscenario imports, taskDomain preamble, `_ =` trailers). `cmd/doc-check` reference gate exit 0 (1188 refs valid); `TestRecipes` GREEN (18s warm) after fixing the first-run "unclassified block" failure.
- **CHANGELOG [Unreleased] Added:** presets entry (symbols-gate verified — 9 citations honest).
- **API golden:** 8432 → 8434 exports (+`Memory`, +`SQLite`).
- **Verification:** module suite green; `-race` green (1.5s); `go vet` green; gofmt clean.

### T21 — Fold-vs-read-model property (complete)

- **`systemscenario/property_test.go`:** NEW `TestProperty_ReadModelMatchesFold` (rapid, 100 iterations, ~1.1s total): random `task.create`/`task.rename`/`task.complete` sequences (rejections included in the adversarial surface — they leave no journal events), oracle = pure fold (`applyTask`) over the ACTUAL journal for the stream, assertion = harness `ThenQuery` against the Lookup read-model row. Pins that a dropped, duplicated, or mis-projected event anywhere between journal write and view breaks the equality.
- Design iterations (honest): (1) one shared scenario across iterations failed — harness enforces one Given per scenario ("Given must run before the first When"); (2) `systemscenario.System(rt, ...)` failed — `*rapid.T` does not implement `testing.TB` (missing `ArtifactDir`, Go 1.27); (3) final shape: per-iteration `system.New` + `Start` + `systemscenario.Adopt(t, ctx, sys)` with `defer sys.Close()` — Adopt keeps the lifecycle caller-owned so teardown is per-iteration, not 100 live systems until test end.
- Discovery: the file ALREADY carried `TestProperty_RandomCommandSequencesKeepJournalOrdered` + `TestSystem_JournalEquivalenceAcrossDeployments` from the prior session — the fold property was the missing leg; now all three live together.

### T22 — Boot bench metrics + README (complete)

- `BenchmarkScenarioBoot` (existed from prior session) measured with `-benchmem -count=3`: **~1.2–1.8 ms/op, ~9.2 MB/op, ~1371 allocs/op** (32-vCPU host).
- **README** gained "Cost of a scenario": the snapshot, the explanation (9 MB = the composition root itself — engines, dispatcher, bus, projection host, middleware chains; one scenario is one real boot, which is the point), when to fall back to decider-pure `scenario`, and a pointer to the two property tests as the correctness-depth story.
- Module `-race` suite green after; `go vet` green.

### T23 — Research (implementation NOT started)

- **Injection seam established:** `system.New` has NO journal-override option; the event store is `NewEventAdapter(engine.(metaengine.StreamLogBackend), "events")` built from the deployment engine. Therefore a DelayedJournal must arrive as an ENGINE wrapper registered through `metaengine.RegisterDriver` — the database/sql-style seam the planner already uses (`system/driver_registry.go`).
- **Chosen design (ready to implement):** `systemscenario` helper that (a) constructs the base engine via `LookupDriver(base)`, (b) wraps it in a struct EMBEDDING `metaengine.Engine` — all capability interfaces (DueClaimer, FactSink, AtomicAppender, SnapshotBackend, …) promote for free through embedding so type-assertion discovery keeps working, (c) overrides ONLY the `StreamLogBackend` methods (`StreamAppend`, `StreamRead`, `StreamVersion`, `JournalReadAll`, `JournalReadFrom`) with a sleep-injected delegating body, (d) registers under a UNIQUE per-call driver name and returns it for `EngineConfig{Driver: <name>}` — unique names avoid process-global registry clashes between tests with different delays.
- Signatures verified in `metaengine/engine.go` (StreamLogBackend at :468) and `metaengine/registry.go` (RegisterDriver :62).
- **ServeSSE helper:** not yet designed (next step after DelayedJournal lands).

---

## b) PARTIALLY DONE

- **F20.4 "pilots adopt presets" — intentionally partial, needs owner ratification.** Findings: all three companion pilots boot through their OWN production facades — cqrs-htmx `systemadapter.RecommendedMemoryDeployment()` (production preset: SourceOfTruth only, no projections/timers roles), go-appkit `NewEventService(EventConfig{Driver})` (facade owns system.New), go-appkit integration pilot `es.System()` (same facade pattern). Their defining property is fixture-from-production-config (stated in their file headers); swapping to the generic `systemscenario.Memory()` would test a DIFFERENT deployment than production recommends. Presets are for tests with no production config to mirror. Dogfooding happened where honest: systemscenario's own fixtures now run on `Memory()`.
- **T23 chaos+SSE:** research complete (above), zero implementation.

## c) NOT STARTED

- **T24** — cqrs-lint advisory rule (system-booting test file without harness usage) + module-catalog meta-test + api golden regen.
- **T25** — cqrs-upgrade suggestion rule (`eventually`-style blocks → `ThenQuery` hint).
- **T26** — fleet rollout (example/taskmanager + example/goal-shaped-app onto harness; cqrs-htmx `awaitNotFound` → `ThenQueryEventuallyFails` sweep) + watermill `ErrReentrantPublish` loud-fail guard.
- **T27** — final `#verify`, companion suites, retro → gotchas, plan addendum, TODO_LIST strikes, CHANGELOG symbols gate, final api golden, superseding report.

## d) TOTALLY FUCKED UP (nothing destructive — process misses, all recovered)

1. **Silent edit loss (caught late):** one CHANGELOG edit raced the auto-commit daemon ("file modified since read"); I initially MISSED the failure output and only discovered the entry was missing via a verification grep ~2 tool calls later. Lesson applied: verify every edit with `rg` immediately.
2. **Three edit-tool daemon races total** (fixtures_test, timer_test, CHANGELOG ×2) — all resolved by re-read + re-edit; no content lost.
3. **First presets_test draft:** called `ThenQueryFunc` on `*Scenario` (it lives on `*WhenPhase`) — compile error, restructured bootPresetScenario to return the phase. Also an initial `sed /dev/null` no-op fumble.
4. **Property test took 3 iterations** (documented in §a) — each failure was a real API fact learned (Given-per-scenario; rapid.T ≠ testing.TB under Go 1.27; Adopt = caller-owned lifecycle).
5. **Recipes gate first-run failure** (88s cold): new fence without a catalog entry — the gate did its job; fixed by classifying #3. Expected cost, not a fuckup, but it cost one gate cycle.

## e) WHAT WE SHOULD IMPROVE

1. **Edit-then-verify discipline:** with the daemon racing every write, EVERY edit needs an immediate `rg`/build confirmation in the same breath — two of this session's misses were verification lag, not the race itself.
2. **Concurrent-agent tree hygiene:** 373 files are currently modified by OTHER agents (benchkit, cqrs-lint sweep, formatting). Before any T27 `#verify`, the tree must be quiesced; per-file `git diff` on my artifacts confirmed only benign formatting/annotation drift on my files (blank line, import order) — but final verify must re-check.
3. **The vacuous-pass guard earned its keep twice** (timers test, property test shape) — consider documenting the `system.New`-direct escape hatch in the README ("wiring checks that are not scenarios").
4. **Benchmark variance:** bench runs ranged 1.23–1.77 ms/op across 3 counts on a loaded host; the README states a range, not a number — keep that convention for future metric snapshots (benchstat discipline for any comparison).
5. **rapid.T vs testing.TB:** Go 1.27's `ArtifactDir` widened the TB interface; rapid lags. Property tests inside the harness should standardize on the Adopt pattern (outer `t`, caller-owned lifecycle) until rapid catches up — worth a gotcha note at T27.

## f) NEXT — up to 50 items (ordered)

**T23 (chaos + SSE):**
1. Implement `chaos.go`: `DelayedDriver(base string, delay time.Duration) string` in systemscenario (embed Engine, override StreamLogBackend methods with sleep).
2. Chaos test: scenario on DelayedDriver(memory, 2–5ms) — random command sequence, assert journal ordering + fold-vs-read-model still hold under latency.
3. Delayed-driver edge: delay on `StreamAppend` must not break optimistic-concurrency detection (AtomicAppender still promotes).
4. ServeSSE helper research: `metaengine.ServeSSE[V]` + `Watcher[V]` surface; decide helper shape (poll-watcher vs httptest SSE client).
5. Implement SSE assertion helper (e.g. `ThenSSECollection(collection, check)` or httptest-based).
6. SSE helper test: act → watcher/HTTP stream sees expected update(s).
7. README row(s) for chaos + SSE; api golden regen; CHANGELOG entry; race gate.

**T24 (cqrs-lint advisory):**
8. Read `cmd/cqrs-lint/pkg/analyzer/` rule architecture (E018 as template).
9. Design E019: test file constructing `system.New` directly without `systemscenario` import → advisory "consider the BDD harness".
10. Implement analyzer + diagnostic message with recipe pointer (§2.43).
11. Fixture tests (positive/negative/edge: non-test files, Adopt usage counts as harness usage).
12. Register rule in the rule catalog + module-catalog meta-test still passes.
13. cqrs-lint docs (README rule table) + `#check-lint-config` gate.
14. api-stability regen (cqrs-lint exports change) + `TestEvery` meta-test.
15. Version/tag decision for cqrs-lint rides next wave (Q2 ruling analog).

**T25 (codemod suggestion):**
16. Read `cmd/cqrs-upgrade/` analyzer architecture.
17. AST matcher for `eventually`-style blocks (`require.Eventually` / `assert.Eventually` / hand-rolled timeout loops around queries).
18. Suggestion emitter: hint text → `ThenQuery` (+ `Await()` when bus-driven), NEVER auto-rewrite.
19. Fixture tests: various eventually shapes (testify Eventually, for+time.After loops, ticker loops).
20. cqrs-upgrade README/docs row.
21. api golden + meta-tests green.

**T26 (fleet rollout + watermill):**
22. Read `example/taskmanager` current test suite shape.
23. Migrate taskmanager suite onto systemscenario (Given/When/Then + presets).
24. Read `example/goal-shaped-app` suite.
25. Migrate goal-shaped-app suite onto harness.
26. Both example modules: build + suites green; `examplePaths` registration unchanged (build-only in CI).
27. cqrs-htmx sweep: `awaitNotFound` hand-rolls → `ThenQueryEventuallyFails`; suite green; commit.
28. Feedback memo: API friction found during both migrations (into the T27 report).
29. watermill: locate publish path + handler goroutine; read ADR-0154 evidence pack repro shape.
30. Implement goroutine-local publisher-depth flag (no goroutine IDs) — `sync` of a context value or per-goroutine depth via entered/publish wrapper.
31. `ErrReentrantPublish` sentinel returned instead of hang; CHANGELOG + ADR-0154 note.
32. watermill tests: nested publish fails fast; non-nested unaffected; broker suite (`#integration-redis`) still green.

**T27 (tail):**
33. `nix run .#verify` — expect my modules green; itemize external reds (core/v5, metaengine typed_reader_scan 367, projectionhost host.go 383, README links core/v5 externals).
34. Companion full suites (cqrs-htmx, go-appkit) green; go-appkit's 2 pre-existing integration failures re-verified as unchanged.
35. `nix run .#check-md-go` (planning docs parse gate).
36. Plan addendum: DONE/PARTIAL/NOT-SHIPPED per section; restate the 5-module wave; annotate (never rewrite).
37. Retro → `docs/agents/gotchas-*`: new lessons (build-green ≠ test-green; `--no-verify` for non-code commits; grep go.mod after dependency surgery; edit-then-verify under daemon races; rapid.T ≠ testing.TB; one-Given-per-scenario).
38. TODO_LIST strike-throughs for wave items; FEATURES.md row updates if any surface changed (presets row!).
39. CHANGELOG symbols gate + final api golden regen.
40. `check-versions-manifest.sh --check` (untagged-trains list — systemscenario v4.1.0 planning for next wave per Q2).
41. Final superseding status report.
42. Owner-ruling defaults from the 05:54 report §g if still unanswered (gjson KEEP; companion cadence theirs; ADR-0152 addendum only on approval).

**Polish / opportunistic (only if owner wants):**
43. SKILL `references/modules.md` systemscenario row: add presets mention.
44. Consider `SQLite(t)` variant with caller-supplied DSN (reopen/restart scenarios) — parked unless asked.
45. Consider exposing the chaos delay knob per-operation (read vs write) if the single-delay shape proves crude.
46. Doc note in gotchas-testing about Adopt-lifecycle pattern for property loops.

## g) QUESTIONS (cannot figure out myself)

1. **F20.4 ruling:** cqrs-htmx + go-appkit pilots keep their production-shaped deployments (fixture-from-production-config is their stated purpose; presets adopted instead in systemscenario's own fixtures + skill recipe). Accept as DONE-with-rationale, or do you want the pilots force-flipped to `systemscenario.Memory()` anyway (weakening the production-config property)?
2. **DelayedDriver seam:** implementing chaos via `metaengine.RegisterDriver` under a per-call unique name is the only seam `system.New` offers today. OK to ship that (process-global registry, names like `systemscenario/delayed-memory-<n>`), or should I first add a `system.Option` journal/engine wrapper seam to go-cqrs-lite itself (a system-module API change riding the next wave)?
3. **Carried from the 05:54 report §g (defaults already chosen, still yours to overturn):** gjson v1.20.0 passenger KEEP; companion repos tag cadence left to them; ADR-0152 "resolved-by-consumer" addendum row drafted at T27 only on your yes.

---

**Tree state at report time:** 373 files modified by concurrent agents (benchkit, cqrs-lint, formatting sweeps); my artifacts verified present and committed via daemon chores (`presets.go` at `ea368b342`, later waves in `f45047326` etc.). Benign drift only on my test files (blank line, import order). No authored commits made this session (all absorbed by daemon; verified via `git show`).
