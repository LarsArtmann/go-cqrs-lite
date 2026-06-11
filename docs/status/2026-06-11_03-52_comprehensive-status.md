# go-cqrs-lite — Comprehensive Status Report

**Date**: 2026-06-11 03:52 UTC
**Reporter**: Crush (automated)
**Branch**: `master`
**Last tag**: v2.2.0 (81 commits since v2.1.0)
**Recent sprint focus**: Superb Types (phantom types, strong IDs), Encryption (XChaCha20-Poly1305, codec wrapper), Catalog (EventCatalog exporter, DocumentInfo consolidation)

---

## a) FULLY DONE ✓

### Core Library (22 modules)

| Module         | Status        | Notes                                                                                                            |
| -------------- | ------------- | ---------------------------------------------------------------------------------------------------------------- |
| `event/`       | ✅ Production | EventSink/Source, EventBus, ImmutableEvent, reactive streams (ro), zero-copy internal reads, tombstone detection |
| `command/`     | ✅ Production | Dispatcher, Handler, CommandBus (ro), FilterCommandType                                                          |
| `query/`       | ✅ Production | Dispatcher, TypedHandler[Q,R], PaginatedResult[T], QueryBus (ro)                                                 |
| `decider/`     | ✅ Production | Pure-function Decider[State], Repository, Execute/Load                                                           |
| `id/`          | ✅ Production | Branded IDs via go-branded-id (id.Of[T] = cbid.ID[T, ulid.ULID])                                                 |
| `dispatcher/`  | ✅ Production | Generic Dispatcher[H,M] with LifecycleMixin                                                                      |
| `schema/`      | ✅ Production | Upcaster, VersionedStore, schema evolution                                                                       |
| `snapshot/`    | ✅ Production | Snapshot stores, EveryNEvents strategy                                                                           |
| `memory/`      | ✅ Production | In-memory Store/Bus/SnapshotStore/CheckpointStore                                                                |
| `catalog/`     | ✅ Production | Registry, SchemaFromType[T], AsyncAPI/D2/EventCatalog/OpenAPI exporters                                          |
| `middleware/`  | ✅ Production | Logging, Retry, Recovery, Validation, Metrics, OTel Tracing+Metrics, SSE, HealthCheck                            |
| `signing/`     | ✅ Production | HMAC-SHA256, Ed25519, multisig, middleware                                                                       |
| `encryption/`  | ✅ Production | XChaCha20-Poly1305, AES-256-GCM, codec wrapper, Algorithm enum, KeyID phantom type                               |
| `projection/`  | ✅ Production | Runner (replay+live), HandlerRegistry, Builder with On[T]()                                                      |
| `storage/`     | ✅ Production | SQLEventStore/SnapshotStore/CheckpointStore (PG/SQLite/Turso)                                                    |
| `otel/`        | ✅ Production | Shared OTel helpers: Tracer, Meter, Spans, Attributes                                                            |
| `listing/`     | ✅ Production | AggregateListing, StatusMiddleware, tombstone detection                                                          |
| `watermill/`   | ✅ Production | Watermill protocol adapter                                                                                       |
| `pebble/`      | ✅ Production | Embedded KV event store                                                                                          |
| `codec/`       | ✅ Production | JSON, Raw passthrough                                                                                            |
| `turso/`       | ✅ Production | Turso database connector                                                                                         |
| `integration/` | ✅ Production | Cross-module tests (command, event, query, signing, encryption)                                                  |

### CMD Tools

| Module               | Status        | Notes                                         |
| -------------------- | ------------- | --------------------------------------------- |
| `cmd/cqrs-gen/`      | ✅ Production | Code generator for typed handler registration |
| `cmd/api-stability/` | ✅ Production | API surface checker against golden file       |

### Infrastructure

- **CI**: GitHub Actions ci.yml — Nix-based build/vet/test/lint/race/coverage, GOWORK=off per-module
- **Nix flake**: flake-parts + treefmt-nix, go_1_26, gofumpt + goimports + nixfmt
- **Module graph**: 7-layer dependency graph enforced by `nix run .#check-layers`
- **Coverage**: 84–100% across 32 packages
- **v2.1.0 + v2.2.0 released**: All modules tagged, /v2 semantic import paths

### Recent Session Work (this session)

- **ErrorExporter deprecation**: Collapsed `ErrorExporter` into `type ErrorExporter = Exporter[error]` — zero-cost type alias with `// Deprecated:` annotation. Zero consumers outside repo.
- **Deduplication analysis**: Ran art-dupl on `catalog/` — all 34 reported clones are idiomatic test setup boilerplate. No harmful duplication. Accepted as-is.

---

## b) PARTIALLY DONE ⚠️

| Item                               | Status                    | What's Missing                                                                                                                                                                         |
| ---------------------------------- | ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Test deduplication in catalog/** | ⚠️ Designed, not executed | `cattest` helpers exist but 34 test sites still inline `catalog.Service{ID:..., Name:..., Version:"1.0.0"}`. API was designed, consumer response was positive. Migration not yet done. |
| **Phantom types rollout**          | ⚠️ In progress            | Strong IDs added to many modules (catalog, encryption KeyID, etc.) but some modules may still use raw strings for IDs in tests or internal paths                                       |
| **example/catalog-server/**        | ⚠️ Broken                 | `gopls` shows compilation errors: `IncompatibleAssign` for `Name` and `Summary` types (phantom type mismatch). Needs fixing.                                                           |
| **example/user/catalog.go**        | ⚠️ Broken                 | `IncompatibleAssign`: `[]string` vs `[]catalog.Protocol`. Phantom type introduction didn't update examples.                                                                            |
| **Golden test drift**              | ⚠️ Recurring              | Golden tests have drifted before (commit e699df0 fixed it). CI catches it but it's a recurring maintenance burden.                                                                     |

---

## c) NOT STARTED 📋

| Item                                    | Priority | Notes                                                                             |
| --------------------------------------- | -------- | --------------------------------------------------------------------------------- |
| v3.0 planning                           | Low      | `Exporter[T]` return contract change (`(*T, error)`) deferred to v3               |
| Dedicated saga module                   | Low      | Explicitly out of scope — saga pattern emerges from projection + command dispatch |
| Benchmark regression CI                 | Medium   | Benchmark baselines exist but automated regression detection in CI is partial     |
| Module-level READMEs for all 31 modules | Medium   | Some modules have READMEs, not all                                                |
| Consumer-facing getting-started guide   | High     | README is good but lacks a step-by-step "import and use" tutorial                 |
| Performance optimization pass           | Medium   | v2.1.0 had major perf work but further alloc reduction possible                   |
| API stability CI gate                   | Medium   | `cmd/api-stability` exists but may not be in CI pipeline                          |
| Coverage gap analysis                   | Low      | Most modules >90% but no automated enforcement                                    |

---

## d) TOTALLY FUCKED UP 💥

| Item                                                      | Severity | Details                                                                                                                                                                                        |
| --------------------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **example/catalog-server/** compilation errors            | 🔴 HIGH  | Two `IncompatibleAssign` errors from phantom type introduction. The `Name` and `Summary` types are now phantom-branded but examples still pass raw strings. Must be fixed before next release. |
| **example/user/catalog.go** compilation error             | 🔴 HIGH  | `[]string` passed where `[]catalog.Protocol` expected. Same root cause: phantom types not applied to examples.                                                                                 |
| **eventcatalog/writer_frontmatter_test.go unused writes** | 🟡 LOW   | `gopls` reports `unusedwrite` for `Version` (line 108) and `Owners` (line 110). Not broken but sloppy.                                                                                         |

---

## e) WHAT WE SHOULD IMPROVE 🏗️

1. **Fix broken examples NOW** — `example/catalog-server` and `example/user` don't compile. This is the highest-priority item. Examples are the first thing consumers see.
2. **Migrate catalog test sites to cattest helpers** — The design is approved. Just do it. Eliminates 34 clones and makes future test writing faster.
3. **Automate golden test drift prevention** — Add a CI check or pre-commit hook that runs `go test ./... -update` and fails if the diff is non-empty (catches tests that assert against stale golden files).
4. **Phantom type audit** — Do a full sweep: grep for raw `string` where phantom types (`Name`, `Summary`, `ServiceID`, etc.) should be used. Fix the stragglers.
5. **Module README consistency** — Every module should have a README with: what it does, import path, one usage example, link to pkg.go.dev.
6. **API stability in CI** — Ensure `cmd/api-stability` runs as a CI gate so breaking changes to public API are caught.
7. **Error context enrichment** — Continue the pattern from commit 2e4274f1 (adding context to error paths). Storage, middleware, and catalog still have bare `fmt.Errorf` without operation context in some paths.

---

## f) Top 25 Things We Should Get Done Next

| #   | Task                                                                    | Impact      | Effort | Rationale                                       |
| --- | ----------------------------------------------------------------------- | ----------- | ------ | ----------------------------------------------- |
| 1   | **Fix example/catalog-server compilation**                              | 🔴 Critical | 10min  | Examples don't compile. Blocks consumers.       |
| 2   | **Fix example/user/catalog.go compilation**                             | 🔴 Critical | 5min   | Same root cause as #1.                          |
| 3   | **Migrate 34 catalog test sites to cattest helpers**                    | High        | 1hr    | Eliminates all reported clones, approved design |
| 4   | **Fix unused writes in eventcatalog frontmatter test**                  | Low         | 5min   | Clean lint                                      |
| 5   | **Phantom type audit: grep for raw string where branded types belong**  | High        | 30min  | Consistency, type safety                        |
| 6   | **Add CI gate for api-stability checker**                               | High        | 30min  | Prevents silent breaking changes                |
| 7   | **Automate golden test drift detection**                                | Medium      | 1hr    | Recurring pain point                            |
| 8   | **Write consumer getting-started guide (README section)**               | High        | 2hr    | First thing new users see                       |
| 9   | **Module README audit: ensure all 31 have README**                      | Medium      | 2hr    | Discoverability                                 |
| 10  | **Add `nix run .#coverage` command**                                    | Medium      | 30min  | Developer experience                            |
| 11  | **Error context enrichment pass on storage/sql**                        | Medium      | 1hr    | Production debugging quality                    |
| 12  | **Error context enrichment pass on middleware/**                        | Medium      | 1hr    | Same                                            |
| 13  | **Add benchmark regression detection in CI**                            | Medium      | 2hr    | Prevent perf regressions                        |
| 14  | **Extract shared test helpers from integration/ into testutil/**        | Low         | 1hr    | Already started (commit 4408c003), finish it    |
| 15  | **Add `// Deprecated:` annotations to any remaining legacy APIs**       | Low         | 1hr    | Consumer migration guidance                     |
| 16  | **Coverage enforcement: fail CI if any module drops below 80%**         | Medium      | 1hr    | Quality gate                                    |
| 17  | **Add doc.go with pkg.go.dev examples to modules missing them**         | Medium      | 2hr    | API documentation                               |
| 18  | **Review catalog/exporter.go for v3 migration path**                    | Low         | 30min  | Plan ahead for Exporter[(*T, error)]            |
| 19  | **Add `go vulncheck` to CI**                                            | High        | 30min  | Security                                        |
| 20  | **Review all `//nolint` directives for necessity**                      | Low         | 1hr    | Some may be stale                               |
| 21  | **Consolidate test assertion helpers across sub-packages**              | Low         | 1hr    | cattest/assertions.go is good, extend pattern   |
| 22  | **Add integration test for full lifecycle: register → export → verify** | Medium      | 2hr    | End-to-end confidence                           |
| 23  | **Document the module dependency graph in README or docs/**             | Low         | 30min  | Architecture clarity                            |
| 24  | **Review pebble/ for production readiness gaps**                        | Medium      | 2hr    | Embedded store needs scrutiny                   |
| 25  | **Plan v3.0 milestone: what breaks, what improves**                     | Low         | 1hr    | Strategic planning                              |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Are the broken examples (`example/catalog-server/main.go` and `example/user/catalog.go`) intended to be updated to use the new phantom types, or should they be deleted as stale demos that don't reflect current best practices?**

The compilation errors come from the phantom type introduction (commit 4a542363). The examples pass raw `string` where branded types (`catalog.Name`, `catalog.Summary`, `catalog.Protocol`) are now required. I can fix them trivially, but I don't know if the examples themselves are still the "right" way to demonstrate the library, or if they should be rewritten to showcase newer patterns (reactive streams, phantom types, cattest helpers, etc.).

---

## Uncommitted Changes

```
 catalog/exporter.go            | 6 +++---
  1 file changed, 3 insertions(+), 3 deletions(-)
```

**Change**: `ErrorExporter` collapsed to type alias with `// Deprecated:` annotation.
