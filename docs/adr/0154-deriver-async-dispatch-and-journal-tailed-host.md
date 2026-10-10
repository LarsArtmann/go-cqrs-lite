# ADR-0154: Deriver Async Dispatch Now, Journal-Tailed Deriver Host at v5

- Status: Accepted (owner granted Full Execution Mode 2026-10-09 for
  [`docs/planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md`](../planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md),
  approving recommendation Q1 as written)
- Date: 2026-10-09
- Deciders: owner (adoption-wave Gate 1), via the plan's §2 embedded recommendation
- Evidence: [`docs/evidence/2026-10-09_deriver-bus-deadlock.md`](../evidence/2026-10-09_deriver-bus-deadlock.md)
  (live-captured stacks, mechanism, option memo, error-surfacing design) +
  raw dump [`docs/status/deriver-deadlock-stacks-2026-10-09.txt`](../status/deriver-deadlock-stacks-2026-10-09.txt)
- Related: ADR-0153 (systemscenario harness — the saga test that found the deadlock),
  ADR-0136 (temporal composability — derivers are the compensable rung),
  ADR-0142 (engine-backed timers — the other Clock consumer),
  ADR-0028 (watermill EventBus as canonical bus)

## Context

The systemscenario harness's first end-to-end saga test (ADR-0153, 2026-10-09)
deadlocked the whole system boot: a **synchronous** `deriver.Deriver` wired via
`AsHandler` on `sys.Bus()` never returns. The evidence pack pinpoints the cycle
(watermill v1.5.3 GoChannel, non-persistent, `BlockPublishUntilSubscriberAck: true`):

1. Outer `Publish` holds the GoChannel **per-topic subscriber mutex** for its whole
   duration and waits for the subscriber's ack.
2. The subscriber is the EventBus's **single event-loop goroutine**, currently inside
   the deriver handler.
3. The derived command's handler emits an event; the nested `Publish` blocks acquiring
   the same per-topic mutex → the ack can never happen.

Any handler that transitively publishes to the same bus topic from the delivering
goroutine hangs forever — derivers are simply the most likely first contact because
`AsHandler` dispatches synchronously by design.

## Decision

### D1 — (b) `deriver.WithAsyncDispatch` NOW (v4.x, additive)

A new `HandlerOption` on `Deriver.AsHandler` moves derived-command dispatch off the
handler goroutine:

```go
// AsyncDispatchErrorHandler receives a failed async dispatch: the source event,
// the command, and the dispatch error. Nil-safe to ignore arguments.
type AsyncDispatchErrorHandler func(evt cqrsevent.Event, cmd cqrscommand.Command, err error)

// WithAsyncDispatch dispatches derived commands on a background goroutine
// (one per source event; that event's commands dispatch in order) instead of
// inside the bus handler. REQUIRED when the deriver subscribes on the same
// event bus its derived commands publish to through a synchronous dispatch
// path (the default watermill EventBus deadlocks otherwise — ADR-0154).
// Context: dispatches run under context.WithoutCancel of the handler context.
// Errors: invoke onError when non-nil; a nil onError DROPS dispatch errors —
// pass a handler that at least logs. Ordering: per-event order preserved;
// cross-event interleaving possible — assert outcomes, not dispatch order.
func WithAsyncDispatch(onError AsyncDispatchErrorHandler) HandlerOption
```

Semantics:

- **Per-event goroutine**: one goroutine per handled event dispatches that event's
  derived commands sequentially — per-event order is preserved; cross-event
  interleaving is possible (documented).
- **Errors**: async dispatch cannot return through the handler (the handler has
  already acked); failures go to `onError`. **`nil` onError drops errors** — a loud
  godoc warning, not a panic: a library must not panic inside a goroutine it does
  not own the lifecycle of. The promoted workaround (`go func() { _ = Dispatch() }()`)
  swallowed errors unconditionally; the option makes the swallow OPT-IN.
- **Context**: `context.WithoutCancel(ctx)` — async work must not die with the
  publish request scope, but values still flow.
- **Cycles**: async dispatch escapes `WithMaxDepth`'s synchronous depth counter;
  async derivation cycles must be bounded by `Deriver.Idempotent` (deterministic
  command IDs + idempotency store), stated in the option docs.
- Default (`WithAsyncDispatch` absent): unchanged synchronous dispatch — zero
  behavior change for existing consumers (ADR-0153's API-stability posture).

### D2 — (c) Journal-tailed deriver host as the v5 direction

The architecturally right endgame is a dedicated host (projectionhost-shaped) that
tails the **journal** (not the bus) and runs derivers as a replayable consumer:

- Survives restarts (cursor persisted; the journal is the source of truth, ADR-0136).
- Totally ordered by journal position — no cross-event interleaving caveat.
- Removes derivers from the bus delivery path entirely — the deadlock class becomes
  structurally impossible, not worked around.
- At-least-once delivery: pairs with `Deriver.Idempotent` for exactly-once effects.

This is new infrastructure (cursoring, retries, DLQ story) — v5-scale, recorded on
the ROADMAP. It is NOT built by this ADR; D1 is the v4.x bridge.

## Alternatives considered

| Option | Verdict | Why |
|---|---|---|
| (a) Async bus delivery in `watermill.EventBus` | Rejected for v4.x | Changes global ordering semantics for EVERY subscriber (the reason `BlockPublishUntilSubscriberAck` exists — ordered live delivery); silent semantic shift of a shipped bus to fix a deriver-specific hazard. May be revisited only with a delivery-mode knob (adoption-wave residue TODO). |
| (b) `deriver.WithAsyncDispatch` | **Adopted (D1)** | Opt-in, additive, ~90 min; promotes the known-safe workaround to a first-class API with error surfacing. |
| (c) Journal-tailed deriver host | **Adopted as v5 direction (D2)** | The endgame; too large for v4.x and unnecessary once (b) unblocks sagas. |
| Do nothing (document the hazard) | Rejected | The deadlock is a whole-boot hang with no stack-trace pointer at the misuse site; silent by nature (see consequences). |

## Consequences

- Sagas on the default bus become possible today: wire `WithAsyncDispatch` and assert
  outcomes asynchronously (`systemscenario.Await()` / poll-style assertions).
- The systemscenario saga fixture flips from the raw-goroutine workaround to the
  sanctioned option (adoption-wave T09) — the harness then teaches the real API.
- Independent hardening (adoption-wave T26): watermill gains reentrancy detection
  (`ErrReentrantPublish`) so the deadlock class fails loudly at the misuse site
  instead of hanging — belt and suspenders; D1 removes the need, T26 catches those
  who forget.
- Error visibility contract for async derivers is the `onError` callback — hosts
  should wire it to their error pipeline (log, DLQ, metric).

## Self-review against related ADRs

- **ADR-0136 (temporal composability)**: derivers sit on the compensable rung;
  async dispatch does not change their invertibility position, but it moves the
  dispatch EFFECT outside the synchronous dispatch call — the `onError` callback is
  the required error observability for that move. Consistent.
- **ADR-0142 (engine-backed timers)**: unrelated surface (Clock consumers); no
  interaction beyond sharing the system boot. No conflict.
- **ADR-0153 (systemscenario)**: the harness deadlock note + fixture workaround are
  superseded by this ADR's D1; the README constraint section updates at T09 to point
  here. The determinism contract is unaffected (assertions poll; async arrival was
  already the asserted reality).
- **ADR-0028 (watermill as canonical bus)**: unchanged — D1 fixes the deriver side,
  not the bus. The bus's ordered-live-delivery guarantee stays intact.
