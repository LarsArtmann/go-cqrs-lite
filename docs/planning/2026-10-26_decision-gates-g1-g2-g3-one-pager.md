# Owner Decision Gates: catalog × auditlog bridge (G1/G2/G3)

**Date:** 2026-10-26 · **From:** [Pareto plan](2026-10-26_11-21_SUPERB-catalog-truth-derivation-pareto-plan.md) T23/F64 · **Takes:** ~10 min
**Context:** defaults are already implemented; a veto reworks only the named tasks.

## G1 — Catalog Service granularity

**Question:** one catalog Service per deployable, or per bounded context?

**Default (implemented):** 1 per deployable — fleet convention (invoices, InboxClean,
PapDashboard all model single services). Derivation leaves room for a context-split
option later; nothing in the shipped API blocks it.

**Impact if vetoed:** T07's auditlog `ExportCatalog` rework only. Costliest gate to
get wrong — please answer this one first.

## G2 — Bridge ownership (catalog ↔ auditlog)

**Question:** which side owns the bridge between static catalogs and runtime DI truth?

**Default (implemented):** auditlog-side exporter — `samber-do-auditlog` gains
`ExportCatalog()` (registrations → catalog resources); `catalog/` stays dep-free pure
data; `systemcatalog` (new) owns the system-side derivation. No reverse edges.

**Impact if vetoed:** T07 implementation moves; catalog purity guarantee drops.

## G3 — `catalog.home.lan` federation hub

**Question:** is the hub still the federation target?

**Default:** yes, keep — standardize toward it (T12: one export entrypoint convention +
ingestion contract doc). The architecture HTML reference is a PARALLEL surface, not a
replacement.

**Impact if vetoed:** T12 becomes "pick a new target or drop federation"; exports stay
per-repo.

## What already shipped under the defaults (2026-10-26 session)

- `systemcatalog/v4` module: `FromDomainConfig` / `FromSystem` / `DeriveInto`
  (derived catalogs from DomainConfig + runtime registrations; design note:
  [2026-10-26_systemcatalog-adapter-design.md](2026-10-26_systemcatalog-adapter-design.md))
- `catalog.AssertInventory` + per-kind variants (drift becomes a build failure)
- `system.CommandRegistrations`/`QueryRegistrations` (runtime-inventory surface)
- `catalog.CommandFromSchema`/`EventFromSchema`/`QueryFromSchema`
  (schema-parameterized message constructors)
