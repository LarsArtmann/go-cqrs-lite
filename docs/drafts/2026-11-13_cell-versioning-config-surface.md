# system: ADR-0141 temporal cells are unreachable through system.New — no EngineConfig/DriverConfig surface for cell versioning

# Draft: temporal cell versioning unreachable through system.New (issue #131)

> **Status 2026-11-13: SHIPPED.** `EngineConfig.CellVersioning` (koanf `cell_versioning: {max_versions, max_age}`) now plumbs through `DriverConfig.CellVersioning` into every registered driver factory — sqlite/turso/memory/bigtable build versioned engines, all other engines fail construction loudly via `metaengine.RejectCellVersioning`. This draft is the historical issue text.

## TL;DR

`sqliteengine.WithCellVersioning` / `WithRetention` are engine-construction options, but `system.New` builds engines from `DriverConfig` through registered driver factories (database/sql model), and neither `DeploymentConfig.Engines` (`EngineConfig`) nor `metaengine.DriverConfig` carries any versioning or retention field — so a composition-root deployment **cannot enable temporal cells at all**. Ask: **a config surface (e.g. `cell_versioning: {max_versions, max_age}`) plumbed from `EngineConfig` through `DriverConfig` into engine construction**.

## Problem

At `5639c0d6` (2026-11-13), `system/v4/engine_config.go`:

```go
// skip-validate
type EngineConfig struct {
	Driver  string   `koanf:"driver"`  // "sqlite", "memory", "pebble", "duckdb", "postgres"
	DSN     string   `koanf:"dsn"`     // connection string (empty for memory)
	Pragmas []string `koanf:"pragmas"` // SQLite pragmas (e.g., "wal", "foreign_keys")
```

…followed by `Priority`, `Durability`, `MaterializedViews`, `MaxOpenConns` — no versioning, no retention. The metaengine side mirrors it (`metaengine/v4/registry.go`):

```go
// skip-validate
type DriverConfig struct {
	DSN     string
	Pragmas []string
```

…plus `Priority`, `Durability`, `MaterializedViews`, `MaxOpenConns`. `DriverFactory` is `func(ctx, DriverConfig) (Engine, error)` and drivers register at init — there is deliberately no caller-supplied-engine path. Meanwhile the temporal options that ADR-0141 shipped live only as direct engine options (`metaengine/sqliteengine/v4/temporal.go`):

```go
// skip-validate
func WithCellVersioning(opts ...CellVersioningOption) EngineOption {
	return func(e *sqliteEngine) {
		e.versioning = true
```

Result: engine-level temporal is green and conformance-pinned (ADR-0141), but any deployment going through the mandated composition root (ADR-0123) — which is every new app, per the skill and README — has no way to turn it on. Registering a second `sqlite` driver factory that closes over the options technically works but duplicates the driver and fights the registry model; that is a workaround, not a surface.

The planner already handles the honest-degradation half (an `AsOf`-declaring query on a non-versioned engine fails loud rather than lying), so the config plumbing is the only thing between ADR-0141 and temporal reads in system deployments.

## Why

- ADR-0123 makes `system.New` the single composition root; an engine capability the composition root cannot reach is a capability most of the fleet cannot use.
- Retention needs to be reachable too: without a `RetentionPolicy`, `meta_cell_versions` grows on every versioned write, and the operator knob (`MaxVersions` / `MaxAge`) is the documented control (ADR-0141 §1).
- The TODO_LIST "Temporal versioned cells — ADR-0141 follow-ups" section tracks BigTable validation and soak only — this gap is untracked (checked 2026-11-13).

## Ask

**One operator-facing surface, plumbed end to end**: an `EngineConfig`/`DriverConfig` field (koanf-tagged, e.g. `cell_versioning: {max_versions, max_age}`) that drivers translate into their construction options — sqliteengine into `WithCellVersioning(WithRetention(...))`, bigtable into its GCPolicy analog, engines without versioning fail construction loudly on any non-empty value (same pattern `Durability` already uses).

💘 Generated with Crush
