# ADR-0027: Defer Postgres LISTEN/NOTIFY Event Bus to v2.8.0

| Field   | Value          |
| ------- | -------------- |
| Date    | 2026-06-19     |
| Status  | Partially Implemented — PostgresBus built but not wired into presets; needs real-PG testing |
| Decider | Lars Artmann   |

## Context

Every v2.7.0 Bundle preset (`stack/memory`, `stack/sqlite`, `stack/pebble`,
`stack/postgres`) wires an **in-process** `memory.NewMemoryBus()` for event
publish/subscribe. That is correct and sufficient for single-process
deployments — the common case for a library consumer.

The natural next step for the Postgres preset is a real distributed bus backed
by Postgres `LISTEN`/`NOTIFY`, so multiple processes sharing one database can
propagate events to each other. This was listed as a candidate v2.7.0 item.

## Decision

**Defer the LISTEN/NOTIFY bus to v2.8.0. Ship v2.7.0 with the in-memory bus
and document the single-process scope clearly.**

A `LISTEN`/`NOTIFY` bus is not a small feature, and doing it carelessly would
harm consumer trust more than help it. The design must solve:

1. **8 KB NOTIFY payload limit.** Event payloads routinely exceed this, so
   `NOTIFY` must carry only a lightweight reference (aggregate ID + event ID +
   type) and the listener must **re-fetch** the full event from the event store.
   That couples the bus to the store and introduces an ordering/visibility
   question (the `NOTIFY` can arrive before the producing transaction is
   visible to the listener's connection).
2. **Listener lifecycle.** A long-lived goroutine per connection running
   `LISTEN`, with context cancellation, reconnect-on-error, and clean
   shutdown — non-trivial concurrency to get right.
3. **Testing.** Meaningful tests require a running Postgres; the
   `POSTGRES_TEST_DSN`-gated path now exists in CI, but a flaky distributed
   test is worse than none.
4. **Scope discipline.** v2.7.0's flagship is the Bundle composition layer +
   persistent read models, both now complete and verified. Adding a
   distributed pub/sub expands the release surface into territory that
   deserves its own design pass.

## Alternatives Considered

- **Ship a minimal version now.** Rejected: the re-fetch + lifecycle concerns
  make "minimal" fragile. A half-correct distributed bus is a reliability
  footgun.
- **Use an outbox pattern instead.** Outbox + polling is more robust than
  `LISTEN`/`NOTIFY` but is a larger design still. Better suited to a dedicated
  v2.8.0 effort, possibly as `transport/`-style modules (ADR-0025).

## Consequences

- The `storage.PostgresBus` exists and implements `event.Bus` via LISTEN/NOTIFY
  with re-fetch. However, it is **not yet wired into `stack/postgres`** — the
  preset still uses `memory.NewMemoryBus()`. Consumers must wire it manually
  via `stack.WithBus(postgresBus)`.
- Real-Postgres integration tests are **not yet written**. The existing tests
  use a mock listener + SQLite, which verifies the bus logic but not the
  actual `pg_notify()` / LISTEN path.
- The `refetchEvent` path uses `LoadByEventID` (indexed lookup) when the store
  supports `EventByIDLoader`, falling back to `LoadFromVersion` scan otherwise.
- Remaining work: wire into preset, provide pgx-based `NotificationListener`,
  add real-PG integration test.

## Forward references

- See `docs/planning/2026-06-19_15-53_V2.7.0_RELEASE_HARDENING_AND_PERSISTENT_READMODELS.md`
  §4 (M20) for the original kill-switch rationale.
- `ROADMAP.md` "Post-Bundle direction" lists this as the top multi-process theme.
