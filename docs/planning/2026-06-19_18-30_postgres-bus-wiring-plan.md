# Plan: Wire PostgresBus + Fix Findings

**Date:** 2026-06-19
**Trigger:** Brutal self-review round 2 of streaming/DistributedRunner/PostgresBus session.

## Problem Summary

The previous session shipped `storage.PostgresBus` (LISTEN/NOTIFY event bus) but never wired it into `stack/postgres` — the only path consumers can reach. The bus is a ghost system. Combined with 2 open pgx CVEs and missing real-Postgres tests, the "completed" claim is dishonest.

## Brutal Self-Review Findings

| # | Finding | Severity |
|---|---------|----------|
| 1 | PostgresBus UNWIRED into stack/postgres | CRITICAL |
| 2 | pgx v5.7.1 has 2 CVEs (memory-safety + SQL-injection) | CRITICAL |
| 3 | No real-Postgres integration test for the bus | HIGH |
| 4 | Split-brain error sentinel (errors.New vs event.NewInfrastructure) | MED |
| 5 | notifyPayload stringly-typed (should use branded types) | MED |
| 6 | NotificationListener missing Listen(channel) method | MED |
| 7 | No otel spans on PostgresBus | MED |
| 8 | Pebble lacks LoadByEventID | LOW |
| 9 | ADR-0027 status stale | LOW |

## Pareto Breakdown

### Tier 1: 1% effort → 51% impact (Security + Ghost Fix)
- T1.1: Upgrade pgx v5.7.1 → v5.10.0 (5m)
- T1.2: Implement pgxListener using pgxpool (12m)
- T1.3: Add Listen(channel) error to NotificationListener (8m)
- T1.4: Wire PostgresBus into stack/postgres via WithDistributedBus (10m)
- T1.5: Real-Postgres integration test (12m)
- T1.6: Update CI (5m)

### Tier 2: 4% effort → 64% impact (Type Model)
- T2.1: notifyPayload uses branded EventID + AggregateRef (12m)
- T2.2: Fix split-brain error sentinel (5m)
- T2.3: Add otel spans to PostgresBus (12m)
- T2.4: Test refetchByVersion fallback path (10m)
- T2.5: Pebble LoadByEventID (12m)

### Tier 3: 20% effort → 80% impact (Polish)
- T3.1: Document PostgresBus usage (10m)
- T3.2: Sync CHANGELOG, TODO_LIST, ROADMAP, ADR-0027 (8m)
- T3.3: nix run .#lint + format + verification (10m)
- T3.4: Regenerate api_surface.txt golden (5m)

### Tier 4: Blocked (deferred)
- gRPC/NATS/Redis transports (ADR-0025, large scope)
- jsonv2/arena experiments (Go stdlib blocked)

## Library Reuse

- `pgxpool` (already transitive via pgx/v5) — native LISTEN/NOTIFY support
- `cqrsotel.StartSpan` / `RecordError` — consistent with SQL stores
- `event.NewAggregateRef` / `id.EventID` — replace stringly-typed payload
- `event.NewInfrastructure` — error family consistency
- `pgDB(t)` test helper — existing postgres integration test pattern

## Verification

After each step: `cd <module> && GOWORK=off go test ./... -count=1`.
Final gate: `nix run .#lint`, full test suite, api-stability golden.
