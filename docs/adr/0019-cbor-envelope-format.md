# ADR-0019: CBOR Envelope Format for Pebble Stores

**Date:** 2026-06-16  
**Status:** Accepted

## Context

The Pebble-backed stores (EventStore, SnapshotStore, CheckpointStore) serialize
values to disk as CBOR-encoded envelopes. The envelope wraps the payload with
metadata fields (version, timestamps, aggregate reference) in a single deterministic
byte sequence.

Prior to this decision, the on-disk format was undocumented. Consumers performing
file-level backups or external tooling needed to understand the format to safely
inspect or migrate data.

## Decision

All Pebble stores use **canonical CBOR** (RFC 7049, sorted map keys, shortest
floats) as the on-disk envelope format. This is the same `CanonicalEncOptions`
mode used by the `codec/` module, ensuring signing-safe deterministic encoding.

### Envelope Structure

```go
type envelope struct {
    Payload       []byte    `cbor:"payload"`
    EventID       string    `cbor:"eventID"`     // events only
    Type          string    `cbor:"type"`         // events only
    AggregateRef  string    `cbor:"aggregateRef"`
    Version       int       `cbor:"version"`
    OccurredAt    time.Time `cbor:"occurredAt"`
    Metadata      []byte    `cbor:"metadata"`     // JSON-encoded event.Metadata
}
```

### Key Layout

```
cqrs_event:{type}:{id}:{version}      — per-aggregate event log
cqrs_journal:{nanoseconds}:{eventID}  — global event ordering index
cqrs_snapshot:{type}:{id}             — one snapshot per aggregate
cqrs_checkpoint:{projectionName}      — one checkpoint per projection
```

### Backward Compatibility

Deserialization uses format sniffing: it reads the first byte to distinguish
CBOR (0xA0–0xBF, 0xD9 map markers) from legacy JSON (0x7B `{`). Legacy JSON
envelopes are transparently decoded for backward compatibility with pre-CBOR
databases.

## Consequences

- Deterministic encoding enables cryptographic verification of stored data.
- CBOR is more compact than JSON (~30% smaller for typical event payloads).
- External tooling must use a CBOR decoder to inspect raw values.
- File-level backup (copying the Pebble data directory) is safe — all state
  is in the DB files, no external WAL or sidecar files.
