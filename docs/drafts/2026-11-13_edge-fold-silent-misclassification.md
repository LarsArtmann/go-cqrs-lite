# metaengine: a fold returning []Edge is silently classified as a Set fold — non-canonical handler shapes misroute instead of failing loud

## TL;DR

An `OnRecord` handler whose single return type matches no canonical fold unit falls into the `default:` setFold branch, so `func(record.Record, E) []Edge` plans and applies "successfully" as an **ADTSet** collection keyed by the slice value — no panic, no error, and not one edge ever written. Ask: **reject slice-of-canonical-unit returns loudly at fold construction** (native `[]Edge` fan-out support can be a separate decision).

## Problem

`metaengine/v4/record_fold.go` at `5639c0d6` (verbatim, the tail of `onRecordFold`):

```go
default:
	invoke := func(rec record.Record, event any) any {
		return callWithRecord(rec, event)[0].Interface()
	}
	f := &setFold{
		recordAware: true,
		eventType:   eventType,
		sample:      sample,
		keyType:     outType,
		invoke:      invoke,
	}
	return f
```

The `default:` branch exists for a good reason — legitimate Set folds return an arbitrary key type (`func(E) K`), so unknown single-return shapes must not be rejected wholesale. But it also swallows shapes whose intent is unambiguous: a user declaring an edge-batch fold.

Probe at `5639c0d6` (throwaway test, construction only):

```text
QueryADT for []Edge fold = set (expected by a user: Graph)
```

Behavioral evidence on the released pin `metaengine/v4@v4.23.0` (throwaway spike in a consumer, 2026-11-13): a query with a `[]Edge` fold constructed, planned onto the sqlite engine, and `Apply` returned nil — with **zero rows** ever landing in `meta_graph_edges`. The writes go to the misclassified Set collection as `[]Edge` slice keys; a rebuild replays the same wrong shape; every stage reports success.

Same class for any `[]T` where `T` is a canonical fold unit: `[]Embedding` (vector intent → Set), `[]MultiEntry`, `[]EdgeRemoval`, `[]IndexedText`.

## Why it matters

This is silent data misrouting, not a missing feature: the projection content is wrong while construction, planning, apply, and rebuild all report success. A developer writing the obvious batch-edges fold (one event carrying N relations is a normal domain shape) gets a Set collection of slice-keys, and the first visible failure is far downstream — a traversal query that returns nothing, or a `doctor`-style verify that sees a Set where a graph was declared. The existing unsupported-signature panics in the same file (`"unsupported handler signature"`) show the intended contract is loud rejection; the `default:` branch is the one path that violates it.

## Ask

**Reject at fold construction** — panic like the existing signature panics — when `outType` is a slice whose element type is a canonical fold unit (`Edge`, `EdgeRemoval`, `Embedding`, `MultiEntry`, `Append`, `IndexedText`, `Point`, `Delta`, `Skip`), with an error naming the two working shapes:

- one event per relation at the write side (replay-exact, journal grows), or
- fold the batch into a Map/Multimap collection and derive edges in a second pass.

Custom key types for Set folds keep flowing through `default:` unchanged. Native multi-value fan-out (`[]Edge` → N `GraphAddEdge`) is the obviously nicer endpoint but is a separate design decision; the loud rejection is the minimal fix for the silent misclassification.

## Evidence index

- Classification: probe test output above, `metaengine/v4/record_fold.go` `default:` branch at `5639c0d6` (2026-11-13).
- Pin behavior: consumer spike against `metaengine/v4@v4.23.0` + `sqliteengine/v4@v4.8.0` — apply nil, `meta_graph_edges` count 0 for the collection, log preserved in the consumer's session record (Code-To-CV-Agent, docs/status/2026-11-13_15-07 addendum).

💘 Generated with Crush
