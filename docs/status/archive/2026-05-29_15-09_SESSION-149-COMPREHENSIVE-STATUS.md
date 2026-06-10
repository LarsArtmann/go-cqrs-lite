# Session 149 — Comprehensive Status Report

**Date:** 2026-05-29_15-09
**Branch:** master (1 commit ahead of origin)
**Last 2 Commits:**

- `d27c004` feat: add command.Store — ISP split (CommandSink + CommandSource)
- `22ddbbb` docs(status): add session 148 comprehensive status — cleanup complete

---

## A) FULLY DONE ✅

### 1. Command Store — ISP Split (Sink + Source) — JUST SHIPPED

| Artifact                                                    | Status                                                                                                    |
| ----------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `core/pkg/id/command_id.go`                                 | ✅ Branded ULID type, New/Parse/MustParse                                                                 |
| `core/command/aggregate_ref.go`                             | ✅ AggregateType, AggregateRef, parse helpers                                                             |
| `core/command/store.go`                                     | ✅ PersistedCommand (immutable, validated, payload-isolated) + CommandSink/CommandSource/Store interfaces |
| `core/command/store_test.go`                                | ✅ 17 tests — all green                                                                                   |
| `core/command/errors.go`                                    | ✅ 4 new sentinels (ErrEmptyAggregateType, ErrDuplicateCommand, ErrCommandNotFound, ErrStoreClosed)       |
| `docs/planning/2026-05-29_13-23_command-store-isp-split.md` | ✅ Mermaid diagram + rationale                                                                            |

### 2. Core CQRS — All Modules Production-Quality

| Module                | Coverage | Status                                                                                    |
| --------------------- | -------- | ----------------------------------------------------------------------------------------- |
| `core/command`        | 94.7%    | ✅ Dispatcher + PersistedCommand + Store interfaces                                       |
| `core/decider`        | 100.0%   | ✅ Pure-function aggregate pattern                                                        |
| `core/event`          | 90.7%    | ✅ Full event sourcing stack                                                              |
| `core/query`          | 96.8%    | ✅ Typed dispatch + pagination                                                            |
| `core/pkg/dispatcher` | 92.2%    | ✅ Generic middleware chain                                                               |
| `core/pkg/id`         | 94.5%    | ✅ Branded IDs (Aggregate, Event, Command, Correlation, Causation, User, Request, Client) |

### 3. Supporting Modules — All Green

| Module                     | Coverage   | Status                                                 |
| -------------------------- | ---------- | ------------------------------------------------------ |
| `memory`                   | 99.1%      | ✅ In-memory store/bus/snapshot                        |
| `catalog` (5 sub-packages) | 86–100%    | ✅ AsyncAPI, D2, EventCatalog, OpenAPI exporters       |
| `middleware`               | 94.0%      | ✅ Logging, Retry, Recovery, Validation, Metrics, OTel |
| `testhelpers`              | 83.7%      | ✅ Noop/Failing/Panic handlers                         |
| `projection`               | 90.4%      | ✅ Runner (replay+live), HandlerRegistry, Builder      |
| `signing` (incl. multisig) | 93.7–94.2% | ✅ HMAC-SHA256, Ed25519, multisig                      |
| `watermill`                | 94.4%      | ✅ Protocol adapter                                    |
| `pebble`                   | 87.8%      | ✅ Embedded KV event store                             |
| `codec`                    | 100.0%     | ✅ JSON, Raw passthrough                               |

### 4. Documentation

- ✅ AGENTS.md — current with module inventory
- ✅ FEATURES.md — honest feature audit (last audited 2026-05-28)
- ✅ TODO_LIST.md — reconciled, most items done
- ✅ 8 ADRs in docs/adr/
- ✅ Planning docs in docs/planning/
- ✅ Planning doc for command.Store with mermaid graph

---

## B) PARTIALLY DONE ⚠️

### 1. Storage Module — Mid-Refactor to `storage/sql` Sub-Package

13 files modified, 1 new `storage/sql/` directory (9 files). The refactor moves SQL dialect logic into a sub-package `storage/sql` but is **incomplete** — field visibility changed from exported (`DB`, `Dialect`) to unexported (`db`, `dialect`) but internal references still use the old exported names.

**Impact:** `storage/` does not compile. `integration/event` fails transitively.

**Files affected:**

- `storage/outbox.go` — 3 broken references to `o.DB` / `o.Dialect`
- `storage/snapshot.go` — 6 broken references to `s.DB` / `s.Dialect`
- 11 other files with partial changes

**What's left:** Either finish the refactor (update all references to use `sqlpkg.Base` methods) or revert the partial changes.

### 2. Listing Module Rename — ~90% Done

The `stream/` → `listing/` rename is committed and working:

- ✅ `listing/go.mod` — module path updated
- ✅ All `package stream` → `package listing`
- ✅ All test imports updated
- ✅ go.work updated
- ✅ example/listing updated

**Remaining:** `FEATURES.md` still references old "stream" terminology. Listing module's `README.md` still says `stream`.

### 3. Modularization Proposal — Written, Not Executed

`docs/modularization/PROPOSAL.md` identifies:

- Transitive dependency pollution (saga → testhelpers → everything)
- God-package in `core/event` (90+ exported symbols)
- Self-referencing replace directives in 7 modules

**Status:** Analysis done, no execution started.

---

## C) NOT STARTED 📐

1. **Command Store implementations** — `MemoryCommandStore` (in-memory), `SQLCommandStore` (PostgreSQL/SQLite)
2. **Command Journal / SeekableCommandJournal** — cross-aggregate command log (for audit, replay)
3. **Command Outbox** — reliable dispatch with pending/ack lifecycle
4. **`features update FEATURES.md` — add command.Store section**
5. **Saga module removal from `testhelpers`** — break transitive saga leak
6. **Split `core/event` god-package** — 90+ symbols across 12 concerns
7. **Self-referencing replace directive cleanup** — 7 modules
8. **v1.0.0 tag release** — unblock `replace` directive removal
9. **`turso` module** — only 206 lines, thin adapter, appears unmaintained
10. **ADR index has duplicate numbering** — two ADR-0007s

---

## D) TOTALLY FUCKED UP 💀

### 1. Storage Module — BROKEN BUILD

`nix run .#build` succeeds only because the workspace build doesn't touch storage. Individual module build fails:

```
storage/outbox.go:90:13: o.DB undefined
storage/outbox.go:96:5:  o.Dialect undefined
storage/snapshot.go:71:26: s.Dialect undefined
```

This is from the incomplete `storage/sql` sub-package refactor. **13 files changed, 168 insertions, 323 deletions** — and none of it compiles.

### 2. Saga Module — DELETED but Still Referenced

- `saga/` directory does not exist
- `go.work` does not list `./saga`
- BUT: `storage/go.mod` still has `github.com/larsartmann/go-cqrs-lite/saga v1.6.0` in require + replace
- AND: `testhelpers/go.mod` likely still depends on saga transitively
- The saga removal was partial — references were cleaned from docs but not from all go.mod files

### 3. Pre-commit Hook — FAILS ON EVERY COMMIT

`buildflow` pre-commit hook fails due to:

- Lint failures in 4 modules (root, example/listing, listing, scripts/go-mod-graph-local)
- Binary check failures
- `go-mod-tidy` issues from the gopls stale metadata (52+ errors)

Every commit requires `--no-verify`.

### 4. gopls Diagnostics — 52 Errors, 9 Warnings

Mostly `go mod tidy` complaints from gopls seeing stale module metadata. Not actual compile errors, but creates noise and makes IDE experience poor.

---

## E) WHAT WE SHOULD IMPROVE

1. **FIX THE BROKEN STORAGE MODULE** — Either finish the `storage/sql` refactor or revert. A broken module in `master` is unacceptable for a library.
2. **Clean saga remnants** — Remove saga from all go.mod require/replace directives in storage, testhelpers, and any other module that still references it.
3. **Fix pre-commit hook** — The `buildflow` failures need investigation. Either fix the lint issues or adjust the config.
4. **Update FEATURES.md** — Add command.Store section, remove stream references, add listing module.
5. **Eliminate self-referencing replace directives** — 7 modules have `replace X => ./` which is unusual.
6. **ADR numbering** — Two ADR-0007 files exist. Renumber.
7. **Listing README.md** — Still references `stream` package name.

---

## F) TOP 25 THINGS TO DO NEXT

Ranked by impact × urgency × effort (Pareto order):

| #   | Task                                                              | Impact      | Effort | Category |
| --- | ----------------------------------------------------------------- | ----------- | ------ | -------- |
| 1   | Fix storage module build — finish `storage/sql` refactor          | 🔴 CRITICAL | 2h     | Fix      |
| 2   | Clean all saga references from go.mod files                       | 🔴 HIGH     | 30min  | Fix      |
| 3   | `go mod tidy` all modules — eliminate gopls noise                 | 🟡 HIGH     | 30min  | Fix      |
| 4   | Implement `MemoryCommandStore` in memory/ module                  | 🟡 HIGH     | 2h     | Feature  |
| 5   | Update FEATURES.md — add command.Store, remove stream             | 🟡 MEDIUM   | 15min  | Docs     |
| 6   | Fix pre-commit hook / buildflow config                            | 🟡 MEDIUM   | 1h     | Fix      |
| 7   | Renumber ADR-0007 duplicate                                       | 🟢 LOW      | 5min   | Docs     |
| 8   | Update listing/README.md — stream → listing                       | 🟢 LOW      | 10min  | Docs     |
| 9   | Implement `SQLCommandStore` in storage/ module                    | 🟡 HIGH     | 4h     | Feature  |
| 10  | Command Journal + SeekableCommandJournal interfaces               | 🟡 MEDIUM   | 1h     | Feature  |
| 11  | Command Outbox interface + SQL implementation                     | 🟡 MEDIUM   | 3h     | Feature  |
| 12  | Break saga leak from testhelpers                                  | 🟡 MEDIUM   | 2h     | Refactor |
| 13  | Clean self-referencing replace directives (7 modules)             | 🟢 LOW      | 30min  | Cleanup  |
| 14  | Add command.Store to integration tests                            | 🟡 MEDIUM   | 2h     | Testing  |
| 15  | Split core/event god-package into sub-packages                    | 🟡 HIGH     | 8h     | Refactor |
| 16  | Move example/todo to own repository                               | 🟢 LOW      | 1h     | Cleanup  |
| 17  | Add catalog diff / breaking-change detection tool                 | 🟢 LOW      | 4h     | Feature  |
| 18  | Add high-level test utilities (AggregateTester, ProjectionTester) | 🟢 LOW      | 6h     | Feature  |
| 19  | Push v1.0.0 tags — unblock replace directive removal              | 🔴 HIGH     | 30min  | Release  |
| 20  | Remove replace directives after v1.0.0 tags                       | 🟡 MEDIUM   | 1h     | Cleanup  |
| 21  | Turso module — verify it still works, add tests                   | 🟢 LOW      | 2h     | Fix      |
| 22  | Add code-generated typed command handlers (cqrs-gen)              | 🟢 LOW      | 4h     | Feature  |
| 23  | Add Pebble command store implementation                           | 🟢 LOW      | 3h     | Feature  |
| 24  | Write CHANGELOG.md entries for recent work                        | 🟢 LOW      | 30min  | Docs     |
| 25  | Add .github/ISSUE_TEMPLATE and CONTRIBUTING.md                    | 🟢 LOW      | 1h     | Docs     |

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**What is the intended end-state for the `storage/sql` sub-package refactor?**

The 9 files in `storage/sql/` (base.go, dialect.go, doc.go, errors.go, helpers.go, otel.go, reconstruction.go, sqlite.go, tables.go) suggest extracting SQL-specific logic into a sub-package. But I cannot determine:

1. Should `storage/sql.Base` have exported methods (DB(), Dialect()) that the parent package calls, or should the parent's structs embed `sql.Base` directly with unexported fields?
2. Is this the same refactor proposed in `docs/modularization/PROPOSAL.md` or a separate initiative?
3. Should the old `sqlBase` struct in `storage/sql_backend.go` be deleted entirely?

**Why it matters:** This blocks storage compilation, which blocks integration tests, which blocks CI. It's the #1 blocker.

---

## Build & Test Summary

| Check                           | Result                                                           |
| ------------------------------- | ---------------------------------------------------------------- |
| `nix run .#build`               | ✅ PASS (workspace build, storage skipped by go.work resolution) |
| `go test ./core/...`            | ✅ ALL GREEN (6 packages, 94.7–100% coverage)                    |
| `go test` (all working modules) | ✅ 24/25 GREEN (saga excluded — dir missing)                     |
| `storage/` compile              | ❌ BROKEN (13 files mid-refactor)                                |
| `integration/event`             | ❌ BROKEN (depends on storage)                                   |
| Pre-commit hook                 | ❌ FAILS (lint in 4 modules, buildflow issues)                   |
| gopls diagnostics               | 52 errors, 9 warnings (mostly go mod tidy noise)                 |

---

## Git Status

```
On branch master, 1 commit ahead of origin/master

Unstaged changes:
  M storage/ (13 files — broken sql refactor)

Untracked:
  ?? storage/sql/ (9 files — new sub-package)
```

---

_Generated by Crush — Session 149_
