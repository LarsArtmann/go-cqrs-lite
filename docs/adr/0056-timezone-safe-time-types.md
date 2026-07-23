# ADR-0056: Timezone-Safe Time Types for Event Payloads

**Date:** 2026-07-18
**Status:** Accepted

## Context

CBOR encoding of `time.Time` fields in event payloads causes silent timezone corruption:

1. The `CBORCodec` uses `cbor.TimeUnixDynamic` (verified at `codec/cbor.go:29`), which encodes as epoch float64 with sub-second fractions. On decode, the resulting `time.Time` uses the local timezone, which may differ from the encoding server's timezone.
2. This shifts timestamps by hours when events cross timezone boundaries (e.g., a server in UTC writes, a server in PST reads).

> **Correction:** An earlier version of this ADR claimed the codec used
> `TimeUnix` (truncating to seconds). That was incorrect — the codec uses
> `TimeUnixDynamic` which preserves sub-second precision. The timezone
> corruption risk remains real: `TimeUnixDynamic` decodes into `time.Time`
> using the local timezone, not UTC.

## Decision

Introduce three purpose-built types for event payload time fields:

### `Instant` (event/time_types.go)

- Wraps `time.Time`, enforces UTC at construction
- CBOR: marshals as bare `int64` UnixNano (exact precision, 9 bytes)
- JSON: marshals as RFC3339Nano string (compatible with `time.Time`)
- Use for: created_at, occurred_at, updated_at, expires_at — any unique physical moment

**Why bare int64 instead of CBOR tag 1?** Tag 1 encodes as float64 seconds (lossy)
or requires fractional encoding. Our internal event storage format prioritizes
exact precision over standard CBOR time interop. This format is never exchanged
with external CBOR consumers.

### `WallTime` (event/time_types.go)

- Stores hour, minute, and IANA timezone name (e.g., "America/New_York")
- Not an instant — resolves to different instants depending on the date
- DST-aware: `NextOccurrence` and `PreviousOccurrence` use the IANA timezone database
- Use for: schedules, reminders, business hours — "9am, for whom?"

### `Date` (event/date.go)

- Stores year, month, day — no time, no timezone
- Timezone-agnostic: "2024-03-15" is the same calendar day everywhere
- Use for: birth dates, employment dates, contract dates

## Alternatives Considered

### Keep `time.Time` and document the gotcha

Rejected — documentation doesn't prevent bugs. Developers will forget to call `.UTC()`.

### Use CBOR tag 1 (standard time encoding)

Rejected — tag 1 with float64 loses nanosecond precision. Tag 1 with integer
requires non-standard encoding mode. Internal format, not external interchange.

### Single `Timestamp` type

Rejected — conflates instants and wall-clock times. A single type cannot correctly
handle both "when did this happen?" (instant) and "9am for whom?" (wall time).

## Consequences

- New payload fields should use `Instant`, `WallTime`, or `Date` instead of `time.Time`
- Existing `time.Time` fields continue to work (CBOR `TimeUnixDynamic` preserves nanos)
- The C013 lint rule detects `time.Time` fields in event payloads and suggests replacements
- `time.Time` fields that are genuinely instants can use `.UTC()` as a stopgap
