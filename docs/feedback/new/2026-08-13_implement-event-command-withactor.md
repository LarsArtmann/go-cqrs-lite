# Proposal: Implement `event.WithActor` and `command.WithActor`

> **From**: overview consumer (via cqrs-htmx `EventOptionsFromContext`)
> **Date**: 2026-08-13
> **Priority**: High (build-breaking gap — cqrs-htmx master already calls `event.WithActor`)
> **Versions**: `event/v4 v4.5.0`, `command/v4 v4.5.0`, `id/v4 v4.3.0`, `metadata/v4 v4.3.0`

---

## Summary

`event.WithActor` is documented in `docs/api_surface.txt` and already called by cqrs-htmx master (`context.go:246`), but the function does not exist in the `event` package source. `command.WithActor` has no documentation and no implementation, but is the natural symmetric counterpart. Both are needed for the actor-chain audit trail that cqrs-htmx's `EventOptionsFromContext` and `CommandOptionsFromContext` already attempt to propagate.

Every prerequisite type already exists and is fully implemented:
- `id.ActorID` — kind-discriminated struct (`id/actor_id.go:64`) with constructors, `IsZero()`, `Equal()`, `PrefixedString()`, and JSON marshal/unmarshal (`id/actor_id_json.go`).
- `metadata.Tracing` — shared struct embedded by both `event.Metadata` and `command.Metadata` (`metadata/metadata.go:16`), already holds `UserID`, `CorrelationID`, `CausationID`, `RequestID`.
- The `apply` helper and `Option` pattern — identical across event and command packages.

Only the field on `Tracing` and the two option functions are missing.

---

## Current state

### What exists

```
id/actor_id.go         — ActorID struct, constructors, IsZero, Equal, PrefixedString
id/actor_id_json.go    — MarshalJSON / UnmarshalJSON (serializes to "kind:raw" string)
metadata/metadata.go   — Tracing struct { CorrelationID, CausationID, UserID, RequestID }
event/options.go       — WithUserID, WithCorrelationID, WithCausationID, WithRequestID, ...
command/metadata.go    — WithUserID, WithCorrelationID, WithCausationID, WithRequestID, ...
```

### What's missing

```
metadata.Tracing       — no ActorID field
event/options.go       — no WithActor function
command/metadata.go    — no WithActor function
docs/api_surface.txt   — lists "event/func WithActor" but source doesn't implement it
```

### Who calls it

cqrs-htmx `context.go:230-250` (`EventOptionsFromContext`):

```go
// Propagate actor chain for audit trail.
// ActorID = who the request acts AS (effective identity).
// ImpersonatorID = who is REALLY authenticated (the admin).
// When both are set, every event carries the full chain for compliance queries.
if actorID := ActorIDFromContext(ctx); !actorID.IsZero() {
    opts = append(opts, event.WithActor(actorID))  // ← undefined
}

if impersonatorID := ImpersonatorIDFromContext(ctx); !impersonatorID.IsZero() {
    opts = append(opts, event.WithCustom(MetadataKeyImpersonatorID, impersonatorID.PrefixedString()))
}
```

cqrs-htmx also re-exports the type chain: `type ActorID = id.ActorID` (`context.go:184`), `ParseActorID` (`context.go:190`), `WithActorID` (`context.go:200`), `ActorIDFromContext` (`context.go:204`). The entire context-propagation layer is built; only the metadata-option terminus is missing.

---

## Proposal

### 1. Add `ActorID` to `metadata.Tracing`

`metadata/metadata.go`:

```go
type Tracing struct {
    CorrelationID id.CorrelationID `json:"correlationId"`
    CausationID   id.CausationID   `json:"causationId"`
    UserID        id.UserID        `json:"userId"`
    RequestID     id.RequestID     `json:"requestId"`
    ActorID       id.ActorID       `json:"actorId,omitempty"`
}
```

Update `IsZero`:

```go
func (t Tracing) IsZero() bool {
    return t.CorrelationID.IsZero() &&
        t.CausationID.IsZero() &&
        t.UserID.IsZero() &&
        t.RequestID.IsZero() &&
        t.ActorID.IsZero()
}
```

Update `Merge`:

```go
func (t Tracing) Merge(other Tracing) Tracing {
    result := t

    if !other.CorrelationID.IsZero() {
        result.CorrelationID = other.CorrelationID
    }
    if !other.CausationID.IsZero() {
        result.CausationID = other.CausationID
    }
    if !other.UserID.IsZero() {
        result.UserID = other.UserID
    }
    if !other.RequestID.IsZero() {
        result.RequestID = other.RequestID
    }
    if !other.ActorID.IsZero() {
        result.ActorID = other.ActorID
    }

    return result
}
```

### 2. Add `event.WithActor`

`event/options.go` — identical pattern to `WithUserID` (`event/options.go:65`):

```go
// WithActor sets the effective actor (user, bot, system, or service) that
// initiated the event. This is the primary audit-trail field for compliance.
func WithActor(v id.ActorID) Option {
    return apply(func(m *Metadata, v id.ActorID) { m.ActorID = v }, v)
}
```

No changes needed to `event.Metadata` — it embeds `metadata.Tracing`, so the `ActorID` field is already there.

### 3. Add `command.WithActor`

`command/metadata.go` — identical pattern to `WithUserID` (`command/metadata.go:82`):

```go
// WithActor sets the effective actor (user, bot, system, or service) that
// issued the command. This is the primary audit-trail field for compliance.
func WithActor(v id.ActorID) Option {
    return apply(func(m *Metadata, v id.ActorID) { m.ActorID = v }, v)
}
```

No changes needed to `command.Metadata` — it embeds `metadata.Tracing`, so the `ActorID` field is already there.

---

## Why `metadata.Tracing` (not per-package fields)

Placing `ActorID` on `metadata.Tracing` rather than on `event.Metadata` / `command.Metadata` individually gives three benefits:

1. **DRY** — one field, one merge rule, one IsZero check. Both packages inherit it via embedding.
2. **Consistency with `UserID`** — `UserID` is already on `Tracing`; `ActorID` generalizes it (a user IS one kind of actor). Keeping them at the same level makes the hierarchy clear.
3. **Future symmetry** — if a `query.WithActor` is ever needed, it inherits the field automatically (query metadata embeds `Tracing` too, or should if it doesn't already).

---

## JSON serialization

No new code needed. `id.ActorID` already implements `json.Marshaler` and `json.Unmarshaler` in `id/actor_id_json.go`, serializing to/from the self-describing `"kind:raw"` string form (e.g. `"user:01JXYZ..."`, `"system:scheduler"`). Adding it to `Tracing` works out of the box:

```json
{
  "correlationId": "01JXYZ...",
  "causationId": "",
  "userId": "01JXYZ...",
  "requestId": "01JXYZ...",
  "actorId": "user:01JXYZ..."
}
```

The `omitempty` tag works because `ActorID.MarshalJSON` returns `"null"` for the zero value (empty `PrefixedString()`), and `encoding/json` with `GOEXPERIMENT=jsonv2` omits null-valued struct fields tagged `omitempty`. If `json/v1` semantics differ, verify the zero-value omission in tests.

---

## Relationship to `UserID`

`ActorID` and `UserID` are related but serve different purposes:

| Field | Scope | Granularity | Use case |
|-------|-------|-------------|----------|
| `UserID` | Authentication identity | One type: user only | "Who is authenticated?" |
| `ActorID` | Effective identity (audit) | Kind-discriminated: user, bot, system, service | "Who/what caused this record?" |

When `ActorID.Kind() == ActorUser`, the `Raw()` value SHOULD match `UserID.String()`. They are NOT redundant: `UserID` answers "who logged in"; `ActorID` answers "who is the system acting on behalf of" (which may be a bot, a scheduler, or an impersonated user). Both fields should be populated when available.

cqrs-htmx already models this correctly: `EventOptionsFromContext` propagates BOTH `WithUserID(userID)` and `WithActor(actorID)` — they carry different information.

---

## Impact

### Breaking changes

None. This is purely additive:
- New field on `Tracing` (zero value = no actor, backward-compatible)
- New option functions (callers that don't use them are unaffected)
- `Tracing.IsZero` gains a check (was true before, still true for zero ActorID)

### Downstream consumers

- **cqrs-htmx** — unblocks master; the `event.WithActor` call in `context.go:246` resolves.
- **All event/command consumers** — gain audit-trail capability with zero migration cost.
- **overview** — allows unpinning cqrs-htmx from v4.7.0 tag back to master (currently pinned via flake `&rev=` due to this gap).

### Versioning

This is a minor version bump for `metadata/v4`, `event/v4`, and `command/v4` (additive API change). Tag `metadata/v4 v4.4.0`, `event/v4 v4.6.0`, `command/v4 v4.6.0` (or similar) once implemented. `id/v4` and `metadata.Tracing` are the foundation — `metadata/v4` must be tagged first, then event/command can require the new version.

---

## Suggested implementation order

1. Add `ActorID` field to `metadata.Tracing` + update `IsZero` and `Merge` (`metadata/metadata.go`)
2. Tag `metadata/v4 v4.4.0`
3. Bump `event/v4`'s `metadata/v4` require to v4.4.0, add `WithActor` to `event/options.go`
4. Bump `command/v4`'s `metadata/v4` require to v4.4.0, add `WithActor` to `command/metadata.go`
5. Tag `event/v4 v4.6.0` and `command/v4 v4.6.0`
6. Update `docs/api_surface.txt` (remove the phantom entry, which will now be real)
7. In cqrs-htmx: bump event/command requires, remove any local workarounds
