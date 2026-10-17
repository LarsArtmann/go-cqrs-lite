# Disk-Truth Plan Execution — Wave I Delivery Status (2026-10-17 23:18)

> Session scope: execute [`docs/planning/2026-10-17_20-13_SUPERB-disk-truth-pareto-plan.md`](../planning/2026-10-17_20-13_SUPERB-disk-truth-pareto-plan.md) (Table A: A1–A24). Session ran ~20:30–23:18 CEST and delivered the **W-wave (the 1%→51% delivery chain)**: A1 fully green, A2 tagged and proxy-verified, A4 two-thirds green with one triage open, A3 prepared but not executed (gated on G2). P/T/X waves (A5–A24) untouched.

---

## a) FULLY DONE

1. **A1 — changed-set suites green (joined trust-green W1).** `GOWORK=off go test -count=1 ./...` PASS for all 9 changed-set modules: event, command, system, storage, commandlifecycle, projectionhost, schema, core/v5, example/goal-shaped-app. Extended verification window later in the session: signing, encryption, kv, storage/pebble, record, middleware — all PASS. No reds to triage inside the changed set (taskmanager excluded — known pre-existing breakage, owned by A17).
2. **Pre-wave gate repair: go-directive lockstep sweep.** Found `check-go-version.sh` RED pre-existing (go.work at `go 1.27.2`, 95 go.mod files at `go 1.27` — go-codec v0.3.2 aftermath). Raised all 95 tracked go.mod directives to `go 1.27.2` (toolchain go1.27.2 confirmed effective). Gate PASS. This was a prerequisite for shipping a clean wave.
3. **Wave audit (T02-style).** versions.json-driven diff of every module vs its latest tag: **31 modules carry unreleased production changes** (~5 days of parallel-session work with only catalog tagged 10-16). Scoped this wave to the plan's delivery chain + forced co-releases rather than a 31-module mega-train (the 2026-10-11 trust-green plan owns the broader waves).
4. **A2 — disk-truth wave I tagged, pushed, proxy-verified: 19 tags** (all proxy-served attempt 1, all smoke-green):
   - **10 wave tags (20:47–20:53):** event/v4.15.0, command/v4.14.0, signing/v4.4.1, encryption/v4.4.5, kv/v4.3.5, metaengine/projectionadapter/v4.6.0, storage/v4.11.0, commandlifecycle/v4.3.0, projectionhost/v4.7.0, system/v4.15.0.
   - **9 pin-settle patch tags (21:01–21:18):** event/v4.15.1, command/v4.14.1, signing/v4.4.2, encryption/v4.4.6, metaengine/projectionadapter/v4.6.1, storage/v4.11.1, commandlifecycle/v4.3.1, projectionhost/v4.7.1, system/v4.15.1. (kv needed none.)
   - **Delivered content:** stream IDs persist BARE on every write path (system WAL envelopes, event/command AsRecord, commandlifecycle recorder, projectionhost DLQ, storage aggregate column + keyset cursor, projectionadapter `EventWithID.ID`); go-codec v0.3.2 train (event/signing/encryption/kv; storage/pebble blocked by ADR-0152 untag policy — documented); projectionadapter decode-failure → Corruption classification (DLQ-able); fleet go-directive floor 1.27.2.
   - **G1 GREEN, two probes:** (1) `go get system/v4@v4.15.0` scratch consumer compiled+ran (revealed the stale-minimum defect, see §d); (2) after settles, a LAZY consumer (`go get system/v4@v4.15.1` only) resolves event/v4@v4.15.1 + command/v4@v4.14.1 through minimums alone — the delivery bar.
   - **Post-wave gates:** `check-versions-manifest --update` + verify GREEN; `pin-sweep --check --remote` GREEN (46+44+10 modules swept across three incremental passes); `check-changelog-symbols` GREEN (19 citations verified).
5. **CHANGELOG cut + TODO_LIST row closure.** New wave section `## [event/v4.15.0, … system/v4.15.0 — 2026-10-17 disk-truth wave I: stream IDs persist bare (10 tags)]` with the three moved entries (stream-IDs-bare, projectionadapter EventWithID, go-codec v0.3.2) + settle-tag paragraph. TODO_LIST release row (line 613) corrected (its "schema v4.6.0" numbers were stale — that wave shipped 10-10..10-13) and closed with the full green matrix + rationale.
6. **A4 partial — two legs green:** `#integration-redis` rc=0 (watermill broker suite incl. roundtrip, Nack redelivery, group exactly-once, 2 MiB payloads); inside `#test-integration`: the sqlite/pebble/bbolt/duckdb legs and the ephemeral-PG leg ran with **zero non-MySQL failures** (all FAIL lines in the log are the 3 MySQL cases below; the ERROR-level log lines are tests exercising failure paths by design).
7. **A3 prep (no mutations):** consumers located (cqrs-htmx 28-module workspace, go-appkit multi-module); per-module pins enumerated; cqrs-htmx AGENTS.md + `bump-dep.sh` runbook + `dependency-train-bump.md` playbook read; go-ecosystem-upgrade skill loaded (baseline-first, go.work rename discipline, absence assertion, family-release-consumable preflight, vet-is-load-bearing).

## b) PARTIALLY DONE

1. **A4 — integration battery: 1 open triage.** `#test-integration` rc=1 caused by exactly 3 failures in `metaengine/mysqlengine` (an ADR-0152 untagged train): `TestStreamLogBackend_MySQLRoundtrip`, `TestMySQLGraph_UndirectedAfterRemoval`, `TestMySQLEngine_TxIsolationFromForeignContext` — all "mysqlengine.init: invalid connection" class, all ~15–16s (connect-timeout signature, NOT data-mismatch). Evidence so far: MySQL strategy = QEMU VM; the VM booted MariaDB (systemd "Starting MySQL Server" in log); every OTHER module's MySQL tests correctly SKIP without `MYSQL_TEST_DSN`; the 3 failures are in mysqlengine's own suite which constructs engines against the DSN. Root cause NOT yet pinned: VM port-forward race, DSN not reaching that module's test invocation, or (least likely — wrong failure shape) a real wave regression. Next probe: `nix run .#test-integration -- --mysql-only` rerun and/or direct `#integration-mysql-vm` leg. **G2 is therefore still OPEN; A3 remains gated.**
2. **CHANGELOG heading drift (cosmetic, immutable where it hurts):** the TAGGED commits' CHANGELOG says "(10 tags)"; the settle-tag paragraph exists only in post-tag commits. The wave heading also still names the superseded `.0` versions rather than the settled `.1` floors. Future readers via git history see the intermediate state.
3. **Wave verification depth:** per-module suites + pin-sweep's tidy/build/test-compile + proxy smokes are green, but the composed `nix run .#verify` has NOT run this session (deliberately deferred past the battery for load-exclusivity reasons — it remains a debt, and lint (`#lint`) has not run post-sweep either).

## c) NOT STARTED

- **A3** — consumer bumps (cqrs-htmx then go-appkit, ADR-0152 order), their suites, and the B10 systemscenario "no prefixed write reaches any store" pin (W-DONE criterion).
- **P-wave:** A5 (delete `inferColumnType`), A6/A7 (raw-bytes pins + goal-story pin), A8/A9 (keycodec sep/esc + legacy KV dual-read), A10 (cqrs-lint branded-`.String()` rule).
- **T-wave:** A11–A14 (typed WAL columns sqlite/pg/mysql/duckdb + adttest conformance), A15 (layout-stamp default-on), A16 (docs truth pack), A17 (taskmanager test repair).
- **X-tail:** A18–A24 (timers gcdc envelope, UnixNano discipline, wire tags, recipes upcasting doc, cursor/sweep audit, legacy fixture + benchkit key, docs-health harvest).
- Standing owner questions (tag timing — now largely resolved by the W-wave shipping; deployed-DB prefixed-row census; taskmanager ownership).

## d) TOTALLY FUCKED UP (and what it cost)

1. **Cut 10 tags WITHOUT running pin-sweep first — the wave's one real self-inflicted wound.** `batch-release.sh`'s `go mod tidy -e` PRESERVES existing requires (tidy never upgrades), so every dependent tag shipped with stale sibling minimums: system/v4.15.0 pinned event v4.14.0, command v4.13.1, projectionadapter v4.5.3 — a lazy `go get system@v4.15.0` would have built the OLD prefixed-writing code. Caught by the G1 scratch-consumer probe (the plan's own gate — it worked). **Cost:** 9 permanent patch versions in the module proxy (v4.15.0/v4.14.0/etc. are immutable and forever substandard-but-served), ~40 minutes of remedy rounds, and a CHANGELOG heading that undercounts. **Root cause:** I read the same-wave pin convention (10-11 wave: "all workspace pins settled on the six tags") in the CHANGELOG but did not connect it to MY cut sequence until the probe failed it. The knowledge was in `pin-sweep.sh`'s own header the whole time.
2. **R3 cut attempted on a dirty tree.** The daemon hadn't absorbed pin-sweep #3's 49 files; batch-release refused (correctly), but my chained `&&` then pushed nonexistent tags → confusing `src refspec does not match any` errors that took a diagnostic round-trip to unwind. My clean-tree wait-loop had raced the daemon.
3. **Battery v1 double-ran the legs.** I chained `#test-integration` + the four per-engine apps without checking that `#test-integration` already covers sqlite/pebble/bbolt/duckdb/pg/mysql/dgraph — killed and restarted as v2 (test-integration + redis). Wasted ~5 minutes, zero damage.
4. **Minor artifacts:** left an empty `### Changed` header in `[Unreleased]` during the CHANGELOG surgery (cosmetic); `/tmp` probe litter (g1probe, g1probe2, battery logs — intentional evidence, harmless).

## e) WHAT WE SHOULD IMPROVE (process lessons this session)

1. **Encode the pin-settle step INTO the release tooling** — `batch-release.sh` should refuse (or warn) when a dependent's go.mod pins a sibling below a tag cut IN THE SAME INVOCATION/SESSION, or auto-run pin-sweep per round. Today the rule lives only in prose + the 10-11 wave's changelog. (Candidate: a `--settle` mode that sweeps before each cut.)
2. **G1's scratch-consumer probe should be a scripted gate** (`tag-release.sh --smoke` proves proxy service, NOT pin quality). The lazy-`go get`-only probe is what caught defect #1; it took me writing it by hand mid-wave.
3. **Clean-tree gating belongs inside my cut/push one-liners**, not in a sleep-loop that can lose the race with the auto-commit daemon (the daemon commits in bursts; `git status` can read 0 and be wrong seconds later).
4. **Read the flake app bodies before composing battery chains** — app names are ambiguous ("test-integration" sounds local-only but runs 7 engines incl. a QEMU VM).
5. **The wave-audit step (31 modules unreleased) deserves a standing script** — I ad-hoc'd it in Python against versions.json; it's exactly the "consumers of untagged API define a wave" T02 rule and would catch scope errors every future wave.
6. **CHANGELOG wave headings should be written AFTER the cut settles** (or a follow-up "settled floors" line added pre-push) so tagged history doesn't undercount tags.
7. **mysqlengine's suite should skip-with-reason like every other module's when the DSN is absent** — the asymmetry (3 tests construct-and-fail instead of skipping) is what makes the battery's rc unreadable. Also worth a `--mysql-only` fast path in triage.

## f) NEXT — up to 50 things, in order

**Close G2 (A4):**
1. Rerun `nix run .#test-integration -- --mysql-only` — confirm the 3 mysqlengine failures reproduce in isolation.
2. Inspect `scripts/test-integration.sh` MySQL env plumbing: is `MYSQL_TEST_DSN` exported into the mysqlengine module's `go test` invocation?
3. Run the dedicated `nix run .#integration-mysql-vm` leg (QEMU, ~131s) as the authoritative MySQL verdict.
4. If connectivity-class → declare environmental, record matrix; if data-class → fix-forward (G2 policy: never revert the wave; no released tag carries bare writes).
5. Record the full integration matrix (sqlite/pebble/bbolt/duckdb/pg/mysql/dgraph/redis) in CHANGELOG under the wave section (B15).

**A3 — consumer delivery:**
6. cqrs-htmx baseline: `nix run .#test` (record pre-existing reds).
7. cqrs-htmx bumps via `scripts/tools/bump-dep.sh` per train: event v4.15.1, command v4.14.1, signing v4.4.2, encryption v4.4.6, kv v4.3.5, storage v4.11.1, commandlifecycle v4.3.1, projectionhost v4.7.1, projectionadapter v4.6.1, system v4.15.1.
8. Absence assertion: `grep -rE 'go-cqrs-lite/(event|command|…)/v4 v4\.(14\.0|13\.1|…)' --include=go.mod .` prints nothing.
9. cqrs-htmx full suite + `nix run .#check-cqrs-lint` + release-train gate `--strict-lag 0 --refresh-cache`.
10. B10: add the systemscenario pin — boot via `systemadapter`/`Adopt`, write an event+command, open the sqlite DB, assert NO `StreamMarker:` substring in any stored row (W-DONE criterion).
11. go-appkit: baseline, bump `cqrs/` + `integration/` pins, suite, same absence assertion.
12. Both repos: confirm the auto-commit daemon absorbed; their CHANGELOGs/TODOs rowed if they keep them.

**Wave hygiene debts:**
13. Run `nix run .#verify` (or verify-fast) on the settled tree — the composed gate has not run since before the sweep.
14. Run `nix run .#lint` post-sweep (pin-sweep only build+vet+test-compiles).
15. Check GitHub Actions on the pushed tags/master (`gh run list`) — triage any CI red on the wave commits.
16. Fix the empty `### Changed` header in `[Unreleased]`.
17. Consider a one-line CHANGELOG correction noting the settled floors are the `.1` series (post-tag commit).

**P-wave (4%→64%):**
18. A5/B16: delete `metaengine/layout.go:169` `inferColumnType` + word lists.
19. A5/B17: `sqlTypeOf` → `(string, error)`; unknown → TEXT + loud warning.
20. A5/B18: repin layout goldens (price/count/age cases → TEXT).
21. A5/B19: CHANGELOG row + `cd cmd/api-stability && GOWORK=off go run . --update` if surface moves.
22. A6/B20-22: raw-bytes WAL pins — write event via sqlite backend, read the raw row; assert bare id + raw payload bytes + JSON metadata + UTC/nano time; command envelope twin.
23. A6/B23: prove the pin red on the pre-fix commit (checkout worktree at pre-fix SHA in a temp worktree — never `git checkout` the tree).
24. A7/B24: goal-shaped-app story asserts `TaskView.ID == doneID.Get()`.
25. A7/B25: grep goal.db for prefixed ids; regenerate if local artifact exists.
26. A8/B26: keycodec escape/unescape + unit table (sep `\x00`, esc).
27. A8/B27: v2 family prefixes wired into all key builders.
28. A8/B28: rapid round-trip law over adversarial corpus (`:`, `\x00`, 255+ bytes).
29. A8/B29: badger/bbolt-meta/pebble-meta suites green.
30. A9/B30: storage/pebble escaped builder + legacy dual-read.
31. A9/B31: storage/bbolt same + `cqrs_journal_idx` rekey.
32. A9/B32: migration notes in both READMEs (G3: dual-read window declared).
33. A10/B33-36: cqrs-lint branded-`.String()` rule — matcher, slog/attribute whitelist, 3-good/3-bad fixtures, warn severity, waiver path, ratchet doc.

**T-wave (20%→80%):**
34. A11/B37-41: sqliteengine typed WAL (guarded ALTER ADD, json_extract backfill, UNIQUE(event_id) + UNIQUE(collection,stream_id,version), writer fills typed columns, reader prefers typed w/ JSON fallback).
35. A12/B42-45: pgengine (JSONB metadata) + mysqlengine (VARCHAR(768) ids, online-DDL indexes, NO table recreate) typed WAL.
36. A13/B46-47: duckdbengine ADD COLUMN path + engine-level unique check + documented constraint story.
37. A14/B48-49: adttest duplicate-append + (stream,version) conflict cases ×4 engines.
38. A15/B50-52: layout-stamp default-ON (warn-first per G4) + column-add/rename drift conformance per engine.
39. A16/B53-57: docs truth pack — gotchas sweep-complete block, cursor-pairing invariant, four-layer compat checklist, raw-bytes-pin convention, CHANGELOG four-layer wording.
40. A17/B58-60: taskmanager `servertiming` import fix, `idStreamID`/`parseIDString` helpers, dep prune, suite green.

**X-tail:**
41. A18/B61-62: timers payload → ADR-0044 gcdc envelope + old→new migration test.
42. A19/B63-64: typed-WAL UnixNano writers + cross-offset ordering test + UTC rule doc.
43. A20/B65: `Causation`/`TombstoneMark` json tags + golden-bytes pin.
44. A21: recipes.md §schema — legacy payload-ID strip via `Chain.Upcasters()` (doc only).
45. A22/B66-67: cursor/sweep audit (pebble/bbolt/watermill keyset readers vs stored form; `StreamRef.Split()` consumer grep).
46. A23/B68-69: legacy prefixed-row golden fixture + benchkit `countKey` → `Get()`.
47. A24/B70: docs-health HARVEST (incl. `ParseStreamID` strictness owner question, go.sum tidy-or-confirm-owner per E4).
48. Bank-sync pilot patch (TODO_LIST row's pending tail — needs owner ruling, see Q3).
49. File the mysqlengine skip-asymmetry improvement upstream in TODO_LIST.
50. Encode the wave-audit + pin-settle-checklist into a `scripts/check-wave-readiness.sh` candidate row.

## g) QUESTIONS FOR THE OWNER (cannot be determined from this machine)

1. **Deployed-DB census (blocking nothing yet, blocking the P-wave's dual-read design):** do any *deployed* fleet databases (pbx-lens, DiscordSync, InboxClean, CV, crush-daily, …) currently hold persisted `StreamMarker:`-prefixed rows written by pre-wave v4 tags? The four-layer compat model says envelope-keyed views heal on rebuild and decode-tolerates everything, but A9's legacy-KV dual-read window and any explicit migration note depend on whether real prefixed bytes exist outside test fixtures. I can enumerate code paths, not production contents.
2. **example/taskmanager ownership (A17):** its tests are broken from a parallel session (undefined `servertiming`, `idStreamID`, `parseIDString`, `systemRegisterQuery` — 7 compile errors). Is that session finished/abandoned, or actively working? Fixing it is T-wave work for me, but I don't want to collide with a live author mid-edit.
3. **Bank-sync pilot patch scope:** the (now-closed) release row's tail — "apply the bank-sync snapshot patch (M13) + drop DiscordSync's `map[any]any` workaround + collapse cqrs-htmx's homegrown UpcasterRegistry onto the chain" — is that IN this plan's X-tail for me, or does it stay with the 2026-10-09 schema-evolution plan's owner? (It touches another repo's consumer code and was explicitly gated on "the next schema tag wave" — which I just shipped.)

---

*Evidence anchors: tag timestamps above (git for-each-ref); battery log `/tmp/battery.log` (test-integration rc=1: 3 mysqlengine cases, redis rc=0); release logs `build/release-logs/batch-20261017-*.log`; G1 probes `/tmp/g1probe{,2}`; sweep logs `/tmp/ps{,2..6}.log`.*
