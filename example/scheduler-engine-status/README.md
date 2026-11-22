# scheduler-engine-status

The v5-forward timer demo: durable deadlines on
`scheduling/engine.NewTimerStore` over a metaengine `DueClaimer` engine
(ADR-0142 — "all durable I/O rides metaengine"):

- OTel counter (`cqrs_scheduler_timers_fired`) on **`/metrics`** via the
  OTel→Prometheus bridge
- JSON snapshot on **`/status`**: fired count, `startedAt` anchor, and a live
  `firedPerMinute` rate
- staggered seed reminders (immediate / 5s / 30s) so both surfaces show live
  activity

The engine is the storage layer: the demo runs the in-memory engine for a
zero-dependency `go run .`; a deployed service swaps in sqlite/postgres/…
by registering the driver and changing the one `NewMemoryEngine` line.

```bash
go run .               # listens on :8080 (ADDR=:8181 to move it)
```

```bash
curl localhost:8080/status
# {"startedAt":"2026-11-22T18:40:02+01:00","timersFired":2,"firedPerMinute":4.1}

curl localhost:8080/metrics | grep cqrs_scheduler
```

The deprecated `scheduling/sqlstore` path (with its `ClaimMetrics` hooks) is
demonstrated by [`example/scheduler-otel-status`](../scheduler-otel-status/)
until v5 removes it.
