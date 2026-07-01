# JSON → CBOR Migration Guide

> **Scope:** How to adopt CBOR encoding in an existing go-cqrs-lite project that uses JSON.

---

## Why CBOR?

CBOR (RFC 8949) is a binary serialization format that is:

- **19–43% smaller** than JSON for typical event payloads
- **25–72% faster** to encode/decode
- **Deterministic** (canonical mode: sorted map keys) — safe for content-addressed storage and cryptographic signing
- **Self-describing** — unlike protobuf, the schema is optional at decode time

See `codec/benchmark_test.go` for measured numbers.

---

## Key Principle: Events Are Self-Describing

Every event created by `event.New()` stamps its encoding (`evt.Encoding()` returns `"json"` or `"cbor"`). This means:

- **Mixed JSON+CBOR event streams work correctly** — `event.DecodePayload()` matches the codec to the event's stamp.
- **You can migrate incrementally** — new events use CBOR, old events stay JSON, both decode correctly.
- **Blind stores (kv/snapshot/command/query) have NO encoding stamp** — these require a full clear-and-rebuild if you change the codec.

---

## Migration Path

### Step 1: Adopt CBOR for Read Models (Safe — No Data Loss)

Read models are projections rebuilt from events. Changing their codec only affects how view records are stored, not the event log.

```go
// Option A: Stack-level (recommended — one call)
bundle, _ := sqlite.New(dsn, stack.WithDefaultCodec(codec.CBORCodec{}))
store, _ := stack.ReadModel[Todo, TodoID](bundle, nil) // nil → uses CBOR

// Option B: Per-store
store, _ := stack.ReadModel[Todo, TodoID](bundle, codec.CBORCodec{})
```

Then **rebuild your projections** (clear the KV store and replay events):

```go
// Delete all view records, then replay from checkpoint 0
store.DeleteAll(ctx)
checkpointStore.Delete(ctx, projectionName) // forces full replay
```

### Step 2: Adopt CBOR for New Events (Gradual — Mixed Stream)

```go
// Option A: Process-wide (affects all event.New() calls without explicit WithCodec)
event.DefaultCodec = codec.CBORCodec{}

// Option B: Stack-level (via bundle)
bundle, _ := sqlite.New(dsn, stack.WithEventCodec(codec.CBORCodec{}))
event.DefaultCodec = bundle.EventCodec()

// Option C: Per-event (explicit)
evt, _ := event.New("user.created", id, "User", 1,
    UserCreated{Name: "Alice"}, event.WithCodec(codec.CBORCodec{}))
```

After this, new events are stamped `"cbor"`. Old events in the store remain `"json"`. Both decode correctly:

```go
// This works for BOTH json and cbor events:
switch evt.Encoding() {
case codec.EncodingJSON:
    p, _ = event.DecodePayload[UserCreated](evt, codec.JSONCodec{})
case codec.EncodingCBOR:
    p, _ = event.DecodePayload[UserCreated](evt, codec.CBORCodec{})
}
```

Or use `event.DefaultCodec` which matches whatever you configured.

### Step 3: Adopt CBOR for Snapshots (Requires Re-snapshot)

```go
repo, _ := decider.NewRepository(store, bus, decider,
    decider.WithSnapshotStore(snapStore),
    decider.WithCodec(codec.CBORCodec{}),
)
```

**Clear existing snapshots** before switching — old JSON snapshots will fail to decode with CBOR:

```go
snapStore.Delete(ctx, "User", userID) // per aggregate
// Or clear all and let them rebuild naturally
```

Snapshots are an optimization (cache), not the source of truth. Deleting them just means the next load rebuilds from events.

### Step 4: Adopt CBOR for Blind Stores (v4 Only)

The typed stores (`kv.NewTypedStore`, `snapshot.TypedStore`, `command.TypedStore`, `query.TypedStore`) are "blind" — they serialize values as raw bytes with no format tag. Changing the default would silently break consumers with existing JSON data.

**v3.x:** Pass the codec explicitly per store:

```go
store := kv.NewTypedStore[UserView, UserID](backend,
    kv.WithTypedCodec(codec.CBORCodec{}))
```

**v4:** The defaults will flip to CBOR. Tracked in `docs/v4-WISHLIST.md` item #8.

---

## Decision Matrix

| What you're changing           | Safe to do incrementally? | Old data breaks?            | Recovery                     |
| ------------------------------ | ------------------------- | --------------------------- | ---------------------------- |
| Read model codec               | Yes                       | Old view records unreadable | Rebuild from events          |
| Event codec (new events)       | Yes                       | No — old events stay JSON   | N/A (mixed stream works)     |
| Snapshot codec                 | Yes (per aggregate)       | Old snapshots unreadable    | Delete + rebuild from events |
| Blind store codec (kv/cmd/qry) | No — must clear           | Yes — old data unreadable   | Rebuild from events/source   |

---

## Encryption + CBOR

Use the **middleware path** (`EncryptMiddleware`/`DecryptMiddleware`), not the codec wrapper (`encryption.NewCodec`):

```go
// CORRECT: middleware preserves encoding stamp
bus.UsePublish(encryption.EncryptMiddleware(enc))
bus.Use(encryption.DecryptMiddleware(enc))

// Events created as CBOR stay stamped "cbor" after encrypt → decrypt.
// event.DecodePayload(evt, codec.CBORCodec{}) works on decrypted events.
```

The codec wrapper (`encryption.NewCodec`) stamps `"encrypted"`, which discards the inner codec's format. It's designed for non-event serialization (e.g. snapshot values in blind stores), not for event payloads.

---

## Verifying the Migration

```go
// Use codec.Size to measure the benefit before committing
jsonSize, cborSize := codec.Size(UserCreated{Name: "Alice", Email: "a@b.c"})
fmt.Printf("savings: %.0f%%\n", float64(jsonSize-cborSize)/float64(jsonSize)*100)

// Use codec.AutoDetect to inspect raw bytes
enc := codec.AutoDetect(evt.Payload())
fmt.Printf("actual encoding: %s (stamped: %s)\n", enc, evt.Encoding())

// Use codec.Diagnose for human-readable CBOR debugging
diag, _ := codec.Diagnose(evt.Payload())
log.Printf("CBOR payload: %s", diag)
```
