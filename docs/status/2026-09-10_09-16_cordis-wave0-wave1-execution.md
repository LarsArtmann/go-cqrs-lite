# Status Report — Cordis Plan Execution: Wave 0 complete, Wave 1 mid-flight

> **RESOLVED (docs-health pass 2026-09-11):** **Superseded — archived by the docs-health pass 2026-09-11.** Continued by `2026-09-10_22-43` and completed by `2026-09-10_23-35` (all 27 M-tasks).
> Open work lives in [`TODO_LIST.md`](../../TODO_LIST.md); shipped surface in [CHANGELOG.md](../../CHANGELOG.md) `[Unreleased]`.


> **When:** 2026-09-10 09:16 · **Session scope:** executing [`docs/planning/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md`](../planning/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md) (user: "GET SHIT DONE! The WHOLE TODO LIST!")
> **Predecessors:** mapping report 08-04 ([`docs/architecture-understanding/2026-09-10_cordis-spatiotemporal-composability-mapping.md`](../architecture-understanding/2026-09-10_cordis-spatiotemporal-composability-mapping.md)) → status report 08-04 ([`docs/status/2026-09-10_08-04_cordis-paradigm-mapping-session.md`](2026-09-10_08-04_cordis-paradigm-mapping-session.md)) → Pareto plan 08-10 (commit `6a0cea374`, pushed).
> **Format note:** user demanded `.md` under `docs/status/`; the status-report skill's HTML default is overridden, same as the 08-04 report.

**Headline:** Wave 0 (M-01..M-05) is **fully done and verified** — including two truth corrections that invalidated prior-session claims. Wave 1's M-06 (Reset warn-guard) is **fully done** (code + tests + golden + CHANGELOG + symbol gate, all green). M-07 (projectionadapter Resettable) is **implementation-complete and test-green** but its release hygiene (api golden regen for metaengine/projectionadapter, CHANGELOG entries) was interrupted mid-step by this status request. Everything else (M-08..M-27) is not started.

---

## a) FULLY DONE (verified this session)

1. **M-01 — `projection %q is n` anomaly resolved: grep artifact, not a bug.** The real strings are `projection %q is not registered` (`projectionhost/host_reset.go:75`, `projectionhost/staleness.go:94`). The prior session's truncated grep pattern matched the prefix of "is not". No code change needed; projectionhost suite green (2.4s, GOWORK=off). The 08-04 report's "unresolved anomaly" is closed as a false alarm.
2. **M-02 — `go mod tidy` ×5 cmd modules + a claim retraction.** All five (`cqrs-bench`, `api-stability`, `cqrs-gen`, `cqrs-lint`, `doc-check`) tidied; only `cqrs-bench/go.sum` gained 3 checksum lines. **The prior session's "phantom/stale `samber/do` require" claim was WRONG**: it is a legitimate `// indirect` transitive dependency via `github.com/larsartmann/cmdguard/v4` (`go mod why` proves it). No phantom existed; nothing to kill. All 5 modules build GOWORK=off + jsonv2 tag.
3. **M-03 — all 19 pre-existing diagnostics triaged via real builds: every one is LSP noise or already-fixed.**
   - 17× gopls `stdversion` (`json.Marshal requires go1.27` in `system/adapter_*.go`): the files import `encoding/json/v2` (a go1.27 symbol set) while the module's `go` directive is 1.26.7 — the `goexperiment.jsonv2` build tag is what makes them legal. `GOWORK=off go build` + `go vet` on `system/` are green. Verdict documented as a new bullet in `docs/agents/gotchas-tooling-build.md`.
   - `cmd/cqrs-upgrade` go-finding "should be indirect": fixed by tidy (now `// indirect`).
   - `integration` genproto "unused": false positive — `go mod tidy` (the authority) keeps it as a legitimate indirect.
4. **M-04 — plan harvested into `TODO_LIST.md`.** New section "Cordis spatiotemporal-composability follow-ups (2026-09-10)" matching the file's legend/style; Wave-0 items deliberately excluded (completed work belongs in CHANGELOG, not the backlog, per the file's own scope rule); M-06..M-27 listed with effort tags and dependencies; M-26 marked `[BLOCKED]` on user decision.
5. **M-05 — mapping doc §9 cross-link appended** (plan + TODO_LIST + status-report links + unchanged guardrail), then `cmd/doc-check` green: 1029 references valid, zero warnings.
6. **M-06 — `Host.Reset` loud partial-revert guard, complete release cycle:**
   - `projectionhost.WithKeepStaleState()` new `ResetOption`; `resetConfig.keepStaleState`; warn branch emitting `h.opts.logger.Warn` (what happened + remedy) when the projection is not `Resettable`; `Resettable`/`Reset` doc comments updated; v5-hardening intent documented.
   - 2 new tests (`TestHost_Reset_NonResettable_WarnsByDefault`, `…WithKeepStaleState_SilencesWarning`) + `hasWarnContaining` helper; full projectionhost suite + vet green; the warn visibly fires in the pre-existing test's log output.
   - api-stability golden regenerated (6739 exports; `projectionhost/func WithKeepStaleState` present) + `TestEvery` green.
   - CHANGELOG `[Unreleased]` "Changed" entry added; `scripts/check-changelog-symbols.sh` green (5 citations honest).
7. **M-07 implementation — metaengine reset primitive + projectionadapter Resettable (code + tests green, see §b for the missing hygiene):**
   - `metaengine/reset.go` (new): `EngineResetter` optional capability interface (ISP-consistent with MapUpdater/PushdownScan/etc.), `ResetResult{ClearedEngines, UnclearableEngines}` + `Partial()` + `String()`, and `Store.Reset(ctx) (ResetResult, error)` — clears event log, idempotency window, poison marks, then `ResetEngine` per supporting engine; engine slice cloned under RLock before I/O; errors only on real ResetEngine failures.
   - `store_collaborators.go`: `poisonTracker.Clear()` + `idempotencyTracker.Clear()` (ring rebuilt at same capacity via `dedup.Ring.Capacity()`; unbounded map drained).
   - `memory_engine.go`: `newMemData()` shared initializer (refactored out of `NewMemoryEngine`) + `memoryEngine.ResetEngine` (drops all ADT maps, version chains, vector/search/spatial indexes).
   - `metaengine/projectionadapter/reset.go` (new): `Adapter.Reset(ctx) error` structurally satisfying `projectionhost.Resettable` (compile-time `var _` assertion in test); delegates to `Store.Reset`; **warns** (not errors) on partial reset in v4 via new `WithLogger` AdapterOption — deliberately parallel to M-06's semantics so no v4 consumer breaks.
   - `projectionadapter/go.mod`: sibling replace `metaengine/v4 => ../` (stripped at tag time, same pattern as `system/`'s matview replace).
   - Tests all green (GOWORK=off): `TestStore_Reset_ClearsMemoryEngineAndReplayState` (materialize→Reset→0 rows, event log 0, idempotency 0, replay re-applies the same event ID), `TestStore_Reset_ReportsUnclearableEngines` (plainEngine named in UnclearableEngines), `TestAdapter_Reset_ClearsMemoryBackedStore`, `TestAdapter_Reset_WarnsAndSucceedsOnUnclearableEngine` (stub engine; warn names it; returns nil).
   - Scope finding that reshaped the task: **metaengine had NO reset primitive at all** (no bulk-clear on Store or any Engine; only per-key `MapDelete`). The plan's "S/45min, delegate to Store" premise was wrong; the honest implementation is what shipped above. `sqliteengine.ResetEngine` was deliberately NOT attempted: it must handle 8 `meta_*` tables + dynamically-created planned tables + matview refresh + `multiSeq` state — too risky to rush (Verschlimmbesserung guard); recorded as follow-up instead.
8. All session work absorbed by the auto-commit daemon (9 `chore: auto-commit` commits after `6a0cea374`); working tree clean; nothing pushed (push was authorized for the plan turn only).

## b) PARTIALLY DONE

1. **M-07 release hygiene — interrupted mid-step by this status request.** Missing: (i) api-stability golden regen for the new metaengine exports (`EngineResetter`, `ResetResult`, `Store.Reset`, `ResetResult.Partial/String`) and projectionadapter exports (`Adapter.Reset`, `WithLogger`); (ii) CHANGELOG `[Unreleased]` entries for both modules; (iii) full-module test runs for metaengine + projectionadapter (only `-run Reset` targeted runs so far); (iv) lint on the three touched modules; (v) `#verify-fast`. Investigation had just confirmed api-stability parses **local source ASTs** (`cmd/api-stability/collect.go`) — it does not resolve the published graph — so the unpublished-symbols problem is moot and regen is safe to run as-is. Risk note: the daemon may commit the M-07 code before the golden catches up (TestEvery/CI would catch any drift loudly, per design).
2. **Phases.md of go-modularize skill** — lines 1–200 read in the prior session, remainder still unread (carried over).
3. **Skill reference updates** — M-10's recipe (Reset → replay) is the vehicle for documenting `WithKeepStaleState`/`Adapter.Reset` to consumers; skill references not yet touched (correctly sequenced after M-07 hygiene per the plan's dependency edges).

## c) NOT STARTED

- **Wave 1:** M-08 (goleak in system tests), M-09 (ADR-0136 temporal contract), M-10 (revert & rebuild recipe).
- **Wave 2:** M-11..M-13 (coeffect gate in `system.New` + tests), M-14 (observational-equivalence scenario test), M-15 (cqrs-lint typo rule), M-16 (catalog validation summary), M-17 (ADR ground-truth pass), M-18 (arXiv PDF fetch), M-19 (figure recounts), M-20 (release hygiene batch — partially covered by §b.1's pending work).
- **Wave 3:** M-21..M-23 (health-driven engine deactivation ADR-0137 + implementation + Doctor surfacing), M-24/M-25 (rapid fuzz equivalence + `AssertUnchanged` DSL helper), M-26 (vocabulary decision gate — blocked on user), M-27 (mapping-report diagram).
- New follow-up surfaced by M-07: `EngineResetter` implementations for persistent engines (sqliteengine first, then pebble/pg/mysql/duckdb/turso/dgraph/bbolt/badger/iroh).

## d) TOTALLY FUCKED UP (this session's own mistakes)

1. **Sibling-replace path bug:** first wrote `replace … => ../metaengine` inside `metaengine/projectionadapter/go.mod` — relative to that go.mod it resolves to `metaengine/metaengine` (nonexistent). Build failed with "replacement directory does not exist". Fixed to `=> ../` (projectionadapter lives INSIDE metaengine/). Should have thought about the relative base before writing it.
2. **exhaustruct near-miss:** wrote `result := ResetResult{}` in production code — `exhaustruct_v5` (enabled for tests too) flags empty literals with unset fields (the repo has `//nolint:exhaustruct_v5` on `event.Checkpoint{}` for exactly this). Caught before lint, fixed to `var result ResetResult`. Wasted a cycle; should have remembered the pattern from host_reset.go.
3. **Left the api golden stale across M-07 edits:** regenerated immediately for M-06 (correct), then continued editing metaengine/projectionadapter without regenerating — the daemon committed intermediate states where code > golden. No permanent harm (golden is regenerable; TestEvery is the tripwire) but it violates the repo's "API change ⇒ golden in the same edit" discipline for the M-07 batch. This report's interruption is the honest reason, not an excuse.
4. **Plan-estimate blindness on M-07:** the plan priced M-07 at 45min on the assumption `Store` already had a reset to delegate to. One `rg` for `func (s *Store) (Reset|Clear|…)` before planning would have caught it at plan time. Lesson: the 08:10 plan was built from the 08:04 report's citations, not from a capability audit of metaengine's reset surface.
5. **Carried-over fuckups now closed:** the 08-04 report's "samber/do phantom requires" (§a.2) and "`is n` anomaly" (§a.1) were both false alarms created by truncated greps in that session — the "verify before claiming" discipline slipped twice there and this session paid two verification detours to retire them.

## e) WHAT WE SHOULD IMPROVE

1. **Grep patterns must be anchored to complete tokens** — both retired false alarms (`is n`, "phantom require" without checking `// indirect`) came from pattern truncation and missing classification context. Rule of thumb: when a grep result looks like corruption ("is n"), suspect the pattern first.
2. **Plan capability-audits before pricing tasks** — M-07's premise (Store.Reset exists) was never checked. For any task phrased "wire X to existing primitive Y", verify Y exists in the plan pass, not mid-execution.
3. **Same-edit golden discipline must survive interruptions** — regen the golden the moment an exported symbol lands, especially with an auto-commit daemon running. Consider regen-per-symbol rather than regen-per-task.
4. **The v4-warn/v5-hard pattern is now twice-proven** (M-06 host-level, M-07 adapter-level) — make it the explicit template for every remaining behavior task in the plan (M-11's coeffect gate especially: error-by-default is tempting but wrong for v4).
5. **`EngineResetter` coverage is the honest gap** — memory only. Until sqliteengine implements it, every SQLite-backed projectionadapter Reset warns. That's correct-but-noisy; prioritize the follow-up.
6. **exhaustruct awareness when authoring new struct types** — prefer `var x T` for zero-value results; reserve `{}` literals for fully-set structs.

## f) NEXT — up to 50 things (ordered; ★ = new this session)

Wave 1 completion:
1. ★ Regenerate api-stability golden (metaengine + projectionadapter exports) + `TestEvery` — closes M-07 hygiene.
2. ★ CHANGELOG `[Unreleased]` entries: metaengine `EngineResetter`/`ResetResult`/`Store.Reset`; projectionadapter `Adapter.Reset`/`WithLogger`; run `check-changelog-symbols.sh`.
3. ★ Full GOWORK=off test runs: metaengine, projectionadapter, projectionhost (M-07/M-06 regression gates).
4. ★ Lint the three touched modules (`nix run .#lint` scope or per-module golangci) — exhaustruct/wrapcheck/line-length on new files.
5. ★ `nix run .#verify-fast` — Wave 1 partial gate.
6. ★ `go work sync` + workspace check after the go.mod replace addition (`scripts/check-workspace-sync.sh`).
7. M-08: add goleak (test-only) to `system` go.mod; wrap TestMain with `VerifyTestMain`; run `-race`.
8. M-09: write `docs/adr/0136-temporal-composability-contract.md` (invertibility ladder: replayable → compensable → must-be-an-event; user decision rule; cite M-06/M-07 as the enforcement points).
9. M-09: link ADR-0136 from mapping doc §9 + AGENTS.md; doc-check.
10. M-10: revert & rebuild recipe into skill `readmodels.md`/`recipes.md` (Reset → Stop → replay; `WithKeepStaleState`; `EngineResetter` capability table).
11. ★ ADR-0136 addendum: document the engine-reset capability ladder (memory=full, persistent=follow-up) as the spatial half of the temporal contract.
Wave 2:
12. M-11: `system.New` coeffect gate — dangling subscription (projection consumes a type nothing produces) → hard error + disable option.
13. M-12: unconsumed event types → warn, wired into New.
14. M-13: gate tests (dangling/unconsumed/disabled; alias-equality via `record.Type`).
15. M-14: observational-equivalence scenario test (A alone vs A+B interleaved, identical A state).
16. M-15: cqrs-lint static rule for `EventTypes()` vs registered producers.
17. M-16: catalog export carries coeffect-validation summary; `#check-eventcatalog`.
18. M-17: open ADRs 0114/0123/0124/0126/0127; verify mapping-doc claims; addendum corrections.
19. M-18: fetch arXiv 2608.25512 PDF; grounding addendum (calculus + equivalence defs).
20. M-19: recount 82 modules / 47 DeferClose sites; correct via addendum.
21. M-20: consolidated release-hygiene pass if anything from §b remains (golden + CHANGELOG + verify-fast).
22. ★ `metaengine/sqliteengine`: implement `ResetEngine` (DELETE FROM 8 meta_* tables + planned-table drop/recreate + matview refresh + multiSeq reset) + tests — the production 80% path for one-call reverts.
23. ★ `metaengine/pebbleengine` + `bboltengine`: `ResetEngine` (truncate/delete-range the key space).
24. ★ Remaining engines (`pg`, `mysql`, `duckdb`, `turso`, `dgraph`, `badger`, `iroh`): `ResetEngine` or documented non-support (Doctor line).
25. ★ `Doctor`/`GetEngineStats`: surface per-engine reset capability ("resettable: yes/no") so operators see it before relying on Reset.
Wave 3:
26. M-21: ADR-0137 design spike — health-driven engine deactivation (errorfamily storm → quarantine/reroute/auto-reprobe).
27. M-22: implement deactivation + failing-engine tests (multi-session).
28. M-23: health state in `Doctor`/`GetEngineStats` + tests.
29. M-24: rapid fuzz — A unchanged under randomized B interleavings.
30. M-25: scenario DSL `AssertUnchanged(projection)` helper + docs.
31. M-26: vocabulary positioning decision (blocked on user; default internal-only).
32. M-27: Mermaid/D2 diagram for the mapping report; optional HTML render.
Housekeeping:
33. ★ Finish reading go-modularize `references/phases.md` (lines 200+).
34. ★ Update skill `references/modules.md` (projectionadapter row gains Reset/WithLogger; metaengine row gains Store.Reset) after M-07 hygiene.
35. ★ `TODO_LIST.md`: mark M-06/M-07 done (delete entries per scope rule) once hygiene closes; add the EngineResetter follow-ups (items 22–25).
36. ★ Tag-wave note: `projectionadapter`'s new sibling replace + metaengine pin bump must ride the next release train (`scripts/tag-release.sh` strips replaces; pins bumped).
37. ★ Consider a `TestAdapter_Resetty`-style meta-test asserting every in-repo engine module either implements `EngineResetter` or carries a `// no-reset: <reason>` marker (split-brain guard).
38. ★ doc-check pass over all skill references after M-10 lands.
39. ★ Status-report hygiene: this file's predecessor (08-04) still cites the two now-retracted false alarms — an addendum note there (or acceptance that point-in-time docs stay unedited) per the repo's addenda policy.
40. ★ Re-run `#check-duplication` — new `Clear()` methods on two trackers + `newMemData` refactor are candidates for art-dupl false positives; annotate or accept.
41. ★ Coverage check on new files (`metaengine/reset.go`, `projectionadapter/reset.go`) — `#check-coverage` drift.
42. ★ Example: `example/metaengine-quickstart` (or a new one) demonstrating Stop → Reset → replay rebuild.
43. M-15 follow-up: wire the cqrs-lint rule into `#lint` config if it proves quiet.
44. ★ Benchmark: Reset cost on large memory stores (document O(1) map-swap behavior) in bench docs.
45. ★ Verify `nix run .#verify` end-to-end before calling Wave 1 closed (per-task gate discipline).
46. ★ Backfill: run the `is n`/samber/do verdicts into the 08-04 report's addendum section so future readers don't re-chase them (same policy as item 39).
47. ★ Check whether `system/` (which composes projectionhost) should default `WithKeepStaleState` off but surface the M-06 warn through its own logging config.
48. ★ Confirm `Adapter.Reset` interacts correctly with `Host.Reset`'s checkpoint-first ordering (host clears checkpoint, then adapter clears state — the reverse-order hazard comment in host_reset.go:81-85 still holds; add a test asserting both orders behave).
49. ★ Grep the skill references for stale "Reset is checkpoint-only" claims that M-06/M-07 changed.
50. ★ If execution resumes: consider batching items 1–6 as the "M-07-hygiene" commit before touching M-08.

## g) Questions I cannot answer myself

1. **Persistent-engine reset priority:** should `sqliteengine.ResetEngine` (and then the other 9 engine modules) be implemented as part of this plan's remaining time, or is memory-only + warn-on-partial acceptable for v4.x with engines as a separate follow-up train? I deferred it on risk (planned tables/matviews/multiSeq), but it gates the "production 80% path" claim.
2. **Partial-reset semantics confirmation:** M-07 ships warn-and-return-nil on partial reset in v4 (hard error at v5), mirroring M-06 — both are behavior-visible changes where "silently did nothing useful" becomes "warns loudly". Is warn-first the confirmed policy for ALL remaining behavior tasks (M-11's coeffect gate especially, where an error-by-default is the tempting but breaking choice)?
3. **Continue or checkpoint?** The interrupted step (§b.1: golden + CHANGELOG + full test runs + verify-fast, items 1–6) would take one more pass to close out M-07 completely. Say "FINISH M-07" (or "GET SHIT DONE" again) and I resume; otherwise I hold here as instructed.

---

**Bottom line:** Wave 0 shipped with two prior-session claims corrected (truth-before-durability working as designed). M-06 is the full release cycle, done. M-07's code and tests are green but its golden/CHANGELOG/test-suite/lint/verify-fast batch is open — that is the single unfinished thread. Waves 2–3 untouched. No consumer-visible breakage introduced anywhere (warn-first held).

*Point-in-time report; corrections via addendum only.*
