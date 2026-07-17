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

The actual on-disk struct is `serializableEvent` (see `storage/pebble/serialization.go`).
fxamacker/cbor reads `json` struct tags by default, so no separate `cbor` tags are needed:

```go
type serializableEvent struct {
    ID            id.EventID     `json:"id"`
    Type          string         `json:"type"`
    AggregateID   id.AggregateID `json:"aggregate_id"`
    AggregateType string         `json:"aggregate_type"`
    Version       int            `json:"version"`
    SchemaVersion int            `json:"schema_version,omitempty"`
    Payload       []byte         `json:"payload"`
    OccurredAt    int64          `json:"occurred_at"` // Unix nanoseconds — NOT time.Time
    Metadata      event.Metadata `json:"metadata"`
    Encoding      string         `json:"encoding,omitempty"`
}
```

`OccurredAt` is stored as `int64` Unix nanoseconds, not `time.Time`, to ensure
exact precision and timezone-independent storage. Deserialization reconstructs
the time via `time.Unix(0, s.OccurredAt).UTC()` — always UTC, never `time.Local`.

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
