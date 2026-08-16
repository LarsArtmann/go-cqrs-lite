# Status Report: Verify Gate Execution — check-arch Root Cause, Lint Fallout

**Date:** 2026-08-15 00:51
**Scope:** This session's run: the first full `nix run .#verify` after the
ADR-0128 shim deletion, the check-arch 94-gap root-cause fix, and lint fallout
repair. Prior-session context (shim migration, Dgraph fix, doctor split,
layout docs, replace-directive audit) is in the
[2026-08-14 report](2026-08-14_20-46_todo-execution-shims-layout-dgraph.md).

## a) FULLY DONE (this session)

1. **Recon** — repo state verified on resume: no conflict markers anywhere,
   111 uncommitted files matching the prior session's change set, two new
   daemon commits on top (`4e9c1190`, `7e6d4da3`).
2. **Full verify gate executed end-to-end for the first time** — doc
   assertions, module coverage, build, vet, test (all ~130 packages), race:
   ALL GREEN on the first pass. Also independently confirmed green:
   check-depguard (115 deps), check-duplication (0 new clones), check-coverage
   (all within ±2.0%), check-api-stability, doc-check (1020 refs / 60 pkgs).
3. **check-arch 94-gap root-cause fix (was a TODO "M" item — the diagnosis in
   the TODO was wrong)**: the gaps were NOT missing catalog entries. The
   LAYER/DEP_BUDGET maps used cosmetic spaced keys (`LAYER[storage / memory]`,
   `LAYER[cmd / cqrs - gen]`) while every lookup builds a literal path — so 47
   multi-segment modules were "missing" AND their budget + layer-ordering
   checks silently no-oped (`"storage / memory/go.mod"` never exists). The
   spaced-key convention had disabled dependency-budget and layer enforcement
   for every engine, stack preset, storage backend, cmd, and example module.
   Fixes:
   - All keys normalized to plain paths (`storage/memory`, `cmd/cqrs-gen`).
   - Dep-path extraction now strips only a TRAILING `/vN` (the old regex
     mangled nested modules like `event/v4/eventtest`).
   - `system/integration` was genuinely missing: added LAYER 7 + budget 7.
   - With enforcement live, 15 apparent violations surfaced (L2–L6 modules
     depending on `event/v4/eventtest` L7) — verified all are test-only usage
     by design (the one non-test importer, `signing/internal/testutil`, is
     itself test-only). Added a `TEST_INFRA_MODULES` exemption (eventtest,
     testutil, testutil/pgtestcontainer) instead of 15 EXCEPTIONS entries;
     trimmed the now-redundant testutil exceptions from projectionhost.
   - Real budget overrun surfaced: cmd/cqrs-bench 19/18 (metaengine dep from
     the `cqrs-bench layout` CLI) — budget bumped to 19 with comment.
   - `check-coverage.sh` keys normalized for consistency (it stripped spaces
     itself, so it was never broken).
   - AGENTS.md gotcha rewritten: plain-path keys are now the enforced rule.
   - `TestExceptionsAreMinimal` + `TestEvery` meta-tests green; **check-arch
     fully green** (Layer 1 + Layer 2). TODO item removed per the TODO's own
     "completed work is never duplicated" policy.
4. **Lint fixes for 4 of 7 failing modules** — decider, projectionhost,
   middleware, stack: 16 files with gci/gofumpt violations (import rewrites
   from the go-flightrecorder/go-idempotency migration, never re-formatted).
   Fixed via `golangci-lint --fix`; module tests re-run green.
5. **SA1019 MemoryStore handling in middleware** — 12 hits (10 call sites +
   2 helper return types). Added a scoped `.golangci.yml` exclusion
   (`middleware/.*_test\.go$`, text-scoped to MemoryStore/NewMemoryStore)
   following the existing tombstone/EnsureCustom precedent, since go-idempotency's
   own deprecation notice names tests as the sanctioned consumer. Removed the
   2 now-redundant per-line nolints. Middleware lints clean.
6. **Corrected my own stale-GREEN claim** — I had written "all verify phases
   green" into the 2026-08-14 report while the confirmation lint re-run was
   still in flight; it came back exit=1. The line now states precisely which
   phases are green and which are not. (Exactly the "stale GREEN is worse than
   no claim" anti-pattern AGENTS.md warns about — caught and fixed same
   session.)

## b) PARTIALLY DONE

1. **Lint gate: 3 modules still failing (15 issues), all mechanical, all
   diagnosed, fixes known**:
   - `cmd/cqrs-lint/pkg/analyzer/module_catalog_data.go:127` — golines ×1
     (long line from the catalog edit in the prior session).
   - `idempotency/kvstore` — gci ×4, gofumpt ×4, SA1019 NewMemoryStore ×3
     (same class as middleware).
   - `idempotency/sqlstore` — gci ×3.
     Fix = `--fix` in those modules + extend the SA1019 exclusion path from
     `middleware/.*_test\.go$` to also cover `idempotency/.*_test\.go$`. I
     stopped here because the user instructed: report, then WAIT.
2. **The change set is uncommitted** (~115 files now). The daemon may commit
   it piecemeal; a deliberate commit has not been made.

## c) NOT STARTED (carried over, not touched this session)

- Tagging engine v4.0.2+ (×4: sqlite/badger/pebble/pg) and watermill/v4.5.0,
  then deleting the 5 temporary replaces in `system/go.mod`. Needs explicit
  user approval.
- The other 33 open TODO_LIST items (layout calibration benches, cqrs-lint
  per-module regression tests, Dgraph JournalReadFrom off-by-one, real
  Redis/NATS broker roundtrips, v5 Phase 8 deletions, v5 migration guide,
  go-codec repo scaffolding, etc.).
- GOWORK=off standalone re-verification of the 3 remaining lint-failing
  modules after their fix (they were verified standalone in the prior
  session; only formatting changed since).

## d) TOTALLY FUCKED UP (this session's honest failures)

1. **Incomplete failure extraction → premature scope conclusion.** After the
   first lint run I extracted failures with hand-rolled grep/awk filters
   instead of parsing every module's section of the log; the filters matched
   decider/middleware/stack but silently dropped `cmd/cqrs-lint` and
   `idempotency/{kvstore,sqlstore}`. I then declared "formatting clean" and
   wrote "all phases green" — the confirmation re-run proved both claims
   wrong. Correct process: per-module sectioning of the full log, or simply
   `grep -c` per `==> Linting` block, BEFORE claiming anything.
2. **Wrote green-before-finished into a durable doc.** The premature "Result:
   all verify phases green" line went into the 2026-08-14 status report
   before the re-run completed. Violates the repo's own stale-GREEN rule.
   Fixed in place; lesson logged here so it is not memory-holed.
3. **Prior-session miss that this session inherited:** the shim migration
   rewrote imports across 15 modules without running lint per affected
   module immediately after. Two sessions later we are still paying that
   debt in 3 modules. Rule going forward: after any mass import migration,
   lint every touched module in the same change.

## e) WHAT WE SHOULD IMPROVE (systemic, beyond this session)

1. **Cosmetic key conventions that diverge from real paths are a silent
   enforcement-killer.** The spaced-keys bug disabled two gates for every
   multi-segment module for months while both gates printed "passed". The
   coverage check caught it only as "94 gaps" that everyone assumed were
   missing entries. Improvement: a meta-test that constructs the path from
   every map key and asserts the go.mod exists (would have failed on day
   one), or move the catalog to a data file the script and Go both parse.
2. **Log discipline for long gates.** `#verify` emits thousands of lines;
   failures get truncated. The lint app already aggregates `failed=1`
   per-module — a final `FAILED MODULES: a, b, c` summary line in the lint
   script would have prevented my extraction mistake entirely. Cheap, high
   value. (Candidate for TODO: infrastructure polish item.)
3. **Migration checklists should include per-module lint**, not just
   build+test. Lint is the only gate that catches formatting/import-order
   drift, and it runs late in the pipeline.
4. **The auto-commit daemon keeps racing sessions** (conflict markers last
   session, mid-session commits this session). It also makes "uncommitted
   work" a fuzzy concept — worth an explicit policy (see questions).

## f) NEXT — up to 50, ordered by leverage

**Immediate (finish this session's work):**
~~1. Fix the 15 remaining lint issues (golines/gci/gofumpt via `--fix` in~~ done at 444be10a7 (15/15 fixed; per-module tests re-run green - see Follow-up below)
cmd/cqrs-lint, idempotency/kvstore, idempotency/sqlstore).
~~2. Extend the SA1019 MemoryStore exclusion to `idempotency/.*_test\.go$`~~ done at 444be10a7 (exclusion widened to (middleware|idempotency)/.*_test.go$)
(same text-scoped rule as middleware).
~~3. Re-run `nix run .#lint` → expect exit 0.~~ done at 444be10a7 (76/76 modules clean, exit 0)
~~4. Re-run `nix run .#verify` FULL — expect the first genuinely green full~~ done at 5f2198189 (first genuinely green full verify since ADR-0128; three GREENs since)
gate since the shim deletion.
~~5. Commit the change set (subject to user's answer on commit policy).~~ done - daemon committed the set as 5127039da + 875bb689b (Follow-up #1)

**Release (unblocks system/ replaces):**
6. User approval + tag engine v4.0.2 for sqliteengine, badgerengine, <- OPEN. awaiting user approval - TODO_LIST 'Release / Tagging'
pebbleengine, pgengine (driver self-registration).
7. User approval + tag watermill/v4.5.0 (errors.Join handler independence). <- OPEN. awaiting user approval - TODO_LIST 'Release / Tagging'
8. Delete the 5 temporary replaces from system/go.mod; re-verify GOWORK=off. <- OPEN. gated on the engine tags - TODO_LIST 'Release / Tagging'
9. After re-tag chain: tidy the ~49 go.mod files carrying stale `// indirect` <- OPEN. TODO_LIST 'Release / Tagging' (~49 stale indirect refs)
shim refs; verify each standalone build.
10. Tag final v4.x patches of transport/http + transport/grpc (deprecation <- OPEN. TODO_LIST 'Release / Tagging' (transport v4.x patches)
notices included) — prerequisite for the v5 deletion.

**Gate hardening:**
~~11. Add `FAILED MODULES:` summary to the lint app script (flake.nix).~~ done at 2e9a2fc28 (failedMods summary line in the lint app)
~~12. Meta-test: every LAYER/DEP_BUDGET key maps to an existing go.mod (kills~~ done at 4a95bd04d (LAYER meta-tests in cmd/api-stability/main_test.go)
the spaced-key bug class forever).
13. Consider check-module-layers.sh → Go tool or TOML catalog (deferred
before; the two bugs found tonight raise the priority).

**TODO_LIST top items (unchanged, 33 open):**
~~14. Dgraph `JournalReadFrom` seq offset off-by-one (S).~~ done at 7c0a62c98 (JournalReadFrom made position-based; shared contract suite wired)
15. cqrs-lint per-module regression tests F004/F007/F009/F012/F017/F023-F029/ <- OPEN. TODO_LIST 'cqrs-lint' (per-module regression tests)
B030 (M).
16. `.golangci.yml` exclusion audit — system/ (20 linters off), cmd/cqrs-lint/ <- OPEN. TODO_LIST 'Code Quality' (.golangci.yml exclusion audit)
(17), metaengine/ (24) (M).
17. DuckDB Columnar calibration bench — the 2.65 vs 2.65 tie (M). <- OPEN. TODO_LIST 'Metaengine' (calibration benchmarks)
18. SQLite/Postgres/MySQL Row-layout calibration (M). <- OPEN. TODO_LIST 'Metaengine' (calibration benchmarks)
19. Multi-engine integration test with two real backends (M). <- OPEN. TODO_LIST 'Metaengine' (multi-engine, two real backends)
20. Real Redis/NATS broker roundtrips — replace the corpse stubs in <- OPEN. Redis roundtrip shipped at d8c73be0a; NATS + broker edges = TODO_LIST 'Code Quality' (Wire broker tests into CI)
watermill/broker_integration_test.go (M).
21. v5 Phase 8 deletions: stack.Materialize, RelationalProjection + view, <- OPEN. TODO_LIST 'v5 Unification Phase 8'
graph.GraphProjection, stack.Bundle + 8 presets, stack.RunProjections,
ADR-0126 compat shells (S/M each).
22. v5 migration guide (L); then cut v5.0.0 (M). <- OPEN. TODO_LIST 'v5 Unification Phase 8' (migration guide)
23. go-codec repo scaffolding (.golangci.yml, CI, FEATURES/ROADMAP/SECURITY)
(M).
24. macOS verification of scripts/ephemeral-pg.sh (M). <- OPEN. TODO_LIST 'Code Quality' (macOS ephemeral-pg)
25. Nix infrastructure polish: #check-lint-config, #verify-ci, wire #sweep, <- OPEN. TODO_LIST 'Code Quality' (Infrastructure polish)
consolidate 7× engine register.go boilerplate (M).
26. Layout roles design doc prerequisites: fold-pipeline sync, async <- OPEN. TODO_LIST 'Metaengine - Layout Planning' (concurrent session designing METAENGINE-LAYOUT-ROLES.md)
replication, role transition API, workload trace format, aggregate
boundary config, per-fold mutex, multi-collection atomicity (L each).

## g) QUESTIONS (cannot figure out myself)

1. **Commit policy vs the daemon:** should I make one deliberate commit of
   the full change set now (clean message, logical unit), or leave the tree
   for the auto-commit daemon to commit piecemeal? Last session's plan said
   "commit the change set," but the daemon has since interleaved its own
   commits (`4e9c1190`, `7e6d4da3`), so a clean single commit is no longer
   possible — I would commit what remains and note the daemon's interleaves.
2. **Tagging approval:** do you want the engine v4.0.2 tags (×4) and
   watermill/v4.5.0 tagged now (annotated, via scripts/tag-release.sh), or
   batched with the final transport/* v4.x patches into one release pass?
   Ordering matters: engine tags also unblock removing the system/go.mod
   replaces and the ~49 stale indirect refs.
3. **kvstore MemoryStore tests:** the 3 remaining SA1019 hits in
   idempotency/kvstore are in test matrices that intentionally compare
   MemoryStore against the KV-backed store. Extend the scoped lint exclusion
   (my recommendation, 5 min), or invest in migrating those tests onto the
   go-idempotency contract suite (bigger, arguably cleaner)?

## Follow-up (same night): Lint + Full Verify GREEN

Outcome of the 3 questions above, by events and by the user's standing
"keep going" instruction:

1. **Commit policy — resolved by events.** The auto-commit daemon committed
   the change set as `5127039da` + `875bb689b` before this follow-up ran.
   Nothing left to commit except the lint fixes below.
2. **Tagging — still awaiting explicit user approval.** Engine v4.0.2 (×4)
   and watermill/v4.5.0 remain untagged; system/go.mod still carries the 5
   temporary replaces.
3. **kvstore SA1019 — took the recommendation.** Widened the scoped
   `.golangci.yml` exclusion path from `middleware/.*_test\.go$` to
   `(middleware|idempotency)/.*_test\.go$` (same SA1019 MemoryStore text).

Fixes applied (15/15 lint issues):

- `cmd/cqrs-lint` — golines ×1 (`module_catalog_data.go`) via
  `golangci-lint --fix`.
- `idempotency/kvstore` — gci ×4 + gofumpt ×4 via `--fix`; SA1019 ×3 via the
  widened exclusion above.
- `idempotency/sqlstore` — gci ×3 via `--fix`.
- Per-module builds + tests re-run green after the fixes (kvstore 5.2s,
  sqlstore 51.5s, cqrs-lint pkg tests ok).

Gate results (verified from full logs, not partial extraction):

- `nix run .#lint` — exit 0, 76 modules, every module `0 issues.`
  (zero non-clean lines in the full log).
- `nix run .#verify` — exit 0, all 18 phase markers present (pre-flight
  checks, Build, Vet, Test, Race, Lint, Arch, Depguard, Duplication,
  Coverage, API Stability, Doc Check), final line
  `✅ All verification checks passed`.

**This is the first genuinely green full verify gate since the ADR-0128
shim deletion.**

Benign noise observed in Doc Check: warnings for `../../flightrecorder`,
`../../retry`, `../../idempotency` ("cannot read / no exports found") —
the doc-checker probing for sibling checkouts of the externalized repos
that are not present locally. Non-blocking; references themselves are the
correct external paths.

---

## Resolution (2026-08-15)

24 of 26 items carry verdicts. Items 1-5 and 11-12 closed by the same-night
follow-up and the next sessions (`444be10a7`, `5f2198189`, `2e9a2fc28`,
`4a95bd04d`); item 14 closed at `7c0a62c98`. The release-chain items (6-10)
stay OPEN pending user approval - TODO_LIST "Release / Tagging". Items 13
(layer script -> Go/TOML) and 23 (go-codec repo scaffolding, external repo)
remain open without routing. Stays active for the release chain.
