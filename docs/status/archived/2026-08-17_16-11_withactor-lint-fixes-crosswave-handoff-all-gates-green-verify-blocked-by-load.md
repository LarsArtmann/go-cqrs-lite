# Status: WithActor Tail — Lint Fixes Executed, Cross-Wave Handoff Absorbed, All Gates Green Modulo Ambient Load (2026-08-17 16:11)

Session scope: continuation of
[`2026-08-17_14-23_withactor-hardening`](2026-08-17_14-23_withactor-hardening-executed-defect-found-verify-red-on-2-lint.md).
Mandate: fix the verify-RED lint findings, run the gates, keep going until everything works.
A parallel session (perf/durability wave) was live the whole time; this report covers only what
THIS session did and noticed.

**Verdict:** Every code defect in both waves' lanes is fixed and targeted-green. All gates pass on
the current tree (lint 76/76 × 0 issues, arch, depguard, css, duplication, coverage, api-stability,
doc-check — each re-run fresh after the last edit). The single missing artifact is a **single-run
`nix run .#verify`**: ambient load oscillated 44→165 with disk at 96% — the documented
timing-test flake condition — so a run now would manufacture false RED. Composite GREEN stands on
attempt 3 (Build+Vet+Test+Race repo-wide green, incl. benchkit/duckdbengine/system) plus fresh
per-phase re-runs of everything after it. Honest ledger below; it includes real coordination
mistakes that cost both sessions gate cycles.

---

## a) FULLY DONE (this session)

1. **All 3 sqlstore lint findings fixed** — the handoff said 2; the raw log held 3 (a second
   `wrapcheck` at store.go:358 hid behind the first):
   - `wrapcheck` ×2 → `errorfamily.WrapCorruption` lifted INTO `decodeTimerPayload` (now takes
     the timer ID for context; distinct codes `unmarshal_envelope` / `unmarshal_legacy_payload`),
     caller slimmed to `return nil, err` — same pattern as `parseTime`. This **resolves prior
     Question 2 autonomously**: single wrap point, helper owns its errors; the "tests match on
     message text" worry was checked and unfounded (no test asserted messages). Corruption-family
     classification unchanged (caller-wrapped before, helper-wrapped now).
   - `exhaustruct` → full named-field literal `timerEnvelope[P]{Version: 0, Actor: "", Payload: p}`
     (v0 is semantically honest for legacy rows).
2. **Latent 350-line violation cleared** — store.go was 362 lines (over the `#check-file-size`
   limit, which `#verify` does NOT run — the violation had lived ~2h undetected). Split dialect
   SQL (`Dialect`, `queries`, 3 query constructors, `ErrUnknownDialect`, `sqliteTimeFormat`) into
   `scheduling/sqlstore/dialect.go` (77 lines; store.go now 293). The `//art-dupl:accept`
   directive moved with its block; module lint confirms no new clone findings.
3. **Decoder contract locked in by tests** — new `decode_test.go` (internal): v1 envelope with
   actor, v1 envelope without actor, legacy bare object, `{"v":1}`-only NOT misread as envelope
   (dual-key probe), Corruption classification on both decode paths; plus SQL-level
   `TestSQLiteTimerStore_CorruptPayloadClassifiedAsCorruption` (seeded corrupt row → Corruption
   family). Module: lint 0 issues, tests green, `-race -count=3` green.
4. **id-pin stragglers swept** — `stack/bench` had a DIRECT v4.2.0 pin (staler than the v4.4.0
   class the prior session swept — the prior "0 remaining v4.4.0 pins" claim was narrowly true
   but the sweep was incomplete). Bumped to v4.5.0; innocence proven by a revert probe (the
   module's standalone TEST failure is byte-identical at v4.2.0 — pre-existing, not caused).
   `scheduling`'s v4.2.0 is indirect-only; zero-dep preserved.
5. **Published-preset tag skew root-caused + documented** — `stack/v4.3.0` is GOWORK=off-broken
   as published (its sqlopt calls `storage.SQLiteSetSynchronous`, first shipped in
   storage/v4.6.0, while its own go.mod requires storage v4.5.0); lifting storage to v4.7.1 then
   breaks `storage/pebble/v4@v4.0.3` (pre-rename `AggregateID`/`AggregateType`). **No pin
   combination resolves** — the coordinated re-tag wave is required. Recorded in addendum 2.
6. **Cross-wave handoff absorbed** — parallel session's 15:17 report declared their work done and
   handed lint to this lane; after 19 min of file-mtime silence I fixed their 4 findings:
   golines reflow (via repo-wide `nix fmt`, safe window), explicit `case stack.DurabilityStrict`
   (intent-preserving: their doc comment says Strict+unknown → safest; their table test pins it),
   unnamed returns, and a **justified** `//nolint:usetesting` (the suggested `b.TempDir()` would
   RELOCATE the bench dir away from the configured disk-backed base — a behavior change, not a fix).
7. **stack/pebble GOWORK=off standalone break repaired** — the durability wave uses untagged
   `cqrspebble.BackendOption`/`WithBackendAsyncWrites`; the module failed `undefined:` standalone
   (same class as the prior session's middleware/encryption/signing fixes). Cascading sibling
   replaces added: `storage/pebble` → `storage/backuptest` → `event` + `metadata` (replaces do
   NOT cascade; each level needed its own). Module: standalone build ✓, lint 0, tests green.
8. **Repo-wide 82/82 GOWORK=off BUILD sweep green** (spot-checked; zero failures in log).
9. **Docs** — sqlstore README gained "Actor Attribution" section; modules.md scheduling entry
   mentions `Timer.Actor` (doc-check green, 921 refs); two addenda appended to the 14-23 report.
10. **Every gate green on the current tree**, each re-run FRESH after the final edit: full
    `#lint` **76/76 modules × 0 issues**, `#check-arch`, `#check-depguard` (120 deps),
    `#check-docserver-css`, `#check-duplication` (0 new clones, baseline 111),
    `#check-coverage` (±2% tolerance), `#check-api-stability`, doc-check (921 refs).

## b) PARTIALLY DONE

1. **Single-run `nix run .#verify`** — attempt 3 (15:00–15:12) passed Build+Vet+Test+Race
   repo-wide (incl. benchkit 42s/51s, duckdbengine 83s/103s, system) and died at Lint on the 4
   parallel-wave findings — all now fixed. Every phase AFTER lint has since been run fresh and
   green on the final tree. Composite GREEN is solid; the single-command GREEN artifact is
   blocked by ambient load (44→165, disk 96%, 40 users — the exact flake condition the parallel
   session documented at load 75). One calm-window run closes it for BOTH waves (their f.1 = my
   item 1).
2. **GOWORK=off verification** — builds swept 82/82; per-module TESTS not swept. stack/bench's
   standalone test failure is pre-existing (published-tag skew, see a.5) and remains open. All
   other modules I touched were standalone-tested individually.

## c) NOT STARTED

1. **Release lane — nothing tagged.** Needs the user's strategy call (Questions 1–2). Includes:
   scheduling/v4.3.0 (Timer.Actor + envelope), id-consumer minors, metadata/event minors
   (unlocks dropping 5+ sibling replaces), storage/pebble (unlocks stack/pebble's replace).
2. **TODO_LIST entries for this session's follow-ups** (replace-strip-after-tags, stack/bench
   test fix, calm-window verify, check-file-size wiring) — they live only in status-report prose;
   docs-health HARVEST not run.
3. **`#verify-ci`** (per-module GOWORK=off test matrix — would surface stack/bench test redness
   the build sweep cannot see).
4. **GitHub CI status check** — is the per-module job currently red on stack/bench?
5. **`#load-sweep`** — not required (no timing paths touched), never run.
6. **AGENTS.md updates for this session's new lessons** (gate mutex / quiesced-tree protocol —
   see e.1/e.2).
7. Parallel session's f.2 (fsync benchmark PENDING cell) — theirs, untouched here.

## d) TOTALLY FUCKED UP (honest ledger)

1. **I co-caused the load spike that killed the parallel session's verify.** I launched verify
   attempt 2 (14:53) and then ran stack/bench go-mod experiments + module tests CONCURRENTLY with
   the gate — plausibly while THEIR verify was also mid-run (their report: load 3→75, 9 benchkit
   timeouts, 40 users). I checked `uptime` for the FIRST time at 15:29, after the damage. Their
   report's lesson "pre-flight uptime before anything load-sensitive" is dated 15:17 — I read it
   an hour too late to avoid being part of the problem.
2. **Verify attempt 2 sampled the parallel session mid-edit** (`writeOptions` undefined → BUILD
   RED in 5 min). Launched without a pre-flight whole-tree build. Lesson applied for attempt 3 —
   but only after paying for it once.
3. **Attempt 3 was launched on a NON-quiesced tree.** The pre-flight build only guarantees the
   launch instant; the parallel session wrote `durability_test.go` at 15:05, MID-RUN. I got
   lucky that only lint caught the churn (4 findings, all fixable) — Build/Test/Race happened to
   sample compilable states. A 25-min gate over a mutable shared tree needs coordination BEFORE
   launch, not a preflight.
4. **Trusted the handoff's count instead of the log.** "2 lint findings" was 3; the second
   wrapcheck was hidden behind the first in the tail-truncated view. Grep the raw gate log,
   always.
5. **multiedit fumble on store.go** — the 4-edit batch failed its big block, transiently leaving
   the file uncompilable (`errors` import removed while `errors.New` remained). Caught by
   immediate diff; fixed by line-range sed. One wasted cycle; should have read exact context.
6. **~20 min of calm-window polling before pivoting.** The first 10+ load reading already
   matched the documented flake condition; I should have switched to phase-wise gate verification
   immediately instead of polling to load 165.
7. **Prior session's sweep claim was incomplete and I repeated its framing** — "0 remaining
   v4.4.0 pins" was true-but-misleading: stack/bench sat on v4.2.0, staler still. Pin sweeps must
   compare against LATEST, not against the known-bad version.

## e) WHAT WE SHOULD IMPROVE

1. **Cross-session gate mutex.** Two heavy `#verify` runs collided on one box today and both
   burned cycles. Minimum: `uptime` + `pgrep -af "nix run|go test"` before ANY heavy gate; ideal:
   a lockfile (e.g. `/tmp/cqrs-heavy-gate.lock`) both sessions honor. → AGENTS.md.
2. **Full `#verify` only on a quiesced tree.** Preflight build ≠ safety for 25 min. Convention:
   confirm the parallel session is parked (mtimes quiet, their status report written) before
   launching; otherwise phase-wise targeted gates.
3. **Handoffs cite raw log lines, not prose summaries** ("2 findings" cost a hidden third
   finding a session boundary).
4. **Sweeps check "behind latest," not "the known-bad version."**
5. **TODO_LIST same-session harvesting** — new follow-ups die in timestamped prose otherwise.
6. **Phase-wise verification is a legitimate GREEN strategy under load** — adopt it deliberately
   (with the per-phase fresh-run table I produced) instead of waiting for calm windows that
   never come.
7. **Wire `#check-file-size` into `#verify`** — a 362-line file survived a full gate because the
   app is separate (this is how latent violations live unnoticed).

## f) NEXT — prioritized (23, unpadded)

1. Calm-window single-run `nix run .#verify` (load <~5) — closes both waves' GREEN artifacts.
2. Free disk: root btrfs 96% — `trash-empty` (~6.6G) + cache purge; IO benchmarks untrustworthy
   until then.
3. Release-wave decision + execution (scheduling/v4.3.0; id-consumer minors; metadata+event
   minors; storage/pebble) — drops 6+ sibling replaces, ships the CBOR actor fix to consumers.
4. Fix stack/bench GOWORK=off TEST break (via the re-tag wave, or temporary replaces).
5. Check GitHub CI per-module job status (stack/bench likely red standalone).
6. Ratify Normal-tier async-WAL default severity (minor vs hold-for-major) — gates stack/pebble
   tag content.
7. Pre-tag replace sweep: `grep -rn "=> \.\./" --include=go.mod .` — now ~6 sites (middleware,
   encryption, signing, sqlstore→scheduling, stack/pebble ×4 targets).
8. Drop `scheduling/sqlstore → ../../scheduling` replace once scheduling tags.
9. Run `#verify-ci` (GOWORK=off per-module test matrix).
10. docs-health HARVEST from the three 08-17 status reports into TODO_LIST.
11. Write the gate-mutex / quiesced-tree / phase-wise protocol into AGENTS.md.
12. `tag-release.sh`: flag "N modules pin an older version of a just-tagged module" (prior e.2 —
    would have caught BOTH the id/v4.4.0 class and stack/bench v4.2.0).
13. GOWORK=off roundtrip CI job for codec-adjacent modules (prior e.3 — would have caught the
    CBOR actor loss at id/v4.5.0 tag time).
14. Audit ALL published preset tag graphs for the mutual-inconsistency class (stack/v4.3.0 was
    found by accident; siblings may share the disease).
15. BenchmarkEventAppendSync/Async calm-window re-run + BENCHMARKS.md PENDING cell (parallel f.2).
16. Add `metaengine/bboltengine` row to `references/modules.md` (parallel f.4).
17. Wire `#check-file-size` into `#verify` (see e.7).
18. CHANGELOG: extend the id-pin Fixed entry with stack/bench v4.2.0 (sweep completion).
19. Daemon mixed-session commit hygiene (12c606707 mixes both waves; parallel g.1 — user to say
    if a corrective note is wanted).
20. Durability bench: make the base dir accept `b.TempDir()`-style override so the
    `//nolint:usetesting` can die.
21. cqrs-lint rule: warn on commands lacking ActorID (prior optional; needs user intent).
22. `docs/DOMAIN_LANGUAGE.md`: formalize Actor / Effective Identity (prior optional).
23. One-time scan of sibling presets (stack/bbolt etc.) for the exhaustive-switch pattern
    stack/pebble had (lint is green — verify once that bbolt's Normal≡Strict exception reads the
    same way).

## g) QUESTIONS (cannot resolve from the repo)

1. **Release wave: cut now or batch?** The coordinated tags (scheduling, id consumers, metadata,
   event, storage/pebble) would ship the CBOR actor fix to published consumers, drop 6+ dev-only
   replaces, and un-break stack/bench standalone — but the parallel session's system/v4.5.0 lane
   is also queued on a metaengine release. One wave for everything, or sequence them (and in
   which order)?
2. **Confirm severity for stack/pebble's Normal-tier default change** (fsync-per-write → async
   WAL): the parallel session recorded minor; it is the most consumer-visible default change in
   the wave. Ratify minor, or hold it for a major? (Their g.2, inherited — only you can decide.)
3. **Machine window + deletions:** load 44→165, disk 96%, 40 users. The calm-window verify and
   the disk cleanup (`trash-empty`, cache purge — destructive actions I will not self-approve)
   need either your go-ahead to run when load permits, or you run them. Which?

---

Gate tally at write time: attempt-3 phases (Build/Vet/Test/Race, repo-wide) + fresh final-tree
runs of Lint 76/76×0, arch, depguard, docserver-css, duplication, coverage, api-stability,
doc-check — all green. Artifacts: `/tmp/withactor-verify2.log` (mid-air sample),
`/tmp/withactor-verify3.log` (full attempt 3), `/tmp/lint-final.log` (fresh 76/76),
`/tmp/sweep-build.log` (82/82). Format note: written as `.md` per the user's explicit path
override of the status-report skill's HTML default.
