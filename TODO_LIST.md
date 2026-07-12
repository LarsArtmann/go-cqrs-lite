# TODO List

**Updated:** 2026-07-12 (session: documentation reconciliation — fixed feedback doc contradiction, getting-started guide, ADR index, module counts)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md). Raw ideas live in [ROADMAP.md § Raw Ideas](ROADMAP.md#raw-ideas-no-design-yet).

## Legend

- `[ ]` = Open
- `[x]` = Done (moved to [CHANGELOG.md](CHANGELOG.md))
- `[v4]` = Breaking change, part of v4 cut
- `[BLOCKED]` = Blocked on upstream dependency

---

## v4 Release — SHIPPED

> **Status:** v4.0.0 tagged and pushed. All per-module v4.0.0 tags exist. Storage/ split
> shipped with backward-compat aliases. View store gaps (IS NULL, RawWhere, ViewUpdater, BLOB)
> shipped as additive features.
>
> **Decisions locked in:**
>
> - BackfillHandler → `*SSEBroker`: **done**
> - Envelope magic string change: **dropped** (collision risk is already near-zero via `"$"` JSON key)
> - License swap + git history scrub: **after v4**
> - Parquet/DuckDB modules: **deferred to v4.1**
> - Storage/ split: **shipped in v4.0.0** (sub-packages with backward-compat aliases)
> - v3 git tags: **backfilled** (v3.0.0 through v3.7.1 now tagged)
> - View store gaps: 4 of 5 shipped (BLOB, IS NULL, RawWhere, ViewUpdater); composite keys rejected (use RelationalProjection)

### v4 Breaking Changes (ALL DONE)

- [x] **Remove deprecated APIs** — 8 event/+schema/ aliases deleted; `event.WithNewCodec` and
      `event.WithReplay` removed; `query.Handler` deprecation notice removed.
- [x] **Blind store codec default flip** — `kv.NewTypedStore`, `snapshot.NewTypedStore`,
      `command.TypedStore`, `query.TypedStore` all default to `CBORCodec`.
- [x] **`event.DefaultCodec` flip** — Changed from `JSONCodec` to `CBORCodec` in `event/codec.go`.
- [x] **BackfillHandler → `*SSEBroker`** — Signature changed from
      `BackfillHandler(journal event.SeekableJournal)` to `BackfillHandler(broker *SSEBroker)`.
      `BackfillHandlerWithTransform` removed (consolidated — transform configured on broker).
- [x] **Event/ god module decomposition** — Explicitly decided: DO NOT SPLIT (27 importers, cohesion is real).
- [x] **Storage/ split execution** — `storage/eventstore/` (SQLEventStore, SQLSnapshotStore,
      SQLCheckpointStore) and `storage/readmodel/` (SQLKVStore) extracted as sub-packages.
      Full backward compat via type aliases in `storage/`. All tests pass.

### v4 Release Blockers (ALL RESOLVED)

- [x] ~~**Strengthen envelope magic string**~~ — DROPPED. Collision risk already near-zero via `"$"` JSON key.
- [x] **Envelope backward-compat integration test** — `kv.TestTypedStore_Migration_*` tests verify
      old raw JSON data reads through new CBOR-default stores, and mixed old+new data coexists.
- [x] **`[v4.0.0]` CHANGELOG section** — Written with 4 breaking changes, migration steps, and link to guide.
- [x] **ADR for codec default flip** — ADR-0053 written (`docs/adr/0053-unified-codec-default-flip.md`).
- [x] **Backfill missing v3 git tags** — v3.0.0, v3.3.0–v3.7.1 tagged.
- [x] **`/v3` → `/v4` module path migration** — All 49 go.mod files, all imports, all docs, all scripts,
      all tools updated. Full workspace build + test + vet pass.

### v4 Release High-Value (ALL DONE)

- [x] **`HealthCheck` on `OwnedDBHandle`** — All SQL stores now inherit `HealthCheck(ctx)` via embedding.
      Removed redundant implementation from `*SQLEventStore`.
- [x] **Update FEATURES.md for v4** — Envelope wrapping, codec flip, health checks, BackfillHandler change,
      storage split sub-packages added.
- [x] **`WithShutdownDependency` integration tests** — Tests through real `stack.New()` constructor
      with close-order tracking. Proves option function, pointer identity, and Close() ordering all work
      through the real integration path.

### Release Status

- [x] **`git tag v4.0.0` + push** — Tagged. All per-module v4.0.0 tags exist.

---

## Post-v4

### v4.1 — Parquet Journal + DuckDB Materializations

> Design complete at `docs/research/2026-07-11_PARQUET_JOURNAL_DUCKDB_MATERIALIZATIONS.md`.
> Three independent phases, all additive (no breaking changes). Deferred from v4 to keep
> the major version cut focused on cleanup.

- [ ] **Phase 1: `storage/parquet`** — Parquet segment journal (`SeekableJournal`). Pure Go
      (`parquet-go/parquet-go`), no CGO. Segment-based append-only log with manifest index.
- [ ] **Phase 2: `storage/duckdb`** — DuckDB connector + `DuckDBDialect` (11 methods). Unlocks
      `SQLViewStore` + `RelationalProjection` for OLAP-grade materializations. Requires CGO.
- [ ] **Phase 3: `stack/duckdb`** — Preset wiring: DuckDB materializations + optional Parquet
      journal. The "lakehouse for events" pattern (DuckDB queries Parquet segments natively).

### v4.1 — Storage/ Split (DONE in v4.0.0)

> Originally planned for v4.1 but pulled into v4.0.0. `storage/eventstore/` and
> `storage/readmodel/` sub-packages created with full backward compat.

- [x] **Extract `storage/eventstore/`** — SQLEventStore, SQLSnapshotStore, SQLCheckpointStore
- [x] **Extract `storage/readmodel/`** — SQLKVStore
- [x] **Dispatch log stays in `storage/`** — SQLCommandStore, SQLQueryStore alongside SQLBackend facade
- [x] **Deprecated re-exports in `storage/`** — type aliases + constructor re-exports for backward compat

### Public Release Readiness

> **Release strategy decided 2026-07-11:** Public release (license swap + history scrub) happens
> **AFTER v4 cut**, not before.

- [ ] **License swap (PROPRIETARY → Apache-2.0)** — Hard blocker for public adoption. **Needs user
      approval (irreversible).** After v4.
- [ ] **Git history scrub for internal docs** — AGENTS.md, docs/planning/\* contain internal
      strategy. **Needs user approval (irreversible).** After v4.
- [ ] **Postgres CI coverage matrix** — Add CI Postgres service or label experimental.
- [ ] **README polish to "sales page" standard** — Per AGENTS.md rule.

---

## Transport (future expansion)

- [ ] **NATS/ValKey Stream adapter** — ADR-0025 accepted. Separate `transport/nats/` and
      `transport/redis/` modules.
- [ ] **Distributed event bus** — No multi-process backend for event distribution.

---

## Experimental / Go-stdlib-blocked

- [BLOCKED] **Remove `goexperiment.jsonv2` tag** — JSON v2 is fully adopted (~25 production files
  import `encoding/json/v2`). The build tag remains only because Go 1.26 hasn't graduated json/v2
  from experimental. Remove the tag when Go stabilizes it (expected Go 1.27+).
- [BLOCKED] **Turso MVCC concurrent-write support** — Blocked on upstream experimental MVCC.

---

## Rejected (with reasons)

- **Strengthen envelope magic string (`"cqrs"` → `"cqrs-envelope-v1"`)** — Dropped. The `"$"` JSON key
  provides 99% of collision avoidance. The value is belt-and-suspenders. Extra bytes per record for
  near-zero benefit.
- **Unify VersionedStore + VersionedSeekableJournal** (item 20) — Different interfaces (Store:
  Load/Save per aggregate, SeekableJournal: ReadFrom position-based). One type can't cleanly
  implement both. YAGNI.
- **VersionedJournal (ReadAll only)** (items 22, 49) — No consumer needs `ReadAll` with upcasters.
  SeekableJournal is the projectionhost interface. YAGNI.
- **Expose `SSEBroker.PayloadTransform()` accessor** (item 24) — Implemented for BackfillHandler
  (necessary), but no standalone demand from consumers.
- **`WithPayloadTransform` on SSEHandler** (item 27) — SSEHandler wraps the broker; adding transform
  there duplicates responsibility (SRP violation).
- **Auto-apply CQRS views by default** (items 35, 50) — Violates "library, not framework" principle.
  Consumers choose their histogram boundaries.
- **VersionedSeekableJournal implementing event.Store** (item 40) — Different scope (position-based
  vs aggregate-based reads). YAGNI.
- **Integration test in `integration/` module** (item 19) — Redundant with
  `projectionhost/versioned_journal_integration_test.go`.
- **`storage/auditstore/` package** — Lying name. "Audit" implies after-the-fact compliance review,
  but these stores serve replay/debugging/accountability. Renamed to "dispatch log" and kept in
  `storage/` — they're small CRUD wrappers (~1,100 lines) that belong with the backend facade.
  See updated `docs/planning/2026-07-09_STORAGE-SPLIT-PROPOSAL.md`.

---

_All completed items have been moved to [CHANGELOG.md](CHANGELOG.md) under `[4.0.0]`._
