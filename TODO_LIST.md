# TODO List

**Updated:** 2026-07-11 (session: v4 release planning, storage split revision, Parquet/DuckDB research)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md). Raw ideas live in [ROADMAP.md § Raw Ideas](ROADMAP.md#raw-ideas-no-design-yet).

## Legend

- `[ ]` = Open
- `[x]` = Done (moved to [CHANGELOG.md](CHANGELOG.md))
- `[v4]` = Breaking change, part of v4 cut
- `[BLOCKED]` = Blocked on upstream dependency

---

## v4 Release

> **Trigger criteria MET** (2026-07-11): 3 breaking changes implemented in code — deprecated alias removal, blind store codec default flip (JSON→CBOR), and `event.DefaultCodec` flip. Ready to cut after blockers below are resolved.
>
> **Decisions locked in (this session):**
>
> - BackfillHandler → `*SSEBroker`: **approved for v4**
> - License swap + git history scrub: **after v4**
> - Parquet/DuckDB modules: **deferred to v4.1** (keeps v4 focused on cleanup)
> - Tag missing v3 releases (v3.3.0–v3.8.0): **yes, backfill before v4**

### v4 Breaking Changes (implemented, await tag)

- [x] **Remove deprecated APIs** — 8 event/+schema/ aliases deleted; `event.WithNewCodec` and
      `event.WithReplay` removed (replaced by `WithCodec` and `WithProcessingMode`); `query.Handler`
      deprecation notice removed (it is the load-bearing dispatch core, not deprecated —
      `TypedHandler` is the recommended ergonomic layer on top).
- [x] **Blind store codec default flip** — `kv.NewTypedStore`, `snapshot.NewTypedStore`,
      `command.TypedStore`, `query.TypedStore` all default to `CBORCodec` instead of `JSONCodec`.
      Safe via ADR-0044 envelope wrapping + ADR-0050 JSON fallback (old data auto-detected).
- [x] **`event.DefaultCodec` flip** — Changed from `JSONCodec` to `CBORCodec` in `event/codec.go`.
      Events are self-describing (`evt.Encoding()` stamped per-event), so `DecodePayloadAuto`
      handles mixed streams transparently.
- [v4] **BackfillHandler taking `*SSEBroker`** — Cleaner architecture; backfill can access broker's
  payload transform directly instead of requiring a separate `BackfillHandlerWithTransform`
  variant. **Approved for v4.** Current: `BackfillHandler(journal event.SeekableJournal)` →
  v4: `BackfillHandler(broker *SSEBroker)`.
- [v4] **Event/ god module decomposition** — Explicitly decided: DO NOT SPLIT (27 importers, cohesion is real).
- [v4] **Storage/ split execution** — Proposal at `docs/planning/2026-07-09_STORAGE-SPLIT-PROPOSAL.md`.
  Updated to **3 packages** (eventstore/, readmodel/, sql/) — dispatch log (SQLCommandStore,
  SQLQueryStore) stays in `storage/` alongside the backend facade. Awaits final approval.

### v4 Release Blockers (must fix before tagging)

- [ ] **Strengthen envelope magic string** — Currently `"cqrs"` (4 chars, collision risk with real
      data). Change to `"cqrs-envelope-v1"` in `codec/envelope.go:10`.
- [ ] **Envelope backward-compat integration test** — Write raw JSON data (pre-envelope format),
      read through `UnwrapDecode` with new CBOR default, verify round-trip. This is the exact
      migration path every consumer walks — if the fallback breaks, data is lost silently.
- [ ] **`[v4.0.0]` CHANGELOG section** — Formal release entry with breaking changes, migration steps,
      and link to `docs/migration/MIGRATION-GUIDE.md`. Currently only `[Unreleased]` exists.
- [ ] **ADR for codec default flip** — No ADR documents the decision to flip all codec defaults to
      CBOR. This is the most impactful v4 change — it needs its own ADR.
- [ ] **Backfill missing v3 git tags** — CHANGELOG documents v3.0.0 through v3.7.1 but only `v3.1.0`
      is tagged. Tag v3.3.0–v3.8.0 so consumers have reference points before the v4 jump.
- [ ] **`/v3` → `/v4` module path migration** — All 49 `go.mod` files need major version suffix
      updated. `api-stability` and `doc-check` tool module lists need updating. Largest mechanical
      effort in the v4 cut.

### v4 Release High-Value (should fix before tagging)

- [ ] **`HealthCheck` on `OwnedDBHandle`** — Currently only `*SQLEventStore` implements it. Moving
      to `OwnedDBHandle` (`storage/sql/base.go`) makes all SQL stores inherit it automatically.
- [ ] **Test `WithShutdownDependency` through real `sqlite.New()`** — Current tests use struct
      literals, not the actual constructor path consumers use.
- [ ] **Update FEATURES.md for v4** — Missing: envelope wrapping, codec default flip, health checks,
      shutdown ordering, alias removal.

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

- **Unify VersionedStore + VersionedSeekableJournal** (item 20) — Different interfaces (Store:
  Load/Save per aggregate, SeekableJournal: ReadFrom position-based). One type can't cleanly
  implement both. YAGNI.
- **VersionedJournal (ReadAll only)** (items 22, 49) — No consumer needs `ReadAll` with upcasters.
  SeekableJournal is the projectionhost interface. YAGNI.
- **Expose `SSEBroker.PayloadTransform()` accessor** (item 24) — No demand. Consumers configure the
  transform at construction.
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

_All completed items have been moved to [CHANGELOG.md](CHANGELOG.md) under `[Unreleased]`._
