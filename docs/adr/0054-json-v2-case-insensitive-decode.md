# ADR-0054: json/v2 Case-Insensitive Decode

> **Status:** ACCEPTED
> **Date:** 2026-07-10

## Context

Go 1.26 introduced `encoding/json/v2` which defaults to **case-sensitive** field
name matching. The v1 encoder was case-insensitive by default. This means:

```json
{ "Name": "Alice" }
```

would silently produce a zero-value `Name` field when decoded into a struct
with `Name string` (no json tag) under json/v2, because `Name` ≠ `name`.

This is a silent data corruption bug. It affects:

- Event metadata decode (`event.Metadata` with promoted `Tracing` fields)
- SQL store scan paths (metadata JSON columns)
- Pebble legacy fallback decode
- Signing/encryption envelope decode

## Decision

All `json.Unmarshal` calls in library code use `json.MatchCaseInsensitiveNames(true)`:

```go
json.Unmarshal(data, &v, json.MatchCaseInsensitiveNames(true))
```

This restores the v1 behavior for decode paths and prevents silent zero-value
bugs when consumers send data with different casing conventions.

The `codec.JSONCodec.Decode` method already includes this option by default
(v3.7.0+). All direct `json.Unmarshal` calls in library code were updated in
the same release.

## Alternatives Considered

- **Require json tags on all struct fields** — Rejected: too much churn,
  breaks untagged structs that rely on Go field name matching.
- **Make json/v2 case-sensitive the default** — Rejected: silent data
  corruption is worse than a minor performance cost.

## Consequences

- Decode is slightly slower (case-insensitive matching has a small cost).
- Consumers sending `{"name": "Alice"}` to a struct with `Name string` will
  correctly populate the field.
- The v4 codec default flip (JSON → CBOR) will not change this behavior
  since CBOR uses field names, not JSON casing.
