# Project Groups Improvement Report — go-cqrs-lite

**Date:** 2026-05-18  
**Scope:** Multi-module Go monorepo (`core`, `memory`, `storage`, `projection`, `catalog`, `middleware`, `testhelpers`, `sync`, `integration`)  
**Status:** Critical fixes applied, architectural recommendations documented

---

## 1. Critical Fixes Applied

### 1.1 DAG Violation: `storage` → `memory` (FIXED)

**Problem:** `storage/pebble_config.go` imported `memory` in production code, violating ADR-0003's rule that infrastructure modules depend only on `core`.

```go
// BEFORE (storage/pebble_config.go:66)
case PebbleBackendMemory:
    return memory.NewMemoryStore(), nil  // ❌ production dep on memory
```

**Fix:** Made `PebbleBackendMemory` return `ErrPebbleProviderRequired`, mirroring `PebbleBackendPebble`. Users must provide a `PebbleEventStoreProvider` explicitly.

```go
// AFTER
case PebbleBackendMemory:
    return nil, fmt.Errorf("%w: use WithPebbleProvider", ErrPebbleProviderRequired)
```

**Impact:**
- Removed `memory` from `storage/go.mod` require block
- Removed `memory` replace directive from `storage/go.mod`
- Cleaned 1 transitive dependency chain
- Verified: `go build ./...`, `go test ./...`, `nix run .#test` all pass

---

### 1.2 File Size Limits: `storage/helpers.go` (FIXED)

**Problem:** `storage/helpers.go` at 433 lines — only production file exceeding 250-line limit.

**Fix:** Split into two files by concern:

| File | Lines | Responsibility |
|------|-------|----------------|
| `storage/helpers.go` | 239 | Event scanning, reconstruction, metadata marshaling, event insertion, outbox transaction orchestration |
| `storage/sql_helpers.go` | 205 | SQL-agnostic shared helpers: deleteByAggregate, sharedInsertEvents, sharedCheckVersion, checkpoint CRUD, outbox ack |

**Impact:** Both files under 250 lines. No API changes. All tests pass.

---

### 1.3 File Size Limits: `catalog/asyncapi/exporter.go` (FIXED)

**Problem:** `catalog/asyncapi/exporter.go` at 258 lines.

**Fix:** Split into:

| File | Lines | Responsibility |
|------|-------|----------------|
| `catalog/asyncapi/exporter.go` | 79 | Exporter struct, options, constructor, constants |
| `catalog/asyncapi/builder.go` | 182 | Export logic, message/channel/operation building, schema generation |

**Impact:** Both files under 250 lines. No API changes. All tests pass.

---

## 2. Architectural Recommendations

### 2.1 Eliminate `replace` Directives from Sub-Module go.mod Files

**Current State:** Every sub-module has `replace` directives pointing to sibling modules. This is redundant because `go.work` already handles local resolution.

| Module | Replace Directives | Should Have? |
|--------|-------------------|--------------|
| `core` | 0 | ✅ |
| `memory` | core, testhelpers | ❌ (go.work handles it) |
| `storage` | core | ❌ (go.work handles it) |
| `projection` | core, memory, testhelpers | ❌ (go.work handles it) |
| `catalog` | core | ❌ (go.work handles it) |
| `middleware` | core, testhelpers | ❌ (go.work handles it) |
| `integration` | core, memory, middleware, projection, storage, testhelpers | ❌ (go.work handles it) |

**Why This Matters:**
1. **Consumer confusion:** Published modules don't use replace directives. If `go.work` is missing, `go mod tidy` in a consumer repo resolves versions from the proxy — but replace directives in the repo's own go.mod files create a false sense of how dependencies resolve.
2. **CI drift:** `nix run .#test` uses `go.work`, but `cd module && go test ./...` uses replace directives. These can diverge silently.
3. **Go ecosystem convention:** `go.work` is the modern, clean way to handle multi-module repos. Replace directives are a legacy pattern for pre-workspace Go.

**Migration Strategy:**
1. Remove all `replace` blocks from sub-module go.mod files
2. Ensure `go.work` lists all modules (already done)
3. Add CI step that tests `GOWORK=off go test ./...` in each module to verify versioned imports resolve correctly
4. Document: "For local development, use `go.work`. For publishing, ensure versions are tagged."

---

### 2.2 Extract `core/pkg/dispatcher` and `core/pkg/id` into Standalone Modules

**Current State:** `core/pkg/dispatcher` (3 files) and `core/pkg/id` (13 files) are sub-packages of `core`. They have zero dependencies on other `core` packages and are imported by `memory`, `storage`, `projection`, `catalog`, `middleware`, `testhelpers`.

**Problem:** Every module that needs `id.Of[T]` or the dispatcher pattern pulls in the entire `core` module (including event system, command/query dispatchers, aggregate logic).

**Proposed Structure:**

```
pkg/
├── id/
│   ├── go.mod  # github.com/larsartmann/go-cqrs-lite/pkg/id
│   └── ...
└── dispatcher/
    ├── go.mod  # github.com/larsartmann/go-cqrs-lite/pkg/dispatcher
    └── ...
```

**Dependency Impact:**

| Module | Currently Depends On | Would Depend On |
|--------|---------------------|-----------------|
| `memory` | `core` | `pkg/id` |
| `storage` | `core` | `pkg/id` |
| `projection` | `core` | `pkg/id` |
| `catalog` | `core` | `pkg/id`, `pkg/dispatcher` |
| `middleware` | `core` | `pkg/dispatcher` |
| `testhelpers` | `core` | `pkg/id` |

**Benefits:**
- `core` becomes a pure domain module (events, commands, queries, aggregates, deciders)
- `pkg/id` can reach v1.0 independently (it's stable)
- Consumers who only need branded IDs don't pull in event sourcing machinery

---

### 2.3 Split `core/event/` God Package

**Current State:** `core/event/` has 33 files, ~5,500 lines. It contains:

| Concern | Files | Lines |
|---------|-------|-------|
| Event model + builder | `event.go`, `builder.go`, `types.go`, `options.go` | ~500 |
| Event bus | `bus.go` | ~42 |
| Event store interface | `store.go` | ~82 |
| Snapshot | `snapshot.go`, `snapshot_strategy.go`, `snapshot_helper.go` | ~130 |
| Outbox | `outbox.go`, `outbox_publisher.go` | ~250 |
| Projection | `projection.go`, `runner.go` | ~270 |
| Catalog/metadata | `catalog.go`, `enricher.go` | ~110 |
| Codec | `codec.go` | ~52 |
| Upcaster | `upcaster.go`, `upcaster_registry.go` | ~130 |
| Error taxonomy | `errors.go` | ~114 |
| Checkpoint | `checkpoint.go` | ~24 |
| Publish helpers | `publish_helper.go` | ~56 |

**Problem:** This is a classic god-package. The interface surface is enormous. Tests for one concern (e.g., snapshots) must compile the entire package including outbox publisher logic.

**Proposed Split (within `core` module):**

```
core/
├── event/          # Event model, builder, types, options, metadata (thin)
├── bus/            # Bus interface + in-memory implementation (moved from memory/)
├── store/          # Store interface, snapshot, checkpoint, codec
├── outbox/         # Outbox pattern, outbox publisher
├── projection/     # Projection interface, runner (already exists at root)
└── catalog/        # Catalog metadata, enricher
```

**Note:** This is a package-level split *within* `core`, not a module split. The `core` module keeps all these packages. The benefit is locality: changing the outbox publisher doesn't require recompiling event builder tests.

---

### 2.4 Consolidate `CatalogMeta` Duplication

**Current State:** `CatalogMeta` is defined identically in `core/event/catalog.go`, `core/command/catalog.go`, and `core/query/catalog.go` (3 fields: `Name`, `Summary`, `Description`). `event.CatalogMeta` adds an `AggregateType` field.

**Options:**
1. **Extract to `core/catalogmeta` package** — single definition, imported by event/command/query
2. **Keep per-package** — intentional duplication for package autonomy
3. **Embed a base type** — `catalog.BaseMeta` with `Name/Summary/Description`, each package embeds and adds fields

**Recommendation:** Option 3 (embed base type). It eliminates duplication while preserving per-package extension:

```go
// core/catalogmeta/meta.go
type Base struct {
    Name        string
    Summary     string
    Description string
}

// core/event/catalog.go
type CatalogMeta struct {
    catalogmeta.Base
    AggregateType AggregateType
}
```

---

### 2.5 Versioning Strategy Clarification

**Current State:** Modules use inconsistent versions:

| Module | Version in go.mod | Published? |
|--------|------------------|------------|
| `core` | v1.1.0 | Yes |
| `memory` | v1.1.0 | Yes |
| `testhelpers` | v1.1.0 | Yes |
| `storage` | v1.1.0 | Yes |
| `projection` | v1.1.0 | Yes |
| `middleware` | v0.0.0-00010101000000-000000000000 | No (pseudo) |
| `catalog` | v0.0.0 | No |
| `integration` | Mixed (some v1.1.0, some pseudo) | No |

**Problem:** Pseudo-versions (`v0.0.0-00010101000000-000000000000`) are placeholders that break `GOWORK=off` builds. The `integration` module mixes versions.

**Recommendation:**
1. Tag all modules at `v0.1.0-alpha` simultaneously (shared version strategy)
2. Update all go.mod files to use consistent versions
3. Document: "go-cqrs-lite uses shared versioning — all modules bump together"
4. Add CI check: `GOWORK=off go mod tidy` in each module must resolve without errors

---

### 2.6 CI/CD Parallelization

**Current State:** CI runs sequentially: format → build → vet → test → test-race → lint → coverage.

**Problem:** No per-module parallelization. A lint error in `catalog` blocks tests for `core`.

**Recommendation:** Split CI into a matrix:

```yaml
strategy:
  matrix:
    module: [core, memory, storage, projection, catalog, middleware, sync]
steps:
  - run: cd ${{ matrix.module }} && go build ./...
  - run: cd ${{ matrix.module }} && go vet ./...
  - run: cd ${{ matrix.module }} && go test -race -count=1 ./...
```

Plus a final "integration" job that runs `nix run .#test` (workspace-level) after all modules pass.

---

### 2.7 `integration/` Module Boundaries

**Current State:** `integration/` imports `core`, `memory`, `middleware`, `projection`, `storage`, `testhelpers`. It has no production code — only tests.

**Problem:** It's a test-only module with a heavy dependency graph. The `integration/go.mod` pulls in Pebble, SQLite, Turso, OpenTelemetry — all for BDD tests.

**Options:**
1. **Keep as-is** — test-only modules are valid in Go workspaces
2. **Merge into individual modules** — move `integration/aggregate/` tests into `core/aggregate/`, etc.
3. **Split by concern** — `integration-core/`, `integration-storage/`, etc.

**Recommendation:** Option 1 for now, but add a CI optimization: skip `integration/` in PR builds unless `core/`, `storage/`, or `projection/` changed. Run full integration only on `master`.

---

## 3. Dependency Graph (After Fixes)

```
                    ┌─────────────┐
                    │     sync    │  ← standalone (no deps)
                    └─────────────┘
                          │
                    ┌─────▼─────┐
                    │  pkg/id   │  ← proposed extraction
                    └─────┬─────┘
                          │
              ┌───────────┼───────────┐
              │           │           │
        ┌─────▼─────┐ ┌──▼────┐ ┌────▼────┐
        │   core    │ │memory │ │testhelpers│
        │ (domain)  │ │(impl) │ │ (fakes)  │
        └─────┬─────┘ └───┬───┘ └────┬────┘
              │           │          │
        ┌─────▼─────┐ ┌──▼────┐     │
        │  storage  │ │projection│   │
        │(SQL/Pebble)│ └─────────┘   │
        └───────────┘               │
              │                      │
        ┌─────▼─────┐               │
        │middleware │ ←─────────────┘
        └───────────┘
              │
        ┌─────▼─────┐
        │  catalog  │
        │(doc gen)  │
        └───────────┘
              │
        ┌─────▼─────┐
        │integration│  ← test-only
        └───────────┘
```

**Verified DAG:** No cycles. All arrows point downward. `core` has zero internal dependencies.

---

## 4. Metrics

| Metric | Before | After |
|--------|--------|-------|
| Files >250 lines (production) | 2 | 0 |
| DAG violations | 1 (`storage`→`memory`) | 0 |
| Modules with replace directives | 6 | 6 (recommendation: 0) |
| Test suite pass | ✅ | ✅ |
| Build pass | ✅ | ✅ |
| Lint pass | ✅ | ✅ |

---

## 5. Next Steps (Prioritized)

### High Impact (1% → 51%)
1. **Remove replace directives** from all sub-module go.mod files → rely on `go.work`
2. **Extract `pkg/id`** to standalone module → reduce `core` dependency footprint
3. **Extract `pkg/dispatcher`** to standalone module → reduce `core` dependency footprint
4. **Tag v0.1.0-alpha** across all modules → enable `GOWORK=off` builds

### Medium Impact (4% → 64%)
5. **Split `core/event/` god-package** into sub-packages (bus, store, outbox, projection)
6. **Consolidate `CatalogMeta`** with embedded base type
7. **Parallelize CI** per module
8. **Add `GOWORK=off` CI job** to verify versioned imports

### Lower Impact (20% → 80%)
9. **Evaluate `integration/` test-only status** — optimize CI to skip on unrelated changes
10. **Document module boundaries** in AGENTS.md → update dependency rules
11. **Add module-level READMEs** → each module explains its public API

---

*Report generated by architectural analysis of go-cqrs-lite multi-module structure.*
