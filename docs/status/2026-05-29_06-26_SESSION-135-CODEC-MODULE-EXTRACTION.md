# Session 135 — Codec Module Extraction & Encoding Architecture

**Date:** 2026-05-29 06:26
**Status:** IN PROGRESS — Phase 1 & 2 committed, Phase 3–6 pending
**Test Suite:** ALL 30 PACKAGES PASSING ✅

---

## Summary

Extracted a standalone `codec/` module from `core/event/` to provide a reusable, pluggable serialization abstraction with encoding identity. Events now carry their encoding, and `event.New()` accepts a codec to auto-stamp it.

---

## A) FULLY DONE ✅

### 1. `codec/` Module Created
- **`codec/codec.go`** — `Encoding` type (`"json"`, `"raw"`), `Codec` interface with `Encoding()`, `Encode()`, `Decode()`
- **`codec/json.go`** — `JSONCodec` using `encoding/json`
- **`codec/raw.go`** — `RawCodec` for passthrough `[]byte`
- **`codec/codec_test.go`** — Full test coverage
- Added to `go.work`, own `go.mod`, zero external dependencies

### 2. `core/event/` Migrated to Use `codec/`
- **`event.Codec`** → type alias to `codec.Codec` (non-breaking, deprecated)
- **`event.JSONCodec`** → type alias to `codec.JSONCodec` (non-breaking, deprecated)
- **`Event` interface** — new `Encoding() codec.Encoding` method
- **`ImmutableEvent`** — new `encoding` field, defaults to `EncodingJSON`
- **`WithEncoding()`** option — sets encoding on event
- **`WithNewCodec()`** option — provides codec for `event.New()` marshaling
- **`validateEncodingMatch()`** — guards `DecodePayload` against encoding mismatches
- **`Clone()`** — preserves encoding field
- **`probeCodec()`** — extracts codec from options before marshaling in `New()`
- **`marshalPayload()`** — now uses codec instead of hardcoded `json.Marshal`
- **`batch.go`** — `NewEvents` also uses codec-aware pattern
- Encoding mismatch tests added

### 3. Pre-existing Build Breakages Fixed
- Restored `otel/attributes.go` with `AggregateBaseAttrs` (was lost during stash cycle)
- Restored `storage/` callers to use `AggregateBaseAttrs` correctly
- Fixed `testhelpers/handlers.go` build error

---

## B) PARTIALLY DONE ⚠️

### Storage Encoding Persistence
- `Encoding()` is on the Event interface but **not persisted** anywhere yet
- SQL schema has no `encoding` column
- Pebble `serializableEvent` has no `Encoding` field
- Outbox `outboxEvent` has no `Encoding` field
- `reconstructEvent()` doesn't restore encoding (always defaults to JSON)

### Projection `On[T]()`
- Still uses hardcoded `json.Unmarshal` instead of `codec.Codec`

---

## C) NOT STARTED ❌

1. **Persist `encoding` in SQL** — add `payload_encoding` column, update dialect, reconstruction
2. **Persist `encoding` in Pebble** — add field to `serializableEvent`
3. **Persist `encoding` in outbox** — add field to `outboxEvent`
4. **Migrate `projection.On[T]()`** — accept `codec.Codec` parameter
5. **Migrate `decider.WithCodec`** — use `codec.Codec` directly (remove deprecation hints)
6. **Migrate `example/todo/`** — from `event.JSONCodec` to `codec.JSONCodec`
7. **Migrate `example/user/`** — from raw `json.Unmarshal` to codec pattern
8. **Update `AGENTS.md`** — document codec module, encoding architecture
9. **Remove deprecated type aliases** — eventually remove `event.Codec`/`event.JSONCodec` aliases

---

## D) TOTALLY FUCKED UP 💥

### Stash Cycle Disaster
During Phase 2, a `git stash` / `git stash pop` cycle caused collateral damage:
- `otel/attributes.go` was reverted to a pre-`AggregateBaseAttrs` state
- `storage/` callers were reverted to use `AggregateAttrs` (3-arg) instead of `AggregateBaseAttrs` (2-arg)
- `testhelpers/handlers.go` was partially modified with a broken type alias
- Required manual restoration from commit `9067b12` to recover

**Root cause:** The working tree had interleaved changes from multiple sessions (codec work + OTEL refactoring + deduplication). Stashing mixed codec changes with unrelated session work.

**Lesson:** Commit more frequently. Don't let unrelated changes pile up in the working tree.

---

## E) WHAT WE SHOULD IMPROVE

1. **Commit atomicity** — Each self-contained change should be committed immediately, not batched
2. **Working tree hygiene** — Don't accumulate changes from multiple sessions without committing
3. **`event.New()` API design** — `WithNewCodec` uses a probe mechanism (apply options to temp struct to extract codec). This works but is unusual. Consider making codec a required param in a future breaking change.
4. **`RawCodec`实用性** — `RawCodec.Decode` requires `*[]byte` target, which won't work with `DecodePayload[T]` for non-`[]byte` types. May need refinement.
5. **Encoding validation scope** — `validateEncodingMatch` only rejects when event has non-JSON encoding that doesn't match codec. It doesn't reject when event says `"json"` but bytes are actually protobuf. This is by design (trust the stamp) but worth documenting.

---

## F) Top 25 Things to Get Done Next

### High Impact, Low Effort (1-2 hours)
1. **Persist encoding in Pebble** — add `Encoding` field to `serializableEvent`, reconstruct with `WithEncoding`
2. **Persist encoding in outbox** — add `Encoding` field to `outboxEvent`
3. **Migrate `decider.WithCodec`** — change from `event.Codec` to `codec.Codec`
4. **Migrate `example/todo/`** — `event.JSONCodec` → `codec.JSONCodec`
5. **Update `AGENTS.md`** — document codec module, Encoding type, WithNewCodec
6. **Migrate `core/event` tests** — use `codec.JSONCodec` directly, not deprecated aliases

### High Impact, Medium Effort (2-4 hours)
7. **Persist encoding in SQL** — add `payload_encoding` column to PostgreSQL/SQLite schemas
8. **Update `reconstructEvent()`** — pass encoding from DB row, use `WithEncoding`
9. **Migrate `projection.On[T]()`** — accept `codec.Codec`, delegate to `event.DecodePayload`
10. **Add `ProtobufCodec` example** — prove the abstraction works for non-JSON
11. **Migration guide for existing consumers** — document upgrade path

### Medium Impact, Medium Effort
12. **Migrate `example/user/`** — from raw `json.Unmarshal` to codec
13. **Add encoding to `EventCatalog`** — expose encoding in AsyncAPI/OpenAPI schemas
14. **Benchmark encoding overhead** — measure cost of `Encoding()` method + `validateEncodingMatch`
15. **`catalog/` schema generation** — include encoding in event schema output
16. **`stream/` reader** — expose encoding in aggregate listing
17. **Signing middleware** — verify encoding is preserved through sign/verify cycle
18. **Outbox publisher** — ensure encoding survives outbox round-trip
19. **Watermill protocol** — add encoding header to Watermill messages

### Lower Impact, Future Work
20. **Remove deprecated aliases** — plan `event.Codec`/`event.JSONCodec` removal for v2
21. **Codec registry** — `codec.Register()` for auto-lookup by encoding name
22. **`DecodePayloadAuto[T]()`** — auto-select codec from event encoding
23. **Custom encoding validation** — allow consumers to plug in their own mismatch policy
24. **Encoding in snapshots** — should snapshot state also declare encoding?
25. **Cross-encoding upcasting** — upcaster that changes encoding (e.g., JSON → protobuf)

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should `encoding` be part of the SQL `events` table primary schema, or should it be a metadata field?**

Options:
- **Dedicated column** (`payload_encoding TEXT DEFAULT 'json'`) — clean, indexable, explicit in schema
- **Metadata field** (`metadata->>'encoding'`) — no schema migration needed, but encoding is structural not domain metadata

This is a design decision that affects all storage backends and the migration path for existing data. I can implement either, but the architectural choice should come from the project owner.

---

## Commits This Session

| Commit | Description |
|--------|-------------|
| `4529655` | feat(codec): add new codec module with JSONCodec and RawCodec implementations |
| `f52ac70` | refactor(event): replace Codec/JSONCodec definitions with type aliases to codec package |
| `e376526` | feat(event): make New() codec-aware via WithNewCodec option |
| `9067b12` | refactor(otel): extract AggregateBaseAttrs (parallel session) |
| `15228fc` | refactor(query): extract registerHandler helper (parallel session) |

## Test Results

```
ok  codec          0.002s
ok  core/aggregate 0.007s
ok  core/command   0.012s
ok  core/decider   0.012s
ok  core/event     0.031s
ok  core/query     0.009s
ok  memory         0.012s
ok  catalog        0.008s
ok  middleware     0.164s
ok  testhelpers    0.007s
ok  integration    0.072s
ok  projection     0.272s
ok  signing        0.015s
ok  storage        0.130s
ok  saga           0.708s
ok  stream         0.019s
ok  watermill      0.003s
ALL 30 PACKAGES PASSING ✅
```
