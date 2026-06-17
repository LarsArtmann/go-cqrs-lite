# Comprehensive Status Update — 2026-06-17 02:20 CEST

**Project:** go-cqrs-lite  
**Branch:** consolidate-catalog  
**Date:** 2026-06-17 02:20:28 CEST  
**Reporter:** Crush (automated status report)

---

## Executive Summary

We are mid-stream on the `consolidate-catalog` branch. A large volume of cleanup, documentation, and test work has landed, but the branch is **NOT ready for merge**. The headline risk is a growing set of "ghost systems" and deferred integration work — most notably the `kv/` module, which has zero consumers. The most recent technical decision is to build a `pebble/adapter.go` mapping `*pebble.DB` to `kv.Store`, but that work is still in design and has not been committed yet.

CI is green for normal tests and lint, but `nix run .#test-race` is red in `turso/indexing` due to pre-existing race conditions. Code quality is otherwise high: zero lint issues, 41/41 test packages passing in normal mode.

---

## a) FULLY DONE

### Architecture & Consolidation

- Catalog consolidation: 5 catalog sub-modules collapsed into a single `catalog/` module with packages under it (ADR-0017 implemented).
- `kv/` module introduced as Layer 0 with `Store`, `Reader`, `Writer`, `Iterator`, `Batch` interfaces and in-memory `MemStore` reference implementation.
- ADR-0022 written documenting the KV abstraction rationale and iterator semantics.
- Layer-budget script updated to account for new module graph.

### CI / Build Infrastructure

- `.github/workflows/ci.yml` updated with `replace-directives` validation job.
- `cmd/api-stability` added to per-module test matrix.
- `kv/` module added to CI test matrix and release workflow.
- `scripts/check-module-layers.sh` budgets adjusted after catalog consolidation and kv introduction.
- `scripts/check-replace-directives.sh` passes.
- `nix run .#build`, `nix run .#test`, `nix run .#lint`, `nix run .#vet`, `nix run .#check-layers` all pass.

### Testing

- Reactive bus tests added: `command/reactive_test.go`, `query/reactive_test.go`.
- Pebble integration tests added: `integration/pebble_test.go`.
- All 41 test packages passing in normal mode (`go test ./...` and `nix run .#test`).

### Reactive Bus Extensions

- `command/` reactive `CommandBus`, `Publisher`, `Subscriber`, `PublishMiddleware` documented and tested.
- `query/` reactive `QueryBus` documented and tested.
- `command/doc.go` and `query/doc.go` updated with reactive bus examples.
- `Compose` helper re-exported via `command/errors.go` and `query/errors.go`.

### Documentation

- `CHANGELOG.md` updated with unreleased entries for reactive buses, PebbleBackend, SQLBackend, kv module, Compose, integration tests.
- `FEATURES.md` updated with correct module count and kv/reactive bus status.
- `TODO_LIST.md` populated with Tier 5 open items.
- `AGENTS.md` updated with kv/ context and reactive bus notes.
- Previous comprehensive status reports written to `docs/status/`.

### Miscellaneous Cleanup

- `.gitignore` updated to ignore `/cmd/api-stability/api-stability` binary.
- `cmd/cqrs-gen/main.go` linted with `//nolint` for `genSpecs` global.
- `turso/indexing/advisor_data.go` lint comments added for static data tables.
- `go.work.sum` and module sums kept in sync.

---

## b) PARTIALLY DONE

### KV Module Integration into Pebble — IN PROGRESS

- **Design complete:** mapping `*pebble.DB` → `kv.Store` is clear.
- **Files not yet created:** `pebble/adapter.go`, `pebble/adapter_test.go`.
- **Dependencies not yet wired:** `pebble/go.mod` does not yet require `kv/v2`.
- **Layer budget not yet raised:** `scripts/check-module-layers.sh` `DEP_BUDGET[pebble]` may need bump from 7 to 8.

This is the current active work item. The previous session was interrupted before implementation started.

### Self-Review Findings

- Brutal self-review initiated via `brutal-self-review` skill.
- `kv/` ghost system identified and selected for remediation (option B).
- Other self-review findings remain unaddressed:
  - Compose tests (sleep-based integration test flagged as brittle).
  - AGENTS.md update for kv/ integration (partially done but may need more).
  - Any additional ghost systems or split brains discovered during review not yet triaged.

### Race Condition Remediation

- `nix run .#test-race` still fails in `turso/indexing` checkpoint scheduler tests.
- Failure is pre-existing and unrelated to current work, but remains un-fixed.

---

## c) NOT STARTED

1. Implement `pebble/adapter.go`.
2. Add `pebble/adapter_test.go` with CRUD/iteration/batch/error-mapping tests.
3. Update `pebble/go.mod` to depend on `kv/v2` and add replace directive.
4. Adjust `pebble` dependency budget if needed.
5. Run full verification after adapter work (lint, layers, tests, race optional).
6. Commit adapter work.
7. Push branch changes.
8. Address remaining self-review items (Compose tests, brittle integration test, AGENTS.md refresh).
9. Merge `consolidate-catalog` back to `master`.
10. Tag release for v2.4.0 (or whatever next version).
11. Investigate and fix `turso/indexing` race failures.
12. Full integration of `kv.Store` into `pebble.EventStore`, `SnapshotStore`, `CheckpointStore` (optional follow-up).
13. Module README updates for `kv/` and `pebble/`.
14. Performance regression baseline update if adapter changes allocations.
15. Update `docs/planning/` with completed adapter milestone.
16. Close related TODOs in `TODO_LIST.md`.
17. Verify no other ghost systems exist (follow-up self-review pass).
18. Update CHANGELOG with adapter completion.
19. Update FEATURES.md adapter status.
20. Add ADR for pebble-kv adapter if architecture significant.

---

## d) TOTALLY FUCKED UP!

- **`turso/indexing` race test failures.** `nix run .#test-race` fails in checkpoint scheduler tests. This is a real, pre-existing quality issue. It is not caused by current work but it blocks any claim of "race-clean" CI.
- **Uncommitted branch state.** The branch has a mix of tracked modifications, staged catalog docs, and new untracked test files. This is workable but not merge-ready. The most important unfinished piece (`pebble/adapter.go`) exists only in design notes, not on disk.
- **`kv/` is still a ghost system.** Until `pebble/adapter.go` lands, the module has zero consumers and is therefore unvalidated as a public API.

---

## e) WHAT WE SHOULD IMPROVE!

1. **Finish the kv→pebble adapter immediately.** It is the highest-value open item: it validates a public module and removes a ghost system.
2. **Fix race failures in `turso/indexing`.** Race-clean CI is a credible quality signal for a storage library.
3. **Tighten branch hygiene.** The working tree mixes many concerns. After adapter work, consider rebasing/squashing or at least grouping commits logically before merge.
4. **Make self-review a recurring gate.** The `brutal-self-review` skill surfaced real issues. Run it again after adapter work.
5. **Add consumer-driven contract tests for `kv.Store`.** Once pebble adapter exists, run the `kv/` test suite against the pebble adapter to prove interface conformance.
6. **Reduce reliance on sleep-based integration tests.** They are flaky and slow; replace with deterministic synchronization where possible.
7. **Document ownership semantics.** The adapter must clarify whether `kv.Store.Close()` closes the underlying `*pebble.DB`.
8. **ADR for kv/pebble integration.** Architecture decision should be recorded.
9. **Update dependency budgets consciously.** Do not just raise budgets; verify the added dependency is pulling its weight.
10. **Re-run full race test after adapter work.** New concurrency code can introduce races.
11. **Consider whether `kv.Store` should be used by `pebble.EventStore` et al.** A thin adapter is step one; internal adoption is step two.
12. **Improve CHANGELOG discipline.** Keep unreleased section updated per commit.
13. **Ensure all new public APIs have pkg.go.dev-friendly doc.go examples.**
14. **Revisit TODO_LIST.md priority ordering.** Some items may be obsolete now.
15. **Verify module layer graph after adapter dependency is added.**

---

## f) Top #25 Things We Should Get Done Next

1. Implement `pebble/adapter.go` mapping `*pebble.DB` to `kv.Store`.
2. Add `pebble/adapter_test.go` with full coverage.
3. Update `pebble/go.mod` with `kv/v2` dependency and replace directive.
4. Adjust dependency budget in `scripts/check-module-layers.sh`.
5. Run `nix run .#test`, `nix run .#lint`, `nix run .#check-layers`.
6. Commit adapter work with detailed message.
7. Push branch to remote.
8. Fix `turso/indexing` race failures.
9. Re-run `nix run .#test-race` and confirm clean.
10. Address Compose test gaps.
11. Replace or harden sleep-based integration tests.
12. Update AGENTS.md with final adapter decisions.
13. Update CHANGELOG.md with adapter completion.
14. Update FEATURES.md adapter status.
15. Add ADR-0023 for pebble-kv adapter.
16. Run `brutal-self-review` again to catch new issues.
17. Verify no other ghost systems remain.
18. Integrate `kv.Store` usage into `pebble.EventStore`/`SnapshotStore`/`CheckpointStore` if beneficial.
19. Add module READMEs for `kv/` and `pebble/`.
20. Update `TODO_LIST.md` and close completed items.
21. Squash/rebase branch commits into logical groups.
22. Open PR / prepare merge to `master`.
23. Tag next release.
24. Update performance baseline if needed.
25. Update `docs/planning/` with completed milestones.

---

## g) Top #1 Question I Cannot Figure Out Myself

**Is the `kv.Store` abstraction intended to be a public, first-class module that consumers use directly (e.g., for custom projections or read models), or is it primarily an internal implementation detail for pebble/storage backends?**

The answer determines:

- Whether `kv.Store` needs a rich public API (transactions, snapshots, range scans) or just the current thin interface.
- Whether `pebble.Adapter` should live in `pebble/` (as planned) or if a separate `kv-pebble/` module is warranted.
- Whether the adapter should own the `*pebble.DB` lifecycle or always borrow it.
- Whether future modules like `storage/` should also consume `kv.Store`.

The interfaces currently look consumer-facing, but there is no consumer documentation, examples, or ADR scope beyond the abstraction itself. Clarifying the intended audience will prevent either over-engineering the adapter or under-investing in the public API.

---

## Current Working Tree Snapshot

```
 M .github/workflows/ci.yml
 M .gitignore
 M CHANGELOG.md
 M FEATURES.md
 M TODO_LIST.md
 M cmd/cqrs-gen/main.go
 M command/doc.go
 M command/errors.go
 M go.work.sum
 M integration/go.mod
 M integration/go.sum
 M query/doc.go
 M query/errors.go
 M scripts/check-module-layers.sh
 M turso/indexing/advisor_data.go
?? command/reactive_test.go
?? integration/pebble_test.go
?? query/reactive_test.go
```

**Test status:** 41/41 packages pass in normal mode.  
**Lint status:** 0 issues.  
**Race status:** `turso/indexing` failures pre-existing.  
**Layer check:** Passes.

---

_End of status report._
