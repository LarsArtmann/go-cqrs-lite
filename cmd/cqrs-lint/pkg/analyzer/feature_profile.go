package analyzer

import (
	"fmt"
	"sort"
	"strings"
)

// FeatureProfile captures which go-cqrs-lite features a consumer project uses.
// It centralizes "what kind of system is this?" as a set of feature flags, each
// mapping directly to a go-cqrs-lite module. Detectors consult the profile via
// ctx.FeatureProfile instead of independently re-deriving project context.
//
// The vocabulary is grounded in the library's own modules — not in abstract
// deployment archetypes — so each flag is unambiguous and auto-detectable from
// import + constructor scans.
type FeatureProfile struct {
	// Store is the persistence backend the consumer wires up.
	Store StoreKind
	// CommandFlow classifies how (or if) the consumer dispatches commands.
	CommandFlow CommandFlowKind
	// HasServer is true when a network listener (HTTP or gRPC) is present.
	HasServer bool
	// HasSoftDelete is true when the domain emits tombstone-like events.
	HasSoftDelete bool
	// Tracing indicates whether OpenTelemetry middleware is wired.
	Tracing TracingKind
	// Snapshot indicates whether a snapshot store or strategy is configured.
	Snapshot SnapshotKind
	// Domain classifies the business domain, enabling severity calibration.
	// Financial domains escalate security and money-handling rules to error.
	Domain DomainKind
	// HasAsyncBus is true when a distributed event bus (Watermill-backed)
	// is wired. In-memory buses don't need dedup; distributed buses do.
	HasAsyncBus bool
	// HasTransport is true when a CQRS transport layer (transport/http,
	// transport/grpc, or an external cqrs-htmx module) is wired. When true,
	// adoption rules that suggest adopting a transport module are suppressed —
	// the project already has one.
	HasTransport bool
	// ServerLocal is true when HasServer is detected but the server lacks
	// production signals (no TLS, no graceful Shutdown, no health endpoint).
	// This classifies CLI tools with embedded dashboards correctly, suppressing
	// server-only rules (health checks, Prometheus, transport suggestions).
	ServerLocal bool
}

// StoreKind enumerates the persistence backends go-cqrs-lite supports.
type StoreKind string

const (
	StoreUnknown  StoreKind = "unknown"
	StoreSQLite   StoreKind = "sqlite"
	StorePostgres StoreKind = "postgres"
	StoreMySQL    StoreKind = "mysql"
	StorePebble   StoreKind = "pebble"
	StoreMemory   StoreKind = "memory"
	StoreTurso    StoreKind = "turso"
	StoreDuckDB   StoreKind = "duckdb"
	StoreBolt     StoreKind = "bolt"
	StoreCustom   StoreKind = "custom"
	StoreNone     StoreKind = "none"
)

// IsSQL reports whether this store kind is SQL-backed (capable of
// ORDER BY / WHERE pushdown). Used by adoption rules (F022) to gate
// pushdown-relevant suggestions. KV stores (Pebble, Bolt, Memory) are not
// SQL-backed — they cannot push down filters/sorts to the storage layer.
func (s StoreKind) IsSQL() bool {
	switch s {
	case StoreSQLite, StorePostgres, StoreMySQL, StoreDuckDB, StoreCustom:
		return true
	default:
		return false
	}
}

// IsEmbedded reports whether this store runs in-process (no separate server).
// Embedded stores (SQLite, Pebble, Bolt, Memory, DuckDB) share the application
// process. Distributed stores (Postgres, MySQL, Turso) run as a separate server.
func (s StoreKind) IsEmbedded() bool {
	switch s {
	case StoreSQLite, StorePebble, StoreBolt, StoreMemory, StoreDuckDB:
		return true
	default:
		return false
	}
}

// IsDistributed reports whether this store runs as a separate server process,
// enabling multi-instance deployment. Distributed stores require network I/O.
func (s StoreKind) IsDistributed() bool {
	switch s {
	case StorePostgres, StoreMySQL, StoreTurso:
		return true
	default:
		return false
	}
}

// AllStoreKinds returns every defined StoreKind value, sorted alphabetically.
// Used by the explain command to derive valid config values programmatically
// instead of maintaining a hand-written copy.
func AllStoreKinds() []StoreKind {
	return []StoreKind{
		StoreSQLite, StorePostgres, StoreMySQL, StorePebble,
		StoreMemory, StoreTurso, StoreDuckDB, StoreBolt, StoreCustom, StoreNone,
	}
}

// CommandFlowKind classifies the command-dispatch pattern.
type CommandFlowKind string

const (
	CommandFlowUnknown  CommandFlowKind = "unknown"
	CommandFlowReadOnly CommandFlowKind = "read-only" // no dispatcher at all
	CommandFlowSync     CommandFlowKind = "sync"      // dispatcher present, batch/sync writes (no Dispatch)
	CommandFlowCommands CommandFlowKind = "commands"  // Dispatch() calls present
)

// AllCommandFlowKinds returns every defined CommandFlowKind value, sorted
// alphabetically. Excludes the Unknown sentinel.
func AllCommandFlowKinds() []CommandFlowKind {
	return []CommandFlowKind{
		CommandFlowReadOnly, CommandFlowSync, CommandFlowCommands,
	}
}

// AllTracingKinds returns every defined TracingKind value, sorted
// alphabetically. Excludes the Unknown sentinel.
func AllTracingKinds() []TracingKind {
	return []TracingKind{TracingOff, TracingOn}
}

// AllSnapshotKinds returns every defined SnapshotKind value, sorted
// alphabetically. Excludes the Unknown sentinel.
func AllSnapshotKinds() []SnapshotKind {
	return []SnapshotKind{SnapshotOff, SnapshotOn}
}

// AllDomainKinds returns every defined DomainKind value, sorted
// alphabetically. Excludes the Unknown sentinel.
func AllDomainKinds() []DomainKind {
	return []DomainKind{DomainFinancial, DomainInternal, DomainSecurity}
}

// DomainKind classifies the business domain of a consumer project.
// Financial domains get stricter severity on security and money rules.
// The domain is auto-detected from event/command type names but can be
// overridden via config.
type DomainKind string

const (
	DomainUnknown   DomainKind = "unknown"
	DomainFinancial DomainKind = "financial"
	DomainInternal  DomainKind = "internal"
	DomainSecurity  DomainKind = "security"
)

// TracingKind indicates whether OTel tracing middleware is wired.
type TracingKind string

const (
	TracingUnknown TracingKind = "unknown"
	TracingOff     TracingKind = "off"
	TracingOn      TracingKind = "on"
)

// SnapshotKind indicates whether a snapshot store or strategy is configured.
type SnapshotKind string

const (
	SnapshotUnknown SnapshotKind = "unknown"
	SnapshotOff     SnapshotKind = "off"
	SnapshotOn      SnapshotKind = "on"
)

// String returns a human-readable multi-line summary for the doctor command
// and --verbose output.
func (fp FeatureProfile) String() string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "store:         %s\n", fp.Store)
	_, _ = fmt.Fprintf(&b, "command-flow:  %s\n", fp.CommandFlow)
	_, _ = fmt.Fprintf(&b, "server:        %t\n", fp.HasServer)
	_, _ = fmt.Fprintf(&b, "soft-delete:   %t\n", fp.HasSoftDelete)
	_, _ = fmt.Fprintf(&b, "tracing:       %s\n", fp.Tracing)
	_, _ = fmt.Fprintf(&b, "snapshot:      %s\n", fp.Snapshot)
	_, _ = fmt.Fprintf(&b, "domain:        %s\n", fp.Domain)
	_, _ = fmt.Fprintf(&b, "transport:     %t\n", fp.HasTransport)
	_, _ = fmt.Fprintf(&b, "server-local:  %t\n", fp.ServerLocal)
	_, _ = fmt.Fprintf(&b, "async-bus:     %t\n", fp.HasAsyncBus)
	return b.String()
}

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
			Disable: []string{
				"E003", "E016", // architecture: domain-package mixing
				"F002", "F006", "F010", "F011", // adoption coaching (consumer's job)
				"F015", "F022", // metaengine coaching (consumer's deployment choice)
				"S002", "S003", // security middleware (consumer wires it)
			},
		},
	},
	PresetReadOnly: {
		Features: ConfigFeatures{
			CommandFlow: new(CommandFlowReadOnly),
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

// ResolveFeatureProfile merges auto-detected values with config overrides.
// For each field: if the config value is non-nil, it wins; otherwise the
// detected value is used. This lets users pin specific flags while letting
// auto-detection handle the rest.
func ResolveFeatureProfile(
	cfg ConfigFeatures,
	preset ConfigPreset,
	detected FeatureProfile,
) FeatureProfile {
	// Start with preset defaults, then let explicit config flags override.
	merged := ResolvePreset(preset)
	mergeConfigFeatures(&merged, cfg)

	result := detected

	if merged.Store != nil {
		result.Store = *merged.Store
	}
	if merged.CommandFlow != nil {
		result.CommandFlow = *merged.CommandFlow
	}
	if merged.Server != nil {
		result.HasServer = *merged.Server
	}
	if merged.SoftDelete != nil {
		result.HasSoftDelete = *merged.SoftDelete
	}
	if merged.Tracing != nil {
		result.Tracing = *merged.Tracing
	}
	if merged.Snapshot != nil {
		result.Snapshot = *merged.Snapshot
	}
	if merged.Domain != nil {
		result.Domain = *merged.Domain
	}
	if merged.Transport != nil {
		result.HasTransport = *merged.Transport
	}
	if merged.ServerLocal != nil {
		result.ServerLocal = *merged.ServerLocal
	}
	if merged.AsyncBus != nil {
		result.HasAsyncBus = *merged.AsyncBus
	}

	return result
}

// mergeConfigFeatures overlays src onto dst (src wins where non-nil).
func mergeConfigFeatures(dst *ConfigFeatures, src ConfigFeatures) {
	if src.Store != nil {
		dst.Store = src.Store
	}
	if src.CommandFlow != nil {
		dst.CommandFlow = src.CommandFlow
	}
	if src.Server != nil {
		dst.Server = src.Server
	}
	if src.SoftDelete != nil {
		dst.SoftDelete = src.SoftDelete
	}
	if src.Tracing != nil {
		dst.Tracing = src.Tracing
	}
	if src.Snapshot != nil {
		dst.Snapshot = src.Snapshot
	}
	if src.Domain != nil {
		dst.Domain = src.Domain
	}
	if src.Transport != nil {
		dst.Transport = src.Transport
	}
	if src.ServerLocal != nil {
		dst.ServerLocal = src.ServerLocal
	}
	if src.AsyncBus != nil {
		dst.AsyncBus = src.AsyncBus
	}
}

// ToConfigFeatures is the inverse of ResolveFeatureProfile: it projects a
// resolved FeatureProfile back into a ConfigFeatures, including only the flags
// that carry meaningful (non-unknown) values. The doctor command uses this to
// emit a copy-pasteable, always-valid JSON config suggestion. Because the
// result is built from explicit pointers and serialized via encoding/json,
// it can never produce the trailing-comma corruption that hand-formatted JSON
// is prone to.
func (fp FeatureProfile) ToConfigFeatures() ConfigFeatures {
	cf := ConfigFeatures{
		Server:     &fp.HasServer,
		SoftDelete: &fp.HasSoftDelete,
	}
	if fp.Store != "" && fp.Store != StoreUnknown && fp.Store != StoreNone {
		cf.Store = &fp.Store
	}
	if fp.CommandFlow != "" && fp.CommandFlow != CommandFlowUnknown {
		cf.CommandFlow = &fp.CommandFlow
	}
	if fp.Tracing != "" && fp.Tracing != TracingUnknown {
		cf.Tracing = &fp.Tracing
	}
	if fp.Snapshot != "" && fp.Snapshot != SnapshotUnknown {
		cf.Snapshot = &fp.Snapshot
	}
	if fp.Domain != "" && fp.Domain != DomainUnknown {
		cf.Domain = &fp.Domain
	}
	if fp.HasTransport {
		cf.Transport = &fp.HasTransport
	}
	if fp.ServerLocal {
		cf.ServerLocal = &fp.ServerLocal
	}
	if fp.HasAsyncBus {
		cf.AsyncBus = &fp.HasAsyncBus
	}
	return cf
}
