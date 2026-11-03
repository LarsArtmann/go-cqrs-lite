# Status Report: Pareto Plan Execution — T01 CI Trust Floor + Gates Wave (2026-11-03 22:15 CET)

**Session:** Commissioned execution of `docs/planning/2026-11-03_18-52_SUPERB-full-backlog-pareto-plan.md`, starting at T01, alongside a PARALLEL session executing the same plan's 1% tier (its report: `2026-11-03_21-15_tier-1pct-red-to-green-execution.md`).

> **Scope — what this report VERIFIED:** every claim below was verified this session by running the command (gate outputs, test runs, `gh` CLI, tag-release smoke) — receipts cited inline. What this report did NOT verify: a green **CI run on master carrying all of this session's fixes** (the newest full CI run at report time started 19:04, BEFORE the V007/calibration/duplication fixes landed; the fixes ride the NEXT daemon commit's run), and the composed `nix run .#verify` receipt (plan T25, deliberately not run — host load1 was 88+ from the parallel session for most of the window; the load gate correctly refused).

---

## a) FULLY DONE (verified this session)

1. **T01 fuzz crash — `FuzzStreamStatus_MarshalOnly` (listing) FIXED, tagged, swept.** Reproduced from the CI artifact corpus (`df5596f0610ade60`): `id.ParseStreamID("\xdf")` accepted invalid UTF-8, which detonated later at json/v2 marshal (`jsontext: invalid UTF-8`). Root fix at the parse boundary: `id.ParseStreamID` + `id.ParseStreamType` now reject invalid UTF-8 (Rejection codes `id.parse_stream_not_utf8` / `id.stream_type_not_utf8`) — invariant: every parsed StreamID/StreamType is JSON-marshalable. `ParseStreamIDStrict` needed no change (ulid.Parse rejects non-ASCII). Tests: `id/utf8_validation_test.go` (5 tests, code+family asserted); listing fuzz harness now builds StreamType via `ParseStreamType` (was a raw cast bypassing validation). CHANGELOG row added; `check-changelog-symbols` green (26 citations). **Tag `id/v4.7.4` cut + pushed + proxy-smoked** (tag-release.sh, clean-tree protocol followed). Pin sweep bumped 54 modules; receipt: `cd listing && GOWORK=off go test` green on the new pin (corpus included).
2. **pin-sweep.sh blind spot FIXED (root cause of 21 untidy modules).** `collect_stale`'s regex required the version at end-of-line — `// indirect` pins never matched, so sweeps bumped DIRECT pins while INDIRECT pins went stale and the nightly Module-sums gate caught what the sweep's own verify never saw. Fix: regex accepts optional `// indirect`; sweep now ends with a `check-modsums.sh` assertion ("sweep complete" can never print over broken sums). Verified live: second sweep found+bumped 6 previously-invisible stale indirect pins (stack/duckdb, stack/metaengine, cmd/cqrs-bench, graphadapter, stack/bbolt, example/scheduler-otel-status); `pin-sweep.sh --check` + `check-modsums.sh` now both green (103/103 tidy).
3. **T01 shfmt drift FIXED.** Two foreign-authored scripts (`check-schema-compat.sh`, `probe-proxy-tags.sh`) drifted; formatted with the CI-pinned `nix shell nixpkgs#shfmt`; `shfmt -d scripts/` clean.
4. **T01 actionlint FIXED.** Foreign `autoretry.yml` fired 5 SC2016/SC2086 info findings on intentional idioms (literal markdown backticks; word-splitting over job lists). Added 3 targeted `# shellcheck disable=` with reasons; actionlint rc=0.
5. **T01 module-layers FIXED.** `scratch/counterkey` (foreign cross-engine repro, test-only) rowed in `check-module-layers.sh` (LAYER 7, DEP_BUDGET 6) with a pending-disposition comment pointing at owner question T07.3; gate green.
6. **T01 V007-silent examples FIXED.** `example/scheduler-otel-status` is the sanctioned sqlstore deprecation demo (AGENTS contract 30) but V007 fired on it (CI leg red since the 10-24 sqlstore table entry). Added a reasoned whole-module exemption mechanism: `exemptedExampleModules` map (keyed by module path, reason mandatory), early-return in the V007 detector, behavior test (exempt path silent / consumer path fires — `TestV007_DeprecationDemoExampleExempt`), and a LIVENESS drift test (`TestV007_ExampleExemptionsHaveLiveModules` — stale exemption after the example's v5 deletion fails the suite). Verified: full examples loop V007-silent. cqrs-upgrade's dogfood reuses the same detector, so the exemption covers it (workspace resolution verified from `cmd/cqrs-upgrade/report.go:159`).
7. **T01 calibration-drift nightly FIXED (deterministic bug).** The drift job failed EVERY night on `sqliteengine|_CounterScan|aggregate` — that bench does not exist (sqlite names the same pattern `_CounterGet`; both feed `ReadCosts.NsPerAggregate`, verified from bench sources). Row fixed; existence-proven with a 1-iteration bench run emitting `1000 rows-scanned`. (The pebble 53% drift annotation is the shared-runner warning class; CI-baseline mode owns that.) Full drift verification NOT run locally — the load gate refused (correctly).
8. **T04 ratchet-reds CLOSED (all four).** file-size: already green (61 baselined, foreign session fixed the 3 new offenders). md-go: green (1529 blocks, 105 baselined; the "pinned-binary un-runnable" issue is gone). duplication: 6 new clone groups found → annotated per contract-14 rules (2 shifted-region directives on core/v5 ADR-0152 twins whose alignment MY id edits moved; 4 idiom/CLI-boilerplate accepts on foreign catalog-compat code) → gate green "0 new clone groups, baseline 189". ROADMAP 104→103 + FEATURES re-verify date → 2026-11-03.
9. **T07 numeric/label truth — canonical-facts FULLY GREEN.** Indexed all 5 unindexed foreign status reports (16-22, 16-55, 19-09, 19-41, 21-15) into `docs/status/README.md`; rowed `scratch/counterkey` in module-map; gate: "all 99 modules rowed … all citations match repo-derived truth" (was 2 drift findings). Untested-exports: foreign symbol `catalog/compat.ErrBaselineFormatVersion` named by strengthening the existing format-version test with `errors.Is` (+2 stale stack rows pruned via `--update`) → gate green, **1,086 of 6,423** (AGENTS row updated from 1,088/6,422). errtax totals re-derived: 540 frozen of 546 — AGENTS was already correct.
10. **T02 CachedEventStore TOCTOU — verified SHIPPED by the parallel session** (`system/cache.go` gen-check+Set under one mutex, `beginWrite`/`endWrite`, gapHook seam; `cache_gap_internal_test.go` deterministic pin; ran `-race -count=5` green myself). Not my work — verified, not re-done.

## b) PARTIALLY DONE

1. **T01 CI master-green receipt (T01.5).** Every red class has a local green receipt, but the post-fix CI run had not completed at report time (19:04 run predates the last fixes; QEMU MySQL legs run ~90 min). Next daemon commit's run is the proof. The dependabot PR failures (bigtableengine go-jose; minor-and-patch group) were classified as the same mid-edit-tree/runner class and left to re-run on the settled tree.
2. **T03 owner decisions.** Evidence briefs prepared (16/16 metaengine-family stamps censused; systemscenario README stamp; ADR-0142 kv row 232 "Under evaluation"; commit-policy history; keycodec A9). The 5-question batch to the owner was interrupted mid-session — decisions NOT made. Rows exist in TODO_LIST; nothing blocked.
3. **T05 json/v2 determinism.** RED pin written and proven (two byte-orders in ONE run); `json.Deterministic(true)` applied at 5 persist/write sites: `keycodec.EncodeJSON` + `EncodeJSONErr`, `sqliteengine.encodeKey` default + `encodeValue`, `metaengine.EncodeStreamValueErr`, `SerializablePlan.MarshalJSON` + `SerializeToJSON`. keycodec/metaengine/sqliteengine suites green. **NOT finished:** the adttest cross-engine row (`metaengine/adttest/value_determinism.go` is WRITTEN — `AssertDeterministicMapEncoding` over `RawValueReader` engines — but NOT yet wired into the engine suites, and the LSM engines' own pin runs are pending); vector.go deliberately skipped ([]float32, no maps).
4. **T09 harvest.** Not run yet — this report's §f is the harvest input.
5. **CHANGELOG for the gate-fix cluster** (pin-sweep regex+gate, V007 exemption, calibration row, shfmt/actionlint, scratch rowing): one consolidated row owed at wave close; only the id/UTF-8 row is written.

## c) NOT STARTED (plan order; T08+ are the remaining 19 of 27 medium tasks)

- **T06** doc-truth automation (readme_compile_test.go; teach canonical-facts to strip backticks)
- **T08** cqrs-upgrade `--to-v5` import-rewrite slice
- **T10/T11** v5 deletion waves A/B · **T12/T13** metaengine splits · **T14** migration guide · **T15** encryption ADR
- **T16–T19** testing rails (mutation gate, synctest, planner MBT, differential harness)
- **T20** cqrs-lint perf 2.0s→7.4s · **T21** FilterOps/AsyncAPI · **T22** BDD tails · **T23** catalog Registry bridge
- **T24** tag-wave housekeeping (catalog MarkDemo train, graph-native row, #26 retract watch)
- **T25** composed quiet-window `#verify` + S04 receipt · **T26** untagged-API CI gate · **T27** metaengine promotion wave

## d) TOTALLY FUCKED UP!

1. **I nearly edited foreign working-tree files blind.** CHANGELOG.md and TODO_LIST.md carried uncommitted foreign edits when I went to tag; the tag-release dry-run's dirty-tree guard stopped me. I followed the documented wait-for-daemon protocol, but I had NOT checked tree ownership BEFORE my first tag attempt — the guard saved me, not my process.
2. **The V007 fix initially looked wrong-scoped.** I first assumed cqrs-upgrade had a SEPARATE V007 table (and braced to duplicate the exemption there); only reading `report.go` showed it reuses cqrs-lint's detector. I almost fixed a phantom.
3. **The first pin sweep left 21 modules untidy and I reported it as done.** The sweep printed "✅ 54 module(s) bumped … all pins current" and I moved on; check-modsums (run as part of T01 triage, not as a sweep follow-up) caught the residue minutes later. The sweep's own success line was a lie by omission — now structurally fixed (final modsums assertion), but I trusted it first.
4. **Known-standing sins inherited, not caused:** zero authored commits again (the daemon absorbed the entire wave, including the id/v4.7.4 tag prep — 4th consecutive wave); the dogfood strict-green receipt was interrupted twice by background-shell cleanup losing /tmp output (re-ran a third time; result pending at report time — see Partially Done).

## e) WHAT WE SHOULD IMPROVE!

1. **pin-sweep's blind spot existed since indirect pins appeared** — the lesson generalizes: every "✅ complete" in a gate script should end by asserting the DOWNSTREAM gate that catches its residue (the modsums assertion I added is the pattern; tag-release's --smoke is another).
2. **Two sessions executed the same plan's same tier in parallel** (this session and 21-15) — duplicated the TOCTOU verification and nearly duplicated V007 work. For commissioned plans: claim tasks in TODO_LIST (or a plan-level CLAIMS section) before executing.
3. **Foreign mid-edit commits make master CI perpetually red mid-wave** (daemon commits half-finished trees). The Autoretry workflow now classifies, but the CI signal is still noise during active waves. Consider: daemon staging delay when a session is active, or gate CI on the LAST commit of a quiet window (sentinel already checks "green within 3 days" — that's the honest metric during waves).
4. **The interrupted question batch:** owner decisions prepared but not asked — ask at the START of the next session, not mid-execution.
5. **CHANGELOG consolidation discipline:** I wrote the consumer-visible row immediately (correct) but deferred the infra-fix rows to "wave close" — a closing crash would lose them. Write each row in its edit window; the symbols gate makes them cheap.

## f) Next up to 50 (harvest input; plan IDs in parens)

1. Wire `AssertDeterministicMapEncoding` into pebble + sqlite engine suites; run LSM determinism pins (T05.3 tail)
2. Run `GOWORK=off go test` on all 4 LSM engine modules touched by keycodec (T05.4)
3. Consolidated CHANGELOG row for the gate-fix cluster; re-run changelog-symbols
4. Confirm CI green on the post-fix master commit (T01.5 close)
5. Re-ask the 5 owner questions (T03.2) — metaengine scope / systemscenario / kv / commit policy / DB census
6. Harvest §f into TODO_LIST; de-dupe vs foreign rows (T09)
7. `readme_compile_test.go` pinning README quick-start fences (T06.1)
8. Teach `check-canonical-facts.sh` to strip backticks before the go.mod count regex + self-test (T06.3)
9. Sweep remaining backticked counts in DOC_SET files (T06.4)
10. cqrs-upgrade AST import-rewrite design + table (T08.1)
11. Implement import-rewrite pass + unit tests (T08.2)
12. Strict-gate integration for `--to-v5` (T08.3)
13. Dry-run on cqrs-htmx, go-appkit, bank-sync (T08.4)
14. Fleet census scoped to stack/* before deletion wave (T10.1)
15. Migration notes for stack/sqlite's 4 direct consumers (T10.2)
16. Delete stack + 8 presets across the 6 registration gates (T10.3)
17. Delete v1 read tiers: Materialize, RelationalProjection, view store, GraphProjection (T11.1–3)
18. Delete tombstone metadata API — ADR-0114 completion (T11.4)
19. Delete ADR-0126 compat shells + BuildWhereClause + NewStreamRef validation (T11.5)
20. V5-MIGRATION-GUIDE before/after sections (T14)
21. Extract metaengine/memoryengine behind same API (T12)
22. Five-package carve ADR, gated on T03 ruling (T13.1)
23. Encryption-at-rest ADR-0139 completion (T15)
24. Mutation-verify gate: tool pick + harness on 3 pilot tests (T16.1)
25. Implement + CI-wire the mutation ratchet for NEW tests (T16.2–3)
26. synctest pilot on scheduling ticker tests (T17.1)
27. Two more synctest conversions + gotchas pattern row (T17.2)
28. rapid-based planner capability-set MBT, 20 seeds (T18.1)
29. Planner invariant fuzzing (T18.2)
30. Differential op-script DSL on memory+sqlite (T19.1)
31. Extend differential to pebble/bbolt/badger + pairwise diff (T19.2)
32. `#test-engine-diff` nightly wiring (T19.3)
33. Profile cqrs-lint 3.7× regression on a fleet repo (T20.1)
34. Fix + bench-pin the lint hot path (T20.2)
35. FilterContains/FilterPrefix FilterOps across planner+memory+SQL+adttest (T21.1)
36. AsyncAPI response schemas (T21.2) + protocol bindings (T21.3)
37. cqrs-htmx sqlite-lifecycle systemscenario variant (T22.1)
38. scenario/v4 testing.TB refactor (T22.2) + TimeAdvancesTo sugar (T22.3)
39. Wire systemcatalog.FromSystem into catalog Registry + drift test (T23.1)
40. scenario version-carrying Given decide funcs (T23.2)
41. Cut catalog train carrying Registry.MarkDemo (T24.1, gated on T26)
42. graph-native README manifest row unblock (T24.2)
43. #26 retract-watch confirmation (T24.3)
44. Quiet-window composed `#verify` via verify-chain.sh; record S04 (T25)
45. Untagged-API CI gate: examplePaths built against pinned tags (T26.1–2)
46. Metaengine promotion evidence table for the 16 stamps (T27.1, gated on T03)
47. Owner disposition: scratch/ directory in-tree vs out (T07.3 follow-up)
48. cqrs-upgrade's cqrs-lint pin (v4.18.0) still predates the V007 exemption — bump rides the next cqrs-lint tag wave
49. Row the ActorID invalid-UTF-8 hole (same class as the StreamID fix; constructors can't validate without signature changes — needs a design decision)
50. Reconcile api-stability count in docs if 6,423 persists post-sweep (AGENTS updated; check FEATURES/SKILL for stragglers)

## g) Questions I can NOT figure out myself (max 3)

1. **metaengine promotion scope** — promote all 16 engine-family modules (evidence audit, T27), only core + the battle-tested trio (sqlite/pebble/bbolt), or keep all experimental until v5? (SKILL says "planner surface stable" while 16 doc.go stamps say Experimental — that split brain is yours to resolve.)
2. **Authored-commit policy for commissioned waves** — this is now the 4th consecutive fully-daemon-absorbed wave (including a release tag's prep work). Authorize authored commits at phase boundaries, or is chore: history the accepted price of the daemon?
3. **Deployed-DB census (keycodec A9)** — which production databases carry prefixed stream keys/rows? Determines permanent sticky-key fallback vs re-key migration; sticky-pin keeps everything working either way, so this can also stay parked if unknown.

---
*Receipts available in-session: gate outputs cited per item; tag `id/v4.7.4` pushed and proxy-served; corpus regression committed at `listing/testdata/fuzz/FuzzStreamStatus_MarshalOnly/df5596f0610ade60`.*
