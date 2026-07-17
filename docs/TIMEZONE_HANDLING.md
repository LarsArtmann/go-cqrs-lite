# Timezone Handling in go-cqrs-lite

How to correctly handle `time.Time` values in event payloads, API boundaries, and storage layers.

## The Two Kinds of Time

There are two fundamentally different kinds of time in distributed systems. Confusing them causes silent, business-critical bugs.

|               | **Instant**                                           | **Wall-clock**                                                    |
| ------------- | ----------------------------------------------------- | ----------------------------------------------------------------- |
| **Question**  | "When did this happen?"                               | "9am, for whom?"                                                  |
| **Examples**  | `created_at`, `occurred_at`, audit log, log timestamp | Meeting schedule, business hours, appointment, recurring reminder |
| **Semantics** | One unique physical moment                            | A time-of-day tied to a location                                  |
| **DST-safe?** | Immune (UTC)                                          | **Fatal if converted to UTC**                                     |
| **Storage**   | `int64` UnixNano or `.UTC()` `time.Time`              | Components + IANA timezone name                                   |
| **API input** | RFC3339 with offset (`...Z`, `...-05:00`)             | Wall time + explicit `timezone` field                             |

### Why This Matters

"Tuesday 9am meeting" stored as a UTC instant (`07:00Z` in winter) silently becomes `08:00Z` after a DST flip — the meeting now fires at 10am local time. This is **not a bug you can fix with a migration** — the original intent ("9am for this user") was never stored.

## UTC Convention for Instants

All `time.Time` values representing instants in event payloads **MUST** be converted to UTC before encoding:

```go
// CORRECT — UTC instant
type UserCreated struct {
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

payload := UserCreated{
    Email:     "alice@example.com",
    CreatedAt: time.Now().UTC(), // <-- .UTC() before encoding
}
```

On the decode side, `go-cqrs-lite` automatically reconstructs instants as UTC via `time.Unix(0, nanos).UTC()` in all pebble deserialization paths.

### Why Not Just Use time.Local?

Server timezone can change between deploys, Docker images, or NixOS generations. A timestamp stored as `time.Local` on one server may decode with a different timezone on another. Always normalize to UTC at the boundary.

## Wall-clock Modeling

**NEVER use `time.Time` for wall-clock/recurring times in event payloads.** The CBOR epoch encoding destroys timezone information. Use one of these alternatives:

### Option 1: String (RFC3339Nano with offset)

For one-off local times where the offset is known:

```go
type AppointmentBooked struct {
    When      string `json:"when"`      // "2026-07-17T09:00:00-05:00"
    Timezone  string `json:"timezone"`  // "America/Chicago" (IANA name for DST rules)
}
```

### Option 2: Struct Components

For recurring schedules where the wall time must survive DST transitions:

```go
type DailyReminder struct {
    Hour     int    `json:"hour"`      // 9
    Minute   int    `json:"minute"`    // 0
    Location string `json:"location"`  // "America/New_York" (IANA timezone name)
}

func (r DailyReminder) NextOccurrence(after time.Time) time.Time {
    loc, _ := time.LoadLocation(r.Location)
    // Resolve the wall time in the target timezone for the given date
    // DST is automatically handled by the timezone database
    next := time.Date(after.Year(), after.Month(), after.Day(),
        r.Hour, r.Minute, 0, 0, loc)
    if !next.After(after) {
        next = next.AddDate(0, 0, 1)
    }
    return next
}
```

### Why Not time.Time for Wall-clocks?

CBOR encodes `time.Time` as a Unix epoch (tag 1). An epoch value carries no timezone — it's an absolute instant. When decoded, the library reconstructs it in `time.Local`, which may differ from the original location. This makes it impossible to recover the original wall-clock time.

Even if you use `TimeRFC3339` (which preserves the offset), you lose the IANA timezone name — and without the name, you can't compute future DST transitions for recurring schedules.

## API Boundary Validation

At API boundaries (HTTP handlers, gRPC services, CLI input), **reject timezone-naive timestamps**:

```go
func parseTimestamp(raw string) (time.Time, error) {
    t, err := time.Parse(time.RFC3339Nano, raw)
    if err != nil {
        return time.Time{}, fmt.Errorf("invalid timestamp: %w", err)
    }

    // Reject timezone-naive input — it's ambiguous
    if t.Location() == time.Local && raw[len(raw)-1] != 'Z' {
        // Check if the string contained an offset
        // (time.Parse with RFC3339Nano assigns time.Local if no zone is present)
        zone := raw[len(raw)-6:]
        if !strings.Contains(zone, ":") && raw[len(raw)-1] != 'Z' {
            return time.Time{}, errors.New("timestamp must include timezone offset (e.g., ...Z or ...+02:00)")
        }
    }

    return t.UTC(), nil
}
```

**Acceptable inputs:**

- `2026-07-17T14:30:45Z` (UTC)
- `2026-07-17T09:30:45-05:00` (explicit offset)
- Wall time + `timezone: "America/Chicago"` (IANA name)

**Rejected:**

- `2026-07-17T14:30:45` (no offset — ambiguous)

## CBOR Time Encoding

go-cqrs-lite's CBOR codec uses `TimeUnixDynamic` (float64 epoch with sub-second precision).

| Mode                        | Bytes | Nano Precision                 | Timezone          |
| --------------------------- | ----- | ------------------------------ | ----------------- |
| `TimeUnix` (old default)    | 5     | **NO** (truncated to seconds)  | **NO**            |
| `TimeUnixDynamic` (current) | 9     | ~YES (~165ns float drift)      | **NO**            |
| `TimeRFC3339`               | 21    | **NO** (no fractional seconds) | YES (offset only) |

### Why TimeUnixDynamic?

- **Precision:** Preserves sub-second values (critical for event ordering)
- **Size:** 9 bytes vs 21 bytes for RFC3339
- **Deterministic:** Same input always produces same output (signing-safe)
- **Timezone is irrelevant for instants:** UTC is UTC everywhere

Since the convention mandates `.UTC()` for instants (the only valid use of raw `time.Time` in payloads), timezone preservation in the codec is unnecessary. Wall-clocks use explicit types, never `time.Time`.

### Float64 Drift

`TimeUnixDynamic` encodes as IEEE 754 float64, which has ~165ns drift per round-trip due to mantissa precision limits. This is negligible for event timestamps. For applications requiring exact nanosecond fidelity across multiple encode/decode cycles, store the timestamp as `int64` UnixNano in the payload struct instead.

### Backward Compatibility

Old payloads encoded with `TimeUnix` (seconds precision) are readable by the new codec — CBOR tag 1 is decoded regardless of whether the value is an integer or float. The old data simply has lower precision (second-level), but decodes without error.

## Event Envelope vs Payload

The event **envelope** (`OccurredAt`) is **not affected** by the CBOR time codec. It is stored as `int64` UnixNano in the `serializableEvent` struct, bypassing the `time.Time` codec entirely. The envelope timestamp is always exact and always reconstructed as UTC.

Only **payloads** (user-defined structs with `time.Time` fields) go through the CBOR codec's time encoding. This is where the `.UTC()` convention applies.

## Consumer Migration Guide

### Step 1: Audit Payloads

Search for `time.Time` fields in all event payload structs:

```bash
grep -rn "time\.Time" --include="*.go" | grep -i "payload\|event"
```

### Step 2: Classify Each Field

For each `time.Time` field found, determine:

- **Is this an instant?** ("When did this happen?" — one unique moment)
  - Add `.UTC()` before passing to `event.NewEvent()`
- **Is this a wall-clock?** ("What time of day, for whom?" — tied to a location)
  - Convert to `string` (RFC3339Nano with offset) or a struct with IANA timezone name
  - **NEVER** store as `time.Time`

### Step 3: Fix Instant Fields

```go
// Before (buggy — may encode with local timezone)
evt, _ := event.NewEvent("UserCreated", aggID, "User", v1,
    mustEncode(UserCreated{CreatedAt: time.Now()}))

// After (correct — UTC instant)
evt, _ := event.NewEvent("UserCreated", aggID, "User", v1,
    mustEncode(UserCreated{CreatedAt: time.Now().UTC()}))
```

### Step 4: Fix Wall-clock Fields

```go
// Before (WRONG — DST will corrupt this)
type ScheduleChanged struct {
    ReminderTime time.Time `json:"reminder_time"` // "9am" as instant
}

// After (CORRECT — wall time + IANA timezone)
type ScheduleChanged struct {
    ReminderHour     int    `json:"reminder_hour"`     // 9
    ReminderMinute   int    `json:"reminder_minute"`   // 0
    ReminderTimezone string `json:"reminder_timezone"` // "America/New_York"
}
```

### Step 5: Verify

After migration, test round-trip encoding:

```go
func TestTimeRoundTrip(t *testing.T) {
    original := MyPayload{CreatedAt: time.Date(2026, 7, 17, 14, 30, 45, 123456789, time.UTC)}

    codec := codec.CBORCodec{}
    data, err := codec.Encode(original)
    if err != nil {
        t.Fatal(err)
    }

    var decoded MyPayload
    _ = codec.Decode(data, &decoded)

    delta := decoded.CreatedAt.Sub(original.CreatedAt)
    if delta < 0 { delta = -delta }
    if delta > time.Microsecond {
        t.Errorf("precision lost: delta=%v", delta)
    }
}
```

## Quick Reference

| Do                                            | Don't                                              |
| --------------------------------------------- | -------------------------------------------------- |
| `.UTC()` all `time.Time` before encoding      | Store `time.Local` in payloads                     |
| Use `string` or struct for wall-clocks        | Use `time.Time` for recurring schedules            |
| Require timezone offset at API boundaries     | Accept naked timestamps like `2026-07-17T09:00:00` |
| Store envelope timestamps as `int64` UnixNano | Store envelope timestamps as `time.Time`           |
| Test sub-second precision round-trips         | Assume integer epoch means UTC on decode           |
