# Execution Plan — Session Continuation

> Generated: 2026-06-03  
> Status: T1-T4 complete. T5-T8 + reflection items remaining.  
> All tasks ≤15min. Sorted by impact÷effort.

---

## Phase A: Quick Wins (High Impact, Low Effort)

| #   | Task                                                                           | Est   | Impact                                                        |
| --- | ------------------------------------------------------------------------------ | ----- | ------------------------------------------------------------- |
| A1  | **Update docs/benchmarks/README.md** with ALL T1-T4 results + new scale curves | 15min | **CRITICAL** — docs are stale, consumers need current numbers |
| A2  | **Run full test suite with `-race`** — verify no data races from T1-T3         | 10min | **CRITICAL** — concurrency library, race = bug                |
| A3  | **Fix ADR numbering** — rename 0007-pebble-scope → 0009                        | 3min  | Correctness                                                   |
| A4  | **Fix CONTRIBUTING.md** — replace `just` with `nix run`                        | 5min  | Unblocks contributors                                         |
| A5  | **Fix integration/go.mod** — `go mod tidy` for codec/v2 + snapshot/v2          | 2min  | gopls warnings                                                |
| A6  | **Remove deprecated `otel.TraceIDLogger`** — replace callers, delete           | 5min  | Dead API cleanup                                              |
| A7  | **Remove deprecated `query.ErrQueryNotSupported`** — replace callers, delete   | 5min  | Dead API cleanup                                              |
| A8  | **Remove unused `backend` field from Pebble store**                            | 3min  | Dead code                                                     |
| A9  | **Clean up pebble/config.go backward-compat aliases**                          | 5min  | Dead code                                                     |
| A10 | **Update CHANGELOG.md** with T1-T4 + session work                              | 10min | Transparency                                                  |

**Phase A: 10 tasks, ~63min**

---

## Phase B: Verification & Hardening

| #   | Task                                                                                  | Est   | Impact                          |
| --- | ------------------------------------------------------------------------------------- | ----- | ------------------------------- |
| B1  | **Run concurrent stress benchmarks with `-race`**                                     | 5min  | Verify thread safety under load |
| B2  | **Add `-race` CI check** — run `go test -race ./memory/... ./listing/... ./event/...` | 5min  | Catch races in CI               |
| B3  | **MemoryStore `globalLog` memory audit** — quantify duplication vs `events` map       | 10min | Understand memory tradeoff      |
| B4  | **Verify scale benchmarks still pass** — `go test ./integration/... -tags=scale`      | 5min  | Regression check                |
| B5  | **Run `benchstat-compare.sh`** — establish new baselines                              | 10min | Trackable performance metrics   |

**Phase B: 5 tasks, ~35min**

---

## Phase C: Deeper Optimizations

| #   | Task                                                                                                     | Est   | Impact                    |
| --- | -------------------------------------------------------------------------------------------------------- | ----- | ------------------------- |
| C1  | **MemoryStore: store events ONLY in globalLog, use index for per-stream lookup** → eliminate duplication | 20min | **2× memory reduction**   |
| C2  | **Listing cache: auto-invalidate via `sync/atomic` counter** — compare store event count                 | 10min | Correctness + performance |
| C3  | **`findCodecOption` elimination** — cache default codec, skip probe for empty opts                       | 10min | 1 alloc per `New()`       |
| C4  | **`sync.Pool` for `ImmutableEvent` construction** — reuse event structs in hot loops                     | 15min | Reduce GC pressure        |
| C5  | **Add missing tests: FakeStore** — test Save/Load/ReadAll/ReadFrom/AppendBatch                           | 15min | Untested public code      |
| C6  | **Add missing tests: api-stability** — basic golden file + comparison test                               | 10min | Untested guard tool       |

**Phase C: 6 tasks, ~80min**

---

## Phase D: Architecture Improvements (Type Models)

| #   | Task                                                                                            | Est   | Impact                      |
| --- | ----------------------------------------------------------------------------------------------- | ----- | --------------------------- |
| D1  | **Design `Option` as interface** — eliminates func closure alloc, enables `findCodecOption` fix | 20min | API improvement + perf      |
| D2  | **Reactive `AggregateReader`** — subscribe to bus, auto-update cache                            | 15min | Real-time listing           |
| D3  | **Compact `Metadata` representation** — reduce 152B per event                                   | 20min | Memory reduction            |
| D4  | **Evaluate faster JSON codec** — benchmark `goccy/go-json` vs stdlib                            | 15min | Potential 2-3× JSON speedup |

**Phase D: 4 tasks, ~70min**

---

## Summary

| Phase     | Tasks              | Est         | When    |
| --------- | ------------------ | ----------- | ------- |
| A         | Quick Wins         | 63min       | **NOW** |
| B         | Verification       | 35min       | After A |
| C         | Deep Optimizations | 80min       | After B |
| D         | Architecture       | 70min       | After C |
| **Total** | **25 tasks**       | **~248min** |         |

---

## Critical Path

```
A1 (docs) → A2 (race) → B1 (stress race) → B4 (regression) → C1 (memory dedup) → C5 (tests)
```

If time is limited: **A1 → A2 → A3-A10 → B1 → C5 → C6** (housekeeping + verification + tests)
