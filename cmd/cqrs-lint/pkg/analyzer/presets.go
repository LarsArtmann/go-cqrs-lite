package analyzer

import "sort"

// ConfigFeatures mirrors FeatureProfile but uses pointer fields so the config
// loader can distinguish "user did not set this" (nil) from "user set the zero
// value". Each non-nil field overrides the auto-detected value.
type ConfigFeatures struct {
	Store       *StoreKind       `json:"store,omitempty"`
	CommandFlow *CommandFlowKind `json:"command-flow,omitempty"` //nolint:tagliatelle // CLI config key
	Server      *bool            `json:"server,omitempty"`
	SoftDelete  *bool            `json:"soft-delete,omitempty"` //nolint:tagliatelle // CLI config key
	Tracing     *TracingKind     `json:"tracing,omitempty"`
	Snapshot    *SnapshotKind    `json:"snapshot,omitempty"`
	Domain      *DomainKind      `json:"domain,omitempty"`
	Monetary    *MonetaryKind    `json:"monetary,omitempty"`
	Transport   *bool            `json:"transport,omitempty"`
	ServerLocal *bool            `json:"server-local,omitempty"` //nolint:tagliatelle // CLI config key
	AsyncBus    *bool            `json:"async-bus,omitempty"`    //nolint:tagliatelle // CLI config key
}

// ConfigPreset is a named set of feature-flag defaults. Presets are sugar:
// they expand to ConfigFeatures values, and explicit flags always override.
// The flags are the source of truth; presets are convenience.
type ConfigPreset string

const (
	PresetNone ConfigPreset = ""
	// PresetLocalCLI is for single-user CLIs and local tools: no network
	// server, so encryption/signing/racing/tracing rules are downgraded or
	// suppressed. Pins Server=false and Tracing=off; leaves Store,
	// CommandFlow, SoftDelete, and Snapshot to auto-detection since local
	// tools vary widely.
	PresetLocalCLI ConfigPreset = "local-cli"
	// PresetProduction is for deployed services: pins Server=true (concurrency
	// and network exposure are real) and Tracing=on (observability matters).
	// Leaves persistence/command/snapshot choices to auto-detection so it fits
	// any backend.
	PresetProduction ConfigPreset = "production"
	// PresetLibrary is for modules consumed by other Go programs: no server of
	// their own, no command dispatch (they expose types, not handlers), no
	// tracing setup, no snapshots. This silences rules that only make sense
	// for runnable applications.
	PresetLibrary ConfigPreset = "library"
	// PresetReadOnly is the narrowest preset: it only pins CommandFlow to
	// read-only, suppressing idempotency and command-flow rules. Use it when a
	// service consumes events/queries but never dispatches commands.
	PresetReadOnly ConfigPreset = "read-only"
	// PresetLibraryFramework extends PresetLibrary by disabling ALL adoption-
	// coaching (F-series) rules. Use this for framework/SDK modules where
	// every F-rule is a false positive — the framework provides building blocks
	// but cannot dictate how consumers compose them. Consumers of the framework
	// should use "production" or "local-cli" instead.
	PresetLibraryFramework ConfigPreset = "library-framework"
)

// PresetDefinition is the single source of truth for a named preset. It bundles
// the feature flags, rule disables, and severity floor that together express
// "what kind of project is this?" Both the init command (which generates
// .cqrs-lint.json) and the runtime (which resolves the config) read from
// PresetDefinitions, eliminating the split-brain drift that occurred when init
// used hardcoded JSON strings and the runtime used a separate Go map.
type PresetDefinition struct {
	Features ConfigFeatures `json:"features,omitzero"`
	Rules    RulesConfig    `json:"rules,omitzero"`
	// MinSeverity sets the lowest severity shown (e.g. "warning" hides info).
	// Empty means use the default ("info").
	MinSeverity string `json:"min-severity,omitempty"` //nolint:tagliatelle // CLI config key
}

// PresetDefinitions is the canonical map of preset names to their full
// definitions. This is the ONLY place presets are defined — init generates
// config files from it, and the runtime resolves features + rules from it.
//
// Rule-disable lists encode rules that are known false-positives for the
// preset's project type (e.g. server-infrastructure adoption rules for a
// local CLI). They are applied as defaults; explicit config disables are
// added on top (union), never subtracted.
//
//nolint:gochecknoglobals // read-only lookup table
var PresetDefinitions = map[ConfigPreset]PresetDefinition{
	PresetLocalCLI: {
		Features: ConfigFeatures{
			Server:  new(false),
			Tracing: new(TracingOff),
		},
		Rules: RulesConfig{
			Disable: []string{"F004", "F009", "F013", "F017"},
		},
		MinSeverity: "warning",
	},
	PresetProduction: {
		Features: ConfigFeatures{
			Server:  new(true),
			Tracing: new(TracingOn),
		},
	},
	PresetLibrary: {
		Features: ConfigFeatures{
			Server:      new(false),
			CommandFlow: new(CommandFlowReadOnly),
			Tracing:     new(TracingOff),
			Snapshot:    new(SnapshotOff),
		},
		Rules: RulesConfig{
			// Rules that are inherent false-positives for library/SDK modules
			// consumed by other Go programs. A library defines types and
			// infrastructure but cannot force its consumers to adopt catalog
			// docs, encryption, signing, or relational projections — those are
			// the DEPLOYING APPLICATION's responsibility (use the "production"
			// preset there).
			//
			// E003/E016: a library's domain package legitimately mixes
			//   command/event/fold concerns — splitting creates artificial
			//   module boundaries that hurt consumers.
			// F002: library defines event types but doesn't own the catalog;
			//   the consumer registers events in their own catalog.
			// F006: library defines PII-bearing payloads but the consumer
			//   configures encryption middleware.
			// F010: library offers hierarchical queries; recursive CTEs or
			//   other strategies are the consumer's choice (graph is opt-in).
			// F011: library performs multi-table reads/writes in read models;
			//   relational projection is the consumer's deployment choice.
			// S002: same as F006 — library cannot force encryption on consumers.
			// S003: library creates events without signing; the consumer wires
			//   signing middleware at the bus boundary.
			//
			// Note: F009 and S007 already self-skip under Server=false, so they
			// are not listed here to avoid redundant disables.
			// V007/F030: a library legitimately references surfaces that v5
			// removes (backward-compat re-exports, deprecated shells) while v4
			// still ships them — parity with IsLibrarySelfLint for in-repo code.
			Disable: []string{
				"E003",
				"E016", // architecture: domain-package mixing
				"F002",
				"F006",
				"F010",
				"F011", // adoption coaching (consumer's job)
				"F015",
				"F022",
				"F023",
				"F024",
				"F025",
				"F026", // metaengine coaching (consumer's deployment choice)
				"F030", // deprecated transport/http adoption (consumer's choice)
				"S002",
				"S003", // security middleware (consumer wires it)
				"V007", // v5-removed-API self-reference (compat surface)
			},
		},
	},
	PresetReadOnly: {
		Features: ConfigFeatures{
			CommandFlow: new(CommandFlowReadOnly),
		},
	},
	PresetLibraryFramework: {
		Features: ConfigFeatures{
			Server:      new(false),
			CommandFlow: new(CommandFlowReadOnly),
			Tracing:     new(TracingOff),
			Snapshot:    new(SnapshotOff),
		},
		Rules: RulesConfig{
			// Same disables as PresetLibrary, PLUS every F-series rule.
			// A framework provides building blocks but cannot dictate how
			// consumers adopt them — all adoption coaching is noise.
			// V007 likewise: a framework carries backward-compat surfaces.
			Disable: []string{
				"E003", "E016",
				"F001", "F002", "F003", "F004", "F005", "F006", "F007",
				"F008", "F009", "F010", "F011", "F012", "F013", "F014",
				"F015", "F016", "F017", "F018", "F019", "F020", "F021",
				"F022", "F023", "F024", "F025", "F026", "F027", "F028",
				"F029", "F030",
				"S002", "S003",
				"V007",
			},
		},
	},
}

// ValidPresetNames returns the sorted list of valid preset names (excluding the
// empty default). Used by init for error messages and by validation to detect
// typos in the "preset" config key.
func ValidPresetNames() []string {
	names := make([]string, 0, len(PresetDefinitions))
	for name := range PresetDefinitions {
		names = append(names, string(name))
	}
	sort.Strings(names)
	return names
}

// IsKnownPreset reports whether name is a recognized preset (including the
// empty default PresetNone).
func IsKnownPreset(name ConfigPreset) bool {
	if name == PresetNone {
		return true
	}
	_, ok := PresetDefinitions[name]
	return ok
}

// ResolvePresetDefinition returns the full definition for a preset name.
// Returns a zero PresetDefinition (no overrides) for PresetNone or unknown
// names. Unknown names should be caught by validation before reaching here.
func ResolvePresetDefinition(name ConfigPreset) PresetDefinition {
	if name == PresetNone {
		return PresetDefinition{}
	}
	return PresetDefinitions[name]
}

// ResolvePreset returns the ConfigFeatures for a preset name.
// Returns an empty ConfigFeatures (no overrides) for PresetNone or unknown names.
func ResolvePreset(name ConfigPreset) ConfigFeatures {
	return ResolvePresetDefinition(name).Features
}
