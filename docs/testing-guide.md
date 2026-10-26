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
go test ./event/... -count=1

# Single module (per-module isolation, mirrors CI)
cd event && GOWORK=off go test ./... -count=1

# Coverage for one module
cd decider && GOWORK=off go test -cover ./...

# Coverage floor for every module AND every package (89 modules, 168 packages)
nix run .#check-coverage

# The rest of the hygiene gates (all run in #verify and CI)
nix run .#check-untested-exports    # exported API named by no test (frozen debt list)
nix run .#check-test-sleeps         # 182 frozen time.Sleep sites; new ones fail
nix run .#check-test-skips          # every skip needs a reason; --report routes env gates
nix run .#check-systemscenario-adoption  # boot a system in a test -> use the harness or say why

# Shuffled suites: seed is logged, replay is one command
nix run .#test                       # -shuffle=<minted seed>, logged
nix run .#rerun-shuffle-seed -- 123456789 ./event/...
nix run .#test-repeat                # -count=5 -cpu=1,4 over 20 stateful modules (nightly)

# Fuzzing: rotate through all 115 targets (every target within 4 nights)
nix run .#fuzz-rotation -- --shard 1/4
nix run .#fuzz-rotation -- --list
```

## Integration tests (database)

Three Docker-free approaches, powered by Nix. See [ADR-0095](adr/0095-nix-based-integration-testing.md).

| Approach            | Command                                          | Speed              | Docker? | Platforms    |
| ------------------- | ------------------------------------------------ | ------------------ | ------- | ------------ |
| Ephemeral PG        | `nix run .#integration-pg`                       | ~3s startup        | No      | Linux, macOS |
| NixOS VM (PG)       | `nix build .#checks.x86_64-linux.postgres-vm -L` | ~17s               | No      | Linux        |
| NixOS VM (MySQL)    | `nix build .#checks.x86_64-linux.mysql-vm -L`    | ~15s               | No      | Linux        |
| VM launcher (PG)    | `nix run .#integration-pg-vm`                    | ~25s to first test | No      | Linux        |
| VM launcher (MySQL) | `nix run .#integration-mysql-vm`                 | ~20s to first test | No      | Linux        |

> **MariaDB limitation**: Ephemeral MySQL without a VM is impossible on NixOS —
> `mariadb-install-db` fails (read-only Nix store plugin dir). VM is required.

> **Build tag:** none since Go 1.27 graduated `encoding/json/v2` (2026-09-19
> sweep removed the former `-tags "goexperiment.jsonv2"` everywhere). Plain
> `go test` is the contract in workspace AND `GOWORK=off` per-module mode.

### Example output (ephemeral PG)

```
$ nix run .#integration-pg
==> Initializing ephemeral PostgreSQL (port 55432, data /tmp/cqrs-pg-XXXXXX)
==> Starting PostgreSQL
==> PostgreSQL ready: postgres://cqrs@127.0.0.1:55432/cqrs_test?sslmode=disable

--- storage ---
=== RUN   TestPostgresEventStore_CRUD
--- PASS: TestPostgresEventStore_CRUD (0.03s)
PASS

✅ Integration tests passed
```

### Example output (NixOS VM check)

```
$ nix build .#checks.x86_64-linux.postgres-vm -L
machine # starting
(machine # machine starting
(machine # finished activating
(postgresql-vm # machine waiting for unit 'postgresql.service'
OK
```

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

Every module in `flake.nix` `testModules` (89 at the 2026-10-26 census) has a
**floor** in `scripts/coverage-baseline.txt`. Before that date the gate pinned
11 modules, so 78 modules could lose their tests without a red run.

```bash
nix run .#check-coverage                       # gate: every module vs its floor
nix run .#check-coverage -- --update           # rewrite the floors (quiet window!)
nix run .#check-coverage -- --report build/coverage --quiet   # CI artifacts
nix run .#check-coverage -- --self-test        # fault-inject the compare logic
nix run .#check-coverage -- --profile x.out    # reuse a merged coverprofile
```

What the gate fails on: a regression beyond the lower tolerance (`-2.0` points
locally, `-6.0` on CI where timeout-skipped tests measure 3-5 points lower), a
module in `testModules` with **no** baseline row, a baseline row that no longer
resolves to a module, a module that gained a package **without tests**, and
coverage data vanishing for a module that had some. Upward drift is reported
`STALE` but does not fail — new tests are never a defect.

Measurement mode is `aggregate`: one workspace test run emits a merged
coverprofile and a module's percentage is covered/total **statements** for the
files it owns (deepest module directory wins, so `metaengine/pgengine` is never
counted inside `metaengine`). The old implementation took the *first package's*
coverage line, which in a multi-package module is an arbitrary package — that is
why the pre-2026-10-26 numbers (e.g. metaengine 83.9) do not match the aggregate
ones (70.7).

`notest` in each row counts packages with no test files: the floor cannot be
satisfied by adding untested packages. A row with mode `none` records a module
with no test statements at all as an explicit decision instead of an invisible
gap.

Optional artifacts: `--report DIR` writes `module-coverage.tsv`,
`pkg-coverage.tsv` and `coverage-report.md`; CI uploads them as the
`coverage-report` artifact on every run.

## Flake gates

Three gates make the difference between "the suite is green" and "the suite
would catch a problem":

| Gate | Fails on | Notes |
| --- | --- | --- |
| `check-test-sleeps` | a new `time.Sleep` in a test, or a growing count in a baselined file | Poll the condition (`Eventually`, a channel, a clock advance). A real quiet window keeps `//go-sleep:accept <reason>`. |
| `check-test-skips` | a skip with no reason (or an empty `//go-skip:accept`) | `--report` prints the env-var route table: a skip gated on an env var that no workflow (or routed flake app) sets only runs on a human's box. That report is how the NATS and MySQL integration suites were found to be unrouted. |
| `check-test-assertions` | — not implemented (see below) | A text scan flagged 1,300 of 7,556 tests, nearly all false positives (they assert through package helpers). Mutation testing is the honest detector; it is TODO_LIST row T10. |

`-shuffle` is on by default for unit suites. Every run logs an explicit seed to
`build/shuffle-seeds.log`; a failure prints the exact replay command. `SHUFFLE=off`
exists only for bisecting a seed loop.

Goroutine leaks are enforced by `goleak.VerifyTestMain` in every module that
spawns goroutines in production code (13 at the 2026-10-26 census: `watermill`,
`storage/{pebble,turso,bbolt}`, `deriver`, `scheduling`, `decider`, `middleware`,
`stack`, `systemscenario`, `metaengine`, `projectionhost`, `system`,
`systemtest`). Ginkgo's signal handler is the single documented ignore.

## Fuzzing

`scripts/fuzz-targets.txt` inventories all 115 targets (`--update-targets`
regenerates it; the nightly `inventory` job fails when it is stale, so a new
`func Fuzz*` cannot silently skip fuzzing). The nightly runs four shards, one per
day of a four-day rotation: target `i` belongs to shard `i % 4`, and night
`day-of-year % 4` runs the matching one. Corpora persist through the actions
cache, so a crash found tonight is a regression case tomorrow.

## Performance ceilings

`scripts/benchmark-regression.sh` has two independent failure modes:

- the **delta gate** (default 25% median regression vs the committed baseline),
  which cannot see both runs being slow, and
- the **absolute ceiling** (`benchmarks/latency-ceilings.txt`, 21 gate
  benchmarks at 3x the committed baseline). Regenerate with
  `./scripts/benchmark-regression.sh --write-ceilings benchmarks/latency-ceilings.txt`
  after an accepted baseline move. Hardware- and runner-specific, like the
  baseline itself.

## System-level tests

`systemscenario` (ADR-0153) is the system tier: `system.New` with a journal, a
projection host and a bus, then assertions on observable outcomes. A module
whose tests boot a real system must use it or be allowlisted with a reason
(`scripts/systemscenario-adoption-allowlist.txt`). At the 2026-10-26 census nine
modules boot a system and seven are allowlisted (unit-level composition-root
tests, real-engine crash/durability suites, and derivation tests with no
dispatch).
