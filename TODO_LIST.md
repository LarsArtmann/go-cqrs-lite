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
- `⚠️` = Partially done — needs completion

---

## Verify Gate

> `nix run .#verify` is GREEN (build + vet + test + race + lint + API stability
>
> - doc-check). All 58 modules pass. Lint is clean (0 issues). File-size gate
>   is GREEN.

- [ ] 🔥 **Fix stale "5-family" references** introduced by go-error-family
      v0.10.0 upgrade. Three living docs still say "5-family" instead of
      "6-family" (Orchestration was added):
      `docs/error-taxonomy.md` (also says "v0.5.1" — 7 versions stale),
      `README.md:125`, `FEATURES.md:108`. This is a **split-brain** introduced
      in the 2026-07-26 execution session.
- [ ] **Wire `testing.Short()` into `#verify`** — benchkit soak tests now skip
      in short mode (35s → 0.05s), but `nix run .#verify` does not pass
      `-short`. Add a `-short` variant or a separate `#verify-fast` app.

---

## Release

> The CHANGELOG `[Unreleased]` section has 260+ lines across 12 subsections.
> go-error-family was upgraded v0.9.0 → v0.10.0 across all 50 modules (added
> Orchestration family). `metaengine/projectionadapter/v4.0.0` is tagged
> locally but NOT pushed.

- [BLOCKED] ⭐ **Cut v4.2.0 release** — flush `[Unreleased]` CHANGELOG, tag all
  58 modules, push tags. Requires user approval for push.
- [BLOCKED] 🔥 **Push `metaengine/projectionadapter/v4.0.0` tag** — exists
  locally, invisible to consumers. `git push origin metaengine/projectionadapter/v4.0.0`.
- [ ] **Regenerate api-stability golden** after v4.2.0 — new exports added
      (idempotency property tests, metaengine gap tests, `queryMessageCol`
      helper).
- [ ] **Add CHANGELOG entry for go-error-family v0.10.0 upgrade** — record the
      Orchestration family addition and the 3 exhaustive-switch fixes.

---

## Documentation Health

- [ ] **Update `docs/error-taxonomy.md`** — change "v0.5.1" → "v0.10.0", add
      Orchestration to the 5-row table, update all code examples.
- [ ] **Add CI check for taxonomy-count consistency** — grep for "5-family" /
      "5 Error Families" in living docs. Prevents future split-brain when
      go-error-family adds families.
- [ ] **Annotate remaining ~4 historical files** — stale openings without
      Resolution sections: analytics-rollup-review, NEXT-LEVEL-EXECUTION-STATUS,
      meta-engine-design, benchkit-implementation-status.
- [ ] **Hand-edit 2 HTML dashboards** — PARETO-EXECUTION-STATUS.html,
      cqrs-ecosystem-audit.html have stale hero sections.

---

## Module Health & Tooling

- [ ] ⚠️ **Complete idempotency property tests for kv + sql** — 4 rapid-based
      property tests were written but only tested against MemoryStore. The plan
      called for all 3 implementations (memory, kv, sql). Run the property
      tests against `kvstore.New()` and `sqlstore.NewSQLiteStore()`.
- [ ] **Move 3-way idempotency contract test to `integration/`** — currently in
      `idempotency/kvstore` (pulls sqlstore+sqlite as test deps).
- [ ] **Cursor round-trip test for non-numeric keys** — string/time keys on
      SQLite engine. Cross-engine meta-test gap.
- [ ] **Promote `wrapInfraOrOK` to storage/sql, signing, codec** — 20+ call
      sites in storage/sql alone. Per ADR-0069 per-module pattern.
- [ ] **spannedRead helper in pebble** — 4+ clone groups remain.
- [ ] **filterDetectors extraction in cqrs-lint** — shared by multiple rules.
- [ ] **Stack preset stackpreset builder** — parallel boilerplate across
      presets.
- [ ] **Test infra helpers** — eventtest.NewTestStreamID, catalogtest,
      storagetest, codectest.
- [ ] **art-dupl CI gate** — golden file + fail-on-new-groups.
- [ ] **Audit accepted clone groups** — verify 72 groups genuinely acceptable.
- [ ] **`--semantic -t 3` art-dupl run** — deeper duplication surface.
- [ ] **Write TestTagContentMatchesChangelog meta-test** — guard against
      tag/CHANGELOG drift.
- [ ] **Turso sync 4-way deep look** — accepted clone, may benefit from
      extraction.

---

## Daemon / CI

- [ ] **Recurring lint-sweep** — the auto-commit daemon occasionally commits
      unformatted code (gci/gofumpt drift), turning `#lint` red. Either gate
      daemon commits behind `nix fmt` or run a scheduled sweep.
- [ ] **Triage auto-commit daemon commit messages** — prior decision was "leave
      as-is"; revisit if garbled messages block `git log` readability or release
      tagging.
- [ ] **Parallel verify** — run independent module tests concurrently to cut
      verify time from ~4min to ~1min.
- [ ] **Investigate dependabot alert** `security/dependabot/10` — `gh api`
      returned no results (auth issue).

---

## Metaengine

- [ ] **Soak test metaengine SQLite** — multi-hour load test for the SQLite
      engine under concurrent writes.
- [ ] **cqrs-bench workload for metaengine** — end-to-end Apply → ExecuteTyped
      benchmark profile.

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

---

_Long-term direction (module extraction execution, NATS/Parquet implementation,
benchkit journey benchmarks, metaengine Phase 2 pushdown, goexperiment.jsonv2 /
Turso MVCC blockers) lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
