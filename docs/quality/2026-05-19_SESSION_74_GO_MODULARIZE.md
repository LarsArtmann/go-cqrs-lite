# Go Modularization Audit — Session 74

**Date:** 2026-05-19 | **Modules:** 11 | **Workspace:** go.work with all modules

## Current State: ALREADY MODULARIZED (Refinement Needed)

The project is already split into 11 Go modules with a `go.work` workspace. This audit focuses on boundary hygiene, not greenfield modularization.

## Module Landscape

| Module       | Path           | Internal Deps                                              | Replace Directives | State                        |
| ------------ | -------------- | ---------------------------------------------------------- | ------------------ | ---------------------------- |
| sync         | ./sync         | none                                                       | none               | ✅ Clean                     |
| testhelpers  | ./testhelpers  | core                                                       | none               | ✅ Clean                     |
| core         | ./core         | memory, testhelpers                                        | none               | ⚠️ Prod deps on test modules |
| memory       | ./memory       | core, testhelpers                                          | 2 replaces         | ✅ Clean (go.work)           |
| catalog      | ./catalog      | core                                                       | 1 replace          | ⚠️ Should use go.work only   |
| middleware   | ./middleware   | core, testhelpers                                          | 2 replaces         | ✅ Clean (go.work)           |
| storage      | ./storage      | core                                                       | 1 replace          | ⚠️ Should use go.work only   |
| projection   | ./projection   | core, memory, testhelpers                                  | 3 replaces         | ✅ Clean (go.work)           |
| integration  | ./integration  | core, memory, middleware, projection, storage, testhelpers | 5+ replaces        | ⚠️ Complex                   |
| example/user | ./example/user | core, memory, catalog, middleware                          | 4 replaces         | ✅ Clean (demo)              |
| example/todo | ./example/todo | core, memory, storage                                      | 1 replace          | 🔴 Build broken              |

## Issues Found

### 1. Mixed replace + go.work Strategy (MEDIUM)

8 of 11 modules have BOTH `replace` directives in go.mod AND are listed in `go.work`. The Go team recommends one or the other:

- `go.work` for local development (current approach)
- No `replace` directives needed when using `go.work`

When `go.work` is present, Go ignores `replace` directives in go.mod. The replace directives are dead code that:

- Confuse readers about the actual resolution strategy
- Must be maintained in parallel with go.work
- Could cause confusion if go.work is deleted

**Recommendation:** Remove all `replace` directives from modules listed in `go.work`. Keep `go.work` as the single source of truth for local development.

### 2. core Depends on memory + testhelpers in Production go.mod (HIGH)

`core/go.mod` lists `memory v1.1.0` and `testhelpers v1.1.0` as direct requires. These are only used in `_test.go` files. In Go, this means:

- Anyone importing `core` transitively gets `memory` and `testhelpers` as dependencies
- Published `core` v1.1.0 would pull in ginkgo/gomega transitively

**Fix:** Move `memory` and `testhelpers` to a separate `go.mod` test block or use the Go 1.26 test-only dependency pattern. Alternatively, keep them as `// indirect` if Go tooling doesn't support test-only requires.

### 3. Published Version Mismatch (CRITICAL)

| Module      | Published Version | Local State                                              |
| ----------- | ----------------- | -------------------------------------------------------- |
| core        | v1.1.0            | Ahead (event.Version, event.SchemaVersion type changes)  |
| memory      | v1.1.0            | Compatible with current core                             |
| testhelpers | v1.1.0            | **INCOMPATIBLE** — uses `int` instead of `event.Version` |
| catalog     | v0.0.0            | No published version                                     |
| middleware  | v0.0.0            | No published version                                     |
| projection  | v0.0.0            | No published version                                     |
| storage     | v0.0.0            | No published version                                     |
| sync        | N/A               | New, no published version                                |

The `testhelpers v1.1.0` published tag is incompatible with current `core`. When building in isolation (GOWORK=off), `core/event`, `core/aggregate`, and `core/decider` fail to compile because they import the published `testhelpers v1.1.0` which uses bare `int` for version parameters instead of `event.Version`.

**Fix:** Bump `testhelpers` to v1.2.0 and update all consumers. Then bump `core` to v1.2.0.

### 4. example/todo Build Broken (HIGH)

`example/todo` fails to build because:

- References stale `storage` API (missing `event.SchemaVersion`, wrong `SaveWithOutbox` signature)
- References `larsartmann/cqrs-htmx` external dependency

**Fix:** Update `example/todo` to current storage/core API.

### 5. Version Inconsistency (MEDIUM)

Some go.mod files reference `core v1.1.0`, others reference `core v0.0.0`. This will cause resolution issues when publishing:

- `catalog/go.mod`: `core v0.0.0`
- `middleware/go.mod`: `core v1.1.0`
- `storage/go.mod`: `core v1.1.0`

**Fix:** Standardize all references to the same version (or v0.0.0 for development).

## DAG Verification

```
sync ──────────────────────── (zero deps)
testhelpers ──→ core ──────── (core only)
memory ──→ core
catalog ──→ core
middleware ──→ core
storage ──→ core
projection ──→ core + memory (test)
integration ──→ core + memory + middleware + projection + storage
example/user ──→ core + memory + catalog + middleware
example/todo ──→ core + memory + storage
```

**No cycles detected.** The DAG is clean with `core` as the single root.

## Recommendations

| Priority | Action                                              | Effort |
| -------- | --------------------------------------------------- | ------ |
| 🔴 P0    | Bump testhelpers to v1.2.0 (fix version mismatch)   | 1h     |
| 🔴 P0    | Fix example/todo build failures                     | 2h     |
| 🟠 P1    | Remove all replace directives (use go.work only)    | 1h     |
| 🟠 P1    | Move test deps out of core's production go.mod      | 2h     |
| 🟡 P2    | Standardize version references to v0.0.0            | 30min  |
| 🟢 P3    | Consider splitting core into core-types + core-impl | Future |
