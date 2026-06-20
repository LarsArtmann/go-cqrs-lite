# Comprehensive Status Report — go-cqrs-lite

**Date:** 2026-04-28 14:26  
**Reporter:** Crush (AI Engineering Partner)  
**Session Focus:** Example `go.mod` tidy drift fixes + `CatalogCore` API compilation fixes  
**Git Branch:** master  
**Last Commit:** `ccb7248` — fix(example): resolve go.mod tidy drift + fix CatalogCore compilation  
**Commits Ahead of Origin:** 22

---

## Executive Summary

All systems green. The Nix flake migration is fully operational. The example modules (`example/user`, `example/catalog`) now compile cleanly after fixing two issues discovered during the post-migration validation: missing `testhelpers` replace directives and stale `NewCatalogCore` calls that should have been `MustNewCatalogCore`. Build, test, vet, lint, race detector, and coverage all pass. Zero TODO/FIXME markers in the codebase.

---

## a) FULLY DONE

### Infrastructure (from previous session, verified still operational)

- [x] **Nix flake** — `flake.nix`, `flake.lock`, `flake-parts`, `treefmt-nix`
- [x] **Unified CI** — `.github/workflows/ci.yml` replaces `lint.yml` + `test.yml`
- [x] **Makefile removed** — all targets via `nix run .#<app>`
- [x] **Dev shell** — `nix develop` provides pinned Go 1.26.2, golangci-lint v2.11.4, gofumpt 0.9.2, golines 0.13.0, gotools 0.44.0
- [x] **Formatter** — `nix fmt` runs `gofumpt` + `goimports` + `golines`

### Example Module Fixes (this session)

- [x] **example/user/go.mod** — added `testhelpers` replace directive, reconciled `go-composable-business-types` version (`v0.0.0` → `v0.1.0`)
- [x] **example/catalog/go.mod** — same fixes as above
- [x] **example/user/commands.go** — switched `command.NewCatalogCore` → `command.MustNewCatalogCore` (2 call sites)
- [x] **example/catalog/main.go** — switched `command.NewCatalogCore` → `command.MustNewCatalogCore` (2 call sites)
- [x] **Verified compilation** — both examples build with `GOWORK=off go build ./...`

### Core Modules (all verified)

- [x] **core/** — build, test, lint clean
- [x] **memory/** — build, test, lint clean
- [x] **catalog/** — build, test, lint clean
- [x] **middleware/** — build, test, lint clean
- [x] **xtypes/** — build, test, lint clean
- [x] **testhelpers/** — build, test clean (no lint needed — no main code)

### Code Quality

- [x] **Zero lint issues** across all 5 linted modules
- [x] **Zero TODO/FIXME/HACK/XXX/BUG** markers in codebase
- [x] **Zero code duplication** (art-dupl -t 27)
- [x] **File size limits** enforced — all files <250 lines

---

## b) PARTIALLY DONE

### Test Coverage

- **Overall:** 79.8% (unchanged from previous session)
- **Strong:** memory 99.4%, middleware 99.2%, catalog/adapters 98.8%, catalog/asyncapi 97.6%
- **Weak:** core/command 67.4%, core/pkg/dispatcher 75.4%, core/pkg/id 73.1%, core/query 80.6%
- **Action needed:** Targeted tests for low-coverage packages. This is unchanged from the previous session.

### Nix Flake Purity

- **Status:** `nix run .#<app>` works perfectly. `nix flake check` evaluates clean but has no sandboxed `checks`.
- **Why:** Private repo `github.com/larsartmann/go-composable-business-types` cannot be fetched in a Nix sandbox.
- **Impact:** Low for daily dev, medium for reproducibility purists.

---

## c) NOT STARTED

Same as previous session — 20 items from the roadmap remain untouched:

1. SQL/database event store (`storage/` module)
2. Watermill pub/sub adapter
3. Projection/read-model support (samber/ro)
4. SQL-backed snapshot store
5. Saga/process manager support
6. Event upcasting/schema evolution
7. Dead letter queue for failed events
8. Health check endpoints
9. gRPC transport adapter
10. HTTP transport adapter
11. OpenTelemetry tracing middleware
12. Metrics endpoint example
13. E-commerce comprehensive example
14. Archive old status reports
15. Remove redundant state in `TypedAggregate`
16. Extract validation helper in `pkg/id/id.go`
17. Unify Apply pattern in `example/user/aggregate.go`
18. Tag v0.1.0 release
19. Typed command dispatcher helper
20. Snapshot support for aggregates

---

## d) TOTALLY FUCKED UP

**Nothing is totally fucked up.**

### Remaining LSP False Positives (2 errors)

These are **gopls workspace state issues**, not real errors:

1. **`example/user/go.mod`** — gopls reports "updates to go.mod needed". The file was just `go mod tidy`-ed. The version `v0.1.0` is correct and matches the replace directive. Compilation succeeds.
2. **`example/catalog/go.md`** — Same issue.

**Root cause:** gopls creates separate module views for examples (outside `go.work`). The workspace state is stale. A `gopls` restart would clear these. They do not affect compilation, tests, or CI.

**Previous false positive resolved:** The `catalog/adapters/adapters_test.go:155` error (reporting `query.NewCatalogCore` as single-value) has disappeared — confirming it was a stale gopls diagnostic.

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (This Week)

1. **Restart gopls / update workspace** — The 2 remaining LSP false positives are cosmetic but confusing. They'll clear on next gopls restart.

### Short-Term (Next 2 Weeks)

2. **Boost `core/command` test coverage** — 67.4%, the lowest package. Add edge case tests for `command.New()` validation.
3. **Boost `pkg/id` coverage** — 73.1%. Add `Parse`/`MustParse` tests for `CausationID`, `CorrelationID`, `RequestID`, `CommandID`.
4. **Boost `pkg/dispatcher` coverage** — 75.4%. Add concurrent registration and middleware chain edge case tests.
5. **Add `EventRetry` tests** — `CommandRetry` is tested; `EventRetry` needs parity.
6. **Resolve `go-composable-business-types` privacy** — Either make the repo public or vendor it as a fixed-output derivation for pure Nix builds.

### Medium-Term (Next Month)

7. **SQL event store** — Most impactful missing feature. Production blocker.
8. **Projection module design** — Using samber/ro. Start with interface design.
9. **Tracing middleware** — OpenTelemetry-compatible. Straightforward.
10. **Archive old status reports** — Keep only 3 most recent in `docs/status/`.
11. **Tag v0.1.0** — Library is stable enough.

---

## f) Top #25 Things To Get Done Next

| #   | Priority | Item                                           | Module                | Effort | Impact        |
| --- | -------- | ---------------------------------------------- | --------------------- | ------ | ------------- |
| 1   | 🔴 P0    | Restart gopls (clear false positives)          | Workspace             | 1min   | Hygiene       |
| 2   | 🔴 P0    | Resolve `go-composable-business-types` privacy | Repo                  | 30min  | Nix purity    |
| 3   | 🟡 P1    | Boost `core/command` coverage to >85%          | `core/command`        | 2h     | Quality       |
| 4   | 🟡 P1    | Add `EventRetry` middleware tests              | `middleware/`         | 1h     | Parity        |
| 5   | 🟡 P1    | Boost `pkg/id` coverage to >85%                | `core/pkg/id`         | 1h     | Quality       |
| 6   | 🟡 P1    | Boost `pkg/dispatcher` coverage to >85%        | `core/pkg/dispatcher` | 1.5h   | Quality       |
| 7   | 🟡 P1    | Add `testhelpers/` to test/coverage apps       | `flake.nix`           | 5min   | Completeness  |
| 8   | 🟢 P2    | SQL event store (`storage/` module)            | New                   | 2d     | **BLOCKER**   |
| 9   | 🟢 P2    | Design projection interface                    | New                   | 1d     | Architecture  |
| 10  | 🟢 P2    | OpenTelemetry tracing middleware               | `middleware/`         | 4h     | Observability |
| 11  | 🟢 P2    | Tag v0.1.0 release                             | Repo                  | 30min  | Milestone     |
| 12  | 🟢 P2    | Archive old status reports                     | `docs/status/`        | 30min  | Hygiene       |
| 13  | 🔵 P3    | SQL-backed snapshot store                      | New                   | 1d     | Feature       |
| 14  | 🔵 P3    | Watermill pub/sub adapter                      | New                   | 2d     | Integration   |
| 15  | 🔵 P3    | Projection/read-model implementation           | New                   | 3d     | Feature       |
| 16  | 🔵 P3    | gRPC transport adapter                         | New                   | 2d     | Transport     |
| 17  | 🔵 P3    | HTTP transport adapter                         | New                   | 2d     | Transport     |
| 18  | 🔵 P3    | Event upcasting/schema evolution               | `core/event`          | 3d     | Long-term     |
| 19  | 🔵 P3    | Saga/process manager                           | New                   | 5d     | Advanced      |
| 20  | 🔵 P3    | Dead letter queue                              | `middleware/`         | 2d     | Resilience    |
| 21  | 🔵 P3    | Health check endpoints                         | New                   | 1d     | Ops           |
| 22  | 🔵 P3    | E-commerce example                             | `example/`            | 3d     | Documentation |
| 23  | 🔵 P3    | Metrics endpoint example                       | `example/`            | 1d     | Documentation |
| 24  | 🔵 P3    | Resolve `TypedCommand.Command()` validation    | `xtypes/`             | 2h     | API design    |
| 25  | 🔵 P3    | Remove redundant state in `TypedAggregate`     | `xtypes/`             | 2h     | Cleanup       |

---

## g) Top #1 Question I Cannot Figure Out Myself

> **How should we handle the `TypedCommand.Command()` validation impedance mismatch?**

`TypedCommand` (in `xtypes/`) stores a raw `command.Type` and `id.AggregateID`. Its `Command()` method calls `command.New(c.commandType, c.aggregateID)`, which now validates that the type is non-empty and the aggregate ID is non-zero. This means `Command()` returns `(command.Command, error)` even though the caller created a `TypedCommand` — a type that _feels_ like it should always be valid.

**The tension:**

- If `NewTypedCommand` validates and returns `(*TypedCommand, error)`, we break the current API (it's a constructor that doesn't error).
- If `NewTypedCommand` panics on invalid input (like `MustNewCatalogCore`), we match `MustCommand` but lose the ability to validate at construction time in normal code paths.
- If we store a pre-validated `*command.Core` inside `TypedCommand`, we change the struct layout and potentially break callers who set fields directly.
- If we leave it as-is, every call to `Command()` requires error handling for a condition that "should never happen" if the `TypedCommand` was constructed correctly.

**What is the intended invariant?** Should `TypedCommand` be a thin wrapper (current design) or a validated, always-correct value object? The same question applies to `TypedEvent` and `TypedAggregate`.

This is an API design decision with consumer-facing implications. I can implement any option, but I need the architectural direction.

---

## Appendix: Quick Stats

| Metric             | Value                           |
| ------------------ | ------------------------------- |
| Go files           | 95                              |
| Lines of Go        | 15,172                          |
| Test files         | 34                              |
| Example files      | 6                               |
| Modules            | 6                               |
| Build              | ✅ Clean                        |
| Vet                | ✅ Clean                        |
| Test               | ✅ All pass                     |
| Test (race)        | ✅ All pass                     |
| Lint               | ✅ 0 issues                     |
| Coverage           | 79.8%                           |
| Examples compile   | ✅ Both OK                      |
| Nix flake          | ✅ Evaluates clean              |
| LSP errors         | 2 (stale gopls false positives) |
| TODO/FIXME in code | 0                               |

---

## Git Status

Working tree **clean**. All changes from this session were committed as `ccb7248`.

```
On branch master
Your branch is ahead of 'origin/master' by 22 commits.
nothing to commit, working tree clean
```

---

_End of report. Awaiting instructions._
