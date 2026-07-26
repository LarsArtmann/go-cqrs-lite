# Status Report: benchkit Module Implementation

**Date:** 2026-07-23 17:10
**Session:** Initial benchkit implementation from design report
**Status:** ~~MVP functional, several gaps and dead-code issues~~ Dead-code bugs fixed in [bugfix session](2026-07-24_05-59_benchkit-bugfix-session-status.md) (commits `19f540ec`–`9ac2cc52`): `Config.Codec` wired, `Config.Duration` enforced, `ReadRatio` implemented, `BenchPayload` typed. ~~Warmup store pollution and `estimateJSONSize` accuracy remain open.~~ Both resolved: warmup isolation in the [critical-fixes session](2026-07-24_15-13_benchkit-critical-fixes-status.md), CBOR-aware probe-encode padding in the [completeness session](2026-07-24_16-45_benchkit-completeness-session-status.md). Test count is now 55 (was 22 at time of writing).

---

## a) FULLY DONE

| Item                       | Details                                                                                      |
| -------------------------- | -------------------------------------------------------------------------------------------- |
| `benchkit/` module created | go.mod, go.sum, wired into go.work                                                           |
| Core types                 | `Config`, `Result`, `LatencyStats`, `ResourceStats`, `DiskStats`, `Factory` in benchkit.go   |
| LatencyCollector           | Sorted-slice + reservoir sampling (10K cap), thread-safe, tested                             |
| Resource sampling          | Peak heap via 100ms polling goroutine, baseline/after deltas                                 |
| CPU measurement            | `/proc/self/stat` user+sys time (Linux only)                                                 |
| Synthetic generator        | Seeded PCG, deterministic, configurable payload size                                         |
| 7 named profiles           | Dev, Small, Medium, Large, Stress, WriteHeavy, ReadHeavy                                     |
| 8-phase runner             | setup → warmup → write → read → readmodel → projection → durability → teardown               |
| Concurrent worker pool     | Channel-based, cancel-on-error, proper WaitGroup                                             |
| `Run()` API                | Single-backend benchmark, returns `*Result`                                                  |
| `Compare()` API            | Multi-backend, handles factory failures gracefully                                           |
| Text report                | Human-readable with latency percentiles, throughput, memory, disk                            |
| JSON report                | JSON v2 with custom `time.Duration` marshaler, indented                                      |
| Markdown report            | Comparison table format                                                                      |
| 22 tests passing           | Unit tests (collector, generator, profiles) + integration (memory, sqlite, compare, reports) |
| `cmd/cqrs-bench` CLI       | `run` + `compare` subcommands, memory/sqlite/pebble backends                                 |
| README.md                  | benchkit/README.md with quick start, profiles table, metrics list                            |
| AGENTS.md updated          | Module list, directory structure, test command                                               |

**Evidence:** `go test ./benchkit/... -tags "goexperiment.jsonv2" -count=1` → 22 PASS, 0 FAIL

---

## b) PARTIALLY DONE

| Item                   | What exists                                                                | What's missing                                                                                               |
| ---------------------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `Config.Codec` field   | Field defined, stored in runner as `r.codec`, `codecName()` computes label | **NEVER passed to `event.New()`** — events always use `event.DefaultCodec`. The field is dead code.          |
| `Profile.ReadRatio`    | Defined on all 7 profiles with sensible values (0.1–0.8)                   | **Never read by the runner.** Write and read phases are sequential, not interleaved by ratio.                |
| `Profile.BatchSize`    | Defined, used in write phase (`min(profile.BatchSize, remaining)`)         | Read phase doesn't vary based on it (acceptable, but undocumented).                                          |
| CLI in workspace       | `cmd/cqrs-bench` builds with `GOWORK=off` + replace directives             | **Not in `go.work`** — workspace resolution fails for unpublished modules. `nix run .#build` won't build it. |
| Pebble backend support | Factory works in CLI (`pebble.New()` → `.Bundle`)                          | No pebble tests. No `Metrics()` integration (LSM-tree health). No `Flush()` durability measurement.          |
| Disk measurement       | `DiskPath` config + `filepath.Walk` size summing                           | Manual path, not auto-detected via `interface{ DiskSize() int64 }` as designed.                              |
| Projection benchmark   | `projectionhost.Host` with counting projection, lag/events collected       | Creates a throwaway projection per run; consumer can't register custom projections.                          |

---

## c) NOT STARTED

| Phase (from design report)     | Description                                                        | Status      |
| ------------------------------ | ------------------------------------------------------------------ | ----------- |
| Phase 2: Durability            | `synchronous=FULL` vs `OFF` comparison, `Flush()` timing           | Not started |
| Phase 6: Production replay     | JSON Lines export/import, `ReplaySource`                           | Not started |
| Phase 7: benchtest suite       | `benchtest.RunSuite(b, factory)` mirroring contracttest            | Not started |
| Multi-DB split benchmarking    | Measure contention reduction from EventDB/QueryDB/ViewDB split     | Not started |
| Pebble `Metrics()` integration | LSM-tree health (block cache hit rate, compaction count)           | Not started |
| SQLite PRAGMA metrics          | `page_count`, `page_size`, WAL file size                           | Not started |
| Postgres/Turso backends        | `pg_database_size()`, connection pool stats                        | Not started |
| CI workflow                    | Regression detection with baseline comparison                      | Not started |
| Nix flake integration          | `nix run .#bench`, `nix run .#bench-compare`                       | Not started |
| ADR                            | Architecture decision record for benchkit                          | Not started |
| Crush skill update             | `.agents/skills/go-cqrs-lite/SKILL.md` doesn't mention benchkit    | Not started |
| CLI `report` subcommand        | Generate reports from saved JSON files                             | Not started |
| CLI `replay` subcommand        | Replay production data dump                                        | Not started |
| CLI `--version` flag           | Standard version output                                            | Not started |
| `cmd/cqrs-bench` tests         | No tests for the CLI itself                                        | Not started |
| Connection to `stack/bench`    | benchkit doesn't reference or complement existing micro-benchmarks | Not started |

---

## d) TOTALLY FUCKED UP

### 1. `Config.Codec` is dead code

The field exists, is documented, is stored in the runner (`r.codec`), and a codec name is computed (`r.codecName`). But **`r.codec` is never passed to `event.New()`**. Every event uses `event.DefaultCodec` (CBOR). The Config.Codec field gives users a false sense of control.

**Location:** `benchkit/runner.go:155` stores `r.codecName`, but `runner.go:268` and `phases.go:85` call `event.New()` without `event.WithCodec(r.codec)`.

### 2. `Profile.ReadRatio` is dead code

All 7 profiles define `ReadRatio` values (0.1–0.8). The runner **never reads this field**. The write and read phases are fully sequential — there is no interleaving. `ProfileWriteHeavy` and `ProfileReadHeavy` are indistinguishable in practice despite having ReadRatio 0.1 vs 0.8.

### 3. `Config.Duration` is dead code

The field is defined as "caps the wall-clock time. Zero means run to completion." The runner **never checks this field** during execution. A user setting `Duration: 30*time.Second` gets no timeout enforcement.

### 4. Generator truncation produces invalid JSON

`generator.go:57`: `return data[:g.size]` slices a JSON byte array mid-token when `len(data) > g.size`. This produces syntactically invalid JSON that will fail on any JSON parser. For `PayloadSize: 64` (below the minimum valid payload size), every generated payload is corrupt.

### 5. Warmup pollutes the event store

The warmup phase (`runner.go:258`) writes events to a throwaway aggregate in the **same store** that is then benchmarked. These extra events inflate the journal scan times (ReadAll/ReadFrom) and the aggregate count.

### 6. `cmd/cqrs-bench` cannot be in go.work

The CLI module uses replace directives + `GOWORK=off` because benchkit isn't published. This means:

- `go work sync` doesn't know about it
- `nix run .#build` won't compile it
- The replace directive list is 25 entries — fragile and high-maintenance

### 7. No `goexperiment.jsonv2` tag documentation

benchkit imports `encoding/json/v2` and `encoding/json/jsontext` directly. Building without `-tags "goexperiment.jsonv2"` fails. This is **not documented** in go.mod build comments, README, or AGENTS.md test command. A consumer will hit a cryptic "build constraints exclude all Go files" error.

---

## e) WHAT WE SHOULD IMPROVE

### Code quality

1. **Replace manual string helpers** — `report.go` has hand-rolled `sortStrings` (insertion sort), `insertCommas`, and `truncate` functions. Use `sort.Strings`, `golang.org/x/exp/slices` or humanize library instead.
2. **Add `errors.go`** — The module wraps errors with `fmt.Errorf` but doesn't define sentinel errors or use `errorfamily` classification, breaking the project convention.
3. **Add `doc.go`** — Package-level documentation should be in a dedicated file, matching the project pattern (every module has `doc.go`).
4. **Generator should produce valid JSON at any size** — Pad with a `_padding` string field, not byte slicing.
5. **Use `sort.Slice` or `slices.SortFunc`** instead of hand-rolled insertion sort for result keys.
6. **The `cpuTime()` function silently returns 0 on non-Linux** — should log or return a distinguishable sentinel.

### Architecture

7. **Wire `Config.Codec` to `event.WithCodec()`** in event creation calls.
8. **Implement ReadRatio** as a mixed read/write phase, or remove the field and document that phases are sequential.
9. **Enforce `Config.Duration`** with a context deadline inside the runner.
10. **Detect disk size via interface** (`interface{ DiskSize() int64 }`) instead of manual `DiskPath` string.
11. **Integrate Pebble `Metrics()`** — the design report specifically called this out as a backend-specific metric.
12. **Add SQLite PRAGMA queries** for `page_count * page_size` and WAL file size.

### Testing

13. **Add pebble tests** — the third local backend is untested.
14. **Add CLI tests** — `cmd/cqrs-bench` has zero tests.
15. **Test with CBOR codec** — verify the codec field actually works once wired.
16. **Add race detector tests** — `go test -race` to verify concurrent phases.
17. **Test generator determinism across codec changes** — ensure seed stability.
18. **Add negative tests** — factory returning nil Bundle, nil EventSink, closed store.

### Integration

19. **Resolve go.work inclusion** — either publish benchkit or add it to go.work with the workspace understanding the local path (the workspace DOES list `./benchkit`, the issue is only cmd/cqrs-bench).
20. **Add benchkit to flake.nix** — build and test targets.
21. **Add ADR** — document why benchkit exists, its factory pattern, and relationship to contracttest.
22. **Update the Crush skill** — `.agents/skills/go-cqrs-lite/SKILL.md` should list benchkit in the module decision matrix.
23. **Add to `.golangci.yml` depguard** — if depguard enforces internal-only imports.
24. **Add `nix fmt` compliance** — code was never formatted with the project's golines (max-len: 120).

### Design gaps

25. **No backend-specific metrics extension point** — the design proposed per-backend metrics (Pebble LSM, SQLite PRAGMAs, Postgres pool stats). There's no mechanism for backends to inject their metrics into the Result.
26. **No baseline comparison** — the design proposed `cqrs-bench compare --baseline baseline.json --threshold 20` for regression detection. Not implemented.
27. **No custom workload support** — the design proposed `Workload` struct with `Setup`/`WriteOps`/`ReadOps`/`Teardown` callbacks for domain-specific benchmarks. The runner is hardcoded to the built-in phases.
28. **Projection phase is fire-and-forget** — creates a Host, starts, immediately stops. No wait for catch-up. No configurable projections.

---

## f) Up to 50 Things to Do Next

### Critical (broken/fraudulent behavior)

1. Wire `Config.Codec` to `event.New()` via `event.WithCodec(r.codec)` — currently dead code
2. Implement `Profile.ReadRatio` as a mixed read/write phase, OR remove the field entirely
3. Enforce `Config.Duration` with a context deadline inside the runner
4. Fix generator truncation — pad with JSON field, not byte slicing
5. Fix warmup pollution — use a separate aggregate that doesn't affect journal scans, or clean up after
6. Document the `goexperiment.jsonv2` build tag requirement in benchkit README and go.mod

### High priority (completeness)

7. Add `benchkit/doc.go` with package documentation
8. Add `benchkit/errors.go` with sentinel errors using `errorfamily`
9. Replace manual `sortStrings` with `sort.Strings` in report.go
10. Replace manual `insertCommas` with a humanize library or stdlib
11. Run `nix fmt` on all benchkit files
12. Run `nix run .#lint` (golangci-lint) and fix all findings
13. Add pebble backend tests in benchkit_test.go
14. Add `-race` flag to test runs and fix any data races
15. Resolve cmd/cqrs-bench go.work inclusion (or document the GOWORK=off pattern)
16. Add disk size auto-detection via `interface{ DiskSize() int64 }` on Bundle

### Medium priority (design report phases)

17. Implement Phase 2: durability comparison (Save vs Save+Flush delta)
18. Integrate Pebble `Metrics()` into Result as backend-specific metrics
19. Add SQLite PRAGMA metrics (page_count, WAL size)
20. Implement Phase 7: `benchtest.RunSuite(b, factory)` mirroring contracttest
21. Add custom workload support (`Workload` struct with callbacks)
22. Add baseline comparison for CI regression detection
23. Write ADR for benchkit design decisions
24. Update Crush skill (SKILL.md) with benchkit module entry
25. Add benchkit to flake.nix build/test/lint targets

### Lower priority (polish and future)

26. Add CLI `report` subcommand (generate from saved JSON)
27. Add CLI `--version` flag
28. Add CLI tests (at least smoke test for each subcommand)
29. Implement Phase 6: production data replay (JSON Lines export/import)
30. Add Postgres backend support to CLI (with graceful skip)
31. Add Turso backend support to CLI
32. Add multi-DB split benchmarking mode
33. Add `--tmpfs` flag for CI environments
34. Add `WithReplayTimeout` equivalent for benchmarks that might hang
35. Add connection pool stats for SQL backends
36. Add per-projection lag breakdown in Result
37. Add GC pause time tracking (MemStats.PauseTotalNs delta)
38. Add RSS (resident set size) measurement beyond heap
39. Add `eventsPerMB` storage efficiency metric
40. Add markdown report for single-result (not just comparison)
41. Add CSV output format for spreadsheet import
42. Add trend tracking (append to historical JSON, compute deltas)
43. Add HDR histogram as optional integration for >1M sample precision
44. Add custom decider benchmark support (`DeciderBenchmark[State]` generic)
45. Add snapshot strategy benchmarking (EveryNEvents, ReadPressure)
46. Add codec comparison mode (JSON vs CBOR vs CBORCompact)
47. Add configurable event types (currently hardcoded to `bench.event`)
48. Add middleware benchmarking (signing, encryption, OTel overhead)
49. Add `cqrs-bench doctor` command (like cqrs-lint) showing detected capabilities
50. Add benchmark result schema versioning for forward compatibility

---

## g) Questions

### 1. Should `cmd/cqrs-bench` be in `go.work` or stay separate?

The workspace fails to resolve `benchkit/v4` when cmd/cqrs-bench is in go.work because it tries to fetch the module from the remote (which doesn't have it). Options:

- **A:** Keep `GOWORK=off` + replace directives (current approach, 25 replace entries)
- **B:** Add cmd/cqrs-bench to go.work and let the workspace's `./benchkit` entry resolve it (requires understanding why this fails — it works for stack/bench which has the same pattern)
- **C:** Merge benchkit into the `stack` module as a sub-package (eliminates the cross-module dependency issue entirely)

I could not figure out why option B fails when stack/bench (which depends on stack/v4 the same way) works fine.

### 2. Should benchkit use the project's `errorfamily` error classification?

The project convention is to use `errorfamily.NewRejection(...)`, `errorfamily.WrapInfrastructure(...)` etc. benchkit currently uses plain `fmt.Errorf`. Should I:

- Add `errorfamily` as a dependency and classify benchmark errors (Rejection for bad config, Infrastructure for factory failures)?
- Or keep plain errors since benchkit is a tool, not a library consumers build on?

### 3. Should the dead `Config.Codec` / `Profile.ReadRatio` / `Config.Duration` fields be fixed or removed?

These three fields exist in the public API but do nothing. Options:

- **A:** Implement them properly (wire codec to events, implement mixed read/write phase, enforce duration timeout)
- **B:** Remove them from the public API for now and add them back when implemented
- **C:** Leave them as-is with `// TODO` comments (current state, but misleading to consumers)

This is a design decision about whether benchkit v1 should be minimal-but-honest or feature-complete-but-rough.

---

## Resolution (2026-07-26)

**Option B chosen — shipped as `benchkit/v4` (minimal-but-honest).**

All stub phases were properly implemented or removed. The published module
has: 7 named workload profiles + analytical profile, 9-phase runner
(setup → warmup → write → read → readmodel → projection → durability →
rawsink → teardown), concurrent workers, latency percentiles, resource
sampling, codec-aware payload sizing, `errorfamily` error classification.
88 tests (`-race`). Soak tests skip in `-short` mode. Race-aware timing
thresholds via build-tag-gated `raceEnabled` constant. Benchmark results
published in `docs/status/2026-07-24_17-54_benchmark-first-real-run.md`.
