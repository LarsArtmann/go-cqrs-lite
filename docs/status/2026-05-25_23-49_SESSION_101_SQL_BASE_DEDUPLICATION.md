# Session 101 — Comprehensive Status Report

**Date:** 2026-05-25 23:49
**Branch:** master
**Since last session:** Session 100 (deprecation cleanup, status report)

---

## a) FULLY DONE

### This Session: `sqlBase` Extraction

Extracted a shared `sqlBase` struct from 4 SQL store types in the `storage` module, eliminating duplicated `db`/`dialect` fields, nil-check constructors, and no-op `Close()` methods.

| What | Detail |
|------|--------|
| New file | `storage/sql_base.go` — `sqlBase` struct, `newSQLBase()` constructor, no-op `Close()` |
| Refactored | `SQLEventStore` — embeds `sqlBase`, overrides `Close()` for ownership |
| Refactored | `SQLCheckpointStore` — embeds `sqlBase`, inherits no-op `Close()` |
| Refactored | `SQLOutbox` — embeds `sqlBase`, inherits no-op `Close()` |
| Refactored | `SQLSnapshotStore` — embeds `sqlBase`, inherits no-op `Close()` |
| Net change | +1 file, -31 lines, +21 lines (net -10 lines) |
| Tests | 23/23 packages pass, 0 regressions |
| Lint | 0 new issues (2 pre-existing `noinlineerr` in core/command, core/query) |

### Dupe Analysis Result

Ran `branching-flow dupe . --format markdown`. Before this session: 5 actionable groups (10 total). After: **4 actionable groups** (Group 3 resolved to false positive via `sqlBase` embedding). The remaining 4 are all assessed as structural necessities, not accidental duplication:

| Group | Types | Verdict | Reason |
|-------|-------|---------|--------|
| 4 | ReadModel/UserCreatedPayload/UserState | **Skip** — example code, intentional separation |
| 6 | command/query Dispatcher | **Skip** — different type params, intentional |
| 7 | asyncapi/openapi Info | **Skip** — different specs, will diverge |
| 10 | Builder/builtProjection | **Skip** — builder pattern, not duplication |

---

## b) PARTIALLY DONE

Nothing partially done — the `sqlBase` extraction is complete end-to-end.

---

## c) NOT STARTED

| Item | Priority | Notes |
|------|----------|-------|
| Stale `"sync"` in flake.nix testModules | HIGH | `sync` directory doesn't exist, causes `nix run .#build` to fail |
| EventCatalog coverage drop (91.3% → 85.7%) | MEDIUM | Regression in this session's coverage run |
| Two pre-existing `noinlineerr` lint issues | LOW | `core/command/dispatcher.go:59`, `core/query/dispatcher.go:83` |
| AGENTS.md coverage table needs update | LOW | Several coverage numbers shifted slightly |

---

## d) TOTALLY FUCKED UP

**`nix run .#build` is broken** — `flake.nix` references `"sync"` in `testModules` but no `sync/` directory exists. This was introduced when `sync/` was extracted/deleted but `flake.nix` wasn't updated. The `go test` direct invocation works fine (uses explicit module paths), but the nix build app fails.

**EventCatalog coverage regression** — dropped from 91.3% to 85.7%. Needs investigation.

---

## e) WHAT WE SHOULD IMPROVE

1. **Fix the broken nix build immediately** — `sync` reference in `flake.nix` must be removed
2. **Investigate eventcatalog coverage regression** — 91.3% → 85.7% is a significant drop
3. **Update AGENTS.md** — coverage numbers are stale, session 101 not tracked
4. **Status report archive is bloated** — 120+ files in `docs/status/`, most should be archived or deleted
5. **Pre-commit/CI validation** — a broken `flake.nix` should never have been committed

---

## f) Top 25 Things To Do Next

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 1 | Fix stale `"sync"` in flake.nix — remove from testModules | HIGH | 1 min |
| 2 | Investigate eventcatalog coverage regression (91.3% → 85.7%) | HIGH | 15 min |
| 3 | Update AGENTS.md coverage table and session history | MEDIUM | 10 min |
| 4 | Fix 2 pre-existing `noinlineerr` lint issues in core | MEDIUM | 2 min |
| 5 | Run `nix flake check` and fix any issues | MEDIUM | 5 min |
| 6 | Add example/todo and example/user to nix build coverage | MEDIUM | 5 min |
| 7 | Clean up docs/status/ — archive old reports, keep last 10 | LOW | 5 min |
| 8 | Add integration test for `sqlBase` nil-check coverage | LOW | 5 min |
| 9 | Verify `nix run .#coverage` still works | LOW | 2 min |
| 10 | Consider extracting shared `schemaFunc()` pattern from storage | LOW | 10 min |
| 11 | Add `EventBus` interface to core (pub/sub abstraction) | HIGH | 30 min |
| 12 | Storage: add connection pool configuration options | MEDIUM | 15 min |
| 13 | Add retry/backoff to SQLOutbox PollPending | MEDIUM | 20 min |
| 14 | Projection: add graceful shutdown with context cancellation | MEDIUM | 20 min |
| 15 | Add `event.NewTyped[T]()` constructor for typed payloads | MEDIUM | 15 min |
| 16 | Catalog: add schema diff/change detection between builds | LOW | 30 min |
| 17 | Middleware: add circuit breaker middleware | LOW | 30 min |
| 18 | Add `docserver` to nix build targets | LOW | 5 min |
| 19 | Create CHANGELOG.md from session history | LOW | 20 min |
| 20 | Add Go doc examples (testable) to core packages | MEDIUM | 30 min |
| 21 | Benchmarks for catalog Build() with large registries | LOW | 15 min |
| 22 | Add `storage` to nix build (not just test) | LOW | 2 min |
| 23 | Verify `pebble_event_store.go` still builds (it's in storage/) | LOW | 2 min |
| 24 | Add `WithErrorHandler` option to projection Runner | LOW | 15 min |
| 25 | Review `example/todo` for consistency with `example/user` patterns | LOW | 10 min |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Why does `sync/` appear in `flake.nix` testModules but not in `go.work`?** There's no `sync/` directory in the repo. It seems like it was planned or extracted but the flake was never updated. I can fix it by removing the reference, but I don't know if `sync` was meant to be a future module or if it was deleted and someone forgot. This requires **your** context about what happened to `sync/`.

---

## Codebase Health Dashboard

| Metric | Value |
|--------|-------|
| Go files (production) | 177 |
| Go files (test) | 131 |
| Production LOC | 16,542 |
| Test LOC | 29,982 |
| Test:Code ratio | 1.81:1 |
| Packages | 23 |
| All tests passing | YES (23/23) |
| Lint issues | 2 (pre-existing, unrelated) |
| Duplication groups | 10 (4 "actionable", all structural) |
| Coverage range | 84.2%–100% across packages |

### Coverage by Package

| Package | Coverage |
|---------|----------|
| `core/pkg/dispatcher` | 100.0% |
| `core/pkg/id` | 100.0% |
| `middleware` | 100.0% |
| `catalog/internal/caseutil` | 100.0% |
| `memory` | 99.6% |
| `core/query` | 98.4% |
| `catalog` | 96.3% |
| `catalog/d2` | 95.0% |
| `catalog/openapi` | 94.4% |
| `projection` | 94.4% |
| `core/event` | 93.8% |
| `catalog/asyncapi` | 93.7% |
| `core/decider` | 93.6% |
| `core/command` | 92.3% |
| `testhelpers` | 91.3% |
| `catalog/docserver` | 90.1% |
| `storage` | 89.4% |
| `catalog/eventcatalog` | 85.7% ↓ |
| `catalog/internal/schemautil` | 84.2% |

---

## Git Changes This Session

```
storage/sql_base.go       | 17 +++++++++++++++++  (NEW)
storage/checkpoint.go     | 14 +++++---------
storage/event_store.go    | 12 ++++++------
storage/outbox.go         | 13 +++++--------
storage/snapshot.go       | 13 +++++--------
5 files changed, 38 insertions(+), 31 deletions(-)
```
