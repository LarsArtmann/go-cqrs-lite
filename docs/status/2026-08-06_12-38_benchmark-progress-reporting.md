# Status Report: 2026-08-06 12:38 — Benchmark Progress Reporting

## Context

User ran `./cqrs-bench compare --profile stress --format benchstat` and asked why there's no real-time progress output. The benchmark ran for minutes with zero feedback. Only `--soak` mode had progress reporting; `run`, `compare`, and `sweep` were completely silent.

---

## A) FULLY DONE

1. **`benchkit/progress.go`** (new file, 100 lines) — `progressReporter` with:
   - Heartbeat goroutine using `time.Ticker` that prints current phase + elapsed time every N seconds
   - Thread-safe (RWMutex on phase/num/start fields)
   - Nil-safe: all methods are no-ops when `ProgressWriter` is nil or reporter is nil
   - `beginPhase()` / `endPhase()` / `beat()` / `start()` / `stop()` lifecycle

2. **`benchkit/runner.go`** — Wired into the benchmark execution:
   - `progress` field added to `runner` struct
   - Reporter created in `run()` after `setup()`, started/stopped via defer
   - `runPhases()` emits `beginPhase`/`endPhase` around each phase execution
   - Extracted `phaseSteps()` method + `phaseStep` type (shared between `runPhases` and `countActivePhases`)
   - `countActivePhases()` counts non-skipped phases for denominator in progress output

3. **`benchkit/benchkit.go`** — Config extended:
   - `ProgressWriter io.Writer` (json:"-") — nil = no progress
   - `ProgressInterval time.Duration` — defaults to 5s in `newProgressReporter`
   - `io` import added

4. **`cmd/cqrs-bench/flags.go`** — `--progress` flag:
   - `progress *time.Duration` field on `benchFlags` (default: 5s, 0 disables)
   - Registered as `fs.Duration("progress", 5*time.Second, ...)`

5. **`cmd/cqrs-bench/main.go`** — CLI wiring:
   - `applyProgress()` helper: sets `ProgressWriter` to `os.Stderr` when interval > 0
   - Called in both `runCmd` and `compareCmd` after config construction

6. **`cmd/cqrs-bench/factory.go`** — Backend-level progress:
   - `compareWithDiskPaths` iterates backends in **sorted order** (deterministic output)
   - Prints `[1/3] backend: memory` header before each backend when progress enabled

7. **Testing completed:**
   - `go build` passes for benchkit + cqrs-bench
   - `go test ./benchkit/... -count=1` — **PASS** (46.6s)
   - `go test ./benchkit/... -count=1 -race -run "TestRun|TestCompare|TestRepeat"` — **PASS** (54.8s)
   - Functional tests: dev profile (fast phases), medium profile (heartbeat fires during 3.2s write phase), compare with 2 backends (headers + phase progress), `--progress 0` (disables cleanly)

8. **Auto-commit daemon committed the work** across 5 commits (3f8ce497 → 444c7a5f, 12:28-12:34)

---

## B) PARTIALLY DONE

1. **`sweepCmd` progress** — `applyProgress` was NOT wired into the `sweep` subcommand. Sweep runs benchmarks sequentially across parameter values but gets no progress output.

2. **`printUsage` help text** — The `--progress` flag is registered but NOT documented in the usage/help text or examples. Users won't discover it from `cqrs-bench help`.

3. **API-stability golden regen** — The AGENTS.md mandates: "API-surface changes require golden regen in the same edit". Adding `ProgressWriter` + `ProgressInterval` to the public `Config` struct changes benchkit's API surface. The golden was NOT regenerated. The `cmd/api-stability` tool also failed to run (needs the `goexperiment.jsonv2` build tag which it doesn't apply).

---

## C) NOT STARTED

1. **Unit tests for `progress.go`** — Zero tests written for the new progress reporter. It has a goroutine, ticker, mutex, and lifecycle methods. No test verifies:
   - Heartbeat fires at correct interval
   - Phase transitions produce correct output format
   - Nil-safety (no panic with nil writer)
   - `stop()` is idempotent
   - Concurrent `beginPhase` / `beat` don't race

2. **Throughput-in-heartbeat** — The heartbeat only shows `elapsed time`. For a stress profile writing 5M events, showing events/sec or events-processed would be far more useful. This requires threading a progress callback into each phase's inner write loop (`runConcurrent`, etc.) — a much bigger change that was not attempted.

3. **Estimated time remaining** — No ETA calculation. Would need historical phase-duration data or event-count-based estimation.

4. **Repeat mode progress** — When `--repeat N` is used, each repeat gets its own progress reporter, but there's no indication of which repeat iteration is running (e.g., "repeat 2/5").

---

## D) TOTALLY FUCKED UP

1. **Auto-commit daemon broke the build (AGAIN)** — The daemon committed my progress work, then proceeded to rewrite the entire `cmd/cqrs-bench` CLI from Go's stdlib `flag` package to `github.com/larsartmann/cmdguard` struct-tag-based flag parsing. This is a 952-line diff across `flags.go` and `main.go`. The rewrite has a **type mismatch bug**: handler functions use `flags RunFlags` (value) but `cmdguard.NewCommand` expects `flags *RunFlags` (pointer). **The build is currently broken:**

   ```
   cmd/cqrs-bench/main.go:85:3: type func(...flags RunFlags) does not match
   inferred type func(...flags *RunFlags)
   ```

   This mirrors the exact "Auto-commit daemon can break the build" anti-pattern documented in AGENTS.md. The daemon shipped a real migration (cmdguard adoption) but broke the generic type signatures. The `benchkit` module (my actual work) compiles and tests fine — only the daemon's CLI rewrite is broken.

2. **Auto-commit daemon changed cqrs-lint too** — Uncommitted changes to `cmd/cqrs-lint/` (README.md, catalog_extra.go, meta_test.go, register.go) that I did not author. Unknown if these compile.

3. **My `--progress` flag survived the daemon rewrite** — The daemon's cmdguard migration preserved the Progress field (as `Progress cmdguard.Duration` with `default:"5s"`), so the feature itself is intact in the daemon's version. But it can't be tested because the build doesn't compile.

---

## E) WHAT WE SHOULD IMPROVE

### Immediate (this session's work)

1. **Fix the daemon's cmdguard build break** — Change `flags RunFlags` → `flags *RunFlags` in all 3 handler signatures (or the reverse, depending on cmdguard's API)
2. **Wire `applyProgress` into `sweepCmd`** — Currently sweep runs silently
3. **Add `--progress` to `printUsage` help text** — Flag exists but is undiscoverable
4. **Write unit tests for `progress.go`** — New code with goroutines/mutexes has zero coverage
5. **Regen api-stability golden** — Public API changed (`Config.ProgressWriter`, `Config.ProgressInterval`)

### Architectural (progress reporting design)

6. **Throughput in heartbeat** — Show `events/s` or `events processed / total` during write phases. Requires a progress callback threaded into `runConcurrent` and each phase's inner loop.
7. **Phase ETA** — Estimate remaining time based on events processed so far + rate
8. **Repeat iteration indicator** — Show "repeat 2/5" when `--repeat` is used
9. **Adaptive interval** — Short phases (sub-second) don't need heartbeats. Only start the ticker if a phase exceeds some threshold (e.g., 2x the interval).
10. **Progress for soak mode unification** — Soak has its own `ProgressWriter` in `SoakConfig`. Could unify with the new system.
11. **Structured progress** — Instead of free-form text, emit JSON progress events that tools/UIs can consume
12. **Colored output** — Use terminal colors for phase status (green=done, yellow=running, red=failed) when stderr is a TTY

### Process / CI

13. **CI gate for daemon commits** — The auto-commit daemon breaks the build regularly. A post-commit `go build ./...` gate that reverts on failure would prevent broken-commit propagation.
14. **api-stability tool needs `goexperiment.jsonv2` tag** — It currently fails to run outside Nix because it doesn't pass the build tag. Should be fixed in `cmd/api-stability/main.go`.
15. **cmdguard migration should be a reviewable PR** — Not an auto-committed blob

---

## F) Next 50 Things to Get Done

### Fix the build (P0)

1. Fix cmdguard pointer/value mismatch in `cmd/cqrs-bench/main.go` (3 handler signatures)
2. Verify `cmd/cqrs-lint/` changes from daemon compile
3. Run `go build -tags "goexperiment.jsonv2" ./...` to confirm full build
4. Run `go test -tags "goexperiment.jsonv2" ./cmd/cqrs-bench/... -count=1`

### Progress feature hardening (P1)

5. Write `progress_test.go` — test heartbeat interval, phase transitions, nil-safety
6. Wire `applyProgress` into `sweepCmd`
7. Add `--progress` to `printUsage` examples and flag list
8. Add progress test to benchkit_test.go (`TestProgressReporter`)
9. Test progress with `--repeat` mode
10. Test progress output doesn't pollute stdout (always stderr)
11. Test progress with `--format json --output file.json` (clean stdout/file)

### API stability (P1)

12. Fix `cmd/api-stability` to pass `goexperiment.jsonv2` build tag
13. Regen api-stability golden for benchkit Config change
14. Verify `TestEveryGoModDirIsInModulesList` still passes

### Progress richness (P2)

15. Add events-processed counter to heartbeat (thread callback through `runConcurrent`)
16. Add throughput (events/s) to heartbeat output
17. Add phase ETA based on events processed / rate
18. Show repeat iteration number ("repeat 2/5") in progress
19. Skip heartbeat for phases under 2x interval (adaptive)
20. Add progress to warmup phase
21. Add progress to setup phase (factory call, stream ID generation for large profiles)
22. Add progress to recovery phase
23. Add progress to durability phase

### CLI polish (P2)

24. Review cmdguard migration — is it actually better than stdlib flag?
25. Add `--quiet` flag (suppresses progress even when stderr is a TTY)
26. Add `--verbose` flag (enables per-event logging in phases)
27. Update README/help text for all new flags
28. Add progress format option (text/json/none)

### Daemon defense (P2)

29. Add `.git/hooks/post-commit` that runs `go build ./...` and reverts on failure
30. Document daemon break pattern in AGENTS.md more prominently
31. Add CI check that validates HEAD always builds

### Soak mode (P3)

32. Unify soak progress with the new progressReporter
33. Add soak-specific metrics to heartbeat (iterations, leak trend)
34. Make soak `ReportInterval` default respect `--progress` flag

### Library API (P3)

35. Document `ProgressWriter` / `ProgressInterval` in Config godoc
36. Add `WithProgress(writer, interval)` option helper
37. Expose `ProgressReporter` as a configurable interface (custom formatters)
38. Add progress callback channel for programmatic consumers

### Testing (P3)

39. Benchmark test for progress reporter overhead (should be negligible)
40. Race test: concurrent beginPhase + beat + endPhase
41. Test: progress with cancelled context (heartbeat stops cleanly)
42. Test: progress with error mid-phase (endPhase still fires)

### Documentation (P3)

43. Update AGENTS.md with `--progress` flag in Quick Reference
44. Update SKILL.md if benchkit CLI is documented there
45. Add progress output example to cqrs-bench README
46. Document the progress output format (stderr, not stdout)

### Metaengine / Other (P4)

47. Verify metaengine phase gets progress (it's in phaseSteps)
48. Verify raw sink phase gets progress (heavy: pre-builds all events)
49. Consider progress for `benchkit.Compare` (library-level, not just CLI)
50. Consider progress for sweep results output

---

## G) Questions I Cannot Answer Myself

1. **Should I fix the daemon's cmdguard migration or revert it?** The daemon rewrote the entire CLI from stdlib `flag` to `cmdguard`. It's a 952-line change I didn't author. It breaks the build. AGENTS.md says "NEVER revert changes you didn't author — investigate first, ASK before touching it." But the build is broken. Do you want me to fix the type mismatch (3 lines) and keep the migration, or revert entirely?

2. **Should the progress heartbeat show throughput (events/s)?** This requires threading a counter/callback through `runConcurrent` into every phase's inner loop — a significant refactor touching 10+ phase files. Is that worth it, or is elapsed-time-only sufficient?

3. **Is the cmdguard migration something you directed the daemon to do?** If so, I should fix it. If the daemon did this autonomously, it may be unwanted churn.
