# Status Report — CI Un-Red Continuation: Turso Cold-Start Race, Catch-Up Double-Fold, Config-Corruption Restore (2026-10-14 17:19)

**Session:** 2026-10-14 ~11:50 → 17:19 CEST. Continuation of the green-CI
wave (status 05-30). Trigger: standing "execute until done" instruction.
Scope: chase every red on the authoritative full-matrix runs
(38548187646, 38569359207) to ground; keep the composed verify chain alive.

## a) FULLY DONE (verified locally + gates green)

1. **State assessment** — tree had uncommitted formatting + go.sum churn
   (daemon/parallel-session residue); verified benign, touched nothing
   foreign. The 05-30 report's "authoritative run" 38505591805 was already
   superseded; identified each newer run as the daemon pushed.
2. **Nightly Gates red FIXED** — `docs/status/README.md` live index was
   missing 7 report rows (the exact rot class the canonical-facts gate
   exists for). Read every report, added honest one-liner rows;
   `check-canonical-facts.sh` green (missing=0 dangling=0).
3. **shfmt drift FIXED** — `scripts/probe-proxy-tags.sh` heredoc-in-if
   drift (committed 25f92b83d); reformatted with the pinned shfmt,
   `--self-test` still green, CI gate clean.
4. **`.golangci.yml` corruption RESTORED** — daemon auto-commit 25f92b83d
   had deleted the entire 92-line depguard allow-list + carve-out comments
   (the THIRD occurrence of the documented gvac/daemon class). Followed the
   hash tripwire's own remediation path 2 (restore, don't re-pin) from
   parent 1c2981eb8; `check-golangci-hash.sh` PASS (4fcb57f9…).
5. **storage/turso cold-start panic ROOT-CAUSED + FIXED** — the
   deterministic `Module storage/turso` CI red (run 38548187646):
   `turso-go-platform-libs` v0.8.2 writes the shared native-library cache
   (`turso-go/<hash8>/libturso_sync_sdk_kit.so`) with plain
   create+copy — no lock, no atomic rename — so two processes cold-starting
   one cache interleave writes and the loser PANICS from
   `tursogo.InitLibrary` ("cached library file hash sum mismatch").
   Reproduced locally FIRST-TRY (cold XDG_CACHE_HOME + `go test ./... -race`).
   Fix: `library_warmup.go` twins in storage/turso + metaengine/tursoengine
   (tier rules forbid a shared home; `//art-dupl:accept`ed, register.go
   pattern): once per process, `turso.Open`/`tursoengine.New` run a
   lockfile-guarded throwaway `:memory:` ping (driver loads the library for
   ANY DSN before DSN parsing → a broken caller DSN can't poison the
   warmup); lock failures degrade to unlocked ping; the driver's PANIC
   becomes a returned Infrastructure error. Pinned by: re-exec test (6
   child processes racing one fresh cache dir, each asserting the extracted
   `.so`), lock-blocking unit test, cold-cache engine-open test. Verified:
   both modules' full `-race` suites 3/3 cold-cache rounds green
   (pre-fix: first-try failure); consumers green (stack/turso tested;
   cqrs-bench, stack/bench build); lint + gofumpt/goimports/golines clean.
6. **metaengine CatchUpEngine double-fold ROOT-CAUSED + FIXED** — the Nix
   Flake Check Test red on run 38569359207
   (`TestEngineHealth_CatchUpUnderConcurrentApplies`: primary ticks 2002 vs
   exactly 2000; first CI exposure of the test; 9 local runs green =
   scheduler-luck window). Mechanism: the catch-up gate
   (`EventLog.reactivateIfStable`) proves log stability and reactivates
   under the append lock + healthMu ONLY — but an apply that recorded its
   event before the final replay pass can be parked between record and
   fold (applyWithRecord holds s.mu across both); when reactivation flips
   the engine active inside that window, the apply's live fold routes into
   the just-rebuilt engine AND the replay already folded the same event.
   Fix: the gate now takes the s.mu WRITE lock before
   `reactivateIfStable` — once it returns, every in-log event has finished
   its live fold, and that fold routed away from the still-quarantined
   target. Lock chain s.mu → l.mu → healthMu verified consistent (no
   healthMu section ever takes s.mu or l.mu); `reactivateIfStable` doc
   contract updated. Verified: stress 8/8, `-race` health 3/3, full
   metaengine suite green; lint + formatters clean.
7. **art-dupl gate kept green** — my twin suppressed by directives; found
   ONE new foreign group (explain_doctor/reset 3-line RLock prologue) —
   annotated both sides; pinned v0.7.0 check: 0 new groups (baseline 187).
8. **`docs-ui.css` rebuilt** — the stale-artifact Verify-fast red (a
   parallel session edited sources without rebuilding); regenerated via
   `nix run .#build-docserver-css` (force-run under the IO storm: cached
   derivation, 156ms tailwind run).
9. **Records** — CHANGELOG [Unreleased] Fixed rows (turso warmup; catch-up
   double-fold; both honest per `check-changelog-symbols.sh`), gotchas
   entry (tursogo loader race, gotchas-language-footguns.md), TODO_LIST
   row (upstream loader-race issue owed; filing owner-gated).
10. **Composed verify chain re-armed RESILIENTLY** — v1 died when
    `#check-coverage` was refused by the no-heavy-builds guard (IO storm,
    exit 12) and the `&&` chain aborted. v2 retries rc=12 only (real gate
    failures still fail fast), 40 × 5min per step; wait-loop GREEN after
    23 attempts on the first arm.
11. Scratch worktree cleaned up; no stray files left.

## b) PARTIALLY DONE (in flight at report time)

- **The referee CI run** — all fixes sit in daemon commits (latest local
  fccfe118b); the push-wait loop was still polling (origin last carried
  38569359207's tree). The NEXT full-matrix run referees: the catch-up fix
  (its red was deterministic on slow runners), the CSS artifact, the
  config restore, and everything from earlier waves in one tree.
- **Composed verify chain (bg shell 08D)** — armed and riding the IO
  storm; retry 38/40 at last check (load5 32+, PSI-gated). Will likely
  exhaust → RE-ARM (same /tmp/verify-chain.sh recipe; consider moving it
  into scripts/ as a first-class runner — see e3).
- **M10 receipts** (prior session §c2: plan §8 receipt, TODO_LIST row,
  F-64 close-out addendum on the 19-26 report) — deliberately NOT written:
  they are gated on CI green + composed-verify green, neither proven yet.

## c) NOT STARTED

- Upstream tursogo loader-race issue filing (owner-gated; TODO row created
  with the full mechanism + ask).
- Deterministic regression test for the catch-up straddle window (the
  interleaving is scheduler-luck; a pin needs an injectable seam in
  dispatchFoldsLocked — documented as e5, not built).
- The tag wave (user-gated per prior §g3; projectionadapter+system
  co-release pairing unchanged).

## d) TOTALLY FUCKED UP (my own mistakes, all caught + fixed same session)

1. **Invalid first "reproduction"** — reused `$TMP` from a previous shell
   (shells don't share state) → XDG_CACHE_HOME=/libcache →
   permission-denied panic that was NOT the defect; nearly mis-concluded.
   Caught it because the panic text differed (cache-dir create vs hash
   mismatch). Lesson: self-contained temp setup per command.
2. **Test bug** — `cmd.CombinedOutput()` after `cmd.Start()` ("already
   started"); rewrote with buffer+Wait.
3. **Wrong art-dupl binary** — first duplication check ran the LOCAL
   `dev` build, which flagged a phantom templ group; the nix-pinned v0.7.0
   flagged only the real group. Almost performed an unnecessary baseline
   surgery on a tool-version artifact.
4. **Verify chain v1 fragility** — didn't anticipate the load guard
   aborting the chain; ~40 min of wait-loop arming wasted before v2.

## e) WHAT WE SHOULD IMPROVE

1. **The depguard/config corruption keeps recurring** (3rd class instance;
   the daemon absorbs a gvac/config-rewrite mid-session). The durable-guard
   TODO row exists — it should outrank most backlog: each occurrence costs
   a CI cycle + a hash-tripwire red.
2. **Pin the local art-dupl** (or alias it to the nix-pinned v0.7.0) — a
   version-skewed clone report invites false baseline surgery (d3).
3. **Ship the resilient composed-verify runner in scripts/** —
   /tmp/verify-chain.sh (retry-on-guard-refusal, fail-fast on real reds)
   is exactly the "re-arm if the window lapses" policy, executable by any
   session instead of re-deriving it.
4. **CI log-fetch friction** — job logs are unfetchable until the run
   completes AND the API 502s frequently; polling cost ~10 min per red.
   A `gh run watch`-based or artifacts-based tail would compress triage.
5. **Injectable seam for concurrency pins** — the catch-up boundary class
   is only CI-refereeable today; a test-only hook in the fold dispatch
   path would make the straddle deterministic (and would have pinned this
   fix).
6. **tursogo upstream engagement as a batch** — two defect classes now
   (SIGSEGV row + loader race row); one coordinated upstream thread beats
   per-defect asks.

## f) NEXT (roughly Pareto-ordered)

1. Watch the referee CI run to green (it carries every fix from this
   session + the 05-30 wave).
2. Re-arm the composed verify chain if 08D exhausts; when green, record
   M10 receipts (plan §8, TODO_LIST row, F-64 addendum).
3. durable guard for .golangci.yml corruption (the recurrence killer).
4. File the tursogo loader-race upstream issue (after owner go-ahead).
5. Move the resilient verify-chain runner into scripts/ (+ --self-test).
6. Pin/alias local art-dupl to v0.7.0.
7. Deterministic seam + regression test for the catch-up straddle.
8. Tag wave (owner-gated): stack/v4.6.0, metaengine/v4.19.1,
   cmd/cqrs-lint/v4.18.0, systemscenario/v4.2.0, projectionadapter+system
   pair — my metaengine/storage-turso fixes ride it.
9. Turso segfault row: track upstream for a fixed release, then bump.
10. Benchmark noise-gate policy verdict (prior §g1, still open).
11. Upstream tursogo issue batch (loader race + segfault context).
12. Consider deduplicating the warmup twins at v5 (single core/v5 home is
    legal there — tier rules only block the v4 shape).
13. Sweep: confirm no OTHER module opens turso DBs outside the two warmed
    entry points (spot-checked: stack/turso routes through storage/turso;
    re-verify at tag time).
14. Harvest any remaining §f rows from the 05-30 report into TODO_LIST if
    not yet there (docs-health HARVEST discipline).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Tag wave go-ahead** (prior §g3, unchanged, now carrying more fixes):
   cut it after the referee run greens, or hold for your review of the
   catch-up lock-order change (s.mu at the gate) first? The lock-order
   reasoning is documented but it IS a core-store concurrency change.
2. **Benchmark noise-gate policy** (prior §g1, still open): keep-fail-loud
   / raise CI threshold / bump repeats / demote write_throughput with
   evidence — which ruling?
3. **File the tursogo loader-race issue upstream in my name?**
   (tursodatabase/turso-go-platform-libs: ask for flock or temp+rename in
   `embeddedLibraryTryCreate`; I have the minimal repro + mechanism ready —
   filing is owner-gated by repo norms.)

## Post-script (state at 17:19)

Origin tip still = 38569359207's tree (failure, pre-fix); local ahead with
all session fixes awaiting the daemon push; load1 57 (storm); chain 08D at
retry 38/40. The referee run + chain green are the two outstanding proofs;
everything else in this report is local-gate-verified.
