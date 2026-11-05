# 2026-11-05 23:52 — Verify fast-failed on ADR-index split brain; window consumed by preventable 32-second fail

Session: 22:00–23:52 CET (continuation of the post-sweep M01–M26 execution arc; plan
`docs/planning/2026-11-05_01-48_SUPERB-post-sweep-release-and-adoption-pareto-plan.md`).
Supersedes: `2026-11-05_21-45_issue-114-landed-all-green-verify-armed.md` for verify
state only (the 21:45 report's landed-work inventory remains accurate).

## Headline

The 12h watcher (launch-14) worked exactly as designed: at 23:15:34 CET, after 97
minutes of polling (iter 585), the quiet window opened (load sum 15.31, loadavg at
fire `6.54 8.77 14.66`) and verify launched. **Verify then fast-failed 32 seconds
in** (23:16:06, rc=1) at the doc-assertions phase:

```
=== ADR index completeness ===
FAIL: 162 ADR files exist but only 161 are indexed in docs/README.md
```

The heavy legs (build / vet / test / race / lint / doc-check) NEVER RAN. The #114
grouped-caps code and the whole evening landing remain unverified by a full verify.
The expensive window was consumed by a one-line documentation miss.

## Root cause (verified this session)

- The repo has TWO ADR indexes: `docs/README.md` (the master docs index; the one
  `scripts/verify-docs.sh` line 83 checks) and `docs/adr/README.md` (the ADR-dir
  index; where ADR-0164's row WAS added, line 269, at 21:02).
- The evening landing session added the ADR-0164 row only to `docs/adr/README.md`.
- `nix run .#check-adr-numbering` (the gate the deprecation/ADR ritual names) checks
  the OTHER index shape and passed green — so the trail looked complete to every
  gate that ran before push. Only a real `#verify` doc-assertion catches it.
- `docs/README.md` was last touched 05:01 today; the fix is one table row (note: that
  file's row shape has no date column — copy the `0163` row shape, not the
  `docs/adr/README.md` shape).

## Self-critique — what I forgot / could have done better

1. **I inherited "ADR-0164 + index row landed" from the 21:45 report without
   re-verifying it against the RIGHT gate.** The prior session indexed one of two
   files; my session verified nothing about the ADR trail and armored a 12-hour
   watcher on top of an unverified doc claim.
2. **No cheap-phase preflight before arming.** `scripts/verify-docs.sh` runs in
   seconds. Running it (plus the other sub-second doc gates) BEFORE arming the
   watcher would have caught the red at 21:50 and tonight's window would have
   produced a FULL verify instead of a 32-second fail. The watcher design guards
   load risk but not fast-fail risk — that was my blind spot in the merge-restructure
   decision (two verifies collapsed into one 12h watcher = the blast radius of any
   pre-existing red became "the whole night").
3. **What still went right:** monitor job 072's handoff (block on rc file) returned
   the failure the instant it existed; root cause was isolated in under 10 minutes;
   no code was implicated; load is quiet again (~23 at 23:49, load1 12) so a
   re-verify after the one-line fix will fire quickly under either policy.

---

## a) FULLY DONE (this session, 22:00–23:52)

- **M04 watcher lifecycle**: armed → window found (23:15:34) → verify fired → rc=1
  captured → triaged to root cause. The launch-14 watcher process has EXITED
  (no watcher armed now; tree exclusivity lifted).
- **M07 plan correction — batch-release same-batch sibling limitation (major
  find).** A single 6-module invocation would tag sqliteengine pinning the OLD
  metaengine v4.20.0 (tidy resolves siblings to latest PUBLISHED tag) while its
  #114 code needs v4.21.0's `CapGroupTrimmer` — the standalone-compile guard would
  refuse mid-wave. Wave corrected to FOUR dependency-ordered invocations:
  A) id v4.7.5, kv v4.4.0, catalog v4.13.0, cqrs-lint v4.21.0 → B) metaengine
  v4.21.0 → C) sqliteengine v4.7.0 → D) systemscenario v4.3.0 (each: batch-release
  → push → smoke). TODO_LIST row rewritten with this.
- **Wave-tail row dispositioned**: pgengine is on `scripts/untagged-trains.txt`
  (ADR-0152 zero-consumer rule) — batch-release REFUSES it; the "fold pgengine"
  idea was impossible, no action needed (bug never tagged). systemscenario has real
  untagged exports (`RaceEnabled`) ⇒ folded into M07 as v4.3.0 (minor, not patch).
- **Untagged-backlog sweep (new evidence)**: systematic last-tag..HEAD sweep over
  all tagged modules (production files only) found ~20 MORE modules with real
  untagged drift beyond the M07 seven, accumulated since ~10-08: event (7 files),
  event/v4/eventtest (ReadAllFn/ReadFromFn ⇒ v0.5.0), scenario (WhenVersioned ⇒
  v4.5.0), testutil (NewFaultInjector ⇒ v4.4.0), middleware (12), query (4),
  queue + queue/postgres + queue/sqlite, storage/bbolt (16) + storage/memory +
  storage/turso, system (5), listing (2), metaengine/pebbleengine (4), signing (1),
  record (2), stack + stack/sqlite, scheduling + scheduling/engine. Rowed in
  TODO_LIST as the next wave (with deprecated sqlstore policy call).
- **Parked Q3 answered with evidence**: all 103 go.mod files AND go.work are
  uniformly patch-form `go 1.27.2` (lockstep-consistent; floor 1.27 satisfied;
  check-go-version green) — the armed verify would NOT have failed on it; leave
  normalization to the sweep session.
- **CI forensics for M05**: old run 41978263864 completed=failure (mysql +
  actionlint + shellcheck legs) — but two GREEN full-CI runs exist AFTER it on
  master (41992212625 08:08Z, 42008305457 10:09Z) ⇒ the 06:07Z actionlint failure
  was run-specific, not standing breakage. A CI run (42096498453) was mid-flight on
  old origin/master (auto-cancels on push). Local master is 14 commits ahead.
- **Consumer collision checks + site research (read-only)**: cqrs-htmx LIVE
  (daemon commit 21:44) — M11 waits; crm LIVE (21:38 + 2 dirty files) — M12/M14
  wait; webphone quiet since 20:03; BuildFlow quiet since 18:52; go-appkit quiet
  since 11-04; **Kith = the crm repo** (plan line 42 "Kith CRM"). M13/M15 sites
  mapped: webphone `wirecontracts.go:130` (WithServer drop), `apidocs.go:50`
  (retryAfterClause → WithResponseHeader), `internal/asr/asr.go:213`
  (Transcribe → multipart), `dialwindow.go:46-52` (cap trims → WithCapPerGroup),
  catalog pin v4.12.0→v4.13.0; BuildFlow `go.mod:157` (cqrs-lint v4.18.0) +
  `flake.nix:286` (vendorHash).
- **Playbook written**: `build/wave-plan-2026-11-05.md` (gitignored) — the full
  post-verify execution sequence M05→M22 in copy-paste form.
- **CHANGELOG cut pattern confirmed**: one wave-section header
  (`## [catalog/v4.13.0, …, systemscenario/v4.3.0 — … (7 tags)] — 2026-11-06`),
  all current [Unreleased] entries move below it.
- **TODO_LIST updated (3 rows, daemon-absorbed)**: wave-tail disposition, corrected
  M07 four-invocation row, untagged-backlog row.

## b) PARTIALLY DONE

- **M04 verify**: watcher mechanics WORKED, but verify is RED on a one-line doc
  miss and the code legs are unproven. Fix = one row in `docs/README.md`, then
  re-run (load is quiet enough that either a direct run at the guard's threshold
  or a re-armed watcher fires fast — owner question 1).
- **M07 tag wave**: plan fully corrected and staged (versions, order, pgengine
  refusal, systemscenario promotion, CHANGELOG header text); execution not started
  (blocked on M04 green).

## c) NOT STARTED (unchanged from 21:45, all blocked behind verify green)

- M05 push + CI-green receipt; M06 changed-set tests + census; M07 execution;
  M19-tail #114 close; M11 cqrs-htmx adapter deletion; M12 crm scorecard pin;
  M13 webphone adoption; M14 crm MaxOpenConns + re-measure; M15 BuildFlow bump;
  M22-tail final report; the six improvement rows (verify-launcher promotion,
  ignore-cascade tripwire, generated-artifact pre-push sweep, wedged-run watchdog,
  erraudit wrapper, sse_test RaceEnabled cleanup).

## d) TOTALLY FUCKED UP

- **The 21:45 arc (my lineage) landed ADR-0164's index row into one of TWO index
  files and declared the trail done.** Every cheap gate passed (check-adr-numbering
  inspects the other file); the miss surfaced only inside a real `#verify`, 97
  minutes into the night, burning the scarce quiet window on a 32-second fail.
  Compounding: the 21:45 restructure MERGED two planned verifies into this single
  12-hour watcher, so the blast radius of that one missing row became the whole
  night's verification.
- **This session armored the watcher without a seconds-long doc-gate preflight**
  (see self-critique 2). Both misses are mine to own across the arc.

## e) WHAT WE SHOULD IMPROVE

- **Kill the ADR two-index split brain (structural)**: either make `docs/README.md`
  the single ADR index (docs/adr/README.md links to it / is generated from it), or
  make `check-adr-numbering` ALSO enforce docs/README.md coverage so both gates see
  both files. Today a row can be "landed" and gate-green while verify-red.
- **Add a pre-arm fast-fail preflight ritual**: before arming any
  quiet-window watcher, run the sub-second phases (verify-docs.sh, canonical
  facts, changelog symbols) directly. Cheap gates first, expensive windows never
  wasted. (Row candidate; also relevant to the verify-launcher promotion row.)
- **Consider doc-assertion ordering**: verify currently fast-fails docs BEFORE
  build/test/race — efficient when docs are the only problem, but it MASKED code
  verification for the whole night. With a preflight in place the current order is
  fine; without one, running the heavy legs first (or in parallel) would degrade
  more gracefully.
- **ADR landing ritual wording**: the deprecation/ADR procedures in AGENTS.md say
  "index row" without naming WHICH files — both indexes must be named, or the
  split brain fixed so there is only one.

## f) NEXT — up to 50 things, in order

1. Add the ADR-0164 row to `docs/README.md` (0163 row shape, no date column).
2. Re-verify: direct `VERIFY_MAX_LOAD=16 nix run .#verify > build/verify-run15.log`
   (load1 ~12 passes the launch guard) OR re-arm the strict watcher — owner Q1.
3. Triage run15 if red from code (grouped-cap files: cap_grouped.go ×3, cap.go,
   fold_classify.go, extract_helpers.go, query_config.go).
4. M05: `git push origin master`; poll the new CI run (old run auto-cancels).
5. M05: watch MySQL QEMU leg (`ConcurrentClaimersDisjoint` recurrence bar), Dgraph
   leg, benchmarks duration; tick T01.5 in TODO_LIST on green.
6. M06: changed-set `GOWORK=off go test ./... -count=1` over catalog, kv, id,
   metaengine, sqliteengine, cqrs-lint, middleware, event(+eventtest),
   storage(+pebble/bbolt/eventstore/relational), core/v5, system, systemscenario.
7. M06: `bash scripts/census-fleet.sh`.
8. M07-A: `./scripts/batch-release.sh "id v4.7.5" "kv v4.4.0" "catalog v4.13.0"
   "cmd/cqrs-lint v4.21.0"` → push tags → smoke.
9. M07-B: `"metaengine v4.21.0"` → push → smoke.
10. M07-C: `"metaengine/sqliteengine v4.7.0"` → push → smoke.
11. M07-D: `"systemscenario v4.3.0"` → push → smoke.
12. `bash scripts/check-versions-manifest.sh --update` + README module→tag manifest
    + `--check`.
13. `go get` smoke per tag from a clean dir.
14. CHANGELOG cut (wave header; 7 tags) + `bash scripts/check-changelog-symbols.sh`.
15. Re-run `nix run .#check-release-scripts` (post-wave rule) + versions.json
    nightly gate re-check.
16. M19-tail: `gh issue close 114` citing ADR-0164, the two tags, recipes §2.50.
17. Improvement row: ADR two-index split-brain structural fix (see e).
18. Improvement row: pre-arm fast-fail preflight (see e).
19. Improvement row: verify-launcher promotion to `scripts/verify-launch.sh`.
20. Improvement row: ignore-cascade tripwire (git ls-files vs disk, pre-push).
21. Improvement row: generated-artifact pre-push freshness sweep.
22. Improvement row: wedged-run watchdog + benchmarks `timeout-minutes` audit.
23. Improvement row: `nix run .#erraudit` wrapper around the vendored-tarball build.
24. Improvement row: `metaengine/sse_test.go` redundant RaceEnabled cleanup.
25. M11: cqrs-htmx delete `usermgmt/view_cache.go` (182 lines) →
    `kv.NewCacheForViewStore` + `WithCacheCopyFunc`; kv pins v4.3.5→v4.4.0
    (adminui/integration_test/loginpage/setup go.mods); suite green.
26. M12: crm `.cqrs-lint.json:53` scorecard.modules projectionhost pin + cqrs-lint
    v4.21.0 bump + "stays unwaived" comment.
27. M13: webphone — drop `WithServer("webphone","","")` (wirecontracts.go:130);
    WithResponseHeader for 429/retryAfterClause (apidocs.go:50);
    WithRequestBodyContent multipart for /api/transcribe (asr.go:213);
    WithTypedParam pagination; catalog v4.13.0 pin; adopt WithCapPerGroup for the
    dialwindow/transcript trim loops (dialwindow.go:46-52).
28. M14: crm `sqliteengine.WithMaxOpenConns` at the engine registration site
    (internal/app/app.go:16 vicinity) + re-measure the read storm.
29. M15: BuildFlow cqrs-lint v4.18.0→v4.21.0 (go.mod:157) + vendorHash re-pin
    (flake.nix:286).
30. Untagged-backlog wave prep: api-diff per module to derive versions
    (eventtest v0.5.0, scenario v4.5.0, testutil v4.4.0 first — new exports named).
31. Untagged-backlog: deprecated `scheduling/sqlstore` — final-tag-or-skip policy
    call.
32. M22-tail: final completion report superseding 06:39/09:46/18:10/19:08/21:45
    (+this one) + docs/status/README.md index + canonical-facts + md-go.
33. Post-everything: `nix run .#check-canonical-facts.sh` re-run (this report's row
    added below).
34. Owner decision sheet (fleet-audit arc leftovers) — still parked in TODO_LIST.
35. cqrs-htmx C03 closure: full `go test ./...` against aligned pins (disk-truth
    residue row) — can ride M11.
36. Fold `go 1.27.2` patch-floor normalization decision to the sweep session (Q3
    evidence recorded; gate green either way).
37. Re-check `.cqrs-lint.json` waiver hygiene in crm after v4.21.0 (waiverAdoption
    drift guard now pins hand-map).
38. Re-run `bash scripts/check-engine-capabilities.sh` receipt post-wave ( TrimGroupedCap
    column already regenerated pre-wave; confirm no drift).
39. Verify the versions.json nightly gate sees the 7 new tags cleanly.
40. Sweep-session residue: watch for go.sum breakage from concurrent sweeps
    (`GOWORK=off go mod tidy` per affected module if `missing go.sum entry`).

## g) Questions for the owner (cannot figure out myself)

1. **Re-verify policy right now**: machine is quiet-ish (23:49: load1 12.1,
   sum 22.97; the internal launch guard allows ≥16 on load1, so a direct
   `VERIFY_MAX_LOAD=16 nix run .#verify` would start immediately), but the strict
   watcher policy wants sum < 15.5. After the one-line ADR fix: **direct run now
   (my recommendation — the window was already paid for and load is falling), or
   re-arm the strict 12h watcher?**
2. **Live sibling sessions in cqrs-htmx (21:44) and crm (21:38, dirty)**: M11/M12/M14
   edit those repos. Take them over now (risk colliding whatever those sessions are
   still doing), or wait for them to settle and only then execute? If waiting — is
   there a signal you want me to use (e.g. you tell me when they're done)?
3. **Untagged-backlog wave (~20 modules, evidence in TODO_LIST)**: cut it as its own
   wave immediately after M07 (my recommendation — it is pure mechanical version
   derivation + the same four-invocation pattern), or fold it into the next planned
   feature wave?

— Receipt trail: watcher `build/verify-launch-14.out` (window 23:15:34, iter 585);
verify log `build/verify-run14.log` (25 lines, fast-fail); rc file
`build/verify-run14.rc` = 1; playbook `build/wave-plan-2026-11-05.md`; TODO_LIST
"Post-sweep plan execution" section updated this session.
