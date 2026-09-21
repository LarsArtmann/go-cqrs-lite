# One-pager: A first-class single-writer/lease story for engines

> **Status:** DESIGN MEMO — awaiting owner ratification before ADR.
> **Date:** 2026-09-21 · **Origin:** CV Phase-0 verdicts (reflection doc §4.2, F109/M20 of the
> [owner-unblock-trust plan](2026-09-20_17-40_SUPERB-owner-unblock-trust-pareto-plan.md))
> **Routing:** becomes ADR-0146 on ratification. Adjacent to the G-T01 direction ruling
> (operator-config vs developer-declaration axis — this is operator territory).

## Problem

A real consumer (CV) had to plan a driver-factory decorator — wrapping
`metaengine.RegisterDriver` (`metaengine/registry.go:61`) — solely to hold a
`<dsn>.lease` single-writer marker file, because the library has **no** engine- or
store-level single-writer mechanism. Two processes opening the same SQLite/libSQL DSN
concurrently must otherwise rely on the database's own locking, which varies per engine
and degrades silently (WAL allows multi-process writes; a second writer of a
file-backed KV engine may corrupt).

**Current reality (verified 2026-09-21):** lease semantics exist only for TASK claims,
not storage: `queue.ClaimDue` + heartbeat + `ErrLeaseNotHeld` (`queue/store.go:24`,
`:52`, `:99-101`; ADR-0134) and the `claiming/` module. Neither arbitrates engine
open. There is no
open-mode, advisory-lock, or marker-file option anywhere in engine construction;
`system.EngineConfig` carries only `Driver/DSN/Pragmas/Priority/MaterializedViews`
(`system/config_types.go:186`).

## Options

1. **Status quo + document the decorator pattern.** Zero API; every consumer
   re-implements CV's decorator. The pattern is already sanctioned
   (`RegisterDriver` is the database/sql-style extension point).
2. **`EngineConfig.SingleWriter` open-mode (recommended).** Operator declares intent
   at deployment: engine construction acquires an advisory lock on `<dsn>.cqrs-lease`
   (flock) and fails loudly with a typed error (Infrastructure family) when held.
   No-DSN engines (memory) are no-ops; engines without file semantics return
   `ErrExclusiveUnsupported` rather than pretending. Additive config field — can land
   in v4.x without breaking; the SEMANTICS (fail-at-open) need ratification only
   before v5 freezes engine construction surfaces.
3. **Storage-level enforcement matrix** (SQLite `locking_mode=EXCLUSIVE`,
   Postgres advisory locks, …). Strongest but per-engine work; not portable; the
   marker file in option 2 composes with it later.

## Recommendation

Option 2, implemented as one shared helper (flock acquire/renew/release + stale-lock
diagnosis — the `scripts/lib/verify-lock.sh` mechanics, productized) called from engine
construction when configured. Wire `system.EngineConfig.SingleWriter bool` +
`LeasePath string` (default `<dsn>.cqrs-lease`). Fail-loud, default-off.

## Non-goals / follow-ons

- Not a distributed-lock service: single-host advisory lock only (NATS/raft tenancy is
  watermill/metaengine routing territory).
- Replaces CV's decorator once shipped; the decorator class then dies.
- Doctor line when a lease is held (`materialized_view_doctor.go` pattern).
