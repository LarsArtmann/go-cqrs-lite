# TODO List

**Updated:** 2026-07-26
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task is finished it is removed from
this list and recorded in CHANGELOG.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- `⭐` = Top 1% impact (do first)

---

## Verify Gate — get to GREEN end-to-end

> Lint is clean (0 issues). The file-size gate is RED on **one file**:
> `cmd/api-stability/main.go` (353 lines, 3 over the 350-line limit).
> The otel flakiness was fixed in the 2026-07-25 session via
> `WithoutGlobalRegistration()`. The full `nix run .#verify` has not been
> confirmed green end-to-end since the latest metaengine + dedup work.

- [ ] ⭐ **Split `cmd/api-stability/main.go`** (353 → two files under 350) so
      `nix run .#check-file-size` passes. This is the **only** file-size
      violation remaining.
- [ ] 🔥 **Run `nix run .#verify` end-to-end** and fix anything red.
      Known flaky: 5 benchkit timing tests (`TestRunSoak_TrendsPopulated`,
      `TestRunSoak_Memory`, `TestWriteSoakJSON_RoundTrip`,
      `TestSnapshotPhase_SQLite`, `TestRun_AnalyticalJournalScans`) pass in
      isolation but fail under full-suite `-race` load. Fix: add
      `testutil.RaceEnabled` thresholds (see AGENTS.md lint conventions).
- [ ] **Document `otel.WithoutGlobalRegistration()`** in AGENTS.md OTel section
      + Crush skill `references/core.md` — public API added during the otel
      flakiness fix, currently undocumented for consumers.

---

## Module Tagging

> 57 of 58 modules are tagged and pushed. `metaengine/v4.0.0`,
> `metaengine/v4.1.0`, `metaengine/v4.1.1`, `idempotency/sqlstore/v4.0.0`, and
> `benchkit/v4.1.0` are all pushed to origin. The 32 missing tags from the
> 2026-07-25 release fix are also pushed.

- [BLOCKED] **Tag `metaengine/projectionadapter/v4.0.0`** — its `go.mod` has a
  local `metaengine/v4 => ../` replace directive. Although `metaengine/v4.1.1`
  is now published, the replace must be removed and the version pinned before
  tagging. Run `./scripts/tag-release.sh metaengine/projectionadapter v4.0.0`.

---

## Release Tooling

- [ ] **Audit `scripts/tag-release.sh`** for other `pipefail` traps like the one
      fixed this session (grep `-P` no-match on non-cqrs replace directives
      aborted the whole release under `set -euo pipefail`). Consider `--dry-run`
      mode and single-module tagging (currently touches all 58 go.mod files).
- [ ] **Investigate `v4.0.4` tag-at-commit question** — `codec/v4.0.4`,
      `event/v4.0.4`, `watermill/v4.0.4` all point to `8285da41` ("strip
      replace directives"). A brutal self-review flagged this as potentially
      wrong (should be `dbddbed6`), but both commits share the same message.
      Verify the tagged tree content matches the intended release.

---

## Documentation Health

- [ ] **Add ADR-0069 to the index** — `docs/README.md` and `docs/adr/README.md`
      ADR tables stop at ADR-0068. ADR-0069 (error-wrapping helpers convention)
      exists at `docs/adr/0069-error-wrapping-helpers.md` but is missing from
      both index tables.

---

## Module Health & Tooling

- [ ] **Fix 5 benchkit timing tests** — add `testutil.RaceEnabled` thresholds
      so they pass under full-suite `-race`. See report
      `docs/status/2026-07-26_18-36_dedup-session-6-brutal-self-review.md`.
- [ ] **Property test for `idempotency.Store`** — generate random Record/Seen/
      CheckAndRecord sequences via `pgregory.net/rapid`, assert contract
      invariants hold across all 3 implementations (memory, kv, sql).
- [ ] **Move 3-way idempotency contract test to `integration/`** — currently in
      `idempotency/kvstore` (pulls sqlstore+sqlite as test deps). Move after
      the projectionadapter tag is sorted.
- [ ] **Fix `#vulncheck` nix app** — newer govulncheck requires explicit package
      patterns (`./...`), not stdin. The pipeline may be broken.
- [ ] **Real gocognit fix for `TestSinkUpsert`** — extract `assertMessageRow`
      helper to genuinely reduce complexity (currently band-aided with
      `//nolint:gocognit`).
- [ ] **Triage auto-commit daemon commit messages** — prior decision was "leave
      as-is"; revisit if garbled messages block `git log` readability or release
      tagging.
- [ ] **Recurring lint-sweep** — the auto-commit daemon occasionally commits
      unformatted code (gci/gofumpt drift), turning `#lint` red. Either gate
      daemon commits behind `nix fmt` or run a scheduled `nix fmt && nix run .#lint`.
- [ ] **Benchkit per-module build broken** — stale `storage/pebble/v4.0.3` tag
      references renamed `Snapshot` fields (`AggregateID`/`AggregateType`).
      Re-tag storage/pebble or update benchkit go.mod.
- [ ] **Dead Codec test code in benchkit** — `soak_test.go:283` Codec branch
      never executes (test Config never sets Codec). Replace with a dedicated
      `TestConfig_CodecRoundTrip`.

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
  (no-op on existing) *because* Option B's sliding window is unsafe (unbounded
  TTL under retry storms).
- **cqrs-lint rule for the `idempotency.Store` Record contract** — dropped
  2026-07-26 (YAGNI). Only 3 Store impls exist (memory, kv, sql), all correct;
  the no-op-on-existing contract is already documented in the interface comment.
- **Centralized cross-module error-wrapping helper** — ADR-0069 decided:
  per-module helpers (`wrapInfraOrOK`, `wrapTransientOrOK`), capped at 3
  modules. A shared `storage/internal/errwrap` package would violate the
  multi-module isolation principle.

---

_Long-term direction (module extraction execution, NATS/Parquet implementation,
benchkit journey benchmarks, metaengine Phase 2 pushdown, goexperiment.jsonv2 /
Turso MVCC blockers) lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
