# Status: Lint Sweep Recovery — Clean Branch Restored

**Date:** 2026-08-12 21:36
**Session:** Recovery from catastrophic concurrent-agent refactor + re-apply lint fixes
**Branch:** `lint-sweep-recovery` (at `2b72de54f` + 11 uncommitted files)
**Master:** `11e7746d6` — STILL contains 3 damaging commits (`a6613ef0d`, `7942a61bb`, `9249d1301`) + 2 auto-daemon follow-ups

---

## a) FULLY DONE

1. **Identified all damage from the concurrent agent's `a6613ef0d` commit** — 644 files changed, 164 deleted, build broken
2. **Categorized every deleted file** by directory and type (production vs test vs config)
3. **Created clean recovery branch** `lint-sweep-recovery` from `2b72de54f` (last known-good commit)
4. **Verified build passes on clean state** — `go build -tags "goexperiment.jsonv2" ./...` clean
5. **Re-applied 4 original lint fixes:**
   - `cmd/cqrs-bench/layout.go` — `enc.SetIndent("", "  ")` → `jsontext.NewEncoder(w, jsontext.WithIndent("  "))`
   - `metaengine/layout_matrix_test.go` — removed redundant `layout, priority := layout, priority` (copyloopvar)
   - `metaengine/sqliteengine/graph.go` — `defer rows.Close()` → `defer func() { _ = rows.Close() }()` (errcheck)
   - `middleware/` — `NewTestIdempotencyStore` helper + all 21 deprecated callsite swaps (SA1019)
6. **Re-applied `.golangci.yml` exclusion** for `codec/alias.go` (gochecknoglobals + wrapcheck)
7. **Fixed 2 pre-existing gci issues** unmasked in `cmd/cqrs-bench/factory.go` and `flags.go`
8. **Per-module lint verification** — `golangci-lint run` returns 0 issues on: `middleware/`, `codec/`, `metaengine/`, `metaengine/sqliteengine/`, `cmd/cqrs-bench/`
9. **Test verification** — `go test` passes on middleware, metaengine, sqliteengine

---

## b) PARTIALLY DONE

1. **Lint sweep across all 80+ modules** — Only verified the 4 modules I knew had issues. Did NOT run a full `nix run .#lint` on the clean branch. Other modules may have latent issues.
2. **Verification gate** — Ran `go build` but did NOT run `nix run .#verify` (build + vet + test + race + lint + doc-check). Nix daemon was down; fell back to direct `go build`.
3. **`nix fmt`** — Did NOT run. AGENTS.md says "Always nix fmt BEFORE placing //nolint directives". I manually shortened nolint comments to stay under 120 chars, but never verified with `nix fmt` / `golines`.

---

## c) NOT STARTED

1. **Committing the 11 changed files** — All work is uncommitted on `lint-sweep-recovery`.
2. **`nix run .#lint`** on the full workspace (authoritative lint path via nix vendor deps).
3. **`nix run .#verify`** — the full verification gate.
4. **api-stability golden regeneration** — `.golangci.yml` change may not affect symbols, but should verify: `cd cmd/api-stability && GOWORK=off go run main.go -update`.
5. **doc-check** — `cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ...`
6. **AGENTS.md update** — Should document the codec/alias.go exclusion pattern.
7. **Master branch cleanup** — Master still has the damaging commits. Needs `git merge` or branch replacement.
8. **Stash cleanup** — `stash@{0}` has orphaned changes from the pre-recovery state on master.

---

## d) TOTALLY FUCKED UP

1. **The concurrent agent catastrophe** — Not my doing, but `a6613ef0d` deleted:
   - 4 entire modules: `metaengine/bboltengine/`, `mysqlengine/`, `tursoengine/`, `storage/backuptest/` (all with their own go.mod, engine.go, full test suites)
   - 26 metaengine production files merged into 2 mega-files (`store.go` 672 lines, `execute.go` 566 lines — both violate the 350-line limit)
   - `codec/alias.go` and `flightrecorder/alias.go` — intentional backward-compat re-exports deleted, reverting the go-codec/go-flightrecorder extraction
   - 32 metaengine test files, 7 engine test suites, `storage/sqlite_wal_concurrency_test.go` (299 lines), multiple backup lifecycle test files
   - This is all preserved on `master` branch — we did NOT lose the code, it's in git history at `2b72de54f`.

2. **My first `git revert` attempt** failed with merge conflicts in `stack/bench/go.mod`, `stack/postgres/go.mod`, `stack/postgres/go.sum` — these had cascading go.mod changes from the concurrent agent. I correctly pivoted to a clean branch instead.

---

## e) WHAT WE SHOULD IMPROVE

1. **NEVER let two agents work in the same workspace** — This caused the entire catastrophe. Use `git worktree` for isolation. Add to AGENTS.md as a hard rule.
2. **Run `nix fmt` before placing //nolint directives** — I violated this AGENTS.md rule. The nolint comments could be in wrong positions after golines reformats.
3. **Commit incrementally** — I left 11 files uncommitted. Each logical fix should have been committed immediately after verification.
4. **Run the full verification gate** — `nix run .#verify` is the authoritative check. Per-module `golangci-lint` and `go build` are necessary but not sufficient.
5. **The `.golangci.yml` exclusion approach for `codec/alias.go`** is a workaround. The real fix is migrating consumers to import `go-codec` directly, then deleting `alias.go`. The exclusion hides the deprecation.
6. **The SA1019 fix uses `//nolint:staticcheck`** in external test packages (`middleware_bdd_test.go`, `idempotency_pipeline_bench_test.go`). These can't access the internal `newTestIdempotencyStore` helper. A better fix would be an exported test helper or moving the helper to a shared `_test.go` utility package.
7. **`nix run .#lint` vs direct `golangci-lint`** — The nix path uses fresh vendor deps; direct golangci-lint uses stale module cache. Always prefer `nix run .#lint`.

---

## f) NEXT 50 THINGS TO DO

### Critical (block merge)

~~1. Commit the 11 changed files on `lint-sweep-recovery`~~ done - branch landed via the merge 0e4a16a8a
~~2. Run `nix run .#lint` on the full workspace~~ done - 76/76 modules clean since 444be10a7
~~3. Run `nix run .#verify` (or at minimum `nix run .#verify-fast`)~~ done at 5f2198189
~~4. Run `nix fmt` and verify nolint comments survive~~ done - lint clean (nolintlint active, no positional fallout)
~~5. Run `go vet -tags "goexperiment.jsonv2" ./...`~~ done - vet phase green in every verify since

### Branch cleanup

~~6. Decide master branch strategy: fast-forward to `2b72de54f` + cherry-pick, or merge `lint-sweep-recovery`~~ done - resolved by merge: 0e4a16a8a restored the deleted modules/infra (see the 01-02 post-merge review)
7. Clean up `stash@{0}` (orphaned changes from pre-recovery master) <- OPEN. stash@{0} (WIP @ e87be3143) still exists - TODO_LIST 'Code Quality' junk-cleanup item
~~8. Delete or archive the old `docs/status/2026-08-12_12-45_lint-sweep-blocked-by-concurrent-refactor.md` (now stale)~~ done - 12-45 annotated item-by-item and archived by the docs-health pass 2026-08-15

### API stability

~~9. Run `cd cmd/api-stability && GOWORK=off go run main.go -update` (regenerate golden)~~ done - golden current (4133 exports)
~~10. Run `cd cmd/api-stability && GOWORK=off go test -run TestEvery .` (meta-tests)~~ done - meta-tests green
~~11. Verify `.golangci.yml` change doesn't affect the api-stability surface~~ done - api-stability green in every verify since

### Doc consistency

~~12. Run `cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md`~~ done - doc-check green (797 refs)
13. Update AGENTS.md with the codec/alias.go exclusion pattern <- NOT-DO - codec/ deleted at 5127039da; the exclusion died with it
14. Add AGENTS.md rule: "Never run two agents in the same workspace — use git worktree"

### Full lint sweep (original goal)

~~15. Run `nix run .#lint` and capture output to `/tmp/lint_clean_baseline.txt`~~ done - zero issues across all modules (76/76)
~~16. For each module with issues: fix, verify per-module, commit~~ done - same
~~17. Target: zero issues across all 80+ modules~~ done - same
~~18. Pay attention to modules NOT yet checked: `event/`, `command/`, `query/`, `decider/`, `storage/*`, `stack/*`, `system/`, `catalog/`, `transport/*`, `watermill/`, `scheduling/`, `signing/`, `encryption/`, `kv/`, `graph/`, `listing/`, `scenario/`, `deriver/`, `commandlifecycle/`, `snapshot/`, `projection/`, `projectionhost/`, `schema/`, `otel/`, `prometheus/`, `idempotency/*`, `record/`, `metadata/`, `dedup/`, `dispatcher/`, `retry/`, `id/`, `testutil/`, `benchkit/`, `integration/`, `example/*`~~ done - full-workspace lint covers all of them

### Pre-existing gci issues (unmasked by fixes)

~~19. Check if other `cmd/*` modules have the same import grouping issue (go-codec + go-error-family in wrong order)~~ done at 2e9a2fc28/444be10a7 - the 95-file gci wave fixed and lint stays clean
~~20. Fix any gci issues found across all modules~~ done - same

### nolint directive audit

~~21. After `nix fmt`, verify all `//nolint:staticcheck` comments are on the correct line~~ done - lint clean; nolintlint enforces directive hygiene
22. Consider extracting an exported `testutil.NewTestIdempotencyStore` so external `_test` packages don't need inline nolint
23. Audit all existing nolint directives in the codebase for staleness

### metaengine recovery (if we want any of the concurrent agent's work)

~~24. Review the 60 deleted metaengine core files — were any of them improvements we want to cherry-pick?~~ done at 0e4a16a8a - merge restored the deleted modules and core infra; nothing left to cherry-pick
~~25. Review the file consolidation (store.go 672 lines, execute.go 566 lines) — this violates the 350-line rule~~ done - the 672/566-line mega-files died with the broken snapshot; 350-line rule enforced since
~~26. Check if any of the 164 deleted files contain improvements worth salvaging~~ done - resolved by the merge (everything restored or superseded)
27. The `codec/` re-import (30+ implementation files replacing alias.go) — was any of this new/updated code? <- NOT-DO - codec/ deleted outright at 5127039da (ADR-0128)

### Test coverage

~~28. Run middleware full test suite with `-race`: `go test -race -tags "goexperiment.jsonv2" ./middleware/...`~~ done - race phase green 3x since 5f2198189
~~29. Run metaengine full test suite with `-race`~~ done - same
~~30. Run `nix run .#check-coverage` — coverage drift detection~~ done - gate repaired at 875bb689b; green since

### Dependency/architecture checks

~~31. Run `nix run .#check-arch` — dependency budget enforcement~~ done - green inside #verify since (layer keys repaired)
~~32. Run `nix run .#check-duplication` — no-new-clones gate~~ done - baseline re-pinned; green since
~~33. Verify no new production deps were added by the lint fixes~~ done - no new production deps (check-arch green)

### Integration tests

~~34. Run `nix run .#test-integration` or at minimum `nix run .#test` ~~ done - verify green 3x (SQLite+Pebble+bbolt+DuckDB legs pass)
~~35. Verify SQLite + Pebble + bbolt + DuckDB backends still pass~~ done - same

### Documentation

~~36. Update the status report with final verification results~~ done - final results captured by the 01-02 post-merge review (archived) and this pass
~~37. Consider a CHANGELOG entry for the lint fixes~~ done - CHANGELOG [Unreleased] 'repo gates' entry covers the lint-fix lineage (444be10a7)
38. Document the recovery procedure for future reference <- NOT-DO - lessons folded into AGENTS gotchas (worktree rule, concurrent-session verify races); no dedicated doc needed

### Hardening

39. Add a pre-commit hook that blocks commits when `git stash list` is non-empty (prevent orphaned stashes)
40. Consider a CI check that prevents two agents from pushing to the same branch simultaneously
    ~~41. Add `metaengine/store.go` and `execute.go` file-size check to CI (they're at 672/566 lines on master — violations)~~ done - 350-line limit is CI-enforced already (internal contract #1)
    ~~42. Add a test that verifies `go.work` module list matches directories with `go.mod` files~~ done - TestEveryGoModDirIsInModulesList + workspace meta-tests enforce this

### Broader code quality

43. Audit all `//nolint` directives across the codebase — many may be stale or unnecessary <- OPEN. nolint staleness audit not done - low priority
44. Check if `idempotency/alias.go` should also get a golangci.yml exclusion (it re-exports deprecated symbols) <- NOT-DO - idempotency/ shim deleted at 5127039da
45. Check if `retry/alias.go` has similar lint issues <- NOT-DO - retry/ shim deleted at 5127039da
46. Review whether `flightrecorder/alias.go` needs the same exclusion as `codec/alias.go` <- NOT-DO - flightrecorder/ shim deleted at 5127039da
47. Consider adding `depguard` rules to prevent importing `go-cqrs-lite/codec/v4` in new code (should use `go-codec` directly) <- NOT-DO - shims deleted; nothing to ban (matches the 10-24 resolution)

### Soak tests

~~48. Run the 50K-event smoke soak: `go test -tags "goexperiment.jsonv2" -run TestSoak ./metaengine/...`~~ done - soak legs green in every verify since
~~49. Run the full soak suite without `-race` to verify no regressions~~ done - same

### Release readiness

50. After all the above: `nix run .#verify` + `nix run .#vulncheck` before any tag <- OPEN. TODO_LIST 'Release / Tagging' (pre-tag checklist: verify + vulncheck)

---

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Master branch strategy:** Should I force `master` to point at `2b72de54f` (effectively deleting the 3 damaging commits + 2 auto-daemon commits), or do you want to preserve any of the concurrent agent's work via cherry-pick? The damaging commits are `a6613ef0d`, `7942a61bb`, `9249d1301`, `c4aabc337`, `04be9f6dc`, `11e7746d6`.

2. **The stash (`stash@{0}`):** It contains uncommitted working-tree changes that were on `master` when I created the recovery branch — a mix of docs status file edits and metaengine file modifications from the concurrent agent. Should I inspect and salvage anything from it, or drop it?

3. **Is the full `nix run .#lint` available?** The nix daemon was down (`cannot connect to socket at '/nix/var/nix/daemon-socket/socket'`). I verified with direct `golangci-lint` per-module instead. Should I try to restart the daemon and run the authoritative nix lint path, or is the per-module verification sufficient for now?

---

## Resolution (2026-08-15, docs-health pass)

44 of 50 items carry verdicts. The recovery story closed: the branch landed
via merge `0e4a16a8a` (modules restored, mega-files died with the broken
snapshot), full lint is 76/76 clean since `444be10a7`, and the verify gate
went fully green at `5f2198189` (3x since). The alias.go block (13, 44-47)
was mooted by the ADR-0128 shim deletion. Open-unrouted: 14 (two-agents
workspace rule in AGENTS), 22 (exported idempotency test helper), 23/43
(nolint staleness audit), 39-40 (stash/branch-protection hooks). g) Q1
answered by the merge; Q2 OPEN (stash still exists, routed to junk-cleanup
item); Q3 answered by events (nix lint authoritative since). Stays active.
