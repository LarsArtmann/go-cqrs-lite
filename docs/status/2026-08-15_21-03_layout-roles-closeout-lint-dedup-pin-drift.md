# Status Report: Layout-Roles Closeout — Panic Fix, Lint/Dedup Gates, Pin-Drift Repairs

**Date:** 2026-08-15 21:03
**Session type:** Continuation of `2026-08-15_19-50_layout-roles-implementation-session.md` (execute its "Exact Next Steps" 1–10 + gate failures discovered along the way)
**Scope:** metaengine layout-roles closeout ONLY — panic fix, test/soak verification, lint, duplication gate, api-stability, standalone-build repairs, docs. Unrelated repo areas untouched.

---

## A) FULLY DONE (verified GREEN this session)

1. **Item-5 panic fixed** — `sharedTypesInResult` called `field.Type.Elem()`
   unconditionally; scalar fields (`ID string`) panicked at plan time. Now:
   `derefStructType` handles Pointer/Slice/Array chains, `Elem()` only called
   under a `reflect.Map` kind guard (map-value children now also match).
   Bonus fix discovered by my own regression test: matches were **per-field
   duplicates** (`[sharedAttachment ×5]`) → now deduplicated per type, so
   diagnostics/RuleTrace carry each type once.
   Regression: `TestSharedTypesInResult_CoversAllFieldShapes` (all field
   shapes: direct/ptr/slice/ptr-slice/map-value/unexported-skip/scalar-no-panic).
2. **Full metaengine suite GREEN** `-race -count=1` (134.7s), incl. all new
   layout-roles tests.
3. **Soak GREEN** `-race -count=3` for `TestFoldLocks|TestReplication|
   TestPromoteEngine|TestTrace|TestSharedTypes` (10.1s).
4. **Lint driven to 0 findings** in metaengine — fixed all 7:
   - 4× `embeddedstructfieldcheck` (blank line after embedded field:
     `shadowQuery`, `renamedEngine`, `failingShadowEngine`, `gatedShadowEngine`)
   - 1× `errchkjson` — `TraceRecorder` swallowed `enc.Encode` errors → added
     `err field` + exported **`TraceRecorder.Err()`** + test
     `TestTrace_SurfacesEncodeError` (failing-writer test)
   - 1× `golines` (WARN message builder hoisted out of struct literal)
   - 1× `wastedassign` in `benchmark.go` (pre-existing; label init restructure)
5. **Duplication gate failure FIXED properly** (`verify-fast` first run RED:
   2 new clone groups from the vector/graph work):
   - **Real dedup:** `decodeVector` + `topKNearest` were byte-identical in
     bboltengine + pebbleengine → extracted as exported
     `metaengine.DecodeVectorJSON` / `metaengine.TopKNearest` (beside
     `VectorDistance`); both engine modules switched, their full suites GREEN.
   - **Intentional similarity baselined:** pg/mysql `graph.go` INSERT blocks
     differ only in dialect SQL (`ON CONFLICT DO NOTHING` vs `INSERT IGNORE`)
     — engine modules are dep-isolated by design; baselined via
     `art-dupl baseline . --threshold 3 --semantic` (same precedent as the
     already-baselined `encodeNodeKey`).
   - `nix run .#check-duplication` GREEN (0 new clones, baseline 97 groups).
6. **API goldens regenerated** (`--update` + meta-tests green):
   `+metaengine/method Err`, `+metaengine/func DecodeVectorJSON`,
   `+metaengine/func TopKNearest`.
7. **Standalone (GOWORK=off) builds of ALL 19 metaengine-dependent modules
   verified** — two pin-drift breaks found and repaired (see C/D):
   - `cmd/cqrs-bench`: + temporary `replace metadata/v4 => ../../metadata`
   - `system`: + 6th temporary `replace metaengine/v4 => ../metaengine`;
     `system` tests now GREEN standalone.
8. **`system/integration` DuckDB standalone failure triaged** — proven
   PRE-EXISTING via `git worktree` probe at `d807deebb` (identical failure);
   logged as TODO with hypothesis (missing duckdbengine replace); not fixed
   (out of session scope, pre-existing).
9. **Docs all updated:**
   - `TODO_LIST.md`: 7 layout-roles items → `[x]` with implementation notes;
     replaces-count note 5→6 (+cqrs-bench); new DuckDB-standalone TODO.
   - `CHANGELOG.md`: "Added — layout roles" (roles/replication/promote/trace/
     shared-collections/fold-locks) + "Fixed — lint + panic + pin drift".
   - ADR-0124: implementation cross-ref under §Runtime Backend Addition.
   - v5 plan: T29–T35 all → "Done 2026-08-15" with pointer lines.
   - Skill `recipes.md` §2.20 (roles/replication/promote/trace/shared —
     verified against golden symbol names before writing).
   - `AGENTS.md`: doc-check needs `-tags "goexperiment.jsonv2"` (both call
     sites), replace-directives-don't-cascade gotcha, dialect-SQL baseline
     policy gotcha.
   - Status report `2026-08-15_19-50…` annotated with a RESOLVED banner
     (stale "RED" claim corrected at source).
   - `doc-check`: 816 references valid across 41 packages.
10. **`nix run .#verify-fast` fully GREEN** — build, vet, test(short),
    race(short), lint **76/76 modules**, check-arch, depguard, duplication,
    coverage (all within ±2%), api-stability, doc assertions.

## B) PARTIALLY DONE

- **Full-length verification**: the *long* metaengine suite ran fully once
  (before the vector-helper extraction); after the extraction I ran the two
  engine module suites + targeted vector/trace/shared tests + verify-fast's
  short-mode race — never the full 135s suite again. Risk: low (extraction is
  mechanical, engine suites cover it), but the "full suite GREEN" claim is
  one generation old.
- **`nix run .#verify` (full gate incl. soak)**: NOT run — only `#verify-fast`.
  AGENTS.md allows verify-fast as session minimum; full verify is the
  release/tag gate.

## C) NOT STARTED (known, deliberately deferred)

- **FEATURES.md entry for layout roles** — 42 metaengine mentions, zero
  covering roles/replication/promote/trace/shared-collections (verified by
  grep this session). The feature inventory is stale for the whole feature.
- **SKILL.md core decision-matrix + `references/modules.md` metaengine row**
  — new APIs only documented in `recipes.md` §2.20; the two lookup surfaces
  don't mention them.
- DemoteEngine (deliberate v2 deferral), durable/cross-process replication
  (v1 is in-process), public replication tunables (buffer/retries/timeout
  are consts).

## D) TOTALLY FUCKED UP (honest ledger)

1. **I wrote the exact bug AGENTS.md warns about.** My dependent-module build
   loop piped through `head`/`echo` and printed `BUILD-OK cmd/cqrs-bench`
   *while the build had failed* — the "Exit codes after pipes lie" gotcha,
   committed to memory docs by a prior session, reproduced by me from scratch.
   I only caught it because the compile error was visible above the OK. The
   loop's exit handling was wrong, not just cosmetically.
2. **First verify-fast run was RED** — I declared "lint clean + goldens
   regen" and ran verify-fast assuming pass; it failed the duplication gate
   (2 new clone groups from the daemon-committed vector/graph work I had
   adopted as mine). Lesson re-learned: gates before claims.
3. **`TraceRecorder.Err()` was API-by-lint** — I added an exported method to
   silence `errchkjson` rather than deciding the error-handling contract
   first; test came after the fact. Right outcome (silent trace loss is a
   real bug class), accidental path.
4. **Baseline-vs-dedup was a unilateral policy call.** I decided "real logic
   → dedup; dialect SQL → baseline" myself. Defensible (encodeNodeKey
   precedent, engine dep-isolation), but it grows the golden baseline and
   future contributors inherit the decision without having made it.
5. Minor: first `edit` on the test file used a non-unique anchor
   (`}` `}` matches everywhere) — recovered with more context; two rounds
   where one should do.

## E) WHAT WE SHOULD IMPROVE (session-derived)

1. **A `verify-standalone` gate** — 2 of 2 sessions hit pin drift only
   GOWORK=off reveals; this session found 2 more instances (cqrs-bench,
   system). The TODO exists; it keeps earning priority.
2. **Replace directives should carry expiry comments** — system now has 6,
   cqrs-bench 1, with removal conditions living in TODO prose. A
   `// remove when <module>/vX.Y.Z tagged` comment per replace line would
   make them self-documenting.
3. **Daemons commit misindented/broken code** — this session: misindented
   closure in `runtime_backend.go` (repaired by gofumpt), clone-introducing
   vector/graph code, and a build-breaking `slices.Contains()` in history.
   A pre-commit `go build ./...` on the daemon would kill the class.
4. **Test the test helper changes too** — my dedup change to
   `sharedTypesInResult` (per-field → per-type) altered diagnostic output
   contract; the old tests happened to pass either way. Contract tests on
   diagnostic *shape* would have caught a wrong-version dedup.
5. **docs gates**: doc-check requires the jsonv2 tag but AGENTS.md (until I
   fixed it) documented the bare command — copy-paste from docs failed for
   anyone. Doc commands must be runnable verbatim.

## F) UP TO 50 NEXT THINGS

**Close out this feature (quick wins)**
1. FEATURES.md: add layout-roles rows (roles, replication, promote, trace,
   shared collections, fold locks) under metaengine section.
2. SKILL.md decision matrix + modules.md metaengine row: mention
   roles/shadows/promote.
3. Re-run full-length metaengine suite post-extraction (135s) for a
   generation-fresh claim.
4. Run `nix run .#verify` (full, incl. soak) before any tag cut.
5. `system/integration` DuckDB standalone: add duckdbengine replace (or
   driver guard) per TODO hypothesis; verify in worktree first.
6. Annotate every temporary replace with `// remove when <tag>` comments.
7. Update the 19-branch-`tests_run` claim in the 19-50 report? (n/a — but
   audit `system/integration` remaining GOWORK=off failures beyond DuckDB).

**Replication hardening (v1.1)**
8. Public tunables: `WithReplicationBuffer/Retries/OpTimeout` (consts today).
9. `ReplicationStatus` → `Doctor()` integration (surface shadow lag/stale
   in `--- Replication ---` section).
10. PromoteEngine with catch-up verification: refuse if `Applied` <
    EventLog count (currently trusts drain).
11. Optional Verify diff of shadow mirrors vs primary (open question §G).
12. DemoteEngine (Active→Backup) for planned retirements (open question §G).
13. Compute/write split in `applyFold*` so fold locks don't span engine I/O
    (hung shadow can stall primaries ≤3s today — documented trade-off).
14. Replication metrics: OTel counters (applied, retries, stale halts).
15. Cross-process/durable replication (design doc §3.5) — needs WAL/journal
    consensus, out of v1 scope.
16. `Backfill` integration with role add: auto-backfill on
   `AddEngine(WithEngineRole(Migration))` (today: manual runbook).
17. Fenced promote: `PromoteEngine(ctx, name, WithMinCaughtUp(n))`.
18. Replicator surge protection: adaptive buffer or backpressure signal
    instead of stale+halt cliff at 1024.

**Trace/calibration**
19. Trace rotation/size caps for long-running production recording.
20. `TraceStats` → percentile latencies (P50/P95/P99) per name, not just counts.
21. Replay with time fidelity (respect recorded inter-op gaps optionally).
22. cqrs-bench `trace` subcommand: record → replay → compare plans
    (calibrate planner against real workloads).
23. Trace schema v2 when payloads/keys needed (keep JSON-round-trip-unsafe
    values out; encode via registered codecs instead).

**Planner/rules**
24. `shared-collection` rule: match nested fields (depth >1) — today
    top-level only (documented).
25. `WithSharedCollection` by FQN (`pkg.TypeName`) to disambiguate same-name
    types across packages.
26. Physical shared-child materialization (normalize is forced but the
    shared collection isn't built — scoring-level only, per design doc §6).
27. WARN spanning-collections: include estimated duplicate-storage cost.

**Pin/standalone hygiene (this session's evidence)**
28. Pin-drift meta-test (TODO 🔥 exists) — this session added 2 more data
    points; implement it.
29. `#verify-standalone` nix app or CI leg (TODO exists).
30. Repo-wide stale-pin sweep (TODO 🔥 exists; needs user policy sign-off).
31. Consider `go.work` `replace`-mirroring script: generate temporary
    replaces into consumer go.mod files from a single source of truth.

**Gate/process**
32. Daemon pre-commit `go build ./...` hook.
33. CI leg: `art-dupl` check on PRs (today only local `#check-duplication`).
34. Baseline review ritual: `.art-dupl-baseline.json` diffs called out in
    PR review (it silently grows otherwise).
35. `check-file-size` (350-line app, exists, unwired) — wire it or delete it;
    store.go is ~750 grandfathered.
36. Exit-code discipline: scripts/ CI must use `set -o pipefail` (my D.1 bug
    is the third instance of this class in repo history).

**Engine/vector follow-ups**
37. `TopKNearest`: partial-select (heap) instead of full sort for large N.
38. `DecodeVectorJSON` → binary encoding for embeddings (JSON is 3-5× larger).
39. mysql/pg `graph.go`: shared recursive-CTE query builder if a third SQL
    engine appears (baselined for 2, dedup at 3 — repo rule of thumb).
40. duckdbengine vector/graph parity tests vs pebble/bbolt now that helpers
    are shared (`adttest` matrix may already cover — verify).

**Docs**
41. METAENGINE-LAYOUT-ROLES.md: add "Implemented 2026-08-15" status header
    linking to CHANGELOG/tests (it reads as proposal-only).
42. Readmodels/advanced references: cross-link §2.20 from the metaengine
    sections.
43. Session-mistakes pattern: add "pipes lie" + "gates before claims" to
    AGENTS.md Cross-Cutting Lessons (D.1/D.2 here).

## G) UP TO 3 QUESTIONS (cannot answer myself)

1. **Dialect-SQL baseline policy** — I baselined pg/mysql graph-INSERT
   similarity instead of deduplicating (engine dep-isolation; encodeNodeKey
   precedent). Confirm the policy, or do you want a shared SQL-dialect
   helper module eventually (accepting a new shared dep for engines)?
2. **Replication tunables now or later?** Buffer(1024)/retries(3)/timeout(3s)
   are unexported consts. Export `WithReplication*` options now (public API
   surface grows before any real deployment data), or wait for a deployment
   that actually needs different values?
3. **Is `nix run .#verify` (full, with soak) wanted before you next tag
   releases, or is verify-fast the accepted session standard with full
   verify reserved for the pre-tag checklist?** (Determines whether I should
   run the ~long gate now.)

---

**Bottom line:** the layout-roles TODO section is fully closed and gated
GREEN (verify-fast all phases). The honest debt: FEATURES.md/SKILL.md
surfaces not yet updated, one-generation-stale full-suite claim, DuckDB
standalone failure triaged-but-not-fixed, and a duplication-baseline policy
decision awaiting your confirmation.
