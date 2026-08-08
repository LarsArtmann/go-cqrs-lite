# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Correctness Bugs

- [x] **Add length-mismatch guard to `DecodeFloatResults`** — ✓ Aug 2026 (M2)
- [x] **Fix `context.Background()` in taskmanager handlers** — ✓ Aug 2026 (M2)
- [x] **Route DuckDB `plans` map reads through `lookupPlan()`** — ✓ Aug 2026 (M2)
- [x] **Fix `mustSQLiteEngine` zombie test helper** — ✓ Aug 2026 (M2)
- [x] **Delete `_skipped_sqlite_test_*` zombie functions** — ✓ Aug 2026 (M2)
- [x] **Fix `querytest.RunStoreSuite` / `querytest.StoreSuite` undefined** —
      ✓ Aug 2026. Root cause: `query/v4 v4.2.0` tag predates `store_suite.go`.
      `GOWORK=off` (CI per-module) couldn't see the symbols. Fix: added
      `replace query/v4 => ../../query` to storage/memory, storage/pebble,
      storage/bbolt go.mod (same pattern as decider→flightrecorder).
      _(Effort: M)_
- [x] **Fix `b029.go` / `b030.go` / `b031.go` compiler errors** — ✓ Aug 2026 (M12)

---

## cqrs-lint

- [ ] 🔥 **Run cqrs-lint against real consumer projects** — validate
      false-positive rates against 8 identified repos (Kernovia,
      Standup-Killer, bank-sync, cqrs-htmx, DiscordSync, timesheets,
      crush-daily, KeyHolderAI). The linter has 192 rules but zero real-world
      false-positive data.
      _(Effort: L)_
- [x] **Build type-checking test helper** — ✓ Aug 2026 (M11)
      `BuildContextWithTypes` implemented in `analyzer/test_helpers.go`.
- [x] **Self-lint CI: tighten severity gate** — ✓ Aug 2026 (M5)
- [x] **10 genuinely-missing rules** — ✓ Aug 2026 (M12-M14).
      B029-B031, D018-D019, F027-F029, C041-C042 all implemented,
      tested, registered, and cataloged. Total now 202 rules.
- [x] **Tag cqrs-lint v4.6.0** — ✓ Aug 2026 (M14). Tagged with 202 rules.

---

## Metaengine v2 — Dgraph Engine

> Dgraph engine integration-tested against live Dgraph 25.4.0 (2026-08-08).
> `nix run .#ephemeral-dgraph` spins up Zero+Alpha from nixpkgs. All 10 tests
> pass with `-race`. DQL injection fixed (`QueryWithVars`), MapDelete fixed
> (Dgraph 25.x explicit null-predicate deletion), regression test added.
> See `docs/status/2026-08-08_22-03_dgraph-integration-testing-and-performance.md`.

- [ ] 🔥 **Fix Dgraph calibration constants** — `DG_NsPerOp=10,000ns` and
      `DG_NsPerRead=8,000ns` are 100-270x too low vs real benchmarks
      (MapSet 2.7ms, MapGet 344µs, CounterIncrement 2.4ms). The planner
      will route queries to Dgraph that should go to Memory/SQLite.
      Either update to measured values or implement `Calibratable.Benchmark()`.
      _(Effort: S)_
- [ ] 🔥 **Add Graph and Search benchmarks** — only Map/Counter/Set are
      benchmarked (Dgraph's weaknesses). Graph traversal and full-text search
      are Dgraph's killer features but have zero benchmark coverage.
      _(Effort: S)_
- [ ] **Fix CounterGet O(N) scan** — returns ALL counters in a collection
      via client-side aggregation (3.4ms, 365KB, 1,143 allocs for 1000
      counters). Should use Dgraph server-side `sum(val(count))` aggregation.
      _(Effort: M)_
- [ ] **Add adversarial DQL injection test** — the regression test checks
      source patterns, not runtime behavior. Send keys containing DQL syntax
      (`func:`, `@filter`, `{ }`, quotes) and verify they're treated as
      literals by `QueryWithVars`.
      _(Effort: S)_
- [ ] **Add Dgraph to `test-all-backends.sh`** — currently lists SQLite,
      Pebble, bbolt, DuckDB, PG, MySQL. Add Dgraph (needs `pkgs.dgraph`
      in flake app `runtimeInputs`).
      _(Effort: S)_
- [ ] **Add Dgraph VM test** (`nix/vm/dgraph.nix`) — NixOS VM test for CI
      reproducibility, matching postgres-vm/mysql-vm/duckdb-vm pattern.
      _(Effort: M)_
- [ ] **Implement MultimapBackend for Dgraph** — model via repeated predicate
      values or edge lists.
      _(Effort: M)_
- [ ] **Implement LogBackend for Dgraph** — append-only ordered nodes.
      _(Effort: M)_
- [ ] **Add per-test data cleanup** — no `DropAll` or per-test cleanup exists.
      Stale data accumulates on persistent Dgraph instances.
      _(Effort: S)_
- [ ] **Tag `dgraphengine/v4.0.2`** — security fix (DQL injection) + MapDelete
      bugfix warrant a patch release.
      _(Effort: S)_

---

## Irohengine / Replicated Engine

- [x] **Add `WithClock` option to `replicatedEngine`** — ✓ Aug 2026 (M18)
- [x] **Add connection pooling to QuicTransport** — ✓ Aug 2026. Implemented
      `WithStreamPooling()` option: persistent BiStreams with length-prefix framing
      replace one-stream-per-op. ~30% latency reduction measured (91K vs 129K ns/op).
      Backward compatible (disabled by default). Tests: `TestQuicPooled_*`.
      _(Effort: M)_
      _(Source: `docs/status/2026-08-08_02-50_irohengine-quic-parity-and-flake-fixes.md`)_
- [x] **Add MapDelete LWW convergence test** — ✓ Aug 2026 (M4). Hardened with
      injectable `WithClock` (deterministic timestamps, no `time.Sleep`).
- [x] **Add graceful shutdown test** — ✓ Aug 2026 (M4). Expanded with 50
      concurrent in-flight ops verified before Close, plus post-close safety check.
- [ ] **Add runtime protocol-mismatch detection for QUIC stream pooling** — a
      pooled sender connected to a non-pooled receiver silently hangs (receiver
      calls `ReadToEnd` waiting for `Finish()` that never comes). Detect via a
      magic byte in the first frame and return a clear error.
      _(Effort: S)_
      _(Source: `docs/status/2026-08-08_21-45_irohengine-clock-pooling-test-hardening.md`)_
- [ ] **Add stream-reuse counter to `peerConn`** — increment each time
      `sendOpPooled` opens a new BiStream. Tests can assert that N ops over a
      pooled connection used only 1 stream (proving reuse, not just correctness).
      _(Effort: S)_
      _(Source: `docs/status/2026-08-08_21-45_irohengine-clock-pooling-test-hardening.md`)_
- [ ] **Extract shared framing constants** — `frameHeaderSize`, `errFrameTooLarge`
      are duplicated between `quic/frame.go` and `loopback/frame.go`. Move to
      `irohengine/framing.go` (protocol constants only; I/O stays per-transport).
      _(Effort: S)_
      _(Source: `docs/status/2026-08-08_21-45_irohengine-clock-pooling-test-hardening.md`)_
- [ ] **Port injectable-clock pattern to QUIC LWW tests** — `TestQuicLWWResolution`
      still relies on replication-time-gap for timestamp ordering. Could use
      `WithClock` for determinism (same pattern as the in-process tests).
      _(Effort: S)_
      _(Source: `docs/status/2026-08-08_21-45_irohengine-clock-pooling-test-hardening.md`)_

---

## Code Quality / Dedup

- [ ] **Per-module `.golangci.yml` split** — golangci-lint v2 `config-dirs`
      would give each module ownership of its own exclusions. The monolithic
      config is documented but sprawls across 30+ blocks.
      _(Effort: L)_
- [ ] **Add per-entry rationale comments to EXCEPTIONS** — the remaining 7
      entries in `scripts/check-module-layers.sh` have only a generic header
      comment. Each entry should explain WHY the exception is legitimate.
      _(Effort: S)_
      _(Source: `docs/status/2026-08-08_21-29_deferClose-exceptions-cleanup-session.md`)_
- [ ] **Add `TestExceptionsAreMinimal` meta-test** — automate dead-exception
      detection: remove EXCEPTIONS entries where `dep_layer <= mod_layer`
      (same/lower-layer deps don't trigger violations). Prevents the
      `schema→snapshot` and `transport/http→testutil` class of stale entries.
      _(Effort: S)_
      _(Source: `docs/status/2026-08-08_21-29_deferClose-exceptions-cleanup-session.md`)_

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must
  v0.1.2).
- [x] **Pin GitHub Actions to commit SHAs** — ✓ Aug 2026 (M15)
- [x] **Add `--fail-on-stale-suppressions` CI gate** — ✓ Aug 2026 (M5)
- [x] **Add CI check for API-version drift** — ✓ Aug 2026 (M16).
      `scripts/check-tag-existence.sh` added.
- [x] **Add calibration benchmark regression baseline** — ✓ Aug 2026 (M22).
      `metaengine/calibration-baseline.md` created.
- [x] **Add `duckdb-vm` and `turso-vm` to CI `nixos-vm-tests` job** — ✓ Aug 2026 (M5)

---

## Integration Test Infrastructure

- [ ] **macOS verification of ephemeral PG** — `scripts/ephemeral-pg.sh` claims
      cross-platform but was never tested on Darwin. (M34)
      _(Effort: M)_
- [ ] **Write actual Redis/NATS integration tests** — `ephemeral-redis.sh` and
      `ephemeral-nats.sh` exist but no tests use them. Watermill Redis Streams
      and NATS JetStream roundtrips untested.
      _(Effort: M)_
      _(Source: `docs/status/2026-08-08_10-18_m35-m48-integration-test-infrastructure.md`)_
- [x] **Add bbolt backup/restore test** — Pebble has `backup_lifecycle_test.go`;
      bbolt now has equivalent coverage (events + snapshots + checkpoints +
      incremental backups).
      _(Effort: S)_ ✓ Aug 2026

---

## Layer Enforcement

- [x] **Delete stale FOUR-TIER-MODEL.d2/.svg artifacts** — ✓ Aug 2026 (M3)
- [x] **Add intra-module architecture config for `cmd/cqrs-lint`** — ✓ Aug 2026
      Added `.go-arch-lint.yml` with 5-layer model covering all 18 packages:
      L0 leaves (analyzer/fix/ruletest/suppression) → L1 lintutil → L2 rule
      categories → L3 rules root → L4 main. Enforced via `scripts/check-arch.sh`.
- [ ] **Consider rewriting `check-module-layers.sh` as `cmd/check-layers`** —
      348 lines of bash. A Go program would add testability but the script is
      stable and only runs in CI. Defer until significantly more complex.
      _(Effort: L)_

---

## Declined / Rejected (do not re-litigate)

> Full rationale in the linked ADRs/reviews.

- **Wire `#verify-parallel` into CI** — declined 2026-07-29. CI already has a
  per-module matrix strategy that provides better isolation.
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Use
  `RelationalProjection` (junction tables). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
- **Redis adapter** — the author is not a fan of Redis. See ROADMAP Non-Goals.
