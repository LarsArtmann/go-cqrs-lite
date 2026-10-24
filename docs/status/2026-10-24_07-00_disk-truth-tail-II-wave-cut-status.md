# Disk-Truth Tail Session II — Wave Cut Status — 2026-10-24 ~05:30–07:00 CEST

> Resuming session for the 2026-10-23 tail plan (v3). Predecessor:
> `docs/status/2026-10-23_17-08_disk-truth-tail-session-status.md`;
> plan: `docs/planning/2026-10-23_17-05_SUPERB-disk-truth-tail-delivery-pareto-plan-v3.md`.
> Parallel sessions were active throughout (metaengine codec-boundary repair —
> their content rode this wave; a cqrs-htmx pin-alignment session).

## a) FULLY DONE (all receipts in TODO_LIST/CHANGELOG)

1. **A10 re-rowed + IMPLEMENTED** — cqrs-lint **C044 `branded-id-display-string`**
   (warn-first, display-position whitelist, real type info via typedfixture,
   3-good/3-bad pins + severity pin). Proven on real code immediately: the
   taskmanager example lint grew 3 findings — two TRUE wire leaks
   (`TaskBlockedByPayload.DependencyID`/`TaskUnblockedPayload.DependencyID` carry
   display forms into event payloads) — golden updated. Rules 213→214.
2. **A12a pgengine typed WAL** — typed columns on BOTH write paths (multi-VALUES
   + pgx COPY), JSONB metadata, partial unique indexes, SQLSTATE-23505 mapping,
   `ADD COLUMN IF NOT EXISTS` migration, jsonb identity backfill
   (`pg_input_is_valid`, PG16 floor). 4/4 pins vs ephemeral PG. Contract
   decision mid-flight: **identity-only backfill everywhere** (payload/metadata
   NULL on legacy rows — a malformed foreign row must never abort a migration).
3. **A12b mysqlengine typed WAL** — `ALGORITHM=INSTANT`-pinned ALTERs (the
   server REFUSES a rebuild — no-recreate is enforced by construction, not
   convention; empirically CREATE_TIME updates on ALTER so it is NOT a
   no-recreate signal), information_schema probes (MySQL 8 lacks ADD COLUMN IF
   NOT EXISTS), VARCHAR(512) event_id sized to the 3072-byte utf8mb4 index
   limit. 4/4 pins rerun-safe vs userspace MariaDB 11.4 (relearned the
   run-unique-collection lesson ×2: first rerun's FIRST append conflicted).
4. **A13 duckdbengine typed WAL** — unique ART indexes verified live (file AND
   :memory:, NULLs-distinct = partial-index semantics) so the planned
   dedup-table fallback was NOT needed; `try_cast` guards the backfill.
   4/4 pins + full suite (171s, incl. soak). One-handle-per-file rule (DuckDB
   file locks).
5. **A14 adttest WAL conformance** — `AssertWAL{DuplicateAppendConflicts,
   StreamVersionConflict}` wired into all four SQL engine suites; green ×4.
6. **A15 layout-stamp default-ON** — `RecordLayoutStamps`/`LayoutStampDiffs`
   were exported-but-UNWIRED; now stamped at both apply paths
   (RegisterQuery + Plan rule pipeline), warn-first
   (`WithLayoutStampsDisabled` opt-out). Honest boundary discovered: column-SET
   conflicts hard-error at apply BEFORE the stamp (stronger guarantee); the
   stamp's lane is SILENT drift — pinned on the type-drift class
   (Priority TEXT→BIGINT, the name-typing-kill shape). Tests deliberately
   sequential (global slog capture races under t.Parallel).
7. **A16/C11 docs truth pack** — sweep-complete + consumer-store-pin lesson,
   cursor-pairing invariant (footguns); raw-bytes-pin convention + tx-iso
   rerun-safety + scratch-DB pattern (testing); four-layer compat checklist
   (module-management); buildcache disk-full runbook + /tmp re-hit
   (tooling-build); CHANGELOG wording swap. doc-check (1349 refs) + md-go green.
8. **A17** deriver coverage verified (3 legs); **A19** time-discipline rule doc;
   **A20** Causation/TombstoneMark wire tags + golden byte pins (found:
   ParseCommandID canonicalizes ULIDs — fixtures must use Crockford-canonical
   literals); **A21** recipes §2.5b legacy payload-ID strip (compile-verified
   via catalog); **A22** cursor audit — NO suspect sites, receipt rowed;
   **A23** sticky-key legacy-row pin (the 2026-10-23 wave's missing test) +
   benchkit's FIVE display-form storage keys fixed to `.Get()`;
   **A18** timers → gcdc envelope (decode-old-wrap-new, mixed-format pin).
9. **C02 RELEASE CHAIN — 14 TAGS LIVE**: metaengine/v4.20.0 → 11-module batch
   (sqliteengine/v4.6.0, pebbleengine/v4.4.4, projectionadapter/v4.6.2,
   storage/turso/v4.3.5, catalog/v4.10.0, cqrs-lint/v4.18.0, stack/v4.5.1,
   otel/v4.5.3, prometheus/v4.3.4, schema/v4.7.1, event/v4.15.2) →
   system/v4.15.4. All proxy-served, smoke-all green. **The v4.15.3 lesson:**
   tag-time tidy raced the proxy (resolved OLD siblings even though the deps
   were pushed first — the proxy hadn't cached them at tidy time); v4.15.4
   re-pinned via explicit `go get`, G1 scratch-consumer probe green on the
   full graph.
10. **C03 (go-appkit half)** — both modules bumped to system/v4.15.4 +
    metaengine/v4.20.0, suites GREEN (one load-contention flake at load≈36,
    passed on rerun), bump committed (their daemon absorbed).
11. **Wave hygiene between sessions**: 35-module go.sum re-tidy (dev-shell
    1.27.3 toolchain — the tidy meta-test was failing 35 modules), api golden
    6270→6301 (parallel sessions' exports + C044), V007 drift-table entry for
    the codec session's EncodeStreamValue deprecation, typedfixture budget 1→2.

## b) IN FLIGHT AT SESSION END

- **C04 verify chain**: `scripts/verify-chain.sh` launched (wait-loop +
  coverage + `#verify`) under load≈21-36; NOT yet concluded.
- **C03 cqrs-htmx half**: a parallel session is concurrently aligning pins
  (metaengine/schema/catalog align commits observed on their master); do NOT
  race them — verify their battery result instead.

## c) NOT STARTED (rowed, next session)

- **A9 keycodec v2 engine adoption** — the ONE remaining disk-truth item
  (pebble reference → badger/bbolt twins → storage/pebble+bbolt); TODO_LIST
  carries the full per-file map + the migration-pin patterns to copy.

## d) FOREIGN DEFECTS SURFACED (not touched, rowed)

- **SQL-engine planned-path suite RED at HEAD** (pgengine explain-proof +
  mysqlengine backfill/pushdown ×2) — pre-existing, verified at `d0e86a117`
  worktree; owned by the parallel planned-tables session. TODO_LIST row.
- One codemod artifact landed in MY in-flight test file mid-session
  (`stampResult(e)` — non-compiling); superseded by rewrite, noted as the
  edit-race class.

## e) OPEN OWNER QUESTIONS (2 of 3 remain)

1. Deployed-DB census (sticky-key permanence vs re-key + v1-range drop, G3).
2. A9 scheduling confirmation (recommended: own migration wave).

(taskmanager ownership RESOLVED — repair landed, coverage verified.)

## f) NEXT

1. Conclude/verify the verify-chain result + CI matrix on the post-wave master
   (C04); re-check cqrs-htmx battery (their session's alignment).
2. A9 per the TODO_LIST row (copy the scratch-DB/run-unique/serializer-fixture
   pin patterns from this wave).
3. The two owner questions above.
