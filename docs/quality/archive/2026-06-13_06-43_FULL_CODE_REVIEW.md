# Full Code Review — go-cqrs-lite 2026-06-13

## Executive Summary

- **22 library modules** reviewed via automated analysis
- **Build:** PASS (all 27 modules)
- **Tests:** 26/27 pass (1 golden file fix applied)
- **Lint:** 2 issues in catalog (goconst, nolintlint)
- **Clones:** 8 groups (dupl ≥80 tokens)
- **Architecture:** Excellent — clean DAG, ISP splits, library-first design

## Pareto Analysis

### 1% → 51% Impact

| Task                                                       | Effort | Impact                       |
| ---------------------------------------------------------- | ------ | ---------------------------- |
| Fix 2 catalog lint issues                                  | 10min  | Zero lint across all modules |
| Fix `gopls mapsloop` hint in encryption/static_resolver.go | 5min   | Cleaner code                 |
| Fix nil context warnings in encryption tests               | 10min  | Zero gopls warnings          |

### 4% → 64% Impact

| Task                                                           | Effort | Impact                   |
| -------------------------------------------------------------- | ------ | ------------------------ |
| Extract shared base64 decode helper (signing/encryption dedup) | 30min  | Eliminate 2 clone groups |
| Update README.md with encryption/turso module sections         | 30min  | Complete documentation   |
| Add `pkg/` to go.work or remove dead code                      | 15min  | Clean module inventory   |

### 20% → 80% Impact

| Task                                        | Effort | Impact                     |
| ------------------------------------------- | ------ | -------------------------- |
| Reactive Bus Bridge (imperative ↔ reactive) | 2hr    | Unified event API          |
| Parameterize SQL load helpers in storage/   | 1hr    | Eliminate 2 clone groups   |
| Add BDD tests for listing, storage/sql      | 2hr    | Comprehensive BDD coverage |
| Add testutil/ to AGENTS.md module list      | 15min  | Accurate documentation     |

## Module-by-Module Quality Assessment

### Production Modules (✅ = Clean, ⚠️ = Minor Issues)

| Module      | Lines | Coverage | Lint | Clone   | BDD | Verdict      |
| ----------- | ----- | -------- | ---- | ------- | --- | ------------ |
| event/      | ~2800 | >90%     | ✅   | 0       | ✅  | ✅ Excellent |
| command/    | ~300  | >85%     | ✅   | 1 (low) | ✅  | ✅ Good      |
| query/      | ~250  | >85%     | ✅   | 0       | ✅  | ✅ Good      |
| decider/    | ~400  | >90%     | ✅   | 0       | ✅  | ✅ Excellent |
| id/         | ~600  | >95%     | ✅   | 0       | ✅  | ✅ Excellent |
| dispatcher/ | ~200  | >90%     | ✅   | 0       | ✅  | ✅ Excellent |
| schema/     | ~250  | >85%     | ✅   | 0       | ✅  | ✅ Good      |
| snapshot/   | ~250  | >85%     | ✅   | 0       | ✅  | ✅ Good      |
| codec/      | ~350  | >90%     | ✅   | 0       | ✅  | ✅ Excellent |
| memory/     | ~800  | >85%     | ✅   | 0       | ✅  | ✅ Good      |
| catalog/    | ~2500 | >80%     | ⚠️ 2 | 1 (low) | ✅  | ⚠️ Good      |
| middleware/ | ~1200 | >85%     | ✅   | 1 (low) | ✅  | ✅ Good      |
| signing/    | ~800  | >90%     | ✅   | 2 (med) | ✅  | ✅ Good      |
| encryption/ | ~600  | >90%     | ✅   | 2 (med) | ✅  | ✅ Good      |
| storage/    | ~1500 | >89%     | ✅   | 2 (low) | ✅  | ✅ Good      |
| projection/ | ~600  | >85%     | ✅   | 0       | ✅  | ✅ Good      |
| otel/       | ~300  | >97%     | ✅   | 0       | ✅  | ✅ Excellent |
| watermill/  | ~400  | >85%     | ✅   | 0       | ✅  | ✅ Good      |
| pebble/     | ~500  | >85%     | ✅   | 0       | ✅  | ✅ Good      |
| turso/      | ~300  | >80%     | ✅   | 0       | ✅  | ✅ Good      |
| listing/    | ~400  | >85%     | ✅   | 0       | ✅  | ✅ Good      |

## Type Safety Review

### Strengths

- Branded IDs via phantom types (`id.Of[T]`) — impossible to mix up AggregateID/EventID/CommandID
- `Version`, `SchemaVersion` are phantom-typed ints — no accidental comparison
- `AggregateRef` pairs type + ID — cannot forget aggregate type
- `TombstoneStatus` is a three-state enum — no boolean ambiguity
- `Store = EventSink + EventSource` — ISP enforced at type level

### Issues

- `any` used in `event.Source` and query dispatch return types — unavoidable for Go generics
- `storage/sql/dialect.go` uses `any` for `database/sql` interop — documented exception
- No compile-time enforcement that `event.New()` payload is serializable — runtime error only

## Split Brain Check

| Concept      | Locations                                                            | Consistent?                              |
| ------------ | -------------------------------------------------------------------- | ---------------------------------------- |
| Metadata     | event.Metadata, command.Metadata (alias)                             | ✅ Alias, not duplicate                  |
| AggregateRef | event/, command/                                                     | ✅ Separate types, same concept          |
| Handler      | event.Handler, command.Handler, query.Handler, middleware.Handler[M] | ✅ Package-scoped, different signatures  |
| Dispatcher   | command.Dispatcher, query.Dispatcher, dispatcher.Dispatcher[H,M]     | ✅ Generic vs specific, correct layering |

## Critical Findings

1. **catalog lint issues** — `Cmd` string repeated 3x, nolint directive doesn't suppress goconst
2. **encryption/static_resolver.go** — `gopls mapsloop` hint (use `maps.Copy`)
3. **pkg/ directory** — config and gracefulshutdown exist but aren't in go.work
4. **README.md** — Missing encryption and turso module sections

## Architectural Quality: 9/10

This is an exceptionally well-structured Go library. The module boundaries are clean, the dependency graph is a proper DAG, ISP splits are textbook, and composability is first-class. The issues found are minor — lint cleanup, documentation gaps, and low-priority dedup opportunities.
