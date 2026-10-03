package main

import (
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// featureKey describes one features.* config key.
type featureKey struct {
	key         string
	typ         string
	validValues []string
	description string
	// derive, when non-nil, supplies valid values from the corresponding
	// All*Kind() enumerator. Attaching the derivation here (rather than a
	// separate string-keyed map) makes the coupling structural: renaming
	// key in one place carries its derivation along.
	derive func() []string
}

//
//nolint:gochecknoglobals // read-only documentation table
var featureKeys = []featureKey{
	{
		key:         "store",
		typ:         "string | string[]",
		description: "Persistence backend(s) the consumer wires up; array form declares mixed pools (e.g. postgres journal + sqlite projections). Detection and precedence: see STORE DETECTION below",
		derive:      deriveStrings(analyzer.AllStoreKinds),
	},
	{
		key:         "command-flow",
		typ:         "string",
		description: "Command-dispatch pattern (read-only = no dispatcher)",
		derive:      deriveStrings(analyzer.AllCommandFlowKinds),
	},
	{
		key:         "server",
		typ:         "bool",
		validValues: []string{"true", "false"},
		description: "Network listener (HTTP or gRPC) is present",
	},
	{
		key:         "soft-delete",
		typ:         "bool",
		validValues: []string{"true", "false"},
		description: "Domain emits tombstone-like events",
	},
	{
		key:         "tracing",
		typ:         "string",
		description: "OpenTelemetry tracing middleware is wired",
		derive:      deriveStrings(analyzer.AllTracingKinds),
	},
	{
		key:         "snapshot",
		typ:         "string",
		description: "Snapshot store or strategy is configured",
		derive:      deriveStrings(analyzer.AllSnapshotKinds),
	},
	{
		key:         "domain",
		typ:         "string",
		description: "Business domain (escalates security/money rules for financial)",
		derive:      deriveStrings(analyzer.AllDomainKinds),
	},
	{
		key:         "monetary",
		typ:         "string",
		description: "Project handles monetary values (C008 auto-INFO when off)",
		derive:      deriveStrings(analyzer.AllMonetaryKinds),
	},
	{
		key:         "transport",
		typ:         "bool",
		validValues: []string{"true", "false"},
		description: "CQRS transport layer (http/grpc) is wired",
	},
	{
		key:         "server-local",
		typ:         "bool",
		validValues: []string{"true", "false"},
		description: "Server without production signals (CLI with embedded dashboard)",
	},
	{
		key:         "async-bus",
		typ:         "bool",
		validValues: []string{"true", "false"},
		description: "Distributed event bus (Watermill-backed) is wired",
	},
	{
		key:         "metaengine",
		typ:         "bool",
		validValues: []string{"true", "false"},
		description: "Metaengine cost-based planner is imported (auto-detected from imports)",
	},
}

// deriveStrings wraps a Kind enumerator (e.g. analyzer.AllStoreKinds) into a
// func() []string suitable for the featureKey.derive field. The generic
// constraint [T ~string] matches all named string types (StoreKind, TracingKind, etc.).
func deriveStrings[T ~string](fn func() []T) func() []string {
	return func() []string {
		kinds := fn()
		values := make([]string, len(kinds))
		for i, k := range kinds {
			values[i] = string(k)
		}
		return values
	}
}

func init() { //nolint:gochecknoinits // derive valid values from featureKey.derive enumerators; runs once at package load
	// Derive feature valid values from the derive field (which calls the
	// All*Kind() enumerators) to eliminate the split-brain risk of maintaining
	// hand-written copies alongside the Kind const blocks in the analyzer package.
	for i := range featureKeys {
		if featureKeys[i].derive != nil {
			featureKeys[i].validValues = featureKeys[i].derive()
		}
	}
}

func renderFeatures(b *strings.Builder) {
	writeSectionHeader(b, "FEATURES")
	b.WriteString("  Each feature flag overrides the auto-detected value. Set only the\n")
	b.WriteString("  ones you want to pin; unset flags use auto-detection.\n")
	b.WriteString("\n")
	b.WriteString("  Why some flags are strings and others bools: string-typed (Kind)\n")
	b.WriteString("  flags are tri-state — the values below pin the fact, while leaving\n")
	b.WriteString("  the key unset (or setting the \"unknown\" sentinel) defers to the\n")
	b.WriteString("  detection heuristics. Bool-typed flags are binary facts: unset still\n")
	b.WriteString("  defers to detection, but the value space has no third state. That is\n")
	b.WriteString("  why \"tracing\" takes on/off yet \"server\" takes true/false.\n")
	b.WriteString("\n")

	rows := make([][]string, len(featureKeys))
	for i, f := range featureKeys {
		rows[i] = []string{f.key, f.typ, strings.Join(f.validValues, ", "), f.description}
	}

	renderTable(b, []string{"Key", "Type", "Valid Values", "Description"}, rows)

	renderStoreDetection(b)
}

// renderStoreDetection documents how the store feature is detected. The
// precedence chain and the memory-inference caveat are load-bearing contract:
// users reading "store: memory" for an engine-less metaengine import need to
// know why, and mixed-pool users need to know every backend is recorded.
func renderStoreDetection(b *strings.Builder) {
	b.WriteString("  STORE DETECTION\n")
	b.WriteString("  ================\n")
	b.WriteString("\n")
	b.WriteString("  Detected per nearest go.mod module, from non-test imports only.\n")
	b.WriteString(
		"  The primary store is the FIRST import signal seen, in visit order\n",
	)
	b.WriteString("  (alphabetical for compiled packages, declaration order otherwise),\n")
	b.WriteString("  among these signal kinds:\n")
	b.WriteString("\n")
	b.WriteString("    1. stack/<backend> preset import (stack/sqlite, stack/postgres, …)\n")
	b.WriteString("    2. metaengine/<engine>engine import (sqliteengine → sqlite, …)\n")
	b.WriteString("    3. storage/ import → custom (refined by constructor scan when possible)\n")
	b.WriteString("\n")
	b.WriteString("  Fallbacks apply only when no signal above was seen:\n")
	b.WriteString("\n")
	b.WriteString(
		"    4. bare SQLite driver import (modernc.org/sqlite, mattn/go-sqlite3)\n",
	)
	b.WriteString("       → sqlite\n")
	b.WriteString("    5. engine-less metaengine → memory (the built-in driver the core\n")
	b.WriteString("       package init-registers — the only import-invisible backend)\n")
	b.WriteString("    6. otherwise → none\n")
	b.WriteString("\n")
	b.WriteString("  Every backend signal is ALSO recorded in the multi-store list, so mixed\n")
	b.WriteString("  pools (journal + projection engines) stay visible: declare them in config\n")
	b.WriteString(
		"  as an array — \"store\": [\"postgres\", \"sqlite\"] — or let detection union them.\n",
	)
	b.WriteString("  Rules quantify over the union (any SQL-backed store enables pushdown\n")
	b.WriteString("  suggestions); the first array entry is the primary.\n")
	b.WriteString("\n")
	b.WriteString("  Caveat: an in-app custom Engine implementation also persists without any\n")
	b.WriteString("  engine import; import analysis cannot distinguish it from the memory case.\n")
	b.WriteString("\n")
}
