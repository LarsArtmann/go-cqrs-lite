# Self-Review & Improvement Plan

> **Date**: 2026-06-08 22:00 UTC+2
> **Session Context**: Post-golden-test session reflection and improvement

## Honest Self-Review

### What I Forgot / Did Wrong

1. **No incremental commits** — All golden tests were committed in one giant blob. Should have committed per-module.
2. **Duplicated `assertGolden` 13 times** — Every module got its own copy of the same 24-line helper. Classic copy-paste violation.
3. **Didn't check for existing golden tests first** — `codec/` already had them; I reported "ALREADY DONE" but should have known before starting.
4. **Shallow tests** — `turso/` golden test just snapshots an error string. `otel/` snapshots constants. These catch accidental renames but don't test behavior.
5. **Named test file `coverage_test.go`** — Tests should describe WHAT they test, not the metric goal.
6. **Signing golden test** — Overwrote an existing (broken) `golden_test.go` without noting what was wrong with it.
7. **Didn't consider go-snaps** — The status doc mentioned it 13 times. I used the manual flag pattern instead.
8. **listing.Page workaround** — Had to create a wrapper struct because `Page[T]` lacks json tags. Should have fixed the root cause.

### What Could Still Be Improved

| Area | Description | Impact |
|------|-------------|--------|
| Golden test dedup | 13 copies of `assertGolden` | Medium |
| go-snaps adoption | Replace manual pattern with library | Medium |
| listing type models | Add json tags to `Page[T]`, `AggregateRef`; rename to `AggregateListing` | High |
| snapshot type model | Add json tags to `Snapshot` | Low |
| Error re-export cleanup | Remove unused re-exports from `command/errors.go` | Medium |
| Test file naming | Rename `coverage_test.go` to describe what it tests | Low |

## Execution Plan (sorted by impact/effort)

### Phase 1: Type Model Fixes (High impact, low effort)
1. Add json tags to `listing.Page[T]` and `listing.AggregateRef`
2. Rename `listing.AggregateRef` → `listing.AggregateListing`
3. Add json tags to `snapshot.Snapshot`
4. Update golden tests to use the fixed types directly

### Phase 2: Golden Test Deduplication (Medium impact, medium effort)
5. Create `event/eventtest/golden.go` shared helper
6. Migrate all 13 golden_test.go files to use shared helper
7. Delete per-module `assertGolden` copies

### Phase 3: Cleanup (Low impact, low effort)
8. Rename `command/coverage_test.go` → `command/errors_test.go`
9. Remove unused error re-exports or make them canonical
10. Update CONTRIBUTING.md with golden test conventions

### Deferred (requires more thought)
- go-snaps migration: requires adding dep to 13 go.mod files. Defer.
- Error re-export consolidation: affects API surface. Defer to v3.
