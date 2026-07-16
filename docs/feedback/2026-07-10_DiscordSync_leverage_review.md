# go-cqrs-lite — Consumer Feedback Round 2 (DiscordSync)

<!-- historical-artifact-banner -->

> **Historical session artifact.** This is a point-in-time snapshot from a past
> session. Many items marked TODO / Open / Not Started / Broken have since been
> resolved. See [CHANGELOG.md](../../CHANGELOG.md) and
> [TODO_LIST.md](../../TODO_LIST.md) for current state.
> Last documentation health audit: 2026-07-16.

**Consumer:** [DiscordSync](https://github.com/LarsArtmann/DiscordSync) — Discord backup bot
**Version used:** commit `f9e0e0bb` (between v3.7.3 and v3.7.4, 15 direct module imports)
**Previous feedback:** [2026-07-05_DiscordSync.md](./2026-07-05_DiscordSync.md) at v3.5.0
**Date:** 2026-07-10
**Context:** Deep leverage review of all 24 library modules against DiscordSync's usage. Full ADR at [DiscordSync ADR-012](https://github.com/LarsArtmann/DiscordSync/blob/master/docs/adr/ADR-012-go-cqrs-lite-leverage-review.md).

---

## Executive Summary

Since the last review, the library has grown from v3.5.0 to v3.7.x. Several
items from the previous report were addressed (projection lag metric shipped,
shared-DB recipe added, projectionhost lifecycle documented). This report
covers **new findings** discovered during a module-by-module source audit of
the vendored library code.

The headline: **three structural gaps force consumers to hand-roll ~570 lines
of code that belong in the library.** Each gap has a concrete, scoped fix.

---

## Gap 1: No `VersionedSeekableJournal` (HIGH priority)

### The problem

`schema.NewVersionedStore(store, upcasters...)` wraps `event.Store` with
upcaster support. It implements `Load`, `LoadFromVersion`, `LoadToVersion`,
`LoadToTimestamp` — all the `EventSource` methods.

But `projectionhost.New()` takes `event.SeekableJournal` (which adds
`ReadAll` + `ReadFrom`), NOT `event.Store`. There is no library type that
wraps a `SeekableJournal` with upcasters.

### The impact

DiscordSync hand-rolls `VersionedJournal` (50 lines +
92 lines of upcaster application logic that duplicates the library's internal
`upcasterRegistry`). This is the code that feeds the projection host:

```go
// DiscordSync: internal/eventschema/versioned_journal.go
func (j *VersionedJournal) ReadFrom(ctx, afterEventID, limit) ([]Event, error) {
    events, err := j.inner.ReadFrom(ctx, afterEventID, limit)
    // ... hand-rolled upcaster application that duplicates schema.upcasterRegistry
}
```

Every consumer using upcasters + projectionhost faces the same gap. The
cross-consumer integration gaps report already identifies this as the
"breaking field renames" risk path — the fix path is upcasters, but the
plumbing to connect upcasters to the projection host doesn't exist.

### Suggested fix

```go
// schema/versioned_seekable_journal.go
func NewVersionedSeekableJournal(journal event.SeekableJournal, upcasters ...Upcaster) *VersionedSeekableJournal
```

The implementation mirrors `VersionedStore` exactly — wrap inner reads,
apply upcaster registry. ~40 lines of code. The projection host becomes
the primary beneficiary.

### Effort estimate

~1 hour. The `VersionedStore` pattern is directly transferable. The
`upcasterRegistry` already handles grouping by type + version.

---

## Gap 2: No payload transform hook on `transport/http/v4.SSEBroker` (HIGH priority)

### The problem

`transport/http/v4.SSEBroker` is the library's built-in SSE solution with
rich features the cqrs-htmx alternative lacks: byte-budget replay, replay
timeout, per-broker event filter, OTel replay metrics, dedup ring tuning.

But `SSEHandler` sends `event.PayloadReadOnly(evt)` as raw bytes:

```go
// vendor transport/http/v4/sse.go line 227
_ = WriteSSEEvent(w, SSEEvent{
    Data: string(event.PayloadReadOnly(evt)),
})
```

Any consumer using CBOR as the default codec (which the library recommends)
gets raw CBOR bytes on the SSE wire. Browsers can't parse CBOR.

### The impact

DiscordSync uses cqrs-htmx's SSE infrastructure instead of the library's
SSEBroker specifically because cqrs-htmx allows a custom transform function.
The hand-rolled CBOR→JSON transcoding layer is 67 lines
(`sseCBORDecMode` + `jsonPayloadForSSE`) within a 239-line SSE file.

This means DiscordSync **cannot adopt** the library's richer SSE features:

- `WithReplayByteBudget` — protects against replay memory bombs
- `WithReplayTimeout` — prevents handler starvation
- `WithEventFilter` — broker-level event type filtering
- `WithReplayMetrics` — OTel replay observability
- `WithDedupRingCapacity` — memory tuning

### Suggested fix

```go
// transport/http/v4/sse_options.go
func WithPayloadTransform(fn func(event.Event) []byte) SSEBrokerOption
```

Applied in `SSEHandler` before `WriteSSEEvent`:

```go
data := event.PayloadReadOnly(evt)
if b.eventTransform != nil {
    data = b.eventTransform(evt)
}
_ = WriteSSEEvent(w, SSEEvent{Data: string(data)})
```

Default: identity (raw payload — backward compatible). Consumers using CBOR
pass a transcoded.

### Effort estimate

~30 minutes. One option field + 3 lines in `SSEHandler` + test.

---

## Gap 3: No SQLite `projectionhost.DeadLetterStore` (MEDIUM priority)

### The problem

`projectionhost/v4` defines the `DeadLetterStore` interface and ships only
`MemoryDeadLetterStore` (in-memory, lost on restart). The `middleware/v4`
package ships `SQLDeadLetterStore` — but it implements `DeadLetterHandler`
(a function type), not `projectionhost.DeadLetterStore` (an interface with
`Store`/`List`/`Delete`/`Purge`).

### The impact

DiscordSync hand-rolls `SQLiteDeadLetterStore` (226 lines) implementing
`projectionhost.DeadLetterStore`. Every SQLite-based projection host user
needs the same thing.

### Two-part fix

**Part A (quick):** Ship `projectionhost.NewSQLiteDeadLetterStore(db)` in the
projectionhost package. DiscordSync's implementation is a direct reference —
table schema, CRUD queries, `storableEvent` serialization pattern.

**Part B (structural):** The two `DeadLetterEntry` types
(`projectionhost.DeadLetterEntry` vs `middleware.DeadLetterEntry`) have
different fields and different interface shapes. This is the deeper problem.

The middleware version has `Kind`/`Type`/`AggregateID` (generic over
command/event/query). The projectionhost version has `ProjectionName`/
`EventID`/`EventType`/`Event` (projection-specific, includes the original
event for replay). They can't share a `SQLDeadLetterStore`.

**Recommendation for Part B:** Don't force unification — the two serve
different purposes (dispatch-side retry exhaustion vs projection poison
messages). But document why they're separate so consumers don't try to
reuse one for the other.

### Effort estimate

Part A: ~2 hours (port DiscordSync's implementation, add tests).
Part B: Documentation only — ~15 minutes.

---

## Gap 4: `otel.Setup()` + `prometheus.Setup()` meter provider overlap (LOW priority)

### The problem

`otel.Setup()` creates a `metric.MeterProvider` with CQRS-optimized views
and calls `otel.SetMeterProvider(mp)`. `prometheus.Setup()` ALSO creates a
`metric.MeterProvider` with a Prometheus reader.

A consumer who wants BOTH tracing (via `otel.Setup`) and Prometheus metrics
(via `prometheus.Setup`) must call both, then re-set the global meter
provider:

```go
tracingProvider, _ := cqrsotel.Setup(cqrsotel.WithService(...))
metricsProvider, _ := cqrsprometheus.Setup(...)
otel.SetMeterProvider(metricsProvider.AsMeterProvider()) // overrides otel.Setup's noop-ish meter
```

The `otel.Setup()` meter provider is silently discarded. This works but
feels accidental. The CQRS views from `otel.Setup()` are lost (though
`prometheus.Setup` doesn't apply them either).

### Suggested fix

Option A: Add `cqrsotel.WithMetricReader(reader)` to `prometheus.Setup` so
it accepts an existing reader. Wait — this already exists on `otel.Setup`
but not on `prometheus.Setup`.

Option B (simpler): Document the intended composition pattern in the
`prometheus/v4` package doc: "Call `otel.Setup()` first for tracing, then
override the meter provider with `prometheus.Setup()`. The CQRS views from
`otel.Setup()` are not lost — pass them to `prometheus.Setup` via
`WithMetricReader`."

Wait — `prometheus.Setup` doesn't accept views. So Option C: Add
`WithViews([]sdkmetric.View)` to `prometheus.Setup` so consumers can pass
`cqrsotel.NewCQRSViews()` when composing the two.

### Effort estimate

Option C: ~20 minutes (one option field + plumbing).

---

## Gap 5: `event.PayloadReadOnly` naming (LOW priority, nitpick)

### The problem

`event.PayloadReadOnly(evt)` returns the raw payload bytes without cloning.
The name says "read-only" from the caller's perspective (don't modify the
returned slice), but it's actually just accessing the internal field. Every
call site that needs payload bytes for non-decode purposes (SSE, catalog
generation, size diagnostics) uses this function.

The name is slightly confusing because it implies there's a "writable"
variant, which there isn't (`evt.Payload()` returns a clone).

### Suggested fix

Rename to `PayloadBytes` or `RawPayload` in v4. Document that the returned
slice must not be modified. No urgency — just a v4 batch item.

---

## What's Improved Since v3.5.0 (positive feedback)

These items from the previous report or general observations are now resolved:

| Item                                         | Status     | Evidence                                                                                                         |
| -------------------------------------------- | ---------- | ---------------------------------------------------------------------------------------------------------------- |
| Projection lag metric                        | ✅ Shipped | `projectionhost.Host.LagDuration()` — DiscordSync uses it for the dashboard gauge                                |
| Shared-DB recipe                             | ✅ Shipped | Skill `references/recipes.md` §2.0                                                                               |
| Projectionhost lifecycle docs                | ✅ Shipped | Full replay→live→DLQ sequence documented                                                                         |
| `WithOnFailed` callback                      | ✅ Shipped | DiscordSync now wires it for terminal failure logging                                                            |
| `otel.Setup()` one-call tracing              | ✅ Shipped | DiscordSync uses it — discovered it was missing from our production wiring during this review                    |
| `WithReplayByteBudget` / `WithReplayTimeout` | ✅ Shipped | Rich SSE replay controls (we just can't adopt them yet — see Gap 2)                                              |
| `OTelBundle`                                 | ✅ Shipped | Pre-wired tracing+metrics for all message kinds (we don't use it for event-only reasons, but it's well-designed) |
| Schema upcaster infrastructure               | ✅ Shipped | `schema.Upcaster` + `VersionedStore` + registry with cycle detection                                             |

---

## What Still Works Superbly

### 1. `DecodePayloadAuto[T]` — still the best codec API

Encoding-agnostic decoding. Mixed JSON+CBOR streams during migration work
flawlessly. DiscordSync's projection builder uses this everywhere:

```go
On(b, events.MessageCreated, func(ctx, db, evt) error {
    payload, err := event.DecodePayloadAuto[events.MessageCreatedPayload](evt)
```

No codec parameter. No encoding check. Just decode. This is exemplary API
design.

### 2. CBOR codec adoption — zero-friction migration

Set `event.DefaultCodec = codec.CBORCodec{}` in an `init()` — all new
events are CBOR. Historical JSON events decode transparently. 19% smaller
payloads, 66% faster decode. The migration was invisible to every consumer
of the event store.

### 3. ISP-respecting interface splits

`EventSink` (write) / `EventSource` (read) / `Store` (both) /
`Journal` (ordered log) / `SeekableJournal` (position-based seek). Each
consumer imports only what it needs. Projectionhost takes
`SeekableJournal`, not `Store` — correct minimal interface.

### 4. `projectionhost.Host` — production-grade, feature-rich

`WithSubscriber`, `WithDeadLetterStore`, `WithBatchSize`, `WithMetrics`,
`WithOnFailed`, `WithMaxRestarts`, `WithBackoff`, `WithShutdownTimeout`.
Every option is useful. The crash-recovery-with-jitter actually works
under load. The DLQ integration is clean. This replaced 297 lines of
custom code and has been rock-solid.

### 5. Middleware chain ordering — documented and correct

The SDK recommends: tracing → recovery → retry → logging → metrics.
DiscordSync follows this exactly. The ordering matters (tracing first so
the span covers everything downstream) and the docs explain why.

### 6. Prometheus bridge — single `/metrics` endpoint

`prometheus.Setup(WithRegistry(prometheus.DefaultRegisterer))` — both
hand-rolled Prometheus counters AND OTel-backed instruments serve from one
registry. No duplicate metrics, no port conflicts. Simple, correct.

---

## Updated Scorecard (vs previous report)

| Dimension             | Previous | Current | Change reason                                                    |
| --------------------- | -------- | ------- | ---------------------------------------------------------------- |
| API design            | 9/10     | 9/10    | Still excellent; VersionedSeekableJournal gap is the main ding   |
| Documentation (skill) | 9/10     | 9/10    | Lifecycle docs added; VersionedSeekableJournal gap not yet noted |
| Ease of adoption      | 7/10     | 7/10    | eventtest still breaks local tidy; 28 modules still a lot        |
| Projection system     | 9/10     | 9/10    | WithOnFailed + lag metric shipped; still needs SQLite DLQ store  |
| Codec support         | 10/10    | 10/10   | CBOR + JSON + auto-detection is still exemplary                  |
| Observability         | 9/10     | 9/10    | otel.Setup is great; SSEBroker can't do CBOR→JSON yet            |
| SSE/transport         | —        | 8/10    | Rich features but no payload transform hook                      |
| Overall               | **8.5**  | **8.5** | Gaps shifted but didn't widen; new features offset new findings  |

---

## Priority Summary

| #   | Gap                               | Priority | Effort | Consumer LOC eliminated                         |
| --- | --------------------------------- | -------- | ------ | ----------------------------------------------- |
| 1   | `VersionedSeekableJournal`        | HIGH     | ~1h    | ~142 lines                                      |
| 2   | SSEBroker `WithPayloadTransform`  | HIGH     | ~30min | ~67 lines (+enables adoption of 5 SSE features) |
| 3   | SQLite `projectionhost` DLQ store | MEDIUM   | ~2h    | ~226 lines                                      |
| 4   | otel+prometheus meter composition | LOW      | ~20min | 0 (docs/ergonomics)                             |
| 5   | `PayloadReadOnly` naming          | LOW      | v4     | 0 (nitpick)                                     |

**Total consumer code eliminatable: ~435 lines across 3 gaps, all of which
already have working reference implementations in DiscordSync.**
