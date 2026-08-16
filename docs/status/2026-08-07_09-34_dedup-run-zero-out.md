# Deduplication Status Report — 2026-08-07 09:34

## Headline

Reduced art-dupl clone groups from **13 → 2** (85% reduction) at threshold `-t 7`.
Removed **~38 → 2 actual clones** (~95% reduction in instance count).

## What I Did — Group by Group

### ✅ FULLY DONE (11 of 13)

| #  | Group                                                               | Before                | After                                                         | Technique                                       |
| -- | ------------------------------------------------------------------- | --------------------- | ------------------------------------------------------------- | ----------------------------------------------- |
| 1  | pg_testcontainer_test.go (projectionhost vs scheduling/sqlstore)    | 2 files × 132 lines   | Shared `testutil/pgtestcontainer` module                      | New Go module + per-package wrapper             |
| 7  | Key encoding helpers (badgerengine.go vs pebbleengine.go)           | ~60 lines duplicated  | `metaengine/v4/keycodec` package                              | New sub-package + var aliases to `keycodec.X`   |
| 3  | Engine Scan tests (duckdbengine vs pgengine)                        | 2 × 103 lines         | `enginetest.RunScanBackendTest`                               | New `metaengine/v4/enginetest` sub-package      |
| 13 | Watcher Replay tests (duckdbengine vs pgengine)                     | 2 × 57 lines          | `enginetest.RunWatcherReplayTest[V]`                          | Generic helper with `WatcherReplaySetup[V]`     |
| 8  | pgTestDSN helpers (metaengine/pgengine vs stack/postgres)           | 2 × 130 lines         | `pgtestcontainer.DSN`                                         | Module-level `pgtestcontainer`                  |
| 10 | TestMain container setup (metaengine/pgengine vs stack/postgres)    | 2 × 50 lines          | `pgtestcontainer.TestMain`                                    | Module-level `pgtestcontainer`                  |
| 5  | DuckDB pushdown tests setup                                         | 5 × 16 lines          | `newDuckDBPushdown` + `enginetest.RunPushdownTest`            | New helper + shared run-loop                    |
| 6  | Postgres pushdown tests setup                                       | 5 × 16 lines          | `newPostgresPushdown` + `enginetest.RunPushdownTest`          | New helper + shared run-loop                    |
| 9  | StreamLog + AtomicAppender tests (duckdbengine vs pebbleengine)     | 2 × 60 + 2 × 30 lines | `enginetest.RunStreamLogBackendTest`, `RunAtomicAppenderTest` | Two new shared helpers                          |
| 11 | Iroh QUIC 2-node preamble (5 occurrences in transport_test.go)      | 5 × 8 lines           | `quicCluster` struct + `newQuicCluster(t)`                    | Local helper with cleanup via t.Cleanup         |
| 12 | FlightRecorder test setup (5 occurrences in flightrecorder_test.go) | 5 × 14 lines          | `newFRTestRecorder(t) (*safeBuffer, *Recorder)`               | Local helper that returns both buf and recorder |

### ⚠️ NOT STARTED (2 of 13 — left for next session)

#### #2 — `bench_duckdb_extensions_cgo_test.go` vs `bench_pebble_extensions_test.go` (metaengine/bench)

**Status:** Started but interrupted by user prompt before the refactor landed.

**What's there:** Two 88-line test bodies that:

1. Build `events` via identical loops over 5 fixture orders
2. Call `memStore.Apply` + `duckStore.Apply` (or `pebStore.Apply`) per event with identical error handling
3. Run 4 parity queries: `find_order`, `count_by_status`, `orders_by_customer`, `total_revenue`
4. Compare results across stores

**Differences:** Only the engine variable name (`duckStore` vs `pebStore`) and the error message prefix (`"duckdb"` vs `"pebble"`).

**Recommended fix:** Extract a generic `RunEngineParityTest(t, memStore, altStore, altName, fixture, queries)` to `metaengine/bench/parity.go` (or a new `metaengine/parity` sub-package). The shared body iterates fixture, calls both `Apply`s, and runs all 4 queries. Each call site (DuckDB and Pebble) reduces to ~10 lines of setup.

#### #4 — `cmd/cqrs-lint/explain.go:97-121` vs `cmd/cqrs-lint/explain.go:468-488` (two `writeSectionHeader` + 25-line literal blocks)

**Status:** Not started.

**What's there:** Two `b.WriteString(...)` calls — one for "CONFIG FILE" with a JSON example, one for "SUPPRESSION SYNTAX" with ignore-directive examples. Both are 25-line literals in `explain.go`.

**Differences:** Only the section title and content of the literal text.

**Recommended fix:** These are documentation literals — extracting them into a helper would replace 25 hard-coded lines with a 25-line slice/struct passed to a `writeSection(b, "TITLE", []string{...})` helper. **Genuine tradeoff:** this may NOT be worth it because the literals are the documentation, not boilerplate. They're read in-place by `nix develop` users running `cqrs-lint explain`. The helper might obscure the section content for maintainers. **Decision needed: refactor or accept.**

### 🔧 WHAT I SHOULD IMPROVE

1. **The metaengine/bench Group #2 fix should have landed.** I had the diagnosis (two 88-line test bodies with only engine name diff) but the user prompted for status before I could extract the shared helper. Easy 30-min win, just needs the actual edit.

2. **Group #4 might be a false positive.** explain.go's two blocks are documentation literals, not boilerplate. Extracting a `writeSection` helper is a judgment call. The skill says "extract, accept, or exclude" — I should have asked the user before doing it, but I also should have analyzed and presented the option.

3. **The shared `enginetest` package's API surface is larger than necessary.** I added `RunStreamLogBackendTest`, `RunAtomicAppenderTest`, `RunScanBackendTest`, `RunPushdownTest`, `RunWatcherReplayTest`, plus a few fixtures. Some are only used by one consumer (e.g. `RunAtomicAppenderTest` is used by both DuckDB and Pebble, but the package is only imported by metaengine/*). That's still a 5-helper public API for a test-helper package. Could be reduced by 1-2 if the test-file-local helpers stay local, but the public API is fine for now.

4. **I created `metaengine/keycodec` AND `metaengine/enginetest` as sub-packages.** The api-stability check (`cmd/api-stability/main.go`) only watches separate `go.mod` modules, not sub-packages of an existing module — so I didn't need to add them to that list. **BUT** I should have verified by reading the api-stability source. I noticed this only after-the-fact via grep.

5. **The gopls `gopls go mod tidy` errors after my changes are mostly stale-cache phantom errors.** gopls caches the workspace state; new sub-packages (`enginetest`, `keycodec`) confuse it until the LSP restarts. I should have run `lsp_restart gopls` after creating new packages instead of staring at 60+ phantom errors. **This bit me several times during the run.**

6. **I never ran `nix run .#verify` or the full test suite.** I verified each module builds in isolation, but the user asked for ZERO clones — not ZERO verified-untouched tests. I should run the full verify gate before declaring done. Currently this is **NOT verified** in the strongest sense.

7. **The pgtestcontainer module I created does NOT follow the existing 77-module naming convention perfectly.** It's at `testutil/pgtestcontainer/go.mod`. That fits, but the `benchkit/pg_testcontainer_test.go` and `storage/pg_testcontainer_test.go` files were NOT migrated to the new helper — they still have their own duplicated setup. Migrating them would have killed additional clones, but art-dupl didn't surface them as clones (different module boundaries).

8. **I used `var foo = keycodec.Foo` aliases for the engine helpers.** This works, but it's a bit of an indirection. Future maintainers seeing `mapKey := keycodec.MapKey` at package init time may be confused. The alternative was inlining all 58 call sites, which is invasive. The var alias is the lesser of two evils but not the cleanest.

9. **The `RunWatcherReplayTest` generic uses reflection (`idOf`) to extract the ID from the value.** This was necessary because DuckDB and Postgres return `map[string]any` while the typed payload has a named `ID` field. The reflection path is correct but adds complexity. A cleaner design would have the engine return its value type, but that's a much bigger refactor.

10. **I did not update the `art-dupl` baseline file** (`.art-dupl-baseline.json`) which the project's `nix run .#check-duplication` gate likely watches. Need to verify whether the gate requires baseline regen.

11. **I did not run `nix fmt` after all the changes.** The `gofumpt -w` + `goimports -w` step is part of the project workflow. The new files (`pgtestcontainer.go`, `keycodec.go`, `enginetest.go`) may not be fully formatted.

12. **The `metaengine/keycodec/keycodec.go` is the single source of truth for key shapes.** If a future engine (e.g. `metaengine/irohengine/quic`?) needs the same key shapes, it should also import this package. I did not add a reference comment in the irohengine or other engine modules. Documentation debt.

## What Got Fucked Up

1. **I tried to `sed -i` the duckdb pushdown file and produced garbage.** The output file had lines like `_ = newDuckDBPushdown(t) / eng, _ := newDuckDBPushdown(t) / eng = eng` stacked on each other. I had to rewrite the whole file with `write`. **Lesson:** never sed-rewrite generated content; always use `multiedit` for in-place edits or `write` for full rewrites.

2. **I confused myself mid-run on the `RunWatcherReplayTest` Store type.** First I wrote it as `metaengine.Store` (interface), but `metaengine.Store` is a struct. I had to change to `*metaengine.Store`. The error came from gopls ("cannot use store as metaengine.Store value in return statement") which was confusing because of the type name. **Lesson:** when a type is named `Store` and is actually a struct, double-check the existing Planner return type before writing helpers.

3. **I forgot to remove unused import `"fmt"` from pebbleengine/engine.go** when refactoring the helpers. The compile caught it but I had to re-add it because of an unused `fmt.Sscanf` call. **Lesson:** after replacing wrappers with `var` aliases, scan for now-orphaned imports — but ALSO scan for indirect references that may not be obvious.

4. **I created `metaengine/enginetest` with a near-duplicate `RunAtomicAppenderTest` AND `RunStreamLogBackendTest` in the same change.** They could have been one combined function, but separating them by behavior (snapshot vs version-conflict semantics) is more readable. Acceptable tradeoff.

5. **The `cmd/api-stability/main.go` modules list may now be incomplete.** I added two new sub-packages (`keycodec`, `enginetest`) to the metaengine module, but these are sub-packages, not new modules, so the `modules` list doesn't need updating. However, I should have confirmed this with a test rather than assuming.

## Up To 50 Things To Get Done Next

In rough priority order (most-impact-first):

1. **Group #2 fix** (metaengine/bench) — extract `RunEngineParityTest` helper, kill the last data-driven test clone
2. **Group #4 decision** (cmd/cqrs-lint/explain.go) — refactor with `writeSection` helper OR document the acceptance in `.art-dupl-baseline.json`
3. **Run `nix run .#verify`** — full test + lint + race + coverage gate to confirm no regressions
4. **Update `.art-dupl-baseline.json`** if the project has one (check `nix run .#check-duplication`)
5. **Run `nix fmt`** — gofumpt + goimports on the new files
6. **Migrate `benchkit/pg_testcontainer_test.go` and `storage/pg_testcontainer_test.go`** to the new `pgtestcontainer` helper (art-dupl didn't surface them but they're still duplicated)
7. **Consider migrating the SQLite `metaengine/sqliteengine` pushdown tests** to the shared `enginetest.RunPushdownTest` helper
8. **Add `metaengine/enginetest` examples to the `SKILL.md`** for the go-cqrs-lite consumer guide
9. **Document the keycodec package** in `AGENTS.md` (Monorepo Structure section)
10. **Document the pgtestcontainer module** in `AGENTS.md` (Monorepo Structure section)
11. **Verify api-stability golden** — does it scan sub-packages? If not, this is fine. If yes, regenerate
12. **Add a `TestEngines_parity_against_pebble` integration test** that runs the bench/duckdb parity helpers against a Pebble engine, validating the keycodec aliases work
13. **Add a comment to `metaengine/enginetest/enginetest.go`** explaining the package's scope: "shared test scenarios for engine backends"
14. **Add a comment to `metaengine/keycodec/keycodec.go`** explaining the on-disk key layout contract: "any future LSM engine MUST match this key shape"
15. **Consider extracting `quicCluster` setupTwoNodeQuic** to a separate test helpers file (`quic_helpers_test.go`) so it's not buried in `transport_test.go`
16. **Refactor `metaengine/bench/bench_*.go`** to use the benchkit framework (if it isn't already)
17. **Add `dedup_test.go` to the `pgtestcontainer` module** to verify the helper works without a real DSN (skip path)
18. **Consider adding a `metaengine/enginetest.RunStreamLogBackendTest` for the journal-backpressure scenario** (5x10K events)
19. **Add coverage for the keycodec package** in `metaengine/keycodec/keycodec_test.go` (currently 0% coverage)
20. **Add coverage for the pgtestcontainer package** in `testutil/pgtestcontainer/pgtestcontainer_test.go`
21. **Run `go mod tidy -e`** on the entire workspace to suppress warnings from the eventtest nested module
22. **Verify all tag-pinned go.mod files** still build with the new sub-packages
23. **Consider whether `RunWatcherReplayTest` should be split** into a per-engine variant (since the seqTimeout parameter is engine-specific)
24. **Check if the `metaengine/bench` test helpers (Group #2) could share the `enginetest` package** instead of a new bench-local helper
25. **Update the storage module's `pg_testcontainer_test.go`** to use the new `pgtestcontainer` module (art-dupl threshold didn't catch it but it's structurally identical)
26. **Add a `metaengine/enginetest/parallel_test.go`** that runs all helpers in parallel to surface race conditions
27. **Verify the new modules work with `GOWORK=off`** (consumer-style isolated builds)
28. **Document the `nix run .#check-duplication` workflow** if the baseline file is updated
29. **Add a paragraph to `AGENTS.md` explaining the new shared test helper pattern** (enginetest, pgtestcontainer, keycodec)
30. **Re-verify that the `nix run .#build` task succeeds for all 79+ modules** including the new pgtestcontainer
31. **Consider whether `metaengine/enginetest` should be its own module** (like pgtestcontainer) or stay a sub-package
32. **Add a `metaengine/keycodec/README.md`** explaining the key format and migration path
33. **Re-run the linter to catch any issues with the new code** (`nix run .#lint`)
34. **Check that gopls restarts correctly** so the phantom errors stop appearing
35. **Look for additional clones at lower thresholds** (`-t 5`, `-t 3`) — the user wanted ZERO at threshold 7, but there are likely more at stricter thresholds
36. **Consider whether `RunScanBackendTest` should accept a filter/sort factory** instead of hard-coding the 4 scenarios — makes it reusable for other engines
37. **Document the `RunPushdownTest` API** with a docstring example (currently minimal)
38. **Add an end-to-end example** in the `getting-started` example module that uses the new `pgtestcontainer` helper
39. **Make `idOf()` more robust** — currently it handles string and Stringer but not pointer-to-Stringer, custom Marshaller, etc.
40. **Consider adding a `metaengine/testdata` golden file** for the keycodec key shapes (regression guard against accidental format changes)
41. **Check that `nix run .#vulncheck` still passes** with the new dependencies
42. **Add `metaengine/enginetest` to the `cmd/cqrs-lint` module catalog** if it tracks test helpers
43. **Verify the API-stability check passes** (regenerate golden if needed)
44. **Run `go test -count=3 -race ./middleware/...` to verify the new flightrecorder test helper doesn't introduce flakes**
45. **Check that the `var mapKey = keycodec.MapKey` aliases don't cause a measurable startup-time regression** (vet the binary for any non-inlined calls)
46. **Add a `metaengine/enginetest/README.md`** with one example per helper
47. **Add `TestFlightRecorder_NewFRTestRecorder` test** to verify the helper itself
48. **Verify the `TestFlightRecorder_PreservesError` and `TestFlightRecorder_LoggerOnError` tests** (which I left unmodified) still work after the helper change
49. **Consider whether the `metaengine/keycodec` package should expose a `Format` method** for debugging (print keys in human-readable form)
50. **Add a CI test step** that runs `art-dupl --type-aware -t 7` and fails if any new clones are introduced

## My Three Questions

1. **Group #4 (cmd/cqrs-lint/explain.go): refactor or accept?** These are 25-line documentation literals that art-dupl sees as duplicated. Extracting a `writeSection` helper might obscure the section content for maintainers. I can't decide between (a) refactor with a slice/struct of section lines, (b) exclude via `.art-dupl-baseline.json`, or (c) accept and add a one-line comment explaining the duplication is intentional. **What do you want?**

2. **Group #2 (metaengine/bench): I had the diagnosis and was about to extract `RunEngineParityTest` when you prompted for status. Should I just continue with that fix in the next turn, or do you want to discuss the API surface first?** (The test bodies are 88 lines, two engines differ only in variable name, helper would land cleanly.)

3. **Should I run `nix run .#verify` (the 3-4 minute full test+lint+race gate) before declaring "zero clones" done?** I have not run it. There's a real risk that one of my refactors broke a build or test in a module I didn't check, and a "GREEN" claim based on the partial verification I did is a "stale GREEN" anti-pattern. I'd rather run it than claim victory without it.
