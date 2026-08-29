# Status: Perf-Backlog Review — Durability Decision + False-Sharing Baselines (2026-08-17 14:23)

**Session scope:** User pasted the TODO_LIST "Performance — 2026-08-16 audit backlog" section (2 open
items) with the READ→UNDERSTAND→RESEARCH→REFLECT→execute-and-verify loop. This report covers only
this session. Session was paused mid-flight for this report; durability implementation has NOT started.

**Verdict:** Both backlog items fully researched and unblocked. The blocked durability item got its
user decision (**Option A — align all layers, minor version, with an explicit naming-quality
requirement**). The benchstat-baselines item is 70% done: all 3 bench suites re-run GREEN at protocol
settings and benchstat-verified in /tmp — but nothing is committed yet, and the durability code work
has not begun.

---

## a) FULLY DONE

1. **Context absorption** — Read the 2026-08-16 perf-audit status report (all 50 P0-P4 items + §g
   questions), TODO_LIST performance section, `docs/BENCHMARKS.md` (performance ledger),
   `docs/benchmarks/2026-08-16_false-sharing-contention.md` (evidence doc + protocol), and
   `scripts/benchmark-regression.sh` (the ONE gate + committed-baseline conventions).
   Note: the paste referenced `status/…` and `planning/…` paths that don't exist at repo root —
   actual homes are `docs/status/` and `docs/planning/` (harmless, resolved immediately).

2. **Item 1 research (durability tier→sync) — complete, decision-ready evidence:**
   - **Doc split brain found:** `stack/durability.go:42` documents Pebble Normal as "WAL enabled,
     default flush behaviour" (async WAL) while `stack/pebble/preset.go:38` says "Normal → same as
     Strict for Pebble (no change)". The two docs contradict each other today.
   - **Cross-backend consistency evidence:** SQLite already de-escalates at Normal
     (`synchronous=NORMAL`), Postgres at Normal (`synchronous_commit=off`). Pebble (fsync-per-write
     for Normal) is the outlier — every other backend honors the tier ladder.
   - **Bare `storage/pebble` state:** all 5 stores + KV adapter default `syncWrites: true`; opt-out
     `WithAsyncWrites`/`With*AsyncWrites` options already exist per store. Clean seam at
     `newBackend` (backend.go:74 already threads per-store options, e.g. `WithKVSyncWrites`).
   - **`stack/pebble` state:** preset maps ONLY Relaxed→`DisableWAL` (preset.go:184); Normal/Strict
     both get full sync.
   - **`stack/bbolt` state:** preset DEFAULTS to `DurabilityStrict` (preset.go:39) — inconsistent
     with pebble/sqlite presets which default Normal. Only knob is `NoSync` (Relaxed).
   - **Engines:** `pebbleengine` has 15 hardcoded `pebble.Sync` sites (engine.go, stream_log.go,
     vector.go); `bboltengine` uses `bolt.DefaultOptions` (engine.go:90), no NoSync path.

3. **User decision obtained (§g Q3 of the 2026-08-16 report):**
   > **Option A — "Align all layers (minor ver)": preset maps Strict→fsync/write, Normal→async WAL,
   > Relaxed→DisableWAL. Bare storage/pebble defaults stay sync. Doc contradiction fixed. CHANGELOG
   > Changed entry.**
   > Plus explicit instruction: **"Make sure our naming is right, good, and easily understandable!"**
   > → The TODO_LIST item is now UNBLOCKED with a decided direction and a naming-quality gate.

4. **benchstat made available** — `nixpkgs#benchstat` does not exist as a flake attr;
   `go install golang.org/x/perf/cmd/benchstat@latest` built to `/tmp/bin/benchstat` (works).
   ⚠️ Ephemeral location — reproducibility gap tracked below (f.16).

5. **All 3 false-sharing bench suites re-run GREEN** at evidence-doc protocol settings
   (`-tags goexperiment.jsonv2 -run '^$' -bench <Pattern> -cpu 16,32 -count=10`), workspace mode
   from repo root (run #2 — see d.2 for run #1). Outputs in `/tmp/fs-base-{sqliteengine,
   projectionhost,metaengine}.txt`, 40 benchmark lines each, and **benchstat reproduces the recorded
   2026-08-16 decisions**:

   | Suite                           | Baseline result (this run)                                          | vs 2026-08-16 evidence doc                                             |
   | ------------------------------- | ------------------------------------------------------------------- | ---------------------------------------------------------------------- |
   | MultiSeqCounter (sqliteengine)  | Unpadded 18.88n±3% @16 / 19.97n±11% @32; Padded 7.29n±2% / 7.15n±2% | ✅ Pad decision confirmed (2.6–2.8x, matches −61..65%)                 |
   | WorkerCounters (projectionhost) | Adjacent 190.4n±1% / 233.8n±7%; Padded 343.6n±3% / 393.8n±7%        | ✅ NO-PAD decision confirmed (padded ~1.7x slower for writer)          |
   | SSEReplaySeq (metaengine)       | Adjacent 81.0n±1% / 81.3n±5%; Padded 50.9n±56% / 41.0n±9%           | ⚠️ See c.1 — this run's padded cells came out FASTER with huge variance |

---

## b) PARTIALLY DONE

1. **Item 2 (benchstat baselines) — ~70%:** raw protocol-correct outputs exist and validate against
   the recorded decisions, but **nothing is committed**: no baseline files under `benchmarks/`, no
   benchstat summary in the evidence doc, TODO_LIST checkbox still open, BENCHMARKS.md not updated.
2. **Item 1 (durability mapping) — research 100%, implementation 0%:** every seam is located
   (a.2) and the decision is in, but no code, tests, docs, golden, or CHANGELOG entries written.
   Session paused here for this report.

---

## c) NOT STARTED

1. **SSEReplaySeq anomaly triage** — this run's baseline shows padded winning at BOTH cpu counts
   (50.9n/41.0n vs adjacent 81.0n/81.3n), which superficially contradicts the recorded NO-PAD
   decision. The recorded rationale was "contradictory deltas (+12% @16, −30% @32) with high
   variance"; this run has ±56% variance on padded@16 and adjacent drifted 54.66n→80.98n @16 vs the
   recorded run — so both runs agree only that the signal is unstable. Needs a tie-breaker run
   (ideally on a quieter machine, see e.4) before touching the layout — the decision stands until
   then, per measure-then-pad protocol.
2. **Durability Option-A implementation** (all of it): stack/pebble tier mapping, storage/pebble
   backend-level option threading + naming review, stack/durability.go + preset.go doc alignment,
   stack/bbolt consistency question (g.1), engine opt-in options (pebbleengine/bboltengine), tests,
   api-stability golden regen, CHANGELOG, skill references, `#verify`.
3. **Baseline artifact plumbing** — file placement/naming under `benchmarks/`, evidence-doc +
   ledger updates, optional CI-gate wiring (g.2).

---

## d) TOTALLY FUCKED UP (or nearly)

Nothing destroyed, no false GREEN claims, no reverts. Honest failures:

1. **Question tool — 3 malformed attempts** before the durability question landed (choice objects
   with empty labels, then a missing `description`). Pure schema sloppiness; wasted 3 round trips
   and put noise in front of the user mid-decision.
2. **First bench run in the wrong mode — had to throw away 2 GREEN suites.** Ran per-module with
   `GOWORK=off` (as AGENTS.md prescribes for _module_ work); `projectionhost` failed with
   `module ../event requires go >= 1.26.6 (running go 1.26.5)` despite `GOTOOLCHAIN=auto` being
   exported (per-module mode doesn't get rescued the way workspace mode does), and mixing
   GOWORK=off numbers with workspace numbers would make the baselines non-comparable — so I
   re-ran ALL THREE from the workspace root. Cost: one full suite cycle. The evidence doc's
   protocol command implies root execution; I should have read it that way the first time.
3. **Benches run at load average 1.3–2.1, 45 user sessions** — not the "quiet machine" the evidence
   doc demands. Results are tight (mostly ±1–7%) so the baselines look usable, but SSEReplaySeq's
   ±56% cell (c.1) may partly be ambient load. Purist protocol says re-run exclusive.
4. **benchstat lives in /tmp/bin** — works now, gone after reboot. The repo's devShell has no
   benchstat and nixpkgs has no package attr; "formal benchstat baselines" without a reproducible
   benchstat is half-formal. Not resolved (f.16).

---

## e) WHAT WE SHOULD IMPROVE (process, from this session)

1. **Match the evidence doc's execution mode exactly** — protocol benchmarks run from the workspace
   root; per-module GOWORK=off is for module tests, not protocol baselines. (AGENTS.md gotcha entry
   already hints at this; I still stepped on it.)
2. **Learn the question tool schema before asking** — labels on every choice, description on every
   question. One shot, not four.
3. **Check `uptime` BEFORE protocol benches**, not after. A 10-second check would have flagged the
   ambient load before 3 suites burned.
4. **Triage outlier baselines immediately** — when a fresh baseline contradicts a recorded decision
   (SSEReplaySeq), the report should say so in the same breath, not as a follow-up item.
5. **Decide baseline durability mechanics before running** — where files live, how benchstat is
   obtained reproducibly, whether CI gates them. Running first, plumbing second risks re-runs.

---

## f) NEXT — prioritized

**P0 — finish item 2 (benchstat baselines), XS**

1. Copy the 3 protocol outputs into `benchmarks/` with dated names (e.g.
   `benchmarks/2026-08-17_falsesharing-{sqliteengine,projectionhost,metaengine}.txt`).
2. Add benchstat summary tables + re-run commands to
   `docs/benchmarks/2026-08-16_false-sharing-contention.md` (or a dated sibling evidence doc).
3. Update `docs/BENCHMARKS.md` micro-paths rows: baselines now committed + benchstat-comparable.
4. Tick TODO_LIST item 2 ([x], Effort XS) with pointer to the committed baselines.
5. SSEReplaySeq tie-breaker run on quiet machine; either confirms NO-PAD (variance) or reopens the
   layout decision per protocol (>10% padded win, reproducibly).
6. Answer g.2 (CI wiring) and, if yes, extend `scripts/benchmark-regression.sh` bench set or add a
   second dated baseline consumed by the benchmarks.yml job.

**P1 — implement item 1 (durability Option A), M**

7. **Naming review FIRST (user's explicit gate)** — candidate shape: preset tier mapping stays
   `WithDurability`; storage/pebble backend-level switch named for what it does (e.g.
   `WithBackendAsyncWrites()` mirroring the existing per-store `WithAsyncWrites` family) — verify
   against `naming-review` conventions before writing code.
8. `stack/pebble/preset.go`: map Strict→sync writes (default), Normal→async WAL
   (app-crash-safe, kernel-crash window), Relaxed→DisableWAL; thread the sync flag through
   `cqrspebble.Open`→`newBackend`→all six stores (backend.go:74 seam).
9. Fix the doc split brain: `stack/durability.go` Pebble tier translations (lines 12, 29, 42, 53) and
   `stack/pebble/preset.go:37-40` must say the same thing.
10. `stack/bbolt` consistency per g.1 (Normal semantics + default-tier question).
11. Engines: opt-in async/no-sync options for `pebbleengine` (15 Sync sites → one writeOptions seam)
    and `bboltengine` (options struct around `bolt.DefaultOptions`), defaults unchanged.
12. Tests: tier→options mapping table test (stack/pebble), backend option threading test
    (storage/pebble), engine option smoke tests.
13. Measure the win: append-throughput bench Strict vs Normal on disk-backed pebble → BENCHMARKS.md
    entry (fsync-per-append cost made visible).
14. CHANGELOG `[Unreleased]` **Changed**: Normal-tier pebble preset writes now async-WAL (behavior
    change, minor version) + new options; api-stability golden regen; skill references
    (`recipes.md`/`modules.md`) if new exports; `nix run .#verify` exclusive.
15. TODO_LIST: close the durability item (unblocked → done) once green.

**P2 — hygiene from this session**

16. Reproducible benchstat: add `golang.org/x/perf/cmd/benchstat` to the flake devShell (or document
    the canonical `go install` in BENCHMARKS.md's gate section); /tmp/bin is not a story.
17. AGENTS.md gotcha candidate: "protocol baselines run from workspace root; per-module GOWORK=off
    runs can fail on sibling go directives AND produce non-comparable numbers" (if not already
    implied by the existing GOWORK entry — merge, don't duplicate).
18. Evidence-doc protocol hardening: add "check uptime/load first" line to the false-sharing
    evidence doc header.

---

## g) QUESTIONS (cannot be answered from the repo)

1. **bbolt Normal-tier semantics (blocks f.10):** Pebble's async-WAL Normal is app-crash-safe; bbolt
   has NO WAL — its only async knob `NoSync` also loses data on APP crash. Should bbolt Normal (a)
   stay sync-on-commit (safe, no perf win, documented exception), or (b) map to `NoSync` to match
   the "Normal de-escalates" pattern (faster, but app-crash-unsafe — a different durability class
   than pebble/sqlite/pg Normal)? Related: `stack/bbolt` preset DEFAULTS to Strict today while
   pebble/sqlite default Normal — align the default too, or keep Strict?
2. **Do the 3 false-sharing baselines belong in the CI regression gate?** (a) evidence-doc baselines
   only — manual re-runs, zero CI cost; (b) wire into `benchmark-regression.sh`/benchmarks.yml —
   regression protection but +runtime on every master push and hardware-pinned comparisons.
3. **How quiet is "quiet enough" for baselines?** This session's runs executed at load 1.3–2.1
   (45 user sessions; results mostly ±1–7%). (a) Accept them as the committed baselines; (b) I
   re-run all three exclusive (idle load) before committing; costs one more cycle, removes the doubt
   — also serves as the SSEReplaySeq tie-breaker (c.1/f.5).

---

_Baseline context: session started from a tree carrying foreign uncommitted changes
(`metaengine/mysqlengine/graph*.go` modified, `event/metadata_cbor_test.go` untracked) — untouched
throughout and still present. Auto-commit daemon may have committed work mid-session as usual._

---

## h) RESUME SESSION (2026-08-17 ~14:35) — autonomous decisions on §g

The resumed session was instructed to execute without waiting. Decisions taken autonomously:

1. **g.1 → (a) bbolt Normal stays sync-on-commit; preset default stays Strict.** The tier contract
   promises Normal = safe against app crash. bbolt has no WAL; its only async knob (`NoSync`) is
   documented by upstream as dangerous and carries a murky corruption story on unclean shutdown —
   a library must not encode a guarantee the storage engine refuses to make. bbolt simply has no
   app-crash-safe middle tier: Normal ≡ Strict for bbolt, recorded as an explicit exception in
   `stack/durability.go` (bbolt row added to the tier table). Default stays Strict (not aligned to
   Normal) deliberately: since Normal ≡ Strict behaviorally, a Normal default would be a no-op
   label today but a silent durability drop the day anyone "fixes" bbolt Normal→NoSync. Strict
   default makes that trap impossible. `DurabilityRange` keeps advertising all three tiers.

2. **g.2 → (a) evidence-doc baselines only, no CI gate.** The three false-sharing benches measure
   cache-line contention geometry — hardware-pinned, and the observed SSEReplaySeq ±56% cell would
   flake a 25%-median gate. Their value is the RELATIVE padded-vs-adjacent comparison (protocol:
   pad only if padded wins >10%), which a median-ns/op gate cannot express.
   `scripts/benchmark-regression.sh` stays the ONE gate (stack/bench set). Committed baselines +
   re-run commands in the evidence doc replace CI wiring.

3. **g.3 → re-run ONLY the anomalous suite (SSEReplaySeq) as tie-breaker; accept the two tight
   suites.** The two stable suites (sqliteengine ±1–7%, projectionhost ±1–7%) reproduce the
   recorded decisions — re-running them at today's HIGHER load (2.28–3.10 vs 1.3–2.1) can only
   add noise. The SSEReplaySeq tie-breaker decides DIRECTION, which ambient load does not invent.
   Mechanism argument from the recorded decision still holds: `record()` touches BOTH `seq` and
   the mutex-guarded fields every call, so padding the pair cannot pay; a padded win must come
   from allocation size-class side effects. NO-PAD stands unless the tie-breaker shows a clean
   (>10%, both cpu counts, tight variance) padded win — then the decision reopens per protocol.

4. **Naming review outcome (user's explicit gate, applied per naming-review checklist):**
   - `cqrspebble.BackendOption` + `WithBackendAsyncWrites()` — extends the existing
     `With{Command,Query,Snapshot,Checkpoint}AsyncWrites` family; scope prefix is the type name
     (`Backend`). Bare `WithAsyncWrites` is taken by EventStore's `StoreOption`.
   - `pebbleengine.Option` + `WithAsyncWrites()` — mirrors `storage/pebble` vocabulary for the
     identical mechanism (per-write WriteOptions). Also added to `NewPebbleEngineFromDB` (per-write
     sync is a store-level concern there, so it applies cleanly).
   - `bboltengine.Option` + `WithNoSync()` — deliberately NOT `WithAsyncWrites`: bbolt has no WAL,
     so "async writes" would falsely suggest the same durability class as pebble's async WAL.
     `WithNoSync` states the actual mechanism (bbolt's own `NoSync` + `NoFreelistSync` companion),
     each engine option names its backend's native knob. Not added to `NewBboltEngineFromDB`
     (NoSync is fixed at `bolt.Open` time; changing it post-open is impossible).
   - Preset API unchanged: `WithDurability` was already the right name; only the tier mapping
     and docs change.

5. **Latent bug found during research:** today's Relaxed mapping sets `DisableWAL=true` but stores
   still write with `pebble.Sync` — with the WAL disabled, pebble turns Sync into a memtable
   flush, so Relaxed (the "fast" tier) forces the most expensive write path. Option A fixes this
   as a side effect (Relaxed now also gets async writes). Recorded in CHANGELOG Changed.
