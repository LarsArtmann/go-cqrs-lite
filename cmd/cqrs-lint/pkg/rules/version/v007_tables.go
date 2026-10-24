package version

// Data tables for V007 (v5-removed-api-usage). Kept separate from the
// detector so rule logic and the curated removal surface evolve independently.

// cqrsModulePrefix is the go-cqrs-lite module path prefix every consumer import shares.
const cqrsModulePrefix = "github.com/larsartmann/go-cqrs-lite/"

// deprecatedV5Module describes a module removed entirely at v5.
type deprecatedV5Module struct {
	fragment    string // module path relative to go-cqrs-lite/ (no /vN suffix)
	replacement string
}

var deprecatedV5Modules = []deprecatedV5Module{ //nolint:gochecknoglobals // static table
	// ADR-0123: stack presets replaced by system.New.
	{fragment: "stack/memory", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/sqlite", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/pebble", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/bbolt", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/duckdb", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/postgres", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/mysql", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/turso", replacement: "system.New with DomainConfig + DeploymentConfig"},
	// ADR-0123: relational + view tiers absorbed into metaengine engines.
	{fragment: "storage/relational", replacement: "metaengine engines with layout planning"},
	{fragment: "storage/view", replacement: "metaengine engines with layout planning"},
	// ADR-0142 I/O rule (2026-10-24): satellite SQL timers retire for the engine path.
	{fragment: "scheduling/sqlstore", replacement: "scheduling/engine.NewTimerStore over a metaengine.DueClaimer engine"},
}
