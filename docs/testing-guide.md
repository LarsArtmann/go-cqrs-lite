# Testing Guide

> Patterns and conventions for testing go-cqrs-lite modules and consumer code.

## Test layers

| Layer          | When to use                                     | Example module                  |
| -------------- | ----------------------------------------------- | ------------------------------- |
| Unit           | Pure logic (decide, fold, helpers)              | `decider/`, `event/`            |
| Property       | Invariants across random inputs                 | `idempotency/kvstore`, `kv/`    |
| Integration    | Cross-module contracts                          | `integration/`                  |
| BDD / scenario | Behavior-focused decider/projection tests       | `scenario/`, `event/` (Ginkgo)  |
| Contract       | Shared suites run by every Store implementation | `kv/viewstoretest`, `eventtest` |
| Cross-engine   | Parity across memory vs SQLite backends         | `metaengine/`                   |

## Commands

```bash
# Full verify gate (build + vet + test + race + lint + api-stability + doc-check)
nix run .#verify

# Fast feedback (skips soak tests)
nix run .#verify-fast

# Parallel test execution
nix run .#verify-parallel

# Single module (workspace mode, required build tag)
go test -tags "goexperiment.jsonv2" ./event/... -count=1

# Single module (per-module isolation, mirrors CI)
cd event && GOWORK=off go test ./... -count=1

# Coverage for one module
cd decider && GOWORK=off go test -cover ./...

# Coverage drift check (compares actual vs AGENTS.md claims)
nix run .#check-coverage
```

> **Build tag:** all tests require `-tags "goexperiment.jsonv2"` in workspace mode.
> CI applies `GOEXPERIMENT=jsonv2` via the env, so the tag is implicit there.
> In `GOWORK=off` per-module mode, pass `-tags "goexperiment.jsonv2"` explicitly.

## Race-aware test thresholds

The `-race` detector inflates allocations and CPU 5–10x. Hardcoded timing or
heap thresholds in tests flake under `-race`. Use the `testutil.RaceEnabled`
build-tag constant to pick a relaxed bound:

```go
hang := 5 * time.Second
if testutil.RaceEnabled {
    hang = 30 * time.Second
}
```

`testutil/race_on.go` + `race_off.go` provide the constant. Modules with a
lean dependency budget that cannot import `testutil` (e.g. `benchkit`,
`transport/grpc`) copy the two-file idiom locally.

**After touching a threshold:** run the affected test 3x with `-count=3 -race`.

## Property tests (rapid)

Use [`pgregory.net/rapid`](https://github.com/flyingmutant/rapid) for invariant
testing. The canonical pattern (from `idempotency/kvstore/property_test.go`):

```go
func runPropertyAllStores(t *testing.T, fn func(t *rapid.T, store idempotency.Store)) {
    t.Helper()
    for name, factory := range allStores() {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            rapid.Check(t, func(rt *rapid.T) {
                store, cleanup := factory(t)
                defer cleanup()
                fn(rt, store)
            })
        })
    }
}
```

Reusable generators live in `testutil/rapidgen.go` (`EventType()`, `Version()`,
`NonEmptyString()`, `MetadataMap()`, `Timestamp()`).

**Existing property tests:** `idempotency/kvstore`, `kv/property_test.go`
(TypedStore + Cache invariants), `snapshot/property_test.go` (round-trip fidelity).

## Cross-engine parity (metaengine)

Verify identical results across `MemoryEngine` and `SQLiteEngine`:

```go
engines := map[string]metaengine.Engine{
    "memory": metaengine.NewMemoryEngine(),
    "sqlite": mustSQLiteEngine(t),
}
// Apply identical operations to each, then deep-compare results.
```

Covered: Map, Multimap, Log, struct results (`cross_engine_meta_test.go`),
LogTail + Graph (`concurrent_gaps_test.go`), Counter + Set
(`cross_engine_adt_test.go`).

## BDD (Ginkgo + scenario DSL)

For behavior-focused decider tests:

```go
scenario.Given[cmd, state](t, fold, initialState, priorEvents...).
    When(cmd, decideFunc).
    Then(expectedEvents...)

scenario.GivenProjection(t, proj, evt1, evt2).ThenNoError()
```

## Contract suites

Shared test suites that every implementation must pass:

- `event/v4/eventtest` — `FakeStore`, `FakeBus`, `FakeSnapshotStore`, golden assertions
- `kv/viewstoretest` — `RunSuite[V,K]` for `ViewStore` implementations
- `id/idtest` — `Parse*(tb, s)` helpers (no panics, `tb.Fatalf` on error)
- `query/querytest` — `New(tb, queryType)` helper

## Coverage

Core modules: 70–96% (decider 95.9%, storage/memory 94.2%, schema 89.9%).
Newer modules: 70–86% (kv 71.9%, codec 70.2%). Coverage is checked for drift
against AGENTS.md by `scripts/check-coverage.sh` (`nix run .#check-coverage`).

Run `nix run .#check-coverage -- --update` to recompute and print the
AGENTS.md-ready values, then update the `EXPECTED` map in the script.
