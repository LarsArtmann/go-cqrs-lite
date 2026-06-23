# ADR-0034: Session Store Boundary

**Date:** 2026-06-23
**Status:** Accepted

## Context

The `cqrs-htmx/usermgmt` consumer built a 200-line `SQLSessionStore` from
scratch — DDL, migrations, placeholder dialect mapping, JSON marshaling —
entirely outside the SDK. This raised the question: should the library
provide a `SessionStore` abstraction (and a `Bundle.SessionStore` field)?

## Decision

**Session management is application-layer, not CQRS infrastructure.** The
library will NOT add a `SessionStore` field to `Bundle` or provide a session
store module.

## Rationale

1. **Domain boundary:** CQRS/ES is about command-event-query flow. Sessions
   are an HTTP/auth concern — they belong in middleware or an auth library,
   not in the CQRS data layer.

2. **No CQRS semantics:** Sessions don't have events, don't fold, aren't
   projected. They're a simple key-value lookup (token → user ID) with TTL.
   Adding them to the Bundle would pollute the CQRS abstraction.

3. **Existing solutions:** Go has excellent session libraries
   (`gorilla/sessions`, `alexedwards/scs`) that handle TTL, cookies, and
   backends. Reimplementing this in a CQRS library would be scope creep.

4. **Consumer pattern:** usermgmt's `SQLSessionStore` works fine. It's
   verbose but correct. The verbosity comes from manual SQL, not from a
   missing abstraction — `kv.TypedStore[Session, SessionToken]` would
   reduce it to ~10 lines if the consumer wants to use the library's KV
   store for sessions.

## Recommendation for Consumers

If you need session storage backed by the same database as your CQRS data:

```go
// Use the Bundle's KV store for sessions (10 lines, no custom SQL)
sessionStore := kv.NewTypedStore[SessionData, SessionToken](bundle.ReadModels)
sessionStore.Set(ctx, token, &SessionData{UserID: userID, ExpiresAt: time.Now().Add(24 * time.Hour)})
```

This reuses the library's connection pool and lifecycle without adding a
session-specific abstraction to the Bundle.

## Consequences

- No `Bundle.SessionStore` field — Bundle stays focused on CQRS capabilities.
- Consumers who need sessions use `kv.TypedStore` or an external session library.
- Documented in [MIGRATION_TO_STACK.md](../MIGRATION_TO_STACK.md) under
  "What NOT to Migrate."
