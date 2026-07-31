# Dedup Acceptance Log

**Session 4 (-t 3):** Eliminated ALL harmful duplication — clone groups
reduced from 8 to 2 (both intentionally accepted). Extractions:
`ruletest.RunDetector`/`AssertRule` (10 copies → 1 shared package),
table-driven conversions (s005, c030, S006 tests), `newMemoryTestStore`/
`newSQLiteTestStore` (13 metaengine test setups), `setupRawScanTest`
(6 pebbleengine test setups). Only 2 groups remain: cross-module golden
helpers and trivial strings.Builder idiom.

**Session 3 (-t 3):** 12 extractions applied across metaengine + cqrs-lint,
reducing clone groups from 19 to 8. All production-code clones eliminated;
the remaining 8 are idiomatic test patterns (table-driven tests, per-package
test helpers) or false positives. See "Session 3 Extractions" below.

**Session 2 (-t 2):** 2 extractions applied (`parseTimePointer`, `iterJSON`),
reducing clone groups from 52 to 50 (test boilerplate unchanged).
Production-code patterns at the 3+ threshold are now exhausted; the
remaining 50 groups are all 2-occurrence test idioms or unique-value sites.

Remaining clone groups from `art-dupl --type-aware -t 2` after two dedup
sessions. Each entry explains why the clone is intentional and should not be
extracted.

**Session 1 (-t 3):** 3 groups accepted (below).
**Session 2 (-t 2):** 7 extractions applied, 53 groups remaining (all accepted below).

---

## Extractions Applied (Session 2)

| Extraction                    | Files                                    | Pattern Eliminated                                |
| ----------------------------- | ---------------------------------------- | ------------------------------------------------- |
| `Bundle.readModelCodec`       | stack/accessors.go                       | ReadModels nil-check + codec default (2 sites)    |
| `lintutil.AppendBuild`        | cmd/cqrs-lint (5 files + new pkg)        | Build-error guard + append (5 sites)              |
| `errContainsAny`              | storage/turso/indexing (2 files)         | err-nil guard + lowercased Contains (2 sites)     |
| `withOutput`                  | cmd/cqrs-bench/output.go                 | openOutput + defer closeOutput (4 sites)          |
| `wrapInfraOrOK`               | storage/turso/errors.go + sync.go        | if-err-nil return WrapInfra; return nil (3 sites) |
| `wrapInfraBytes`              | encryption/errors.go + cose.go + hkdf.go | if-err-nil return nil, WrapInfra (3 sites)        |
| `unmarshalJSONString`         | event/date.go + time_types.go            | json.Unmarshal + WrapRejection (2 sites)          |
| `sliceIteratorOrErr`          | storage/memory/stream.go                 | if-err + WrapInfra + SliceIterator (4 sites)      |
| `mergeKnows` + `knowsEdgeRef` | graph/graphtest/contract.go              | RunInTx + MergeEdge + EdgeRef literal (5 sites)   |

---

## Session 3 Extractions (-t 3)

| Extraction                     | Files                                                       | Pattern Eliminated                                                                          |
| ------------------------------ | ----------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| `buildPlannedSelectQuery`      | metaengine/planned_sqlite.go + raw_reader.go                | SQL query builder (SELECT/WHERE/ORDER BY/LIMIT) for planned tables (2 sites)                |
| `scanSingleColumn[T]`          | metaengine/raw_reader.go + sqlite_backends.go               | QueryContext + rows.Scan + transform generic (2 sites: raw bytes + JSON decode)             |
| `Watcher.addWatcherEntry`      | metaengine/dx.go                                            | watcherEntry creation + lock + register (2 sites: Watch + WatchWithSeq)                     |
| `pebbleEngine.getPebbleRaw`    | metaengine/pebbleengine/engine.go + raw_reader.go           | db.Get + ErrNotFound + closer.Close (2 sites: MapGet + GetRawValue)                         |
| `lintutil.ModuleImportsPath`   | cmd/cqrs-lint security/s005.go + s006.go                    | Package import path search (2 sites: signing + encryption)                                  |
| `lintutil.FirstFilePos`        | cmd/cqrs-lint adoption/helpers.go + architecture/helpers.go | First non-test file package position (2 sites, now one-line delegates)                      |
| `lintutil.SelectorMatches`     | cmd/cqrs-lint lintutil.go + d012.go + c032.go               | SelectorExpr pkg.name check (3 sites: fmt.Errorf, context.Context, context.Background/TODO) |
| `lintutil.ExprCallSelector`    | cmd/cqrs-lint api/a002.go + correctness/swallow_helpers.go  | expr → CallExpr → SelectorFromExpr (2 sites: json.Marshal + Payload)                        |
| `lintutil.CallSelectorMatches` | cmd/cqrs-lint correctness/c021.go + c024.go                 | SelectorFromExpr + name check (3 sites: Lock/RLock, Unlock/RUnlock, Begin/BeginTx/RunInTx)  |

### Session 3 Accepted Groups (8 remaining, all test/false-positive)

| Group              | Location                                             | Reason                                                                       |
| ------------------ | ---------------------------------------------------- | ---------------------------------------------------------------------------- |
| Test helpers       | 6 `rules_test.go` files                              | `runDetector`/`assertRule` across 6 rule packages; Go per-package test model |
| Table-driven tests | s005_test, c030_test, new_rules_test, features*_test | `t.Parallel()` + setup + assert; idiomatic Go test structure                 |
| Golden helpers     | catalog/cattest + event/eventtest                    | Per-module `AssertGolden`; go-snaps config is path-relative                  |
| strings.Builder    | observability.go + plan_types.go                     | `var b strings.Builder` + different content; false positive                  |

---

## 1. Stack preset DB open with named-return cleanup

**Files:** `stack/postgres/preset.go:147`, `stack/sqlite/preset.go:122`
**Category:** assignment | 2 occurrences

```
db, err = sqlopt.OpenDBOrErr(driver, dsn, label)
if err != nil {
    return nil, nil, err
}
defer func() {
    if err != nil && db != nil {
        _ = db.Close()
    }
}()
ctx := context.Background()
```

**Reason:** The `defer func() { if err != nil ... }` cleanup is tied to the
enclosing function's named return `err`. Extracting a helper would still leave
the conditional defer in each caller because the defer must capture the named
return from its own scope. The logic diverges immediately after this preamble
(sqlite does WAL/FK/pool config; postgres does schema init). This is the
idiomatic Go resource-lifecycle pattern for "open, then close-on-error".

---

## 2. Decider cache mutex idiom

**Files:** `decider/cache.go:69`, `decider/cache.go:117` (same file, Get + Put/Invalidate)
**Category:** unknown | 2 occurrences

```
c.mu.Lock()
defer c.mu.Unlock()

key := ref.String()
```

**Reason:** Standard Go mutex idiom. The `defer c.mu.Unlock()` must live in
the caller's scope. A closure-based `withLockKey(ref, fn)` would require
captured variables for Get's three return values, making the code worse. The
three lines are the minimal Go locking preamble.

---

## 3. Catalog writer strings.Builder with different content

**Files:** `catalog/eventcatalog/writer.go:102`, `catalog/eventcatalog/writer_schemas_txt.go:15`
**Category:** expression | 2 occurrences

```
var cfg strings.Builder
cfg.WriteString(...)
```

**Reason:** Standard `strings.Builder` usage writing completely different
content (JS config export vs Markdown schemas header). The structural
similarity is just `var X strings.Builder` followed by `X.WriteString(...)`.
The values are intentionally unique; the structure is the standard library
API.

---

## Session 2 Accepted Categories

### Test Setup Boilerplate (9 groups, ~165 clones)

`t.Parallel()` followed by a setup line (`NewTestRegistry()`, `t.TempDir()`,
`id.NewStreamID()`, `NewWithT(t)`, `context.Background()`, `CBORCodec{}`,
`errors.New("...")`, `idtest.ParseStreamID(...)`).

**Rationale:** Idiomatic Go test setup. Each test independently chooses its
setup combination; extracting a helper would take more parameters than lines
saved. Per dedup skill: "Table-driven tests, standard assertions" are
acceptable. Unique values (errors.New messages, ID literals) are parameters.

### Per-Module Error Helpers (Group 10, 3 clones)

`wrapInfraOrOK` / `wrapClosed` in storage/memory, storage/pebble,
storage/readmodel. Identical 5-line body.

**Rationale:** ADR-0069 mandates per-module helpers. Cross-module sharing
would create cyclic deps or a new shared module for 5 lines.

### Span Creation with Unique Names (~12 groups, ~24 clones)

`span := start{Type}Span(ctx, "unique.span.name", ...); defer span.End()`

**Rationale:** Each call uses a unique OTel span name (the operational
identifier). Per dedup skill: "Unique values are parameters, not duplication."

### Error Wrapping with Unique Codes (~15 groups, ~30 clones)

`if err != nil { return errorfamily.WrapX(err, "unique.code", "msg") }`

**Rationale:** The unique error codes ARE the convention. Where 3+ same-
pattern sites exist in one module, per-module helpers were added (see
Extractions). Remaining sites are 2-occurrence pairs across different
modules with unique codes.

### Sibling Preset Parallelism (~5 groups, ~10 clones)

stack/sqlite, stack/postgres, stack/turso share near-identical structure.

**Rationale:** Each preset owns its own error sentinels, doc comments, and
convenience wrappers. They already delegate to `sqlopt`. The parallelism is
intentional API consistency across deployment targets.

### Cross-Module Structural Patterns (3 groups)

- **command.Metadata vs query.Metadata**: type alias per ADR-0031. Each
  module owns its Metadata to prevent event-shape leakage.
- **signing/event.go vs encryption/event.go**: identical 4-line
  `Classify(err) != Rejection` body. The `IsRejection` predicate belongs in
  go-error-family (external); duplicated pending upstream addition.
- **transport/grpc/otel.go vs transport/http/otel.go**: near-identical
  `tracer()` wrapper. 2-line cost of per-module tracer naming.

### 2-Occurrence Guard Clauses (~10 groups, ~20 clones)

Mutex Lock+defer, filter guards, `if i == 0 { return false }`, reportScanErr,
rebuildHandlerChain, circuit_breaker validate+apply, sql rows Close defer.

**Rationale:** Below the 3+ extraction threshold per dedup skill.
