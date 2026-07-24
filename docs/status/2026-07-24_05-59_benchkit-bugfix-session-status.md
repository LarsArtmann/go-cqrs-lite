# Status Report: benchkit — Bug Fix Session & Self-Review

**Date:** 2026-07-24 05:59
**Session:** Fixed critical dead-code bugs, replaced hand-rolled stdlib, added doc.go
**Status:** All 5 critical bugs fixed, 23 tests passing with `-race`, pushed to remote

---

## a) FULLY DONE

| Item                              | Details                                                                                                                                                           |
| --------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Config.Codec wired to event.New() | All event creation paths now pass `event.WithCodec(r.codec)` — was dead code, events always used DefaultCodec (CBOR) regardless of Config.Codec setting           |
| Config.Duration enforced          | Creates `context.WithTimeout(ctx, r.config.Duration)` wrapping all measurement phases — was dead code, user timeout silently ignored                              |
| Profile.ReadRatio implemented     | Scales read passes: `readPassesFor(0.1)` = 1 pass, `readPassesFor(0.8)` = 8 passes — was dead code, WriteHeavy and ReadHeavy profiles were behaviorally identical |
| Generator typed struct            | `BenchPayload` struct replaces `map[string]any` + byte truncation. Padding field fills to target size. No more invalid JSON at small sizes                        |
| Hand-rolled stdlib replaced       | `sortStrings` → `sort.Strings`, `splitFields` → `strings.Fields`, `parseUint` → `strconv.ParseUint`                                                               |
| /proc/self/stat comm bug fixed    | CPU time parser now parses from last `)` delimiter, not field index — process names with spaces no longer corrupt all field indices                               |
| doc.go with build tag docs        | Documents the `goexperiment.jsonv2` requirement that causes cryptic "build constraints exclude all Go files" without it                                           |
| 23 tests passing                  | 10 integration + 7 generator/profile + 6 metrics/percentile tests                                                                                                 |
| Race detector clean               | `go test -race` passes with 0 data races across all concurrent phases                                                                                             |
| All changes pushed                | 4 commits: `19f540ec`, `c1ad8a50`, `4b1a0c07`, `9ac2cc52`                                                                                                         |

**Evidence:** `go test -tags "goexperiment.jsonv2" -race ./benchkit/... -count=1` → 23 PASS, 0 FAIL, 2.1s

---

## b) PARTIALLY DONE

| Item                  | What exists                                                                                                                     | What's missing                                                                                                                                                                                               |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Workspace integration | `cmd/cqrs-bench` builds from workspace root (`go build ./cmd/cqrs-bench/...` works, exit 0). `go.work` lists `./cmd/cqrs-bench` | No replace directives needed anymore (the stale go.work.sum was the root cause), BUT `GOWORK=off` builds fail because benchkit is unpublished. The workspace path works because go.work resolves local paths |
| AGENTS.md             | benchkit mentioned 4 times (module list, test command, directory structure)                                                     | Test command shows `./benchkit/...` but doesn't document the `-tags "goexperiment.jsonv2"` requirement inline                                                                                                |
| CLI functionality     | `run` and `compare` subcommands work for memory/sqlite/pebble                                                                   | No `--version` flag, no `report` subcommand, no CLI tests                                                                                                                                                    |
| Warmup                | Warmup runs before measurement, writes to throwaway aggregate                                                                   | Still pollutes the same store (adds N events to journal). This inflates ReadAll/ReadFrom scan times. Not fixed — see section d                                                                               |

---

## c) NOT STARTED

| Item                            | Description                                                                                                                | Status      |
| ------------------------------- | -------------------------------------------------------------------------------------------------------------------------- | ----------- |
| errors.go with errorfamily      | Project convention is `errorfamily.NewRejection(...)`, benchkit uses plain `fmt.Errorf`                                    | Not started |
| Pebble backend tests            | Third local backend untested in benchkit                                                                                   | Not started |
| CLI tests                       | `cmd/cqrs-bench` has zero tests                                                                                            | Not started |
| flake.nix integration           | benchkit not in `nix run .#build`, `nix run .#test`, `nix run .#lint` targets                                              | Not started |
| ADR for benchkit                | No architecture decision record documenting factory pattern, 8-phase design, relationship to contracttest                  | Not started |
| SKILL.md update                 | `.agents/skills/go-cqrs-lite/SKILL.md` doesn't mention benchkit (0 matches)                                                | Not started |
| Durability comparison (Phase 2) | `synchronous=FULL` vs `OFF`, `Flush()` timing                                                                              | Not started |
| Production replay (Phase 6)     | JSON Lines export/import for replaying real production data                                                                | Not started |
| benchtest.RunSuite (Phase 7)    | `benchtest.RunSuite(b, factory)` mirroring contracttest for Go benchmark framework                                         | Not started |
| Pebble Metrics() integration    | LSM-tree health (block cache hit rate, compaction count)                                                                   | Not started |
| SQLite PRAGMA metrics           | `page_count`, `page_size`, WAL file size                                                                                   | Not started |
| Custom workload support         | `Workload` struct with `Setup`/`WriteOps`/`ReadOps`/`Teardown` callbacks                                                   | Not started |
| Baseline comparison             | `cqrs-bench compare --baseline baseline.json --threshold 20` for CI regression detection                                   | Not started |
| Analytical benchmarks           | Projection catch-up speed, scan/aggregate latency, pre-aggregation overhead (from ANALYTICAL-VS-TRANSACTIONAL.md analysis) | Not started |

---

## d) TOTALLY FUCKED UP

### 1. Warmup still pollutes the event store

The warmup phase (`runner.go:268`) writes events to a throwaway aggregate in the **same store** that is then benchmarked. These extra events inflate:

- Journal scan times (ReadAll/ReadFrom scan N extra events)
- Aggregate count (1 extra aggregate in the journal)
- Read model phase (if warmup events trigger projections)

**Why I didn't fix it this session:** The fix requires either (a) creating a separate Bundle just for warmup (complexity: factory must be called twice, or warmup must be separate from the benchmarked Bundle), or (b) tracking warmup events and subtracting them from journal metrics (fragile). This is a design decision, not a one-line fix.

### 2. `estimateJSONSize` in generator.go is a guess

The `computePadding` method uses `estimateJSONSize` which is a hardcoded template string approximation, not an actual measurement. It doesn't account for:

- Variable-length numeric formatting (float64 → JSON digits)
- Map key ordering differences between runs
- Tag/metadata content length variation

The result: payloads are approximately the target size, not exactly. For benchmarks comparing codec efficiency (JSON vs CBOR), this imprecision undermines the storage efficiency metric.

**Better approach:** Marshal the payload once without padding, measure the actual byte length, then compute the padding delta. One extra marshal per payload — negligible cost for accuracy.

### 3. No negative/failure-path tests

All 23 tests are happy-path. There are zero tests for:

- Factory returning `nil` Bundle
- Factory returning `nil` EventSink
- Store returning errors during write/read
- Context cancellation mid-phase
- Config with Streams=0 or EventsPerStream=0
- Profile.ReadRatio edge cases (0.0, 1.0, negative)

### 4. `insertCommas` is still hand-rolled

I removed `sortStrings`, `splitFields`, and `parseUint` — all replaced with stdlib. But `insertCommas` and `truncate` remain because there's no perfect stdlib equivalent (Go has no `humanize.Comma` in stdlib). The `golang.org/x/exp` package has `slices.Format` but it's not a drop-in. Adding `dustin/go-humanize` as a dependency just for comma formatting is overkill for a benchmarking tool.

### 5. The `nix fmt` / golines formatting was never applied manually

BuildFlow's auto-fix ran `nix-fmt` and `golangci-lint` repair on every commit, but I never ran `nix fmt` explicitly to verify golines (max-len: 120) compliance. The code might have lines exceeding 120 chars that the linter didn't catch (golines and golangci-lint have different line-length enforcement).

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. **Add errors.go** — Define sentinel errors (`ErrFactoryFailed`, `ErrInvalidConfig`, `ErrPhaseTimeout`) using `errorfamily` classification. Project convention.
2. **Fix warmup pollution** — Either use a separate Bundle or document that warmup adds N events to the journal (choose one approach).
3. **Fix `estimateJSONSize`** — Marshal once, measure actual bytes, compute padding delta. Replace the template string guess.
4. **Add negative tests** — Factory returning nil, store errors, context cancellation, zero-value configs.
5. **Add `--version` flag** to CLI.
6. **Replace `insertCommas`** with `golang.org/x/exp/slices` or accept the hand-rolled version as adequate.

### Architecture

7. **Add `errors.go` with errorfamily** — Match project convention for error classification.
8. **Add disk size auto-detection** via `interface{ DiskSize() int64 }` on Bundle instead of manual `DiskPath` string.
9. **Integrate Pebble `Metrics()`** — Backend-specific metrics in Result.
10. **Add SQLite PRAGMA queries** — `page_count * page_size`, WAL file size.
11. **Consider analytical benchmarks** — Projection catch-up speed, scan latency (see ANALYTICAL-VS-TRANSACTIONAL.md).

### Integration

12. **Add benchkit to flake.nix** — Build and test targets.
13. **Update SKILL.md** — Add benchkit to module decision matrix.
14. **Write ADR** — Document factory pattern and 8-phase design.
15. **Add CLI tests** — At least smoke tests for each subcommand.

### Documentation

16. **Document warmup behavior** — Whether it pollutes the store or not.
17. **Document goexperiment.jsonv2** — Already in doc.go, but AGENTS.md test command should note it.

---

## f) Up to 50 Things to Do Next

### Critical (correctness)

1. Fix warmup store pollution — use separate Bundle or document the inflation
2. Fix `estimateJSONSize` — marshal-and-measure instead of template guess
3. Add negative tests: factory returning nil Bundle, nil EventSink, closed store
4. Add test for Config.Duration actually aborting a long-running phase
5. Add test verifying ReadRatio produces different read counts for WriteHeavy vs ReadHeavy

### High priority (completeness)

6. Add `benchkit/errors.go` with errorfamily classification
7. Add Pebble backend tests in benchkit_test.go
8. Add CLI smoke tests (at least `run --backend memory --profile dev`)
9. Add `--version` flag to CLI
10. Add disk size auto-detection via `interface{ DiskSize() int64 }`
11. Add Config validation (reject Streams=0, EventsPerStream=0)
12. Run `nix fmt` explicitly and verify golines compliance
13. Document warmup behavior in README

### Medium priority (design report phases)

14. Implement Phase 2: durability comparison (Save vs Save+Flush delta)
15. Integrate Pebble `Metrics()` into Result as backend-specific metrics
16. Add SQLite PRAGMA metrics (page_count, WAL size)
17. Implement Phase 7: `benchtest.RunSuite(b, factory)`
18. Add custom workload support (`Workload` struct with callbacks)
19. Add baseline comparison for CI regression detection
20. Write ADR for benchkit design decisions
21. Update Crush skill (SKILL.md) with benchkit module entry
22. Add benchkit to flake.nix build/test/lint targets
23. Add GC pause time tracking (MemStats.PauseTotalNs delta)
24. Add RSS measurement beyond heap
25. Add `eventsPerMB` storage efficiency metric

### Lower priority (polish and future)

26. Add CLI `report` subcommand (generate from saved JSON)
27. Implement Phase 6: production data replay (JSON Lines export/import)
28. Add Postgres backend support to CLI (with graceful skip)
29. Add Turso backend support to CLI
30. Add multi-DB split benchmarking mode
31. Add `--tmpfs` flag for CI environments
32. Add connection pool stats for SQL backends
33. Add per-projection lag breakdown in Result
34. Add markdown report for single-result (not just comparison)
35. Add CSV output format for spreadsheet import
36. Add trend tracking (append to historical JSON, compute deltas)
37. Add HDR histogram as optional integration for >1M sample precision
38. Add custom decider benchmark support (`DeciderBenchmark[State]` generic)
39. Add snapshot strategy benchmarking (EveryNEvents, ReadPressure)
40. Add codec comparison mode (JSON vs CBOR vs CBORCompact)
41. Add configurable event types (currently hardcoded to `bench.event`)
42. Add middleware benchmarking (signing, encryption, OTel overhead)
43. Add `cqrs-bench doctor` command (like cqrs-lint) showing detected capabilities
44. Add benchmark result schema versioning for forward compatibility
45. Add analytical benchmark profiles (projection catch-up, scan latency)
46. Add read-model query benchmarking (SQLViewStore.Query with conditions)
47. Add graph projection benchmarking (GraphProjection node/edge merge throughput)
48. Add projection host catch-up benchmark (replay 100K events, measure time-to-caught-up)
49. Add streaming throughput benchmark (SSE/BackfillHandler delivery rate)
50. Add comparison report with statistical significance (t-test or bootstrap CI)

---

## g) Questions

### 1. Should warmup use a separate Bundle (double factory call) or stay inline with documented inflation?

The warmup phase writes events to the benchmarked store to warm caches and connection pools. This inflates journal metrics by N warmup events. Options:

- **A:** Call the factory twice — once for warmup (discarded), once for measurement (clean store). Cost: factories that create temp dirs must handle double-call correctly.
- **B:** Keep warmup inline but subtract warmup events from journal scan counts. Cost: fragile bookkeeping.
- **C:** Keep as-is and document that warmup adds N events. Cost: journal scan times are slightly inflated.

I cannot determine which approach aligns with the project's quality bar without your input.

### 2. Should benchkit add `errorfamily` as a dependency for error classification?

The project convention (AGENTS.md) says all modules use `errorfamily.NewRejection(...)`, `errorfamily.WrapInfrastructure(...)`. benchkit currently uses plain `fmt.Errorf`. benchkit already has `go-error-family` as an indirect dependency (via event/), so adding it directly wouldn't increase the dependency footprint. But benchkit is a tool, not a domain module — consumers don't build error-handling logic on benchmark errors. Is this convention-bound or context-dependent?

### 3. Should the generator's `estimateJSONSize` be replaced with marshal-and-measure?

The current implementation uses a hardcoded template string to estimate payload size before computing padding. This means payloads are approximately (not exactly) the target size. For codec comparison benchmarks (JSON vs CBOR), this imprecision undermines the storage efficiency metric. The fix is one extra `json.Marshal` per payload — negligible cost, but changes the generator's output characteristics (deterministic payloads would have slightly different byte counts). Should I prioritize this fix?
