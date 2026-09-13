# scheduler-otel-status

Runnable wiring for `scheduling/sqlstore` claim observability (the module
deliberately carries no OTel dependency — this example is the recipe):

- `ClaimMetrics` hooks → OTel counters (`cqrs.scheduler.claim.*`), exposed
  on **`/metrics`** via the OTel→Prometheus bridge
- the built-in `Metrics()` snapshot → JSON on **`/status`**, including the
  `startedAt` anchor and a live `claimedPerMinute` rate

```bash
go run .               # listens on :8080 (ADDR=:8181 to move it)
```

```bash
curl localhost:8080/status
# {"claimedBatches":4,"claimedTimers":1,"renewed":0,"renewRejected":0,
#  "startedAt":"2026-09-13T17:57:32+02:00","claimedPerMinute":26.77}

curl localhost:8080/metrics | grep cqrs_scheduler
# cqrs_scheduler_claim_batches_total{...} 4
# cqrs_scheduler_claim_timers_total{...} 1
```

`recorder.go` is the copy-paste wiring (`ClaimMetrics` → counters);
`main.go` adds the poll loop, `/status` rate math, and the HTTP server.
