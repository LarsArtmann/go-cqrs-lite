# ADR 0050: Envelope JSON Fallback — Keep Forever

> **Status:** ACCEPTED
> **Date:** 2026-07-10
> **Related:** ADR-0044 (blind store encoding stamps), v4 codec flip

## Context

ADR-0044 introduced envelope wrapping for blind stores (kv, snapshot, command,
query). Every write goes through `codec.WrapEncode`, which serializes the
payload with the store's codec (CBOR by default in v4), then wraps it in a JSON
envelope `{"$":"cqrs","enc":"cbor","dat":"..."}` that stamps the encoding.

The read path uses `codec.UnwrapDecode(data, fallback)`. If it finds an
envelope, it extracts the stamped codec and inner data. If no envelope is
found (pre-v4 data — raw JSON without wrapper), it returns the fallback codec
(`JSONCodec`) and the original bytes unchanged.

**The question:** Should this JSON fallback be kept forever, sunsetted in a
future major version, or made configurable?

## Decision Analysis

|                              | **Keep Forever**                                                                                                                                                    | **Sunset in v5**                                                                                                                                                                                                           | **Make Configurable**                                                          |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| **What it means**            | `UnwrapDecode` always tries JSON fallback for non-envelope data                                                                                                     | Deprecation warning now, remove fallback in next major                                                                                                                                                                     | `UnwrapDecode(data, fallback)` + new `UnwrapDecodeStrict(data)` variant        |
| **Read overhead**            | 1 wasted JSON parse attempt on non-envelope data only; envelope data costs the same (the JSON parse IS the envelope extraction)                                     | Zero overhead after removal — envelope is the only path                                                                                                                                                                    | Zero when strict mode enabled; same as "keep" otherwise                        |
| **Migration burden**         | **None.** Old data reads correctly forever.                                                                                                                         | **High.** Every consumer must bulk-rewrite all blind store data (KV rows, snapshots) before v5 or face data loss.                                                                                                          | **Optional.** Consumers opt into strict after their own migration is complete. |
| **Who gets burned**          | Nobody.                                                                                                                                                             | Consumers with long-lived KV/snapshot stores who upgrade without migrating. Blind store data is materialized state — it can't be replayed from the event store like projections can. There is NO automatic migration path. | Nobody (strict is opt-in).                                                     |
| **Code complexity**          | One `if` branch in `UnwrapDecode`. Never changes.                                                                                                                   | Cleaner long-term, but adds a deprecation cycle + logging + eventual removal diff.                                                                                                                                         | Two functions instead of one. Minor API surface increase.                      |
| **The "dead code" argument** | Fallback stays but is exercised every time old data is read. Not truly dead — just rarely hit after migration.                                                      | Truly removed. But "removing rarely-hit code" is a weak motivation for a breaking change.                                                                                                                                  | Same as "keep" by default. Strict variant exists for those who want it.        |
| **Blind store reality**      | KV stores and snapshots hold materialized state that persists across deployments. Some rows may be years old. The fallback is the ONLY thing keeping them readable. | Same reality, but now you're forcing a migration that has no automation. Manual `Scan` + `Set` loop for every typed store.                                                                                                 | Reality acknowledged — default is safe, strict is earned.                      |

## Decision

**Keep the JSON fallback forever.**

The fallback costs nothing measurable (one failed JSON parse on envelope data
costs approximately zero — the JSON parse IS the envelope extraction step, so
there is no wasted work on envelope-wrapped data). The safety is maximal: old
data with no automatic migration path stays readable forever.

Removing the fallback would be a breaking change for purely cosmetic reasons.
The one-line `if` in `UnwrapDecode` is not technical debt — it is a
compatibility layer doing exactly what it should.

### Non-goals

- We will NOT add a deprecation warning for non-envelope data. It is valid,
  supported, and permanent.
- We will NOT add a `UnwrapDecodeStrict` variant. The complexity is not
  justified given the negligible overhead.
- We will NOT document a "migration path" for rewriting old data to envelopes.
  No migration is needed.

## Consequences

- `UnwrapDecode` signature stays `func(data []byte, fallback Codec) (Codec, []byte)`.
- All blind stores pass `codec.JSONCodec{}` as the fallback — this is permanent,
  not a transitional measure.
- Future documentation should describe the fallback as a feature, not a
  deprecation path.
