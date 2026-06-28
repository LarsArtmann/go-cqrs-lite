# Status Report — 2026-06-15 07:41

> **Comprehensive project status for go-cqrs-lite v2.3.0**
> Generated: 2026-06-15 07:41 UTC

---

## Executive Summary

go-cqrs-lite is a **28-module CQRS + Event Sourcing library** for Go at v2.3.0. All 25 testable modules pass `GOWORK=off go test` with 0 failures. The codebase has 700 Go files, 30 READMEs (one per module), 20 ADRs, and 1222 tracked API surface exports.

This session focused on **documentation overhaul** and **fixing a critical build regression** introduced by the CQRS audit trail feature (missing `replace` directives in 6 go.mod files).

---

## a) FULLY DONE ✅

### Documentation Overhaul (This Session)

| Task                             | Status  | Details                                                                                                                  |
| -------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------ |
| Root README.md rewrite           | ✅ Done | 745 → 159 lines. Lean sales page with module catalog linking to all 28 module READMEs                                    |
| TODO_LIST.md recreation          | ✅ Done | 372 → 63 lines. Purged 197+ done items. Only 15 open actionable items remain                                             |
| 4 missing module READMEs created | ✅ Done | `cmd/api-stability`, `cmd/cqrs-gen`, `example/encryption`, `testutil`                                                    |
| command/README.md updated        | ✅ Done | Added `PersistedCommand`, `CommandStore`, `CommandJournal`, `SeekableCommandJournal` documentation                       |
| query/README.md updated          | ✅ Done | Added `PersistedQuery`, `QueryStore`, `QueryJournal`, `SeekableQueryJournal` documentation                               |
| memory/README.md fixed           | ✅ Done | Fixed wrong API names (`NewStore`→`NewMemoryStore`), added `MemoryCommandStore`/`MemoryQueryStore`                       |
| Stale refs fixed                 | ✅ Done | Removed phantom `core` module from catalog/listing deps; removed stale Turso code from storage; removed stale Pebble dep |
| Badge standardization            | ✅ Done | All 22 library modules now have consistent `/v2` badges with lowercase paths                                             |
| Badges added to 7 READMEs        | ✅ Done | middleware, codec, turso, watermill, integration, listing, memory                                                        |

### Build Regression Fix (This Session)

| Task                            | Status  | Details                                                                                            |
| ------------------------------- | ------- | -------------------------------------------------------------------------------------------------- |
| 6 go.mod files fixed            | ✅ Done | Added `replace ... query/v2 => ../query` to event, decider, listing, projection, schema, watermill |
| API surface golden file updated | ✅ Done | `docs/api_surface.txt` regenerated: 1156 → 1222 exports (66 new from audit trail feature)          |

### Prior Completed Work (v2.3.0 Release)

| Area                 | Status         | Highlights                                                                                             |
| -------------------- | -------------- | ------------------------------------------------------------------------------------------------------ |
| Core types           | ✅ Done        | event, command, query, decider, id, dispatcher, codec — all stable                                     |
| Event sourcing       | ✅ Done        | Store/Sink/Source ISP split, Journal/SeekableJournal, BackwardsSource, StreamLoader                    |
| CQRS audit trail     | ✅ Done (code) | CommandJournal, SeekableCommandJournal, PersistedQuery, QueryStore, QueryJournal, SeekableQueryJournal |
| Event bus + reactive | ✅ Done        | EventBus, MemoryBus, middleware chains, publish-side middleware                                        |
| Command dispatch     | ✅ Done        | Typed handlers, middleware, lifecycle, BasicCommand                                                    |
| Query dispatch       | ✅ Done        | TypedHandler[Q,R], pagination, PaginatedResult[T]                                                      |
| Decider pattern      | ✅ Done        | Pure-function aggregate, Repository, snapshot support                                                  |
| Storage              | ✅ Done        | SQL (PostgreSQL/SQLite), Pebble, Turso — all Store implementations                                     |
| Schema evolution     | ✅ Done        | Upcaster, VersionedStore, schema version strong type                                                   |
| Projections          | ✅ Done        | Runner with replay+live, HandlerRegistry, DLQ, health checks                                           |
| Middleware           | ✅ Done        | 24 factories: logging, retry, validation, recovery, circuit breaker, OTel, SSE, health                 |
| Signing              | ✅ Done        | HMAC-SHA256, Ed25519, multi-sig                                                                        |
| Encryption           | ✅ Done        | XChaCha20-Poly1305, AES-256-GCM, key rotation, EncryptedStore wrapper                                  |
| Catalog              | ✅ Done        | AsyncAPI 3.0, EventCatalog, OpenAPI, D2 exporters                                                      |
| Listing              | ✅ Done        | Aggregate listing, tombstone detection, StatusMiddleware                                               |
| Watermill adapter    | ✅ Done        | Publisher/Subscriber adapters                                                                          |
| CI/CD                | ✅ Done        | GitHub Actions: build, vet, test, lint, race, coverage, gosec, benchmark regression                    |
| Testing              | ✅ Done        | BDD (Ginkgo), property-based (rapid), golden/snapshot (go-snaps), 84–100% coverage                     |
| Lint                 | ✅ Done        | 0 issues across all modules                                                                            |

---

## b) PARTIALLY DONE ⚡

### CQRS Audit Trail — Tests & SQL Backends

The interfaces and in-memory implementations are **done and working**, but tests and SQL backends are missing:

| Item                                               |    Code     | Tests |           SQL           |
| -------------------------------------------------- | :---------: | :---: | :---------------------: |
| CommandJournal / SeekableCommandJournal interfaces |     ✅      |  ❌   |           ❌            |
| MemoryCommandStore (ReadAll, ReadFrom)             |     ✅      |  ❌   |           N/A           |
| PersistedQuery / QueryStore interfaces             |     ✅      |  ❌   |           ❌            |
| MemoryQueryStore                                   |     ✅      |  ❌   |           N/A           |
| SQLCommandStore                                    | ✅ (exists) |  ✅   | Missing journal methods |
| SQLQueryStore                                      |     ❌      |  ❌   |   ❌ (doesn't exist)    |
| Command-event causality tracing                    |     ✅      |  ✅   |           N/A           |
| MemoryCommandBus                                   |     ✅      |  ✅   |           N/A           |

### Documentation Gaps

| Item                                               | Status                                                                             |
| -------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `query/doc.go` — new store types undocumented      | Partially done (README updated, doc.go not)                                        |
| `command/doc.go` — new store types undocumented    | Partially done (README updated, doc.go not)                                        |
| `go-snaps` snapshot tests across remaining modules | ~50% done (event, command, query, catalog, projection have them; 12 modules don't) |

---

## c) NOT STARTED ❌

| Item                                                | Priority | Notes                                                                  |
| --------------------------------------------------- | -------- | ---------------------------------------------------------------------- |
| SQLQueryStore                                       | HIGH     | SQL backend for query persistence (parity with SQLCommandStore)        |
| SQLCommandStore journal methods (ReadAll, ReadFrom) | HIGH     | SQL implementation of CommandJournal/SeekableCommandJournal            |
| `SQLBackend.QueryStore()` facade                    | MEDIUM   | One-stop shop for all SQL stores                                       |
| Query module sentinel errors                        | MEDIUM   | `ErrQueryStoreClosed`, `ErrQueryNotFound`, `ErrDuplicateQuery` missing |
| Playwright E2E tests for example/user               | LOW      | Requires Node.js browser testing infrastructure                        |
| Docker multi-arch CI build                          | LOW      | linux/amd64 + linux/arm64 GitHub Actions job                           |
| `jsonv2` codec experiment                           | LOW      | Behind build tag, performance experiment                               |
| Arena allocation experiment                         | LOW      | Go experiment for high-throughput event creation                       |

---

## d) TOTALLY FUCKED UP! 🔥

### Build Regression (FIXED THIS SESSION)

**What happened**: The CQRS audit trail feature added `query/v2` as a dependency to `memory/v2`. Six modules (event, decider, listing, projection, schema, watermill) use `memory` as a test dependency. Their `go.mod` files listed `query/v2` as `// indirect` but **lacked the `replace` directive** needed for `GOWORK=off` local resolution.

**Impact**: 6 out of 25 modules failed `GOWORK=off go test` — the CI per-module verification step would have failed. This was a **silent regression** because `go.work` (workspace mode) masks missing replace directives.

**Root cause**: When `memory/go.mod` was updated to add `query/v2`, the dependent modules' replace blocks were not updated. The `go.work` workspace hides this — only `GOWORK=off` per-module testing exposes it.

**Fix**: Added `replace github.com/larsartmann/go-cqrs-lite/query/v2 => ../query` to all 6 affected `go.mod` files. All 25 modules now pass.

**Lesson**: ALWAYS run `GOWORK=off go test ./...` for every module after adding a new cross-module dependency. The workspace masks missing replace directives.

### Pre-existing Issues (Not Fixed)

| Issue                                                  | Severity | Notes                                                                                                                                                                                                |
| ------------------------------------------------------ | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| api-stability parallel test race                       | LOW      | `TestAPISurfaceCheck` and `TestAPISurfaceUpdateIdempotent` both use `t.Parallel()` and access the same golden file — one reads while the other writes. Pass individually, can race when run together |
| `MemoryQueryStore.Close()` returns nil unconditionally | LOW      | Doesn't use Lifecycle pattern like `MemoryCommandStore` does                                                                                                                                         |
| `query.BasicQuery` has no metadata                     | MEDIUM   | Inconsistent with `BasicCommand` which carries correlation/tracing context                                                                                                                           |

---

## e) WHAT WE SHOULD IMPROVE! 💡

### Architecture & Design

1. **Replace directive maintenance is fragile** — Every time a module gains a new internal dependency, ALL modules that transitively depend on it need their replace blocks updated. Consider a `go.work`-based CI check that validates replace completeness.

2. **Test deps pollute production go.mod** — Go doesn't support separate test-only require blocks. 12+ modules carry test-only dependencies in their production `require` blocks. This inflates consumer transitive deps. Accept the tradeoff or push for Go tooling improvement.

3. **API surface golden file is fragile** — The parallel test design causes intermittent failures. Either serialize the tests or use separate golden files.

4. **Documentation drift detection** — No automated check that module READMEs match actual exports. The api-stability tool could be extended to verify README code examples compile.

### Code Quality

5. **Query module parity gaps** — Command module has `ErrStoreClosed`, `ErrCommandNotFound`, `ErrDuplicateCommand`. Query module has none despite having equivalent store interfaces. This asymmetry will confuse consumers.

6. **Missing tests for new audit trail code** — All new journal/store interfaces and memory implementations lack dedicated tests. Coverage will drop if CI runs per-module.

7. **Doc.go files not updated** — `query/doc.go` and `command/doc.go` don't document the new store/journal types. READMEs are updated but pkg.go.dev examples are missing.

### Developer Experience

8. **No module-level integration test runner** — Running all 25 modules requires a bash loop. The `nix run .#test` command exists but could be more discoverable.

9. **Root README was bloated** — The old README had 745 lines of tutorials that belonged in module READMEs. Fixed this session, but the pattern should be: root = sales page, module = deep dive.

10. **Module README template inconsistency** — Fixed this session (badges, structure), but no enforcement. A linter or template generator would prevent drift.

---

## f) Top 25 Things to Get Done Next 🎯

### HIGH Priority (Must Do)

1. **Write tests for `MemoryCommandStore` journal methods** — `ReadAll`, `ReadFrom`, closed-store behavior, empty-store edge cases
2. **Write tests for `query/store.go`** — `NewPersistedQuery` validation, `Payload()` defensive copy, `Metadata()` clone
3. **Write tests for `MemoryQueryStore`** — `SaveQuery`, `LoadQueries`, `ReadAllQueries`, `ReadQueriesFrom`
4. **Add query module sentinel errors** — `ErrQueryStoreClosed`, `ErrQueryNotFound`, `ErrDuplicateQuery`
5. **Update `query/doc.go`** — Document `PersistedQuery`, `QuerySink/Source/Store`, `QueryJournal`, `SeekableQueryJournal`
6. **Update `command/doc.go`** — Document `PersistedCommand`, `CommandSink/Source/Store`, `CommandJournal`, `SeekableCommandJournal`
7. **Add `SQLCommandStore` journal support** — `ReadAll`, `ReadFrom` for SQL-backed command audit
8. **Create `SQLQueryStore`** — SQL backend for query persistence (parity with `SQLCommandStore`)
9. **Fix `MemoryQueryStore.Close()`** — Use Lifecycle pattern for consistency with `MemoryCommandStore`
10. **Fix api-stability parallel test race** — Remove `t.Parallel()` or use mutex on golden file access

### MEDIUM Priority (Should Do)

11. **Add `replace` directive CI check** — Script that verifies all modules pass `GOWORK=off go test`
12. **Add `query.BasicQuery` metadata** — Match `BasicCommand` pattern for distributed tracing consistency
13. **Complete `go-snaps` across remaining modules** — signing, middleware, storage, listing, watermill, pebble, turso, codec, otel, schema, snapshot, memory
14. **Fix `storage/README.md` Pebble example** — `db` variable shadows `pebble` package name
15. **Add `SQLBackend.QueryStore()` facade** — One method to access all SQL stores
16. **Add Docker multi-arch CI build** — linux/amd64 + linux/arm64
17. **Create module README template/linter** — Enforce badge presence, section structure, cross-links
18. **Add `event/doc.go` causality docs** — Document `CommandCausalityEnricher`, `WithCommandCausality`, `ProcessingMode`

### LOW Priority (Nice to Have)

19. **Add `jsonv2` codec experiment** — Behind build tag, performance comparison
20. **Add Playwright E2E tests** — Health + command→event→query flow for example/user
21. **Add arena allocation experiment** — Go experiment for high-throughput event creation
22. **Create CQRS-lite dashboard** — Web UI for inspecting aggregates, events, projections
23. **Add streaming event reads** — Iterator pattern without materializing full slice
24. **Add gRPC transport adapter** — Protobuf-based command/query/event transport
25. **Add NATS/Redis Stream adapter** — Message bus integration beyond Watermill

---

## g) Top #1 Question I Cannot Figure Out 🔴

**How should the `replace` directive maintenance burden scale as the module count grows?**

The current approach requires manually adding `replace github.com/larsartmann/go-cqrs-lite/X/v2 => ../X` to every module that transitively depends on X. With 28 modules and growing, this is O(n²) maintenance — adding one new internal dependency can require updating 6+ go.mod files.

The `go.work` workspace masks this completely — everything "just works" in workspace mode. But CI requires `GOWORK=off` per-module verification (for consumer import isolation), which exposes every missing replace directive.

**Options I'm considering but can't decide between:**

1. **Script-generated replace blocks** — A tool that scans `go.work` and auto-injects replace directives into all go.mod files
2. **Accept the workspace-only model** — Drop `GOWORK=off` CI verification, accept that modules only work in workspace context
3. **Reduce module count** — Consolidate tightly-coupled modules (the `go-modularize` approach)
4. **Go upstream feature request** — Push for `go.mod` to support workspace-aware replace resolution

This is a fundamental architecture decision that affects the project's long-term maintainability.

---

## Project Metrics

| Metric                   | Value                                                |
| ------------------------ | ---------------------------------------------------- |
| Modules                  | 28 (22 library + 2 cmd + 3 examples + 1 integration) |
| Go files                 | 700                                                  |
| Testable modules passing | 25/25 (100%)                                         |
| Lint issues              | 0                                                    |
| API surface exports      | 1222                                                 |
| ADRs                     | 20                                                   |
| Module READMEs           | 30 (100% coverage)                                   |
| Test coverage            | 84–100% across modules                               |
| Go version               | 1.26.3                                               |
| CI                       | GitHub Actions (Nix-based)                           |
| Version                  | v2.3.0                                               |

---

_Test methodology: `cd <module> && GOWORK=off go test ./... -count=1` for each of 25 testable modules. All 25 pass._
