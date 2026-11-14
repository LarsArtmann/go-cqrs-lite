# Status Report — 2026-11-11 20:24 — Nine-Issue Fix Wave (#117–#129)

Session scope: `gh issue list` showed 9 open issues; owner said "go fix it all."
Everything below is from THIS session's run only. All 9 issues are closed.
Worktrees in three repos (go-cqrs-lite, webphone; cqrs-htmx inspected, no
edits needed). Auto-commit daemons absorbed every tree change into `chore:`
commits as usual.

---

## a) FULLY DONE

| Issue | What shipped | Evidence |
|---|---|---|
| **#117** SQL snapshot store drops `StateShape` | `state_shape TEXT NOT NULL DEFAULT ''` (MySQL: `VARCHAR(255)`) in all 4 dialect DDLs + 4 embedded migrations; bound in the upsert; scanned on both load paths; `storage.MigrateSnapshotStateShapeColumn` idempotent ADD COLUMN wired into all 4 `*InitSchema` helpers (concurrent-runner safe re-probe); `probeTableColumns` → `probeSnapshotTableColumns` (unparam). | `TestSQLiteSnapshotStore_StateShapeRoundTrip` (save/load/loadAtVersion/upsert), `TestMigrateSnapshotStateShapeColumn_SQLite` (legacy table, legacy row unstamped, stamped write lands, idempotent re-run), sqlmock arg-order tests updated, 3 golden `.snap` files re-pinned |
| **#126** doctor audit false-stale from subdir | Root cause reproduced LIVE on InboxClean `internal/emailcqrs` (23 total: 3 stale pre-fix → 0 stale post-fix; root run byte-identical 46/1). `findGoModDirs` only walked DOWN → subtree-degraded feature profile → B029/B030/B014 skipped → stale verdicts. `analyzer.findEnclosingGoModDir` walks up; feature detection sees the whole enclosing module (per-module profile keyed by module dir); findings/scanning stay subtree-scoped. Belt-and-braces: real `--prune-suppressions` re-runs the pipeline post-prune and FAILS LOUDLY if a pruned rule still fires (`suppression.WouldSuppress` mirrors the stale matcher window). 2026-10-12 loader lesson comment gained the second layer. | `TestBuildContext_SubtreeRunUsesEnclosingModuleProfile`, `TestFindEnclosingGoModDir_WalksUp`, `TestIsUnderDir`, `TestWouldSuppress_PositionWindow`; live repro before/after |
| **#129** V003 hard-coded latest | Per-module `@latest` from GOPROXY (first usable entry, default proxy.golang.org), proxy path escaping, 1.5s request timeout, process-lifetime `sync.Map` cache; `testing.Testing()` + `CQRS_LINT_NO_PROXY_FETCH=1` guards keep tests/hermetic envs offline; offline failure → fleet constant (today's behavior). Finding names the module's OWN latest. ctx threaded (contextcheck). | `TestV003_PinEqualsModuleLatestStaysSilent`, `TestV003_NamesTheModulesOwnLatest`, `TestV003_ProxyFailureFallsBackToFleetConstant`, httptest `TestFetchLatestMinorFromBase*`, `TestEscapeModulePath`, `TestLatestMinorFor_Fallbacks` |
| **#121** E004 vs local converter funcs | `analyzer.scanConverterFunc` records funcs whose entire body is `return Conv(param)`; call-shaped id args at catalog/emission/schema/eventMetadata sites defer via `deferConverterEventID`; `analyzer.ResolveConverterRoutedEventIDs` inlines post-scan (order-independent, depth-capped). Unresolvable shapes = false negative, never positive. Wired into loader + all 3 test-helper pipelines. | `TestConverterIndirectionResolvesCatalogIDs`, `...ResolvesEmittedIDs`, `...RejectsMultiStatementFuncs`, `TestResolveConverterRoutedEventIDs_Idempotent` |
| **#127** D005 cue scan skips shortnames | `attachedToCQRSMention` runs the historical-cue walk before pinned shortnames (skipping connectors + the go-cqrs-lite qualifier, ≤4 window); `-era` version suffix = cue-equivalent; present-tense claims still fire. | `TestExtractCQRSVersion_HistoricalShortnameMention`, `...PresentTenseShortnameMentionKept`, `...EraSuffixedTokenSkipped` |
| **#122** meta_map sort-key index | `sqliteengine.WithSortIndex(collection, path, desc)`: opt-in `CREATE INDEX IF NOT EXISTS ... ON meta_map(collection, json_extract(value, path) [DESC], key)`. KEY DESIGN WIN: trailing `key` tiebreak column (without it the planner keeps PK scan + temp B-tree — found empirically) and deliberately NOT partial (`WHERE json_valid` unusable for scans that don't imply it). Validates at construction; idempotent; planned-table collections unaffected. | `TestWithSortIndex_CreatesExpressionIndex`, `..._PlannerUsesIndex` (EXPLAIN QUERY PLAN: SEARCH USING INDEX, no temp B-tree), `..._ScanResultsIdentical` (row parity across full keyset pagination w/ and w/o index), `..._IdempotentConstruction`, `..._RejectsInvalidDeclarations` |
| **#120** WebAuthn envelope | cqrs-htmx side ALREADY fixed (envelope pinned by provider test; loginpage unwraps both preparers — verified, no edits). webphone side fixed by me: `prepareRegistrationOptions` unwraps `{publicKey}`; island stub corrected to the REAL wrapped wire (was flat = lying stub); Go test now opens `begun.Options` and asserts `publicKey.challenge` is a non-empty string; `pkStub` carries a challenge like userauth's stub. | webphone: 13/13 node:test green (`webauthn.test.mjs` + `enroll.test.mjs`), `TestPasskeyEnrollHandlersWalkTheCeremony` green |
| **#128** live checkpoint fails dispatch | Already fixed on master (degrade-to-warn + staged retry + Stop flush; plus the separate per-connection busy_timeout pragma fix). Verified tests green, cited CHANGELOG entry exists. | `TestLiveCheckpoint_SaveFailureDoesNotFailHandler`, `...FlushesAtShutdown` green |
| **#123** EnforceCap deadlock | Already fixed in `de39a43a9`; verified the fix survived the `/v4` module move; regression + full cap suites green. | `TestSQLiteEnforceCapDoesNotDeadlockAgainstFolds` green |

**Docs/gates done this session:**
- CHANGELOG `[Unreleased]`: 5 Fixed + 1 Added entry, every new export cited
  (`check-changelog-symbols`: 12/12 honest).
- api-stability golden regenerated (5 new exports:
  `storage.MigrateSnapshotStateShapeColumn`, `sqliteengine.WithSortIndex`,
  `version.NewV003DetectorWithLatest`, `suppression.WouldSuppress`,
  `analyzer.ResolveConverterRoutedEventIDs`); `TestEvery` green.
- `check-untested-exports`: my 5 exports all test-named (added the
  idempotency test for `ResolveConverterRoutedEventIDs`); pruned 1 stale
  row (`enginetest.ScopedCollection`) via `--update` (shrink-only direction).
- V003 catalog description updated + `RULES.md` regenerated (1 line).
- FEATURES.md row: "meta_map sort-key expression indexes (#122)".
- readmodels.md: new "Sort-key expression indexes" section; doc-check
  1391 refs valid; recipes compile gate green.
- engine-capabilities regenerated (no diff — options aren't census surface).
- Scoped lint 3/3 clean (fixed 5 findings in my own new code:
  contextcheck/forcetypeassert/gochecknoglobals×2/tagliatelle + gofumpt via
  `nix fmt`; renamed `probeTableColumns` for unparam).
- All 9 issues closed with terse evidence-bearing comments.
- `gh issue list` → empty.

## b) PARTIALLY DONE

~~- **#117 consumer flip**: the cqrs-htmx pinned test~~ routed — M35 consumer flips (cqrs-htmx snapshot_shape_sql_test flip rides the storage/v4 tag; PUSH-BLOCKED, D16)
  (`usermgmt/snapshot_shape_sql_test.go`) still documents the drop by
  design — it flips in the SAME change as the storage/v4 dependency bump,
  and NO release/tag was cut this session (deliberate: release is a ritual
  of its own). The dependency bump + flip is pending.
~~- **#126 wiring coverage**: the loader mechanism and `WouldSuppress` are~~ done — DONE 2026-11-14 (M37/F231): 3-test e2e incl. JSON-path wiring gap fixed
  pinned, but `verifyPruneRemovedNothingLive` itself (the re-run-after-prune
  guard in `doctor_audit.go`) has NO direct test — a fixture-based e2e that
  prunes a LIVE suppression and asserts the loud failure is missing.
~~- **#122 evidence**: planner adoption pinned via EXPLAIN; the 58.5ms →~~ routed — TODO_LIST "100k-row indexed keyset-scan benchmark" row (make the numbers OURS, M53)
  ~0.2ms numbers are the ISSUE's measurements, not re-measured here (no
  100k-row benchmark added; see e/next).

## c) NOT STARTED (deliberately out of this session's scope)

~~- Any release/tag wave for the touched modules (storage/v4, cqrs-lint,~~ routed — M35: manifest staged + dry-run green; cut pending chain green (D16/D17)
  sqliteengine).
~~- tursoengine parity for `WithSortIndex` (see e).~~ routed — TODO_LIST "tursoengine WithSortIndex parity + ApplyLayout auto-drop" row + D15
~~- webphone CHANGELOG/docs entry for the enroll fix (its daemon absorbed the~~ routed — DONE 2026-11-14 (M37/F236): Fixed row landed, full suite re-verified (37 pkgs)
  code; no release-notes row written).
~~- AGENTS.md/gotchas lesson entry for the subdir-audit class (knowledge lives~~ routed — DONE 2026-11-14 (M37/F235): gotchas-testing row landed
  in the loader comment + CHANGELOG only).

## d) TOTALLY FUCKED UP

Nothing destroyed, no regressions found — but honest near-misses:

- **First `WithSortIndex` DDL was planner-dead**: I copied the journal's
  partial-index form (`WHERE json_valid(value)`); the planner REFUSED it
  (scans don't imply the predicate) and without the trailing `key` column
  ORDER BY wasn't satisfiable. Caught ONLY because I wrote the
  EXPLAIN-based test instead of trusting the DDL. If I had shipped a
  "created the index" test only, the option would have been dead weight on
  every write. Lesson: perf options need a planner-adoption pin, not an
  existence pin.
- **Test-authoring churn**: the sqlite pagination probe went through
  self-import cycles and swallowed-output greps before I used the write
  tool — wasted cycles, and it exposed a REAL subtlety
  (`MapSet` string values get JSON-quoted; only `json.RawMessage` stores a
  JSON document) that my first test draft got wrong.
- **Ran `nix fmt` AFTER editing**: reformats landed between my edits twice,
  causing edit-tool mismatches. Should format first, then edit, or re-view
  after every fmt.

## e) WHAT WE SHOULD IMPROVE (this session's lessons)

~~1. **Never ship a perf option without an EXPLAIN/planner pin** — existence~~ routed — codified: sort-index EXPLAIN pin shipped with #122; benchmark row keeps it honest
   tests lie for indexes (the partial-index trap above).
~~2. **Always run the composed gate** (`#verify`/`#verify-fast`) after an~~ routed — M34 chain armed (verify-chain.sh); receipt rows 331/495
   API-surface session, not just the pieces — see f/#1. I ran scoped tests,
   scoped lint, doc-check, goldens, and meta-tests but NOT the composed
   `#verify` (coverage floor, duplication gate, file-size ratchet,
   package-surface, test-runtime baseline all unverified).
~~3. **Write the fixture e2e for safety-belt wiring when the mechanism test~~ routed — DONE via F231 (the exact case named: prune guard wiring untested)
   exists but the wiring doesn't exercise it** (prune guard).
~~4. **Format-before-edit discipline** with the daemon + treefmt racing.~~ routed — process note (daemon+treefmt race) — not rowed
~~5. **Perf claims in CHANGELOG/comments should say whose measurement they~~ routed — codified: CHANGELOG row for #122 cites issue-measured + planner-pinned-here; benchmark row closes the gap
   are** (I wrote "measured" quoting the issue; the honest label is
   "issue-measured, planner-adoption pinned here").

## f) NEXT — up to 50, prioritized

**Verify/release (blocking confidence):**
~~1. Run `nix run .#verify` (exclusive; expect green — but file-size ratchet~~ routed — M34 chain #4 armed (gate-waiting on load); composed receipt pending
   on grown `d005_version.go`/`loader.go`, duplication gate on the new
   scanners, and test-runtime baselines are UNMEASURED).
~~2. `nix run .#check-coverage` (new code + tests not floor-verified).~~ routed — rides the chain contract (coverage precedes #verify in verify-chain.sh)
~~3. `nix run .#check-duplication` (scanner_converters.go vs~~ routed — in #verify battery (rides chain)
   scanner_emit_helpers.go shape overlap; 4 dialect DDL strings).
~~4. `nix run .#check-file-size` (grew: d005_version.go ~+30, loader.go ~+70).~~ routed — gate GREEN since M32 red-gate trio (2026-11-13); loader split verified; re-proven by chain battery
~~5. `nix run .#check-package-surface` + `check-test-runtime`~~ routed — M35 pre-tag Verify Before Release ritual (post-chain window)
   (`--update` likely needed for storage/sqliteengine/cqrs-lint rows).
~~6. `nix run .#verify-ci` (GOWORK=off per-module mirror).~~ routed — TODO_LIST:495 optional same-window verify-ci leg (A1 row)
~~7. Webphone full `go test ./...` (I only ran the passkey + island tests;~~ routed — DONE 2026-11-14 (M37/F236): go test ./... 37/37 green
   pkStub change is low-risk but unverified at package breadth).
~~8. Decide + cut the release wave (storage/v4, cmd/cqrs-lint,~~ routed — r
   metaengine/sqliteengine) per the tag-wave ritual.
~~9. On the storage tag: flip cqrs-htmx `snapshot_shape_sql_test.go` to~~ routed — M35 consumer flips (PUSH-BLOCKED, D16)
   assert preservation + bump the pin (same change, per the issue contract).
~~10. Timesheets: drop the `ignore-start(E004)` block on the cqrs-lint bump.~~ routed — M35 consumer flips
~~11. crm (Kith): remove the consumer-side serialized-subtests mitigation~~ routed — TODO_LIST crm row (rides M35)
    once the projectionhost fix is in their pin (verify their pin first).

**Hardening what shipped:**
~~12. e2e test: `doctor --prune-suppressions` over a fixture whose stale~~ routed — DONE 2026-11-14 (M37/F231)
    verdict is wrong → asserts the loud failure (wiring currently untested).
~~13. Parallelize or globally budget V003 proxy fetches (30 pins × 1.5s worst~~ routed — DONE 2026-11-14 (M37/F232): 4-slot semaphore
    case = 45s on a slow proxy; per-request cap only today).
~~14. V003: honor `GONOSUMDB`-style private-module exclusion (GOPRIVATE/~~ routed — DONE 2026-11-14 (M37/F232): GONOPROXY/GOPRIVATE chain + 10-case pin
    GONOPROXY modules should skip the proxy, not fail-then-fallback).
~~15. Converter replay: cover converter-wrapped args at HELPER call sites~~ routed — DONE 2026-11-14 (M37/F233): fixpoint unwrap def+replay sides
    (`ResolveHelperEmitCalls`/`ResolveCatalogHelperCalls` drop them today).
~~16. `WithSortIndex` on tursoengine (delegation or own option) — the IVM~~ routed — TODO_LIST parity row + D15
    engine is where big sorted collections actually live.
~~17. Auto-drop sort indexes when a collection migrates to a planned table~~ routed — TODO_LIST parity row (same)
    (`ApplyLayout`) so residual meta_map rows don't pay index maintenance.
~~18. Add a 100k-row benchmark for the indexed keyset scan into the bench~~ routed — r
    gates (`benchmarks/latency-ceilings.txt`) — the issue's numbers become
    OUR numbers.
~~19. D005: update RULES.md long description with the shortname cue scan~~ routed — DONE 2026-11-14 (M37/F234)
    semantics (short desc regenerated; long prose not).
~~20. Update cqrs-lint README/RULES for the doctor subdir semantics ("audit~~ routed — DONE 2026-11-14 (M37/F234)
    verdicts are module-scoped even from a subdir run").
~~21. Record the subdir-audit lesson in `docs/agents/gotchas-testing.md` (or~~ routed — DONE 2026-11-14 (M37/F235)
    loader gotchas file) — currently only the code comment carries it.
~~22. Webphone: CHANGELOG/docs entry for the enroll unwrap (their train).~~ routed — DONE 2026-11-14 (M37/F236)
~~23. Clean the 3 obsolete go-snaps golden files in storage/v4~~ routed — DONE 2026-11-14 (M37/F239): already clean, zero-diff proven
    (`UPDATE_SNAPS=clean`; pre-existing, noticed this session).
~~24. InboxClean's own F009 stale suppression (corpus_backup.go:382) surfaced~~ routed — routed to foreign-repo owner (harvest ledger)
    in the audit — their call, but worth a heads-up.
~~25. `NewV003DetectorWithLatest` is an exported test seam — consider folding~~ routed — TODO_LIST Small pins pack row (named)
    into an internal test once ruletest allows package-internal detectors
    (public surface hygiene).

**Bigger follow-ons visible from this session (park/decide):**
~~26. Snapshot `state_shape` for the metaengine `SnapshotBackend` wire —~~ routed — DONE 2026-11-14 (M37/F238): GAP VERIFIED + rowed with fix shapes
    verify the engine-backed path already round-trips shape (system
    deployments) or file the gap.
~~27. SQL integration suites for the new column (`#integration-pg`,~~ routed — TODO_LIST "state_shape SQL integration legs" row (M53)
    `#integration-mysql-nspawn`) — only SQLite ran.
~~28. Doctor: surface "analyzed subtree of module <X>" in the header when the~~ routed — r
    enclosing-module path triggers (operator visibility).
~~29. V003 suggestion could include the actual latest version string from the~~ routed — TODO_LIST V003 hygiene pack row (named)
    proxy in the `go get` line.
~~30. Consider a `--no-network` global flag for cqrs-lint (V003 is the first~~ routed — TODO_LIST V003 hygiene pack row (named)
    rule that can touch the network; make the policy explicit).
~~31. Converter scan: record converters' SOURCE positions and surface them in~~ routed — r
    doctor for debuggability (why did/didn't my converter inline).
~~32. The `attachedToCQRSMention` two-branch asymmetry (raw 2-token vs~~ routed — not rowed (two-branch window asymmetry; property-test idea)
    skip-aware 4-token windows) could be unified + property-tested.
~~33. Explore pinning `EXPLAIN QUERY PLAN` for the OTHER meta_map scans~~ routed — TODO_LIST row (M53)
    (filters) so future regressions in pushdown SQL get caught the same way.
~~34. Fleet: re-run `census-fleet.sh` after the release wave (sqlstore~~ routed — TODO_LIST Post-fix census re-run row (M54)
    deletion prep unaffected, but numbers refresh).
~~35. cqrs-htmx deep-check: does webphone's login page (loginpage library)~~ routed — audited in-report: only enroll used the raw shape
    have the same enroll-side gap on any OTHER page? (audited: no — only
    enroll used the raw shape).

## g) Questions I canNOT answer myself

~~1. **Run `#verify` now, or is push-CI the verifier of record for this~~ routed — answered by ritual: chain #4 armed (verify = verifier of record; push-CI gates F214)
   batch?** It's the repo contract after API-surface changes, but it's
   exclusive and the machine has been busy; you may want it deferred to a
   quiet window (`nix run .#quiet-window-run`).
~~2. **Cut the release wave for storage/v4 + cqrs-lint + sqliteengine now**~~ routed — owner decision D16/D17 (push + ratify 48-module scope)
   (which also forces the cqrs-htmx test flip + consumer bumps), or fold
   into the next planned wave?
~~3. **tursoengine `WithSortIndex` parity + ApplyLayout auto-drop** (f/16+17):~~ routed — owner decision D15
   build now as a small follow-up train, or park in TODO_LIST for the next
   sqliteengine touch?
