# Status Report — 2026-04-02 23:02

**Branch:** `master` | **HEAD:** `1ecdcb9` | **Ahead of origin:** 4 commits (not pushed)
**Go:** 1.26.0 (go.mod) / 1.26.1 (installed, Nix) | **Source:** 39 files, 3,222 LOC | **Tests:** 17 files, 4,663 LOC

---

## a) FULLY DONE ✅

### Bug Fixes

| #   | Fix                                                                                                                                | Commit    |
| --- | ---------------------------------------------------------------------------------------------------------------------------------- | --------- |
| 1   | **`LoadFromHistory` calls `Apply()`** — Was only incrementing version, never rebuilding domain state. Now accepts `Root` parameter | `0e5ec14` |
| 2   | **MemoryStore thread safety** — Added `sync.RWMutex` to all map operations (Save/Load/LoadFromVersion/Delete)                      | `412a9a3` |
| 3   | **Dispatcher data race** — `Handlers` map replaced with unexported `handlers` + `handlersMu sync.RWMutex` + `GetHandler()`         | `412a9a3` |
| 4   | **Store interface alignment** — `event.Store` now uses `id.AggregateID` + `event.Version` instead of `string`/`int`                | `64ffdba` |

### Refactoring

| #   | Change                                                                                | Commit    |
| --- | ------------------------------------------------------------------------------------- | --------- |
| 5   | **`ApplyEvent` → `RecordEvent`** — Disambiguates from `Root.Apply()`. 5 files updated | `78491c0` |
| 6   | **YAML field ordering** — Structs preserve definition order, maps sorted              | `031fb3b` |
| 7   | **`toSnakeCase` → `toDotAddress`** — Truthful naming                                  | `b8e2997` |
| 8   | **`fmt.Appendf` + tagged switch** — gopls hints                                       | `05cd4e3` |

### Features

| #   | Feature                                                                                        | Commit(s) |
| --- | ---------------------------------------------------------------------------------------------- | --------- |
| 9   | **Catalog system** — `catalog/`, `catalog/asyncapi/`, `catalog/eventcatalog/`, `catalog/yaml/` | multiple  |
| 10  | **EventCatalog frontmatter** — `schemaPath` + `sends`/`receives`                               | `0a46627` |
| 11  | **AsyncAPI functional options** — `WithServer()`, `WithDescription()`                          | `25bc5b7` |
| 12  | **Example user CQRS** — Full event sourcing flow                                               | `1050a9d` |
| 13  | **CI workflows** — test.yml + lint.yml with Go 1.26                                            | earlier   |

### Test Coverage (all 11 packages PASS)

| Package                 | Coverage |
| ----------------------- | -------- |
| `aggregate/`            | ~92%     |
| `catalog/`              | 91.9%    |
| `catalog/asyncapi/`     | 92.6%    |
| `catalog/eventcatalog/` | 86.4%    |
| `catalog/yaml/`         | 79.8%    |
| `command/`              | 90.5%    |
| `event/`                | 93.2%    |
| `internal/dispatcher/`  | 100%     |
| `pkg/id/`               | 88.0%    |
| `query/`                | 92.6%    |
| `xtypes/`               | 95.6%    |

---

## b) PARTIALLY DONE ⚠️

| #   | Item                              | Status                                                                                              | Remaining                                                                |
| --- | --------------------------------- | --------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| 1   | **Justfile creation**             | Research done (`dupl` is best Go-native tool, already in `.golangci.yml`). `just` v1.46.0 installed | Not created yet                                                          |
| 2   | **Deep code analysis**            | Two comprehensive reviews completed (21:22 and 22:58 reports)                                       | ~25 identified fixes not yet executed                                    |
| 3   | **MemoryStore `LoadFromVersion`** | Mutex added                                                                                         | Still returns sub-slice of internal backing array (shared mutation risk) |

---

## c) NOT STARTED ❌

### High Priority

| #   | Item                                                                                         | Impact |
| --- | -------------------------------------------------------------------------------------------- | ------ |
| 1   | Remove duplicated `Lifecycle` methods in `internal/dispatcher`                               | Medium |
| 2   | Fix `Register()` error semantics — use `ErrDispatcherClosed` not `ErrHandlerNotFound`        | Medium |
| 3   | Fix `query.Handler` to accept `context.Context`                                              | HIGH   |
| 4   | Fix `query.Handler` type alias (`=`) → type definition                                       | HIGH   |
| 5   | Add `aggregate.Repository` interface                                                         | HIGH   |
| 6   | Add full CQRS roundtrip integration test                                                     | HIGH   |
| 7   | Remove dead exports: `query.Result[T]`, `event.StreamOptions`/`BatchSize`/`Streamer`         | Medium |
| 8   | Remove unused sentinels: `ErrCommandValidation`, `ErrQueryValidation`, `ErrInvalidEventType` | Medium |
| 9   | Add validation to `command.New()`, `query.New()`, `NewEvent()`                               | Medium |
| 10  | Fix `catalog/registry.go Build()` — shared backing array + non-deterministic map iteration   | HIGH   |

### Medium Priority

| #   | Item                                                                                 | Impact |
| --- | ------------------------------------------------------------------------------------ | ------ |
| 11  | Standardize errors: replace `fmt.Errorf` with `cockroachdb/errors` in event/ xtypes/ | Medium |
| 12  | Fix `MemoryBus.Subscribe` nil handler check + error wrapping consistency             | Medium |
| 13  | Remove unused `Dispatcher.Dispatch` `_ H` parameter                                  | Low    |
| 14  | Make exporter fields unexported (asyncapi, eventcatalog)                             | Medium |
| 15  | Add nil-metadata + `WithMetadata` tests for event options                            | Medium |
| 16  | Add `LoadFromHistory` error path tests (aggregate + xtypes)                          | Medium |
| 17  | Fix `MemoryStore.LoadFromVersion` — copy slice instead of sub-slice                  | Medium |
| 18  | Fix asyncapi component key collision — namespace by message kind                     | Medium |
| 19  | Add `context.Context` to MemoryStore operations (currently `_ context.Context`)      | Medium |
| 20  | Create justfile with `build`, `test`, `lint`, `fd` targets                           | Medium |

### Lower Priority

| #   | Item                                                                                | Impact            |
| --- | ----------------------------------------------------------------------------------- | ----------------- |
| 21  | Remove duplicated state in `xtypes.TypedAggregate` (redundant with embedded `core`) | Low               |
| 22  | Fix `TypedCommand` to implement `command.Command` interface                         | Low               |
| 23  | Make `Root.ID()` return `id.AggregateID` instead of `string` (breaking)             | HIGH but breaking |
| 24  | Make `Event.AggregateID()` return `id.AggregateID` (breaking)                       | HIGH but breaking |
| 25  | Make `Command.AggregateID()` optional or remove from interface                      | HIGH but breaking |

---

## d) TOTALLY FUCKED UP 💥

### 1. Go Build Cache Corruption (environment, NOT code)

Nix-installed Go 1.26.0 + go.work requiring 1.26.1 causes cache corruption. Stdlib packages (`io/fs`, `runtime`, `runtime/debug`) can't be found after concurrent runs.

**Workaround:** `GOCACHE=$(mktemp -d) go build ./...` — slow but reliable.

### 2. `query.Handler` Design Violation

- Returns `any` — violates project's own "no any types" rule
- Missing `context.Context` — breaks tracing/timeouts/cancellation
- Uses type alias (`=`) — can't add methods

### 3. `catalog/registry.go Build()` Data Corruption

Returns `Catalog` whose `Service` slices share backing arrays with Registry's internal state. Subsequent `AddCommand`/`AddEvent` calls silently mutate previously-built catalog data.

### 4. 4 Commits Ahead of Origin

`78491c0`, `507c97f`, `412a9a3`, `0e5ec14` are not pushed yet.

---

## e) WHAT WE SHOULD IMPROVE

### Type Safety (THE most impactful area)

1. **`Root.ID()` returns `string`** — defeats branded type system. Forces `id.MustParseAggregateID(u.ID())` everywhere.
2. **`Event.AggregateID()` returns `string`** — same issue.
3. **`Command.AggregateID()` forced on all commands** — create commands don't have an aggregate yet.
4. **`event.Core.Payload()` returns mutable `[]byte`** — callers can corrupt internal state.
5. **`event.Core.Metadata()` returns mutable `*Metadata`** — same.

### Architecture

6. **No `aggregate.Repository` interface** — every user reimplements the load/save/publish pattern.
7. **`Lifecycle` duplicates `LifecycleMixin`** — two identical types for same purpose.
8. **`xtypes.TypedAggregate` stores redundant state** — `aggregateID` and `aggregateType` duplicated from embedded `core`.
9. **Duplicated validation** in `xtypes.EventBuilder.Build()` and `event.NewEvent()`.

### Consistency

10. **Mixed `fmt.Errorf` / `cockroachdb/errors`** — some errors have stack traces, some don't.
11. **`go-json-experiment/json` barely used** — only in `pkg/id/`, rest uses `encoding/json`.
12. **`query.Handler` inconsistent with `command.Handler`/`event.Handler`** — no context, returns `any`.

---

## f) Top 25 Next Steps (sorted by impact × urgency / effort)

| #   | Task                                                                                              | Impact | Effort |
| --- | ------------------------------------------------------------------------------------------------- | ------ | ------ |
| 1   | **`git push`** — 4 commits not pushed                                                             | HIGH   | Zero   |
| 2   | **Create justfile** with `build`/`test`/`lint`/`fd`                                               | MED    | Small  |
| 3   | **Remove duplicated `Lifecycle` methods**                                                         | MED    | Tiny   |
| 4   | **Fix `Register()` error sentinel**                                                               | MED    | Tiny   |
| 5   | **Fix `query.Handler` +`context.Context` + type def + no `any`**                                  | HIGH   | Medium |
| 6   | **Remove dead exports** (`Result[T]`, `StreamOptions`, `BatchSize`, `Streamer`, unused sentinels) | MED    | Small  |
| 7   | **Add validation to constructors** (`command.New`, `query.New`, `NewEvent`)                       | MED    | Small  |
| 8   | **Fix `Build()` shared backing array + sort maps**                                                | HIGH   | Small  |
| 9   | **Add `aggregate.Repository` interface**                                                          | HIGH   | Medium |
| 10  | **Add CQRS roundtrip integration test**                                                           | HIGH   | Medium |
| 11  | **Standardize error wrapping** (cockroachdb/errors everywhere)                                    | MED    | Small  |
| 12  | **Fix `MemoryBus.Subscribe` nil check + error consistency**                                       | MED    | Small  |
| 13  | **Make exporter fields unexported**                                                               | MED    | Small  |
| 14  | **Add missing tests** (nil-metadata, WithMetadata, LoadFromHistory error path)                    | MED    | Small  |
| 15  | **Fix `MemoryStore.LoadFromVersion`** — copy slice                                                | MED    | Small  |
| 16  | **Fix asyncapi component key collision**                                                          | MED    | Small  |
| 17  | **Add `context.Context` to MemoryStore**                                                          | MED    | Small  |
| 18  | **Remove unused `Dispatcher.Dispatch` `_ H` param**                                               | LOW    | Small  |
| 19  | **Remove redundant state in `TypedAggregate`**                                                    | LOW    | Small  |
| 20  | **Fix `TypedCommand` to implement `command.Command`**                                             | LOW    | Small  |
| 21  | **Breaking: `Root.ID()` → `id.AggregateID`**                                                      | HIGH   | Large  |
| 22  | **Breaking: `Event.AggregateID()` → `id.AggregateID`**                                            | HIGH   | Large  |
| 23  | **Breaking: Make `Command.AggregateID()` optional**                                               | HIGH   | Large  |
| 24  | **Fix `Payload()`/`Metadata()` immutability**                                                     | MED    | Small  |
| 25  | **Run `golangci-lint` + `dupl` and fix findings**                                                 | MED    | Medium |

---

## g) Top #1 Question 🤔

**Should we do the breaking API changes now (items #21-23)?**

Making `Root.ID()`, `Event.AggregateID()`, `Command.AggregateID()` return typed IDs instead of `string` would cascade through every file. The library is v0.x so breaking changes are fine per semver, and the branded type system is a key selling point — having core interfaces return `string` undermines it completely.

**My recommendation:** Yes, do it now. The library has few consumers, the change is mechanical, and every day we wait means more downstream code to break. But I need your go-ahead because it's a coordinated multi-file change.

**Second question (if I may):** How do you reliably run `go build`/`go test`? The `GOCACHE=$(mktemp -d)` workaround works but takes 30-90 seconds. Is there a faster way, or should I update `go.mod` to `go 1.26.1`?
