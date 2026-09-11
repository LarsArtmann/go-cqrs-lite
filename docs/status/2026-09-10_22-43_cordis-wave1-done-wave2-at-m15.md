# Status: Cordis Pareto Execution — Wave 1 complete, Wave 2 at M-15

> **RESOLVED (docs-health pass 2026-09-11):** **Superseded — archived by the docs-health pass 2026-09-11.** M-15..M-27 completed by `2026-09-10_23-35_cordis-all-27-done.md` (same day).
> Open work lives in [`TODO_LIST.md`](../../TODO_LIST.md); shipped surface in [CHANGELOG.md](../../CHANGELOG.md) `[Unreleased]`.


> **When:** 2026-09-10 22:43 · **Session:** resumed "READ, UNDERSTAND, RESEARCH, REFLECT / keep going until done" after the 09-16 status break
> **Input:** [`docs/planning/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md`](../planning/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md) (M-01..M-27)
> **Prior state:** Wave 0 done; M-06 done; M-07 code done, release hygiene interrupted (see [`2026-09-10_09-16_cordis-wave0-wave1-execution.md`](2026-09-10_09-16_cordis-wave0-wave1-execution.md))
> **This session:** M-07 hygiene → M-14 all closed with verify gates; M-15 research-phase only. Working tree clean (daemon absorbed ~14 commits). Nothing pushed.

---

## a) FULLY DONE (this session, each with its verify gate)

### M-07 — release hygiene (the interrupted thread, closed)
- **api golden regen:** 6739 → 6751 exports (+12 metaengine/projectionadapter reset symbols); `TestEvery` meta-test green.
- **CHANGELOG:** `[Unreleased]` "Added — metaengine + projectionadapter: one-call read-model revert" entry (cites `EngineResetter`, `ResetResult`, `Adapter.Reset`, `WithLogger`, sibling-replace note); `check-changelog-symbols.sh` green (7 citations).
- **Tests:** full `GOWORK=off` suites green for metaengine, metaengine/projectionadapter, projectionhost.
- **Lint:** 3 touched modules 0 issues (after `nix fmt` — see fuckup #2).
- **`go work sync`** + `check-workspace-sync.sh` OK (the M-07 replace is synced).

### M-08 — goleak in system tests
- `go.uber.org/goleak v1.3.0` promoted to direct test dep; `system/main_test.go` `TestMain` wrapped with `goleak.VerifyTestMain` (doc comment ties it to teardown-completeness-as-CI).
- Full system suite green **with `-race`** — zero leaks on first run (no ignores needed).

### M-09 — ADR-0136 temporal composability contract
- `docs/adr/0136-temporal-composability-contract.md`: invertibility ladder (**replayable → compensable → must-be-an-event**), the user decision rule ("what is its inverse?"), M-06/M-07 as enforcement points, engine-reset capability ladder (memory=full, persistent=follow-up), v4-warn/v5-hard stance, anti-patterns.
- Linked from mapping doc §9 (append-only policy held), AGENTS.md internal contract **#22** added.
- **Collateral fix:** `scripts/verify-docs.sh` ADR-index check glob `00*.md` only saw ADRs 0001–0099 → fixed to `0*.md`. Indexed **0100–0136 into docs/README.md** (+37 rows) and **0130–0136 into docs/adr/README.md** (+7 rows). Check now honest: 135 files = 135 indexed.
- doc-check green (1029 refs).

### M-10 — revert & rebuild recipe
- `readmodels.md` new section "Revert & rebuild: Reset → replay from zero (ADR-0136)" (+ TOC entry): Stop→Reset→Start flow, warn-guard semantics, `WithKeepStaleState`, Resettable covers table (projectionadapter free / SQLViewStore `DeleteAll` wrap / hand-rolled), engine capability table, direct `store.Reset` + `ResetResult.Partial()` inspection, scope guard (external effects → deriver; facts → tombstone).
- `recipes.md` §2.32 cross-ref entry. doc-check green (**1038 refs**, +9).

### M-11/M-12/M-13 — coeffect validation gate in system.New
- **Honest discovery:** deciders emit events only at runtime — the plan's "collect produced event types" premise had no static source. Gate built as **opt-in declaration** instead:
  - `DomainConfig.Events []event.Type` — declared journal universe (own emissions + external importers); empty = gate off (zero v4 breakage).
  - `DomainConfig.DisableCoeffectValidation bool` — escape hatch.
  - Dangling subscription (consumed ∉ declared) → hard error `system.ErrDanglingEventSubscription` naming type + consumers + remedy; fires fail-fast in `New` before engine creation.
  - Unconsumed declared type → `TierAdvisory` ScreamReport diagnostic (`coeffect.unconsumed_event`) + `slog.Warn`.
- New `system/coeffect_gate.go`; `buildProjections` now also returns the consumed map (event.Type → consumer labels); `system/constructor.go` wired.
- **4 tests** (dangling `errors.Is` + nil-system + message content; advisory in ScreamReport; disabled skips; **record.Type alias lockstep** — declares the universe via `record.Type` values) — all green under `-race`.
- Full system suite green (goleak active), lint 0 issues, golden regen **6751 → 6752** (`system/var ErrDanglingEventSubscription`).

### M-14 — observational-equivalence scenario test
- `scenario/observational_equivalence_test.go`: task (A) + audit (B) collections on ONE shared metaengine Store; `interleavedProjection` composite runs B's Handle after A's per event; asserts via DSL: baseline (A alone, `ThenNoError`) then A+B interleaved `ThenQueryResult(DeepEqual)` against the baseline value. This is the Cordis observational-equivalence theorem as a regression gate.
- scenario go.mod gained **test-only** deps: metaengine v4.13.0, projectionadapter v4.4.1, record v4.5.0 (budget-exempt; all published tags verified before adding).
- Suite green, lint 0 issues, workspace sync OK.

---

## b) PARTIALLY DONE

### M-15 — cqrs-lint event-type typo rule (research phase, ~15 min in)
- Surveyed rule layout (correctness/c###, consistency/d###, architecture/e###).
- **Key finding mid-analysis:** **E006 already exists** ("event emitted but no projection or fold handles it") — that is the *emitted→unhandled* direction. The plan's M-15 ("projection `EventTypes()` vs registered producers") is the **reverse** direction: a projection consuming a type **nobody emits** (the typo class). The registry (`ctx.Registry.EventTypesEmitted`, `Projections[].EventTypes`, `CollectFoldCaseStrings`) has everything needed. Interrupted here — rule not written.

### M-20 — consolidated release hygiene (deliberately deferred, not started as a pass)
- The plan itself makes M-20 the consolidated gate; this session already did golden (6752) + symbol gate + CHANGELOG entries incrementally. Still outstanding as a *final pass*: CHANGELOG entries for M-08..M-14 surface (goleak is test-only → no entry needed; scenario test-deps → likely no consumer-visible entry; **coeffect gate IS consumer-visible and needs its `[Unreleased]` entry — NOT yet written**), and a green `nix run .#verify-fast`.

---

## c) NOT STARTED
- **M-16** catalog validation summary · **M-17/18/19** ground-truth pass (ADR claims, arXiv PDF, 82/47 recount — note module count is now **84**, AGENTS.md still says 82, verify-docs only checks "core docs" so it passed) · Wave 3 in full: **M-21** ADR-0137 deactivation, **M-22** impl, **M-23** Doctor health, **M-24** rapid fuzz, **M-25** `AssertUnchanged` DSL helper, **M-26** vocabulary decision gate, **M-27** diagram.
- Housekeeping: TODO_LIST M-06/M-07 entry deletion (its completed-work rule), EngineResetter follow-up entries, tag-wave note for the projectionadapter replace.

## d) TOTALLY FUCKED UP (and what happened after)

1. **CHANGELOG python merge dropped a release header.** Merging the orphan `## [Unreleased]` (see below) into the live one, my script lost the `## [command/v4.9.0, …] — 2026-09-08` header line and doubled a `---`. Caught by my own post-check (header count 54, `## [command...]` missing); repaired with a surgical edit; then validated **2044 bullets before = 2044 after** (no content loss), exactly 1 `[Unreleased]`, no double rules, symbol gate green. Lesson: structural-validate BEFORE writing, not after.
2. **Wrote unformatted code three separate times.** golines/gofumpt failures in projectionadapter (M-07 hygiene), then system (coeffect gate), then scenario (equivalence test): 2+2+5 lint findings, each round fixed by `nix fmt`/edits. Should run `nix fmt` as part of every edit batch, not as lint-failure feedback.
3. **ADR index script created 11 duplicate rows.** docs/README.md already had 0100–0110 rows that the old `00`-prefixed regex couldn't see; my inserter didn't check for existing rows. Deduped keeping originals; check now 135/135.
4. **First verify-fast run failed on a pre-existing landmine, not my code:** duplicate `## [Unreleased]` sections (live one at line 7, orphan at 3860). Forensics: the orphan (194 lines: encryption docs, P014, tripwires, release tooling, docs-truth batch, snapshot-migration/watermill/benchkit/tursoengine/SARIF fixes) was auto-committed 2026-09-09 02:31 — AFTER the 09-08 train tags; `git cat-file` proved **P014 is NOT in `cmd/cqrs-lint/v4.10.0`**. So it is genuinely unreleased work a prior session parked below the released trains. Merged into the live `[Unreleased]` (see #1 for the damage I caused doing it).
5. **`TestTursoEncryption_RoundTrip/aes256gcm` failed under verify-fast parallel load** ("Decryption failed for page=1"), passed 3/3 in isolation → classified transient load flake (embedded-turso page decryption under contention). **Not re-verified inside a full verify-fast run yet** — M-20 must confirm.

## e) WHAT WE SHOULD IMPROVE (session-honest)

1. **Single-file python edits on 4k-line files are dangerous** — the dropped-header class. Prefer anchor+validate (assert expected structure before AND after) or several small `edit` calls.
2. **`nix fmt` belongs in the edit loop** (before lint, not after it fails).
3. **The ADR-index blind spot survived 36 ADRs** — checks that "pass" for months can be passing vacuously; when adding a check-adjacent feature, re-derive what the check actually covers (this session fixed it; the same skepticism should apply to other `verify-docs.sh` checks).
4. **Plan premises keep being wrong about repo reality** (M-07: no metaengine Reset existed; M-11: no static producer set). The verify-first reflex caught both — keep it.
5. **CHANGELOG orphan class is systemic:** the auto-commit daemon absorbs doc edits that never merge into the live `[Unreleased]`. A cheap tripwire (exactly-one `[Unreleased]`) already exists in verify-docs — consider a second one: `[Unreleased]` must sit directly under `# Changelog` header block (position check), so a second one anywhere fails loudly at the next verify, not days later.
6. **Flake ledger:** the turso encryption flake under load should get a TODO_LIST entry if M-20 sees it again (two sightings = pattern).

## f) NEXT — up to 50, in execution order

**Finish Wave 2:**
1. M-15: decide gap vs E006 (see question 1) → implement handled-but-not-emitted rule (correctness c### or consistency d### slot) + fixture tests + self-suite run
2. M-15: register rule in `rules.AllRules()`, preset/help-text goldens (TestPresetHelpTextListsAllPresets will trip otherwise)
3. M-15: severity positioning (typo = Warning?) + docs line in cqrs-lint README if rules are listed there
4. M-16: catalog export carries coeffect/validation summary (render + gate + `#check-eventcatalog` run)
5. CHANGELOG `[Unreleased]` Added entry for the coeffect gate (M-11/12) — consumer-visible, currently undocumented
6. M-17: open ADRs 0114/0123/0124/0126/0127; verify mapping-doc claims; addendum corrections only
7. M-18: fetch arXiv 2608.25512 PDF (non-HTML URL); extract calculus + equivalence definitions; grounding addendum in mapping doc
8. M-19: recount modules (84 now!) + production DeferClose sites; correct via addendum + AGENTS.md module-count line
9. M-20: full `nix run .#verify-fast` to green (watch the turso flake); `#verify` if budget allows
10. M-20: doc-check final (refs should be ≥1038), api golden final, symbol gate final

**Wave 3:**
11. M-21: ADR-0137 design spike — health-driven engine deactivation (errorfamily storm → quarantine)
12. M-21: quarantine + reroute + auto-reprobe loop design; hysteresis interaction with `WithRoutingHysteresis`
13. M-22: error-family classification hook in the engine dispatch loop
14. M-22: quarantine state + reroute path in `store_routing.go`
15. M-22: auto-reprobe (reuse `ProbeEngine` loop) + tests with a failing engine
16. M-23: `Doctor` line + `GetEngineStats` health field + tests
17. M-24: rapid generator for randomized B-interleavings; property: A unchanged (generalize M-14)
18. M-25: `scenario.AssertUnchanged(projection)` DSL helper + tests + doc line
19. M-26: vocabulary positioning one-pager (INTERNAL until decided — see question 2)
20. M-27: Mermaid/D2 diagram into mapping report; optional html-report-kit render

**Follow-ups surfaced this session:**
21. `sqliteengine.ResetEngine` (8 `meta_*` tables + planned tables + matviews + `multiSeq`) — see question 3
22. EngineResetter on remaining persistent engines (pebble, bbolt, badger, pg, mysql, turso, duckdb, dgraph, iroh)
23. Surface reset capability in `Doctor`/`GetEngineStats`
24. TODO_LIST: delete completed M-06/M-07 entries; add EngineResetter ladder follow-ups; add turso-flake entry if M-20 sees it again
25. Tag-wave note: projectionadapter sibling replace + metaengine pin bump ride the next release train (scripts/tag-release.sh strips it — verified by design, smoke at cut time)
26. Consider `[Unreleased]`-position tripwire in verify-docs.sh (improvement #5)
27. Skill references: coeffect gate recipe (DomainConfig.Events usage) once M-16 lands — `core.md` conventions section
28. E006 and the new M-15 rule should cross-reference the system gate in their suggestions (static ↔ runtime story)
29. goleak for `metaengine` and `projectionhost` suites (M-08 covered system only)
30. M-14's `interleavedProjection` could graduate into `scenario` as exported test scaffolding if M-25 wants it shared

## g) QUESTIONS (cannot resolve myself)

1. **M-15 scope:** E006 already catches emitted-but-unhandled. For the reverse (projection consumes a type nothing emits — the actual typo class): hard **Error** severity (fail `cqrs-lint` runs), **Warning** default, or skip the rule because the runtime `system.New` gate (M-11) already covers composed systems?
2. **M-26 vocabulary:** default per plan is internal-only until v5. Confirm keep-internal, or approve public positioning now (SKILL.md cheat-sheet + `docs/DOMAIN_LANGUAGE.md` entries)?
3. **sqliteengine.ResetEngine priority:** implement inside this execution (before/within Wave 3), or leave as tagged follow-up after M-27? It unblocks "one-call revert" on the production-default engine but touches 8 system tables + matviews + `multiSeq` state — non-trivial risk surface.

— Session paused here per instruction. Working tree clean, all absorbed commits local-only.
