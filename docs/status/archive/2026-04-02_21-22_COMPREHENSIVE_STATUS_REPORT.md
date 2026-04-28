# Status Report — 2026-04-02 21:22

**Branch:** `master` | **Latest commit:** `412a9a3` | **Go:** 1.26.0 (installed: 1.26.1)
**Source:** 39 files, 3,222 lines | **Tests:** 17 files, 4,663 lines | **Packages:** 11

---

## a) FULLY DONE ✅

| #   | Item                                                                                                                                                                 | Commit(s) |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| 1   | **Fix `LoadFromHistory` to call `Apply()`** — Was only incrementing version, never rebuilding domain state. Now accepts `Root` parameter and calls `root.Apply(evt)` | `0e5ec14` |
| 2   | **Add mutex to `MemoryStore`** — Data race: concurrent Save/Load on unprotected `map[string][]Event`. Added `sync.RWMutex` with `Lock`/`RLock`                       | `412a9a3` |
| 3   | **Catalog system** — `catalog/`, `catalog/asyncapi/`, `catalog/eventcatalog/`, `catalog/yaml/`                                                                       | multiple  |
| 4   | **EventCatalog frontmatter** — `schemaPath` + `sends`/`receives` arrays                                                                                              | `0a46627` |
| 5   | **AsyncAPI functional options** — `WithServer()`, `WithDescription()`                                                                                                | `25bc5b7` |
| 6   | **YAML marshaler field ordering** — Preserves struct definition order, maps sorted for determinism                                                                   | `031fb3b` |
| 7   | **Rename `toSnakeCase` → `toDotAddress`**                                                                                                                            | `b8e2997` |
| 8   | **`fmt.Appendf` + tagged switch** — gopls hints                                                                                                                      | `05cd4e3` |
| 9   | **Integration tests + benchmarks** for catalog                                                                                                                       | `584fa89` |
| 10  | **Catalog coverage** 85.5% → 91.9%                                                                                                                                   | `a439186` |
| 11  | **Dispatcher tests** 0% → 100%                                                                                                                                       | earlier   |
| 12  | **Aggregate tests** 64% → 100%                                                                                                                                       | earlier   |
| 13  | **ID tests** 48% → 88%                                                                                                                                               | earlier   |
| 14  | **xtypes tests** 53% → 95.6%                                                                                                                                         | earlier   |
| 15  | **Event tests** 75% → 92.8%                                                                                                                                          | earlier   |
| 16  | **Example user CQRS flow** — aggregate.go, commands.go, events.go, handlers.go, main.go                                                                              | `1050a9d` |
| 17  | **CHANGELOG updated** with all catalog improvements                                                                                                                  | `a9593c7` |
| 18  | **CI workflows** — test.yml, lint.yml with Go 1.26 matrix                                                                                                            | earlier   |

---

## b) PARTIALLY DONE ⚠️

| #   | Item                          | Status                                                                                                                   | Remaining                                                                               |
| --- | ----------------------------- | ------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------- |
| 1   | **Comprehensive code review** | Deep analysis complete, execution plan drafted                                                                           | None of the identified fixes have been applied yet (see NOT STARTED)                    |
| 2   | **Create justfile**           | `just` v1.46.0 is installed. `.golangci.yml` already has `dupl` linter. Research done: `dupl` is the best Go-native tool | justfile not yet created, `just build`/`just lint`/`just fd` not run                    |
| 3   | **MemoryStore mutex**         | Added in commit `412a9a3` by user                                                                                        | Race with `LoadFromVersion` returning sub-slice of internal backing array still present |

---

## c) NOT STARTED ❌

### From This Session's Analysis (not yet executed):

| #   | Item                                                                                                  | Impact                                            | Effort |
| --- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------- | ------ |
| 1   | Fix `Build()` non-deterministic map iteration in `catalog/registry.go`                                | HIGH — tests produce different output on each run | Small  |
| 2   | Fix `Build()` shared backing array corruption in `catalog/registry.go`                                | HIGH — silent data corruption                     | Small  |
| 3   | Remove dead exports: `query.Result[T]`, `event.StreamOptions`/`BatchSize`/`Streamer`                  | MEDIUM — dead code confuses users                 | Small  |
| 4   | Remove unused sentinels: `ErrCommandValidation`, `ErrQueryValidation`, `ErrInvalidEventType`          | MEDIUM — dead code                                | Small  |
| 5   | Add validation to `command.New()`, `query.New()`, `NewEvent()`                                        | MEDIUM — empty types silently accepted            | Small  |
| 6   | Add nil handler check to `MemoryBus.Subscribe` + fix error wrapping inconsistency                     | MEDIUM — nil panic, inconsistent errors           | Small  |
| 7   | Remove unused `_ H` parameter from `Dispatcher.Dispatch`                                              | LOW — dead API surface                            | Small  |
| 8   | Unify `Lifecycle`/`LifecycleMixin` duplication in `internal/dispatcher`                               | LOW — confusing dual types                        | Small  |
| 9   | Make `asyncapi`/`eventcatalog` exporter fields unexported (use options only)                          | MEDIUM — exported mutable state                   | Small  |
| 10  | Add tests for nil-metadata option branches + `WithMetadata` in `event/`                               | MEDIUM — 8 uncovered branches                     | Small  |
| 11  | Add tests for `LoadFromHistory` error path in `aggregate/` and `xtypes/`                              | MEDIUM — error wrapping untested                  | Small  |
| 12  | Fix `query.Handler` uses `any` — violates project's own "no any types" rule                           | HIGH — design violation                           | Medium |
| 13  | Fix `query.Handler` missing `context.Context` — inconsistent with `command`/`event`                   | HIGH — context propagation broken                 | Medium |
| 14  | Fix `query.Middleware` readability — use `Handler` type alias                                         | LOW — readability                                 | Small  |
| 15  | Add `context.Context` to `MemoryStore.Save`/`Load` (currently ignored with `_`)                       | MEDIUM — cancellation impossible                  | Small  |
| 16  | Fix `internal/dispatcher` `Register` uses wrong sentinel `ErrHandlerNotFound` instead of closed error | MEDIUM — confusing error messages                 | Small  |
| 17  | Create justfile with `build`, `lint`, `fd` (dupl), `test`, `coverage` targets                         | MEDIUM — no justfile exists, only Makefile        | Small  |
| 18  | Run `golangci-lint run` and fix findings                                                              | HIGH — unknown lint state                         | Medium |
| 19  | Run `dupl` / `golangci-lint run --enable-only dupl` to find code duplication                          | MEDIUM — `.golangci.yml` has dupl enabled         | Small  |
| 20  | Fix `catalog/asyncapi/exporter.go` — component message key collision when command/event share same ID | MEDIUM — silent data loss                         | Small  |

### From TODO_LIST.md (not yet started):

| #   | Item                                                           | Priority  |
| --- | -------------------------------------------------------------- | --------- |
| 21  | Add aggregate `Repository` interface                           | 🔴 HIGH   |
| 22  | Add integration test: full CQRS roundtrip                      | 🔴 HIGH   |
| 23  | Add middleware (logging, recovery, retry, validation, metrics) | 🟡 MEDIUM |
| 24  | Add fuzzing for Parse functions                                | 🟡 MEDIUM |
| 25  | Add `AppendBatch` to Store                                     | 🟡 MEDIUM |
| 26  | Add snapshot store interface                                   | 🟡 MEDIUM |
| 27  | Add `query/pagination.go`                                      | 🟡 MEDIUM |

---

## d) TOTALLY FUCKED UP 💥

| #   | Issue                                         | Details                                                                                                                                                                                                                                                                         |
| --- | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Go toolchain hangs**                        | `go build` and `go test` commands hang indefinitely when run via background shell. Likely caused by Go 1.26.0 module requiring 1.26.1 toolchain download (GOWORK=off + GOTOOLCHAIN=auto triggers download but background shell never completes). **Blocking all verification.** |
| 2   | **`Build()` shared backing array**            | `catalog/registry.go:Build()` returns `Catalog` whose `Service.Commands`/`Events`/`Queries` slices share backing arrays with Registry's internal state. Subsequent `AddCommand` can corrupt previously-built catalog. **Silent data corruption bug.**                           |
| 3   | **`query.Handler` returns `any`**             | The project's own `AGENTS.md` says "No `any` types" but `query.Handler` and `query.Dispatch` return `any`. This is a fundamental design violation in a type-safe CQRS library.                                                                                                  |
| 4   | **`query.Handler` missing `context.Context`** | `command.Handler` and `event.Handler` both take `context.Context`. `query.Handler` does not — breaks context propagation, making tracing/timeouts/cancellation impossible for queries.                                                                                          |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

- **`query.Handler` needs `context.Context`** — This is a breaking API change but critical for correctness. Every Go HTTP/RPC handler needs context propagation.
- **`query.Result[T]` should replace `any`** — Instead of `func(Query) (any, error)`, use `func(Query) (Result[T], error)` or generic `DispatchTyped[T]` pattern.
- **Unify `Lifecycle`/`LifecycleMixin`** — Two identical types serving the same purpose. One should suffice.
- **`catalog.Registry.Build()`** should deep-copy slices, not return shared backing arrays.
- **Exporter fields should be unexported** — Both `asyncapi` and `eventcatalog` exporters have exported mutable fields alongside an options pattern. Pick one.

### Type Safety

- **`aggregate.Root.ID()` returns `string`** — Loses the branded `id.AggregateID` type. Should return `id.AggregateID`.
- **`event.Core.Payload()` returns mutable `[]byte`** — Callers can corrupt internal state.
- **`event.Core.Metadata()` returns mutable `*Metadata`** — Same issue.
- **`catalog/asyncapi/types.go` uses `any` in `Components.Schemas`** — Should use `*catalog.Schema`.

### Library Usage

- **`go-json-experiment/json`** is imported but only used in `pkg/id/id.go`. Rest of project uses `encoding/json`. Should be consistent.
- **`cockroachdb/errors`** is imported but barely leveraged — project mostly uses `fmt.Errorf("...: %w", err)` instead of `errors.Wrap`/`errors.WithDetail`.
- **Custom YAML marshaler** (`catalog/yaml/`) could be replaced with `gopkg.in/yaml.v3` — but this is a deliberate design choice (zero-dependency YAML).

### Testing

- **Go toolchain issue blocks all test execution** — Must resolve before any verification can happen.
- **Event option nil-metadata branches** (8 uncovered paths) need one test.
- **`LoadFromHistory` error paths** untested in both `aggregate/` and `xtypes/`.
- **`MemoryBus.Subscribe` nil handler** — no nil check, will panic.

---

## f) TOP 25 THINGS TO DO NEXT

Sorted by **impact × urgency / effort**:

| #   | Task                                                                                             | Impact   | Effort |
| --- | ------------------------------------------------------------------------------------------------ | -------- | ------ |
| 1   | **Fix Go toolchain / test execution** — unblock all verification                                 | CRITICAL | Small  |
| 2   | **Create justfile** with `build`, `test`, `lint`, `fd` targets                                   | HIGH     | Small  |
| 3   | **Run `just build`** and fix build errors                                                        | HIGH     | Small  |
| 4   | **Run `just lint`** (`golangci-lint run`) and catalog findings                                   | HIGH     | Medium |
| 5   | **Run `just fd`** (`dupl`) and catalog code duplication                                          | HIGH     | Small  |
| 6   | **Fix `Build()` shared backing array corruption** in registry.go                                 | HIGH     | Small  |
| 7   | **Fix `Build()` non-deterministic map iteration** — sort by key                                  | HIGH     | Small  |
| 8   | **Remove dead exports**: `Result[T]`, `StreamOptions`, `BatchSize`, `Streamer`, unused sentinels | MEDIUM   | Small  |
| 9   | **Add validation** to `command.New()`, `query.New()`, `NewEvent()`                               | MEDIUM   | Small  |
| 10  | **Fix `query.Handler`** — add `context.Context`, remove `any` return type                        | HIGH     | Medium |
| 11  | **Fix `MemoryBus.Subscribe` nil handler check** + error wrapping consistency                     | MEDIUM   | Small  |
| 12  | **Unify `Lifecycle`/`LifecycleMixin`** — remove duplication                                      | LOW      | Small  |
| 13  | **Remove unused `Dispatcher.Dispatch` `_ H` parameter**                                          | LOW      | Small  |
| 14  | **Make exporter fields unexported** (asyncapi, eventcatalog)                                     | MEDIUM   | Small  |
| 15  | **Add nil-metadata + `WithMetadata` tests** for event options                                    | MEDIUM   | Small  |
| 16  | **Add `LoadFromHistory` error path tests** (aggregate + xtypes)                                  | MEDIUM   | Small  |
| 17  | **Fix `MemoryStore.LoadFromVersion`** — copy slice, don't return sub-slice                       | MEDIUM   | Small  |
| 18  | **Fix `internal/dispatcher Register`** — use proper closed error sentinel                        | MEDIUM   | Small  |
| 19  | **Fix asyncapi component key collision** — namespace by message kind                             | MEDIUM   | Small  |
| 20  | **Add `aggregate.Repository` interface**                                                         | HIGH     | Medium |
| 21  | **Add full CQRS roundtrip integration test**                                                     | HIGH     | Medium |
| 22  | **Add `context.Context` to MemoryStore operations** (currently ignored)                          | MEDIUM   | Small  |
| 23  | **Fix `Payload()` / `Metadata()` immutability** — return copies                                  | MEDIUM   | Small  |
| 24  | **Update TODO_LIST.md** with completed items + new findings                                      | LOW      | Small  |
| 25  | **Update CHANGELOG.md** with all fixes from this session                                         | LOW      | Small  |

---

## g) TOP #1 QUESTION 🤔

**The Go toolchain issue is blocking all verification.** Running `go build` or `go test` hangs indefinitely (background shell never completes). The installed Go is 1.26.1 but `go.mod` says `go 1.26.0`. With `GOWORK=off` + `GOTOOLCHAIN=auto`, Go tries to download the 1.26.0 toolchain which may hang.

**Question for you:** How should I run `go build` and `go test` in this project? What exact command/flags do you normally use? Should I update `go.mod` to `go 1.26.1`? Or is there a specific `GOTOOLCHAIN` / `GOWORK` / environment variable combination I should use?

---

## Session Summary

This session spanned multiple interruptions. The deep analysis phase was thorough — all 39 source files and 17 test files were read, bugs were identified, and a comprehensive plan was drafted. However, **only one fix was actually committed and pushed** (`0e5ec14` — LoadFromHistory). The second fix (MemoryStore mutex) was committed by the user (`412a9a3`). The remaining 20+ identified issues have not been touched.

The primary blocker is **inability to run Go commands reliably** — every `go build` / `go test` invocation hangs in background shell, making verification impossible and creating a feedback loop of uncertainty.
