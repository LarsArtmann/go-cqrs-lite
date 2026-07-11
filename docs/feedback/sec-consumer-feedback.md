# go-cqrs-lite — SDK Feedback from SEC

**Consumer:** [SEC](https://github.com/larsartmann/sec) — dice-based game (CQRS + event sourcing)
**Date:** 2026-07-05
**Version used:** v3.5.0 (workspace replaces)
**Session:** Full stack.Bundle adoption, Repository.Execute(), snapshot/v4, scenario/v4 BDD, idempotency/v4, catalog/v4

---

## What worked superbly

### 1. `decider.Repository.Execute()` — the golden path

One call: `repo.Execute(ctx, aggID, aggType, func(state, ver) ([]event.Event, error))`. Loads events, folds to state, calls the decide callback, saves events with optimistic concurrency, publishes to bus — all atomic. My command handlers shrank from 60+ lines of manual loadState/persistAndPublish to a single callback. This is the best abstraction in the library.

### 2. `stack.Bundle` — infrastructure wiring in one call

`stack.New(WithEventStore(store), WithBus(bus), WithSnapshotStore(snap), WithCheckpointStore(cp))` wires everything. `stack.Repository[State](bundle, decider, opts...)` creates a repository over the bundle's store+publisher. The `Close()` method handles all resource cleanup via registered closers. Replaced my manual wiring cleanly.

### 3. `stack/memory.New()` preset

For dev/test, `memory.New()` returns a fully-wired bundle (store, bus, snapshot, checkpoint, read-models). Zero config. Perfect for `NewCQRSApp()`.

### 4. `command.RegisterTyped` / `query.RegisterTyped`

Typed handlers eliminate runtime type assertions. `command.RegisterTyped(d, seccmd.PlayRound, h.handlePlayRound)` where `handlePlayRound` is `func(ctx, *PlayRoundCmd) error`. Compile-time safety, no `cmd.Command.(type)` boilerplate.

### 5. `scenario.Given().When().Then()` / `ThenError()`

The BDD DSL for testing deciders is excellent. `scenario.Given[any, State](t, applyEvent, initial).When(nil, decideFn).Then(eventType)` reads like English. `ThenError(ErrGameNotFound)` asserts the error directly. I wrote 15 scenario tests across game and islands domains with this DSL.

### 6. `event.NewEvents()` with auto-incrementing versions

`event.NewEvents(aggID, aggType, baseVersion, []event.Type, []any)` batch-creates events with auto-incrementing versions. No manual `version+1` arithmetic. Payloads auto-marshal to JSON.

### 7. Snapshot support (`WithSnapshotStore`, `WithCodec`, `WithSnapshotStrategy`)

`EveryNEvents(50)` snapshots every 50 events. `repo.Load()` uses snapshots when available. The snapshot round-trip test I wrote confirmed byte-identical state between snapshot path and full replay. Transparent to the consumer — just add 3 options to the repository constructor.

### 8. `middleware.CommandIdempotency` middleware

Wire `MemoryStore` + `middleware.CommandIdempotency(idemStore, ttl, nil)` on the dispatcher. Deduplicates by command ID. Works correctly with `X-Command-Id` header from frontend retries.

### 9. `query.DispatchTyped[T](ctx, dispatcher, query)`

Typed query dispatch eliminates `result.(MyType)` assertions at call sites. `DispatchQueryTyped[GameState](app, ctx, q)` returns `(GameState, error)` directly.

### 10. `catalog` auto-documentation

`catalog.Command[T]("game.create", WithSummary("..."))` auto-derives JSON schemas from Go types. The catalog generates an event catalog document that serves as living API documentation. Adding new commands/events to the catalog is a one-liner per message.

---

## Pain points and friction

### 1. `stack.Bundle.eventStore()` is unexported

The `eventStore()` method on `Bundle` is unexported (lowercase). This means I can't access the bundle's `event.Store` directly for things like `NewIslandQueryHandler(store, journal)` which takes a raw `event.Store`. I had to keep a separate reference to the store I passed into `stack.New()`.

**Impact:** `CQRSApp.Store` is still set from the `store` parameter, not from `bundle.eventStore()`. The bundle and the struct both hold the same store reference, which is fine but redundant.

**Suggestion:** Export `Bundle.EventStore() (event.Store, bool)` or add `Bundle.Store` as a public field. Consumers frequently need the raw store for query handlers, journal access, or SSE broker registration.

### 2. `event.Store` vs `event.EventSink` / `event.EventSource` segregation

The Bundle stores `EventSink` and `EventSource` separately. `event.Store` is the composite interface. The `stack.Repository` accessor recovers it via `b.EventSink.(event.Store)` type assertion. This works when `WithEventStore` was used (common case) but fails silently when `WithEventSink` was used with a write-only sink.

**Impact:** No real impact on SEC (I always use `WithEventStore`), but the type assertion pattern feels fragile. If a future preset uses a split sink/source, the repository constructor would fail at runtime instead of at wiring time.

**Suggestion:** Consider storing `event.Store` directly on the Bundle when `WithEventStore` is called, as a typed field rather than recovering it via assertion.

### 3. `foldIslandEvent` must live in the `app` package, not `islands` package

The `decider.Repository.Apply` function is called by the repository internally. For the game decider, I use `cqrsshared.FoldEvent(state, evt)` from the shared package. For islands, I had to define `foldIslandEvent` in `internal/cqrs/app/island_handler.go` because it needs to decode event types using `codec.JSONCodec` and the islands event types are in `modules/islands/`.

This is a Go packaging issue, not a go-cqrs-lite issue per se, but the `Apply` function signature (`func(State, event.Event) (State, error)`) forces the fold logic to be where the event types are known — which is the `app` package, not the domain package. The domain package works with its own `IslandEvent` type, not `event.Event`.

**Impact:** Fold logic is split across packages: game fold in `shared/`, island fold in `app/`.

**Suggestion:** Consider a `Codec` interface on the decider that handles the `event.Event → domain event` conversion automatically. The consumer provides `func(state State, domainEvent DomainEvent) (State, error)` and the framework handles decoding. This would let fold logic live in the domain package.

### 4. `scenario.Given[any, State]` — the first type parameter is unused

The `scenario.Given[Command, State]` generic takes a command type parameter that I never use — I always pass `any` and use `nil` as the command. The `When` callback receives `(state, command)` but the command is always `nil` because the decide function is called inline.

**Impact:** Visual noise. `scenario.Given[any, GameState](...)` appears in every test. The `any` adds no information.

**Suggestion:** Consider a `scenario.GivenState[State](...)` variant that doesn't require the command type parameter, or make the command parameter inferable from the `When` callback.

### 5. `event.Version` is `uint64` — no protection against underflow

Optimistic concurrency uses `event.Version` which is `uint64`. If a bug causes `version - 1` on a zero-value version, it wraps to `MaxUint64`. This hasn't bitten me, but it's a theoretical concern.

**Suggestion:** Consider a `Version` type with safe arithmetic, or document that zero-value version must never be decremented.

### 6. Module proliferation — 13+ submodules

The library has event, command, query, decider, snapshot, codec, idempotency, storage/memory, watermill, catalog, scenario, stack, projection, otel, id, schema, signing, encryption, kv... Importing the right combination requires knowing the module map. The skill file's "module decision matrix" is essential.

**Impact:** Every new file needs 5-8 import lines from different submodules. Go workspace replaces make this manageable, but adding a new submodule dependency requires a `go get` + potentially a `replace` directive.

**Suggestion:** This is inherent to the library's design (which is correct — consumers should only depend on what they use). But consider a `go-cqrs-lite/all` meta-module that re-exports the common types for prototyping. Or better: keep as-is but ensure the skill docs have a "minimal imports for event sourcing" cheat sheet.

---

## Ideas for improvement

### 1. `stack.Bundle.EventStore()` accessor

Export a typed `EventStore()` method so consumers can access the store for query handlers without keeping a separate reference.

### 2. Domain-aware fold helper

```go
decider.Decider[State]{
    Initial: InitialState(),
    ApplyDomain: func(state State, evt DomainEvent) (State, error), // typed domain event
    EventDecoder: codec.JSONCodec{},
}
```

The framework decodes `event.Event` → `DomainEvent`, then calls `ApplyDomain`. Consumer never touches `event.Event` or `event.DecodePayload`.

### 3. `scenario.GivenState[State]` variant

Remove the unused command type parameter for the common case where the decide function is called inline in `When`.

### 4. Bundle `Debug()` in production

`Bundle.Debug()` prints capability status (✓/✗ for each field). Consider a `DebugStructured() map[string]bool` variant for programmatic checks (e.g., health endpoint).

### 5. Idempotency: content-hash key mode

I tested content-hash dedup and rejected it because `PlayRound` isn't idempotent. But for commands that ARE idempotent (`CreateGame`, `StartRun`), content-hash dedup would be useful. Consider a custom key extractor for content-hash dedup.

---

## Overall verdict

go-cqrs-lite v3.5.0 is the best CQRS/event-sourcing framework I've used in Go. The `Repository.Execute()` golden path is exactly right — one call handles load→fold→decide→save→publish with OTel tracing and causality enrichment. The `stack.Bundle` presets eliminate infrastructure boilerplate. The `scenario` BDD DSL makes testing deciders genuinely enjoyable.

The module proliferation is inherent to the design principle (pay-for-use), and the skill docs compensate well. The main gaps are minor: exporting `Bundle.EventStore()`, domain-aware fold helpers, and simplifying the scenario DSL's type parameters.

The library has clearly evolved through real-world use — the API shows evidence of lessons learned (version is uint64, snapshots are transparent, idempotency is middleware-based, the Bundle uses interface segregation). This is mature, production-grade infrastructure.

---

## Appendix: Session Response (2026-07-05)

> Tracking which feedback items were addressed. See `docs/status/2026-07-05_05-14_consumer-feedback-execution.md`.

### Pain Points

| #   | Feedback Item                                                         | Status             | What changed                                                                                                                                                                                     |
| --- | --------------------------------------------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | `stack.Bundle.eventStore()` is unexported                             | ✅ **SHIPPED**     | Exported `Bundle.EventStore() (event.Store, bool)`. Consumers can now access the raw store for query handlers, journal access, and SSE broker registration without keeping a separate reference. |
| 2   | `event.Store` vs `EventSink`/`EventSource` — type assertion fragility | ✅ **Documented**  | The `EventStore()` accessor handles the assertion internally. The typed field suggestion is noted but would change the Bundle struct shape.                                                      |
| 3   | `foldIslandEvent` must live in app package, not domain package        | ❌ **Not started** | Domain-aware fold helper (`ApplyDomain` + `EventDecoder`) noted as P2. Would let fold logic live where domain types are defined.                                                                 |
| 4   | `scenario.Given[any, State]` — unused first type parameter            | ✅ **SHIPPED**     | Added `scenario.GivenState[State]()` — eliminates the redundant `[any]` type parameter for the common case where Cmd is unused.                                                                  |
| 5   | `event.Version` is `uint64` — no underflow protection                 | ✅ **Documented**  | Noted in status report as P3: a `Version` type with safe arithmetic, or documenting that zero-value version must never be decremented.                                                           |
| 6   | Module proliferation — 13+ submodules                                 | ✅ **Documented**  | Inherent to the pay-for-use design. Skill docs compensate. Bundle meta-module noted as P3.                                                                                                       |

### Ideas for Improvement

| #   | Feedback Item                                      | Status             | What changed                                                                                                |
| --- | -------------------------------------------------- | ------------------ | ----------------------------------------------------------------------------------------------------------- |
| 1   | `stack.Bundle.EventStore()` accessor               | ✅ **SHIPPED**     | See pain point #1 above.                                                                                    |
| 2   | Domain-aware fold helper                           | ❌ **Not started** | Noted as P2 feature.                                                                                        |
| 3   | `scenario.GivenState[State]` variant               | ✅ **SHIPPED**     | See pain point #4 above.                                                                                    |
| 4   | Bundle `DebugStructured()` for programmatic checks | ✅ **SHIPPED**     | Added `Bundle.DebugStructured() map[string]bool` — returns capability status as a map for health endpoints. |
| 5   | Idempotency content-hash mode                      | ❌ **Not started** | Noted as P2. `Idempotency(ContentHashKey)` selector for commands that ARE idempotent.                       |
