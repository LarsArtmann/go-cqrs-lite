# Status: Issues #130 (fold misrouting) + #131 (cell-versioning config surface) — implementation session

Report written 2026-11-13 20:22 CET. Session started ~19:30 CET from `gh issue ls`
(issues #131, #130 open, both with draft docs already in the tree).

## Scope — what this report VERIFIED, and what it did NOT

**Verified by command output during this session:**

- metaengine/v4 package suite: `GOWORK=off go test . -count=1` → ok (48s, full
  package, includes the new slice-of-unit specs)
- Targeted new-test runs green in metaengine/v4, sqliteengine/v4,
  pebbleengine/v4, bigtableengine/v4, tursoengine/v4, system/v4, systemtest/v4
  (`-run 'CellVersioning|GCPolicy|RetentionPolicy|RejectCell|On'` per module)
- All 11 touched modules compile (`GOWORK=off go build ./...` each)
- Gates green at least once this session: `check-untested-exports`,
  `check-error-taxonomy-tests`, `check-engine-capabilities`,
  `check-changelog-symbols`, `check-changelog-coverage --fail-on-new`,
  api-stability `--update` + `TestEvery`, `nix fmt` clean, systemtest standalone
  (GOWORK=off + replaces) tidy+green
- Gates RED at report time: `check-file-size` (3 rows, ALL foreign),
  `check-error-taxonomy` (stack/v4 extraction, provenance unknown),
  `check-md-go` (4 fence errors, all inside `docs/drafts/` — the two untracked
  issue-text drafts)

**NOT verified (not attempted / out of scope):**

- Full `nix run .#verify` (build+vet+test+race+lint+doc-check composed) — NOT
  run; parallel sessions keep the tree moving and #verify demands exclusivity
- `doc-check` (skill-reference corpus), `lint-scope`, `check-duplication`,
  `check-cycles`, `check-coverage`, `#test-integration`, `-race` over the new
  tests — NOT run yet (planned next)
- Full (non-targeted) suites of systemtest/system/modules — only targeted runs
- Turso remote-DSN path, duckdb (cgo) leg beyond compile — not exercised
- No as-of-read-through-projection e2e (decision: config plumbing pinned at
  three layers instead; engine semantics already conformance-pinned per
  ADR-0141 §7)

---

## a) FULLY DONE (implemented, tests green, docs updated)

### Issue #130 — `[]Edge` folds silently classified as Set (fix complete)

1. **Loud rejection at fold construction** — `metaengine/v4/fold_single_return.go`
   (new file): `rejectSliceOfFoldUnit` + `canonicalFoldUnits` (9 units: Delta,
   Edge, EdgeRemoval, Embedding, IndexedText, Point, Skip, MultiEntry, Append).
   Wired into BOTH `default:` branches (`classifySingleReturn` for the
   deprecated `On` path, `onRecordFold` for `OnRecord`). Panic names the two
   working shapes (one event per relation; fold batch into Map/Multimap and
   derive in a second pass) and cites issue #130. Custom Set key types (incl.
   slices of non-unit elements like `[]string`) flow through unchanged — pinned.
2. **Six Ginkgo specs** in `metaengine/v4/on_test.go`: OnRecord panic, message
   names both shapes, non-Edge units rejected, `[]string` Set still constructs,
   deprecated `On` panics identically, existing table extended.
3. **Repo-wide consumer grep**: zero in-repo handlers return slice-of-unit
   (all hits are loop-range values, not handler signatures) — no consumer
   regresses.
4. **CHANGELOG** `[Unreleased]` Fixed row with full root-cause narrative.
5. **fold.go SHRANK 496 → 407 lines** (the whole single-return classification
   block moved to the new `fold_single_return.go`, 126 lines) — under the
   file-size ratchet baseline, no baseline update needed.

### Issue #131 — `cell_versioning` config surface (code + tests + docs complete)

6. **metaengine core**: `DriverConfig.CellVersioning *RetentionPolicy`
   (registry.go); `RetentionPolicy.Validate()` (negative knobs rejected);
   `RejectCellVersioning(driver, cfg)` + `ErrUnsupportedCellVersioning`
   (temporal_write.go, errors.go) mirroring the `RejectDurabilityTier` contract.
7. **Memory driver** (metaengine/v4/register.go): translates to
   `NewMemoryEngineWithVersioning(WithRetention(...))` after validation.
8. **sqlite driver**: translates to `WithCellVersioning(WithRetention(...))`,
   invalid policy fails naming the knob; `register_versioning_test.go` pins
   versioning on/off + retention trim (3 writes → 2 surviving versions).
9. **turso**: new `WithCellVersioning` Option (matview.go options struct),
   forwarded to the SQLite core in `New`; driver factory validates + passes.
10. **bigtable**: `New` refactored to a forwarder over new `dial(...)` so the
    factory can inject engine Options; `gcPolicyFor` maps RetentionPolicy →
    `MaxVersionsPolicy` / `MaxAgePolicy` / `IntersectionPolicy` (both knobs);
    zero policy → no GC policy. Versions natively, so the config never gates
    construction.
11. **Rejectors** — pebble, bbolt, badger, postgres, mysql, dgraph, duckdb all
    call `RejectCellVersioning` first; pebble pinned by test
    (`errors.Is(..., ErrUnsupportedCellVersioning)`).
12. **system**: `EngineConfig.CellVersioning *CellVersioningConfig` (koanf
    `cell_versioning`, knobs `max_versions` int + `max_age` Go duration string)
    + `retention()` translation (parse + validate, engine-labeled errors);
    `createEngineFromDriver` passes it into `DriverConfig`. Internal tests:
    `TestCellVersioningConfigRetention`, `TestCreateEngineFromDriver_CellVersioning`.
13. **systemtest**: `TestDeployment_CellVersioningFlowsToDriver` (YAML → koanf →
    `system.New` boots versioned) + `TestDeployment_CellVersioningInvalidAge`
    (malformed `max_age` fails naming cell_versioning/max_age). Module-level
    sibling replaces (system, metaengine, sqliteengine) added per the pre-tag
    sibling-adoption pattern with a drop-in-next-wave comment.
14. **Docs**: ADR-0141 implementation-status addendum (config surface);
    recipes §2.37 deployment-YAML snippet (yaml fence — no recipes-catalog
    churn); FEATURES temporal row updated; TODO_LIST ADR-0141 section got a
    DONE row; CHANGELOG `[Unreleased]` Added row (symbols de-qualified where
    the changelog-symbols resolver can't reach submodule aliases).
15. **api-stability golden regenerated**: exactly 4 new exports
    (`tursoengine.WithCellVersioning`, `metaengine.RejectCellVersioning`,
    `metaengine.ErrUnsupportedCellVersioning`, `system.CellVersioningConfig`);
    `TestEvery` green.

## b) PARTIALLY DONE

~~16. **Gate sweep for #131/#130** — run set is listed in Scope; still missing:~~ routed — rides chain #4 battery (doc-check preflight; lint/duplication/coverage/race in #verify); M32 fixed the named gates
    doc-check, lint-scope, duplication, cycles, coverage, full suites, -race,
    #verify. Three gate failures at report time are detailed in (d).
~~17. **Issue close-out** — neither #130 nor #131 commented/closed yet; no~~ routed — #130/#131 evidence comments + closed 2026-11-13 (M31/F199-F200; gh list empty); boundary-commit discipline ruled (authored at phase boundaries)
    hand-authored boundary commit yet (auto-commit daemon absorbed everything
    into ~10 `chore:` commits; ground rule 4 authorizes a real-message commit
    at phase boundaries — not yet exercised).

## c) NOT STARTED

~~18. `nix run .#verify` (or `verify-chain.sh`) — the composed proof.~~ routed — M34 chain #4 armed (receipt pending)
~~19. Loading `github-voice` + posting evidence comments and closing #130/#131.~~ routed — DONE 2026-11-13 (#130comment/#131comment links in TODO_LIST harvest section)
~~20. `check-systemscenario-adoption` run (new system.New boots in systemtest —~~ routed — rides chain/nightly battery (adoption gate)
    module should be allowlisted already, but unverified).
21. Load-sweep n/a (no timing paths touched). Nightly-bench n/a.

## d) TOTALLY FUCKED UP / RED AT REPORT TIME (with provenance)

~~22. **`check-md-go` RED — 4 fence errors, ALL inside `docs/drafts/`** (the two~~ routed — FIXED 2026-11-13 (M32/F203): 4 fences carry // skip-validate; gate GREEN
    untracked 2026-11-13 issue-text drafts contain verbatim Go fragments that
    don't parse; `docs/drafts/` is LIVE for this gate). The drafts describe the
    exact issues I fixed. Fix is mine to make (annotate `// skip-validate`,
    or re-home/archive the now-resolved drafts) — NOT yet done. This is my
    most actionable open gate.
~~23. **`check-file-size` RED — 3 rows, NONE mine**: `cmd/cqrs-lint/v4/pkg/analyzer/loader.go`~~ routed — FIXED 2026-11-13 (M32/F203): loader split 377→320, doctor_audit 360→203+render, engine.go 664→540; gate GREEN
    377 lines and `cmd/cqrs-lint/v4/doctor_audit.go` 360 (both NEW offenders —
    the parallel #126 cqrs-lint doctor session), and `sqliteengine/v4/engine.go`
    663→664 (parallel #122 sort-index session's +1). My own fold.go row was
    cleared by the shrink in (a)5. Deliberately NOT touched — foreign in-flight
    work, not mine to restructure or revert.
~~24. **`check-error-taxonomy` RED — `generate-error-taxonomy.sh --pool` rc=1 on~~ routed — REPAIRED 2026-11-13 (M32): gate table re-pointed to post-/v4 layout + doc regenerated
    [stack/v4]** ("extraction yielded 16 codes, floor 50"). I never touched
    stack; provenance unknown (possibly the parallel sessions, possibly
    pre-existing). NOT diagnosed beyond reading the error — out of my session's
    blast radius, flagged for the owner.
~~25. **Parallel-session artifacts in my tree** (observed, untouched):~~ routed — h
    `store_folds.go` TOUCHBASE_DEBUG_STACK hook removal (daemon-committed),
    sqliteengine sort-index fields/constructor call (uncommitted when first
    seen), cqrs-lint loader/doctor growth. My `nix fmt` run formatted one of
    their test files (`sort_index_test.go`) — formatter output is canonical,
    kept.

## e) WHAT WE SHOULD IMPROVE (session self-critique — process, not code)

~~26. **Check the file-size baseline BEFORE editing baselined files.** I added~~ routed — codified: M32 executed the split pattern; file-size-baseline consult now part of the ritual
    29 lines to baselined fold.go and learned at gate time; the shrink-via-new-
    file move was reactive. `scripts/file-size-baseline.txt` should be consulted
    before any edit to a >350-line file.
~~27. **Verify after EVERY edit batch, immediately.** The turso matview multiedit~~ routed — process note
    half-failed and left an orphaned doc comment that surfaced only at build
    time two steps later; a `go build` after each batch would have caught it in
    seconds.
~~28. **My first build loop's exit-code handling was wrong** (`cmd | head && echo~~ routed — codified: gotchas "piped cmd|grep masks exit codes" + A7 SIGPIPE row
    OK` reports OK even when cmd fails — `head`'s exit code wins). Misleading
    GREENs until I rewrote the loop with `&&`/`||` around the go command itself.
    Shell habit to kill permanently.
~~29. **Check target filenames before `write`.** I created `fold_classify.go`~~ routed — open
    blind and hit the existing-file guard (the name was taken by ADT
    classification). One `ls`/glob first.
~~30. **`labelOr(cfg.Driver, 0)` in the new error path prints `#0`** for an~~ routed — cosmetic wart, consciously left
    empty driver label — cosmetic wart; driver can't actually be empty there
    (LookupDriver fails first), but `#1` would read better. Left as-is.
~~31. **Boundary commits**: ground rule 4 authorizes real-message commits at~~ routed — r
    phase boundaries; I let the daemon absorb all of it. The #130 work was a
    clean boundary I should have committed before starting #131.
~~32. **Issue close-out should be part of the fix loop**, not a trailing todo —~~ routed — h
    the issues sat open through the whole implementation with no pointer from
    the commits to them.

## f) Next things to get done (impact-ordered, brainstorm-grade)

~~1. Fix the 4 `check-md-go` fence errors in `docs/drafts/` (`// skip-validate`~~ routed — DONE (M32/F203)
   or archive the resolved drafts) — unblocks the md-go gate NOW.
~~2. Run `doc-check` corpus gate (`cd cmd/doc-check/v4 && GOWORK=off go run . ...`)~~ routed — h
   — recipes.md/ADR changed under it.
~~3. Run `GT_LINT_MODULES=... nix run .#lint-scope` over the 11 changed modules.~~ routed — r
~~4. Run `check-duplication` (7 near-identical reject blocks in register.go's —~~ routed — r
   expect art-dupl noise; annotate if flagged).
~~5. Run `check-cycles`, `check-package-surface`, `check-test-matrix`,~~ routed — r
   `check-systemscenario-adoption`, `check-test-sleeps`, `check-test-skips`.
~~6. Full per-module suites (not just targeted runs) for the changed set.~~ routed — r
~~7. `-race` over the new tests (metaengine, sqliteengine, systemtest).~~ routed — rides chain battery (#verify includes race)
~~8. `nix run .#verify` (exclusivity window; tree must settle) or~~ routed — r
   `scripts/verify-chain.sh`.
~~9. Post evidence comments + close #130 and #131 (github-voice).~~ routed — h
~~10. Hand-authored boundary commit(s) with real message(s) — daemon already~~ routed — r
    absorbed the content, so this is a `git commit --allow-empty`-style marker
    or simply skipped; decide consciously.
~~11. Diagnose the `check-error-taxonomy` stack/v4 extraction failure (owner or~~ routed — h
    owning session).
~~12. The parallel sessions' file-size debts (#122 sqliteengine +1; #126~~ routed — FIXED 2026-11-13 (M32/F203: all three shrunk; gate GREEN)
    cqrs-lint loader/doctor) — their owners must shrink or baseline.
~~13. Turso driver-level versioning test at the DRIVER factory layer~~ routed — r
    (currently the option is pinned via `New`, the "turso" driver path via the
    systemtest chain).
~~14. Consider an as-of-read-through-`system.New`-projection e2e in systemtest~~ routed — r
    (DomainConfig query with `AsOf` field) — was descoped this session.
~~15. systemtest sibling replaces: reminder row for the next tag wave's~~ routed — TODO_LIST row ("rides the next tag wave" = M35)
    replace-strip sweep + pin bumps (system, metaengine, sqliteengine).
~~16. cqrs-lint scorecard/doctor: surface `cell_versioning` capability in~~ routed — r
    engine introspection (Doctor/Explain) — optional polish.
~~17. README module table: mention temporal config surface (currently engine-level~~ routed — not rowed (readme-gated polish)
    only) — check readme gates after.
~~18. `docs/engine-capabilities.md` stayed green — no action, just note.~~ routed — noted; gate stayed green (generated doc)
~~19. Harvest this report's (f) into TODO_LIST if the session ends here~~ routed — h
    (docs-health HARVEST rule).
~~20. Deeper: native `[]Edge` fan-out (multi-value fold contract) — the issue's~~ routed — h
    "separate design decision" — belongs on the ROADMAP if wanted.

## g) Questions I CANNOT answer myself

~~1. **The `docs/drafts/` fence errors**: the two untracked drafts are the issue~~ routed — answered by action: annotated // skip-validate (M32); drafts remain as history
   texts for #130/#131 (now fixed). Annotate their fences with
   `// skip-validate`, or are you about to delete/archive them (making the
   md-go failures moot)? They're live for the gate until they move.
~~2. **Cross-session gate debt**: the three file-size failures and possibly the~~ routed — answered by action: M32 fixed all three rows same-session (foreign debts cleared at gate time)
   stack/v4 taxonomy failure belong to parallel sessions (#122, #126). Do you
   want me to fix those on sight (restructure THEIR in-flight code), or is
   that strictly their session's job? I left them untouched.
~~3. **Close timing for #130/#131**: close now on the strength of the targeted~~ routed — answered by events: closed on targeted suites + gate greens; composed verify rides chain #4 (M34)
   suites + green gates listed above, or hold until a full `#verify` pass
   completes (needs an exclusivity window and a settled tree)?

— End of report. Waiting for instructions.
