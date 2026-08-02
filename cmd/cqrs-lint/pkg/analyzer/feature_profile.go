package analyzer

import (
	"fmt"
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
	StoreCustom   StoreKind = "custom"
	StoreNone     StoreKind = "none"
)

// CommandFlowKind classifies the command-dispatch pattern.
type CommandFlowKind string

const (
	CommandFlowUnknown  CommandFlowKind = "unknown"
	CommandFlowReadOnly CommandFlowKind = "read-only" // no dispatcher at all
	CommandFlowSync     CommandFlowKind = "sync"      // dispatcher present, batch/sync writes (no Dispatch)
	CommandFlowCommands CommandFlowKind = "commands"  // Dispatch() calls present
)

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

// Presets maps preset names to their feature-flag defaults. A nil pointer
// means "leave as auto-detected" — the preset only pins the flags that matter
// for its intent.
//
//nolint:gochecknoglobals // read-only lookup table
var Presets = map[ConfigPreset]ConfigFeatures{
	PresetLocalCLI: {
		Server:  new(false),
		Tracing: new(TracingOff),
	},
	PresetProduction: {
		Server:  new(true),
		Tracing: new(TracingOn),
	},
	PresetLibrary: {
		Server:      new(false),
		CommandFlow: new(CommandFlowReadOnly),
		Tracing:     new(TracingOff),
		Snapshot:    new(SnapshotOff),
	},
	PresetReadOnly: {
		CommandFlow: new(CommandFlowReadOnly),
	},
}

// ResolvePreset returns the ConfigFeatures for a preset name.
// Returns an empty ConfigFeatures (no overrides) for PresetNone or unknown names.
func ResolvePreset(name ConfigPreset) ConfigFeatures {
	if name == PresetNone {
		return ConfigFeatures{}
	}
	return Presets[name]
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
	return cf
}
