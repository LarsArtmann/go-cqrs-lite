# Library Adoption Audit — What to Adopt, Keep, or Watch

**Date:** 2026-07-28
**Scope:** All 58 modules. Surveyed every hand-rolled subsystem against the Go policy stack (`how-to-golang` skill) and the Go ecosystem.

---

## Summary Matrix

| # | Subsystem | Location | LOC | Current | Recommendation | Priority |
|---|-----------|----------|-----|---------|----------------|----------|
| 1 | LRU cache | `decider/cache.go` | 126 | Hand-rolled (`container/list` + `map`) | **ADOPT otter/v2** | P0 |
| 2 | Circuit breaker | `middleware/circuit_breaker.go` | 243 | Hand-rolled (`atomic` + `sync.Mutex`) | **ADOPT failsafe-go** | P0 |
| 3 | Postgres integration tests | `stack/postgres/*_test.go` | — | `t.Skip` without `POSTGRES_TEST_DSN` | **ADOPT testcontainers-go** | P1 |
| 4 | Snapshot/golden tests | `eventtest/golden.go`, `cattest/catalog.go` | ~60 | Custom `AssertGolden` helpers | **ADOPT go-snaps** | P2 |
| 5 | Retry module | `retry/` | 217 | Hand-rolled (zero-dep, extract planned) | **KEEP** (extract per ADR-0064) | — |
| 6 | Dedup ring | `dedup/ring.go` | 95 | Hand-rolled ring buffer | **KEEP** (no library fit) | — |
| 7 | SQL migrations | `storage/migrations/` | — | `//go:embed .sql` | **KEEP** (correct for library) | — |
| 8 | SSE broker | `transport/http/sse.go` | — | `net/http` stdlib | **KEEP** (correct for library) | — |
| 9 | Singleflight | `decider/load.go` | — | `golang.org/x/sync/singleflight` | **KEEP** (already correct) | — |
| 10 | CLI tooling | `cmd/cqrs-lint/main.go` | — | `cmdguard` (on cobra) | **KEEP** (owner's wrapper) | — |
| 11 | SQL queries | `storage/**/*.go` | ~800 | Hand-written queries + manual scan | **KEEP** (dynamic SQL doesn't fit sqlc) | — |
| 12 | Security tooling | `.github/workflows/ci.yml` | — | gosec + govulncheck in CI | **KEEP** (already adopted) | — |

---

## TIER 1: ADOPT NOW (P0)

### 1. Otter v2 → Replace `decider/cache.go` LRU

**Why:** The Go policy BANS hand-rolled caches (`ristretto`, `bigcache`, `hashicorp/lru`, and hand-rolled
`container/list` implementations). `kv/cache.go` already uses `maypok86/otter/v2` correctly — `decider/cache.go`
is a **split brain** with the same project's own policy-compliant pattern sitting 3 modules away.

**What changes:**
- `decider/cache.go`: Replace `lruCache[State]` (`container/list` + `map`) with `otter.Cache[string, stateEntry[State]]`.
- The `StateCache[State]` interface stays identical — only the implementation changes.
- Otter is lock-free for reads (TinyLFU admission), faster than the mutex+linked-list approach.

**Cost:** ~126 LOC → ~50 LOC. Same public API. One new dep: `maypok86/otter/v2` (already in the workspace).

**Risk:** LOW — exact same pattern as `kv/cache.go:63-97`.

---

### 2. Failsafe-go → Replace `middleware/circuit_breaker.go`

**Why:** The Go policy mandates `failsafe-go/failsafe-go` for resilience. The hand-rolled circuit breaker
(243 LOC) is a reinvented wheel with a simpler feature set than failsafe-go's production-grade implementation:
count-based and time-based sliding windows, ratio/rate thresholds, half-open with configurable permits,
built-in metrics, and state-change listeners.

**What changes:**
- `middleware/circuit_breaker.go`: Replace `circuitBreaker` struct with `failsafe-go/circuitbreaker`.
- The public API (`CommandCircuitBreaker`, `EventCircuitBreaker`, `QueryCircuitBreaker`, `CircuitBreakerConfig`)
  stays the same — failsafe-go is wrapped behind the existing middleware signatures.
- `CircuitBreakerConfig` maps directly: `FailureThreshold` → `WithFailureThreshold`, `SuccessThreshold` →
  `WithSuccessThreshold`, `Timeout` → `WithDelay`, `IsFailure` → `HandleIf`.

**Bonus capability:** failsafe-go enables **composable resilience** — consumers can chain
`retry → circuitBreaker → timeout` in one executor, which the hand-rolled version cannot do.

**Cost:** ~243 LOC → ~80 LOC. One new dep: `failsafe-go/failsafe-go` in `middleware/go.mod`.

**Risk:** LOW-MEDIUM — need to preserve the exact middleware constructor signatures and error classification
(`errorfamily.WrapTransient`).

---

## TIER 2: ADOPT FOR QUALITY (P1-P2)

### 3. Testcontainers-go → `stack/postgres` Integration Tests (P1)

**Why:** Postgres tests currently `t.Skip("POSTGRES_TEST_DSN not set")` — they NEVER run locally and only
run in CI if the DSN is configured. `testcontainers-go` spins up a real `postgres:16-alpine` Docker container
in `go test`, so the real Postgres code path is exercised everywhere.

**What changes:**
- `stack/postgres/go.mod`: Add `testcontainers-go/modules/postgres` (test-only dep).
- Replace `postgresDSN(t)` (reads env var, skips if empty) with a testcontainer helper:
  ```go
  func postgresContainer(t *testing.T) (*postgres.PostgresContainer, string) {
      ctr, _ := postgres.Run(ctx, "postgres:16-alpine", postgres.BasicWaitStrategies())
      testcontainers.CleanupContainer(t, ctr)
      dsn, _ := ctr.ConnectionString(ctx, "sslmode=disable")
      return ctr, dsn
  }
  ```
- SQLite tests stay as-is (`:memory:` — no container needed, already correct).

**Cost:** Test-only dep. No library code changes. Tests that previously skipped now run.

**Risk:** LOW — test-only, requires Docker locally (already needed for CI).

---

### 4. Go-snaps → Replace Custom Golden Helpers (P2)

**Why:** Three separate `AssertGolden` implementations exist (`eventtest/golden.go`,
`catalog/internal/cattest/catalog.go`, `cmd/api-stability/main.go`). The policy mandates `go-snaps` for
snapshot testing. go-snaps provides automatic `-update` flag, inline diff, nested snapshots, and cleanup.

**What changes:**
- Replace each `AssertGolden(t, path, got, update)` call with `snapshot.MatchSnapshot(t, got)`.
- Remove the custom `update` flag registration (go-snaps handles it).

**Cost:** Migrating 3 helpers across modules. Each is simple (read file → compare bytes → write on update).

**Risk:** LOW but cross-cutting. Verdict: **WORTH CONSIDERING** — the custom helpers work, so this is
polish, not urgency.

---

## TIER 3: KEEP AS-IS (Intentionally Hand-Rolled)

### 5. Retry Module (`retry/`) — KEEP, extract per ADR-0064

`retry/` is a **zero-dep standalone module** (217 LOC, only depends on `go-error-family`). It serves a
different audience than failsafe-go: "I just want exponential backoff + jitter, no framework." The ADR-0064
plan extracts it to `github.com/larsartmann/go-retry`. This is NOT a reinvented wheel — it's an intentionally
minimal standalone library that happens to live in the monorepo for now.

**Decision:** Do NOT replace with failsafe-go. Extract per ADR-0064. `middleware/retry.go` continues wrapping
it for the CQRS middleware API.

---

### 6. Dedup Ring (`dedup/ring.go`) — KEEP

Fixed-capacity ring buffer for stream-boundary deduplication (replay→live overlap). Domain-specific:
O(1) Add/Has, memory-bounded at capacity, not thread-safe by design (single-goroutine consumption).
No library covers this niche — it's a data structure, not a reinvented wheel.

---

### 7. SQL Migrations — KEEP

`//go:embed` of `.sql` DDL files is the **correct pattern for a library**. The library applies schema on
construction; the consumer owns runtime migrations. golang-migrate/goose would add opinionated migration
infra that doesn't belong in a library.

---

### 8. SSE Broker — KEEP

`net/http` stdlib is correct for a transport library. No opinion on HTTP framework (that's the consumer's choice).

---

### 9. Singleflight — KEEP

Already uses `golang.org/x/sync/singleflight` — the standard, correct choice.

---

### 10. CLI Tooling (cmdguard/cobra) — KEEP

`cmdguard` is the owner's own struct-tag CLI wrapper built on cobra. Provides config files, struct tags,
and auto-help. `fang` (charm.land) is for user-facing CLIs; `cqrs-lint`/`cqrs-gen` are developer tools where
cmdguard's config-file support is more valuable than fang's TUI polish.

---

### 11. SQL Queries — KEEP

The `storage/` module generates **dynamic SQL** (view mappers, relational projections, auto-generated column
lists). sqlc is designed for static queries with known column sets — it cannot express the dynamic schema
generation that `storage/view/`, `storage/relational/`, and `storage/view/AutoMapper` perform.

---

### 12. Security Tooling — KEEP (already adopted)

`gosec` (SARIF output in CI) and `govulncheck` (release workflow, all modules standalone) are already wired.
`nix run .#vulncheck` runs locally. No gap here.

---

## What NOT to Adopt (Application-Framework Deps in a Library)

The Go policy stack (`gin+huma`, `koanf`, `charm.land/log`, `fang`, `samber/do`, `govalid`) is for
**applications**, not libraries. This is a CQRS **library/SDK** — adopting application-framework deps in
core modules would violate the "minimal dependencies" principle (Design Principle #1, #4, #12).

The `example/taskmanager` intentionally uses `net/http` stdlib to demonstrate the library is framework-agnostic.
An example showing gin+huma integration would be valuable, but it belongs in a **new example**, not in existing
library modules.

---

## Execution Priority

| Priority | Task | Effort | Impact |
|----------|------|--------|--------|
| **P0** | otter → decider/cache.go | 1h | Unifies cache strategy, policy compliance |
| **P0** | failsafe-go → circuit_breaker.go | 2-3h | Production-grade resilience, composable |
| **P1** | testcontainers-go → stack/postgres | 1-2h | Real DB tests run everywhere |
| **P2** | go-snaps → golden helpers | 2-3h | Standardized snapshot testing |
