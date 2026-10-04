> [!NOTE]
> This filing was drafted by GLM-5.3 via Crush from an AI-run investigation, not at my request. When the failure traces to an external report, that source is linked in the body.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

## TL;DR

Fractional-seconds `time.Time` values encode as float64 unix seconds (tag 1), so every CBOR round-trip quantizes them to ~±256 ns at current epochs. Proposed: default `canonicalEncOptions().Time` to `cbor.TimeRFC3339Nano`.

## Problem

`canonicalEncMode` sets `opts.Time = cbor.TimeUnixDynamic` (`cbor.go:32`, `cbor_compact.go:43`). fxamacker encodes fractional times as float64 seconds; at 2026-era epochs (~1.76e9 s, 31 integer bits) the mantissa leaves ~22 fraction bits → ~238 ns worst-case per round-trip.

Measured 2026-10-04 (go-cqrs-lite systemtest, timestamp `…02:55:00.123456789Z`):

| path                                              | drift    |
| ------------------------------------------------- | -------- |
| CBORCodec single round-trip                       | 165 ns   |
| full system + sqliteengine e2e (dispatch → view)  | 165 ns   |
| JSONCodec, same paths                             | 0 ns     |

Consumer impact: LarsArtmann/go-cqrs-lite#50 — dedupe/idempotency comparisons on `time.Time` payload fields fail nondeterministically; that consumer now enforces ±2 µs tolerances in every domain test.

## Proposal

Flip the default `Time` mode to `cbor.TimeRFC3339Nano`:

- 0 ns round-trip (verified), 36-byte wire vs 9-byte float.
- Decode-compatible in both directions: the fxamacker decoder reads tag-0 strings and tag-1 floats, so already-stored data decodes unchanged; only new writes change form.
- `cbor.TimeRFC3339` is NOT an option: it truncates to whole seconds (verified: −123456789 ns on the same timestamp).

Cost: +27 bytes per `time.Time` on the wire. For compact-hot paths the exact alternative is a wrapper type marshaling bare int64 UnixNano (go-cqrs-lite's `Instant`), but that shifts work to every consumer.

Decision needed: exactness-by-default (larger wire) vs status quo + documented tolerance. The go-cqrs-lite side is already amended (ADR-0056 scoped its wrong "preserves nanos" claim; regression test pins both codecs).

💘 Generated with Crush
