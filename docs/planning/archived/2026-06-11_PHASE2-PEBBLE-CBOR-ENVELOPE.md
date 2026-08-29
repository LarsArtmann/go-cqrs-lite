# Phase 2: Pebble CBOR Envelope Migration

**Date:** 2026-06-11
**Status:** Plan
**Depends on:** Phase 1 (done — CBORCodec at `a576dadc`)

## Goal

Replace `json.Marshal`/`json.Unmarshal` in `pebble/serialization.go` with CBOR encoding for the event envelope. This eliminates the 33% base64 tax on `[]byte` payload fields and gives deterministic envelope encoding for signing safety.

## Scope

**One file changes behavior:** `pebble/serialization.go`
**Zero files change API:** No consumer-facing changes. `EventStore`, `Save`, `Load`, etc. are identical.
**Migration strategy:** Format sniff (JSON vs CBOR) for backward compatibility.

## What stays the same

- SQL store — already stores raw `[]byte` payload in BLOB column, no envelope serialization
- `codec.Codec` interface — unchanged
- `event.New()` / `event.WithCodec()` — unchanged
- `serializableEvent` struct shape — unchanged (fxamacker reads `json` tags by default)
- Key schemes (`cqrs_event:`, `cqrs_journal:`) — unchanged
- All tests in `store_test.go`, `journal_test.go`, `time_travel_test.go` — should pass without changes

## What changes

- `serializeEvent`: `json.Marshal(s)` → `cborEncMode.Marshal(s)` (CBOR canonical encoding)
- `deserializeEvent`: `json.Unmarshal(data, &s)` → sniff format, then CBOR or JSON decode
- `pebble/go.mod`: `fxamacker/cbor/v2` moves from indirect → direct dep
- Golden test: new CBOR golden fixture
- Benchmarks: add before/after comparison benchmark
