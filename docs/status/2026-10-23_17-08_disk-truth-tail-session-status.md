# Disk-Truth Tail Session Status — 2026-10-23 ~13:15–17:05 CEST

> Session resuming the 2026-10-17 disk-truth Pareto plan (v2). Predecessor status:
> `docs/status/2026-10-17_23-18_disk-truth-plan-execution-wave-I-delivery-status.md`.
> Successor plan: `docs/planning/2026-10-23_17-05_SUPERB-disk-truth-tail-delivery-pareto-plan-v3.md`.
> Parallel sessions were ACTIVE all day (catalog/architecture ADR-0160 work, census-fleet runs,
> a benchkit race fix, the taskmanager repair); their content rides [Unreleased] alongside ours.

## a) FULLY DONE

1. **A4/G2 closed** — integration battery green. The 3 mysqlengine "invalid connection"
   failures were QEMU slirp resets of fresh dial bursts (documented infra limit). Authoritative
   verdict via ephemeral userspace MariaDB 11.4.12 (port 13306): stack/mysql,
   idempotency/sqlstore, metaengine/mysqlengine (rerun-proof ×2), scheduling/sqlstore,
   queue/mysql, claiming, metaengine — all rc=0, zero FAIL lines. Matrix recorded in CHANGELOG (B15).
2. **Real pre-existing adttest bug found & fixed** — `AssertTxIsolationFromForeignContext`:
   fixed collection name → rerun false-positive "dirty read"; failure path leaked the tx
   goroutine → engine Close() in cleanup deadlocked → package burned 600s. Fix: run-unique
   collection + guarded defer close + error-path `inside` close. Verified 3× standalone + 2×
   full suite (pre-fix: deterministic red on run 2); sqlite/duckdb twins green.
3. **CI-red set fixed** (7 failing jobs on master): go.work restored to tracking (daemon had
   deleted it 10-16; the parallel restore branch `hermes/restore-go-work` never merged — disk
   file was identical, staged + absorbed); shfmt drift (2 scripts); check-private-deps (go-etag
   + httputil audited PUBLIC, added to AUDITED_PUBLIC); erraudit + doc-assertions verified
   green locally (stale CI failures); Examples Test fixed via (4).
4. **getting-started wave-completion miss fixed** — `reader.Get(counterID.String())` looked up
   the display form against the bare-keyed projection → `found=false` forever. Now `.Get()`;
   full examples suite green (`nix run .#test-examples`).
5. **A3 consumers + StreamKey completion wave SHIPPED** — the go-appkit B10 store pin
   (`TestScenarioPilot_StoreRowsCarryBareStreamIDs`: scans every column of every table of a
   real SQLite store for `StreamMarker:`) went RED and caught a **7th write path** the
   2026-10-17 wave missed: `meta_stream_log.stream_id` keyed by `id.StreamRef.StreamKey()`
   embedding the display form. Fix: bare `StreamKey()` (v4 + core/v5 mirror), system adapters
   with STICKY stream-key resolution (legacy rows stay under their key — no split histories,
   no journal rewrite), snapshot adapter bare-write + dual-read + dual-delete. Released
   **id/v4.7.3, system/v4.15.2, systemscenario/v4.2.1** — proxy-served, pin-sweep 51 modules,
   G1 scratch-consumer green. go-appkit re-bumped: both modules green, pin green with `-race`
   against published tags. cqrs-htmx: id bumped, full `nix run .#test` battery ALL GREEN.
6. **A5 done** — `inferColumnType` + word lists deleted; `BuildLayoutPlan` all-TEXT;
   `sqlTypeOf` → `(string, error)` advisory + slog.Warn; new `BuildLayoutPlanForType`;
   BOTH auto-layout paths (RegisterQuery + Plan rule pipeline) now type from the result type
   (fixed a latent DuckDB keyset VARCHAR/BIGINT binder error). Test repinned, api golden
   regen (6270 exports). Core + sqlite/duckdb/turso green in workspace mode.
7. **A6 done** — raw-bytes WAL pins (`TestWALPin_{Event,Command}EnvelopeBareAndRaw`) reading
   RAW rows: bare keys, plain base64 payload (no double-encode), base64-of-JSON metadata,
   RFC3339Nano UTC. Command twin caught `received_at` persisted LOCAL (+02:00) → normalized
   UTC. Red-on-pre-fix proven via worktree at system/v4.15.1 (both pins red at the bare-key
   row read — the leak IS the failure).
8. **A7 done** — goal-shaped-app story pin (`view.ID == doneID.Get()`) green; goal.db grepped
   clean.
9. **A8 done (codec half)** — keycodec v2: `Escape`/`Unescape`, full V2 builder family
   ("2"-suffixed tags; v1 ranges intact for dual-read), adversarial rapid laws whose
   generators DRAW control bytes (plain `rapid.String` never emits NUL — vacuous-law trap),
   collision-pair kill, v1/v2 range isolation. All green.
10. **A11 done (sqliteengine typed WAL)** — 8 typed columns, guarded ALTER + json_extract
    backfill + 2 partial UNIQUE indexes, typed inserts, UNIQUE violations →
    `ErrVersionConflict`, `StreamLoadByEventID` prefers typed column, idempotent in-place
    migration. **4/4 acceptance pins green** (duplicate-event conflict, (stream,version) slot
    conflict, typed-column fill, legacy-DB migration). Found+fixed a real double-decode bug
    (json already base64-decodes `[]byte` fields; re-decoding corrupted payloads). Module
    suite green. CHANGELOG row cut.
11. Disk incident handled: `/mnt/buildcache` 100% full → trashed golangci-lint (28G) + tmp
    (4G) → 53G freed. CHANGELOG rows cut for everything; TODO_LIST "Disk-truth plan residue"
    section added.

## b) PARTIALLY DONE

- **A19** — the CODE shipped (received_at UTC + occurred_at nanos column); the RULE DOC remains.
- **A17** — suite green via parallel session's repair; deriver.go coverage verification remains.
- **Wave hygiene** — all known CI-red classes fixed locally; `#verify` never ran (load-guard
  refused at load≈295; wait-loop timed out after 1h) and the next CI run hasn't been confirmed
  green yet.

## c) NOT STARTED

A9 (keycodec v2 engine adoption — precisely rowed in TODO_LIST §Disk-truth residue), A12
(pg+mysql typed WAL), A13 (duckdb typed WAL), A14 (adttest WAL conformance ×4), A15
(layout-stamp default-ON), A16 (docs truth pack), A18 (timers→gcdc), A20 (wire tags+golden),
A21 (recipes doc), A22 (cursor/sweep audit), A23 (legacy fixture + benchkit countKey),
A24 (HARVEST + closeout).

## d) TOTALLY FUCKED UP (session misses, all recovered)

1. Session started believing it was 2026-10-17 (summary six days stale) — parallel sessions
   had moved the tree substantially; cost triage time until `git log`/CI dates revealed it.
2. ~30 min lost to /tmp volatility (MariaDB server + proof worktree wiped mid-debug) — the
   gotcha (line 15) documents it explicitly and wasn't re-read before building.
3. First MySQL battery piped `go test | tail` — exit codes swallowed; reran with rc capture.
4. `rg -r` (replace flag) used by accident 3× — mangled displayed output (file lists correct).
5. One CHANGELOG edit clobbered the neighboring WithExact row (restored immediately).
6. A11's first fixture set hand-written base64 — invalid strings silently disabled typed
   extraction and burned a diagnosis cycle (lesson rowed: wire fixtures are generated through
   the producing serializer, never hand-written).

## e) IMPROVE

- W-DONE acceptance needs the consumer store-pin BEFORE the wave closes, not after (it caught
  the 7th write path 6 days post-tag and forced 3 extra tags).
- Read gotchas BEFORE environment setup; never pipe away exit codes on gates; generate wire
  fixtures via real serializers.
- Closeout must not assume `#verify` succeeds first try — schedule quiet windows deliberately.

## f) NEXT (see plan v3 Table A/B for the full 22+67 breakdown)

C01–C04 release chain (metaengine tag wave + consumers + verify/CI) → C05–C10 typed-WAL
parity + stamp default + wave 2 → C11–C17 docs truth + A9 keycodec adoption + taskmanager
coverage + timers + cursor audit → C18–C22 tail + HARVEST.

## g) OWNER QUESTIONS (do not block on)

1. **Deployed-DB census:** which production databases carry prefixed stream keys/rows?
   Determines permanent sticky-key fallback vs. re-key migration + v1-range drop (G3).
2. **taskmanager ownership:** is the parallel session done, or do we verify deriver coverage (A17)?
3. **A9 scheduling:** confirm keycodec-v2 engine adoption as its own migration wave (recommended)
   vs. pulling it ahead of the 20% tail.
