# Status Report — T18 Snapshot Wire Tags (v5) Session

**Date:** 2026-09-06 08:41 CEST
**Session scope:** Execute TODO_LIST §v5 item "Honest snapshot wire tags at v5 (T18 audit)" — rename `aggregateId`/`aggregateType` wire tags per-backend with dual-read fallbacks + SQL column rename with migration. All work auto-committed by the daemon (commits through `f946a16dd`).

---

## a) FULLY DONE

**Wire-tag rename with dual-read fallbacks**
1. **Research pass** — located the T18/P10 prescriptions (2026-08-22 core-data-model review, C9 note in `docs/planning/v5-deprecation-sweep.md` §4, `docs/V5-MIGRATION-GUIDE.md` §3) and inventoried every snapshot serialization site across all backends (snapshot, pebble, bbolt, SQL, memory, metaengine, turso).
2. **`snapshot/v4`** — `Snapshot` tags renamed to `stream_id`/`stream_type`; new `snapshot/wire.go` with `UnmarshalJSON` + `UnmarshalCBOR` implementing the decode-only legacy fallback (writers emit new keys only, readers accept both). 6 wire tests added; goldens regenerated to the new shape.
3. **Key discovery (load-bearing)** — fxamacker/cbor v2.9 falls back to the **json tag** as the CBOR map key when no cbor key exists. The tag rename therefore moves CBOR bytes too; the CBOR fallback is mandatory, not cosmetic. Pinned by `TestWire_CBORCarriesNewKeys`; recorded as a footgun in AGENTS.md.
4. **pebble** — `serializableSnapshot` tags renamed; legacy JSON row **and** legacy CBOR row load tests prove pre-v5 rows keep working (identity rebuilt from the Pebble key, version/state/created_at keys unchanged).
5. **bbolt** — verified via git history it never carried `aggregate_*` snapshot tags; correctly excluded from the rename.
6. **SQL column rename** — all four dialect schemas (`storage/sql/migrations/{postgres,sqlite,duckdb,mysql}.sql`), the four `Dialect.SnapshotSchema()` methods, the eventstore snapshot INSERT/SELECT/DELETE queries, `storage/sql.DeleteByStream`, and the turso advisor example query. DDL assertion test + 3 golden files updated.
7. **Migration** — new exported `storage.MigrateSnapshotColumnsToStream(ctx, db, dialect)`: idempotent, per-dialect column probe (SQLite `PRAGMA table_info`; `information_schema` elsewhere; MySQL scoped via `DATABASE()`; dialect-correct placeholders), metadata-only `ALTER TABLE ... RENAME` (data moves with the column — the "backfill" is free), explicit `storage.snapshot_column_mixed` corruption error for half-migrated states. Wired into **all four** `InitSchema` helpers, so existing databases upgrade on first boot with zero operator steps.
8. **Tests for the migration** — SQLite unit tests (legacy table + seeded row → migrate → columns renamed → row survives → store loads it; idempotent re-run; fresh-schema and missing-table no-ops) and `TestPostgresSnapshotColumnMigration` against live PostgreSQL.

**Gates & docs**
9. `nix run .#integration-pg` — **full suite ✅ EXIT=0** (after go.sum repairs, see d/e).
10. api-stability golden regenerated (`storage.MigrateSnapshotColumnsToStream` captured).
11. Per-module golangci-lint: **0 issues** across snapshot, storage, storage/pebble, storage/sql (after fixes: recvcheck/tagliatelle exclusion for `snapshot/` mirroring the `storage/` precedent — `check-lint-config` green; S1016, errcheck, gofumpt, slices.Contains modernize all fixed).
12. `check-changelog-symbols.sh` green (150 citations honest; one citation fixed from `sqlpkg.DeleteByStream` → `storage/sql.DeleteByStream`).
13. `cmd/doc-check` green (956 references, 42 packages) after the AGENTS.md edit.
14. `check-duplication`: **zero clone groups from my files** (flagged clones all belong to a concurrent session's WIP).
15. CHANGELOG entry under [Unreleased] (honest symbol citations, explicit "not renamed here" scope note), TODO_LIST item checked off with DONE annotation, sweep-doc §4 executed-row struck with commit-wave citation, V5-MIGRATION-GUIDE §3 annotated as implemented (incl. the CBOR nuance), AGENTS.md footgun entry.
16. Regression suites green: storage/bbolt, stack/sqlite, stack/postgres, system (8 pkgs), integration (7 pkgs), storage/eventstore, storage/readmodel, storage/relational, storage/view; workspace-mode root build clean.

**Incidental repairs (blocking the named gate)**
17. go.sum rot repaired additively in **8 modules** (storage, benchkit, stack/postgres, metaengine/pgengine, projectionhost, idempotency/sqlstore, system, integration) — fresh-GOMODCACHE verification exposed missing `/go.mod` sums that the integration runs need.

---

## b) PARTIALLY DONE

1. **Sweep §4 "wire tags" umbrella** — my item covered the *snapshot* half only. Watermill metadata keys (`aggregate_id`/`aggregate_type`), events/commands table columns, benchkit `aggregates` JSON key, and transport/grpc proto fields still carry aggregate vocabulary. Deliberate scoping per the TODO item, but the §4 wave is not closable yet.
2. **Error-code batch rename** — untouched (separate v5 item; ~14 family codes listed in sweep §4).
3. **Migration verified live on 2 of 4 dialects** — SQLite (unit) and PostgreSQL (integration) verified end-to-end. MySQL/MariaDB and DuckDB migration paths are implemented + version-floored in docs but only code-reviewed, not run against live servers (`#integration-mysql-nspawn` / DuckDB CGo not exercised this session).
4. **check-duplication overall exit** — red, but every flagged clone belongs to the concurrent session's in-flight files (`cmd/cqrs-lint/pkg/suppression/fix.go`, `metaengine/*engine/planned_parity*.go`). Repo-level gate not green; my diff contributes nothing.
5. **go.sum hygiene root cause** — I repaired the 8 modules additively but did not investigate *why* the sums were missing repo-wide (suspect: `go mod tidy` run under a warm cache, or a pin-bump wave without a tidy sweep). Root cause open.
6. **Consumer-facing migration docs** — the fallback contract lives in CHANGELOG + migration guide + code comments; no copy-paste recipe yet in the skill `references/` for consumers decoding pre-v5 snapshots.

---

## c) NOT STARTED (adjacent, noticed this session)

1. Watermill metadata key rename + dual-read (v5 §4).
2. Error-code family rename batch (v5 §4, "batch at v5 with a changelog note").
3. events/commands table column rename (v5 §4 "any commands table variants") — undecided whether it lands in 5.0.0 or a later 5.x.
4. benchkit `aggregates` wire-key rename + re-golden.
5. v6 deletion of the fallback shims (scheduled; needs a ROADMAP marker).
6. ~~HARVEST of section (f) below into TODO_LIST/ROADMAP (per status-report skill: report first, harvest when session continues).~~ done (docs-health pass 2026-09-06 evening).
7. All other TODO_LIST §v5 items (transport/http+grpc module deletion, tombstone metadata API deletion, E1/E7/E8/E11/E13/E15, more extended-review follow-ups) — untouched.

---

## d) TOTALLY FUCKED UP (honest list)

1. **Fabricated-adjacent comment shipped briefly** — I asserted "canonical CBOR keys structs by Go field name" from memory in both `snapshot/wire.go` and `pebble/snapshot.go` and wrote a test asserting it. The test **falsified** the claim (fxamacker v2.9 uses json tags). The test doing its job is good; writing confident comments about external library behavior before verifying is exactly the verify-external-claims failure class.
2. **Speculative compatibility test** — I invented `TestWire_UnmarshalLegacySnakeKeys` for a snake-case spelling "some consumers adopted" that **never existed** in this library. Testing a contract that never shipped is fabrication with extra steps. Deleted.
3. **PG probe shipped with a raw `?` placeholder** — Postgres needs `$1`; the bug was invisible to the sqlite-only unit suite and failed the first live PG run. Should have used `Dialect.Placeholder(1)` from line one.
4. **First `wire.go` draft invented non-existent helpers** (`jsonTime`, `wireTime`, `eventVersion`, `legacyWire`) — caught at compile, but it was a sloppy draft for the single most important file of the task.
5. **Python-heredoc editing bit me twice** — one script silently didn't apply (escaping miss), one aborted on AssertionError mid-rewrite, and I later hit a contradictory mod-time guard because a foreign process touched a file between my read and edit. Fragile pattern; the edit tools exist for this.
6. **Three wasted integration-pg runs** — I repaired go.sum module-by-module as failures surfaced instead of sweeping all integration modules up front. Runs 1 and 2 were avoidable.
7. **Test hygiene noise** — stray invalid JSON line, non-comparable struct `==` (twice), unparenthesized composite literal in `if`, a `parallel := t.Parallel(); _ = parallel` artifact, and an artificial "compile guard" test added only to justify an import. All caught/fixed, but each cost a cycle.
8. **Pre-existing, not mine, but broken: go.sum rot in 8 modules** — the repo's integration gates fail from a fresh module cache. This will bite every future session and every tag-wave verify until the root cause is fixed (see e/f).

---

## e) WHAT WE SHOULD IMPROVE

1. **Verify external library claims before writing comments or tests around them** — a 30-second probe test beats a confident wrong comment (fxamacker incident). Encode this as the default: claim → probe → comment.
2. **Never hand-write SQL placeholders** — always via `Dialect.Placeholder(n)`, even in tests that "only target sqlite today".
3. **Use the edit tools, not python heredocs**, for multi-part edits — exact-match discipline beats string surgery, and the mod-time guard works with it instead of against it.
4. **When an env-dependent gate fails on missing sums, sweep all gated modules immediately** — don't pay per-run discovery tax.
5. **Test only contracts that actually existed** — speculative compatibility surface is fiction; the fallback list should be derived from git history of the wire format (which is exactly what I did for bbolt — do that everywhere).
6. **Add a fresh-cache go.sum verification to CI** (e.g. `GOFLAGS=-mod=readonly go build ./...` per module with a cold `GOMODCACHE`) so missing sums fail in CI once instead of in every future integration run.
7. **Central wire-key table** — one table in the sweep doc listing JSON/CBOR/SQL keys per backend + fallback status + v6 deletion deadlines; §4 prose is getting hard to audit.
8. **Consumer migration recipe** in the skill references (how to decode pre-v5 snapshot JSON/CBOR post-upgrade).
9. **Concurrent-session provenance**: one lint fix (`containsString` → `slices.Contains`) appeared in the tree without my edit landing; state was verified green, but I never reconciled who wrote it. When racing a parallel session on shared files, re-read immediately before AND after each edit, and diff the committed result.

---

## f) TOP 50 NEXT (brainstorm, sorted roughly by impact; most are ROADMAP fuel — harvest with routing rigor)

**Wire-format continuity (rest of sweep §4)**
1. Watermill metadata key rename + dual-read fallback (`aggregate_id`/`aggregate_type` → `stream_id`/`stream_type`).
2. Error-code family batch rename (~14 codes) with CHANGELOG migration note + "update dashboards" warning.
3. Generalize `MigrateSnapshotColumnsToStream` into a table/column-pair migration helper (prereq for #4/#5).
4. events table column rename + migration.
5. commands table column rename + migration.
6. benchkit `aggregates` output-key rename + golden refresh.
7. Confirm transport/grpc dies at wave C *before* anyone burns effort renaming its proto fields.
8. Execute `storage/relational` + `storage/view` deletion (ADR-0123) — removes the remaining `aggregate_*` SQL surfaces wholesale instead of renaming them.
9. bbolt/catch-all grep audit: prove no other aggregate-vocabulary wire keys remain post-waves.
10. ROADMAP marker for the v6 deletion of snapshot fallback shims (with the one-release-cycle timer).
11. Central wire-key table doc (JSON/CBOR/SQL × backend × fallback status) — rewrite sweep §4 as the table.
12. Consumer recipe in skill references: decoding pre-v5 snapshots post-upgrade.

**Testing / verification**
13. Live MySQL/MariaDB migration test for `MigrateSnapshotColumnsToStream` (`#integration-mysql-nspawn`).
14. Live DuckDB migration test (CGo env).
15. Mixed-state corruption test (both column sets present → `storage.snapshot_column_mixed`).
16. Mid-migration failure-path test (rename error → clean error code, no partial state claimed).
17. Concurrent-init idempotency test (two processes booting InitSchema on the same DB).
18. CI fresh-GOMODCACHE go.sum check (kills the 8-module rot class permanently).
19. Root-cause the 8 go.sum holes (tidy-with-warm-cache? pin wave without tidy?).
20. Re-run check-duplication after the concurrent cqrs-lint session lands (clear the foreign clones).
21. `-shuffle=on` for the new migration tests (sweep adoption wave is pending for pg/mysql apps anyway).
22. `-race -count=3` pass over the new wire/migration tests per the race-aware threshold discipline.
23. Property test: rapid-generated legacy JSON docs (arbitrary field subsets) decode without error through `Snapshot.UnmarshalJSON`.

**Docs**
24. Document the v5 snapshot wire contract + fallback in the skill references (snapshot section).
25. V5-MIGRATION-GUIDE: operator verification snippets (pragma/information_schema queries to confirm a DB migrated).
26. FAQ entry: "old snapshot JSON/CBOR not decoding → fallback contract, what's accepted".
27. Fix the guide's own historical key-name inconsistency (`aggregate_id` vs `aggregateType` in §3 prose) — my annotation notes it, the original line still lies.

**Code quality**
28. Make the pebble identity-from-key invariant a named, tested contract (it silently carries legacy-row compatibility).
29. Dialect-probe capability interface (unexported) if a third dialect-specific probe ever appears — replace the concrete type-switch.
30. Mark the legacy wire shadows with the v6 deadline convention so the deletion wave can grep for it.

**Repo hygiene**
31. Coordinate `scripts/pin-sweep.sh` uncommitted foreign changes with the parallel session.
32. Reconcile the concurrent-session `containsString` fix provenance (confirm committed state is intended).
33. Confirm the foreign `PlanStaleInlineSuppressions` golden entry (landed inside my api regen) is the owning session's intent.
34. Re-run `check-lint-config` self-heal after the next daemon reformat attempt (my session added a `.golangci.yml` touchpoint).
35. Auto-commit daemon vs `.golangci.yml`: the gci re-add loop (TODO daemon Q2) is still the root cause of formatter drift — close it.

**Remaining v5 TODO_LIST waves (order per execution rules)**
36. Delete `transport/http` + `transport/grpc` modules (final tags exist; drop from go.work/flake/api lists).
37. Delete deprecated tombstone metadata API (ADR-0114 completion; `listing` type-driven status is the bridge).
38. Wave B stragglers: `schema.VersionedStore`/`VersionedSeekableJournal`, `signing.Rejecting*`, `encryption.ErrInnerStoreNot*`, `metadata.CustomData`, `storage/sql.BuildWhereClause`.
39. E-items: E1 (event-envelope Encoding → `record.Encoding`), E7 (watermill/middleware RetryConfig), E8 (typed Message Kind).
40. E-items: E11 (AdapterCore.Encode error return), E13 (SQLTimerStore phantom param), E15 (middleware signature unification).
41. Extended-review follow-ups harvested 2026-09-06 (E3 bbolt …).
42. Sweep §6 consumer scans re-run at the cut (BuildWhereClause, transports, stack presets).
43. Wave-order discipline: A → B → C → wire tags → error codes, each its own commit family + gates.

**Release engineering**
44. `nix run .#verify` + `#vulncheck` + `#verify-ci` over all modules before any tag (today's go.sum holes would have failed a tag wave).
45. GOWORK=off build matrix over every swept module at tag time (per multi-module tag-wave mechanics).
46. Decide 5.0.0 cut sequence and which §4 renames ride it vs later 5.x.
47. cqrs-lint taskmanager golden version-set refresh (V006) in the same wave as any dependency-graph change.
48. Update CONTRIBUTING release notes: schema migration is now automatic inside InitSchema (operators no longer apply manual DDL for the snapshots rename).
49. One bench sanity run post-wire-change (no timing paths touched, but cheap insurance against denial).
50. Schedule the v6 wave: delete snapshot wire fallbacks + set the legacy-row support window for pebble.

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **SQL scope & timing:** should the events/commands table columns rename in **5.0.0** (same wave as snapshots, sweep doc hints yes: "snapshots table + any commands table variants"), or in a later 5.x? This decides whether I generalize the migration helper now and how much consumer SQL breakage we accept in one release.
2. **Backport expectation:** master now writes new wire keys while published v4 readers only read old keys (pre-v5 data stays readable; the reverse needs the upgrade). Is there any expectation of a v4.x patch release carrying the dual-read readers so old binaries can read new writes — or is the upgrade cut strictly one-way at 5.0.0?
3. ~~**Concurrent session ownership:** the other session's `cqrs-lint/pkg/suppression` WIP currently keeps `check-duplication` red, and `scripts/pin-sweep.sh` + `metaengine/duckdbengine/planned_parity_cgo_test.go` are foreign-edited. Are those expected to land soon (I leave their gates alone), or is that session done/abandoned (then I should reconcile and clean)?~~ resolved — the concurrent waves landed; master is pushed and in sync with origin (2026-09-06 evening; repo-gate GREEN not re-claimed). Residual clone-group cleanup is tracked in TODO_LIST.

---

*Report generated per status-report skill; format override: user explicitly requested `.md` (skill default is styled HTML — honored the user, flagging the divergence). WAITING FOR INSTRUCTIONS.*
