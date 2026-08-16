# Perf Pareto Plan — Safety-First Execution (2026-08-16 03:18)

> **Source:** `docs/status/2026-08-16_03-10_perf-audit-cache-line-sql-batching-deserialize-wins.md`
> (this session's audit + 3 shipped wins) cross-referenced with `TODO_LIST.md`.
> **Prime directive:** NO VERSCHLIMMBESSERUNG. The session shipped a 33x batch-size
> increase that is unverified on MySQL/MariaDB — closing that hole is worth more
> than any new optimization. Safety > speed > polish.

**Session context (what's already DONE and committed):**

- `cdc525fd5` — false-sharing pad on `workloadMeter` (−46..51% contended ops), dialect-aware SQL batch chunking (99→3276 rows for PG/MySQL/DuckDB; PG integration GREEN), metadata JSON round-trip eliminated on pebble+bbolt read (−46% ns, −53% allocs, measured), AGENTS.md gotchas, benchmarks + equivalence tests.
- `a298ea388` — api-stability golden for the two new exports.
- **Open risk:** 3276-row multi-VALUES can exceed MariaDB default `max_allowed_packet` (16MB) with realistic payloads. Unverified. No byte guard.

---

## Step 1 — Pareto Breakdown

### The 1% that delivers 51%: **Make the shipped batching change SAFE**

Three tiny tasks that neutralize the only consumer-breaking risk introduced this session.
One packet-overflow defect in a consumer's batch save undoes the entire session's value —
and the session's own headline metric (33x fewer round-trips) is worthless if the batch fails.

1. Byte-size guard for MySQL multi-VALUES chunking (estimate serialized args, cap conservatively).
2. `MaxParametersForDialect` unit test (SQLite→999 / PG→32767 / custom→999).
3. Real MySQL/MariaDB integration run (`#integration-mysql-vm`, nspawn needs root).

### The 4% that delivers 64%: **Make the wins REAL and PROVEN**

The wins currently exist only in workspace mode. Consumers get nothing until the
release chain closes, and the repo is red at HEAD from two pre-existing gate failures.

4. Run `#verify-fast` + `#check-coverage` (session claimed GREEN from components — close it honestly).
5. Fix pre-existing exhaustruct lint (`commandlifecycle/recorder.go:196`) + art-dupl baseline drift → repo GREEN at HEAD.
6. CHANGELOG entries (with measured before/after) + skill-reference docs for the 2 new exports.
7. **[BLOCKED on user auth]** Tag `event/v4 v4.6.1`, bump pebble/bbolt pins, run pre-tag checklist (`#vulncheck`). Until then standalone builds of storage/pebble+bbolt cannot resolve the new API.

### The 20% that delivers 80%: **Top no-regret IO wins (additive, opt-in only)**

Highest-impact remaining audit findings that change NO behavior for existing consumers:

8. Projectionhost live-checkpoint batching — `WithCheckpointInterval`/N-batch (opt-in; default unchanged).
9. bbolt `db.Batch` group commit on a dedicated batch-write path (opt-in; document callback-retry caveat).
10. PG `COPY FROM` bulk path for stream-log replay/backfill (internal path; INSERT stays default).
11. Pebble tuning knobs exposed (MemTableSize, Cache, WALBytesPerSync, Compression) — operator-deploys vision; defaults unchanged.

**GATED — durability tier→sync mapping** (`storage/pebble` Strict vs Normal per-write sync, metaengine engines NoSync paths): real perf win (fsync per append today) but a **behavior change for existing Normal-tier consumers** — minor-version risk. Awaits user decision (Status report §g Q3). Not in the 20%; it's the first item AFTER the gate opens.

### The other 80% (to reach 100%)

RAM micro-opts (payload adopt-variant, envelope sniff, ScanSlice pre-size, idempotency bound),
benchmark/observability infrastructure (benchstat baselines, perf ledger), cache-line maybes
(measure-then-pad worker counters, multiSeqCounter, SSEReplay.seq), hygiene
(ignored-error fix, TODO fold-in), plus the standing TODO_LIST sections (release/tagging
blockers, pin hygiene, metaengine Phases 6b/7, cqrs-lint, WithActor hardening, correctness
sweep, docs honesty) — all tracked in `TODO_LIST.md` as the living source; represented below
by their top items only.

---

## Step 2 — Comprehensive Plan (30–100 min per task, sorted by importance/impact/effort/customer-value)

| #   | Task                                                                                                                    | Tier | Impact                       | Effort | Est. | Customer value                            |
| --- | ----------------------------------------------------------------------------------------------------------------------- | ---- | ---------------------------- | ------ | ---- | ----------------------------------------- |
| T1  | MySQL byte-size chunk guard (estimate args bytes, cap ≤ ~8MB, unit tests w/ oversized payloads)                         | 1%   | CRITICAL (defect prevention) | S      | 60m  | Batch saves can't fail on default MariaDB |
| T2  | `MaxParametersForDialect` unit test + edge cases (nil/unknown dialect → 999)                                            | 1%   | HIGH                         | XS     | 15m  | Confidence in dispatch correctness        |
| T3  | Real MySQL/MariaDB integration run (`#integration-mysql-vm`); verify 3276-row batches + guarded sizes                   | 1%   | CRITICAL (verification)      | M      | 90m  | Proof the 33x win holds on MySQL          |
| T4  | DuckDB integration run (view BatchSet now chunks at 32767)                                                              | 4%   | MEDIUM                       | M      | 60m  | DuckDB consumers verified                 |
| T5  | `#verify-fast` end-to-end + `#check-coverage`; fix any drift                                                            | 4%   | HIGH (honest GREEN)          | M      | 90m  | Repo-level proof, not component claims    |
| T6  | Fix pre-existing exhaustruct (recorder.go:196) + art-dupl drift (asrecord trio, scenario/dsl)                           | 4%   | HIGH (repo GREEN at HEAD)    | S      | 45m  | Every future session's gates clean        |
| T7  | CHANGELOG entries with measured before/after for the 3 wins                                                             | 4%   | MEDIUM                       | XS     | 20m  | Consumers see honest numbers              |
| T8  | Skill references: `ReconstructEventWithMetadata`, `MaxParametersForDialect`, `WithActor` gap (TODO_LIST)                | 4%   | MEDIUM                       | S      | 45m  | Discoverable new API                      |
| T9  | **[BLOCKED:user]** Tag `event/v4 v4.6.1` + bump pebble/bbolt pins + pre-tag checklist + `#vulncheck`                    | 4%   | CRITICAL (release)           | M      | 90m  | Wins become importable                    |
| T10 | Projectionhost checkpoint batching: `WithCheckpointInterval` + N-batch flush (opt-in, tests incl. crash-restart window) | 20%  | HIGH (IO)                    | M      | 100m | Live-phase throughput on slow cp stores   |
| T11 | bbolt opt-in batch-write path via `db.Batch` (retry-caveat docs + tests)                                                | 20%  | MEDIUM-HIGH (IO)             | M      | 90m  | Cross-writer fsync coalescing             |
| T12 | PG `COPY FROM` bulk replay path (pgengine stream-log + eventstore backfill; bench vs INSERT)                            | 20%  | HIGH (IO)                    | L      | 100m | 2-10x bulk ingest                         |
| T13 | Pebble knobs: expose MemTableSize/Cache/WALBytesPerSync/Compression options (defaults unchanged)                        | 20%  | MEDIUM (IO)                  | M      | 90m  | Operator tuning (deploy-time vision)      |
| T14 | Perf ledger doc + fold perf backlog into TODO_LIST.md (this plan harvests)                                              | 20%  | MEDIUM (process)             | XS     | 20m  | Wins can't silently regress               |
| T15 | bbolt deserialize benchmark (convert extrapolated claim → measured)                                                     | 80%  | LOW-MED                      | S      | 30m  | Honest numbers                            |
| T16 | Payload adopt-variant on reconstruct path (skip 2nd clone; API design + equivalence tests)                              | 80%  | MEDIUM (RAM)                 | M      | 90m  | −1 alloc/event read                       |
| T17 | `idempotencyTracker` bound (TTL/ring) — unbounded sync.Map leak                                                         | 80%  | MEDIUM (RAM)                 | S      | 60m  | No slow leak in long-lived stores         |
| T18 | Envelope first-byte sniff in `UnwrapDecode` (go-codec external repo)                                                    | 80%  | MEDIUM (RAM)                 | M      | 90m  | Cheaper every blind-store read            |
| T19 | Measure-then-pad: worker counters @32P, multiSeqCounter, SSEReplay.seq (pad ONLY if contended)                          | 80%  | LOW-MED                      | S      | 60m  | Evidence-based, no cargo-cult padding     |
| T20 | Benchstat baselines committed for the 3 new benchmarks; contention at -cpu=16,32                                        | 80%  | MEDIUM (infra)               | S      | 45m  | Regression detection                      |
| T21 | SQLite durability tier when WAL off (preset.go:243 guard) — bugfix-class, no semantics debate                           | 80%  | MEDIUM (IO bug)              | S      | 45m  | Relaxed tier actually relaxed             |
| T22 | Fix ignored `MarshalMetadataJSON` error (system/adapter_event_serial.go:31)                                             | 80%  | MEDIUM (correctness)         | XS     | 15m  | No silent metadata loss                   |
| T23 | ScanSlice `RowCount()` pre-size + Custom-map size hints                                                                 | 80%  | LOW (RAM)                    | S      | 45m  | Fewer reallocs on big reads               |
| T24 | `#integration-mysql-nspawn` standing TODO (also covers stack/mysql live verify) — merge with T3 if root available       | 80%  | MEDIUM                       | M      | 60m  | Real-env MySQL signal in CI reach         |
| T25 | From TODO_LIST: pin-drift meta-test (🔥 catches both known skew classes)                                                | 80%  | HIGH (repo)                  | M      | 100m | Standalone-build rot caught at test time  |
| T26 | From TODO_LIST: seq-carrying journal reads (OFFSET skip → O(log n) seek)                                                | 80%  | HIGH (metaengine perf)       | M-L    | 100m | Large-journal replay perf                 |
| T27 | From TODO_LIST: engine capability conformance test (6 engines over-declare)                                             | 80%  | HIGH (honesty)               | M      | 100m | Planner stops lying about support         |

**Deferred by explicit decision (not forgotten):** durability tier→sync mapping [GATED on user Q3], PG COPY as default write path, unsafe string/bytes helpers [likely REJECT], struct reordering [measured: zero waste], sync.Pool [prior rejection stands], remaining TODO_LIST sections (cqrs-lint rules, WithActor coverage, docs honesty, Dgraph/DuckDB/Turso engine work, vector at scale) — all tracked in `TODO_LIST.md`.

---

## Step 3 — Fine Breakdown (max 12 min per task)

### Wave 1 — SAFETY (the 1%; do first, sequentially)

| #   | Task                                                                                                         | Est. | Depends |
| --- | ------------------------------------------------------------------------------------------------------------ | ---- | ------- |
| F1  | Read `metaengine/mysqlengine/dialect.go` + MariaDB packet docs; pick guard strategy (est-bytes cap)          | 10m  | —       |
| F2  | Write `estimateBatchBytes(events)` helper (sum payload+metadata+row overhead) in `storage/sql/helpers.go`    | 12m  | F1      |
| F3  | Split `maxPerBatch` computation: dialect param cap AND byte cap (cap loop: shrink batch when estimate > 8MB) | 12m  | F2      |
| F4  | Unit tests: oversized payloads force smaller chunks; boundary at cap; SQLite unaffected                      | 12m  | F3      |
| F5  | `MaxParametersForDialect` table test (SQLite/PG/MySQL/DuckDB/unknown/nil-safe)                               | 10m  | —       |
| F6  | Launch `nix run .#integration-mysql-vm` (background, ~131s); while waiting do F7-F9                          | 5m   | F4      |
| F7  | Add MySQL batch-size regression test to integration suite (2000 events × 8KB payload)                        | 12m  | F4      |
| F8  | Grep pgengine `calibration_bench_test.go` chunk constants; unify with `MaxParametersForDialect` if trivial   | 12m  | —       |
| F9  | Record MySQL results in status-report follow-up notes                                                        | 6m   | F6      |
| F10 | DuckDB integration run (`#test-all-backends` or targeted)                                                    | 12m  | F4      |

### Wave 2 — PROOF & GREEN (the 4%)

| #   | Task                                                                                                 | Est. | Depends |
| --- | ---------------------------------------------------------------------------------------------------- | ---- | ------- |
| F11 | Run `nix run .#verify-fast` (background ~15m); babysit log, no concurrent heavy jobs                 | 12m  | F10     |
| F12 | Run `#check-coverage`; if drift: add tests to uncovered new code                                     | 12m  | F11     |
| F13 | Fix exhaustruct: complete `event.Metadata{}` literal in recorder.go (explicit empty fields)          | 10m  | —       |
| F14 | Verify lint gate GREEN after F13                                                                     | 6m   | F13     |
| F15 | art-dupl: inspect asrecord trio — decide fix vs `//art-dupl:accept` (TODO_LIST already leans accept) | 12m  | —       |
| F16 | Re-pin or annotate dupl baseline; `#check-duplication` GREEN                                         | 10m  | F15     |
| F17 | CHANGELOG: false-sharing entry w/ ns numbers                                                         | 8m   | —       |
| F18 | CHANGELOG: dialect-aware batching entry w/ 33x + guard note                                          | 8m   | F4      |
| F19 | CHANGELOG: deserialize round-trip entry w/ ns/B/allocs                                               | 8m   | —       |
| F20 | modules.md: `ReconstructEventWithMetadata` row                                                       | 8m   | —       |
| F21 | modules.md/recipes.md: `MaxParametersForDialect` + batch-size note                                   | 10m  | F4      |
| F22 | core.md: `WithActor` documentation (TODO_LIST gap)                                                   | 12m  | —       |
| F23 | doc-check run on references                                                                          | 6m   | F20-F22 |
| F24 | **[BLOCKED:user]** Pre-tag checklist: `#vulncheck`, `#check-arch`, GOWORK=off tests on event         | 12m  | F11     |
| F25 | **[BLOCKED:user]** Tag `event/v4 v4.6.1` (script `scripts/tag-release.sh`, annotated)                | 10m  | F24     |
| F26 | **[BLOCKED:user]** Bump storage/pebble + storage/bbolt go.mod to event v4.6.1; workspace test        | 12m  | F25     |
| F27 | **[BLOCKED:user]** Re-run `#vulncheck` (standalone builds resolve new API)                           | 10m  | F26     |

### Wave 3 — TOP IO WINS (the 20%; each independently shippable)

| #   | Task                                                                                                                 | Est. | Depends |
| --- | -------------------------------------------------------------------------------------------------------------------- | ---- | ------- |
| F28 | projectionhost: add `WithCheckpointInterval(d)` + `WithCheckpointEvery(n)` options (default: every event, unchanged) | 12m  | —       |
| F29 | Live handler: track dirty-since-flush; flush on interval/N/shutdown; preserve last-event semantics                   | 12m  | F28     |
| F30 | Tests: interval flush cadence, crash-window ≤N reprocess (at-least-once), shutdown flush                             | 12m  | F29     |
| F31 | Docs: recipe note on checkpoint tuning (recipes.md §readmodels)                                                      | 8m   | F30     |
| F32 | bbolt: `BatchWrite` API sketch; read bbolt docs on fn-retry semantics                                                | 10m  | —       |
| F33 | Implement opt-in `WithBatchCommit()` on bbolt stores; single-writer path unchanged                                   | 12m  | F32     |
| F34 | Tests: concurrent writers via Batch produce identical journal; retry-idempotence of fn                               | 12m  | F33     |
| F35 | pgengine: pgx COPY spike behind interface (stream_log backfill path only)                                            | 12m  | —       |
| F36 | Bench: COPY vs multi-VALUES INSERT on 10K/100K rows; keep winner, gate behind option                                 | 12m  | F35     |
| F37 | pebble: expose `WithMemTableSize/WithCacheSize/WithWALBytesPerSync/WithCompression` on DefaultOptions chain          | 12m  | —       |
| F38 | Wire options through stack/pebble preset; defaults byte-identical test                                               | 12m  | F37     |
| F39 | Perf ledger: `docs/BENCHMARKS.md` — table path→benchmark→baseline→last measured                                      | 10m  | —       |
| F40 | Harvest: fold this plan's waves into TODO_LIST.md sections (done with this plan)                                     | 12m  | —       |

### Wave 4 — THE OTHER 80% (highest-value remainder)

| #   | Task                                                                                                    | Est. | Depends |
| --- | ------------------------------------------------------------------------------------------------------- | ---- | ------- |
| F41 | bbolt deserialize benchmark (mirror pebble bench)                                                       | 10m  | —       |
| F42 | Payload adopt-variant design: `reconstructEventAdopt` unexported first; equivalence + race tests        | 12m  | —       |
| F43 | Wire adopt path into pebble+bbolt deserialize; bench delta                                              | 12m  | F42     |
| F44 | idempotencyTracker: TTL bound option (default unbounded for compat? decide + document)                  | 12m  | —       |
| F45 | Implement tracker eviction; test memory bound under 1M IDs                                              | 12m  | F44     |
| F46 | go-codec: `UnwrapDecode` first-byte sniff (0xC0-0xDF/0xA0-0xBF major types vs `{`) — external repo PR   | 12m  | —       |
| F47 | Contention benches @-cpu=16,32 for workloadMeter; commit benchstat baselines                            | 12m  | —       |
| F48 | Measure worker-counter false sharing @32P; pad only if delta >10%; record decision either way           | 12m  | —       |
| F49 | Measure multiSeqCounter + SSEReplay.seq; same measure-then-pad protocol                                 | 12m  | —       |
| F50 | SQLite durability tier outside WAL guard (preset.go:243) + test that Relaxed≠FULL sync                  | 12m  | —       |
| F51 | Fix `system/adapter_event_serial.go:31` ignored error (wrap + propagate or explicit discard w/ comment) | 10m  | —       |
| F52 | ScanSlice RowCount pre-size; Custom map size hints in WithCustom/EnsureCustom                           | 12m  | —       |
| F53 | TODO_LIST item: pin-drift meta-test (compare required vs latest tags; fail on staleness)                | 12m  | —       |
| F54 | TODO_LIST item: fix `system/integration` DuckDB standalone (replace directive or driver guard)          | 12m  | —       |
| F55 | TODO_LIST item: `DecorateJournal` for VersionedSeekableJournal (ADR-0126 completion)                    | 12m  | —       |
| F56 | TODO_LIST item: singleflight leader-ctx capture fix (decider/load.go)                                   | 12m  | —       |
| F57 | TODO_LIST item: turso DSN leak in errors (register.go:69)                                               | 10m  | —       |
| F58 | TODO_LIST item: sqliteengine/tursoengine Close() leak                                                   | 12m  | —       |
| F59 | TODO_LIST item: seq-carrying journal reads — design `StreamLogEntry{Seq,Value}` shape                   | 12m  | —       |
| F60 | TODO_LIST item: engine capability conformance test skeleton (table of declared vs implemented)          | 12m  | —       |

**Standing TODO_LIST sections not itemized above** (tracked in `TODO_LIST.md`, unchanged):
Release/Tagging (7 items, mostly BLOCKED on user), Pin & Standalone-Build Hygiene (6),
Metaengine Phase 6b remainder + Phase 7 (22), cqrs-lint (4), WithActor Hardening (3),
Code Quality/Infrastructure (18), Correctness Defect Sweep remainder (4), Docs Honesty (4+).

---

## Step 4 — Execution Graph

```mermaid
flowchart TD
    subgraph W1["Wave 1 — SAFETY (the 1% → 51%)"]
        F1[F1 read dialect + pick guard] --> F2[F2 estimateBatchBytes]
        F2 --> F3[F3 dual cap: params AND bytes]
        F3 --> F4[F4 guard unit tests]
        F5[F5 dialect table test]
        F4 --> F6[F6 launch MySQL VM run]
        F6 -.while waiting.-> F7[F7 8KB-payload regression test] & F8[F8 unify chunk constants] & F9[F9 record results]
        F4 --> F10[F10 DuckDB run]
    end

    subgraph W2["Wave 2 — PROOF & GREEN (the 4% → 64%)"]
        F10 --> F11[F11 verify-fast]
        F11 --> F12[F12 check-coverage]
        F13[F13 exhaustruct fix] --> F14[F14 lint GREEN]
        F15[F15 asrecord decision] --> F16[F16 dupl GREEN]
        F17[F17-19 CHANGELOG x3] 
        F20[F20-22 skill references] --> F23[F23 doc-check]
        F12 --> F24{"F24-27 [BLOCKED:user]<br/>tag event v4.6.1"}
    end

    subgraph W3["Wave 3 — TOP IO WINS (the 20% → 80%)"]
        F28[F28-31 checkpoint batching opt-in]
        F32[F32-34 bbolt Batch opt-in]
        F35[F35-36 PG COPY spike + bench]
        F37[F37-38 pebble knobs]
        F39[F39 perf ledger]
    end

    subgraph W4["Wave 4 — THE OTHER 80%"]
        F41[F41-43 adopt-variant]
        F44[F44-45 idempotency bound]
        F46[F46 envelope sniff, external]
        F47[F47-49 measure-then-pad]
        F50[F50 SQLite durability outside WAL]
        F51[F51 ignored-error fix]
        F52[F52 ScanSlice pre-size]
        F53[F53-60 TODO_LIST top items]
    end

    GATE{"DURABILITY GATE<br/>(user Q3): tier→sync mapping<br/>= behavior change?"}
    F11 --> W3
    F23 --> W3
    W3 --> GATE
    GATE -->|approved| DUR[tier→sync mapping<br/>storage/pebble + engines]
    GATE -->|rejected/deferred| W4
    W3 --> W4

    classDef safety fill:#fdd,stroke:#c00,color:#000
    classDef proof fill:#ffd,stroke:#aa0,color:#000
    classDef io fill:#dfd,stroke:#0a0,color:#000
    classDef rest fill:#ddf,stroke:#00a,color:#000
    classDef blocked fill:#eee,stroke:#888,color:#666
    class W1 safety
    class W2 proof
    class W3 io
    class W4 rest
    class F24 blocked
```

**Execution order:** W1 → W2 → W3 (items parallelizable within wave) → W4. The durability
gate is a USER decision (Status report §g Q3), not a task. MySQL byte-guard (W1) blocks
nothing else but must land before any tag (F25) — never ship the unguarded 33x.

---

## Standing decision recommendations (autonomous defaults, overridable)

1. **MySQL guard:** byte-estimate cap ~8MB (50% of default 16MB `max_allowed_packet`) — safe default, no operator action needed. (Status Q1)
2. **Docs timing:** update skill references NOW in separate hunks (files carry prior-session edits; hunks don't collide). (Status Q2)
3. **Durability:** GATED — do not change storage/pebble semantics without explicit user approval. (Status Q3)
