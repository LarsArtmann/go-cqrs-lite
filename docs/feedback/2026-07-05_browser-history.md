# Consumer Feedback — go-cqrs-lite

**From:** browser-history project (github.com/larsartmann/browser-history)
**Date:** 2026-07-05
**Versions used:** event v3.4, command/query v3.4, decider v3.1/v3.3, storage v3.3, projection v3.4, watermill v3.3
**Consumer:** Crush (AI assistant) + Lars

---

## What Works Great

### The Decider pattern (pure Fold + Decide functions)

`decider.Decider[State]{Initial: ..., Apply: Fold}` with pure functions is the cleanest CQRS/ES design I've used. Testability is excellent — Fold and Decide are pure, no I/O, no mocks needed.

### `event.New()` / `event.NewEvents()` — auto-encoding

Pass `any` payload, get an event with correct encoding. No manual `codec.Encode()` step. This is the right API. We migrated all test helpers to this pattern and removed 60 lines of boilerplate.

### `event.DecodePayloadAuto[T](evt)` — encoding-agnostic decoding

Auto-detects encoding from `evt.Encoding()` field. Beautiful — it means we can switch codecs without changing decode call sites. This is the correct design.

### Copy-on-write DefaultCodec

`event.DefaultCodec` is a `var` (not `const`), so consumers can swap it at startup: `event.DefaultCodec = codec.CBORCodec{}`. Clean global configuration without dependency injection complexity.

### Event causality enrichment

`WithCommandCausality(ctx, cmdType, cmdID)` + `CommandCausalityEnricher` → events automatically carry causation metadata. Two lines of setup, full traceability. Excellent.

---

## Pain Points

### 1. **CRITICAL: watermill EventBus loses payload_encoding during round-trip**

**This is the #1 bug I encountered.** The watermill `EventBus` serializes events via `EventToMessage` / `MessageToEvent`. During this conversion, the `payload_encoding` field (JSON vs CBOR) is **lost** — it's not included in the Watermill message metadata.

**Impact:** CBOR codec is completely unusable with the watermill bus. Events created with CBOR arrive at the projection with empty encoding, `DecodePayloadAuto` defaults to JSON, and decoding fails with `"invalid character 'ª'"`.

**Root cause:** `EventToMessage` converts event → Watermill message (UUID + bytes + metadata map), but doesn't put encoding into metadata. `MessageToEvent` reconstructs the event but has no encoding to set.

**Fix (estimated 5 lines):**

```go
// In EventToMessage:
msg.Metadata.Set("payload_encoding", string(evt.Encoding()))

// In MessageToEvent:
enc := codec.Encoding(msg.Metadata.Get("payload_encoding"))
// set enc on the reconstructed event
```

**Severity:** Blocks the #1 recommended optimization from the SKILL.md (CBOR: 19% smaller, 32% faster). This makes `DecodePayloadAuto` fundamentally incompatible with the watermill bus for non-JSON codecs.

### 2. The eventtest pseudo-version dependency hell

**Problem:** `event/v3/eventtest` go.mod requires `event/v3`, `id/v3`, `snapshot/v3` at pseudo-versions (`v0.0.0-00010101000000-000000000000`). These resolve via module-level `replace` directives within the go-cqrs-lite repo. But when a **consumer** workspace replaces eventtest, Go's workspace replaces **override** module-level replaces — they don't inherit them.

**Impact:** Consumer workspaces need 14 `replace` directives to use go-cqrs-lite:

```go.work
replace (
    github.com/larsartmann/go-cqrs-lite/codec/v3 => ../go-cqrs-lite/codec
    github.com/larsartmann/go-cqrs-lite/command/v3 => ../go-cqrs-lite/command
    // ...12 more
)
```

**This took 3 iterations to get right** (path changed from `event/eventtest` to `event/v3/eventtest`, needed transitive deps for all 14 modules).

**Suggestion:** Consider either:

- (a) Publish eventtest as a real tagged module (not pseudo-version)
- (b) Document the full set of consumer-side replaces needed
- (c) Make eventtest a `//go:build test` package within event/v3 instead of a separate module

### 3. `WithEnricher` can't infer the type parameter

**Problem:** `decider.WithEnricher(event.CommandCausalityEnricher)` fails with "cannot infer State". The enricher function type `event.ContextEnricher` doesn't carry State information, so the compiler can't infer `T`.

**Workaround:** Explicit type parameter: `decider.WithEnricher[aggregate.BrowserHistoryState](event.CommandCausalityEnricher)`.

**Suggestion:** This is a Go generics limitation, not a design flaw. But documenting the explicit type parameter in the example would save 5 minutes of confusion.

### 4. `event.New()` rejects nil payloads — silently inconsistent with `event.NewEvent()`

**Problem:** `event.New(eventType, aggID, aggType, version, nil)` returns an error. `event.NewEvent(eventType, aggID, aggType, version, []byte{})` works fine. In tests, we need to test the "empty payload" error path, but `event.New` won't let us create such an event.

**Impact:** `TestFold_EmptyPayload` has to use `event.NewEvent` directly instead of `event.New`, which is inconsistent with all other test helpers.

**Suggestion:** Document this behavior difference prominently, or add `event.NewRaw()` as a clearer name for the `[]byte`-accepting constructor.

### 5. projectionhost integration story is unclear

**Problem:** projectionhost provides per-projection goroutines, crash-restart, DLQ, backoff. But it's a **catch-up processor** — it drains the SeekableJournal then stops. For live events, the docs say "pair it with CatchUpSubscriber."

**Confusion:** How does CatchUpSubscriber integrate with an existing watermill EventBus? Do I replace the bus? Run both? The example in `example/projectionhost/main.go` seeds events before starting the host — it doesn't show live delivery.

**Impact:** We deferred the projectionhost migration (T11) because the integration path was too uncertain to risk in a working system.

**Suggestion:** Add a recipe showing: existing EventBus + projectionhost + CatchUpSubscriber coexisting. Show the wiring, not just the theory.

### 6. decider version fragmentation

**Problem:** Different modules resolve different decider versions:

- `domain/` and `extraction/`: decider v3.1.0 (pinned due to eventtest)
- `api/` and `cmd/server/`: decider v3.3.0 (transitive via usermgmt v3.3.0)

This works via go.work replaces but is fragile and confusing.

**Suggestion:** Once eventtest is properly published/tagged, all modules should converge on the latest v3.x.

---

## Minor Notes

- **`event.NewEvents()` API is excellent** — takes `[]any` payloads, auto-encodes each. Clean batch creation.
- **`command.BasicCommand` embed** — provides `ID()`, `Type()`, `AggregateID()` for free. Good default implementation.
- **`query.RegisterTyped` / `query.DispatchTyped`** — type-safe query dispatch without runtime type assertions. Nice.
- **The 28-module structure is overwhelming at first** but actually makes dependency boundaries very clear once you learn it.

---

## Summary

go-cqrs-lite is a **powerful, well-designed CQRS/ES framework** with the best decider pattern I've used. The #1 issue by far is the **watermill encoding metadata loss** — it blocks CBOR and undermines `DecodePayloadAuto`. The #2 issue is the eventtest consumer workspace complexity. Fix those two and the DX is excellent.

---

## Appendix: Session Response (2026-07-05)

> Tracking which feedback items were addressed. See `docs/status/2026-07-05_05-14_consumer-feedback-execution.md`.

### Pain Points

| #   | Feedback Item                                             | Status             | What changed                                                                                                                                                                                                                                                                                         |
| --- | --------------------------------------------------------- | ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **CRITICAL: watermill EventBus loses `payload_encoding`** | ✅ **FIXED**       | `watermill/protocol.go` now writes `payload_encoding` to message metadata in `EventToMessage` and restores it via `event.WithEncoding()` in `MessageToEvent`. Added 2 tests: JSON+CBOR round-trip preservation, backward compat (old messages without encoding → JSON default). Golden test updated. |
| 2   | eventtest pseudo-version dependency hell                  | ✅ **Documented**  | Added to skill FAQ with consumer-side `replace` directive. Structural fix (tag/restructure) is a maintainer decision.                                                                                                                                                                                |
| 3   | `WithEnricher` can't infer type parameter                 | ✅ **Documented**  | Added to skill FAQ: `decider.WithEnricher[UserState](event.CommandCausalityEnricher)` with explicit type param.                                                                                                                                                                                      |
| 4   | `event.New()` rejects nil payloads                        | ✅ **Documented**  | Added to skill FAQ: explains `New()` validates (typed constructor) vs `NewEvent()` accepts raw `[]byte` (low-level).                                                                                                                                                                                 |
| 5   | projectionhost integration story unclear                  | ✅ **Documented**  | Added full integration recipe to `references/advanced.md`: EventBus + projectionhost + CatchUpSubscriber coexisting with wiring example.                                                                                                                                                             |
| 6   | decider version fragmentation                             | ❌ **Not started** | Blocked by eventtest module resolution. Will resolve once eventtest is tagged.                                                                                                                                                                                                                       |
