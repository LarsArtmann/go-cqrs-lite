# ADR-0035: Branded DSN Types (Considered and Rejected)

**Date:** 2026-06-24
**Status:** Rejected

## Context

Each stack preset accepts a connection string as a plain `string` parameter:

```go
sqlite.New("app.db")
postgres.New("postgres://host/db")
turso.New("/var/lib/app.db")
```

The question was raised: should each preset define a branded DSN type
(e.g. `sqlite.DSN`, `postgres.DSN`) to prevent accidentally passing the wrong
DSN to the wrong preset at compile time?

## Decision

**Do NOT implement branded DSN types.** Connection strings remain plain `string`.

## Rationale

1. **Breaking change for the common case.** The most common production pattern
   reads the DSN from an environment variable:

   ```go
   dsn := os.Getenv("DATABASE_URL")
   b, _ := sqlite.New(dsn) // breaks if dsn must be sqlite.DSN
   ```

   A branded type forces `sqlite.New(sqlite.DSN(dsn))` at every call site —
   friction for zero practical benefit.

2. **Non-breaking alternative provides no safety.** Using `type DSN = string`
   (alias) is non-breaking but provides zero compile-time safety — it's the
   same type. Only `type DSN string` (named type) provides safety, and only
   for cross-type assignments, not for string literals.

3. **Preset names already disambiguate.** `sqlite.New()` vs `postgres.New()`
   are self-documenting. A developer who passes a Postgres DSN to
   `sqlite.New()` has a configuration error that type safety cannot prevent
   (both are valid strings).

4. **YAGNI.** No consumer has reported a DSN-mixing bug. The theoretical safety
   gain does not justify the API churn across 5 presets and every caller.

## Recommendation

If a consumer wants DSN type safety in their own code:

```go
type AppDSN string

func main() {
    dsn := AppDSN(os.Getenv("DATABASE_URL"))
    b, _ := sqlite.New(string(dsn))
}
```

This keeps the type safety at the application boundary without imposing it on
every library caller.

## Consequences

- All preset `New()` functions continue to accept `string`.
- No branded DSN types in the library.
- Consumers who want DSN type safety can brand at their application boundary.
