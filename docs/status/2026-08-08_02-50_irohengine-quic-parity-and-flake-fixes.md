# Status: Irohengine QUIC Parity Tests + Flake Fixes

**Date:** 2026-08-08 02:50
**Session scope:** Irohengine TODO items (4 tasks), decider singleflight flake

---

## What This Session Did

Resolved all 4 open Irohengine TODO items + fixed the decider singleflight flake.

### Files Changed

| File                                                | Change                                                                                                    |
| --------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `decider/decider_singleflight_test.go`              | Fixed singleflight coalescing flake: 50ms→200ms window, added `runtime.Gosched()` after barrier release   |
| `metaengine/irohengine/quic/transport_test.go`      | Fixed `TestQuicSetConvergence` + `TestQuicPNCounter` flakiness; added `TestQuicMapUpdateDoesNotReplicate` |
| `metaengine/irohengine/quic/adt_matrix_test.go`     | **NEW** — `TestQuicADTMatrix`: full 10-ADT `adttest.RunMatrix` against QUIC transport                     |
| `metaengine/irohengine/loopback/adt_matrix_test.go` | **NEW** — `TestLoopbackADTMatrix`: full matrix against loopback transport                                 |
| `TODO_LIST.md`                                      | Marked all 4 Irohengine items `[x]` with implementation details                                           |

---

## A) FULLY DONE

### 1. TestLoad_ConcurrentLoadsCoalescedBySingleflight flake — FIXED

**Root cause:** The coalescing window (50ms sleep in `countLoadStore.Load`) was too short. Under `-race` (5-10x scheduling inflation) or high parallel test load, goroutines released by the `close(start)` barrier didn't all arrive at `singleflight.Do` before the first one completed. Some goroutines would miss the in-flight call and trigger separate `store.Load` calls, failing the `count == 1` assertion.

**Fix:**

- Extracted `delay()` method, increased from 50ms to 200ms (sufficient even under `-race`)
- Added `runtime.Gosched()` after `close(start)` to immediately yield the main goroutine, giving the waiting goroutines CPU time to enter `singleflight.Do` promptly
- Updated the comment explaining the rationale

**Verification:** 50 iterations with `-race -parallel 16` all pass.

### 2. TestQuicSetConvergence flakiness — FIXED

**Root cause:** The test used `Eventually` for the first SetAdd element ("go") but a direct assertion for the second ("cqrs"). Since SetAdd ops arrive over QUIC as independent stream operations, "cqrs" may not have converged when "go" passes the `Eventually` check.

**Fix:** Both elements now checked inside the same `Eventually` block — the poll only succeeds when both are present.

**Also fixed in the same file:** `TestQuicPNCounter` had the same anti-pattern (`time.Sleep(200ms)` then direct assertions). Replaced with `Eventually` on both nodes.

### 3. Non-CRDT op rejection on QUIC path — DONE

**What was added:** `TestQuicMapUpdateDoesNotReplicate` — the QUIC equivalent of the in-process `TestMapUpdateDoesNotReplicate`.

**Key challenge:** CBOR round-trip over QUIC converts `int(0)` to `uint64(0)`. The test handles this via:

- The `MapUpdate` callback's type switch handles both `int` and `uint64`
- `BeEquivalentTo` instead of `Equal` for value comparisons (type-agnostic)
- `Eventually` for the local assertion (MapUpdate is synchronous, but the pattern is consistent)

**Verification:** Confirms MapSet replicates (value reaches node B) but MapUpdate stays local (node B still sees 0).

### 4. QUIC transport integration with adttest.RunMatrix — DONE

**What was added:** `TestQuicADTMatrix` in `quic/adt_matrix_test.go` — runs the full `adttest.RunMatrix` (10 CRDT-safe ADTs + StreamLog auto-skipped) against a QUIC-backed replicated engine.

**Also added:** `TestLoopbackADTMatrix` in `loopback/adt_matrix_test.go` — the same matrix against the loopback transport, closing the middle tier of the three-tier transport testing pyramid (InProcess → loopback → QUIC).

**Result:** All 10 ADTs pass cross-engine parity (canonical output matches bare Memory engine). StreamLog is auto-skipped because `replicatedEngine` does not implement `StreamLogBackend` (stream logs are not CRDT-safe).

---

## B) PARTIALLY DONE

Nothing partially done — all 4 items were completed.

---

## C) NOT STARTED (out of scope this session)

These are items I noticed during the work but did not address:

1. **`TestQuicLWWResolution` still uses `time.Sleep(100ms)`** between the two writes to ensure timestamp ordering. This is a timing-dependent pattern that could theoretically flake, but the 100ms gap is large enough for LWW ordering. Not fixed because it's not currently flaking.
2. **`TestQuicRTTMeasurement` uses a `for i := 0; i < 10; i++` loop** — gopls rangeint hint. Cosmetic, not functional.
3. **`TestQuic3NodeRelayConvergence` and `TestQuicWriteAfterReconnect` use `time.Sleep`** for convergence — same anti-pattern as the old PNCounter test. Not currently flaking but could be hardened with `Eventually`.
4. **latency_test.go gopls rangeint hints** — 6 instances of `for i := 0; i < N; i++` that could be modernized to `for i := range N`. Cosmetic.
5. **`TestQuicPNCounter` timing** — both increments fire from different nodes in rapid succession. If the second increment arrives at node A before the first increment replicates to node A, the `Eventually` on node A might briefly see only 3 or only 5 before converging to 8. The `Eventually` handles this correctly, but the test could be clearer about the expected convergence path.

---

## D) TOTALLY FUCKED UP

Nothing. All changes compile, all tests pass across 4 modules (decider, irohengine, loopback, quic).

---

## E) WHAT WE SHOULD IMPROVE

### Self-Critique of This Session's Work

1. **CBOR type drift is a real problem, not just a test issue.** The `int → uint64` conversion over CBOR round-trip on the QUIC path means consumer payloads change type silently. This affects MapSet values, counter deltas, and any numeric data. The test works around it with `BeEquivalentTo`, but consumers will hit `fmt.Sprintf("%T", val)` mismatches. **This is a production bug in the codec layer**, not just a test issue. The CBOR decode mode (`opDecMode` in `quic/latency.go`) uses `DefaultMapType: map[string]any` but doesn't control integer width.

2. **No `Eventually` helper for non-Map ADTs.** The `eventuallyGet` helper in the QUIC test file only works for MapBackend. SetContains, CounterGet, LogTail, etc. each need their own inline `Eventually` block. A shared `eventuallyConverged` helper would reduce boilerplate.

3. **The loopback matrix test is trivial because loopback has no peers.** `TestLoopbackADTMatrix` creates a loopback transport with no `Connect()` call — so replication never happens and the replicated engine is effectively just a Memory engine wrapper. The test still proves the wrapper doesn't break semantics, but it doesn't test actual cross-node convergence over TCP. A real 2-node loopback matrix (like the existing `setupTwoNodeLoopback` pattern) would be more valuable.

4. **The QUIC matrix test also has no peers connected.** Same issue as loopback — each factory.Create spins up an isolated QUIC endpoint with no `Connect()`. The matrix proves the wrapper+transport doesn't corrupt single-node operations, but doesn't test cross-node CRDT convergence under the matrix scenarios. The dedicated convergence tests (`TestQuicMapConvergence2Node`, etc.) cover that, but it's a gap in the matrix itself.

5. **No `-race` run of the full QUIC test suite.** I ran individual tests with `-race` but not the full QUIC suite with `-race` (CGo + race is slow). Should be done in CI.

### Broader Observations

6. **The transport testing pyramid is now complete** (InProcess → loopback → QUIC) for ADT matrix parity, but the convergence tests are still per-transport duplication. A shared `RunConvergenceSuite(t, transportFactory)` would reduce the ~200 lines of nearly-identical convergence tests across the 3 transport test files.

7. **`TestQuicWriteAfterReconnect` creates a new `nodeCoord` engine after closing nodeA** — the coordinator transport was created at the top but the engine wrapping it was only created in the middle of the test. This is confusing but functionally correct (the transport is shared, only the engine wrapper is new).

---

## F) Up to 50 Things We Should Get Done Next

### Irohengine / QUIC / Loopback

1. **Fix CBOR int→uint64 type drift** — the codec layer silently changes numeric types over the wire. Either use `cbor.IntDec` to force `int` decoding, or document the contract.
2. **Add 2-node convergence to the loopback matrix** — `TestLoopbackADTMatrix` currently has no peers connected.
3. **Add 2-node convergence to the QUIC matrix** — same gap as loopback.
4. **Extract shared convergence test suite** — `RunConvergenceSuite(t, factory)` to eliminate ~200 lines of per-transport duplication.
5. **Harden `TestQuicLWWResolution`** — replace `time.Sleep(100ms)` with a deterministic timestamp injection (or accept the sleep as necessary for real LWW ordering).
6. **Harden `TestQuic3NodeRelayConvergence`** — replace `time.Sleep(500ms)` with `Eventually` for peer readiness.
7. **Harden `TestQuicWriteAfterReconnect`** — replace `time.Sleep` calls with `Eventually`.
8. **Add `Multimap` convergence test for QUIC** — exists for in-process but not QUIC.
9. **Add `Multimap` convergence test for loopback** — exists for in-process but not loopback.
10. **Run full QUIC suite with `-race`** — CGo + race detection, verify no data races in the transport layer.
11. **Run all QUIC tests 20x to establish flake baseline** — the fixes look good, but network tests need repetition to build confidence.
12. **Add QUIC transport to CI** — currently CGo tests may not run in CI. Verify the GitHub Actions workflow includes `CGO_ENABLED=1` for the quic module.
13. **Consider `//go:build integration` tag for slow QUIC tests** — the reconnect tests take 5-10s each; they could be tagged for optional CI runs.
14. **Add a `TransportConvergenceSuite` exported test harness** — similar to `adttest.RunMatrix` but for convergence patterns (Map, Set, Counter, Log, Multimap, LWW).
15. **Profile QUIC transport overhead** — `bench_test.go` has `BenchmarkQuicMapSet` but no comparison with loopback or InProcess for the same workload.
16. **Add connection pooling to QuicTransport** — each `Publish` opens a new BiStream; reusing streams would reduce latency.

### Decider

17. **Run the full decider test suite 100x with `-race`** — verify the singleflight fix holds under sustained parallel load.
18. **Consider increasing `delay()` further for CI** — CI runners are slower; 200ms may still be marginal on shared infrastructure.
19. **Add a benchmark for singleflight coalescing hit rate** — measure what percentage of concurrent loads are actually coalesced under various goroutine counts.

### Code Quality / Hygiene

20. **Fix gopls rangeint hints in `latency_test.go`** — 6 instances of `for i := 0; i < N; i++` → `for i := range N`.
21. **Fix gopls rangeint hint in `quic/transport_test.go:200`** — `for i := 0; i < 10; i++`.
22. **Fix gopls waitgroupgo hints in `decider_singleflight_test.go`** — 2 instances of goroutine creation that can use `WaitGroup.Go`.
23. **Update QUIC module go.mod** — add `adttest` dependency if not already present (it's in the parent metaengine module, so the replace directive covers it).
24. **Run `nix fmt` on all changed files** — formatting pass.
25. **Run `nix run .#lint` on the affected modules** — verify no new lint issues.
26. **Update api-stability golden** — no exported symbols were added/removed, but verify.
27. **Update AGENTS.md** — note the CBOR type drift issue and the transport testing pyramid completion.

### Testing Infrastructure

28. **Add a flaky test detection CI job** — run critical tests 20x with `-count=20` and fail if any iteration fails.
29. **Add `-race` to the default test command for non-CGo modules** — race detection catches real bugs.
30. **Consider a nightly "stress" CI job** — run all tests with `-count=50 -race -parallel 16` to catch flakes that only appear under sustained load.
31. **Add test timeout governance** — some QUIC tests have 15s `Eventually` timeouts; document the max acceptable test duration.

### Documentation

32. **Document the CBOR type contract** — if `int → uint64` is intentional, document it; if not, fix it.
33. **Document the transport testing pyramid** — the three tiers and what each catches.
34. **Update irohengine README** — mention the matrix tests and the three-tier testing approach.
35. **Add an ADR for the singleflight coalescing window** — why 200ms, why `Gosched()`.
36. **Add the QUIC convergence tests to the README** — document what's tested and what's not.

### Metaengine

37. **Run `adttest.RunMatrix` against all engines** — verify DuckDB, Pebble, Badger, Postgres, Dgraph all pass the updated 11-scenario matrix.
38. **Fix the stale "10 ADTs" comment in adttest/harness.go** — it now has 11 scenarios.
39. **Fix the stale "7-ADT" nolint comment in adttest/harness.go** — same issue.
40. **Add StreamLogBackend to replicatedEngine** — or explicitly document why it's excluded (stream logs are not CRDT-safe; this is correct, but the compile-time assertion list includes it via the broad Engine interface).

### Broader Project

41. **Verify the full verify gate passes** — `nix run .#verify` or at minimum `nix run .#verify-fast` after these changes.
42. **Check if the auto-commit daemon has touched any of these files** — verify no conflicts.
43. **Review whether the decider delay increase slows the test suite significantly** — 200ms × parallel tests could add up.
44. **Consider extracting the QUIC test helpers** (`setupTwoNodeQuic`, `newQuicCluster`, `waitForPeers`, `eventuallyGet`) into a shared `quictest` package.
45. **Add a transport-agnostic test helper** — `newTwoNodeCluster(t, transportFactory)` that works for all 3 transports.
46. **Review the `markSeen` dedup set in QUIC** — resets at 10K entries, which could cause double-application under high throughput. Consider a bounded ring buffer (like `dedup.Ring`).
47. **Review QUIC `sendOp` error handling** — silently returns on `OpenBi()` failure; should log or retry.
48. **Add QUIC transport metrics** — peer count, ops published, ops received, average RTT.
49. **Consider QUIC transport health check** — `HealthChecker` implementation for `system/` introspection.
50. **Review the relay mode dedup** — `relayToOthers` uses the same `markSeen` set as incoming dedup, which could cause legitimate relay ops to be dropped.

---

## G) Questions I Cannot Answer Myself

### 1. Should the CBOR int→uint64 type drift be fixed in the codec, or documented as expected behavior?

The QUIC transport uses `fxamacker/cbor/v2` with `TimeUnixDynamic` encoding and `DefaultMapType: map[string]any`. When a consumer does `MapSet(ctx, "counters", "c1", 0)` (Go `int`), the remote node receives `uint64(0)`. This is a CBOR spec behavior (integers are encoded as unsigned when ≥0), but it means `val.(int)` type assertions fail on the receiver. Should I:

- (a) Fix the decoder to coerce back to `int` (breaking CBOR spec semantics but matching Go expectations)?
- (b) Document it and tell consumers to use type-assertion-safe patterns (`BeEquivalentTo`, switch statements)?
- (c) Something else?

I cannot determine this without knowing whether any consumers depend on the current behavior.

### 2. Should the loopback/QUIC matrix tests connect real peers (2-node convergence), or is single-node wrapper-parity sufficient?

The current `TestLoopbackADTMatrix` and `TestQuicADTMatrix` create isolated engines with no peers — they prove the wrapper doesn't corrupt semantics but don't test cross-node CRDT convergence under matrix scenarios. A 2-node variant would require a `Factory` that can create connected pairs, which `adttest.RunMatrix` doesn't support (it creates one engine per factory). Should I:

- (a) Extend `adttest.Factory` with an optional `CreatePair` for convergence testing?
- (b) Write separate 2-node convergence matrix tests outside `RunMatrix`?
- (c) Accept the current single-node matrix + the existing per-ADT convergence tests as sufficient coverage?

This is an architectural decision about the test harness design.

### 3. Is the 200ms singleflight coalescing window acceptable for CI performance, or should it be adaptive?

The delay increased from 50ms to 200ms. With 3 singleflight tests each taking 200ms, that's 600ms added to the decider test suite. Under `-race`, the scheduler may still need more time. Options:

- (a) Keep 200ms fixed (simple, slightly slow)
- (b) Use `testutil.RaceEnabled` to pick 200ms under race, 100ms without (faster non-race CI)
- (c) Use a `sync.WaitGroup` inside `countLoadStore` that waits for N concurrent Load calls before proceeding (deterministic, no sleep)

Option (c) is the most correct but changes the test's structure. I cannot decide this without knowing the CI performance budget.

---

## Session Verdict

**All 4 TODO items resolved. All tests pass.** The CBOR type drift (item E.1) is the most significant issue discovered during this work — it's a production-level concern, not just a test issue, and needs a decision before the QUIC transport is used in anger.
