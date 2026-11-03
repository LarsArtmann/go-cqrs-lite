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
	{
		fragment:    "scheduling/sqlstore",
		replacement: "scheduling/engine.NewTimerStore over a metaengine.DueClaimer engine",
	},
}

// exemptedExampleModules lists example apps whose WHOLE PURPOSE is demoing a
// deprecated surface through its deprecation window. V007 fires on examples
// by design (they are the consumer copy-paste surface), so a deprecation-demo
// example needs an explicit, reasoned exemption instead of a silent gate
// skip in CI. Each entry dies with its example at the v5 cut — the liveness
// test in v007_drift_test.go fails if the module directory disappears while
// the exemption stays.
var exemptedExampleModules = map[string]string{ //nolint:gochecknoglobals // static table
	"github.com/larsartmann/go-cqrs-lite/example/scheduler-otel-status":
		"deprecation-window demo of scheduling/sqlstore ClaimMetrics (ADR-0142; deleted with sqlstore at v5)",
}
