# Status Report — 2026-08-04 10:28

## Session Goal

Two milestones from the TODO list:

- **M43**: projectionhost crash-restart PG integration test — verify checkpoint replay after crash
- **M44**: scheduling durable timers across restarts test — timer survives process restart

---

## A) FULLY DONE

### M44: `scheduling/sqlstore` — Durable SQL Timer Store (NEW MODULE)

Created an entirely new Go module `scheduling/sqlstore/` following the established `idempotency/sqlstore` pattern.

**Files created:**

| File                                    | Purpose                                                                                           |
| --------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `scheduling/sqlstore/store.go`          | `SQLTimerStore[P]` — generic SQL-backed `TimerStore[P]` with 3 dialects (SQLite, Postgres, MySQL) |
| `scheduling/sqlstore/store_test.go`     | 5 CRUD tests: Schedule+Due ordering, idempotent schedule, MarkFired, Cancel, empty Due            |
| `scheduling/sqlstore/restart_test.go`   | 2 durability tests: store-level restart recovery + full Scheduler integration recovery            |
| `scheduling/sqlstore/go.mod` / `go.sum` | Standalone module (scheduling/v4 + go-error-family + modernc/sqlite for tests)                    |
| `scheduling/sqlstore/README.md`         | Consumer-facing docs with quickstart, durability explanation, API table                           |

**Design:**

- Payloads serialized as JSON into `payload` BLOB/BYTEA column
- Time formatting is dialect-aware: SQLite uses fixed-width RFC3339 text (lexicographic ordering), Postgres/MySQL use native `time.Time` / `DATETIME(3)`
- Idempotent Schedule via `ON CONFLICT(id) DO NOTHING` (SQLite/PG) / `ON DUPLICATE KEY UPDATE id=id` (MySQL)
- DDL matches the `timers` table already embedded in `storage/sql/migrations/` — no schema conflict for dual consumers
- Error wrapping via `go-error-family` (Corruption for marshal/parse failures, Infrastructure for SQL errors)

**Tests (7 total, all PASS including -race):**

1. `TestSQLiteTimerStore_ScheduleAndDue` — 3 timers, ordering, payload round-trip
2. `TestSQLiteTimerStore_ScheduleIsIdempotent` — re-schedule is no-op, original payload preserved
3. `TestSQLiteTimerStore_MarkFired` — timer removed after dispatch
4. `TestSQLiteTimerStore_Cancel` — timer removed before fire
5. `TestSQLiteTimerStore_EmptyDue` — nil slice on empty store
6. **`TestSQLiteTimerStore_SurvivesRestart`** (M44 core) — schedule timer → close DB → wait → reopen DB → timer present and due with correct payload → MarkFired clears it
7. **`TestSQLiteTimerStore_SchedulerIntegration_Recovery`** — full end-to-end: schedule → crash → restart Scheduler → overdue timer dispatched + marked fired

**Coverage:** 66.1% of statements (uncovered: MySQL/Postgres dialect branches + error paths — see improvements)

**Wiring:**

- Added to `go.work` (was already present)
- Added to `cmd/api-stability/main.go` modules list (was already present)
- Regenerated `docs/api_surface.txt` (13 new exports)
- Added to AGENTS.md: modules list, test command, structure tree, module count (64→65)

### M43: `projectionhost` PG Crash-Restart Integration Test

**Files created:**

| File                                      | Purpose                                                                                                   |
| ----------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `projectionhost/pg_integration_test.go`   | `//go:build integration` — `TestIntegration_ProjectionHost_CrashRestart_CheckpointReplay`                 |
| `projectionhost/pg_testcontainer_test.go` | `//go:build integration` — TestMain + shared container + per-test DB isolation (mirrors storage/ pattern) |

**The crash-restart test proves:**

1. Seed 10 events → host1 processes all 10 → checkpoint persisted to PG
2. Stop host1 (simulated crash)
3. Seed 5 MORE events while host is down
4. Start host2 with SAME PG checkpoint store
5. host2 processes ONLY the 5 new events (not 15) — checkpoint recovery works

**Wiring:**

- Added `projectionhost` to `scripts/ephemeral-pg.sh` PG_MODULES
- Added `pgx/v5` + `testcontainers-go` + `testcontainers-go/modules/postgres` to `projectionhost/go.mod`
- Verified against real PostgreSQL via `nix run .#integration-pg` — PASS

**Verification gate (this session):**

- `go build -tags "goexperiment.jsonv2" ./...` — clean
- `go vet` on both modules — clean
- `go test -race` on both modules — PASS
- `nix run .#integration-pg` — all 4 projectionhost integration tests PASS
- `doc-check` — 521 references valid
- `api-stability` golden regenerated + meta-test PASS
- `check-module-layers.sh` — no new violations (pre-existing retry violation unrelated)
- `gofumpt` + `goimports` applied

---

## B) PARTIALLY DONE

### MySQL support — code exists, no syntax test

`idempotency/sqlstore` ships `mysql_queries_test.go` verifying MySQL SQL syntax (backtick quoting, `ON DUPLICATE KEY UPDATE`, `IF()` conditional) without a live MySQL connection. My `scheduling/sqlstore` has `mysqlQueries()` but **no equivalent syntax test**. The MySQL SQL path is completely unverified.

### Postgres timer store — code exists, no PG integration test

`NewPostgresStore[P]` exists and builds, but **`scheduling/sqlstore` is NOT in `scripts/ephemeral-pg.sh` PG_MODULES**. The Postgres dialect path (native `time.Time` scanning, `$N` placeholders, `BYTEA` column) has never been tested against a real PG instance. This is the single biggest gap.

---

## C) NOT STARTED

### Property-based testing

`idempotency/sqlstore` has `property_test.go` using `pgregory.net/rapid` for concurrent stress testing. No equivalent exists for `scheduling/sqlstore` — concurrent Schedule/MarkFired/Due races are untested.

### Benchmarks

No performance benchmarks for the SQL timer store (insert throughput, Due scan latency at scale).

### Codec option

The store hardcodes `encoding/json` for payload serialization. The project defaults to CBOR in many modules. A `WithCodec(codec.Codec)` option would allow consumers to choose CBOR for ~35% smaller payloads. Design decision deferred — JSON is more debuggable in DB tools.

---

## D) TOTALLY FUCKED UP

Nothing. Both milestones delivered, tested, and passing. No broken builds, no stale claims.

---

## E) WHAT WE SHOULD IMPROVE

1. **`scheduling/sqlstore` is NOT in `ephemeral-pg.sh` PG_MODULES** — the Postgres path is 100% untested against real PG. This is the most critical gap. One line fix.
2. **No MySQL syntax test** — `idempotency/sqlstore` has one; `scheduling/sqlstore` doesn't. Copy the pattern.
3. **Coverage is 66.1%** — the Postgres/MySQL dialect branches and several error paths are uncovered. Adding PG integration + MySQL syntax test would close this.
4. **`SchedulerIntegration_Recovery` test takes 2.05s** — the scheduler's blocking `Start` with a 2s context timeout makes this slow for a unit test. Could use a shorter timeout + signal-based completion.
5. **The M43 test uses in-memory event store** — checkpoint recovery is isolated correctly, but a more complete test would use PG for BOTH event store + checkpoint store (proving the full SQL stack survives restart).
6. **Duplicate `requireEventually` helper** — defined in `scheduling/sqlstore/restart_test.go` AND `projectionhost/host_test.go`. Different modules so no conflict, but it's the same 10-line pattern copy-pasted. Could live in `testutil/` — but that would add a dep.
7. **No `Schedule` return distinction** — caller can't tell if a timer was newly inserted vs. already existed (silent no-op). The `MemoryTimerStore` has the same behavior, so this is consistent — but `idempotency.Store` returns `ErrDuplicate` for the analogous case. Worth considering for future API maturity.
8. **Full `nix run .#lint` not run** — I ran `gofumpt` + `goimports` locally but did NOT run the full golangci-lint gate. Should verify no lint issues in the new code.

---

## F) Up to 50 Things to Get Done Next

### Immediate (this session's gaps)

1. Add `scheduling/sqlstore` to `scripts/ephemeral-pg.sh` PG_MODULES
2. Write PG integration test for `scheduling/sqlstore.NewPostgresStore` (restart durability against real PG)
3. Write `mysql_queries_test.go` for `scheduling/sqlstore` (syntax verification without live MySQL)
4. Run full `nix run .#lint` on new code
5. Run `nix run .#verify` or `nix run .#verify-fast` for the full gate

### M44 maturation

6. Add property-based test (rapid) for concurrent Schedule/Due/MarkFired
7. Add benchmarks (insert throughput, Due scan at 1K/10K/100K timers)
8. Add `WithCodec(codec.Codec)` option for CBOR payload support
9. Consider returning `(bool inserted, error)` from Schedule to distinguish new vs duplicate
10. Test timer payload with complex/nested structs (current test uses flat struct)
11. Test `Due` with zero-value `time.Time` (edge case — all timers due?)
12. Test timer FireAt in the past (immediately due)
13. Add `ListAll` method for admin/debug visibility (current API can only query Due)
14. Add `Count` method for health checks
15. Add Sweep/TTL mechanism for orphaned timers (timers whose process crashed before MarkFired)

### M43 maturation

16. Add PG-backed event store variant (full SQL stack crash-restart)
17. Add crash-restart with DLQ (poison event survives restart)
18. Add multi-projection crash-restart (multiple projections, independent checkpoints)
19. Add crash-mid-batch test (host crashes between checkpoint saves — verify no double-processing)
20. Test projectionhost with Pebble-backed checkpoint store across restart

### Integration infrastructure

21. Add `scheduling/sqlstore` to MySQL VM test (`nix run .#integration-mysql-vm`)
22. Add projectionhost to MySQL VM test
23. Consider a shared testcontainer harness across modules (reduce duplication of `pg_testcontainer_test.go`)

### Documentation

24. Add `scheduling/sqlstore` usage example to SKILL.md recipes
25. Update `docs/architecture-understanding/FOUR-TIER-MODEL.md` with scheduling/sqlstore position
26. Add ADR for scheduling/sqlstore (following ADR-0065 pattern for idempotency/sqlstore)
27. Update FEATURES.md with the new module status
28. Update TODO_LIST.md — mark M43/M44 as done

### Testing quality

29. Add `TestSQLiteTimerStore_PayloadCorruption` — manually corrupt payload BLOB, verify Corruption error
30. Add `TestSQLiteTimerStore_ConcurrentSchedule` — two goroutines schedule same ID, verify exactly one wins
31. Add `TestSQLiteTimerStore_TimezoneRoundTrip` — verify FireAt survives UTC↔local conversion
32. Add fuzz test for payload serialization
33. Add stress test: 10K timers scheduled, Due query performance

### Code quality

34. Verify `created_at` column is populated correctly across all dialects
35. Add `context.Context` propagation verification (cancel mid-query)
36. Verify connection pool behavior under load
37. Consider prepared statement caching for hot queries

### Release

38. Tag `scheduling/sqlstore/v4.0.0` (first release)
39. Verify `git tag -l 'scheduling/sqlstore/v4*'` before tagging
40. Update `cmd/cqrs-lint` module catalog (28 scored modules → 29)
41. Run `nix run .#vulncheck` on the new module
42. Run `nix run .#secrets-scan`

### Broader project

43. Run full `nix run .#verify` gate (3-4 min)
44. Check if `flake.nix` testModules needs updating for the new module
45. Verify `nix run .#check-coverage` passes with the new module's 66%
46. Consider extracting the shared `pg_testcontainer_test.go` into a test helper module
47. Run `nix flake check`
48. Update `CONTRIBUTING.md` module list if needed
49. Check if `cmd/doc-check` needs the new README added to its scan paths
50. Consider whether `scheduling/sqlstore` should be re-exported from a `stack/` preset

---

## G) Questions I CANNOT Answer Myself

1. **Should the SQL timer store use the `codec/` module (CBOR default) instead of raw `encoding/json`?** The project defaults to CBOR in most blind stores (kv, snapshot, event.New), but the timers table payload is a command payload — JSON makes it debuggable in DB tools. The `idempotency/sqlstore` doesn't use codec either (it stores UnixNano ints, not serialized payloads), so there's no direct precedent. What's the right call?

2. **Should `scheduling/sqlstore` be added to the MySQL VM test suite (`nix run .#integration-mysql-vm`)?** The MySQL path is implemented but the MySQL VM test is heavy (QEMU VM, required because MariaDB init is broken on NixOS). Is the MySQL syntax test sufficient, or do you want live MySQL integration coverage?

3. **Should the M43 crash-restart test also use a PG-backed event store (not just checkpoint store)?** The current test isolates checkpoint recovery using an in-memory event store — the event store durability is already covered by `storage/pg_integration_test.go`. But a full-stack version (PG events + PG checkpoints + PG read model) would be a stronger end-to-end proof. Worth the extra complexity?
