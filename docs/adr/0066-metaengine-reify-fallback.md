# ADR-0066: Cross-engine JSON reification in ExecuteTyped

**Date:** 2026-07-25
**Status:** Accepted

## Context

`metaengine.ExecuteTyped[Q, R]` returns a typed result `R`. The memory engine
stores Go values directly, so a direct type assertion `raw.(R)` succeeds. The
SQLite engine round-trips values through JSON (the `database/sql` driver returns
`any`), so a struct value comes back as `map[string]any` — the direct assertion
`raw.(R)` fails for struct results, breaking typed reads on the SQL path.

Without a bridge, `ExecuteTyped` would work on memory and fail on SQLite for the
same query, making engines non-substitutable.

## Decision

When the direct assertion fails, `ExecuteTyped` re-reifies the value via a JSON
marshal/unmarshal into `R` (`reify[R](raw)`). Only if reification also fails does
it return `errExecuteTypeMismatch`. The reify cost is paid **only** on the SQLite
path (memory values pass the direct assertion); memory-path reads are unaffected.

```go
result, ok := raw.(R)
if !ok {
    if reified, rErr := reify[R](raw); rErr == nil {
        return reified, nil
    }
    return zero, fmt.Errorf("%w: got %T", errExecuteTypeMismatch, raw)
}
```

## Consequences

- **+** Engines are substitutable: the same `ExecuteTyped` call works against
  memory and SQLite without caller changes.
- **+** The cost is confined to the SQL read path (JSON marshal+unmarshal per
  struct result). Memory reads stay zero-overhead.
- **−** A struct→incompatible-struct reification (e.g. extra/missing fields) is
  silently "successful" because Go's `json.Unmarshal` ignores unknown fields.
  A truly impossible reification (e.g. struct→int) surfaces the mismatch error.
  Callers must declare result types whose shape matches the fold's value type.
- **−** Unexported fields are lost across the SQL boundary (JSON cannot see
  them). Documented: result types for SQL-backed queries must use exported
  fields.

## Alternatives considered

- **Always JSON-reify (even memory):** rejected — penalizes the memory hot path
  with a per-read marshal/unmarshal for no benefit.
- **Store type tags alongside values:** rejected — couples the projection to a
  serialization format and complicates the schema. The JSON fallback is simpler.
