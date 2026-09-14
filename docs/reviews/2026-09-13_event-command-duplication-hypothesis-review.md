# Review: Is the event/command duplication an abstraction mistake?

> **Date:** 2026-09-13
> **Kind:** Point-in-time hypothesis review (no code changes)
> **Repo state:** `master` @ 0711ef9e7
> **Hypothesis under test:** "We build a lot for Events and Commands twice, NOT because
> they are so different BUT because we have some Go type or general abstraction mistake."
> **Verdict up front:** **Rejected for the logic layer, confirmed at one specific seam.**
> The "built twice" feeling comes from the file layout (parallel `bus.go`/`store.go`/
> `dispatcher.go`/`errors.go` per module), not from the code inside those files — the
> behavior was already unified into Tier-0 generic cores. What remains genuinely wrong
> is the `command.AsRecord` bridge fidelity, detailed in §3.
> **Companion reviews:** [event/ split re-review](2026-09-13_event-module-split-re-review.md) ·
> [command-side depth](2026-09-13_command-side-depth-review.md)
> **Series:** extends [2026-08-22 core data-model review](2026-08-22_core-data-model-review.html)
> (whose P5 finding — AsRecord bridges dropping instance ID — was fixed via `record.Record.ID`;
> the asymmetry below is the next defect in the same family)

---

## 1. What looks duplicated but is NOT (behavior already unified)

The repo's doctrine is "generic cores, injected policies" (ADR-0126). Verified per item:

| Parallel surface                                                          | What actually lives there                                                                                                                                                                           |
| ------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `dispatcher.go` ×2 (command/query)                                        | Both embed Tier-0 `dispatcher.Dispatcher[H, M]` — thin facades over one generic core                                                                                                                |
| `middleware.CommandIdempotency` / `EventIdempotency` / `QueryIdempotency` | One generic `NewIdempotency(adapter, store, ttl, key)` core + three ~15-line adapters differing only in key strategy (command/event have minted IDs; query requires an extractor and panics on nil) |
| Per-kind storage (`storage/memory`, `storage/sql`)                        | Generic `LogStore[T, ID]` (ADR-0126) and `Inserter[T]`; `storage/memory/command_store.go` embeds `*LogStore[*command.PersistedCommand, id.CommandID]`                                               |
| `Metadata` ×3                                                             | Aliases of one generic `metadata.Metadata[K]` (ADR-0031)                                                                                                                                            |
| Codec envelopes                                                           | One ADR-0044 envelope (`WrapEncode`/`UnwrapDecode`) shared by all blind stores                                                                                                                      |
| `asrecord.go` ×3                                                          | Necessary per-kind bridges to `record.Record`; the command/query twins carry `//art-dupl:accept "dep-isolated twin; lockstep Record population is by design"`                                       |

## 2. Why the port TYPES are not copies

The interface declarations rhyme (`Publisher`, `Subscriber`, `Bus`, `Store`,
`Middleware` per kind), but the method sets encode genuinely different capabilities:

- `event.EventSink.Save(ctx, ref, events, expectedVersion)` — **optimistic concurrency check**;
  `AppendBatch` explicitly documents "without concurrency checks" as the exception.
- `event.EventSource` reads by **version** (`LoadFromVersion`/`LoadToVersion`); events
  additionally get `SeekableJournal`, `StreamingJournal`, `BackwardsSource`, checkpoints.
- `command.CommandSink.Save` — no OCC (commands are intents, not append-only facts);
  `CommandSource` reads by **timestamp** (`LoadFromTimestamp`/`LoadToTimestamp`).

The core message types are essentially different shapes — 11 fields (version, encoding,
schemaVersion, occurredAt, payload, opts) vs 4 (`BasicCommand`) vs 2 (`BasicQuery`).
Facts are rich because they are persisted and replayed; intents are thin because they are
transient. Collapsing the ports into a generic `Publisher[M]`/`Store[M]` module — or the
watermill-style single `Message` type — would trade ~15 thin declarations for
type-assertions everywhere and violate the repo's strong-types rule. The per-kind port
families are the idiomatic Go trade, and further unification is explicitly v5-frozen
(ADR-0111 item g: growing core interfaces is breaking).

Genuinely copied residue today: the ~5-line capability interfaces
(`command.MetadataCarrier` vs `query.MetadataCarrier`/`PayloadCarrier`) — known, accepted
until v5.

## 3. The one real abstraction mistake: `AsRecord` bridge asymmetry

```go
func AsRecord(evt Event) record.Record              // event:  bridges the runtime type   — full fidelity (payload, encoding, version)
func AsRecord(q *PersistedQuery) record.Record      // query:  bridges the PERSISTED form — payload ✓ encoding ✓
func AsRecord(cmd *BasicCommand) record.Record      // command: bridges the THIN form     — NO payload, NO encoding, NO timestamp
```

Commands exist in two shapes — `BasicCommand` (intent: ID, type, streamID, metadata) and
`*PersistedCommand` (audit form with payload/encoding) — and the record bridge picked the
thin one. Consequence: a command reaching metaengine routing/folding arrives
**payload-less**, a second-class `record.Record`. This is the same root cause as the
command-side depth gap (see companion review): because command records are hollow,
`commandlifecycle/` had to wrap commands in its OWN event payloads (ADR-0117), and
ADR-0112 (command sourcing — folding over command history) has nothing to fold.

**Fix direction:** `command.AsRecord` should take (or offer a variant for)
`*PersistedCommand` — mirroring the query bridge. Non-breaking (additive
variant or signature change at v5). This also feeds the v5 port-unification list.
**Planned:** W2 of
[`docs/planning/2026-09-13_11-45_SUPERB-command-side-depth.md`](../planning/2026-09-13_11-45_SUPERB-command-side-depth.md).

## 4. The Go-level answer to "did we pick the wrong abstraction?"

No. The repo chose **generic cores + thin per-kind facades** over **generic message
types**. The evidence above shows the doctrine is applied consistently where it matters
(dispatcher, idempotency, WAL, metadata, envelope, record spine). The mistake isn't the
pattern — it's one bridge that disobeys it.

## 5. v5 unification list (deferred by ADR-0111 g, collected here)

1. `command.AsRecord` over `*PersistedCommand` (§3 — can land pre-v5 as an additive variant).
2. Generic capability interfaces in `record/` or `metadata/` (replace the per-module
   5-line copies).
3. Revisit generic port families (`Publisher[M]`, `Store[M]`) only if a consumer
   population that wants them emerges — same falsifier test as the event/ split decision.
