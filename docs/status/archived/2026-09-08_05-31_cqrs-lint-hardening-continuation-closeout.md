# cqrs-lint Hardening Continuation — Closeout Report

> **RESOLVED + ARCHIVED (docs-health pass 2026-09-08).** This batch closed
> session-3's items 1–3/6–13/24/47/50 (that report is annotated accordingly).
> Harvest done: the open remainder (f1 `IsQualifierFor` sweep, f2 doctor-JSON
> pre-merge ruling [BLOCKED], f3 ApplyLayout, f4/f5 F091 Tier 2 + F090(b),
> f7 fixture module, f8 stack/sqlite v4.3.1, f9 completeness meta-tests,
> f11 check-coverage, family audits, release/blocked items) lives in
> TODO_LIST → cqrs-lint / Release / CI. f6 (amend the T20/T21 report's
> T20-3 note) done inline by this pass. Struck below where closed.

**Date:** 2026-09-08 05:31 CEST
**Scope:** the continuation work queued by the session-3 closeout
(`docs/status/2026-09-07_23-10_cqrs-lint-hardening-session3-closeout.md`):
NEXT-50 items 1, 2, 3, 6–13, plus the two items that fell out of the gates
mid-run (24, 47, 50). Same box, parallel Turso session intermittently
active; `nix run .#verify` was again NOT runnable — every green claim below
is gate-scoped to what was actually run.

---

## a) FULLY DONE (implemented, tested, gated)

1. **Duplication gate verified — and it caught me (NEXT-50 #2).**
   `nix run .#check-duplication` on a clean tree: the session-3
   `//art-dupl:accept` placements in `benchkit/load_aware_test.go` and
   `system/load_aware_test.go` DO suppress (absent from the report). The
   gate was red for THREE new groups though:
   (a) MY OWN — `formatSeverityOverrides` in `cmd/cqrs-lint/doctor.go`
   (session-3 code, never verified) semantically cloned the
   collect-sort-keys loop in `metaengine/adttest/harness.go`;
   (b)+(c) FOREIGN — per-engine `calibration_constants_dump_test.go` (6
   engines) and `restart_safety_test.go` (3 engines), committed 2026-09-08
   AFTER the 2026-09-07 baseline pin (attribution via git log). Resolution:
   my group ELIMINATED at the root (`ids := slices.Sorted(maps.Keys(...))`
   — no directive needed, `sort.Strings` import retained for the other
   call site); foreign groups absorbed by `art-dupl baseline` re-pin on the
   clean tree (133 groups). `art-dupl check` exit 0. Learned: the nix app's
   dirty-tree guard on the baseline is wrapper-only — verified via the
   underlying `art-dupl check` command while the baseline awaited the
   daemon.
2. **S011 reclassification (NEXT-50 #1).** The conflated "Financial
   escalation split-brain fixed" CHANGELOG bullet split: comment drift +
   completeness meta-test stay under "Fixed"; the S011 escalation (a
   severity TIGHTENING — new findings for financial-domain consumers)
   moved to a new `### Changed — cqrs-lint S011 escalates for financial
   domains — 2026-09-07` section that names the governing Q3 question and
   the revert path. TODO_LIST's [BLOCKED] Q3 entry now states it governs
   S011. `check-changelog-symbols.sh`: 175 citations, honest, exit 0.
3. **V007 docs re-check (NEXT-50 #3).** `VALIDATION_REPORT.md` verified
   fine as-is (correctly labeled 2026-07-30 historical snapshot — no
   rewrite per the point-in-time-report rule). `V007-DEMO.md` extended
   with the dot-import capability (F090) using the exact finder strings
   from `v007.go` (message + suggestion verbatim; module normalization
   `storage/relational/v4` → `storage/relational` verified in
   `cqrsModuleOf`), dated honestly as shipping in the next minor.
4. **T20-3 — store resolution made deterministic.** Pass 1 of
   `detectFeatureSignals` iterated `pkg.Imports` (a Go map) and the
   `stack/*` branches overwrote `fp.Store` unconditionally: a package
   importing two store signals got a random store per run. Now: packages
   sorted by PkgPath, imports via `slices.Sorted(maps.Keys(...))`, and a
   single first-wins guard covering all stack presets + the `storage/`
   fallback. Pinned by `TestDetectFeatureSignals_MultiPresetStoreDeterministic`
   (40 runs must agree + exact winner `StorePostgres` by path sort — fails
   on old code with probability ≈ 1−2⁻³⁹). IMPORTANT DESIGN CORRECTION: the
   review report's fix note ("first-wins guard") is imprecise — a
   first-wins guard over map iteration is STILL random; sorted iteration
   is what makes it deterministic.
5. **T20-1 — store detection covers every shipped engine.**
   `metaengineEngineFromImport` gained mysql/badger/dgraph/turso/bbolt
   mappings. New `StoreKind` constants `StoreBadger` (embedded KV),
   `StoreDgraph` (distributed), `StoreIroh` (embedded P2P) with honest
   `IsSQL`/`IsEmbedded`/`IsDistributed` classification; `AllStoreKinds`
   extended (and its lying "sorted alphabetically" doc comment fixed).
   Engine→store switch completed for all ten engines; `a009`'s exhaustive
   suggestion switch got the three new kinds. BONUS ROOT-CAUSE FIX: the
   old `"memory"` engine case was DEAD CODE for real import paths
   (`.../metaengine/v4` contains `metaengine/`, so its guard could never
   fire) and contradicted its own doc comment — removed, core module now
   honestly maps to `""`. Pinned by
   `TestMetaengineEngineFromImport_CoversShippedEngines` (12 rows: 10
   engines + core + projectionadapter; fails on both drift directions).
   `isPersistentStore` (c017) needed nothing (negative logic).
6. **T20-4 — constructor handlers no longer poison type lookups.**
   `RegisterTyped(d, NewMyCommand(bus))` recorded its CALL TEXT as a
   `CommandTypesRegistered` key no struct-name lookup could match. New
   exported `analyzer.ConstructorHandlers` set + `constructorHandlerText`
   helper; `handlerTypeFromCall` down to two honest patterns. Pinned by
   `TestScanCallExpr_ConstructorHandlerRecordedSeparately`. DISCOVERED
   MID-TASK: `ExprString` elides call args (`NewCreateUserHandler(...)`)
   — the documented AGENTS.md gotcha — so the pin asserts semantics
   (record exists, prefix matches, zero call-text keys in the type map),
   not printer cosmetics.
7. **T20-5 — `CommandInfo.Embeds` split from `Fields`.** Embedded type
   expressions land in the new `Embeds` slice; `Fields` is names-only;
   B004 counts `Fields+Embeds` so its size heuristic is byte-identical in
   behavior. Pinned by `TestScanStructFields_SplitsNamesFromEmbeds`.
8. **T20-8 — alias-blindness closed (the batch's biggest correctness
   win).** New `analyzer.IsQualifierFor` (typed qualifier → import path;
   authoritative semantics; name fallback only without type info) and
   `analyzer.IsEventTypeParam` (+ bounded `typeFromEventPackage` walk
   through pointer/defined/alias layers). Adopted in all three remaining
   alias-blind helpers: `capturePayloadTypeFromVar` (now takes `gf`),
   fold-function detection (`looksLikeEventType` demoted to explicit
   fallback — C038/C040 alias-fold-blindness dead), and
   `IsInsideUpcasterClosure`. Six typed tests on the synthesized-`types.Info`
   harness: aliased import matches, shadowed qualifier rejected, name
   fallback preserved, consumer defined-type-over-event recognized,
   consumer package's own `events.Event` REJECTED (the old string
   heuristic false-positived on exactly that), missing-types fallback.
9. **T20-7 — upcaster-closure detection memoized.** `IsInsideUpcasterClosure`
   did a full `ast.Inspect` per candidate call (O(file)×queries for every
   A014/C005 check). Now: package-level `sync.Map` keyed by `*GoFile`
   stores per-file closure bodies (O(file) once per FILE, O(closures) per
   query), with the typed qualifier check folded into the collector.
   New `upcaster_test.go`: position semantics + cache stability (the
   memoized path is what production hits).
10. **Item 13 — doctor JSON shape golden, which immediately caught two
    REAL surface bugs.** New `TestDoctorJSONReport_Golden`
    (`testdata/doctor_json_report.golden`, `UPDATE_GOLDEN=1`, zero new
    deps — no go-snaps in this module). Catch #1: the `features` and
    per-module `profile` objects emitted GO-STYLE KEYS (`"Store"`,
    `"HasServer"`) — `analyzer.FeatureProfile` had NO json tags while
    every other field on the surface is camelCase. All 14 fields now carry
    camelCase tags (fixed BEFORE the surface ships in a tag — this is the
    consumer-script-breaking class). Catch #2: `report.Modules` was built
    by ranging a map — nondeterministic JSON ordering (same class as
    T20-3/T21-3); now sorted by module dir. Golden pins the shape with
    volatile values (paths, rule counts) stamped to constants.
11. **Item 24 — RULES.md vs formatter ROOT CAUSE CORRECTED.** The
    freshness test broke AGAIN mid-session and the closeout's attribution
    ("treefmt pads the tables") is WRONG: treefmt has NO markdown
    formatter. The re-padded tables come from `dprint.json`'s markdown
    plugin (the daemon/BuildFlow reformat layer). `**/RULES.md` is now
    excluded in dprint.json next to `**/CHANGELOG.md`, RULES.md
    regenerated to the generator's minimal-table output, and
    `TestRULESMD_Fresh` is green — and should now STAY green.
12. **Item 47 — verified, not fixed.** The "stray TestX placeholder" from
    the closeout is INSIDE a fixture source string of the real test
    `TestV007_SkipsTestFiles` (dot-import test file content). The closeout
    claim was a misread; nothing to change. (See section d.)
13. **Item 50 — AGENTS.md gotcha recorded** at the end of Tooling &
    Build: generated markdown must be excluded at BOTH formatter layers,
    the dprint-not-treefmt correction, and the generalized lessons
    (identify the actual mutating layer; completeness meta-tests where
    docs and code co-name IDs).
14. **golangci self-heal fired again (4th incident).** After the batch,
    lint went red with 10 `gci` findings — the daemon had re-added `gci`
    to `.golangci.yml` `formatters.enable` yet again. `nix run
    .#check-lint-config` REPAIRed it in place; lint 0 issues after. The
    self-heal loop works; it is also a treadmill (Daemon Q2 remains open).
15. **Ledgers.** TODO_LIST: F091 entry's REMAINING clause updated (T20-8
    adoption done, Tiers 2–3 remain), new comprehensive hardening-batch
    entry, Q3 entry cross-linked to S011. CHANGELOG: new
    `### Fixed — cqrs-lint analyzer hardening: deterministic detection,
    engine coverage, doctor JSON — 2026-09-08` section. API golden
    regenerated twice (6733 → 6735 exports: three StoreKind constants,
    `ConstructorHandlers`, `IsQualifierFor`, `IsEventTypeParam`, the
    `Embeds` field).

**Gates actually run and green:** cmd/cqrs-lint full `go test ./...`
(18 packages, exit 0, run twice after each batch), golangci-lint 0 issues
(after gci self-heal), self-lint `--strict-load
--fail-on-stale-suppressions` exit 0, `art-dupl check` exit 0 (baseline
133 groups), `check-changelog-symbols.sh` exit 0 (175 citations), doc-check
1240 references / 63 packages (canonical invocation), api-stability
`--update` (6735 exports), RULES.md freshness + doctor JSON golden green,
analyzer/api/boilerplate/rules package tests green per-task.

## b) PARTIALLY DONE

1. **NEXT-50 #26 (post-change gates).** Only `check-duplication` of the
   three was run. `check-coverage` and `check-file-size` did NOT run —
   file-size will be red regardless pending the owner's 350-line policy
   (Q2), but coverage is unverified for this batch.
2. **Hygiene item 44 (doctor JSON pre-merge config).** I sorted `modules`
   and golden-locked the surface, but did NOT resolve whether
   `doctor --format json` should run `applyConfigOverrides` — the JSON
   path still reports raw (pre-merge) config in places the text path shows
   merged. The golden now pins whatever is chosen, so the fix is now
   cheap; the ruling is missing.
3. **T20/T21 review report accuracy.** The report's T20-3 fix note still
   says "the same first-wins guard" — imprecise (sorted iteration is the
   determinism, the guard alone is not). The correct design lives in the
   TODO_LIST batch entry and the code comments; the report itself was not
   amended.
4. **F091 Tier 2 / F090(b)** — the machinery is now COMPLETE (typed
   qualifier + typed event-param + per-file caches all shipped), but the
   C008 confirmation and typed dot-import attribution remain zero
   implementation (as in session 3).
5. **Alias-blind adoption sweep.** `scanCallExpr`'s OTHER qualifier
   comparisons (`pkgName == "system"`, `== "catalog"`, `== "decider"`,
   `== "event"`) are still name-based. `IsQualifierFor` makes them all
   adoptable cheaply, but only the three T20-8 targets were converted.

## c) NOT STARTED (untouched this session)

1. **NEXT-50 #12 — ApplyLayout rule implementation + fixtures.** Design
   pass complete (appendix in the T23 design doc), 0.5-day estimate,
   unblocked — deliberately scoped out to avoid rushing a whole new rule
   at the end of the context budget rather than half-ship it.
2. NEXT-50 #14–#23: replace-based end-to-end fixture module for typed-path
   rules; the `stack/sqlite` published-pin breakage issue (file it);
   completeness meta-tests for `consumerOnlyRules` + preset disable lists;
   remaining family audits (S001/rules.go spot-check remainder, V, T, E,
   D, B, A×2).
3. NEXT-50 #25: `nix run .#verify` end-to-end on a quiet box (load ≤ 15).
4. NEXT-50 #27–#34: release prep fold-in, Q3, 350-line policy, Daemon Q2,
   F040, tag wave, GitHub Releases, indirect-dep sweep — blocked/pending
   as before.
5. Hygiene items #35–#43, #45–#46, #48–#49: all untouched (resolveMinSeverity
   source flag, sortedBreakdown tie-break, pathDepth clarity, health-route
   literal scan, bare-ctx-alias guard, isOOAggregate boundaries,
   scanConstDecl multi-value, capturePayloadType arg priority,
   trackVarAssignments scoping, parallel loads, findGoModDirs worktrees,
   load-helper synthesized-loadavg test, the last inline C025 in init.go).

## d) TOTALLY FUCKED UP (honest self-assessment)

Nothing destructive or dishonest, but a real list:

1. **Session-3's own code created a new dup clone group** that I shipped
   without running the gate (the e-9 lesson from THAT session, repeated
   once). The gate caught it; I fixed it at the root. The rule "run the
   dup gate in the same task as new helper/table code" exists precisely
   because I keep deferring it.
2. **Two false claims propagated from the session-3 closeout, both mine to
   catch:** (a) "stray TestX placeholder" — fixture content, no such test;
   (b) "treefmt reformats RULES.md" — it was dprint. I caught (b) only
   because the freshness test broke mid-session; (a) I caught by reading
   the file before "fixing" it. Lesson: re-verify report claims against
   the artifact BEFORE acting on them, even (especially) my own reports.
3. **The T20-3 fix description in the review was technically wrong**
   ("first-wins guard" alone does NOT kill map-order randomness). I
   corrected it in implementation, but wrote the imprecise note in the
   first place.
4. **Tool-discipline slips, all caught and corrected:** one multiedit with
   old_string/new_string SWAPPED (the test-insert edit); two
   read-before-edit rejections (bash `sed` output does not satisfy the
   read requirement — types.go, b004_b008.go); a duplicate
   `case *ast.CallExpr` build error from an edit matching the wrong
   instance; three unused-variable build fails in the typed test file
   (sloppy single-pass writing); one wrong doc-check invocation (wrong
   cwd/module path — "outside main module") costing a cycle.
5. **The doctor JSON golden initially PINNED THE BUGGY OUTPUT** — I
   regenerated before noticing the Go-style keys, then caught it by
   reading the generated golden. The save was reading what the test had
   just written; the failure was not inspecting the first marshal before
   baselining it.
6. **The fixture for the golden was unrealistic on the first pass**
   (zero-value `Domain`/`Monetary`), which would have pinned
   unrepresentative state; fixed to `unknown` kinds before the final
   golden.
7. **Context-budget honesty:** I scoped ApplyLayout (item 12) out instead
   of starting a rule I could not finish cleanly. Correct call, but it
   means the 6–13 batch is 7 of 8, not 8 of 8.

## e) WHAT WE SHOULD IMPROVE (process + code, grounded in this session)

1. **"Run the gate in the same task" must include art-dupl for ANY new
   helper/loop/table code** — my doctor.go clone was 5 lines of innocent
   idiom. The accept-or-refactor decision belongs in the change, not in a
   later gate run.
2. **Re-verify prior reports' claims at the artifact before acting** —
   two of the closeout's claims were wrong and one of them (treefmt) had
   already shipped into NEXT-50 item 24's description and nearly caused a
   wrong fix (formatter config surgery at the wrong layer).
3. **A new JSON/CLI surface ships WITH its tags and its golden** — the
   doctor JSON surface from session 3 had neither. Shape tests are not a
   follow-up; they are part of the surface.
4. **Read the first output of a generator before baselining it** — the
   golden-regen-then-inspect order cost nothing this time only because I
   did inspect.
5. **Write test files in one complete pass** — three unused-var build
   fails came from incremental assembly.
6. **The self-heal loop (gci) is now at 4 incidents** — it works, but the
   daemon is the root cause; Q2 needs the owner, and each recurrence
   wastes a lint cycle.
7. **Baseline re-pin with git-log attribution worked well** — doing it
   again: attribute every absorbed group before pinning, so the baseline
   never silently eats MY diff.
8. **`IsQualifierFor` adoption should be swept, not sprinkled** — the
   remaining name-based qualifier checks in `scanCallExpr` are the same
   bug class T20-8 fixed; a single adoption pass beats three future
   findings.
9. **Golden fixtures should be realistic from commit one** — stamped
   volatile values are the pattern; zero-value enums are not.

## f) NEXT 50 (prioritized, 1 = first)

**Finish this batch's direct follow-ups**

1. Adopt `IsQualifierFor` in the remaining `scanCallExpr` qualifier
   checks (`system`, `catalog`, `decider`, `event`) — same class as
   T20-8, machinery already shipped.
2. Item 44: decide + implement doctor-JSON pre-merge semantics
   (`applyConfigOverrides`) and re-pin the golden.
3. Item 12: ApplyLayout rule implementation + fixtures (design done,
   0.5 day).
4. Item 4: F091 Tier 2 — C008 payload-flow confirmation behind
   `--typed-info=auto`.
5. Item 5: F090(b) — typed dot-import attribution on the Tier-1 machinery.
6. ~~Amend the T20/T21 review report's T20-3 note (sorted iteration is the
   determinism, not the guard alone).~~ done 2026-09-08 (docs-health pass
   amended the archived T20/T21 report's T20-3 heading with the correction)
7. Item 14: commit a replace-based end-to-end fixture module for
   typed-path rules (schema/v4-based).
8. Item 15: file the `stack/sqlite` published-pin breakage
   (`sqlopt` → `undefined: storage.SQLiteSetSynchronous`) as a tracked
   issue.
9. Item 16: extend the completeness-meta-test pattern to
   `consumerOnlyRules` and preset disable lists.
10. ~~Confirm `**/RULES.md` survives the next daemon commit cycle (dprint
    exclude should hold — verify once, then stop watching it).~~ closed by
    mechanism: `**/RULES.md` + `**/CHANGELOG.md` are dprint-excluded and the
    freshness meta-test is the standing verifier (no manual watching needed)
11. Run `nix run .#check-coverage` for this batch (item 26 remainder).
12. Single-source the engine-name list: `metaengineEngineFromImport` and
    the engine→store switch duplicate the ten engine strings; the table
    test pins drift, but a data table would remove the pair.
13. Doctor JSON: decide nil-vs-`[]` for `metaengineEngines` on the public
    surface (json/v2 renders `[]`; golden pins `[]` — confirm intended).
14. severityFloor renders `""` when neither preset nor config sets a
    floor (visible in the golden) — render `"info"` or document.
15. Mention the doctor JSON golden regen (`UPDATE_GOLDEN=1`) in
    CONTRIBUTING next to the RULES.md regen instructions.

**Family audits (T13–T19 remainder, unchanged order)**
16. S-family remainder: `rules.go` (S001) line-by-line (item 17).
17. V-family audit (7 rules) (item 18).
18. T-family audit (8 rules) (item 19).
19. E-family audit (17 rules) (item 20).
20. D-family audit (19 rules) (item 21).
21. B-family audit (31 rules) (item 22).
22. A-family audit (34 rules, 2 waves) (item 23).

**Hygiene (items 35–49, unchanged)**
23. resolveMinSeverity doctor source-attribution flag (#35).
24. sortedBreakdown name tie-break in health.go (#36).
25. pathDepth clarity simplification (#37).
26. health-route literal scan → handler-registration calls (#38).
27. handlerTypeFromClosure bare-ctx-alias guard (#39).
28. isOOAggregate word-boundary match (#40).
29. scanConstDecl multi-value ValueSpec (#41).
30. capturePayloadType index-4 priority (#42).
31. trackVarAssignments scope-awareness or documented FP class (#43).
32. Per-module parallel loads for wall-time (#45).
33. findGoModDirs `.worktree*`/example-worktree exclusions (#46).
34. benchkit/system load-helper synthesized-loadavg unit test (#48).
35. Last inline C025 in init.go — wrap a sentinel (#49).

**Verification & release**
36. `nix run .#verify` end-to-end on a quiet box, load ≤ 15 (#25) — bank
the first repo-wide GREEN covering sessions 3 + 4.
37. `nix run .#check-file-size` after the Q2 policy lands (#26 remainder).
38. Release prep: fold the analyzer hardening + S011 Changed entry +
doctor-JSON key fix into the next cqrs-lint minor (#27).
39. Post-tag: CHANGELOG callout that doctor JSON feature/profile keys
changed case (consumer scripts).
40. SKILL.md/references sweep: confirm no consumer doc quotes the old
Go-style doctor JSON keys.
41. After the daemon commits the two remaining dirty files
(.golangci.yml self-heal, CHANGELOG): `go build` sanity re-check.

**Blocked (owner input required — do not start alone)**
42. Q3 ruling — now governs S011 AND the doctor-JSON key rename riding
the same minor (#28).
43. 350-line gate policy (#29).
44. Daemon Q2 — gci re-add recurred a 4th time (#30).
45. F040 branch protection (#31).
46. Next v4 tag wave (#32) — the analyzer API additions
(StoreBadger/StoreDgraph/StoreIroh, ConstructorHandlers,
IsQualifierFor/IsEventTypeParam, Embeds) should ride it or wait for
Tier 2 (see questions).
47. GitHub Releases for outstanding tags (#33).
48. Indirect-dep consolidation after the wave (#34).
49. go-finding #27/#28 upstream fixes, then delete any local workarounds.
50. Revisit "load-aware helpers consolidation into testutil" if either
lean module touches testutil anyway (session-3 e-6, unchanged).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Release shaping for the pending cqrs-lint minor:** the accumulated
   unpublished surface now includes F089 (presets/overrides), V007
   dot-import + typed resolution, the S011 "Changed" escalation, AND this
   batch's analyzer API additions + the doctor-JSON key-case fix. Ship it
   all as one CHANGELOG-led minor at the next tag wave, or split the
   S011 tightening off per Q3? (Q3's scope question, now concrete.)
2. **Doctor JSON pre-merge ruling (item 44):** should
   `doctor --format json` report RAW config (today's behavior, now
   golden-pinned) or EFFECTIVE post-`applyConfigOverrides` values where
   the text path already shows merged? I can implement either in minutes
   with the golden; I cannot know which contract consumers should script
   against.
3. **Daemon Q2, again with evidence:** the gci re-add recurred a 4th time
   this session and the self-heal repaired it a 4th time. Keep the
   self-heal loop as the permanent answer, or is fixing/disabling the
   daemon's golangci formatter step worth the upstream effort now that we
   have four data points?
