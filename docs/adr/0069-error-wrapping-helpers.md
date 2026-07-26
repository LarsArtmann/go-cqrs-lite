# ADR-0069: Error-wrapping helper convention

| | |
|---|---|
| **Status** | Accepted |
| **Date** | 2026-07-26 |
| **Context** | go-cqrs-lite multi-module monorepo |

## Context

Every module wraps errors with `errorfamily.WrapInfrastructure(err, code, msg)` (or `WrapTransient`, `WrapCorruption`, etc.). Each call site uses a **unique code string** for traceability (e.g., `"pebble.batch.commit"`, `"kv_sql.set"`). This is the documented convention and is NOT changing.

However, the **control flow** around the wrapping was duplicated:

```go
if err != nil {
    return errorfamily.WrapInfrastructure(err, "pebble.batch.commit",
        "commit batch")
}
return nil
```

This 5-line pattern appeared 20+ times across pebble, kv_sql, encryption, codec, signing, and storage modules. The art-dupl clone detector flagged these as the largest production clone groups.

## Decision

Extract **per-module** helper functions that collapse the control flow while keeping the unique code as a parameter:

```go
// Returns nil when err is nil; otherwise wraps with WrapInfrastructure.
func wrapInfraOrOK(err error, code, msg string) error {
    if err == nil {
        return nil
    }
    return errorfamily.WrapInfrastructure(err, code, msg)
}
```

Call sites become one-liners:

```go
return wrapInfraOrOK(batch.Commit(opts), "pebble.batch.commit", "commit batch")
```

### Where helpers live

Helpers are **unexported and per-module**, NOT promoted to `go-error-family` or a shared internal package. Rationale:

1. **Module independence** — go-cqrs-lite modules have separate `go.mod` files. A shared helper would create a new cross-module dependency for a 5-line function.
2. **Naming flexibility** — modules that need `WrapTransient` define `wrapTransientOrOK`; modules that need `WrapInfrastructure` define `wrapInfraOrOK`. A shared package would need to export all variants.
3. **Small helpers are cheap to duplicate** — the 5-line helper is less code than the dependency it would require.
4. **`go-error-family` stays minimal** — it provides classification primitives (`WrapInfrastructure`, `NewRejection`). Convenience wrappers are the consumer's choice.

### When to extract vs inline

| Pattern | Extract? |
|---------|----------|
| `if err != nil { return WrapX(err, code, msg) }; return nil` appearing 3+ times in a module | **Yes** — extract `wrapXOrOK` |
| Same pattern appearing 1-2 times | **No** — inline is clearer |
| `if err != nil { return nil, WrapX(err, code, msg) }` (tuple return) | **No** — a generic helper adds complexity for marginal gain |
| Module-specific sentinel error definitions (`var ErrX = errorfamily.NewRejection(...)`) | **No** — these ARE the unique values, not duplication |
| `if err := helper(...); err != nil { return err }` (guard pattern) | **No** — standard Go early-return idiom |

## Consequences

- **Positive**: Clone groups reduced by ~30% in modules where helpers were extracted (pebble: 14→11 groups, kv_sql: 3 groups eliminated). Error codes remain unique per call site.
- **Negative**: Each module that needs the helper defines its own unexported copy. This is intentional and aligns with the module-isolation design principle.
- **Cross-module helpers**: When a pattern spans modules AND the modules already share a dependency (e.g., encryption + signing both depend on codec), push the helper into the shared dependency. Example: `codec.MarshalBase64JSONWithModule` eliminates MarshalJSON wrapping in both encryption and signing.
