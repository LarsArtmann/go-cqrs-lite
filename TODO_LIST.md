# TODO List

**Updated:** 2026-07-11 (session: v4 release execution — all blockers resolved, ready to tag)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md). Raw ideas live in [ROADMAP.md § Raw Ideas](ROADMAP.md#raw-ideas-no-design-yet).

## Legend

- `[ ]` = Open
- `[x]` = Done (moved to [CHANGELOG.md](CHANGELOG.md))
- `[v4]` = Breaking change, part of v4 cut
- `[BLOCKED]` = Blocked on upstream dependency

---

## v4 Release — EXECUTION COMPLETE

> **Status:** All code changes done, all tests pass, all docs updated. Ready to `git tag v4.0.0` and push.
>
> **Decisions locked in:**
>
> - BackfillHandler → `*SSEBroker`: **done**
> - Envelope magic string change: **dropped** (collision risk is already near-zero via `"$"` JSON key)
> - License swap + git history scrub: **after v4**
> - Parquet/DuckDB modules: **deferred to v4.1**
> - Storage/ split: **deferred to v4.1**
> - v3 git tags: **backfilled** (v3.0.0 through v3.7.1 now tagged)

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
- [x] **Storage/ split execution** — Deferred to v4.1 (NOT bundled with path migration to avoid verschlimmbessern).

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
- [x] **Update FEATURES.md for v4** — Envelope wrapping, codec flip, health checks, BackfillHandler change added.

### Remaining Release Step

- [ ] **`git tag v4.0.0` + push** — All code, tests, and docs are ready. Awaiting user approval.

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

### v4.1 — Storage/ Split

> Proposal at `docs/planning/2026-07-09_STORAGE-SPLIT-PROPOSAL.md`. Deferred from v4 to avoid
> doubling import-path churn. 3 packages: `eventstore/`, `readmodel/`, `sql/` (existing).

- [ ] **Extract `storage/eventstore/`** — SQLEventStore, SQLSnapshotStore, SQLCheckpointStore + migrations
- [ ] **Extract `storage/readmodel/`** — SQLKVStore, SQLViewStore, RelationalProjection, RelationalStore
- [ ] **Dispatch log stays in `storage/`** — SQLCommandStore, SQLQueryStore alongside SQLBackend facade

### Public Release Readiness

> **Release strategy decided 2026-07-11:** Public release (license swap + history scrub) happens
> **AFTER v4 cut**, not before.

- [ ] **Test `WithShutdownDependency` through real `sqlite.New()`** — Deferred from v4. Current struct-literal
      tests cover the topological sort logic, but a test through the real constructor path would add confidence.
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
