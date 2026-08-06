package main

import (
	"context"
	"fmt"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func setupExplainCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"explain",
		cmdguard.NoFlags{},
		func(_ context.Context, _ *AppConfig, _ cmdguard.NoFlags) error {
			fmt.Print(renderExplain())
			return nil
		},
		cmdguard.WithShort(
			"Explain the .cqrs-lint.json config format, all keys, presets, and features",
		),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "explain", cmd, err)
}

// presetDescriptions maps preset names to human-readable descriptions.
// These explain WHY each preset exists and WHEN to use it.
//
//nolint:gochecknoglobals // read-only documentation table
var presetDescriptions = map[analyzer.ConfigPreset]string{
	analyzer.PresetLocalCLI:   "Single-user CLIs and local tools: no network server, no tracing.",
	analyzer.PresetProduction: "Deployed services: pins server=true and tracing=on for production safety.",
	analyzer.PresetLibrary:    "Library/SDK modules consumed by other Go programs: silences app-only rules.",
	analyzer.PresetReadOnly:   "Event/query consumers that never dispatch commands.",
}

func renderExplain() string {
	var b strings.Builder

	renderConfigFileSection(&b)
	renderTopLevelKeys(&b)
	renderPresets(&b)
	renderFeatures(&b)
	renderRulesConfig(&b)
	renderHealthConfig(&b)
	renderResolutionOrder(&b)
	renderSuppressionSyntax(&b)

	return b.String()
}

// writeSectionHeader writes a titled section header: the title, an
// underline of box-drawing dashes matching its length, and a blank line.
func writeSectionHeader(b *strings.Builder, title string) {
	b.WriteString(title + "\n")
	b.WriteString(strings.Repeat("─", len(title)) + "\n")
	b.WriteString("\n")
}

// renderKeyTable writes a 4-column aligned table: three padded columns plus
// an unpadded description column. Column widths are auto-computed from the
// header and row content. Terminated by a blank line. Shared by
// renderTopLevelKeys and renderFeatures.
func renderKeyTable(b *strings.Builder, headers [4]string, rows [][4]string) {
	w0, w1, w2 := len(headers[0]), len(headers[1]), len(headers[2])

	for _, r := range rows {
		if len(r[0]) > w0 {
			w0 = len(r[0])
		}
		if len(r[1]) > w1 {
			w1 = len(r[1])
		}
		if len(r[2]) > w2 {
			w2 = len(r[2])
		}
	}

	fmt.Fprintf(b, "  %-*s  %-*s  %-*s  %s\n",
		w0, headers[0], w1, headers[1], w2, headers[2], headers[3])
	fmt.Fprintf(b, "  %s  %s  %s  %s\n",
		strings.Repeat("─", w0), strings.Repeat("─", w1),
		strings.Repeat("─", w2), strings.Repeat("─", len(headers[3])))

	for _, r := range rows {
		fmt.Fprintf(b, "  %-*s  %-*s  %-*s  %s\n",
			w0, r[0], w1, r[1], w2, r[2], r[3])
	}

	b.WriteString("\n\n")
}

func renderConfigFileSection(b *strings.Builder) {
	writeSectionHeader(b, "CONFIG FILE")
	b.WriteString("  Location:  .cqrs-lint.json (in the directory where you run cqrs-lint)\n")
	b.WriteString("  Format:    JSON with Comments (JSONC)\n")
	b.WriteString("             // line comments and /* block comments */ are supported.\n")
	b.WriteString(
		"             Comments are stripped before parsing — document every setting inline.\n",
	)
	b.WriteString("\n")
	b.WriteString("  Example:\n")
	b.WriteString("    {\n")
	b.WriteString("      // Use the production preset for deployed services\n")
	b.WriteString("      \"preset\": \"production\",\n")
	b.WriteString("\n")
	b.WriteString("      // Only show warnings and errors (hide info-level noise)\n")
	b.WriteString("      \"min-severity\": \"warning\",\n")
	b.WriteString("\n")
	b.WriteString("      \"rules\": {\n")
	b.WriteString(
		"        // D002 is a false positive: our API structs mirror Discord's snake_case\n",
	)
	b.WriteString("        \"disable\": [\"D002\"]\n")
	b.WriteString("      }\n")
	b.WriteString("    }\n")
	b.WriteString("\n")
	b.WriteString("\n")
}

// topLevelKey describes one top-level config key.
type topLevelKey struct {
	key         string
	typ         string
	def         string
	description string
}

//
//nolint:gochecknoglobals // read-only documentation table
var topLevelKeys = []topLevelKey{
	{"preset", "string", `""`, "Named set of feature-flag and rule defaults (see PRESETS below)"},
	{
		"min-severity",
		"string",
		`"info"`,
		"Minimum severity to show: info, warning, error, critical",
	},
	{"min-confidence", "string", `"low"`, "Minimum confidence to show: low, medium, high"},
	{"format", "string", `"text"`, "Output format: text, json, sarif, markdown"},
	{"exclude", "string", `""`, "Paths to exclude (comma-separated glob patterns)"},
	{"exclude-rules", "string", `""`, "Rule IDs to exclude (comma-separated, e.g. \"C007,A001\")"},
	{"color", "string", `"auto"`, "Color output: auto, always, never"},
	{"group-by", "string", `""`, "Group findings by: none, module, aggregate"},
	{"features", "object", "{}", "Override auto-detected feature profile (see FEATURES below)"},
	{"rules", "object", "{}", "Rule-specific configuration (see RULES below)"},
	{"health", "object", "{}", "Health-score tuning (see HEALTH below)"},
}

func renderTopLevelKeys(b *strings.Builder) {
	writeSectionHeader(b, "TOP-LEVEL KEYS")

	rows := make([][4]string, len(topLevelKeys))
	for i, k := range topLevelKeys {
		rows[i] = [4]string{k.key, k.typ, k.def, k.description}
	}

	renderKeyTable(b, [4]string{"Key", "Type", "Default", "Description"}, rows)
}

func renderPresets(b *strings.Builder) {
	writeSectionHeader(b, "PRESETS")
	b.WriteString("  Presets are convenience bundles. They expand to a set of feature flags,\n")
	b.WriteString("  rule disables, and a severity floor. Explicit config always overrides\n")
	b.WriteString("  preset values. The severity floor is a LOWER BOUND — you can raise it\n")
	b.WriteString("  (e.g. to \"error\") but cannot lower it below the preset floor.\n")
	b.WriteString("\n")

	names := analyzer.ValidPresetNames()

	for _, name := range names {
		preset := analyzer.ConfigPreset(name)
		def := analyzer.ResolvePresetDefinition(preset)
		desc := presetDescriptions[preset]

		fmt.Fprintf(b, "  %s\n", name)
		fmt.Fprintf(b, "    %s\n", desc)

		if f := formatConfigFeatures(def.Features); f != "" {
			fmt.Fprintf(b, "    Features pinned:  %s\n", f)
		}

		if len(def.Rules.Disable) > 0 {
			fmt.Fprintf(b, "    Rules disabled:   %s\n", strings.Join(def.Rules.Disable, ", "))
		}

		if def.MinSeverity != "" {
			fmt.Fprintf(b, "    Severity floor:   %s\n", def.MinSeverity)
		}

		b.WriteString("\n")
		fmt.Fprintf(b, "    \"preset\": \"%s\"\n", name)
		b.WriteString("\n\n")
	}
}

// formatConfigFeatures renders a ConfigFeatures as a human-readable list of
// "key=value" pairs, omitting nil fields.
func formatConfigFeatures(cf analyzer.ConfigFeatures) string {
	var parts []string

	if cf.Store != nil {
		parts = append(parts, fmt.Sprintf("store=%s", *cf.Store))
	}
	if cf.CommandFlow != nil {
		parts = append(parts, fmt.Sprintf("command-flow=%s", *cf.CommandFlow))
	}
	if cf.Server != nil {
		parts = append(parts, fmt.Sprintf("server=%t", *cf.Server))
	}
	if cf.SoftDelete != nil {
		parts = append(parts, fmt.Sprintf("soft-delete=%t", *cf.SoftDelete))
	}
	if cf.Tracing != nil {
		parts = append(parts, fmt.Sprintf("tracing=%s", *cf.Tracing))
	}
	if cf.Snapshot != nil {
		parts = append(parts, fmt.Sprintf("snapshot=%s", *cf.Snapshot))
	}
	if cf.Domain != nil {
		parts = append(parts, fmt.Sprintf("domain=%s", *cf.Domain))
	}
	if cf.Transport != nil {
		parts = append(parts, fmt.Sprintf("transport=%t", *cf.Transport))
	}
	if cf.ServerLocal != nil {
		parts = append(parts, fmt.Sprintf("server-local=%t", *cf.ServerLocal))
	}
	if cf.AsyncBus != nil {
		parts = append(parts, fmt.Sprintf("async-bus=%t", *cf.AsyncBus))
	}

	return strings.Join(parts, ", ")
}

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
		typ:         "string",
		description: "Persistence backend the consumer wires up",
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

func init() {
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

	rows := make([][4]string, len(featureKeys))
	for i, f := range featureKeys {
		rows[i] = [4]string{f.key, f.typ, strings.Join(f.validValues, ", "), f.description}
	}

	renderKeyTable(b, [4]string{"Key", "Type", "Valid Values", "Description"}, rows)
}

// ruleConfigKey describes one rules.* config key.
type ruleConfigKey struct {
	key         string
	typ         string
	description string
	example     string
}

//
//nolint:gochecknoglobals // read-only documentation table
var ruleConfigKeys = []ruleConfigKey{
	{
		"disable", "[]string",
		"Rule IDs to suppress project-wide",
		`["P012", "C007"]`,
	},
	{
		"external-api-struct-prefixes",
		"[]string",
		"Struct-name prefixes whose JSON tags mirror an external API (Discord, Stripe). Excludes them from D002 mixed-casing check.",
		`["Discord", "Stripe"]`,
	},
	{
		"c008-ignore-fields", "[]string",
		"Field names to exclude from C008 (float64-for-money). Case-insensitive.",
		`["CostEstimate", "PriceIndex"]`,
	},
	{
		"c008-ignore-structs", "[]string",
		"Struct type names to exclude entirely from C008. Case-insensitive.",
		`["PricingMetrics"]`,
	},
}

func renderRulesConfig(b *strings.Builder) {
	writeSectionHeader(b, "RULES")

	keyWidth := len("Key")
	typeWidth := len("Type")

	for _, r := range ruleConfigKeys {
		if len(r.key) > keyWidth {
			keyWidth = len(r.key)
		}
		if len(r.typ) > typeWidth {
			typeWidth = len(r.typ)
		}
	}

	fmt.Fprintf(b, "  %-*s  %-*s  %s\n",
		keyWidth, "Key",
		typeWidth, "Type",
		"Description")
	fmt.Fprintf(b, "  %s  %s  %s\n",
		strings.Repeat("─", keyWidth),
		strings.Repeat("─", typeWidth),
		strings.Repeat("─", 60))

	for _, r := range ruleConfigKeys {
		fmt.Fprintf(b, "  %-*s  %-*s  %s\n",
			keyWidth, r.key,
			typeWidth, r.typ,
			r.description)
		fmt.Fprintf(b, "  %sExample: %s\n",
			strings.Repeat(" ", keyWidth+typeWidth+6),
			r.example)
	}

	b.WriteString("\n\n")
}

func renderHealthConfig(b *strings.Builder) {
	writeSectionHeader(b, "HEALTH")
	b.WriteString("  Key       Type    Default  Description\n")
	b.WriteString("  ───       ────    ───────  ───────────\n")
	b.WriteString("  info-cap  int     0 (→20)  Maximum health-score penalty from info findings.\n")
	b.WriteString("                             Set to 0 for the built-in default (20).\n")
	b.WriteString("                             Negative is treated as 0 (no cap).\n")
	b.WriteString("\n")
	b.WriteString("  Example:\n")
	b.WriteString("    {\"health\": {\"info-cap\": 15}}\n")
	b.WriteString("\n\n")
}

func renderResolutionOrder(b *strings.Builder) {
	writeSectionHeader(b, "CONFIG RESOLUTION ORDER")
	b.WriteString("  Settings are resolved in this order (later overrides earlier):\n")
	b.WriteString("\n")
	b.WriteString("    1. Built-in defaults (from struct tags)\n")
	b.WriteString("    2. Preset (if set: features, rule disables, severity floor)\n")
	b.WriteString("    3. Config file (.cqrs-lint.json: explicit overrides)\n")
	b.WriteString("    4. Auto-detection (fills in what's not pinned)\n")
	b.WriteString("    5. CLI flags (highest priority)\n")
	b.WriteString("\n")
	b.WriteString("  Severity floor: the preset's min-severity is a LOWER BOUND.\n")
	b.WriteString(
		"  You can raise it (e.g. to \"error\") but not lower it below the preset floor.\n",
	)
	b.WriteString("\n")
	b.WriteString("  Rule disables from preset and config are UNIONED (never subtracted).\n")
	b.WriteString("  Parent .cqrs-lint.json files (ancestor directories) are also merged\n")
	b.WriteString("  (monorepo config inheritance).\n")
	b.WriteString("\n\n")
}

func renderSuppressionSyntax(b *strings.Builder) {
	writeSectionHeader(b, "SUPPRESSION SYNTAX")
	b.WriteString("  Inline (single rule):\n")
	b.WriteString("    //cqrs-lint:ignore(C007) reason text\n")
	b.WriteString("\n")
	b.WriteString("  Inline (multiple rules):\n")
	b.WriteString("    //cqrs-lint:ignore(C007,A001) reason text\n")
	b.WriteString("\n")
	b.WriteString("  Block:\n")
	b.WriteString("    //cqrs-lint:ignore-start\n")
	b.WriteString("    ...code...\n")
	b.WriteString("    //cqrs-lint:ignore-end\n")
	b.WriteString("\n")
	b.WriteString("  Block (specific rules):\n")
	b.WriteString("    //cqrs-lint:ignore-start(C007,A001)\n")
	b.WriteString("    ...code...\n")
	b.WriteString("    //cqrs-lint:ignore-end\n")
	b.WriteString("\n")
	b.WriteString("  Disable project-wide via config:\n")
	b.WriteString("    {\"rules\": {\"disable\": [\"P012\"]}}\n")
	b.WriteString("\n")
	b.WriteString("  Both //cqrs-lint: and // cqrs-lint: (with space) are accepted.\n")
	b.WriteString("  Place inline suppressions on the line above the code or at end of line.\n")
}
