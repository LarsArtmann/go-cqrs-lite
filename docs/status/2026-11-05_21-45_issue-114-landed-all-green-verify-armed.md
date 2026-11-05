# Status Report — 2026-11-05 21:45 CET: #114 Landed, All Tree Work Green, Verify Watcher Armed Overnight

**Session:** resumed 19:14 from the 19:08 park under the standing "GET SHIT DONE — the WHOLE TODO LIST, all the time in the world, do not stop" directive. The directive answered the parked questions by implication (don't park; overnight runs sanctioned; restructure rather than wait). This report covers 19:14–21:45 CET.

**The strategic restructure:** the quiet window never opened (load sums 91–1919 all evening, 10+ concurrent agent sessions). Instead of parking again, the session killed the 3h-bound watcher (launch-13, would have expired 21:15 rc=12) and MERGED #114 into the release window: land ALL tree work now, then ONE overnight verify covering everything, then push/tag. Both planned verifies collapse into one; all evidence-derived wave versions stay valid (#114's new exports are minor-class, and metaengine was already minor-bound at v4.21.0 by the `BuildLayoutPlan*` exports).

---

## a) FULLY DONE THIS SESSION (each verified by running the command)

1. **M16–M19: per-group projection caps (#114, ADR-0164) landed complete** — only the GitHub close awaits the wave tag:
   - `metaengine/cap_grouped.go`: `WithCapPerGroup` config + accessors, `validateCapGroupedConfig`, `EnforceCapGrouped` (standalone), `enforceGroupedTrim` (never locks — callers hold the fold lock), `CapGroupTrimmer` capability interface, `ErrCapGroupPushdownUnsupported`, pushdown-first with portable Go fallback (one orderBy-DESC scan, per-group counters, `capDelete` per row).
   - `metaengine/sqliteengine/cap_grouped.go`: `TrimGroupedCap` — one `DELETE … WHERE rowid IN (SELECT … ROW_NUMBER() OVER (PARTITION BY … ORDER BY … DESC) …) RETURNING key`; meta_map `json_extract` path + planned-table real-column path; cell-versioning refusal (temporal history must not be bulk-deleted); interface pin in engine.go.
   - `metaengine/adttest/grouped_cap.go`: `AssertGroupedCapContract` (variadic query options steer layout: `WithColumnarLayout` exercises the real-column path) + `AssertPlainCapContract`.
   - Tests GREEN: metaengine (9 tests: memory fallback, standalone, degenerates, EnforceCaps composition, validation rejections, json-tag case-insensitive match, accessor roundtrip), sqliteengine (meta_map pushdown + planned columns + versioned Go fallback + plain-cap map-items + refusals), tursoengine (delegation).
   - `metaengine/query_config.go` + `cap.go` edits (grouped pre-pass composes into `EnforceCap`; `EnforceCaps` reaches grouped via composition).
   - ADR-0164 + index row (`check-adr-numbering` green before AND after).
   - Engine-capabilities census: NEW "Grouped-cap trim" column + delegation inheritance (turso via sqliteengine) + a new growth-detection self-test leg; `docs/engine-capabilities.md` regenerated; gate green.
   - recipes.md §2.50 (compiled fence + catalog entry; `TestRecipes` green, 64 s).
   - CHANGELOG (Added row + Fixed row; 46 pkg.Symbol citations verified) + FEATURES "Bounded projections" row extended.
   - api-stability golden 6,449; untested-exports 1,083→1,081 (two stale rows GAINED tests, pruned); package-surface re-pinned metaengine 835/168 (after unexporting `fieldValueAt`).
2. **TWO LATENT BUGS found and fixed en route** (see §d for the honesty): (a) a grouped declaration WITHOUT its WithCap base silently skipped ALL validation (`validateCapConfig` returned early before the grouped tail-call — fixed by running grouped validation first); (b) `EnforceCap` PANICKED on every SQL engine since #72 shipped (key extractor `reflect.Value.Field` over `map[string]any` scan items — memory was the only engine that ever exercised it; fixed via one shared `fieldValueAt` helper reading BOTH item renderings, pinned forever by `AssertPlainCapContract` on sqlite). Additionally `buildKeyExtractor` gained a group-column exclusion that resolves type-ambiguous keys (string ID + string Owner — exactly the #114 shape) for grouped declarations.
3. **M24**: `baseline_rematch.go` warn de-explosion (string-concat form, strconv loc builder); `waiverAdoptionRules` hand-map policy pinned by a catalog-side drift-guard test (generation from the catalog is impossible — adoption rules carry no module key; documented in the test).
4. **M26**: `metaengine/sqliteengine/pragma_override_test.go` — caller-pragmas-win (cache_size=-8192, busy_timeout=1234) AND defaults-pinned golden tests, green.
5. **M21**: `scripts/lint-scope.sh` + `nix run .#lint-scope` (GT_LINT_MODULES changed-set gate, same ✅/❌ summary as #lint, git-toplevel root resolution, 3-leg self-test green) + AGENTS quick-ref row. DOGFOODED immediately: caught 4 real golangci findings (3 in my new code + 1 latent from the interrupted verifies in `sse_ordering_test.go`) — all fixed, scoped lint now 5/5 clean.
6. **M23**: README "Optional engine interfaces: token-resumable journal reads" subsection (SeqSeekableStreamLog fence) + fence-ratchet entry + `readmeQuickStart` preamble extension; enginetest `seqseek.go` doc note (fallback-cost semantics); `TestReadme*` green; capabilities gate green (no-op); readme links 946/0.
7. **M22 (harvest portion)**: TODO_LIST "Post-sweep plan execution" section (done-list + 11 open rows incl. all improvement rows); errata recorded (metaengine v4.20.1→v4.21.0 draft correction; crm untested-exports baseline already empty — the "drop 7 waiver-dup rows" step was moot; go-jose bump a no-op); SEVEN status reports indexed into `docs/status/README.md` (+1 foreign 18-16 report); canonical-facts GREEN; phase-boundary hand-commit `f077f564d`.
8. **All pre-verify gates green**: md-go (1,530 blocks), canonical-facts, file-size (cap.go 351→335 via `cappedQueryNames` move to cap_grouped.go), scoped lint 5/5, ADR numbering, changelog-symbols, doc-check (1,385 refs), engine-capabilities, package-surface, untested-exports, check-release-scripts untouched.
9. **M04 re-armed**: `build/verify-launch-14.sh` — detached (setsid) quiet-window watcher, polls every 10 s for load1+load5 < 15.5, then `VERIFY_MAX_LOAD=16 nix run .#verify > build/verify-run14.log`, writes `build/verify-run14.rc`; **12-hour bound** (expires ~09:44 CET, rc=12); blocking monitor job armed on the rc file.

## b) PARTIALLY DONE

- **M04**: watcher armed and healthy; NO verify receipt yet (load sum ~105 at report time vs ceiling 15.5).
- **M22**: harvest + errata done; the FINAL completion report still pending (it must cite the verify/CI/tag receipts).
- **M05/M07 prep**: versions evidence-derived and re-validated against the merged window (catalog v4.13.0, kv v4.4.0, id v4.7.5, metaengine v4.21.0, sqliteengine v4.7.0, cqrs-lint v4.21.0); execution blocked on M04.

## c) NOT STARTED (all blocked on the verify→push→tag chain)

M05 (push + CI-green receipt), M06 (changed-set per-module tests + census), M07 (tag wave + versions manifest + CHANGELOG cut + fold the untagged pgengine/systemscenario patch trains), M11–M15 (consumer updates — each needs a collision check first), M19-tail (comment + close #114 citing the shipped tag).

## d) TOTALLY FUCKED UP (the honest list)

1. **The #114 landing kit's hand-verification missed two design-level bugs** its first real test run caught within seconds: the grouped-without-WithCap validation skip and the type-ambiguous-key rejection (the kit's OWN fixture shape `{ID, Owner string}` was unusable as written!). Scratch kits are compile-blind by construction — land-then-iterate immediately instead.
2. **Truncated `JSONFieldName`'s body** with a sloppy edit (old/new mismatch) — repaired immediately, but it should not have happened.
3. **A pointless edit ate a newline** in cap_grouped.go (wrong edit intent) — reverted.
4. **`FieldValueAt` was initially exported** — the package-surface ratchet caught the growth (829→836); unexported to `fieldValueAt`, baseline re-pinned at the justified 835/168.
5. **cap.go hit 351/350** — the kit's "328 lines, small additions OK" arithmetic was wrong once the grouped pre-pass landed; fixed by moving `cappedQueryNames` into cap_grouped.go.
6. **Wasted three kill attempts** on the dead watcher before learning mvdan/sh's `kill` builtin silently no-ops for signals; `/run/current-system/sw/bin/kill` works. (Environment lesson now proven.)
7. **One wrong test expectation of mine** (WithSort ascending misread) — 1 of 8 first-run failures was my test, not the code.
8. **Concurrent dependency sweep interfered mid-landing**: a sibling session rewrote 94 go.mod/go.sum files at 21:14 (pgx 5.11→5.12 among others), breaking module sums I was actively testing (sqliteengine, metaengine, doc-check, api-stability) — repaired each with `go mod tidy`. My feature was unaffected; the wave's version derivations are unaffected (dep-only changes are not API). But the sweep left `go 1.27.2` PATCH-form floors in go.mod files where the documented contract is minor-form `go 1.27` — foreign state, question §g.3.
9. **lint-scope self-test trap bug** (trap fired after its local `tmp` went out of scope under `set -u`) — fixed with explicit cleanup before return.
10. The daemon absorbed the #114 CODE mid-phase into `chore:` commits before the phase boundary — the boundary hand-commit could only capture the doc tail. Sub-boundary commits would have preserved real messages for the feature itself.

## e) IMPROVE

- Landing kits must get a compile probe against the real tree in the SAME session they are written (a throwaway `go vet` of the concatenated target package beats hand-verification).
- `EnforceCap`-on-SQL was broken since #72 with zero detection because caps were memory-tested only — the adttest plain-cap contract now pins it on sqlite; consider wiring `AssertPlainCapContract` into more engine suites (duckdb/pg/mysql) when their suites are next touched.
- LSP diagnostics under load ~1900 are noise (timeouts + stale paths); direct `go build`/`go test` is the only truth in loaded sessions.
- The quiet-window ceiling (15.5) may be structurally unattainable while 10+ sibling sessions run — the load policy question below decides between patience and justified burns.
- Consider an authored-commit cadence at sub-boundaries (feature-level, not just phase-level) so daemon `chore:` blobs don't swallow feature history.

## f) NEXT (ordered)

1. Watcher fires → read `build/verify-run14.log` + `.rc`; triage any failure by class (known flake classes are fixed; timing failures → rerun the specific test at a quiet moment).
2. If the watcher expires (~09:44) with no window: decide per §g.1 (re-arm vs justified burn vs scheduled window).
3. M05: `git push origin master` (cancels CI run 41978263864 via concurrency) → poll the new run; watch the MySQL QEMU leg (the 18:10 g3 recurrence bar), Dgraph leg, benchmarks duration, flake ledger.
4. Tick T01.5 in TODO_LIST on the first fully-green CI run on origin/master.
5. M06: changed-set `GOWORK=off go test ./... -count=1` over catalog, kv, id, metaengine, metaengine/sqliteengine, cmd/cqrs-lint, middleware, event(+eventtest), storage(+pebble/bbolt/eventstore/relational), core/v5, system, systemscenario.
6. M06: `bash scripts/census-fleet.sh` (writes `docs/evidence/fleet-census-*.json`).
7. M07: `./scripts/batch-release.sh "catalog v4.13.0 …" "kv v4.4.0 …" "id v4.7.5 …" "metaengine v4.21.0 …" "metaengine/sqliteengine v4.7.0 …" "cmd/cqrs-lint v4.21.0 …"`; versions-manifest `--update` + README manifest + `check-versions-manifest --check`; push tags; `go get` smoke per tag from a clean dir; CHANGELOG cut (`[Unreleased]` → versioned sections; changelog-symbols gate); fold the untagged pgengine + systemscenario patch trains.
8. M19-tail: `gh issue close 114` with a comment citing ADR-0164 + the shipped tags + recipes §2.50.
9. M11: cqrs-htmx — collision check, then delete `usermgmt/view_cache.go` → `kv.NewCacheForViewStore` (+WithCacheCopyFunc) + kv pin + suite + hand-commit.
10. M12: crm — `scorecard.modules` projectionhost pin + cqrs-lint v4.21.0 bump + "stays unwaived" comment update.
11. M13: webphone — WithResponseHeader (429/retryAfterClause) + WithRequestBodyContent (multipart transcribe) + WithTypedParam (pagination) + drop `WithServer("webphone","","")` + catalog v4.13.0 pin + adopt WithCapPerGroup for the transcript/dial-window trim loops.
12. M14: crm/Kith — `sqliteengine.WithMaxOpenConns` wiring + read-storm re-measure (expected to also lower the workstation load floor).
13. M15: BuildFlow — cqrs-lint bump + vendorHash.nix re-pin AFTER its live session settles.
14. Improvement rows (TODO_LIST): verify-launcher promotion to scripts/, ignore-cascade tripwire, generated-artifact pre-push sweep, wedged-run watchdog + benchmarks timeout-minutes, `#erraudit` wrapper, sse_test RaceEnabled cleanup.
15. Final completion report superseding 06:39/09:46/18:10/19:08 + this report; post-wave `check-release-scripts` if batch-release.sh was touched; re-check the versions.json nightly gate.

## g) QUESTIONS (max 3, cannot resolve myself)

1. **Load policy, restated with tonight's data:** the quiet window (sum < 15.5) never opened between 18:15 and 21:45 (sums 91–1,919; currently ~105; the floor is 10+ sibling agent sessions, not the documented 8–45 services). The 12 h watcher expires ~09:44. If no window opens overnight: (a) keep re-arming — potentially for days; (b) justified burn at a raised ceiling (e.g. `VERIFY_MAX_LOAD=45–60 VERIFY_FORCE=1`, accepting per-run timing-flake triage duty); or (c) you schedule an exclusive window and I fire then. This supersedes 18:10 g2 / 19:08 g1.
2. **Session-collision policy, restated:** the dependency sweep (94 files, 21:14) landed mid-feature; a cqrs-htmx session and a BuildFlow uncommitted run were flagged at 19:08 and may still be live when the wave ships. Should M11/M15 proceed when their target repos are concurrently active (merge-conflict/lost-work risk), or does the wave wait for those sessions to settle?
3. **Foreign go.mod floors:** the sweep wrote patch-form `go 1.27.2` directives into module go.mod files (metaengine confirmed; likely fleet-wide) where the repo contract since the 92-tag wave is minor-form `go 1.27` (AGENTS contract 10; `check-go-version.sh` polices the floor + lockstep). Normalize back to minor-form before the tag wave (my read: yes — but it is another session's in-flight work and reverting it under them could break their sweep), or leave it for the sweep session to reconcile?

---
*Receipts: this report's "done" claims are each backed by a green command run this session (tests, gates, self-tests); logs under `build/`. Watcher: PID live, `build/verify-launch-14.out` trail, monitor job armed on `build/verify-run14.rc`.*
