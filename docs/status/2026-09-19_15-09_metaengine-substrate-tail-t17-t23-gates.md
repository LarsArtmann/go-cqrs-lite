# Status Report: Metaengine Universal Storage Substrate — tail execution (T17–T23) + gate repair

**Date:** 2026-09-19 15:09 CEST
**Session scope:** Resume the owner-stopped plan execution (T13–T16 were done,
this session worked the tail): finish the clone-gate annotations, T17 doc
rows, T18a benches, T22 taskmanager-on-queue, T23 semantic-diff memo, the
MySQL/PG live legs, the integration sweep, and the verify attempt.

**Plan:** [`docs/planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md`](../planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md)

---

## a) FULLY DONE (verified green this session)

### Clone gate (check-duplication) — GREEN in one round

- All 16 reported new clone groups annotated with `//art-dupl:accept`
  (28 directives across 27 files): queue dialect twins (cancel/enqueue/
  engine/open/claim/details/lifecycle), queue/conformance scenario
  boilerplate, metaengine dueclaim wiring 4-way, duckdb/pg DDL loops, and
  the claimkit Exec+RowsAffected idiom pair. Both sides of every mirror
  annotated; directive placement follows the verified geometry (directly
  above the region's first line / last doc-comment line).
- `nix run .#check-duplication`: **0 new clone groups** (baseline 60).
- All 9 touched modules build; queue/sqlite + sqliteengine smoke-tested.

### T17 remainder — docs rows

- recipes.md §2.38 was ALREADY shipped + catalog-classified (the 12:12
  report's "not written" was stale — verified by content + 3 catalog
  entries + doc-check green).
- ADDED: FEATURES "ADR-0142 Capability Matrix" section (engine × DueClaim/
  Dedup: native/degraded/refused + reasons) and the FAQ entry "Why can't my
  timers or queue claims live on dgraph (or the iroh Replicated wrapper)?"
  (RefusedADTs → Doctor → capability-audit universality rule → routing
  guidance).
- doc-check: exit 0, 1183 references valid (the 5 ⚠ are pre-existing
  union-resolved alias ambiguity notes). CHANGELOG symbols gate: 142
  citations honest (fixed a false positive — prose mentions of stdlib
  `packages.Load` de-backticked). error-taxonomy: 525 codes / 20 modules
  green.

### T18a — claimkit micro-benches vs direct-SQL (the abstraction-tax proof)

- NEW `metaengine/claimkit/bench_test.go`: ClaimDue steady-state (claimkit
  vs `claiming.ClaimStmt` composed by hand), the composite timer
  round-trip (insert → due+fact → epoch-guarded fire), dedup fresh-key and
  live-window shapes (vs hand-rolled upsert).
- **Measured (this machine): claimkit is FASTER than the direct path on
  every pair** — claim 63µs vs 82µs, dedup fresh 11µs vs 19µs, live-window
  hit 9.4µs, timer round-trip 104µs. No perf regression smuggled in by the
  abstraction.
- 4 claimkit benches added to `benchmark-regression.sh`'s explicit gate set.

### T22 — example/taskmanager on the engine-backed queue

- The deriver's auto-assign cascade is now a DURABLE job on `queue/sqlite`
  (`workqueue.go`: dedup-keyed enqueue, lease-fenced ClaimDue worker,
  backoff retries, FailPermanent dead-lettering for malformed IDs) instead
  of a fire-and-forget goroutine. Wired into Server lifecycle
  (Start/Stop), README row + architecture note added.
- Tests: `TestWorkQueue_AssignmentSurvivesRestart` (enqueue → close →
  reopen → still claimable + lease fencing) and
  `TestWorkQueue_AutoAssignEndToEnd` (create → deriver → queue → worker →
  task.assigned in the read model). FULL example suite green `-race`
  (including all pre-existing integration tests, now flowing through the
  queue).
- Live demo run: server on a file DB, log shows `task.create` → enqueue →
  worker claim → `task.assign` dispatched + succeeded. One first-boot
  transient (WAL-conversion race between the two pools on a shared file)
  is absorbed by the worker's retry and logged at Warn.
- go.mod: sibling replaces (queue, queue/sqlite, claiming, metaengine) —
  the established pre-tag-wave pattern; stripped when the family is tagged.

### T23 — go-taskqueue semantic-diff memo

- [`docs/research/2026-09-19_go-taskqueue-semantic-diff.md`](../research/2026-09-19_go-taskqueue-semantic-diff.md)
  — donor `internal/queue/queue.go` (read in full, 521 lines) vs our
  `Store[T]` + ADTs: core semantics 1:1; divergences are strengthenings
  (claim tokens vs owner strings, enqueue dep validation), genericity
  (typed payloads), or product surface correctly left to the consumer
  (RecordAnswer/questions, priority-score cache). **No silent drift;
  upstreaming subtyped, not forked.**

### Live DB legs

- **MySQL (local ephemeral MariaDB 11.4, `-race -count=2`):**
  mysqlengine GREEN, queue/mysql GREEN — including the 16-racer CAS
  exclusivity test that forever flakes under QEMU slirp (local server has
  no such instability).
- **PG (`PG_MODULES="metaengine/pgengine queue/postgres" nix run
  .#integration-pg`):** both GREEN on the ephemeral PG — pgengine's live
  FactSink leg (`FactsRideTheClaimTx`, `DeleteFactsAtomicBothDirections`)
  and queue/postgres full conformance.
- **`nix run .#test-integration` (EXIT 0):** ephemeral-PG sweep (storage ×6
  packages, stack/postgres, pgengine, projectionhost, scheduling/sqlstore,
  idempotency/sqlstore, benchkit) + testcontainers MySQL (stack/mysql 214s)
  — all green post-1.27-sweep.

## b) PARTIALLY DONE / BLOCKED

- **`nix run .#verify` EXIT 1 — lint phase only.** Everything before lint
  is GREEN: documentation assertions, module coverage, Build, Vet, Test
  (module `ok`s across the tree), api-stability (twice), doc-check. The
  lint phase reports **76 style findings across 22 modules** (wsl_v5 ×13,
  embeddedstructfieldcheck ×12, modernize ×8, mnd ×6, …) — accumulated
  ADR-0142-wave code that was daemon-committed without a lint run
  (`.golangci.yml` itself was rewritten 12× today; the concurrent session
  is ACTIVELY fixing this exact list — claiming/benchkit edits within
  minutes of this report, new path exclusions added today). I did NOT race
  them on those files (documented collision hazard). Notable: several
  findings are deliberate patterns needing judgment calls, not blind fixes
  (`gochecknoinits` on the contract-19 register.go `init()`s, `err113` on
  sentinel-style errors, `exhaustruct_v5` on partial EngineProfile
  literals).
- **T18b (load-sweep + benchmark baseline regen):** not run. Machine load
  has been 28–74 all day (two llama servers + foreign nix builds + the
  concurrent session); these are the only genuinely load-SENSITIVE
  deliverables left. The baseline also needs the new claimkit entries.
- **`#verify-ci`:** not run (heavy, and the tree is being edited).

## c) NOT STARTED (by design)

- T19–T21 (v5 fold / duplicate-stack deletion / release train): v5-gated
  per ADR-0142; TODO_LIST now carries them as an explicitly v5-gated item.
- Tag waves (claiming + queue family): owner-gated release action.

## d) TOTALLY FUCKED UP (found broken, now fixed)

1. **queue/mysql `SaveWatermark` built syntactically invalid SQL** —
   `IF(watermarks.seq < VALUES(seq), watermarks.seq)` is a two-argument IF
   with an unbalanced paren: EVERY watermark save failed (MariaDB Error
   1064). Invisible to fresh-DB runs that never hit the duplicate path?
   No — invisible to runs whose conformance leg predates the Watermarks M4
   subtest. Fixed as the MySQL/MariaDB translation of the sqlite/postgres
   upsert-WHERE guard: `seq = GREATEST(...)` + conditional `updated_at`.
   Green `-race -count=2` vs live MariaDB.
2. **mysql/pg engine reset tests assumed a pristine shared test DB** —
   ADR-0143 made journals SURVIVE resets, so fixed-name journal
   collections/keys accumulate across runs and break the survival length
   assertions on any long-lived server (my 2.5h-old MariaDB hit it on run
   #2). Fixed with run-unique journal collections (mysql: both tests,
   incl. the whole-collection `JournalReadAllWithSeq` counts) and keys (pg).
3. **`metaengine/memory_engine.go` grew 354 → 364** (ADR-0143's journal
   save/restore + doc comment) tripping the file-size ratchet. Fixed by
   extracting `newMemData` + `ResetEngine` verbatim to
   `metaengine/memory_reset.go` (319 + 48 lines; gate green; reset tests
   green).
4. **CHANGELOG symbols false positive**: prose `packages.Load` mentions
   read as pkg.Symbol citations — de-backticked.

## e) WHAT WE SHOULD IMPROVE

1. **The daemon-absorbed ADR-0142 wave never ran lint** — 76 findings
   surfaced only at my verify attempt. The daemon-gate-hook proposal
   (12:12 report e1) keeps earning itself.
2. **Two sessions, one module tree** bit again: I held off lint fixes
   because the other session was mid-list on the same files. The
   single-writer protocol proposal (12:12 report e2) stands.
3. Fresh-DB-only test legs hide shared-state bugs (SaveWatermark survived
   a "full conformance green" claim). Long-lived-server legs (like my
   MariaDB run) are worth keeping around for exactly this.
4. `nix run .#verify` ordering: lint runs before race — the race phase
   never got a chance to validate the ADR-0143 fix this run.

## f) NEXT (prioritized)

1. Land the lint cleanup (in progress by the concurrent session —
   claiming/benchkit/queue engines/stack/*; 76 → 0) and re-run
   `nix run .#verify` end-to-end including the race phase.
2. T18b: on a quiet window (load < ~10): `nix run .#load-sweep`, then
   `./scripts/benchmark-regression.sh --save benchmarks/benchmark-baseline.txt`
   (picks up the new claimkit entries + 1.27 toolchain) — then the gate.
3. `nix run .#verify-ci` (per-module GOWORK=off matrix).
4. Tag waves when the owner authorizes: claiming/v4.0.0 + queue family
   (strips the new example/taskmanager replaces too).

## g) DECISIONS MADE AUTONOMOUSLY (owner was unavailable)

- **g-1 (commits):** no explicit commits this session either — the daemon
  absorbed everything into `chore: auto-commit` history (owner never
  answered; committing without instruction is against my ground rules).
  CHANGELOG + this report remain the narrative record.
- **g-2 (clone gate):** annotated all mirror groups (contract 14's stated
  preference); no baseline re-pin. Result: green in one round.
- **g-3 (MariaDB):** kept alive until BOTH mysql legs re-ran green, then
  shut it down cleanly at 14:5x (`process gone` verified).
