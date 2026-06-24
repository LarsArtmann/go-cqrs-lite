# Status Report: 2026-06-24 Session — SQL View Stores, gRPC Transport, Doc-Check, MemoryBus

**Date:** 2026-06-24
**Session Start:** ~02:00
**Session End:** ~11:41

---

## a) FULLY DONE

### 1. SQL-Backed Views for stack.Materialize (commit `6e70bca5`)

| Feature | Files | Tests |
|---------|-------|-------|
| `kv.ViewStore[V,K]` interface | `kv/view_store.go` | compile-time assertions |
| `kv.ViewQuerier[V]` (server-side WHERE/ORDER BY/LIMIT) | `kv/view_store.go` | 7 query tests |
| `kv.TombstoneQuerier[V]` (server-side tombstone filtering) | `kv/view_store.go` | 4 tombstone tests |
| `storage.SQLViewStore[V,K]` (dedicated SQL table per view) | `storage/view_store.go`, `view_store_crud.go`, `view_store_query.go`, `view_store_options.go` | CRUD, Scan, Query tests |
| `stack.Materialize` decoupled from `*kv.TypedStore` | `stack/materialize.go` | 4 Materialize tests |
| `storage.AutoMapper[V]` (struct tag → ViewMapper) | `storage/view_store_auto.go` | AutoMapper test |
| `storage.ViewMapper.Indexes` (secondary indexes) | `storage/view_store.go` | Index test |
| `storage.SQLViewStore.BatchSet` (chunked upsert) | `storage/view_store_batch.go` | BatchSet test |
| `kv.ViewCounter[V]` (COUNT without loading) | `storage/view_store_count.go` | Count test |
| `kv.ViewResetter[V]` (DeleteAll) | `storage/view_store_batch.go`, `kv/typed_store.go` | DeleteAll test |
| `kv.ViewFilter` + `FilteredQuerier` (injection-safe filters) | `kv/view_store.go`, `storage/view_store_count.go` | QueryFiltered test |
| `sqlite.SQLViewModel[V,K]` + `postgres.SQLViewModel[V,K]` | `stack/sqlite/view_models.go`, `stack/postgres/view_models.go` | Integration test |
| `Bundle.Database()` + `WithDatabase()` | `stack/bundle.go`, `stack/options.go` | Preset tests |
| `turso.NewViewStore[V,K]` | `storage/turso/connector.go` | — |
| ViewStore contract test suite | `kv/viewstoretest/contract.go` | — |
| Concurrent race tests | `storage/view_store_race_test.go` | `-race` clean |
| Benchmarks (KV vs SQL, multi-DB) | `storage/view_store_bench_test.go`, `view_store_multidb_bench_test.go` | — |
| Test file split (335 → 165 + 110) | `storage/view_store_query_test.go`, `view_store_validation_test.go` | — |

### 2. gRPC Transport Adapter (commit `81d29455`, lint fixes pending commit)

| Feature | Files | Tests |
|---------|-------|-------|
| Proto definitions | `transport/grpc/proto/cqrs.proto`, generated `.pb.go` | — |
| Server adapters (Command + Query) | `transport/grpc/command_server.go`, `query_server.go` | 3 round-trip tests |
| Client adapters | `transport/grpc/client.go` | — |
| `command.WithCustomMetadata` | `command/metadata.go` | used in gRPC tests |

### 3. In-Memory Command Bus (commit `81d29455`)

| Feature | Files | Tests |
|---------|-------|-------|
| `command.NewMemoryBus()` implementing `command.Bus` | `command/memory_bus.go` | 4 tests (publish/subscribe, error, middleware, nil handler) |

### 4. Doc Cross-Reference Tool (commit `81d29455`, lint fixes pending commit)

| Feature | Files | Tests |
|---------|-------|-------|
| `cmd/doc-check` tool | `cmd/doc-check/main.go` | verified 412 refs across 24 packages |
| CI step in ci.yml | `.github/workflows/ci.yml` | — |
| Fixed 3 stale doc references found by tool | `SKILL.md`, `AGENTS.md` | — |

### 5. SEC Consumer Migration (commit `81d29455`)

- Added `replace` directives pointing to local go-cqrs-lite
- Added `stack/v3` + `stack/sqlite/v3` dependencies
- Added `NewCQRSAppFromBundle()` constructor
- **Fixed data-loss bug**: `server.go` was passing `nil` journal → projections couldn't replay on restart

### 6. Documentation & ADRs

- SKILL.md: SQL-backed views section, CatchUpSubscriber canonical pattern, Bundle.Debug() section
- ADR-0034: Session store boundary (accepted — no SessionStore in Bundle)
- ADR-0035: Branded DSN types (rejected — breaks `os.Getenv` pattern)
- CI: Turso sync test job, doc-check step

---

## b) PARTIALLY DONE

### Lint Fixes (uncommitted — 6 files changed)

**Status:** All lint issues identified and fixed, but the commit failed because the pre-commit hook runs golangci-lint on ALL modules including the new ones. The fixes are ready but need one more commit attempt.

| File | Issue | Fix |
|------|-------|-----|
| `.golangci.yml` | grpc/protobuf not in depguard allow list | Added entries |
| `transport/grpc/*.go` | depguard, err113, exhaustruct, noctx, noinlineerr, nilerr, nolintlint | All fixed |
| `cmd/doc-check/main.go` | forbidigo (fmt.Printf), gosec (G304, G706), gocognit | Switched to log.Printf, filepath.Clean, split functions |

### AGENTS.md Update

The module list, test command, module graph, and Key Patterns sections are **out of date** — they don't mention:
- `transport/grpc`, `cmd/doc-check`, `kv/viewstoretest`
- `command.NewMemoryBus`, `command.WithCustomMetadata`
- New view store features (AutoMapper, BatchSet, Count, ViewFilter, DeleteAll, Indexes)
- `Bundle.Database()`, `WithDatabase`, `sqlite.SQLViewModel`, `postgres.SQLViewModel`

---

## c) NOT STARTED

- AGENTS.md comprehensive update (modules, test cmd, layer graph, patterns)
- SKILL.md update for gRPC transport, MemoryBus
- CI per-module matrix entries for `transport/grpc` and `cmd/doc-check`
- Status report committed to `docs/status/`

---

## d) TOTALLY FUCKED UP

### gRPC genproto Workspace Conflict

The `transport/grpc` module pulls `google.golang.org/genproto/googleapis/rpc` which conflicts with the old monolithic `google.golang.org/genproto` dragged in by `cockroachdb/errors` (transitive from `event/`). **The module cannot be in `go.work`** — it must be tested with `GOWORK=off`. This is documented but fragile.

**Impact:** `go test ./...` from workspace root doesn't cover transport/grpc. CI must use `GOWORK=off` for it.

### Pre-Commit Hook Whack-a-Mole

The BuildFlow pre-commit hook runs golangci-lint on every module. Each lint fix revealed new issues from the strict linter config (err113, exhaustruct, nolintlint, nilerr). This took 5 iterations to resolve — each iteration requiring a full re-commit cycle because the hook runs synchronously.

---

## e) WHAT WE SHOULD IMPROVE

1. **Lint new code BEFORE committing** — run `golangci-lint` on the module before attempting a commit, not after the hook fails
2. **AGENTS.md must stay current** — every new module/feature must be reflected in the module list, test command, and layer graph
3. **Transport module isolation** — transport/grpc can't be in go.work due to genproto conflict. Consider excluding it from workspace lint or fixing the genproto version
4. **Dual ViewQuery API** — `ViewQuery.Where` (raw SQL) and `ViewFilter` (structured) coexist. Pick one as canonical and deprecate the other
5. **`toAnySlice` uses reflection** in view_store_count.go — should be a simple type switch
6. **Unused `errors` import in command_server.go** — the `var _ = errors.New` hack should be removed
7. **SEC's `NewCQRSAppFromBundle` is dead code** — server.go still calls `NewCQRSAppWithStore`. The bundle constructor was added but never wired
8. **Turso go.mod was manually edited** — the replace directive for `kv/v3` was added by hand instead of `go mod tidy`

---

## f) Top 25 Things to Get Done Next

| # | Task | Impact | Effort | Ratio |
|---|------|--------|--------|-------|
| 1 | Commit the pending lint fixes (6 files, all ready) | 🟠 High | Tiny | ⭐⭐⭐⭐⭐ |
| 2 | Write and commit the status report | 🟠 High | Tiny | ⭐⭐⭐⭐⭐ |
| 3 | `git push` all commits | 🟠 High | Tiny | ⭐⭐⭐⭐⭐ |
| 4 | Update AGENTS.md: modules, test cmd, layer graph, patterns | 🟠 High | Small | ⭐⭐⭐⭐⭐ |
| 5 | Update SKILL.md: gRPC section, MemoryBus, new view store features | 🟠 High | Small | ⭐⭐⭐⭐ |
| 6 | Add transport/grpc + cmd/doc-check to CI per-module matrix | 🟡 Medium | Tiny | ⭐⭐⭐⭐ |
| 7 | Remove `var _ = errors.New` hack from command_server.go | 🟢 Low | Tiny | ⭐⭐⭐⭐ |
| 8 | Fix `toAnySlice` reflection → type switch | 🟢 Low | Tiny | ⭐⭐⭐⭐ |
| 9 | Remove unused `fmt` import from doc-check/main.go | 🟢 Low | Tiny | ⭐⭐⭐⭐ |
| 10 | Wire SEC's `NewCQRSAppFromBundle` into server.go | 🟡 Medium | Small | ⭐⭐⭐ |
| 11 | Fix transport/grpc genproto conflict for go.work inclusion | 🟠 High | Medium | ⭐⭐⭐ |
| 12 | Decide: raw SQL `ViewQuery.Where` vs structured `ViewFilter` | 🟡 Medium | Medium | ⭐⭐⭐ |
| 13 | Write transport/grpc query round-trip test | 🟡 Medium | Small | ⭐⭐⭐ |
| 14 | Add proto regen target to flake.nix | 🟡 Medium | Small | ⭐⭐⭐ |
| 15 | Document gRPC wire format in SKILL.md §6 | 🟡 Medium | Small | ⭐⭐⭐ |
| 16 | Add `transport/grpc` to the module dependency layer graph | 🟢 Low | Tiny | ⭐⭐⭐ |
| 17 | Consider removing `kv.ViewQuery.Where` raw SQL entirely | 🟡 Medium | Medium | ⭐⭐⭐ |
| 18 | Write automated ViewStore contract test for SQLViewStore | 🟡 Medium | Small | ⭐⭐⭐ |
| 19 | Add `nix run .#lint-grpc` target for standalone lint | 🟢 Low | Small | ⭐⭐⭐ |
| 20 | Migrate SEC to use `sqlite.New()` bundle in production | 🟠 High | Medium | ⭐⭐⭐ |
| 21 | Add Postgres integration test for SQLViewStore | 🟡 Medium | Small | ⭐⭐⭐ |
| 22 | Consider structured filter replacing raw SQL as the ONLY API | 🟡 Medium | Medium | ⭐⭐ |
| 23 | Add `command.Bus` to Bundle (optional field) | 🟢 Low | Small | ⭐⭐ |
| 24 | Explore `go-json-experiment/json` for transport/grpc proto encoding | 🟢 Low | Medium | ⭐⭐ |
| 25 | Consider NATS transport adapter (ADR-0025 next transport) | 🟢 Low | Large | ⭐ |

---

## g) Top #1 Question I Cannot Figure Out Myself

**The genproto conflict.** The root cause is:

- `cockroachdb/errors` (used by `event/` via `go-error-family`) depends on the old monolithic `google.golang.org/genproto`
- `google.golang.org/grpc` depends on the split `google.golang.org/genproto/googleapis/rpc`
- Both contain `googleapis/rpc/status` → ambiguous import in workspace mode

**What I tried:**
1. Adding `google.golang.org/genproto/googleapis/rpc` to transport/grpc's go.mod — didn't help because the old `genproto` is pulled by other workspace members
2. Removing transport/grpc from go.work — works but means workspace `go test ./...` doesn't cover it
3. Pinning a specific genproto version — Go's MVS doesn't support conflicts, only minimum versions

**The question:** Is there a way to make this work in workspace mode, or is `GOWORK=off` for transport/grpc the permanent answer? Upgrading `cockroachdb/errors` might help, but it's a transitive dependency I don't control.
