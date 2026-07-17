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
}

// StoreKind enumerates the persistence backends go-cqrs-lite supports.
type StoreKind string

const (
	StoreUnknown  StoreKind = "unknown"
	StoreSQLite   StoreKind = "sqlite"
	StorePostgres StoreKind = "postgres"
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
	fmt.Fprintf(&b, "store:         %s\n", fp.Store)
	fmt.Fprintf(&b, "command-flow:  %s\n", fp.CommandFlow)
	fmt.Fprintf(&b, "server:        %t\n", fp.HasServer)
	fmt.Fprintf(&b, "soft-delete:   %t\n", fp.HasSoftDelete)
	fmt.Fprintf(&b, "tracing:       %s\n", fp.Tracing)
	fmt.Fprintf(&b, "snapshot:      %s\n", fp.Snapshot)
	return b.String()
}

// ConfigFeatures mirrors FeatureProfile but uses pointer fields so the config
// loader can distinguish "user did not set this" (nil) from "user set the zero
// value". Each non-nil field overrides the auto-detected value.
type ConfigFeatures struct {
	Store       *StoreKind       `json:"store,omitempty"`
	CommandFlow *CommandFlowKind `json:"command-flow,omitempty"`
	Server      *bool            `json:"server,omitempty"`
	SoftDelete  *bool            `json:"soft-delete,omitempty"`
	Tracing     *TracingKind     `json:"tracing,omitempty"`
	Snapshot    *SnapshotKind    `json:"snapshot,omitempty"`
}

// ConfigPreset is a named set of feature-flag defaults. Presets are sugar:
// they expand to ConfigFeatures values, and explicit flags always override.
// The flags are the source of truth; presets are convenience.
type ConfigPreset string

const (
	PresetNone       ConfigPreset = ""
	PresetLocalCLI   ConfigPreset = "local-cli"
	PresetProduction ConfigPreset = "production"
	PresetLibrary    ConfigPreset = "library"
	PresetReadOnly   ConfigPreset = "read-only"
)

// Presets maps preset names to their feature-flag defaults. A nil pointer
// means "leave as auto-detected" — the preset only pins the flags that matter
// for its intent.
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
}

//go:fix inline
func ptr[T any](v T) *T {
	return new(v)
}
