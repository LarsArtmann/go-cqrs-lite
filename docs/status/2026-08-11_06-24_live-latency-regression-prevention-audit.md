# Status Report: Live-Latency Regression Prevention + Session Audit

**Date:** 2026-08-11 06:24
**Session start:** ~06:00 (resumed from prior session handoff)
**Branch:** master
**Head commit:** `755f4f579` — fix(metaengine): make embedding regression of live-latency impossible to reintroduce silently

---

## a) FULLY DONE

### 1. Verified ALL engines have embedded Calibration (not named field)

All 8 engines that embed `metaengine.Calibration` were confirmed to use anonymous embedding (not a named `cal` field). The embedding fix was applied by the auto-commit daemon in commit `99f8601a6` before this session started, not by me. I verified each engine:

| Engine | Struct location | Embeds Calibration? | TrackerHost assertion? |
|--------|----------------|--------------------|-----------------------|
| pgengine | engine.go:72 | Yes (anonymous) | Yes (engine.go:371) |
| dgraphengine | engine.go:60 | Yes (anonymous) | Yes (engine.go:365) |
| sqliteengine | engine.go:32 | Yes (anonymous) | Yes (engine.go:609) |
| badgerengine | engine.go:55 | Yes (anonymous) | Yes (engine.go:224) |
| bboltengine | engine.go:66 | Yes (anonymous) | Yes (engine.go:53) |
| pebbleengine | engine.go:66 | Yes (anonymous) | Yes (engine.go:53) |
| duckdbengine | engine.go:55 | Yes (anonymous) | Yes (engine.go:45) |
| mysqlengine | engine.go:52 | Yes (anonymous) | Yes (engine.go:43) |
| irohengine | engine.go:14 | No (wrapper design) | N/A |

### 2. Added compile-time interface assertions to 4 engines

I added `TrackerHost`, `Prober`, `TransactMeasurer`, and `Calibratable` assertions to the 4 engines that lacked them (the other 4 already had them via daemon commits):

- **pgengine/engine.go**: Added `Calibratable`, `TrackerHost`, `Prober`, `TransactMeasurer` (lines 370-373)
- **dgraphengine/engine.go**: Added `TrackerHost`, `Prober`, `TransactMeasurer` (lines 365-367)
- **sqliteengine/engine.go**: Added `Calibratable`, `TrackerHost`, `Prober` (lines 608-610)
- **badgerengine/engine.go**: Added `TrackerHost` (line 224)

Committed by daemon in `ce39ac187`.

### 3. Added ProbeEngine warn-on-missing-TrackerHost

**File:** `metaengine/probe.go` (lines 263-277)

Added `slog.Warn` in `ProbeEngine` when an engine implements `Prober`/`TransactMeasurer` but not `TrackerHost`. The warning includes the engine name and a hint to embed `metaengine.Calibration`. This is the runtime safety net that catches the exact bug class the compile-time assertions prevent.

Committed by daemon in `755f4f579`.

### 4. Added test for the ProbeEngine warning

**File:** `metaengine/probe_warn_test.go` (63 lines, UNCOMMITTED)

`TestProbeEngine_WarnOnMissingTrackerHost` creates a fake engine that implements `Prober` but NOT `TrackerHost`, captures slog output via a `bytes.Buffer`, and asserts the warning fires with the engine name. Test passes.

### 5. bboltengine hygiene

- **go.mod**: Ran `go mod tidy`, removed unused `dustin/go-humanize` indirect dependency. Committed in `ce39ac187`.
- **lint**: Files pass gofumpt + go vet + verify-fast lint pipeline.

### 6. api-stability golden verified

Golden file is up to date: "API surface OK: 3999 exports verified". Meta-tests (`TestEveryGoModDirIsInModulesList`, `TestEveryGoModDirIsInTestModules`) pass.

### 7. Documentation updated

- **TODO_LIST.md**: 7 items marked `[x]` DONE (tursoengine verification, TrackerHost assertions, ProbeEngine warn, api-stability golden, bboltengine tidy + lint).
- **CHANGELOG.md**: Added "Live-latency regression prevention" section documenting the compile-time assertions, ProbeEngine warning, and bboltengine tidy.

### 8. Integration tests verified

- `metaengine/pgengine`: `TestProbeEngine_RealPostgres_LiveRTT` PASS (0.10s), `TestProbeEngine_RealPostgres_StaleAfterStop` PASS (0.60s). Real Postgres testcontainer.
- `metaengine` core: All ProbeEngine tests pass including new `TestProbeEngine_WarnOnMissingTrackerHost`.
- doc-check: 708 references valid across 42 packages.

---

## b) PARTIALLY DONE

### 1. Full verify gate (`nix run .#verify`)

Ran `nix run .#verify-fast` instead of the full `#verify`. The fast gate passed for all modules I touched. The full gate includes race detection, coverage drift, and duplication checks that were NOT run. Risk: low — the changes are assertions (compile-time), a slog.Warn call, and a test file. No runtime logic changed.

### 2. CHANGELOG does not mention probe_warn_test.go

The CHANGELOG entry for the ProbeEngine warning was added before I wrote the test. The test file (`probe_warn_test.go`) is not mentioned in the CHANGELOG.

---

## c) NOT STARTED

### From the prior session handoff (paste_1.txt items)

1. **Fix `record_stamp.go` GOWORK=off build failure** — Already resolved by prior session (record/v4.1.0 tag). Marked DONE in TODO_LIST.
2. **Add `CounterIncrement` benchmark to pebbleengine calibration** — Not started. pebbleengine lacks calibration data for this ADT operation.
3. **bboltengine parity gaps** — Not started. bboltengine lacks `edge_cases_test.go`, `fuzz_test.go`, `stream_log_test.go`, `watcher_test.go`, `scan_bench_test.go` that pebbleengine has.

### From this session

4. **Run full `nix run .#verify`** — Not run (only verify-fast).
5. **Run `nix run .#check-arch`** — Dependency budget enforcement not run after changes.
6. **Run `nix run .#check-duplication`** — No-new-clones gate not run.

---

## d) TOTALLY FUCKED UP

### Nothing is fucked up — but here's what was wasteful

1. **I planned fixes for bugs that were already fixed.** The prior session's handoff said dgraphengine and badgerengine still had the named-field bug. I spent tool calls on agents reading those files only to discover the auto-commit daemon had already fixed ALL engines (commit `99f8601a6`). I should have run a single grep for `cal\s+metaengine\.Calibration` across ALL engine directories FIRST, before dispatching 4 parallel agents to read individual engine files.

2. **I wrote the ProbeEngine warning without a test.** The first thing I did after adding the warning was run existing tests — which all passed because they use engines that DO embed Calibration. The warning code path was untested until the user's "what did you forget?" prompt. This is the exact anti-pattern the prior session's bug embodied: "shipped without ever being tested against the failure case."

3. **Two intermediate TODO updates were wasteful.** I marked items done in TODO_LIST.md, then discovered more work (the test), then had to update again. Should have done ALL the work first, then updated docs once.

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements

1. **Verify state BEFORE trusting handoff docs.** The prior session's report was a point-in-time snapshot. By the time I resumed, the daemon had already applied fixes. A 5-second grep (`grep -rn 'cal\s+metaengine\.Calibration' metaengine/*/engine.go`) would have saved 4 agent calls.

2. **Always write the test for new behavior in the same edit.** Adding the ProbeEngine warning without immediately writing a test for it repeated the exact bug pattern this whole task was about: untested code paths that silently misbehave.

3. **The compile-time assertion pattern should be a codebase convention.** Every engine that embeds `Calibration` should assert `TrackerHost`. This should be documented in AGENTS.md as a required pattern for new engines. Currently it's just "something we did" — not "something we enforce."

### Code improvements

4. **irohengine is the odd one out.** It doesn't embed `Calibration` at all — it uses a passthrough/wrapper design. If iroh ever needs live latency probing, it will need special handling. This is a design decision, not a bug, but it should be documented.

5. **The ProbeEngine warning fires once per ProbeEngine call.** If a consumer calls ProbeEngine on the same broken engine multiple times, they get multiple warnings. Consider deduplicating with a `sync.Once`.

6. **The `proberNoTrackerHost` test type in probe_warn_test.go is dead simple.** It could be extracted into a shared test helper if other engine modules want to verify the same warning behavior. Low priority.

7. **No metric/counter for the warning.** The slog.Warn is good for logs, but in production a Prometheus counter (`probe_engine_missing_tracker_host_total`) would be more actionable.

---

## f) Up to 50 things to do next

### Immediate (this session's loose ends)

1. Commit `probe_warn_test.go` (currently uncommitted)
2. Update CHANGELOG to mention `probe_warn_test.go`
3. Run full `nix run .#verify` gate
4. Run `nix run .#check-arch` (dependency budget)
5. Run `nix run .#check-duplication` (no-new-clones gate)

### Regression prevention hardening

6. Add a lint rule (cmd/cqrs-lint) that flags engine structs with a named `Calibration` field instead of embedding
7. Document the TrackerHost assertion convention in AGENTS.md Internal Contracts section
8. Add `TrackerHost` assertion to the engine template/recipe in `.agents/skills/go-cqrs-lite/references/recipes.md`
9. Consider adding `sync.Once` dedup to the ProbeEngine warning (item 5 above)
10. Add a Prometheus counter for missing TrackerHost (item 7 above)

### Pre-existing failures (NOT caused by this session)

11. Fix `metaengine/bench` fold reflect panic: `reflect: Call using map[string]interface {} as type bench_test.OrderView` — affects 3 tests: `TestPromise_CostModelAccuracy`, `TestPromise_CrossEngine_ParityAtScale`, `TestPromise_ParityWithDuckDB`
12. Investigate `metaengine/on_test.go:56` gopls syntax error (pre-existing, possibly build-tag related)

### Calibration parity

13. Add `CounterIncrement` benchmark to pebbleengine calibration (from paste_1.txt)
14. Verify all engines have calibration data for all ADT operations they support

### bboltengine parity (from paste_1.txt)

15. Add `edge_cases_test.go` to bboltengine (pebbleengine has it)
16. Add `fuzz_test.go` to bboltengine
17. Add `stream_log_test.go` to bboltengine
18. Add `watcher_test.go` to bboltengine
19. Add `scan_bench_test.go` to bboltengine
20. Prioritize by which ADT operations bbolt supports differently from pebble

### Integration test coverage

21. Write dgraphengine ProbeEngine integration test (mirrors pgengine probe_live_test.go) — dgraph now has compile-time assertions but no integration test proving live RTT flows end-to-end
22. Write sqliteengine/tursoengine ProbeEngine integration test (Turso remote mode)
23. Write mysqlengine ProbeEngine integration test
24. Add a cross-engine ProbeEngine parity test in `metaengine/bench/` or `metaengine/enginetest/`

### Live-latency system polish

25. Add `GetEngineStats` assertions to the ProbeEngine tests (verify stats show "rtt=live" format)
26. Test `Store.Replan(ctx)` with live latency data — does it actually re-route on drift?
27. Test `Store.StartAutoReplan` lifecycle (start, drift detection, stop, cleanup)
28. Test `Doctor` command output format includes the `--- Routing ---` section with live data
29. Verify `NsForRead` RTT amortization works correctly for scan-pattern fallback costs
30. Test hysteresis deadband (`WithRoutingHysteresis` / `WithRoutingMinDelta`) under latency shifts

### irohengine

31. Decide whether irohengine should embed Calibration (currently wrapper design)
32. If yes, add the embedding + TrackerHost assertion
33. If no, document why iroh is exempt from live-latency probing

### Fold system (pre-existing bench failures)

34. Fix the `OnRecord` fold inference to handle `map[string]interface{}` → typed struct reflect.Call
35. The fold panic suggests the auto-inferred fold calls a function with a `map[string]interface{}` when it expects `bench_test.OrderView` — investigate the type mismatch in `metaengine/store.go` fold application
36. Add a guard in the fold execution path that catches reflect.Call type mismatches with a clear error instead of panicking

### Architecture/debt

37. The `Calibration` struct is embedded in 8 engines — consider documenting this as a mandatory pattern in the seven-tier model docs
38. `metaengine/on_test.go:56` has a syntax error that gopls reports but builds pass — investigate if this is a gopls false positive (jsonv2 build tag)
39. Run `nix run .#check-coverage` to verify no coverage drift from the changes
40. Update the METAENGINE-LIVE-LATENCY-MODEL.md design doc to mention the new compile-time assertions and ProbeEngine warning

### Recipes and skill docs

41. Update `.agents/skills/go-cqrs-lite/references/recipes.md` §2.11 to mention the TrackerHost assertion requirement for new remote engines
42. Add a "Creating a new engine" checklist to the skill docs that includes: embed Calibration, add TrackerHost assertion, implement Prober/TransactMeasurer if remote
43. Update `docs/METAENGINE_DOMAIN_LANGUAGE.md` with TrackerHost interface definition

### GOWORK=off hygiene

44. Verify `GOWORK=off go build` works for all engine modules after record/v4.1.0
45. Clean up any remaining `replace` directives that are no longer needed

### Monitoring

46. Add a `Doctor` test that exercises the `--- Routing ---` section with a real Store
47. Add an `EXPLAIN` test that surfaces live RTT labels
48. Verify `GetEngineStats` labels stale measurements correctly after probe loop stops

### Misc

49. Consider extracting the `proberNoTrackerHost` test type into `enginetest/` as a reusable harness
50. Review whether the ProbeEngine warning should also fire for `TransactMeasurer`-only engines (currently shares the same else branch)

---

## g) Questions

### Q1: Should the ProbeEngine warning be an error instead of a warning?

Currently `ProbeEngine` logs `slog.Warn` and continues with a no-op handle. This is backward-compatible (local engines that don't implement any probing interface are unaffected), but it means a wiring bug in a remote engine still produces a "working" ProbeHandle that silently does nothing. Should this be a hard error (`return nil, error`) for engines that implement `Prober` but not `TrackerHost`? That would be a breaking API change for `ProbeEngine` (currently returns only `*ProbeHandle`), but would make the bug impossible to ignore.

### Q2: Should irohengine embed Calibration?

irohengine uses a passthrough/wrapper design (`replicatedEngine` wraps an inner engine). It does NOT embed `Calibration`. This means it cannot participate in live-latency probing. Is this intentional (iroh handles latency at a different layer), or is it a gap that should be fixed? If intentional, I'll document it as a design decision. If not, it needs the embedding fix + TrackerHost assertion.

### Q3: Scope of the bench fold panic fix?

The 3 pre-existing failures in `metaengine/bench/` (`TestPromise_CostModelAccuracy`, `TestPromise_CrossEngine_ParityAtScale`, `TestPromise_ParityWithDuckDB`) are caused by a reflect.Call type mismatch in fold execution: `map[string]interface{}` vs `bench_test.OrderView`. This looks like the auto-inferred fold calls the update function with untyped data instead of the typed struct. Should I fix this in this session, or is it tracked elsewhere? It blocks `nix run .#verify` from going fully GREEN.
