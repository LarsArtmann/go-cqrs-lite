# ADR-0075: ADT test harness extraction (metaengine/adttest)

|             |                                                                              |
| ----------- | --------------------------------------------------------------------------- |
| **Status**  | Accepted                                                                     |
| **Date**    | 2026-07-31                                                                   |
| **Context** | The 7-ADT cross-engine parity test harness needed to be reusable by engines in separate modules |

## Context

The metaengine defines 7 abstract data types (Map, Set, Counter, Graph, Scan,
Log, Multimap) and multiple engine implementations (`memoryEngine`,
`sqliteEngine`, `pebbleEngine`). As the number of engines grew, ensuring they
all produce identical results for the same operations became critical — a
silent divergence between engines would undermine the cost-based planner's
promise that the cheapest engine is also correct.

The parity tests were initially inline in the `metaengine` package's test
files. This created two problems:

1. **Inaccessibility**: The `pebbleengine` module (separate `go.mod` per
   ADR-0062) could not run the parity tests because they were unexported test
   helpers in the `metaengine` package.
2. **Duplication risk**: Without a shared harness, each engine module would
   copy-paste the test scenarios, leading to drift.

## Decision

Extract the test harness into `metaengine/adttest/` as an **exported
sub-package** with:

- `Factory` struct: `{ Name, Create, Supports }` — pluggable engine factory
- `RunMatrix(t, factories)`: runs all 7 ADT scenarios across all factories,
  asserting cross-engine parity via canonical-string comparison
- `Scenarios()`: returns the 7 scenario definitions, each declaring a required
  backend interface (auto-skipped if an engine doesn't implement it)

The harness is a **test-only package** (consumers add it to `_test.go` files).
It uses `reflect.TypeOf().Implements()` to auto-skip scenarios for engines
that don't implement the required backend interface.

Engine modules import `adttest` and pass their factory:

```go
func TestPebbleADTMatrix(t *testing.T) {
    adttest.RunMatrix(t, []adttest.Factory{
        {Name: "memory", Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() }},
        {Name: "pebble", Create: func(t *testing.T) metaengine.Engine { ... }},
    })
}
```

## Consequences

- **Transitivity**: memory↔sqlite parity is tested in `metaengine/adt_matrix_test.go`;
  memory↔pebble parity is tested in `metaengine/pebbleengine/adt_matrix_test.go`.
  By transitivity, all three engines are verified to agree.
- **No import cycle**: `adttest` imports `metaengine` but not the engine modules.
  Engine modules import `adttest` — a clean DAG.
- **Extensibility**: new engines (Postgres, DuckDB) add their factory to their
  module's `adt_matrix_test.go` and get full parity testing for free.
