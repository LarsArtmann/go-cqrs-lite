# Module Quality Assessment — go-cqrs-lite

**Date:** 2026-05-21 | **Mode:** READ, UNDERSTAND, RESEARCH, REFLECT

## Overall Verdict

The module structure is **GOOD but not SUPERB**. The DAG is clean, core is properly isolated, and the dependency direction is correct. However, there are 7 concrete issues preventing a "superb" rating — ranging from hygiene problems (replace/go.work conflict) to architectural concerns (god-package in `core/event`) to policy violations (banned libraries).

---

## What's Excellent ✅

### 1. Clean DAG — Zero Circular Dependencies

Production dependency graph is a perfect DAG:

```
core ← memory       ← example/todo, example/user, integration
core ← catalog      ← example/user, integration
core ← middleware    ← example/user, integration
core ← storage      ← example/todo, integration
core ← testhelpers  ← memory, middleware, projection, integration
core ← projection   ← integration
sync  (standalone leaf)
```

**No upward dependencies. No lateral dependencies between infra modules. Core imports nothing.**

### 2. Core Isolation

`core` has zero internal module dependencies in production code. It's independently buildable (`GOWORK=off go build ./...` succeeds). This is the most critical quality gate and it passes.

### 3. Sync Module

`sync/` is a standalone leaf with zero dependencies. Perfect isolation for a domain-primitive module.

### 4. Interface-First Design

All key types are interfaces (`Store`, `Bus`, `Publisher`, `Subscriber`, `SnapshotStore`, `Outbox`, `CheckpointStore`). Infrastructure modules provide implementations. This is textbook Go library design.

### 5. Test Coverage

21 test packages, >90% total coverage. Every module has substantive tests.

### 6. File Size Discipline

Only 1 production file exceeds 250 lines (`storage/pebble_event_store.go` at 266 lines).

---

## Issues Found 🔴🟡

### 🔴 Critical: Replace Directives + go.work Conflict (7 modules)

The go-modularize skill says: *"Never mix replace directives AND go.work for the same module pair."*

7 of 12 modules have `replace` directives in their `go.mod` AND are listed in `go.work`:

| Module | Replace Count | Targets |
|--------|:---:|---------|
| `memory` | 2 | core, testhelpers |
| `catalog` | 1 | core |
| `middleware` | 2 | core, testhelpers |
| `projection` | 3 | core, memory, testhelpers |
| `integration` | 6 | core, memory, middleware, projection, storage, testhelpers |
| `example/todo` | 4 | core, memory, storage, testhelpers |
| `example/user` | 4 | catalog, core, memory, middleware |

**Why this matters:** With `go.work` active, `replace` directives are redundant noise. Without `go.work` (consumers), `replace` directives break because they reference local paths. The project should pick ONE strategy. Since `go.work` already exists and handles local development, the `replace` directives should be removed.

**Exception:** `storage` has NO replace directives and CANNOT build without the workspace. It needs replace directives added (or the version bumped and published).

### 🔴 Critical: Storage Can't Build Independently

`storage/go.mod` requires `core v1.3.0` with no replace directive. `GOWORK=off go build ./...` fails:

```
missing go.sum entry for module providing package github.com/larsartmann/go-cqrs-lite/core/event
```

Every other infrastructure module has replace directives for local development. Storage is the sole outlier.

### 🔴 Critical: core/testhelpers Circular Module Dependency

`core/go.mod` requires `testhelpers v1.1.0`, and `testhelpers/go.mod` requires `core v1.1.0`. This creates a circular module dependency:

```
core → testhelpers → core  (chicken-and-egg for publishing)
```

It works because the imports are only in `*_test.go` files within core. But `go mod tidy` with `GOWORK=off` would fail if neither is published yet.

### 🟡 Medium: Version Inconsistency Across Modules

Three different version numbers for `core`:

| Version | Used by |
|---------|---------|
| `v0.0.0` | catalog |
| `v1.1.0` | memory, middleware, projection, testhelpers, example/user |
| `v1.3.0` | storage, integration, example/todo |

Similar inconsistency for other modules (`middleware` is `v0.0.0` in example/user but `v0.0.0-00010101000000-000000000000` in integration).

### 🟡 Medium: God-Package in `core/event` (23 files, ~90 exports)

`core/event` contains 8+ distinct responsibility clusters that could be sub-packages:

| Cluster | Files | Suggested Package |
|---------|:-----:|--------------------|
| Event core | 5 | `event` (keep) |
| Codec/serialization | 2 | `event/codec` |
| Bus interfaces | 1 | `event` (keep interfaces) |
| Projection | 2 | `event/projection` or consolidate with `projection/` |
| Snapshot | 3 | `event/snapshot` |
| Outbox | 2 | `event/outbox` |
| Upcasting | 2 | `event/upcaster` |
| Catalog (deprecated) | 1 | Remove |

**Counter-argument:** This is a Go package, not a Java class. Having related types in one package is idiomatic. Splitting into sub-packages increases import complexity for consumers. The current structure works well for navigation. **This is a judgment call, not a clear violation.**

### 🟡 Medium: Overlapping Runner Logic

`core/event/runner.go` (`InMemoryRunner`, 217 lines) and `projection/runner.go` (`Runner`, 204 lines) both implement replay+subscribe patterns. The relationship between them is unclear:
- `InMemoryRunner` uses `MemoryStore` directly for replay
- `Runner` uses `event.GlobalLoader` interface for replay
- Both subscribe to `event.Bus` for live events

Are these two different abstractions for two different use cases, or a DRY violation?

### 🟡 Low: Dead Replace Directive in example/todo

`example/todo/go.mod` has `replace testhelpers => ../../testhelpers` but no file in example/todo imports testhelpers. The replace directive is dead code.

### 🟡 Low: Banned Library (testify) in example/todo

`example/todo` uses `stretchr/testify` in 4 test files, which is banned per the how-to-golang skill. Should use `ginkgo/v2` + `gomega` instead for consistency with the rest of the project.

### 🟡 Low: Root go.mod Is Dead Weight

`/go.mod` declares `module github.com/larsartmann/go-cqrs-lite` with zero requires and zero `.go` files. It's a module path claim with no build function. Either:
1. Delete it (go.work is the workspace coordinator)
2. Add a `doc.go` with re-exports if the root path should be importable

### 🟢 Info: pebble_event_store.go Exceeds 250 Lines

At 266 lines, this is the only production file exceeding the project's 250-line limit. Already has helpers extracted to `pebble_helpers.go` (176 lines) and `pebble_serialization.go` (145 lines). Low priority.

---

## Module-by-Module Report Card

| Module | DAG | Independent Build | go.mod Hygiene | Replace Strategy | Verdict |
|--------|:---:|:-----------------:|:--------------:|:----------------:|---------|
| **core** | ✅ Leaf | ✅ Yes | 🟡 Circular w/ testhelpers | ✅ None | Good |
| **memory** | ✅ → core | ✅ Yes | ✅ Clean | 🔴 Has replace | Good* |
| **catalog** | ✅ → core | ✅ Yes | ✅ Clean | 🔴 Has replace | Good* |
| **middleware** | ✅ → core | ✅ Yes | ✅ Clean | 🔴 Has replace | Good* |
| **testhelpers** | ✅ → core | ✅ Yes | 🟡 Circular w/ core | ✅ None | Good |
| **projection** | ✅ → core | ✅ Yes | ✅ Clean | 🔴 Has replace | Good* |
| **storage** | ✅ → core | 🔴 **No** | ✅ Clean | 🟡 **Missing** replace | Needs Fix |
| **sync** | ✅ Leaf | ✅ Yes | ✅ Clean | ✅ None | **Superb** |
| **integration** | ✅ → many | N/A (test-only) | 🟡 Version chaos | 🔴 Has replace | Adequate |
| **example/todo** | ✅ → core, memory, storage | ✅ Yes | 🟡 Dead replace, banned lib | 🔴 Has replace | Adequate |
| **example/user** | ✅ → core, memory, middleware, catalog | ✅ Yes | ✅ Clean | 🔴 Has replace | Good* |

\* = Would be "Superb" after removing replace directives

---

## Prioritized Action Plan (Pareto)

### Tier 1: 1% → 51% Impact (Foundational Fixes)

| # | Issue | Effort | Impact |
|---|-------|--------|--------|
| 1 | Remove all `replace` directives from go.mod files (go.work handles this) | 30 min | Eliminates replace/go.work conflict, simplifies all 7 modules |
| 2 | Add `replace` to `storage/go.mod` for core (or remove version pin) | 5 min | Fixes independent build failure |
| 3 | Normalize versions across all go.mod files | 15 min | Eliminates version chaos |

### Tier 2: 4% → 64% Impact (Hygiene)

| # | Issue | Effort | Impact |
|---|-------|--------|--------|
| 4 | Remove dead testhelpers replace from example/todo | 2 min | Dead code removal |
| 5 | Move core's test-only deps (memory, testhelpers) to a test-only pattern | 30 min | Breaks circular module dependency |
| 6 | Remove root go.mod (dead weight) or add doc.go | 5 min | Clean project root |

### Tier 3: 20% → 80% Impact (Policy Compliance)

| # | Issue | Effort | Impact |
|---|-------|--------|--------|
| 7 | Migrate example/todo from testify to ginkgo/gomega | 30 min | Banned library removal |

### Tier 4: Polish (Optional, Judgment Call)

| # | Issue | Effort | Impact |
|---|-------|--------|--------|
| 8 | Split core/event god-package into sub-packages | 2-3 hours | Navigation, but increases import complexity |
| 9 | Clarify InMemoryRunner vs projection.Runner relationship | 1 hour | Architecture clarity |
| 10 | Split pebble_event_store.go (266→<250 lines) | 15 min | File size compliance |

---

## Questions for Reflection

1. **Should `replace` directives be removed?** The go-modularize skill recommends go.work-only. But replace directives enable `GOWORK=off` builds for modules that reference unpublished versions. If modules are published to the Go proxy, replace is unnecessary. If not, replace is needed for independent builds. **Decision needed: Is this project published or purely local?**

2. **Should `core/event` be split?** 23 files in one package is large, but Go favors flat packages. The sub-clusters are tightly coupled (all use `Event`, `Type`, `Version`). Splitting would require consumers to import 5+ sub-packages. **Recommendation: Keep as-is but document the cluster map.**

3. **What's the versioning strategy?** No ADR or documentation specifies whether modules share versions, have independent semver, or use root-only versioning. This should be decided and documented.

4. **Is `core → testhelpers` circular dependency acceptable?** It only exists in test files, but it prevents publishing either module independently. Options:
   - Inline test helpers into `core/internal/testutil`
   - Create a separate `core_test` module
   - Accept the circular dependency (test-only, works in practice)
