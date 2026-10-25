# 50 Testing Improvements: Functional and Non-Functional

**Date:** 2026-10-24
**Status:** PROPOSAL (nothing in this document is implemented; items are candidates, not commitments)
**Scope:** the whole workspace, 101 `go.mod` files

## Why this document exists

The suite is large but unevenly defended. Everything below is grounded in a
census run on 2026-10-24, not intuition. The numbers that drive the priorities:

| Measurement | Value | Read |
| --- | --- | --- |
| Test files / test functions | 1,513 / 7,556 | large suite, no trend line |
| Benchmarks / fuzz targets | 420 / 115 | fuzzing exists, nightly mutation does not |
| Modules with a coverage floor | 11 of ~97 non-testdata | the ratchet covers ~11% of the product |
| `t.Skip` call sites | 224 | no gate, no reason baseline |
| `testing.Short()` gates | 14 | skips are mostly not short-mode-aware |
| `t.Parallel()` uses | 6,919 | genuinely parallel suite |
| Ginkgo spec files | 62 | BDD exists, `systemscenario` adoption is thin |
| `adttest` consumers | 65 files | engine conformance is the best-defended area |
| `rapid` / `goleak` / `testing/synctest` files | 32 / 6 / 0 | property testing partial, leak and virtual-time testing almost absent |
| `-shuffle` scope | integration legs only | unit suites never shuffle |
| Nightly fuzzed targets | 4 of 115 | 96% of fuzz targets never mutate |
| Automated mutation testing | none | "prove the test goes red" is manual discipline |

Two structural conclusions follow. First, the repo is excellent at
**conformance** (engines, wire forms, cross-module contracts) and weak at
**reach** (coverage floors, skip inventory, untested public surface). Second,
the expensive flake classes this project has already paid for (clock
dependence, goroutine leaks, order dependence, load sensitivity) each have a
mechanical cure that is either absent or applied in one corner only.

## Priority summary

| Tier | Items | Rationale |
| --- | --- | --- |
| Do first | 1, 2, 9, 12, 13, 24, 33 | Eliminate whole defect classes or close the widest blind spots |
| Next | 3, 5, 6, 10, 11, 20, 26, 27, 41, 42, 49 | Structural, moderate effort, high recurrence value |
| Then | remainder | Valuable, narrower, or dependent on the tiers above |

---

## A. Coverage and reach

The coverage ratchet is the narrowest gate in the repo. `scripts/check-coverage.sh`
pins 11 modules; everything else can regress to zero silently.

1. **Baseline every production module.** Generate `scripts/coverage-baseline.txt`
   for all ~97 non-testdata modules and gate on delta. Keep the existing
   quiet-window policy (baseline from quiet numbers, widen only the lower bound
   on CI).

2. **Ratchet per package, not per module.** A growing module denominator hides
   new uncovered code, exactly as the 2026-10-23 re-baseline note in
   `check-coverage.sh` documents for `metaengine`. Parse `-coverprofile` output
   per package and fail on any package dropping more than 1 point.

3. **Gate "untested exported symbol".** `cmd/api-stability` already enumerates
   the entire public surface. Cross that golden against identifiers referenced
   in `_test.go` files and require at least one test per exported symbol,
   with an explicit allowlist for deprecated v5-removal shells.

4. **Detect zero-assertion tests.** AST-scan `_test.go` files for `Test*`
   functions with no reachable `t.Error`, `t.Fatal`, `require`, `gomega`, or
   `cmp` assertion. A test that cannot fail is worse than no test, because it
   inflates the count.

5. **Inventory the 224 `t.Skip` sites.** Require a reason string and keep a
   checked-in baseline (the `scripts/file-size-baseline.txt` pattern: new
   skips fail, removals shrink the baseline).

6. **Close the short-mode blind spot.** Only 14 sites consult
   `testing.Short()`, while 224 sites skip for other reasons. Classify each
   skip as environment-conditional or permanent, and route the conditional
   ones into a nightly job that runs them for real.

7. **Execute the examples, not just build them.** `example/*` is registered in
   `examplePaths` and therefore build-only in CI. Add at least one
   `systemscenario` run per example so the documented composition is proven
   to boot.

8. **Publish test-count and coverage trends.** Emit per-module JUnit XML plus
   coverage as artifacts and alert on drops. Without a trend line, deleting
   500 tests is invisible.

## B. Determinism and flake elimination

Every flake class this repo has already paid for has a mechanical cure. Most
of them are applied in one place only.

9. **Shuffle unit tests.** `-shuffle=on` currently appears only in the
   integration legs (`ephemeral-pg.sh`, `ephemeral-dgraph.sh`,
   `ephemeral-redis.sh`, `vm-mysql*.sh`). Add it to `#test` and `#test-race`,
   log the seed, and keep the seed in the failure output.

10. **Replay shuffle seeds automatically.** `build/shuffle-seeds.log` is
    uploaded by CI but nothing consumes it. Add a nightly job that reruns the
    previous day's seeds.

11. **Repeat-mode CI leg.** Run the top-20 stateful packages with
    `-count=5 -cpu=1,4` nightly. Order dependence and shared-state bugs
    require repetition to surface; single runs mis-classify them as noise.

12. **`goleak` in every goroutine-owning module.** Only 6 test files import it
    across 97 modules, although the bus, projection host, deriver, scheduler,
    and `StartAutoReprobe` all start goroutines. Add
    `goleak.VerifyTestMain` to every such module's `TestMain`.

13. **Adopt `testing/synctest`.** Zero usages today. It belongs in
    `scheduling/`, `scheduling/sqlstore`, snapshot TTLs, middleware
    circuit-breaker half-open transitions, and projection-host retry backoff:
    virtual time removes the entire wall-clock flake class.

14. **Ban `time.Sleep` in tests by lint rule.** Replace with poll helpers
    (`systemscenario.WithAwaitTimeout` and deadline-bounded `Await`). Allowlist
    only the measured soaks.

15. **Flake ledger with retry reporting.** Track `flaky-tests.txt` (test,
    owner, first seen, failing seed). CI may retry once, but a retried-green
    result must be reported as FLAKY, never as pass.

16. **Per-package runtime budget.** Record durations from `go test -json` and
    fail on a large regression. Catches an accidentally un-skipped soak and
    cross-package lock contention before it burns the 30m timeout.

## C. Property-based, model-based, differential

17. **Expand `rapid` beyond 32 files.** Highest-value invariants: codec
    roundtrip (JSON and CBOR), the ADR-0044 envelope (`WrapEncode` /
    `UnwrapDecode` / `decodeEnvelopeOrLegacy`), defensive clones on public
    accessors, and `record.ParseEncoding` totality.

18. **Model-based test for the metaengine planner.** A reference in-memory
    model versus `Store` routing over randomized engine capability sets:
    assert cost ordering, routing hysteresis (`WithRoutingHysteresis`,
    `WithRoutingMinDelta`), and the invariant that no plan targets a
    quarantined engine.

19. **Differential engine testing.** Feed one randomized operation script to
    every engine (memory, sqlite, turso, pebble, bbolt, badger, duckdb, pg,
    mysql, dgraph) and diff the results. `adttest` already supplies the
    fixture vocabulary in 65 files; this adds a fuzz driver on top.

20. **Fuzz replay determinism.** Random event sequences through decider,
    journal, reset, and replay must be identical and panic-free. Deterministic
    replay is the core promise of the library and is currently example-tested
    only.

21. **Fuzz the envelope and codec decoders.** Malformed base64, adversarial
    metadata JSON, encoding-byte confusion. Live mutation currently exists
    only for four SQL identifier/WHERE-gate targets.

22. **Rotate all 115 fuzz targets nightly.** `fuzz.yml` fuzzes 4 targets for
    60s each. Shard the remainder across a nightly rotation with a persisted
    corpus per target. An unrun fuzz target is documentation, not a defense.

23. **Snapshot equals full replay.** For random command sequences, decisions
    from snapshot-load must equal decisions from replay-load. Snapshot
    currently has isolation tests only, not an equivalence pin.

## D. Functional depth

24. **Ratchet `systemscenario` adoption.** ADR-0153 built the harness that
    exercises the same `DomainConfig` / `DeploymentConfig` a consumer ships.
    Add a per-module count of system-level specs and grow it; today it is
    concentrated in a few packages.

25. **Fix and pin the documented deriver deadlock.** A synchronous deriver on
    `sys.Bus()` deadlocks (single-topic watermill bus with
    `BlockPublishUntilSubscriberAck`). Write the timeout-bounded failing test
    first, then the async-delivery fix, so the bug cannot return unnoticed.

26. **Deployment-config boot matrix.** `system.New` must boot, project, query,
    and `Close` cleanly for every documented engine permutation, including the
    iroh-via-local path. One table-driven test covers the entire matrix.

27. **Reset and `CatchUpEngine` parity per engine.** For every
    `EngineResetter`: reset, replay, byte-identical read models, and journal
    survival per ADR-0143. Not every engine carries this pin.

28. **Idempotency conformance matrix.** The same command twice against every
    engine must produce exactly one event, including the degraded
    `MapDedupStore` path (ADR-0142).

29. **Consumer compile fixtures per released tag.** Price the documented
    recipes against the PUBLISHED tag, not only the working tree, so a break
    is caught before a release wave rather than after.

30. **Behaviour tests for the error taxonomy.** `docs/error-taxonomy.md` is
    checked against source, never against behaviour. Assert each documented
    code's family and wrapping, and that every exported sentinel has one test.

31. **Promote recipes to executable scenarios.** `TestRecipes` compile-checks
    104 fenced blocks. Promote the top 20 to runnable `systemscenario` specs;
    compiling is not behaving.

32. **Deprecation behaviour pins.** The v5-removal shells (`schema.VersionedStore`,
    `signing.Rejecting*`, `metadata.CustomData`, `idempotency/*`) must keep
    WORKING, not merely compiling, until they are removed.

## E. Non-functional: performance, memory, durability

33. **Absolute latency ceilings per engine.** `benchkit` compares against a
    prior baseline, so "both got slower" passes. Add SLO ceilings (P50/P99)
    per engine and per profile.

34. **Tail-latency concurrency tests.** `#load-sweep` is manual. Add a nightly
    concurrency matrix over singleflight coalescing, circuit-breaker
    transitions, WAL writes, and projection fan-out that asserts P99, not just
    correctness.

35. **Heap-growth assertions per engine.** `TestSoak_MemoryBounded` pins the
    memory engine only. Add projection replay and `metaengine.ServeSSE`
    ring-buffer backpressure (drop-old) heap ceilings.

36. **Goroutine baseline after `Close`.** Dispatch N operations, close the
    system, assert goroutines return to baseline. Pairs with item 12 and
    catches leaked watch loops that goleak's process-exit check can miss.

37. **Crash-consistency tests.** `kill -9` a child writer mid-transaction
    against sqlite, pebble, and bbolt, then reopen and assert no journal gaps,
    no duplicates, and a clean replay. Append-only facts deserve a
    power-loss pin.

38. **Disk-growth and compaction budgets.** Pebble, bbolt, badger: append N
    events, close and reopen, assert size and read-latency bounds. The bbolt
    sweep exists but is skipped inside `#verify`.

39. **Un-skip the heavy soaks into a weekly job.**
    `SOAK_SKIP_BOLT`, `SOAK_SKIP_DGRAPH`, `SOAK_SKIP_DUCKDB` are correct
    inside `#verify` and wrong as a permanent state (bbolt alone measures
    509-1145s). Give them a dedicated weekly workflow.

40. **Race coverage completeness gate.** Assert that the module set run by
    `#test-race` equals `testModules` in `flake.nix`, and add
    `-race -count=3` for the 20 most allocation-sensitive packages nightly.

## F. Resilience and fault injection

41. **Deterministic failure injection.** A capability-forwarding
    `FailingPublisher` / `FailingStore` wrapper that fails the Nth call with
    each error family. The wrapper shape (explicit forwarding of every optional
    interface) is required anyway per the 2026-10-10 handoff lesson.

42. **Broker partition and redelivery matrix.** The Redis and NATS suites
    cover roundtrip and Nack. Add mid-flight disconnect, redelivery
    duplication, poison to DLQ, and a crash between database commit and outbox
    publish for the watermill Forwarder.

43. **Quarantine-and-recover property test.** Force N infrastructure failures,
    assert reroute, then assert reactivation happens ONLY after a clean
    `CatchUpEngine` rebuild (ADR-0137). No timeout-based reactivation may
    ever pass this test.

44. **Context-cancellation matrix.** Every public blocking call (`Dispatch`,
    `Load`, `Store.Get`, `Project`, `ServeSSE`) against pre-cancelled,
    mid-flight, deadline-exceeded, and nil contexts. Assert error family and
    no goroutine leak; the 2026-09-20 `select` coin-flip lesson generalizes to
    all of them.

45. **Encryption and signing negative corpus.** Tampered ciphertext, wrong
    key, swapped metadata, replayed old signature, truncated envelope: must
    fail as `Corruption`, never panic, never silently pass.

## G. Security and observability

46. **HTTP-surface injection pins.** SQL identifiers are fuzzed; metadata
    keys, stream IDs, SSE headers (CRLF), and catalog/docserver path traversal
    are not. Wire-form pins exist for storage rows; add raw-response pins at
    the HTTP layer.

47. **Span-shape assertions.** `docs/SPAN_NAMING.md` defines
    `{component}.{action}`. Use an in-memory exporter to assert span names,
    attributes, and that no high-cardinality attribute reaches a metric label.

48. **Companion auth-flow coverage.** `cqrs-htmx` and `go-appkit` are in-scope
    companions: WebAuthn, OAuth2, TOTP, CSRF, and CSP need end-to-end
    request/response tests. Only docserver CSP is browser-validated today.

## H. Test infrastructure itself

49. **Mutation-verify new regression tests, mechanically.** The "prove it goes
    red" discipline is documented but unenforced. Add a PR check that mutates
    the touched packages for new tests, plus targeted mutation scores for the
    Tier-0/1 modules (`id`, `record`, `event`, `decider`).

50. **Finish gate-script self-tests and standardize test fixtures.** Twenty-plus
    gates still lack `--self-test` (`check-arch`, `check-coverage`,
    `check-file-size`, and others). Add them where fault-injection or repair
    paths exist, and standardize `testdata` factories plus `t.Cleanup` plus
    run-unique naming so the 2026-10-23 persistent-surface lesson becomes
    structural instead of remembered.

---

## How to use this document

Each item is independently actionable. When picking work, prefer items that
remove a defect CLASS over items that fix one test: 1, 2, 9, 12, 13, 24, and
33 are in the first tier for that reason. Items 3 through 8, 10 through 11,
20, 26 through 27, 41 through 42, and 49 are the second tier. Everything else
is valuable but narrower or depends on a tier above it.

This document is a proposal record, not a live backlog. Promote accepted items
into `TODO_LIST.md` with an owner; do not let this file become a second,
drifting task list.
