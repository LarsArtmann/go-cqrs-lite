# Status Report: Integration Test Infrastructure Hardening — Self-Review

**Date:** 2026-08-03 09:36
**Session goal:** Verify and close gaps from prior session's Nix integration test work (48-task Pareto plan), then self-review brutally.

---

## a) FULLY DONE (verified this session)

| #   | Item                                                          | Verification                                                                                                                               |
| --- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **Metaengine boundary key validation committed** (`cbc572c8`) | Tests pass (2 new tests, full suite green). ReadMembership now validates key type at `execute.go:112`.                                     |
| 2   | **Stale ADR-0094 → ADR-0095 fixed**                           | 3 occurrences in plan doc corrected (lines 61, 204, 514).                                                                                  |
| 3   | **Plan doc M01-M48 status corrected**                         | 18 task statuses updated from NOT DONE to DONE/RESEARCHED/N/A. Accurate reflection of actual work state.                                   |
| 4   | **AGENTS.md Quick Reference updated**                         | Added `integration-all` and `verify-integration` flake apps row.                                                                           |
| 5   | **doc-check verified GREEN**                                  | 519 references valid across all modified markdown.                                                                                         |
| 6   | **api-stability golden regenerated**                          | 3 new exports captured: `WatchTyped`, `WatchTypedWithSeq`, `ErrKeyTypeMismatch`. Test passes.                                              |
| 7   | **verify-fast GREEN (2x)**                                    | Initial run + final run after all changes. All 64 modules pass.                                                                            |
| 8   | **Ephemeral PG verified working**                             | `bash scripts/ephemeral-pg.sh -short` — PostgreSQL starts, tests run.                                                                      |
| 9   | **Standalone pg-vm + mysql-vm packages build**                | `nix build .#pg-vm` and `.#mysql-vm` — both succeed.                                                                                       |
| 10  | **M24: Timeout on ephemeral-pg.sh**                           | `TEST_TIMEOUT` env var (default 300s), `timeout` command wrapper.                                                                          |
| 11  | **M22: Shellcheck clean**                                     | All 3 scripts pass `shellcheck`. Fixed SC2086 via array-based arg passing in ephemeral-pg.sh.                                              |
| 12  | **M26: --keep-alive flag**                                    | Both VM scripts — VM stays alive after tests for interactive debugging.                                                                    |
| 13  | **M20: Example outputs in testing-guide**                     | Added ephemeral PG output and VM check output examples.                                                                                    |
| 14  | **TODO_LIST updated with all M34-M48 tasks**                  | 6 missing P4 tasks added (M36, M37, M42, M45, M47, M48). 3 completed tasks marked [x].                                                     |
| 15  | **Daemon's calibration fix verified**                         | `c45b39c8` fixed the `CalibrateEngine` persistence bug that was causing the flaky `TestCalibrateEngine` failure. Test now passes reliably. |

---

## b) PARTIALLY DONE

| #   | Item                                  | What's done                                                                         | What's missing                                                                                                                                                                           |
| --- | ------------------------------------- | ----------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **M14: systemd-nspawn investigation** | Marked RESEARCHED in plan doc                                                       | No actual prototype attempted. Only concluded "not stable in nixpkgs for runNixOSTest." Should attempt `NspawnMachine` in a test branch.                                                 |
| 2   | **CI YAML** (prior session)           | Matrix strategy + ephemeral-pg-tests job added, YAML valid                          | Never run in actual GitHub Actions. May fail on runner due to permissions, caching, or Nix installer issues.                                                                             |
| 3   | **VM launcher scripts E2E**           | Scripts verified working in prior session (`vm-pg.sh -short`, `vm-mysql.sh -short`) | Not re-tested this session after the `--keep-alive` flag addition. Low risk (additive change), but not verified.                                                                         |
| 4   | **Plan doc accuracy**                 | M01-M33 statuses corrected                                                          | M34-M48 (P4 tasks) still all show "NOT DONE" in plan doc table — correct, but the micro-task decomposition sections below each M-header were not updated with actual status annotations. |

---

## c) NOT STARTED

| #   | Item                                | Why                                                                                                                                                                                                                                                        |
| --- | ----------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **M02: Push to remote**             | 29 commits ahead of `origin/master`. Not pushed — user must approve.                                                                                                                                                                                       |
| 2   | **M34-M48: All P4 future tasks**    | Correctly deferred. 15 tasks documented in TODO_LIST. Includes DuckDB VM, SQLite WAL VM, Turso VM, Pebble VM, projectionhost crash-restart, scheduling timers, PostgresBus VM, contract suite across backends, Redis/NATS, test-integration.sh aggregator. |
| 3   | **CI badge for VM tests (M33)**     | Marked N/A — existing CI badge covers the whole workflow. Could add a separate badge if desired.                                                                                                                                                           |
| 4   | **Remove redundant status reports** | `docs/status/2026-08-03_09-00_nix-integration-test-session2-final.md` is a duplicate of the brutal review. Still present.                                                                                                                                  |

---

## d) TOTALLY FUCKED UP / SERIOUSLY WRONG

| #   | What                                                                                             | Impact                                                                                                                                                                                  | Root cause                                                                                                                                                                                                                                                                                                                                                                       |
| --- | ------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **gocyclo lint warning on `executeQueryInner` (complexity 31, limit 30)**                        | Lint noise. Not a build failure, but CI lint step will report it.                                                                                                                       | The boundary key validation added 2 lines that pushed the function from ~29 to 31 complexity. The auto-commit daemon (`c45b39c8`) then refactored `executeQueryInner` by extracting `executePointLookup`, `executeMembership`, and `checkKeyTypeMatch` — but the diff shows the refactored version still has the same complexity somehow. **Still uncommitted in working tree.** |
| 2   | **Untracked test artifact: `idempotency/sqlstore/testdata/rapid/TestProperty_SQLiteTTLExpiry/`** | Garbage in the working tree. A rapid property test failure dump file was not cleaned up.                                                                                                | The `.fail` file from a flaky `TestProperty_SQLiteTTLExpiry` test run was written to `testdata/` and never removed. This should be in `.gitignore` or cleaned up.                                                                                                                                                                                                                |
| 3   | **3 untracked + 18 modified files in working tree from auto-commit daemon races**                | Working tree is messy. The daemon committed some of my changes (into `c45b39c8`) but then made MORE changes (formatting, ADR-0096 edits, status report edits) that are now uncommitted. | The auto-commit daemon and I were both editing simultaneously. Some of my edits were included in its commit; others are orphaned in the working tree. The diff shows 600 insertions / 491 deletions across 18 files — a mix of my changes and the daemon's.                                                                                                                      |
| 4   | **No isolation between my work and daemon commits**                                              | Cannot cleanly attribute which changes are mine vs the daemon's.                                                                                                                        | The daemon races with manual edits. I attempted to commit `metaengine/execute.go + boundary_keys_test.go` but got `fatal: cannot lock ref 'HEAD'` because the daemon had moved HEAD underneath me. The daemon then committed my changes with its own message.                                                                                                                    |

---

## e) WHAT WE SHOULD IMPROVE

| #   | Area                                                                             | Problem                                                                                                                            | Fix                                                                                                                                                                                                                                          |
| --- | -------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **gocyclo on executeQueryInner**                                                 | Complexity 31 > 30 limit. Lint warning persists.                                                                                   | Extract more sub-functions or raise the gocyclo limit for this specific function via `//nolint:gocyclo` (after verifying it's genuinely irreducible). The daemon already extracted 3 functions but complexity didn't drop — investigate why. |
| 2   | **Test artifact cleanup**                                                        | `testdata/rapid/*.fail` files leak into working tree                                                                               | Add `**/testdata/rapid/*.fail` to `.gitignore` or add cleanup in `TestMain`.                                                                                                                                                                 |
| 3   | **Daemon race conditions**                                                       | Auto-commit daemon races with manual work, causing `HEAD` lock failures and orphaned diffs                                         | Either disable the daemon during active sessions, or always work in a `git worktree` to isolate from the daemon.                                                                                                                             |
| 4   | **Ephemeral PG integration test failure**                                        | The ephemeral PG script reported "Some integration tests failed" even though pgengine tests passed                                 | The `storage` module integration tests failed (pre-existing). Should investigate which specific tests fail and either fix or skip them with clear documentation.                                                                             |
| 5   | **Status report proliferation**                                                  | 3 status reports from session 2 (08-27, 08-59, 09-00) cover the same ground. Plus 2 more untracked from the daemon (09-26, 09-35). | Consolidate to one report per session. Delete redundant reports.                                                                                                                                                                             |
| 6   | **Plan doc micro-task sections stale**                                           | The master table was updated but the detailed micro-task breakdowns (M14.1, M14.2, etc.) still say NOT DONE for completed tasks    | Update the micro-task sections to match the master table.                                                                                                                                                                                    |
| 7   | **M14 systemd-nspawn was dismissed without trying**                              | Marked "RESEARCHED (not stable)" without actually prototyping                                                                      | `NspawnMachine` exists in the NixOS test driver. Should actually try it in a branch.                                                                                                                                                         |
| 8   | **No integration test for `integration-all` or `verify-integration` flake apps** | Added two new flake apps but never ran them                                                                                        | Run `nix run .#integration-all` and `nix run .#verify-integration` to confirm they work.                                                                                                                                                     |
| 9   | **docs/README.md has 192 line diff**                                             | The daemon reformatted this file extensively                                                                                       | Review the diff — the reformatting may have broken table alignment or link structure.                                                                                                                                                        |
| 10  | **ADR-0096 appeared**                                                            | A new ADR (`0096-iroh-distributed-engine-bridge-evaluation.md`) was created by the daemon with a 20-line diff                      | This is outside the scope of integration test work. Should be reviewed for accuracy.                                                                                                                                                         |

---

## f) Up to 50 Things to Get Done Next

### Critical (blocking trust)

| #   | Task                                                                                               | Effort |
| --- | -------------------------------------------------------------------------------------------------- | ------ |
| 1   | Investigate and clean up uncommitted working tree — decide which daemon changes to keep vs discard | 10min  |
| 2   | Run `nix run .#integration-all` to verify the aggregator app works                                 | 5min   |
| 3   | Run `nix run .#verify-integration` to verify the composite gate works                              | 5min   |
| 4   | Fix gocyclo warning on `executeQueryInner` (complexity 31)                                         | 10min  |
| 5   | Add `**/testdata/rapid/*.fail` to `.gitignore`                                                     | 2min   |
| 6   | Push 29 commits to `origin/master` (requires user approval)                                        | 1min   |
| 7   | Investigate which storage integration tests fail under ephemeral PG                                | 15min  |

### High value

| #   | Task                                                                                     | Effort |
| --- | ---------------------------------------------------------------------------------------- | ------ |
| 8   | Remove redundant status report `2026-08-03_09-00_nix-integration-test-session2-final.md` | 1min   |
| 9   | Review daemon's ADR-0096 (iroh distributed engine bridge) for accuracy                   | 10min  |
| 10  | Review daemon's `docs/README.md` reformatting (192-line diff)                            | 5min   |
| 11  | Update plan doc micro-task sections (M15.1, M16.1, etc.) to match master table           | 10min  |
| 12  | Run `nix run .#verify` (full gate, not just verify-fast)                                 | 5min   |
| 13  | Run `nix flake check` to verify all flake outputs build                                  | 3min   |
| 14  | Prototype `NspawnMachine` for M14 (systemd-nspawn containers) in a worktree              | 20min  |
| 15  | Add `scripts/test-integration.sh` aggregator (M48 — auto-detect best strategy)           | 12min  |

### Medium value

| #   | Task                                                                    | Effort |
| --- | ----------------------------------------------------------------------- | ------ |
| 16  | Add CI step that runs `nix run .#verify-integration`                    | 5min   |
| 17  | Document the NixOS firewall gotcha in AGENTS.md troubleshooting section | 5min   |
| 18  | Add `pkill -f qemu` to VM script startup to prevent port conflicts      | 3min   |
| 19  | Add `SELECT 1` health check before running tests in ephemeral-pg.sh     | 5min   |
| 20  | M34: Test ephemeral-pg.sh on macOS                                      | 15min  |
| 21  | M35: Cache ephemeral PG data dir for faster startup                     | 10min  |
| 22  | M36: Performance profiling ephemeral PG vs testcontainers               | 15min  |
| 23  | M37: Explore `nixos-container` as lighter-weight VM alternative         | 12min  |
| 24  | Add connection retry with exponential backoff (replace linear polling)  | 10min  |
| 25  | Add VM serial console log capture to a file for CI artifacts            | 10min  |
| 26  | Run actionlint on CI YAML (beyond just YAML validation)                 | 5min   |

### Lower priority but worthwhile

| #   | Task                                                                              | Effort |
| --- | --------------------------------------------------------------------------------- | ------ |
| 27  | M38: DuckDB CGo VM test (GCC in VM)                                               | 20min  |
| 28  | M39: SQLite WAL concurrency VM test                                               | 15min  |
| 29  | M40: Turso sync VM test (real libSQL server)                                      | 20min  |
| 30  | M41: Run Go test binaries inside QEMU VM                                          | 30min  |
| 31  | M42: Pebble backup/restore lifecycle VM test                                      | 15min  |
| 32  | M43: projectionhost crash-restart PG integration test                             | 20min  |
| 33  | M44: scheduling module durable timers across restarts test                        | 20min  |
| 34  | M45: storage.PostgresBus Go code inside NixOS VM                                  | 20min  |
| 35  | M46: Contract test suite across ALL backends in VMs                               | 30min  |
| 36  | M47: Ephemeral Redis/NATS for Watermill adapter testing                           | 20min  |
| 37  | Consolidate all session 2 status reports into a single retrospective              | 10min  |
| 38  | Add integration test section to README.md for discoverability                     | 5min   |
| 39  | Add `nix run .#integration-pg-vm -- --keep-alive` example to CONTRIBUTING.md      | 3min   |
| 40  | Verify CI matrix strategy actually parallelizes (check GitHub Actions runs)       | 5min   |
| 41  | Add timeout to VM launcher scripts (currently can hang forever if VM never boots) | 5min   |
| 42  | Add `--dry-run` flag to VM scripts (build driver but don't boot)                  | 8min   |

### Polish

| #   | Task                                                                                 | Effort |
| --- | ------------------------------------------------------------------------------------ | ------ |
| 43  | Pin GitHub Actions to SHA (BuildFlow flagged 78 actions pinned to tags)              | 15min  |
| 44  | Fix `go.work` use-path mismatch warnings (57 paths, BuildFlow preflight)             | 10min  |
| 45  | Add `shellcheck` to devShell so it's available without `nix-shell -p`                | 3min   |
| 46  | Add `.shellcheckrc` for consistent linting config                                    | 3min   |
| 47  | M32 already done (magic-nix-cache) — verify it actually caches VM builds             | 5min   |
| 48  | Add integration test coverage to `nix run .#check-coverage`                          | 5min   |
| 49  | Document the auto-commit daemon's behavior in AGENTS.md (it races with manual edits) | 5min   |
| 50  | Review `metaengine/sse_replay_test.go` daemon formatting change (line split)         | 2min   |

---

## g) Questions I Cannot Answer Myself

1. **Should I push the 29 unpushed commits to `origin/master` now?** The repo is 29 commits ahead. Some of those are the auto-commit daemon's work (metaengine calibration fix, ADR-0096, formatting). I don't know if you want to review the daemon's changes before pushing, or if auto-pushing is your normal workflow.

2. **The auto-commit daemon made significant changes I didn't author (ADR-0096, docs/README.md 192-line reformat, metaengine execute.go refactor, calibration fix). Should I review and vet these, or trust the daemon?** The daemon's commit `c45b39c8` mixes my work with its own (calibration persistence fix, dead code removal, formatting). I cannot tell which parts you'd want to keep vs investigate.

3. **The working tree has 18 modified + 3 untracked files — a mix of my edits, the daemon's edits, and test artifacts. Should I commit everything, selectively stage only my changes, or let the daemon handle it?** The daemon will likely auto-commit these, but the mix makes it hard to attribute authorship cleanly.
