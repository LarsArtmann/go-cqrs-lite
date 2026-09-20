> **RESOLVED — docs-health 9th pass (2026-09-20):** the Pareto list below was executed —
> items 1–2 shipped 2026-09-13 (`decider.ExecuteCommandRef` + `CausedCommand` causation
> stamping; commandlifecycle upcast composition pinned), and the release train published the
> decider/command/commandlifecycle surfaces (92-tag wave, 2026-09-19). Item 3 is routed to the
> [BLOCKED] ADR-0138 row in [TODO_LIST.md](../../../TODO_LIST.md). Archived as a completed review.

# Review: Is the Command side taken seriously?

> **Date:** 2026-09-13
> **Kind:** Point-in-time maturity review (no code changes)
> **Repo state:** `master` @ 0711ef9e7
> **Prompt:** "I feel like this project is taking the Command side not serious enough."
> **Verdict up front:** The feeling is half right. The command side is NOT neglected
> operationally — it is **shallow architecturally**. It received transport/persistence
> parity, but never the domain-side depth the event side has.
> **Companion reviews:** [event/ split re-review](2026-09-13_event-module-split-re-review.md) ·
> [event/command duplication hypothesis](archived/2026-09-13_event-command-duplication-hypothesis-review.md)
> (the `AsRecord` wart found there is the same root cause as Gap 1 below)

---

## 1. The steelman: where the command side IS solid

| Capability   | Evidence                                                                                                                                                                                                                       |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Type parity  | `command/` mirrors `event/` structurally: 89 exported symbols — `Dispatcher`, `Bus`, `MemoryBus`, `Store` (Sink/Source ISP split), `CommandJournal`, `TypedCommandStore[P]`, `Middleware`, `MetadataCarrier` capability        |
| Lifecycle    | `commandlifecycle/` (ADR-0117): received/failed/retried/completed/deadlettered as **replayable event streams** + `Recorder` + `system.WithCommandLifecycle` one-call setup — more sophisticated than most CQRS libraries' DLQs |
| Idempotency  | `idempotency/` + `idempotency/sqlstore` + `idempotency/kvstore` + `middleware.CommandIdempotency`                                                                                                                              |
| Testing      | `command/commandtest` store suite                                                                                                                                                                                              |
| Docs/tooling | catalog covers commands (`catalog/build.go`); `system.RegisterCommand` / `UseCommandMiddleware` / `CommandStore` / `CommandDispatcher`                                                                                         |
| Activity     | Authored commits since 2026-06-15: command 173 vs event 248 vs query 150 — comparable, not abandoned                                                                                                                           |

## 2. Where the feeling is justified: three domain-side gaps

### Gap 1 — The Decider has no Decide (structural second-class-ness)

[`decider/decider.go:99`](../../decider/decider.go):

```go
// DecideFunc is the signature for a decision function.
// ... Return an error to reject the command (no events will be saved).
type DecideFunc[State any] func(state State, currentVersion event.Version) ([]event.Event, error)
```

The doc comment knows decide is about commands — yet **the signature does not receive the
Command**. The command is closed over in the consumer's handler, invisible to the
abstraction. Meanwhile `Decider[State]` = `Initial` + `Apply` only: the type named after
decision-making contains only the evolve half. Compare the event side: `Apply(state, evt)`
is fully typed, property-tested, snapshot-aware, schema-evolvable. The command side gets
an anonymous closure. (See also the duplication review: commands are likewise second-class
`record.Record`s.)

### Gap 2 — No command schema evolution

`schema/` contains **zero** command references. Commands ARE persisted — audit
(`command.Store`), `TypedCommandStore`, and `commandlifecycle` event streams replayed to
rebuild DLQ projections — yet old command payloads decode with today's codec only. Events
got `schema.UpcastSourceTransform` and 8 event-side ADRs; commands got 1 ADR total (0117).

### Gap 3 — No automatic command→event causation

`Repository.ExecuteRef` never stamps `event.Causation` from the triggering command;
correlation is consumer-side via `ContextEnricher` (context-derived metadata only). The
cmd→event lineage — the audit spine of event-sourced systems — is not first-class.

## 3. The softer tells

- ROADMAP explicitly lists **ADR-0112 (command sourcing — fold over command history) as
  planned-not-built**.
- cqrs-lint: 87 rule files reference `event.`, 18 reference `command.`.
- Skill docs run ~2:1 event:command (`faq.md` is 51:6 — command pitfalls barely documented).
- Strategic ambition concentrates on the read side (metaengine); commands are treated as
  input surface, not as a domain to deepen.

## 4. Pareto order, if acting

1. ~~**`DecideFunc` variant receiving the command + auto-stamping `Causation`** — additive,
   non-breaking (`ExecuteCommandRef`-style method); fixes Gap 1 and Gap 3 together.~~
   done 2026-09-13 — shipped as `decider.ExecuteCommandRef` + `CausedCommand`/`CommandDecideFunc` with causation stamping (CHANGELOG `[Unreleased]`; plan W1).
2. ~~**Generalize upcasting to commandlifecycle replay** — small, non-breaking; fixes Gap 2
   where it hurts most (DLQ projection rebuilds).~~
   done 2026-09-13 — upcast composition confirmed and pinned by `commandlifecycle/upcast_composition_test.go` (plan W3).
3. **ADR-0112 command sourcing** — the big one; wait for real consumer demand. → routed: [BLOCKED] ADR-0138 row in TODO_LIST (consumer demand).

~~None of this is started; items 1–2 are planned in
[`docs/planning/2026-09-13_11-45_SUPERB-command-side-depth.md`](../planning/2026-09-13_11-45_SUPERB-command-side-depth.md)
(filed in TODO_LIST §Command-side domain depth).~~ Executed 2026-09-13 (items 1–2) and closed 2026-09-20 (W4 gates + release train) — see the banner above and the archived plan at [`docs/planning/archived/2026-09-13_11-45_SUPERB-command-side-depth.md`](../../planning/archived/2026-09-13_11-45_SUPERB-command-side-depth.md).
