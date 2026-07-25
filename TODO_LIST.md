# TODO List

**Updated:** 2026-07-25
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

## Verify Gate — confirm GREEN end-to-end

> The file-size gate is GREEN (all 11 oversized files split under 350 lines).
> The otel flakiness is fixed (`-race -count=10` clean). However the **full**
> `nix run .#verify` has not been run in one pass since the splits — lint,
> doc-check, and module-coverage sub-checks are unconfirmed on the 16 new files.

- [ ] ⭐ **Run `nix run .#verify` end-to-end** and fix anything red (lint nits
      on split files, doc-check symbol drift, module-coverage gaps).
- [ ] **gofmt all 16 new split files** in one pass (`gofmt -w`), confirm zero diff.
- [ ] **Document `otel.WithoutGlobalRegistration()`** in `AGENTS.md` OTel section + Crush skill `references/core.md` — new public API added during the otel
      flakiness fix, currently undocumented for consumers.

---

## Module Tagging

> 2 of 3 untagged modules tagged locally. `benchkit/v4.1.0` was pushed to origin.
> 56 of 58 modules now have tags (55 pushed + 1 local-only).

- [ ] 🔥 **Push `metaengine/v4.0.0` and `idempotency/sqlstore/v4.0.0`** to origin
      (tags exist locally, annotated, release-clean go.mod). Requires user
      push authorization.
- [BLOCKED] **Tag `metaengine/projectionadapter/v4.0.0`** — its `go.mod` has a
  local `metaengine/v4 => ../` replace; cannot resolve metaengine from the
  Go proxy until `metaengine/v4.0.0` is pushed. After the push above, run
  `./scripts/tag-release.sh metaengine/projectionadapter v4.0.0 "..."`.

---

## Release Tooling

- [ ] **Audit `scripts/tag-release.sh`** for other `pipefail` traps like the one
      fixed this session (grep `-P` no-match on non-cqrs replace directives
      aborted the whole release under `set -euo pipefail`). Consider `--dry-run`
      mode and single-module tagging (currently touches all 58 go.mod files).

---

## Documentation Health

- [ ] **Update `docs/README.md` ADR index** — the table stops at ADR-0035, then
      jumps to ADR-0046. ADRs 0036–0065 are missing from the index (files exist
      in `docs/adr/`).
- [ ] **Add the 3 newly-tagged modules** (`metaengine/v4`, `idempotency/sqlstore/v4`,
      `metaengine/projectionadapter/v4` once tagged) to `FEATURES.md` status table.

---

## Module Health & Tooling (from 2026-07-25 self-review sweep)

- ✅ **[RESOLVED] Broken published module graph** — 32 missing tags created locally
  at commit `8285da41` (17× v4.0.3, 3× v4.0.4, 13× v4.0.2, 1× v0.2.1). All 84
  require refs now resolve. **Push pending:** `git push origin --tags`. See
  `docs/release-fix-2026-07-25.md`.
- ✅ **Configure gopls with `goexperiment.jsonv2`** — Already configured in
  `~/.config/crush/crush.json` (`GOEXPERIMENT: jsonv2` in gopls env).
- [ ] **Recurring lint-sweep** — the auto-commit daemon occasionally commits
      unformatted code (gci/gofumpt drift), turning `#lint` red. Either gate
      daemon commits behind `nix fmt` or run a scheduled `nix fmt && nix run .#lint`.
- [ ] **`idempotency.RefreshTTL(ctx, key, ttl)`** — optional capability to
      extend a live key's dedup window (design note item 3). New API; track
      until a consumer needs sliding-window dedup.
- [ ] **cqrs-lint rule for the `idempotency.Store` Record contract** — flag
      custom `Store` implementations whose `Record` extends the TTL (violates
      the no-op-on-existing contract). New lint rule.
- [ ] **cqrs-bench profile for the metaengine SQLite engine** — no benchmark
      covers the SQLite planner path end-to-end. Add a named profile.
- ✅ **CI badge for the api-stability gate** — Already in README; `#check-api-stability`
  now runs inside `#verify` with `-race` and the `TestEveryGoModDirIsInModulesList`
  meta-test.
- [ ] **Triage auto-commit daemon commit messages** — prior decision was "leave
      as-is"; revisit if garbled messages block `git log` readability or release
      tagging.
- [ ] **Property test for `idempotency.Store`** — generate random Record/Seen/
      CheckAndRecord sequences via `pgregory.net/rapid`, assert contract invariants
      hold across all 3 implementations (memory, kv, sql).
- [ ] **Fix `#vulncheck` nix app** — newer govulncheck requires explicit package
      patterns (`./...`), not stdin. The pipeline is broken.
- [ ] **Move 3-way idempotency contract test to integration/** — currently in
      `idempotency/kvstore` (pulls sqlstore+sqlite as test deps). Move after
      pushing the missing tags.
- [ ] **Push the 32 missing module tags** — `git push origin --tags` after user
      approval. See `docs/release-fix-2026-07-25.md`.
- [ ] **Soak test for metaengine SQLite** — multi-hour load test.
- [ ] **cqrs-bench workload for metaengine** — end-to-end Apply → ExecuteTyped.

---

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

---

_Long-term direction (module extraction execution, NATS/Parquet implementation,
benchkit journey benchmarks, metaengine Phase 2 pushdown, goexperiment.jsonv2 /
Turso MVCC blockers) lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
