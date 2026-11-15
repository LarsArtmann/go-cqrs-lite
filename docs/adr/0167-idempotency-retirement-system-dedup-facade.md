# ADR-0167: Retire idempotency/{kvstore,sqlstore} — dedup moves behind system over metaengine.DedupStore

- Status: Proposed (2026-11-15, W3/M51 — captures the 2026-10-11 placement-analysis verdict; the
  facade design questions route to the owner via the decision sheet)
- Date: 2026-11-15

## Context

`idempotency/{kvstore,sqlstore}` are pre-ADR-0142 fossils: satellite-storage backends for the
idempotent-receiver contract (`Record` no-op-on-existing, `CheckAndRecord` atomic claim, `Seen`
lazy-delete), built when the parent `idempotency/v4` module was the only dedup home. The parent
survives (its `MemoryStore` is the sanctioned single-process implementation), but the two
satellite backends predate the universal-storage-substrate rule.

- **ADR-0158** (2026-10-11) deprecated both modules: every exported symbol carries a
  `// Deprecated:` notice pointing at `metaengine.DedupStore`; nothing is removed in v4,
  deletion at v5.0.0.
- **ADR-0142** made dedup an engine capability: `DedupStore` is the one dedup runtime, ridden by
  every durable-I/O consumer (contract 30).
- The **2026-10-11 placement analysis** (docs/status/2026-10-11_07-28 report) ended with a
  verdict but no artifact: pointing consumers directly at `metaengine.DedupStore` repeats the
  original mistake — which engine, which collection, what fallback, and what clock are
  DEPLOYMENT decisions, not domain decisions. The facade belongs at `system`.
- **Fleet truth (census 2026-11-15, `docs/evidence/fleet-census-2026-11-15_12-34.json`):**
  `idempotency/sqlstore/v4` has 5 direct consumers (DiscordSync, browser-history ×2,
  KeyCountdown, discordsync) — deletion waits for v5.0.0 per the no-delete-with-consumers
  rule; `idempotency/kvstore/v4` was ALREADY deleted 2026-10-17 at zero consumers
  (ADR-0152 addendum); `idempotency/v4` itself has 8 direct consumers and is NOT in scope
  (MemoryStore is cache-class, not satellite SQL).

## Decision (proposed)

1. **Retire the surviving satellite module (`idempotency/sqlstore`) at v5.0.0** (the deletion
   itself is already ruled by ADR-0158; `idempotency/kvstore` already died 2026-10-17 at zero
   consumers). The SUCCESSOR surface is a `system`-exposed dedup facade over
   `metaengine.DedupStore` — consumers must not wire engines or satellite SQL for dedup
   directly.
2. **The facade owns the deployment decisions:** engine selection from `DeploymentConfig`,
   collection naming, the `now` clock default, and the non-engine in-memory fallback
   (`MapDedupStore` semantics) for deployments without a dedup-capable engine.
3. **The design questions route to the owner** (decision sheet D27, 2026-11-15): facade shape
   (constructor arg vs `DeploymentConfig` field vs accessor), external `idempotency.Store`
   (`Seen/Record/CheckAndRecord`) vs `DedupStore` alignment, `DedupCheckAndRecord` bool vs
   sentinel return for middleware symmetry, adapter placement (system / metaengine / external
   go-idempotency), health/quarantine interaction (ADR-0137) when the dedup engine
   quarantines, and reset/catch-up semantics (ADR-0136/0143) — claim rows are data, not
   journal facts, so a dedup window does NOT survive an engine reset by default.
4. **Migration is codemod-shaped:** `cmd/cqrs-upgrade` grows a v5 rule replacing
   `sqlstore.NewSQLiteStore(db)`-style constructors with the system path once the facade
   lands; the 5 sqlstore consumers carry the row. `versions.json` tag truth stays intact
   through the deprecation ladder (no edits).

## Consequences

- One dedup stack end-to-end: `system` facade → `metaengine.DedupStore` → engine picked at
  deployment (contract 30: all durable I/O rides metaengine; no satellite SQL anywhere).
- `system` consumers get dedup as configuration; the planner/health/reset machinery
  (ADRs 0136/0137/0143) applies to dedup for free.
- The 5 sqlstore consumers owe a migration; blocked on the facade design (D27), not on this
  ADR.
- `idempotency/v4` (MemoryStore) and the external `go-idempotency` repo are unaffected.

## Addendum routing

- **ADR-0152 ghost-verdict for `idempotency/kvstore` (KEEP-DRIVER, pending owner sign-off):**
  resolved-by-deletion — the module died 2026-10-17 under the delete-at-zero addendum before
  the verdict was ever signed off; no owner action remains. Recorded in ADR-0152's addendum
  block.
- The facade design questions (source report §f5–f12) live on decision-sheet row D27; the
  migration/codemod rows (§f13–f20) stay parked in TODO_LIST until the owner rules D27.
