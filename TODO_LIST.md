# TODO List

**Updated:** 2026-07-27
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task is finished it is removed from
this list and recorded in CHANGELOG.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- `⭐` = Top 1% impact (do first)
- `⚠️` = Partially done — needs completion

---

## Verify Gate

> ✅ **`nix run .#verify` is GREEN end-to-end** (re-confirmed 2026-07-27 after
> fixing a hidden cqrs-lint build break: go-output root was bumped to v0.33.0
> by the auto-daemon but `go-output/table` has no v0.33.0 release — downgraded
> root back to v0.32.0). api-stability golden verified: `EnsureSQLiteDSNBusyTimeout`
> IS present at `docs/api_surface.txt` (2676 exports verified); the prior
> "golden anomaly" was a wrong-path grep (`cmd/api-stability/` vs actual
> `docs/api_surface.txt`). Race-aware thresholds, DSN-level SQLite `busy_timeout`,
> and `soakTestScale` consolidation all in place. File-size gate GREEN.
>
> Coverage drift is now checked by `scripts/check-coverage.sh`
> (`nix run .#check-coverage`) — AGENTS.md coverage claims verified 2026-07-27.

---

## Release

> ✅ **v4.2.0 RELEASED** (2026-07-27). 53 modules tagged and pushed. The
> CHANGELOG `[v4.2.0]` section has shipped; a fresh `[Unreleased]` is open.
> `metaengine/projectionadapter/v4.0.0` re-tagged at `be818c91` (was orphaned).
> `codec/v4.2.0` tagged alongside v4.1.1 (semver correction — both kept).
> `cmd/cqrs-lint` and `example/taskmanager` deps fixed (go-finding pseudo-versions
> → real v1.4.0, go-must v0.1.2) so the release batch could strip local replaces.

- [ ] **Run `nix run .#vulncheck`** — verify no known vulnerabilities across
      all module deps after v4.2.0.
- [ ] **Verify v4.2.0 tags resolve from a clean module** — `cd /tmp && mkdir
test-resolve && cd test-resolve && go mod init test && GOWORK=off go get
github.com/larsartmann/go-cqrs-lite/event/v4@v4.2.0`. Confirm every
      published module resolves without workspace-local replaces.
- [ ] **Publish go-finding + go-must as tagged modules** — the go.mod replace
      directives are needed for dev; consumers resolving the published modules
      depend on the real tagged versions (go-finding v1.4.0, go-must v0.1.2).

---

## CI / Daemon

> CI now runs: format, build, vet, test, test-race, lint, **api-stability**,
> **duplication** (`#check-duplication`), **dependency-layers** (`#check-layers`),
> **coverage-drift** (`#check-coverage`), doc-check, doc-assertions, coverage.
> Still not wired: `#verify-parallel` (speed), `#verify-fast` (pre-merge fast path).

- [ ] **Wire `#verify-parallel` into CI** — the app splits module tests into N
      batches for concurrent execution (~4min → ~1-2min); CI still runs
      sequential. Low-risk optimization, but the sequential path is proven.
- [ ] **Add `#verify-fast` as a pre-merge CI gate** — fast feedback (skips
      soak tests), keep full `#verify` for nightly.
- [ ] **Recurring lint-sweep** — the auto-commit daemon occasionally commits
      unformatted code (gci/gofumpt drift), turning `#lint` red. The `#sweep`
      app recovers, but gating daemon commits behind `nix fmt` prevents the
      drift. Either gate the daemon or run a scheduled sweep. The hidden
      cqrs-lint build break (go-output v0.33.0 daemon bump) is exactly this
      failure mode — discovered 2026-07-27 after 3+ sessions of stale "green
      gate" claims.
- [ ] **Investigate dependabot alert** `security/dependabot/10` — `gh api`
      returned no results (auth issue). Cannot diagnose without GitHub token
      permissions.

---

## Module Health & Tooling

- [ ] **Run cqrs-lint against the real codebase** — C015/C016/D006 were shipped
      in v4.2.0 with only the meta-test guard (instantiate-all-detectors).
      `cd cmd/cqrs-lint && go run . ../../...` — check the 3 new rules for false
      positives. D006 especially could flag legitimate sentinel errors; if it
      spews hundreds of findings, tighten `isPackageLevelVar` exemption logic
      in `cmd/cqrs-lint/pkg/rules/consistency/d006.go`.
- [ ] **Update cqrs-lint README rules table** — C015/C016/D006 were added to the
      source (`register.go`, `catalog.go`) but never documented in
      `cmd/cqrs-lint/README.md`. Consumers reading the README see 62 rules; the
      actual count is 65.
- [ ] **Consolidate remaining 5 `wrapClosed()` sites** — the 2026-07-27 session
      converted 12 of 17 sites across `store.go`, `command_store.go`,
      `query_store.go`, `snapshot.go` into `withWriteLock`/`withReadLock[T]`
      helper pairs (clone groups: 34 → 19). Remaining: `checkpoint.go` (2,
      `wrapClosedf` format variant) + `store_load.go` (3, mixed read/write).
      Same pattern — straightforward follow-up.
- [ ] **Add SortedMap to cross-engine parity tests** — the TODO said 4 ADTs to
      test; `metaengine/cross_engine_adt_test.go` covers Counter + Set (2 of 4).
      SortedMap routes through `MapBackend`/`ScanBackend` (no dedicated backend
      interface). Test via `ExecuteTyped` with a sorted result type.
- [ ] **Update AGENTS.md with dedup helper patterns** — the 2026-07-27 dedup
      session introduced `withWriteLock(code, msg, fn)` closures in
      `storage/memory`, `parallelTimeoutCtx(t, timeout)` in benchkit,
      `parallelViewStore(t)` in storage/view, variadic `NewTestRegistry(svc...)`
      in catalog — none documented in AGENTS.md.
- [ ] **Audit accepted clone groups** — verify 19 art-dupl groups genuinely
      acceptable, not just tolerated (was 34 before the 2026-07-27 wrapClosed
      consolidation; now 19 after the withWriteLock/withReadLock extraction).
- [ ] **`--structural` art-dupl pass** — AST-shape clones beyond the semantic
      mode the current gate uses.
- [ ] **`--type-aware` art-dupl run** — eliminates false-positive clone groups
      (`time.Time.String` vs `*big.Int.String`).

---

## Documentation

- [ ] **Write `docs/performance.md`** — benchmark results, expected throughput
      (stretch — lower priority than testing-guide + release-checklist, which
      shipped 2026-07-27).

---

## Declined / Rejected (do not re-litigate)

> Kept here so decisions are not re-litigated. Full rationale in the linked
> ADRs/reviews.

- **Strengthen envelope magic string (`"cqrs" → "cqrs-envelope-v1"`)** — the `"$"`
  JSON key provides 99% of collision avoidance. Extra bytes per record for
  near-zero benefit.
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Composite keys
  are relational territory — use `RelationalProjection` (junction tables,
  multi-table atomic writes). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
  `OrClause`/`NotClause`/nested groups is ORM creep. Principle #1: "Library, not
  framework."
- **Unify VersionedStore + VersionedSeekableJournal** — different interfaces
  (Store: Load/Save per stream, SeekableJournal: ReadFrom position-based). YAGNI.
- **VersionedJournal (ReadAll only)** — no consumer needs `ReadAll` with
  upcasters. YAGNI.
- **Expose `SSEBroker.PayloadTransform()` accessor** — implemented for
  BackfillHandler (necessary), but no standalone demand.
- **`WithPayloadTransform` on SSEHandler** — duplicates responsibility (SRP
  violation); SSEHandler wraps the broker.
- **Auto-apply CQRS views by default** — violates "library, not framework."
- **VersionedSeekableJournal implementing event.Store** — different scope
  (position-based vs stream-based). YAGNI.
- **Integration test in `integration/` module** — redundant with
  `projectionhost/versioned_journal_integration_test.go`.
- **`storage/auditstore/` package** — lying name. Renamed to "dispatch log" and
  kept in `storage/`.
- **Split `event/` module** — 27 importers, real cohesion. Explicitly decided in
  v4.
- **RollupSpec / RollupProjection** — premature abstraction. `sink.Increment` is
  the composable primitive. See [analytics rollup review](docs/feedback/reviewed/2026-07-23_analytics-rollup-support-review.md).
- **IncrementWhere on ProjectionSink** — footgun: silently updates multiple rows.
  Use `RelationalProjection` with explicit `Upsert`.
- **Redis adapter** — see ROADMAP Non-Goals (ValKey/NATS/Kafka preferred).
- **`idempotency.RefreshTTL(ctx, key, ttl)`** — dropped 2026-07-26 (YAGNI).
  Deferred across 6 sessions with no consumer; the design doc chose Option A
  (no-op on existing) _because_ Option B's sliding window is unsafe (unbounded
  TTL under retry storms).
- **cqrs-lint rule for the `idempotency.Store` Record contract** — dropped
  2026-07-26 (YAGNI). Only 3 Store impls exist (memory, kv, sql), all correct;
  the no-op-on-existing contract is already documented in the interface comment.
- **Centralized cross-module error-wrapping helper** — ADR-0069 decided:
  per-module helpers (`wrapInfraOrOK`, `wrapTransientOrOK`), capped at 3
  modules. A shared `storage/internal/errwrap` package would violate the
  multi-module isolation principle.
- **Move 3-way idempotency contract test to `integration/`** — dropped
  2026-07-26. Would add 3 new direct deps to integration/ and wouldn't fix the
  stated smell (property_test.go also imports sqlstore). Cross-implementation
  contract tests in the published kvstore module catch regressions for
  consumers.
- **Promote `wrapInfraOrOK` to storage/sql, signing, codec** — dropped
  2026-07-26. ADR-0069 explicitly caps at 3 modules. storage/sql has only ~6-8
  real candidates, signing/codec have effectively zero matching call sites.
- **Stack preset `stackpreset` builder** — dropped 2026-07-26. ~45 lines of
  trivial Go idiom; the real SQL consolidation lives in `stack/sqlopt`. A shared
  builder would create a cross-module dependency for a 5-line function.
- **Test infra helpers (catalogtest, storagetest, codectest)** — dropped
  2026-07-26. `idtest` (100+ call sites), `eventtest` (~30 helpers), `cattest`
  (20+ helpers) already cover all real needs. `codectest.NewCBORCodec()` would
  wrap a zero-value struct literal — an anti-pattern.
- **Turso sync 4-way deep look** — dropped 2026-07-26. Correctly accepted per
  ADR-0069; the 4 clone sites have unique error codes for traceability.
- **Triage auto-commit daemon commit messages** — dropped 2026-07-26. Prior
  decision stands (leave as-is). Garbled messages don't block release tagging
  (annotated tags override) and git log readability is acceptable.
- **`cqrs-bench` workload for metaengine** — dropped 2026-07-26.
  `metaengine.Store` is not a `*stack.Bundle`; the benchkit runner rejects it
  with `ErrIncompleteBundle`. Coverage already exists in
  `metaengine/planner_bench_test.go` (deliberately separated).
- **`filterDetectors` extraction in cqrs-lint** — dropped 2026-07-27
  (over-engineering). The "duplication" is 5 one-line `if !ctx.FeatureProfile.X
{ return nil, nil }` early-return guards, each checking a DIFFERENT profile
  field (HasServer, CommandFlow, HasSoftDelete). The real detector filtering
  (`FilterByCategory`/`FilterByRuleIDs`) is already extracted in `register.go`.
  A helper for the profile guards would obscure intent without reducing real
  complexity.

---

_Long-term direction (module extraction execution, NATS/Parquet implementation,
benchkit journey benchmarks, metaengine Phase 2 pushdown, goexperiment.jsonv2 /
Turso MVCC blockers) lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
