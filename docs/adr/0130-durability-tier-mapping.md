# ADR-0130: Durability-Tier Mapping (Engine × Tier → Mechanism)

- **Status:** Accepted
- **Date:** 2026-08-29
- **Context:** plan V3 T35.2; the mapping existed only in per-engine code and
  a one-line modules.md summary. Operators setting `DriverConfig.Durability`
  had no single document stating what each tier means per engine.

## Context

`metaengine.DriverConfig.Durability` accepts three tiers (`DurabilityStrict`,
`DurabilityNormal`, `DurabilityRelaxed`) plus the empty string (unspecified →
engine defaults). Each engine maps a tier onto its native durability knob.
Engines without a meaningful knob must reject an EXPLICIT tier request loudly
(`RejectDurabilityTier`) instead of silently ignoring it.

## Decision

### Mapping table (verified against engine code, 2026-08-29)

| Engine   | strict                                | normal                              | relaxed                                      |
| -------- | ------------------------------------- | ----------------------------------- | -------------------------------------------- |
| SQLite   | `PRAGMA synchronous=FULL`             | `PRAGMA synchronous=NORMAL`         | `PRAGMA synchronous=OFF`                     |
| Turso    | (inherits SQLite mapping over libSQL) | (inherits SQLite)                   | (inherits SQLite)                            |
| Postgres | `synchronous_commit=on` (via DSN)     | `synchronous_commit=off` (via DSN)  | `synchronous_commit=off` (via DSN)           |
| Pebble   | engine defaults (`pebble.Sync`)       | `WithAsyncWrites()` (WAL, no fsync) | `WithAsyncWrites()` + `WithDisableWAL()`     |
| bbolt    | engine defaults (`db.Update` fsync)   | engine defaults (same as strict)    | `WithNoSync()` (`NoSync` + `NoFreelistSync`) |
| Badger   | engine defaults (sync writes)         | `WithAsyncWrites()`                 | `WithAsyncWrites()` (same as normal)         |
| MySQL    | rejected (`RejectDurabilityTier`)     | rejected                            | rejected                                     |
| DuckDB   | rejected                              | rejected                            | rejected                                     |
| Dgraph   | rejected                              | rejected                            | rejected                                     |
| Memory   | rejected                              | rejected                            | rejected                                     |

Notes per engine:

- **Postgres**: the tier becomes a DSN startup parameter (`durabilityDSN`), so
  every pooled connection receives it — not a post-connect `SET` that reaches
  one connection. Postgres collapses normal and relaxed: `synchronous_commit`
  has no safe "skip the WAL" state.
- **Pebble**: relaxed disables the WAL entirely (process crash loses recent
  writes even from the page cache path).
- **bbolt**: normal ≡ strict (bbolt has no WAL to relax — every commit fsyncs);
  only relaxed diverges.
- **Badger**: normal ≡ relaxed (Badger's `SyncWrites=false` already keeps data
  safe against application crashes via its value-log replay; there is no
  intermediate knob).

### Two-sources-of-truth rule

An engine must FAIL construction when the operator names a durability tier AND
also sets the engine's native knob themselves (operator `PRAGMA synchronous`

- tier, or an existing `synchronous_commit` in the DSN + tier). Two sources of
  truth for one durability knob is a configuration error, not a
  last-writer-wins race.

### Defaults

The empty tier means "engine defaults" and is always accepted — this is what
every engine has shipped behavior-wise for years.

## Consequences

- `metaengine` Doctor/Introspection surfaces the EFFECTIVE tier per engine
  (see `GetEngineStats`/`Doctor` output) so operators can verify the tier
  actually took effect instead of trusting config.
- New engines implement either the three-way `tierToOptions`/pragma mapping
  or call `RejectDurabilityTier` first in their driver factory.
- The mapping is behavioral contract: the `adttest` capability conformance
  harness asserts the declared-vs-implemented durability surface per engine.
