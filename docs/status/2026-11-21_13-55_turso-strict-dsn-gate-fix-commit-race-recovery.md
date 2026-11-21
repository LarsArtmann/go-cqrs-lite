# Status Report — Turso Strict DSN, the changelog-symbols Gate Fix, and the Commit-Race Recovery

**Date:** 2026-11-21 13:55 CET (session spanned ~02:20–02:55 and ~08:40–09:00 activity windows; report cut at 13:55)
**Session scope:** Resumed the 2026-11-15 W3 summary under the standing NOW-GET-SHIT-DONE order. Six days had passed; the session re-oriented, shipped one commissioned code unit + one gate fix, closed two doc/evidence gaps, recovered the repo from a commit race, and arbitrated lanes against a 38-session fleet.
**Scope line:** This report covers ONLY what this session did and directly noticed. No fresh research beyond that.

---

## a) FULLY DONE (verified this session)

| Item | Evidence |
| --- | --- |
| **Turso strict-vs-lenient DSN policy** (RULED 2026-10-08, blanket-authorized, TODO row :1486-class): `strictDSNParams` runs in `New()` on the caller-owned DSN (embedded AND remote); unknown `*encrypt*`/`*key*` params rejected via `errorfamily.WrapRejectionf` around sentinel `tursoengine.ErrUnknownEncryptionParam`, naming the param + close-match suggestion (`did you mean "encryption_hexkey"?`) + `WithEncryption` remedy; known set {experimental, encryption_cipher, encryption_hexkey} case-insensitive; non-encryption unknowns (`mode=memory`, `authtoken`) untouched | `metaengine/tursoengine/v4/dsn.go` + `dsn_strict_test.go` (13-case table, New-level rejection, suggestion/distance tables); full tursoengine suite green 122s; api golden regen (+1 export, 6514); committed by the fleet as `d8ffb1a3e` "feat(tursoengine): fail-loud strict DSN check..." |
| **check-changelog-symbols gate bug fixed at root cause**: tier-1 golden regex could not match `/v4`-nested module symbols (kind sits after the version segment — latent since the 2026-11-10 module move, masked for funcs by the source fallback), and the fallback matched only func/type — so the first-ever nested-module VAR citation (mine) false-positived as FICTION | `scripts/check-changelog-symbols.sh`: tier 1 now `/${alias}(/v[0-9]+)?/(kind)` + fallback gains `^var`/`^const` shapes. Red observed pre-fix, green post-fix (36→37 citations honest); landed inside fleet commit `a71fb361a` ("changelog-symbol const/var tier") via daemon absorption |
| **CHANGELOG + TODO row closure for the Turso unit**: [Unreleased] Changed row citing the sentinel (changelog-symbols gate green on it); TODO row flipped RESOLVED 2026-11-21 with full receipt | `CHANGELOG.md`; `TODO_LIST.md` :1486 (row landed with the fleet harvest commits) |
| **FEATURES.md core/v5 section** (the 01-28 brutal-review's missed-ritual gap): 7-row table — StreamRef (ADR-0166), fused Payload, branded core types, Cause-at-birth, structural Tracing.Actor, honest bridge stamps, deprecated-twins ⚠️ — every cited symbol verified against source first (`id.NewStreamRef/OrZero/ParseStreamRef`, `record.PayloadFrom`, `EncodingEnvelope`, `event.Cause()/WithCause/WithCausation`, `metadata.RecordActor`); link to V5-MIGRATION-GUIDE §2b | `FEATURES.md` (+23 lines, dprint-clean: `nix fmt` 0 changed) |
| **Standalone-build resolution receipt**: GOWORK=off `go build ./...` per module over every go.mod (testdata excluded) — **101/101 green** under load. Closes the 01-28 verify-ci gap's resolution half (the tidy sweep's cold-cache breakage class); the test half is proven by M20·d's green `#verify` (2026-11-20) | build sweep log in session; module count reconciles (104 = 101 + 3 testdata) |
| **Re-orientation of the 6-day-stale summary**: discovered the 2026-11-15→11-21 session gap (W3 chain #6 logs trashed by the 02:27 build-cleanup session; v5-core plan executed; CI heal round 2; fleet-repair plan complete; D27–D39 decisions minted); verified every c-list §f item's disposition before acting | `docs/status/` index walk; newest reports read in full |
| **Lane arbitration under a 38-session fleet**: detected live sibling ownership by mtime (go-cqrs-lite README sweep 07:45–07:59; precheck `login_pages.go` edited 1 min before my survey; browser-history §f-3/4 landed as `ed87c29` while I surveyed) and pivoted without touching in-flight work | mtime checks logged in session |
| **F214 CI watch (partial, cite-verified)**: Gates Rollup `44873971680` = failure at the aggregate job; Autoretry ×2 green. Legs NOT enumerated — API rate-limited (fleet exhausted the quota) | `gh run list`/`run view` outputs in session |

## b) PARTIALLY DONE

| Item | State |
| --- | --- |
| **Hand-commit of the gate fix** | Attempted at the 08:45 phase boundary per the 2026-11-03 ruling; raced (see d-1); recovered; the fix then landed via daemon absorption inside a fleet commit. Nothing lost — but my authored commit never existed and the commit raced catastrophically before the daemon path won |
| **verify-ci gap** | Build half DONE (101/101 above). The full `#verify-ci` flake app (build+TEST per module) never ran — load-gated all session (30–1124 observed); in-workspace `#verify` green (M20) + CI-on-push are the covering evidence |
| **F214 origin-green receipt** | Aggregate-red verified; which legs are red is unknown (rate limit). Gates Rollup is a pure aggregator that self-heals with owners, but nobody has enumerated this run |

## c) NOT STARTED (this session)

- `check-coverage --update` (load-gated; would pin the fleet's new tests + the Turso suite + the 3 STALE improvements carried since 11-15).
- Full `#verify-ci` app run (load-gated).
- Quiet-window metaengine/v4 suite re-run — MOOT: adjudicated green by M20·d's full `#verify` (300 test packages) — no longer rowed by me.
- The Turso upstream T25 filings pack (approved, blanket) — untouched; not started because fleet lanes + load consumed the session; claims need re-verification against latest upstream main first (verify-before-filing).

## d) TOTALLY FUCKED UP (and fixed, honestly)

1. **THE COMMIT RACE (worst incident of the session).** My path-limited `git commit scripts/check-changelog-symbols.sh` under the 38-session fleet produced commit `83e0a9b7c` recording **7004 files changed, 1 insertion, 1,482,816 deletions** — a gutted tree holding only my one file. Mechanism (reconstructed): a parallel session staged mass adds concurrently while TWO pre-commit hook systems ran (BuildFlow pre-commit FAILED — shfmt `context deadline exceeded`, nix roundtrip temp-repo error — and `.githooks/pre-commit` passed), and the commit landed anyway with a gutted tree. Working tree was never touched. **Recovery:** `git reset --mixed 739c20df1` (allowed mode), orphaned the gutted commit, `git fsck` then flagged unreadable commit-graph objects → trashed the commit-graph cache (safe, regenerable), final fsck rc=0 clean. My fault chain: (a) I hand-committed knowing the daemon convention and knowing a parallel session was mid-phase — the phase-boundary ruling permits authored commits, but nothing about a 38-session storm; (b) I did not verify the commit stat in the SAME command (the check came one command later — fast enough to recover, late enough to have landed).
2. **Started from a 6-day-stale summary.** The first todo ("check owner instructions + current state") executed, but I read the summary's world (chain #6 riding, W3 freshest) harder than the filesystem's (`date`, newest report). It took the chain-log disappearance + an uptime/date contradiction (14:26 vs 02:36 across resume gaps) to force full re-orientation. A `date` + newest-`docs/status/` read FIRST would have saved the detour.
3. **Near-miss lane collision on precheck.** I surveyed browser-history-precheck to take c-list §f-1/2 before checking live ownership; the mtime check (login_pages.go modified 1 minute prior) caught the active sibling just in time. Correct save, wrong order: ownership check should precede lane selection, not follow it.
4. **Incomplete gate battery on the Turso unit.** I ran changelog-symbols, todo-hygiene, file-size, untested-exports — but skipped check-error-taxonomy and check-package-surface in the moment. Both later verified CLEAN for my code (the taxonomy gate's only reds are foreign `pebble_preset.checkpoint`/`flush` rows from the M-wave's in-flight work), but skipping them was process debt, not diligence.
5. (Minor) `xargs` blocked by permissions (yolo mode); re-planned with `while read` — no damage, just a round trip.

## e) WHAT WE SHOULD IMPROVE

1. **No path-limited hand-commits while other sessions are active.** The daemon exists precisely for this; the phase-boundary authored-commit ruling needs a fleet-activity carve-out (or a serialized lock). One gutted-tree commit cost a reset + cache repair; next time the worktree might not be so lucky.
2. **Commit self-verification is part of the commit.** `git show --stat HEAD | head -3` belongs in the SAME command chain; a gutted commit discovered one command late is one command too late.
3. **Session-start ritual: `date`, then newest status report, THEN any summary.** Summaries lie by aging; the filesystem doesn't.
4. **Lane-claim = mtime check on the target repo BEFORE adopting a report's §f item.** This fleet executes fast (browser-history's lane closed in the ten minutes I spent reading).
5. **The double pre-commit hook stack needs one investigation.** BuildFlow pre-commit and `.githooks/pre-commit` both ran on my commit; one failed (load-induced deadlines), one passed, and the commit landed through the failure. Under which conditions does a failed hook still permit a commit — hook wiring, exit-code plumbing, or the second hook masking the first? Un-investigated; rowed below.
6. **Gate batteries should be a named checklist, not recall.** The repo's Verify list names error-taxonomy and package-surface; my recall under load named four of six.
7. **`git fsck` health is invisible until someone runs it.** The commit-graph cache had no self-heal; a corrupted cache only surfaces as parse errors. Cheap fsck spot-check belongs in the post-incident ritual (it did save this session — keep it).

## f) NEXT — up to 50, honest (not padded; 20 genuine items from this session's run and notices)

| # | Next thing | Why | Effort |
| --- | --- | --- | --- |
| 1 | Enumerate the red legs of Gates Rollup `44873971680` once the API quota resets; heal at owners | The aggregate is red and nobody has looked inside | S |
| 2 | F214 first-green-master-CI watch (pushes flowing again post-D33; daemon pushes master) | The origin-green receipt is the last F209–F213 open item | S, observe |
| 3 | `check-error-taxonomy`: add `pebble_preset.checkpoint`/`flush` (Infrastructure) rows to docs/error-taxonomy.md — or confirm the M-wave owns it and lands them | Only reds on that gate right now | S |
| 4 | Investigate the double pre-commit hook stack (BuildFlow + .githooks; failed-hook commit landed) | d-1's un-investigated half | M |
| 5 | Codify the no-hand-commit-under-fleet rule + commit-self-verification step (AGENTS.md gotcha or docs/agents/gotchas-tooling-build.md) | Turns d-1/e-1/e-2 into durable policy | S |
| 6 | Row the commit-graph-cache corruption + trash-repair as an ops note (gotchas-tooling-build.md) | Nothing records that the cache was trashed 2026-11-21 | S |
| 7 | `check-coverage --update` in a quiet window (pins fleet tests + Turso suite + 3 STALE improvements from 11-15) | Load-gated since two waves | M, load-gated |
| 8 | Full `#verify-ci` app run in an exclusive window (build+test per module) | The test-matrix half of the 01-28 gap; CI-on-push partially covers | M, load-gated |
| 9 | Prune the 1 STALE untested-exports row (`cmd/cqrs-lint/v4 HealthConfig` gained a test) | Shrink-only ratchet, 30 seconds | S |
| 10 | Turso upstream T25 filings pack (approved): re-verify (a) missing DriverContext/OpenConnector, (b) remote-BYOK unrepresentable, (c) mistyped-DSN-silently-ignored against latest turso-go main, then file per verify-before-filing | Approved blanket; my strict-DSN fix is the in-repo half of (c) | M |
| 11 | D34 confirmation: record the owner's "T fine" reading (no T9 wire-shape port; master stands) in the 07-27 report's §g resolution | Owner answer exists; the row still says "confirmation pending" | S |
| 12 | Owner-gated queue: D35 (v4.16 wave blessing), D37 (go.sum policy), D38 (gls go.work), D39 (SEC bump gate), Q1 (verify strategy, 6th session), D31 (DiscordSync push) | Not mine to execute | gated |
| 13 | dependabot owner pings: alerts 138/139 (otel trace/sdk, 8+ days stale) | Owner-protocol row | S, owner |
| 14 | Observe the Sunday 2026-11-22 nightly log (first clean load-sweep post-+x-fix) | Carried row; tomorrow | S, observe |
| 15 | Verify the README ADR-link sweep + claiming/v4 changes landed green once the M-wave commits (lint-scope on their modules; `check-readme-links.sh`) | M-wave was mid-flight at session end | S |
| 16 | tursoengine docs: consider a FAQ/skill line for the strict-DSN behavior (mistyped params now fail loud) | Consumer-visible behavior change beyond the CHANGELOG | S |
| 17 | Row the build-sweep receipt (101/101 standalone, 2026-11-21) wherever the 01-28 verify-ci item is tracked (report §f-6 is now substantively closed) | Receipts without homes get lost | S |
| 18 | Session-start ritual (`date` + newest report first) — add to the crush-config references/lessons or repo gotcha if it generalizes | e-3 | S |
| 19 | Nightly-bench quiet-window verification vs baseline (deferred all session by load) | Load-gated routine | M, load-gated |
| 20 | If quiet windows stay scarce under fleet load: run the eight gate apps individually with recorded receipts (C5/Q1 strategy) instead of waiting for `#verify` exclusivity | Q1's practical fallback | M |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **D34 reading confirmation:** your answer "T fine" to the T9 port question — I read it (with the fleet) as *master's corruption-only `verify_tenant_payloads` stands, NO port of precheck's wire-shape tripwire, scaffolding preserved on `origin/precheck`*. Correct? (The alternative — "port is fine [wanted]" — inverts the decision.)
2. **Hand-commits under a multi-session fleet:** after today's gutted-tree race, should the 2026-11-03 phase-boundary authored-commit ruling gain a carve-out (daemon-only while another session has touched the repo within N minutes), or should hand-commits be serialized through a lock (the `verify-lock.sh` pattern)? Owner taste; it changes how every fleet session closes a wave.
3. **Q1 (carried, 6th session):** under sustained fleet load (30–1124 observed today), is "run the eight gate apps individually with recorded receipts; treat composed `#verify`/`#verify-serial` as eventual confirmation" the standing verification strategy? Today's work rode exactly that fallback (scoped suites + gate apps + a build sweep), and item f-7/f-8 stay blocked without a ruling either way.

---

**Per harness contract:** no manual commit of this report — the auto-commit daemon absorbs it.

**WAITING FOR INSTRUCTIONS.**
