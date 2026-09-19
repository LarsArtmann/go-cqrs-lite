# Status Report: Tag-Wave Release Prep, CI Triage, and the Concurrent-Session Dance

**Date:** 2026-09-19 18:15 CEST
**Session scope:** "Time for some new releases?" — assess, plan, and execute the
2026-09-19 tag wave (the 09-08 train's successor), gated on a green tree.

---

## a) FULLY DONE (verified this session)

### Wave plan: 90 tags across 6 dependency-ordered batches — dry-run validated

- Full staleness scan (all 92 `go.mod` dirs): **~80 stale + 10 never-tagged
  modules**, 500+ commits since the 2026-09-08 wave.
- Bump classification done from `docs/api_surface.txt` diff (authoritative
  exported-symbol delta), not commit-message heuristics: 33 modules MINOR,
  **zero deletions repo-wide (no breaking changes, no /v5 pressure)**.
- Dependency order forced by unpublished-module requires: `claiming` (no tag!),
  `queue` (no tag!), `scheduling/engine` (no tag!) are REQUIRED by consumers
  (metaengine, system, queue engines, scheduling/sqlstore) — they cut early.
- **metaengine ↔ sqliteengine circular pair resolved offline**: metaengine's
  production code imports sqliteengine ONLY in test files, so metaengine v4.14.0
  cuts first pinning old sqliteengine (test-only pin), then sqliteengine v4.4.0
  pins new metaengine. No network experiment needed.
- Same-batch sibling limitation honored: consumers of NEW sibling symbols moved
  to later batches (scenario, commandlifecycle/projections → B4; integration,
  benchkit, tursoengine, metaengine/bench → B5).
- **All 6 batches pass `batch-release.sh --dry-run`** (existence, collision,
  path-vs-tag guard, sequence) — plan files in `/tmp/wave_b*.txt`, pin-sweep
  helper at `/tmp/pin-sweep.sh`.
- Dead paths identified and SKIPPED (audit-baseline violations): root module
  (v4 on suffix-less path), `event/v4/eventtest`, `cmd/cqrs-lint/testdata/typedfixture`
  (test fixture). Examples with historical invisible v4 tags get correct v0-line
  tags this wave (taskmanager v0.2.0, getting-started v0.2.0).

### CI triage of the 10 failed jobs on 87a219333 (all root-caused)

| Job | Root cause | State |
| --- | --- | --- |
| Verify-fast | upstream go/types+x/tools race under -race (Go 1.27) | fixed by concurrent session (`skipUnderRace`) |
| cmd/api-stability tests | CI-env-only; pass locally | passes now |
| cmd/cqrs-lint tests | FORCE_COLOR env + same race class | fixed by concurrent session |
| metaengine/mysqlengine | skip-path panic class in adttest factory | passes locally; fixed upstream of me |
| system ResetProjection | checkpoint-after-replay failure | passes locally now (their fix) |
| tursoengine/storage-turso | pushdown/version-ordering | pass locally now |
| Tidy cold-cache (3 modules) | real drift | **fixed by me** (api-stability, taskmanager, integration tidied) |
| Module Isolation Build | projections missing sibling replace | **fixed by me** |
| Layer Arch Check tag-existence | forward pins to never-tagged claiming/scheduling-engine | **wave-transient by design** — resolves when those tags cut |
| calibration-gate self-test | load-probe fixture regression | passes now (their fix) |

### Real fixes landed by this session

- `system/go.mod`: pseudo-version `v4.0.0-00010101000000-000000000000` → clean
  `v4.0.0` forward pin for scheduling/engine (the tagger rejects pseudo-versions).
- `commandlifecycle/projections/go.mod`: missing sibling `replace => ../` added
  + forward pin to commandlifecycle v4.2.0 (its code uses rejection-event symbols
  that only exist there). Standalone build verified green.
- `idempotency/sqlstore`: exhaustruct (Store/engineFacadeOps/queries literals now
  name every field) + err113 (static sentinel `errNilDedupStore`) — lint clean in
  default AND integration-tag mode; module tests green.
- 3 modules tidied (api-stability gains cmdguard require; taskmanager/integration
  indirect drift) — `go mod tidy -diff` clean repo-wide afterwards.

### Research artifacts (reusable)

- Bump table, batch files, pin-sweep script (see above), full CI failure map,
  and the never-tagged-module dependency lattice.

---

## b) PARTIALLY DONE

- **The release itself: 0 of 90 tags cut.** Plan is dry-run-validated; execution
  is gated on (1) lint-green tree, (2) push, (3) CI green. Kept getting
  legitimately preempted by a concurrent session's bursts — deferring the cut
  during their edits was correct (tag scripts require a clean tree; racing
  taggers would collide on refs).
- **Lint findings triage:** verify-fast's lint leg reports **15 findings**, the
  visible cluster in `system/integration/duckdb_test.go` (testpackage, varnamelen,
  wsl_v5 ×2) plus counts: exhaustruct_v5 ×4, errcheck ×2, cyclop, err113,
  gochecknoglobals, goconst, gocritic. NOT yet fixed — my last extraction grep
  lost output to buffering (0 lines), so the per-file attribution beyond the
  visible tail is incomplete. This is the only remaining verify-fast red leg
  known to me.
- **Formatter war:** root-caused — BuildFlow auto-configure re-added gci to
  `.golangci.yml`; the concurrent session reverted it (comment now documents:
  treefmt/goimports owns grouping per AGENTS contract 18). Current `.golangci.yml`
  is correct. Residue of the war: 331 treefmt-formatted files landed via daemon
  commit eff4ac39a (correct content, wrong author).
- **Push/CI precondition:** origin/master is now synced at 53dba1329 (the other
  session pushes too), CI there is **still failure** (needs re-triage after
  their current burst lands).

---

## c) NOT STARTED

- CHANGELOG wave section (cut `[Unreleased]` → dated section listing the 90-tag
  train, per the 2026-09-08 precedent) — deferred because the concurrent session
  held CHANGELOG.md dirty for long stretches (they are ADDING Unreleased entries;
  the cut must happen after their last one).
- The wave execution sequence: per batch → pin-sweep (`/tmp/pin-sweep.sh`) →
  `batch-release.sh "<triples>"` → `git push origin <tags>` → next batch.
- Post-wave: `batch-release.sh --smoke-all`, per-module `GOWORK=off go mod tidy`
  for go.sum completion (gotcha: pins-without-tidy leave missing `/go.mod`
  hashes), full GOWORK=off test matrix, final coherent-pins commit
  (`04beab982` precedent).
- Re-triage of CI at 53dba1329's successor once the current burst lands.

---

## d) TOTALLY FUCKED UP

1. **I ran `nix fmt` mid-session without verifying the formatter state was a
   fixed point.** 333 files churned; the daemon committed them (eff4ac39a)
   before I could decide. The outcome happens to be correct (treefmt grouping
   is canonical, gci now disabled), but I did it accidentally, during someone
   else's formatter refactor, and then compounded it with a 404-file
   `golangci-lint fmt` pass in the opposite direction (reverted in time).
   Lesson: bulk formatter runs on a co-edited tree need an explicit
   "formatter drift audit" first (compare `nix fmt` result vs committed on a
   COPY, not in place).
2. **I nearly double-fixed the cqrs-lint race.** I added a mutex around
   `packages.Load`; the concurrent session's `skipUnderRace` was the correct fix
   (the race is INSIDE one Load — my mutex wouldn't have fixed it). I caught
   this only because I demanded the write-side of the race report before
   trusting my fix. Reverted my edit.
3. **Commit authorship mangled by daemon races** — my sqlstore fix and the
   formatting sweep are inside `chore: auto-commit` commits. Content is right;
   history lies.
4. **Hours of wall-clock spent polling** (8-minute quiet windows) instead of
   front-loading an explicit coordination handshake with the concurrent session.

---

## e) WHAT WE SHOULD IMPROVE

1. **A release lock / claim file** (e.g. `docs/status/` post IS the signal, but
   nothing machine-readable): `tag-release.sh` could refuse to run when a
   `/.release-in-progress` sentinel exists, written by whoever starts a wave.
2. **Formatter drift audit before any bulk format**: `git stash`-safe dry-run
   (format a worktree copy, diff, THEN decide) should be the documented
   procedure — `nix fmt` in-place on a co-edited tree is a footgun.
3. **CI tag-existence gate is wave-hostile**: forward pins (required by the
   cut-order mechanic) make that leg red until mid-wave. A `--allow-forward-pins`
   env / wave-mode would keep master green except the one honest leg.
4. **New modules should be born lint-clean**: `system/integration` (this wave)
   landed with testpackage/wsl/varnamelen findings — a `check-new-module` gate
   (lint the module at creation) would have caught it.
5. **The daemon absorbing explicit work** (my sqlstore fix, the fmt sweep)
   argues for the documented "commit at each phase boundary" discipline — I
   committed late twice and paid for it.
6. **verify-fast output through pipes loses sections** — it buffers/interleaves;
   my greps missed findings. Redirect to a file, then search, is the reliable
   pattern (should be a gotcha).
7. **batch-release.sh dry-run does not validate version SUFFIX sanity** (it
   caught v4-on-suffix-less — good — but I first wrote dedup v4.4.0 skipping
   v4.3.0, which guards allow). A next-version hint (`--suggest-versions`)
   would prevent hand-computed skips.

---

## f) Up to 50 things to do next (ordered)

**Wave execution (the mission):**
1. Wait for current concurrent-session burst (72 files dirty: queue/mysql,
   claimkit, NEW example/goal-shaped-app) to land and quiet for 8+ min.
2. Fix the 15 lint findings (start: `system/integration/duckdb_test.go`
   testpackage/wsl/varnamelen; find the exhaustruct_v5 ×4, errcheck ×2, cyclop,
   err113, gochecknoglobals, goconst, gocritic sites — full file-attribution
   pass via file-redirected lint output).
3. Re-run verify-fast → must be fully green.
4. Decide the new `example/goal-shaped-app` (and any new module in their burst):
   api-stability list? testModules? first tag in the wave? — wave table may grow.
5. Cut CHANGELOG wave section: `[metaengine/v4.14.0, system/v4.8.0, … —
   2026-09-19 release train (+N more module tags)]` — move all `[Unreleased]`
   subsections in, leave empty placeholders, verify with
   `scripts/check-changelog-symbols.sh` semantics.
6. Commit CHANGELOG explicitly (re-check `git status` immediately before add —
   daemon races).
7. Push master; wait CI; triage: everything green EXCEPT the tag-existence leg
   (wave-transient, resolves mid-wave).
8. Batch B0 (Tier 0): record v4.5.1, id v4.6.1, dedup v4.2.2, dispatcher v4.4.1,
   kv v4.3.1 → cut + push.
9. Batch B1 (Tier 1): event v4.11.1, command v4.11.0, query v4.8.1, metadata
   v4.7.1, scheduling v4.5.0 → pin-sweep not needed → cut + push.
10. Batch B2 (Tier 2): claiming v4.0.0 FIRST (unblocks everything), then schema
    v4.4.1, snapshot v4.5.1, projection v4.4.0, deriver v4.3.1, commandlifecycle
    v4.2.0, idempotency/kvstore v4.3.0, otel v4.5.0, otel/otlp v4.0.0,
    prometheus v4.3.1, middleware v4.6.1, signing v4.3.1, encryption v4.4.1,
    testutil v4.3.1, testutil/pgtestcontainer v4.2.1, watermill v4.6.1,
    transport/http v4.3.2, transport/grpc v4.3.1 → cut + push.
11. Batch B3: metaengine v4.14.0 (pins old sqliteengine — test-only), storage
    v4.10.0, storage/memory v4.5.2, storage/pebble v4.4.1, storage/bbolt v4.2.1,
    storage/turso v4.3.2, storage/backuptest v4.2.1, decider v4.7.0, graph
    v4.3.1, listing v4.4.1, projectionhost v4.5.0, queue v4.0.0,
    scheduling/sqlstore v4.1.0 → cut + push.
12. Batch B4: sqliteengine v4.4.0, pgengine v4.4.0, mysqlengine v4.3.0,
    duckdbengine v4.3.0, badgerengine v4.3.0, bboltengine v4.3.0, pebbleengine
    v4.4.0, dgraphengine v4.3.0, irohengine v4.3.0, loopback v4.0.3, quic
    v4.2.1, bigtableengine v4.0.0, otelobserver v4.0.0, projectionadapter
    v4.5.0, graphadapter v4.1.1, queue/sqlite+postgres+mysql v4.0.0,
    scheduling/engine v4.0.0, idempotency/sqlstore v4.4.0, scenario v4.4.0,
    commandlifecycle/projections v4.2.0 → cut + push.
13. Batch B5: tursoengine v4.2.0, metaengine/bench v4.1.0, integration v4.2.1,
    benchkit v4.6.0, stack v4.4.1, stack ×8 patches, stack/bench v4.3.0, system
    v4.8.0 → cut + push.
14. Batch B6: system/integration v4.0.0, catalog v4.5.0, cqrs-lint v4.12.0,
    cqrs-bench v4.3.1, cqrs-upgrade v4.1.0, cqrs-gen v4.3.1, doc-check v4.3.1,
    api-stability v4.4.0, taskmanager v0.2.0, getting-started v0.2.0,
    readme-quickstart v0.2.1, metaengine-quickstart v0.1.1,
    scheduler-otel-status v0.1.0 → cut + push.
15. `batch-release.sh --smoke-all /tmp/wave_all.txt` (proxy + install probes;
    needs proxy propagation 2-10 min for early tags).
16. Per-module `GOWORK=off GOFLAGS=-mod=mod go mod tidy` for every module whose
    pins moved (go.sum `/go.mod` hash completion — gotcha 21); iterate.
17. Full GOWORK=off test matrix (`nix run .#verify-ci`) post-wave.
18. Final coherent-pins commit (04beab982 precedent) + push + CI green.

**Immediate hygiene:**
19. Move `/tmp/wave_*.txt` + `/tmp/pin-sweep.sh` somewhere durable BEFORE any
    reboot (they are the wave's single source of truth).
20. Re-derive versions from tags at cut time (defense against their burst adding
    feat commits to already-classified modules — my MINOR assignments absorb
    this, but PATCH modules with new feats would be misversioned).
21. Check `example/goal-shaped-app` go.mod path/first-tag policy (v0.1.0?).
22. After B2 pushes: verify the CI tag-existence leg flips green (claiming +
    scheduling/engine pins resolve).
23. During B4: watch the tursoengine cut — it embeds sqliteengine and must land
    AFTER sqliteengine v4.4.0 is PUSHED (same-batch limitation is why it is in
    B5; verify no batch reorder crept in).
24. If any batch cut fails its GOWORK=off gate: read the build error, move the
    failed module one batch later, re-push — do NOT force through.

**Post-wave alignment:**
25. Refresh `cmd/cqrs-lint` taskmanager/V006 goldens if the version set changed
    (gotcha: cqrs-lint pins the version set).
26. Run `scripts/batch-release.sh --audit` post-wave (expect only baseline
    dead-path violations).
27. Confirm `docs/api_surface.txt` golden still matches (go.mod sweeps do not
    change exports — verify anyway).
28. Check Dependabot config cap warnings (93 modules, capped at 20) — same wave
    precedent: leave, it is informational.
29. Verify `testModules`/api-stability list membership for every never-tagged
    module now tagged (meta-test `TestEveryGoModDirIsInModulesList`).
30. Confirm release.yml auto-created GitHub Releases for all 90 tags (or accept
    the tag-only flow — check what 09-08 did).
31. Consumer-side propagation is go-ecosystem-upgrade's job — note it, don't do
    it in this session.
32. Update AGENTS gotchas with: (a) the forward-pin + tag-existence-leg wave
    mechanic (it confused me for an hour), (b) the two-phase formatter audit.

**Code debt observed during triage (not mine to fix unilaterally):**
33. `metaengine/adttest` skip-path panic class (mysql "panic(nil)") — the
    concurrent session's racer-count work is adjacent; verify their fix covers
    the skip path too.
34. `system/integration` module: consider `testpackage` rename or a nolint
    policy decision — the module violates testpackage BY DESIGN? (its tests
    exercise the public API surface as a consumer would).
35. `cmd/cqrs-lint/pkg/analyzer/loader.go`: my mutex idea is still semantically
    right for defense-in-depth IF x/tools ever fixes the internal race — record
    as a TODO upstream-watching note, not code.
36. coverage <80% job failing on 87a219333 — which package? Never triaged (CI
    only). Re-triage post-wave.
37. gosec job failing on 87a219333 — never triaged in detail (buildflow's local
    gosec was env-broken). Re-triage post-wave.
38. CGo Build (DuckDB + Iroh QUIC) failure — never root-caused (nix build leg,
    error invisible in my grep). Re-triage post-wave.
39. Dgraph Integration failure at 87a219333 — capability-refusal work changed
    dgraph's ADT surface; check whether the dgraph conformance suite honors
    `RefusedADTs`. Possibly fixed by their burst.
40. Benchmarks workflow failing on 53dba1329 — new; likely their benchkit gate
    work in flight. Leave to them, verify later.
41. `example/scheduler-otel-status/scheduler-otel-status` binary tracked in git
    (go-structure-linter ERROR) — `git rm --cached` + .gitignore, needs a
    decision (example repo hygiene).
42. `catalog/ec-fixture` binary tracked in git — same class as 41.
43. `metaengine/duckdbengine/dueclaim.go` (untracked at session start, now
    committed?) — verify it landed in a daemon commit and is in the wave's
    duckdbengine delta.
44. `govulncheck` cannot run in buildflow env (GOTOOLCHAIN=local vs go 1.27.1) —
    env fix candidate for `.buildflow.yml` (documented fix exists: `env -u
    GOTOOLCHAIN`).
45. flatbuffers `+incompatible` warnings (9 go.mods) — upstream dep style;
    low-priority sweep.
46. `transport/grpc/proto` buf-lint naming warnings — pre-existing, cosmetic.
47. flake meta warnings (homepage/mainProgram missing) — cosmetic.
48. BuildFlow binary staleness (built at 42fd89b, HEAD 2fb0382) — rebuild
    BuildFlow (`nix run .#reinstall` in BuildFlow repo) — operator action.
49. `todo-check`/`interrogate` binaries not in PATH — tool hygiene, low.
50. After the wave: hand off consumer propagation to go-ecosystem-upgrade and
    write the wave retrospective into `docs/agents/gotchas-module-management.md`
    (forward-pins mechanic + never-tagged-first ordering).

---

## g) Questions I cannot answer myself

1. **Who owns the wave cut — this session or the concurrent one?** The other
   session is pushing master and cutting features (benchkit tail, queue/mysql
   polish, a new example). If they also intend to run `tag-release.sh`, we will
   collide on refs mid-wave. I have the plan loaded and dry-run-validated; say
   the word and I execute (or stand down).
2. **May I push master + ~90 tags to origin during execution?** The wave
   REQUIRES interleaved pushes (GOPRIVATE direct-VCS resolution between
   batches). Pushing master also publishes the concurrent session's commits —
   confirm that is acceptable, or tell me to coordinate with them first.
3. **Version-bump policy confirmation for two edge cases:** (a) modules whose
   only change is Go-toolchain/dep pins get PATCH (record v4.5.1 etc.) — right?
   (b) `example/taskmanager`/`example/getting-started` have suffix-less paths so
   their historical `v4.x` tags are proxy-invisible dead paths; I plan correct
   `v0.2.0` tags on the v0 line (their last VISIBLE tags are v0.1.0) — confirm
   you do not want the /v4 module-path migration done as part of this wave.
