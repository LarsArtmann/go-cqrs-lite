# Comprehensive Status Report — Multi-Module Monorepo Migration

**Date:** 2026-04-24 23:28 CEST
**Author:** Crush (GLM-5.1)
**Branch:** master (up to date with origin)
**Commits since last status:** 8 (563f126..889e7c3)

---

## Executive Summary

The go-cqrs-lite monorepo has been **fully migrated to a 5-module Go workspace**. All extraction phases (0–4) are complete, all 13 test packages pass with race detection, CI workflows are updated, examples are fixed, and vestigial code has been removed. The codebase is clean, green, and ready for Phase 5 (new module creation).

**Total modules:** 5 (`core`, `memory`, `catalog`, `middleware`, `xtypes`)
**Total lines of Go code:** ~11,339
**All tests passing:** 13/13 packages, race-clean
**Examples:** 2/2 building

---

## A) FULLY DONE

### Module Extraction (Phases 0–4)

| Phase | What                                                                              | Status  |
| ----- | --------------------------------------------------------------------------------- | ------- |
| 0     | Fix query handler ctx, delete pkg/errors, replace custom YAML with go-faster/yaml | ✅ Done |
| 1     | go.work + move into `core/` subdirectory                                          | ✅ Done |
| 2     | Extract `memory/` module                                                          | ✅ Done |
| 3     | Extract `catalog/` module                                                         | ✅ Done |
| 4a    | Extract `middleware/` module (with inlined testhelpers)                           | ✅ Done |
| 4b    | Extract `xtypes/` module                                                          | ✅ Done |
| 4c    | Full verification (all modules build + test + race)                               | ✅ Done |
| 4d    | Clean up core (remove stale deps, empty dirs, vestigial store_config.go)          | ✅ Done |

### Infrastructure Updates

| Item           | What                                                                   | Status  |
| -------------- | ---------------------------------------------------------------------- | ------- |
| Makefile       | Updated for 5-module workspace with per-module test targets            | ✅ Done |
| CI test.yml    | Matrix strategy per module (core, memory, catalog, middleware, xtypes) | ✅ Done |
| CI lint.yml    | Matrix strategy per module                                             | ✅ Done |
| Examples       | Fixed import paths + go.mod for both user and catalog examples         | ✅ Done |
| .gitignore     | Added example binary patterns                                          | ✅ Done |
| AGENTS.md      | Updated to reflect 5-module structure                                  | ✅ Done |
| Migration plan | Phase 4 marked DONE                                                    | ✅ Done |

### Current Module Dependency Graph

```
core/          (errors, uuid, json; test-dep on memory)
  ↑
memory/        (core, errors)
  ↑
catalog/       (core, yaml, json; replace for memory transitive)
  ↑ (none)
middleware/     (core, errors)
  ↑ (none)
xtypes/        (core)
```

### Production Dependencies per Module

| Module       | Direct Deps                                              |
| ------------ | -------------------------------------------------------- |
| `core`       | cockroachdb/errors, google/uuid, go-json-experiment/json |
| `memory`     | core, cockroachdb/errors                                 |
| `catalog`    | core, go-faster/yaml, go-json-experiment/json            |
| `middleware` | core, cockroachdb/errors                                 |
| `xtypes`     | core                                                     |

---

## B) PARTIALLY DONE

### Test Coverage

| Package                | Coverage | Note                        |
| ---------------------- | -------- | --------------------------- |
| `catalog/asyncapi`     | 96.3%    | ✅ Excellent                |
| `xtypes`               | 95.7%    | ✅ Excellent                |
| `event`                | 95.4%    | ✅ Excellent                |
| `query`                | 91.5%    | ✅ Good                     |
| `catalog`              | 91.2%    | ✅ Good                     |
| `catalog/eventcatalog` | 89.7%    | ✅ Good                     |
| `pkg/id`               | 85.4%    | ✅ Good                     |
| `middleware`           | 84.6%    | ⚠️ Adequate                 |
| `command`              | 84.4%    | ⚠️ Adequate                 |
| `aggregate`            | 77.3%    | ⚠️ Could improve            |
| `pkg/dispatcher`       | 77.4%    | ⚠️ Could improve            |
| `catalog/adapters`     | 66.0%    | ❌ Lowest — needs attention |

### Linter State

`golangci-lint` reports ~161 issues in core alone. Most are style preferences:

- `varnamelen`: 50 (variable naming)
- `exhaustruct`: 24 (struct exhaustiveness)
- `revive`: 29 (various)
- `ireturn`: 10 (interface returns)
- `thelper`: 12 (test helper declarations)
- `forcetypeassert`: 7 (type assertion safety)

These are **not bugs** but indicate code quality could be improved.

---

## C) NOT STARTED

### Phase 5: Storage Module (sqlc Event Store)

- Create `storage/` module with `sqlc.yaml`
- Multi-engine: PostgreSQL, MySQL, SQLite
- Implement `core/event.Store` via generated SQL code
- Transactional outbox pattern
- Schema migrations

### Phase 6: Watermill Module (Pub/Sub)

- Create `watermill/` module
- Implement `core/event.Bus` via Watermill
- Support: Redis Streams, NATS, Kafka, Google PubSub, AMQP
- See `docs/planning/2026-04-23_WATERMILL_PRO_CONTRA.md` for analysis

### Phase 7: Projection Module (Read Models)

- Create `projection/` module with samber/ro as internal stream engine
- `projection/internal/stream/` encapsulates Observable types
- Users call `projector.On()`, never see reactive types
- Checkpoint tracking (SQL-backed)
- See `docs/planning/2026-04-24_SAMBER_RO_PRO_CONTRA.md` for analysis

### Phase 8: Snapshot Module

- SQL-backed SnapshotStore
- Snapshot strategies (every N events, time-based)

### Phase 9: Test Utilities Module

- `AggregateTester`, `ProjectionTester`, `BusTester`
- BDD-style testing utilities

### Phase 10: Tag Releases

- Tag each module independently (core/v1.0.0, memory/v1.0.0, etc.)

### Event Codec

- Payload is `[]byte` — need pluggable `Codec` interface (JSON, protobuf, msgpack)
- Interface in `core/event/` or own module?

### Event Upcasting

- Events evolve over time (`UserCreatedV1` → `V2`)
- Need upcasting mechanism
- Interface in `core/upcasting/`, implementation in storage module?

### go-import Meta Tags

- Required for Go 1.25+ subdirectory module resolution
- Need GitHub Pages or similar to serve `<meta name="go-import">` tags

---

## D) TOTALLY FUCKED UP (Issues Found)

### 1. .gitignore Line Corruption

**Fixed in this session.** Line 54 of `.gitignore` had two entries merged into one:

```
/report/jscpd-report.jsonexample/*/catalog
```

Should have been two separate lines. Fixed and committed.

### 2. Example Binaries Committed

**Fixed in this session.** `go build` in example directories created `example/user/user` and `example/catalog/catalog` binaries that were accidentally committed. Removed and added to `.gitignore`.

### 3. Middleware Test Syntax Errors (From Prior Session)

**Fixed in this session.** Commit `563f126` left two bugs:

- Detached `if` blocks (premature closing `}` in two test functions)
- Duplicate import of `errors` and `github.com/cockroachdb/errors`
- Wrong `MaxAttempts=1` in retry test expecting 2 retries

All three fixed in commit `569adf7`.

### 4. .gitignore Doesn't Prevent Binary Recreation

The gitignore patterns (`example/*/catalog`, `example/*/user`) work for git tracking, but running `go build` in example dirs recreates them on disk. Not a git problem, but confusing locally. Consider adding `*.test` style patterns or a Makefile clean target.

---

## E) WHAT WE SHOULD IMPROVE

### Code Quality

1. **`catalog/adapters` coverage at 66%** — the lowest in the entire codebase. Should target 80%+.
2. **Linter issues (161 in core)** — primarily `varnamelen`, `exhaustruct`, `thelper`. A cleanup pass would improve consistency.
3. **`core/go.mod` test dependency on `memory`** — This causes transitive dep issues for downstream modules. Consider extracting shared test helpers into a `testutil/` module (Phase 9).
4. **`middleware/retry.go` uses `cockroachdb/errors` only for `errors.Wrapf`** — Could use `fmt.Errorf("%w")` to eliminate the dependency entirely.

### Architecture

5. **No `Streamer` interface implementation** — `event/store.go` defines `Streamer` but nobody implements it. Either implement or remove.
6. **No Event Codec** — Payload is raw `[]byte`. Need a pluggable Codec interface for JSON/protobuf/msgpack.
7. **No Event Upcasting** — Events can't evolve over time yet.
8. **No `projection/` interface in core** — The "Q" in CQRS is missing entirely. No read model support.

### Developer Experience

9. **`go.work` is gitignored** — Each developer must create it locally. CI doesn't have it (by design). Could provide a `go.work.example` or generate it in Makefile.
10. **No CONTRIBUTING.md update** — Should reflect multi-module structure and how to work with the workspace.

### Documentation

11. **Example modules not in go.work** — They build standalone with `GOWORK=off`. Should they be in go.work for development convenience?
12. **No README update** — Root README still references old single-module structure.

---

## F) Top 25 Things to Do Next

Ranked by impact × feasibility:

| #   | Task                                                                | Impact | Effort | Module     |
| --- | ------------------------------------------------------------------- | ------ | ------ | ---------- |
| 1   | Create `storage/` module with PostgreSQL event store (sqlc)         | 🔥🔥🔥 | Large  | New        |
| 2   | Define `projection/` interface in `core/projection/`                | 🔥🔥🔥 | Small  | core       |
| 3   | Update root README.md for 5-module structure                        | 🔥     | Small  | docs       |
| 4   | Add `go.work.example` file for developer onboarding                 | 🔥     | Tiny   | root       |
| 5   | Improve `catalog/adapters` test coverage (66% → 80%+)               | 🔥🔥   | Medium | catalog    |
| 6   | Create `watermill/` module (implements event.Bus)                   | 🔥🔥🔥 | Large  | New        |
| 7   | Create `projection/` module with samber/ro internally               | 🔥🔥🔥 | Large  | New        |
| 8   | Add Event Codec interface in `core/event/`                          | 🔥🔥   | Small  | core       |
| 9   | Remove cockroachdb/errors from middleware (use fmt.Errorf)          | 🔥     | Tiny   | middleware |
| 10  | Clean up linter warnings (thelper, varnamelen in core)              | 🔥     | Medium | core       |
| 11  | Implement or remove `Streamer` interface                            | 🔥     | Small  | core       |
| 12  | Add Event Upcasting interface in `core/event/` or `core/upcasting/` | 🔥🔥   | Medium | core       |
| 13  | Create `snapshot/` module (SQL-backed)                              | 🔥🔥   | Medium | New        |
| 14  | Create `testutil/` module (AggregateTester, etc.)                   | 🔥🔥   | Medium | New        |
| 15  | Remove core test-dep on memory (move to testutil/)                  | 🔥     | Medium | core       |
| 16  | Add `go-import` meta tags (GitHub Pages)                            | 🔥     | Small  | infra      |
| 17  | Add example to go.work for development convenience                  | 🔥     | Tiny   | root       |
| 18  | Update CONTRIBUTING.md for multi-module workflow                    | 🔥     | Small  | docs       |
| 19  | Add SQLite support to storage/ module                               | 🔥🔥   | Medium | storage    |
| 20  | Add MySQL support to storage/ module                                | 🔥     | Medium | storage    |
| 21  | Tag releases (core/v1.0.0, memory/v1.0.0, etc.)                     | 🔥     | Small  | infra      |
| 22  | Add outbox pattern to storage/ module                               | 🔥🔥   | Medium | storage    |
| 23  | Create benchmark suite for event store performance                  | 🔥     | Medium | storage    |
| 24  | Add schema migration tool (golang-migrate or goose)                 | 🔥     | Small  | storage    |
| 25  | Add CI step to verify examples build                                | 🔥     | Tiny   | CI         |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should the `storage/` module use `pgx/v5` (native PostgreSQL driver) or `database/sql` (standard library) as its primary interface?**

Arguments:

- `pgx/v5`: Better performance, native PostgreSQL features (LISTEN/NOTIFY, COPY, batch), type mapping. But locks users into PostgreSQL.
- `database/sql`: Standard, works with any driver. But loses PostgreSQL-specific optimizations.

The migration plan says "multi-engine: postgres, mysql, sqlite" with engine-specific build tags. This implies `database/sql` for MySQL/SQLite but `pgx` for PostgreSQL. But the interface presented to users should probably be `database/sql` compatible for portability, with `pgx` as an optimization for PostgreSQL.

**The question is:** Do we use `pgx` directly and lose the MySQL/SQLite story, or abstract behind `database/sql` and lose PostgreSQL-specific features? Or use build tags with engine-specific implementations?

---

## Session Log

Commits in this session (8 total):

```
889e7c3 chore(core): remove vestigial store_config.go and its test
1f36eb8 chore: remove accidentally committed example binaries, add to gitignore
5883c6a fix(examples): update import paths and go.mod for multi-module structure
58805de chore(ci): update workflows for 5-module monorepo structure
f35c131 chore: update Makefile for 5-module workspace structure
f12d71b docs: update AGENTS.md and migration plan for Phase 4 completion
d91990a chore(core): remove stale dependencies after middleware/xtypes extraction
4f4b0c7 refactor: extract xtypes package from core/ into standalone top-level module
```

Prior session commits (2):

```
569adf7 fix(middleware): fix test syntax errors and add go.sum from prior extraction
563f126 refactor: extract middleware package from core/ into standalone top-level module
```

## File Tree (Current State)

```
go-cqrs-lite/
├── go.work (gitignored)
├── go.work.sum
├── core/                  # github.com/larsartmann/go-cqrs-lite/core
│   ├── go.mod             # errors, uuid, json, ginkgo/gomega
│   ├── command/
│   ├── query/
│   ├── event/
│   ├── aggregate/
│   ├── pkg/id/
│   ├── pkg/dispatcher/
│   ├── internal/testhelpers/
│   └── internal/testutil/
├── memory/                # github.com/larsartmann/go-cqrs-lite/memory
│   ├── go.mod             # core, errors
│   ├── store.go
│   ├── bus.go
│   └── snapshot.go
├── catalog/               # github.com/larsartmann/go-cqrs-lite/catalog
│   ├── go.mod             # core, yaml, json
│   ├── types.go, registry.go, schema.go
│   ├── adapters/
│   ├── asyncapi/
│   ├── eventcatalog/
│   └── internal/cattest/
├── middleware/             # github.com/larsartmann/go-cqrs-lite/middleware
│   ├── go.mod             # core, errors
│   ├── logging.go, metrics.go, recovery.go, retry.go, validation.go
│   └── middleware.go
├── xtypes/                # github.com/larsartmann/go-cqrs-lite/xtypes
│   ├── go.mod             # core
│   ├── command.go, event.go, aggregate.go, id.go
│   └── xtypes_test.go
├── example/
│   ├── user/              # standalone module, builds with GOWORK=off
│   └── catalog/           # standalone module, builds with GOWORK=off
├── docs/
│   ├── planning/          # 6 planning docs
│   └── status/            # 10+ status reports
├── Makefile               # Updated for 5-module workspace
├── AGENTS.md              # Updated for 5-module structure
└── .github/workflows/     # Matrix CI for all 5 modules
```
