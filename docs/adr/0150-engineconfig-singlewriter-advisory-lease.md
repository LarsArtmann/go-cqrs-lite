# ADR-0150: EngineConfig.SingleWriter Advisory Lease

- Status: Accepted
- Date: 2026-10-08
- Deciders: owner blanket execution authorization 2026-10-08 ("GET SHIT DONE") adopting the one-pager recommendation; originally proposed as M20 (a) 2026-09-21
- Verification: docs/planning/2026-09-21_engine-single-writer-lease-one-pager.md (option 2, recommended)
- Note: the TODO rows called this "ADR-0146 candidate"; slot 0146 was claimed by no-federated-query-engine, so the ruling lands here

## Context

CV's Phase-0 ADR conditions every library-store cutover on a CV-owned
`metaengine.RegisterDriver` decorator wrapping their `<dsn>.lease` single-writer
marker, because the library has no engine/store-level lock. Lease semantics
today exist only in `queue/` (ClaimDue + heartbeat + `ErrLeaseNotHeld`) and
`claiming/` — neither arbitrates engine open. Multiple writers opening the same
SQLite/DuckDB file is silent corruption territory.

## Decision

Adopt one-pager option 2: **`EngineConfig.SingleWriter` open-mode advisory lease.**

- Operator declares intent at deployment: engine construction acquires an
  advisory flock on `<dsn>.cqrs-lease` (override: `LeasePath`) and fails loudly
  with a typed Infrastructure-family error when the lease is held.
- One shared helper (flock acquire/renew/release + stale-lock diagnosis) called
  from engine construction when configured — the `scripts/lib/verify-lock.sh`
  mechanics, productized.
- No-DSN engines (memory) are no-ops; engines without file semantics return
  `ErrExclusiveUnsupported` rather than pretending.
- Fail-loud, default-off. Additive config field, so the surface can land in
  v4.x; the SEMANTICS freeze at v5 (engine construction surfaces must be final
  before the cut — tracked as plan task T11).

## Consequences

- CV's decorator class dies once shipped; the library owns the guarantee.
- Not a distributed-lock service: single-host advisory lock only (NATS/raft
  tenancy is watermill/metaengine routing territory).
- Doctor gains a line when a lease is held (materialized_view_doctor pattern).
- Composes with a future storage-level enforcement matrix (SQLite
  `locking_mode=EXCLUSIVE`, Postgres advisory locks) — option 3 stays open.

## Implementation slot

v5-GOAL plan T11/f077 (W2, BEFORE the v5 branch freezes construction surfaces).
