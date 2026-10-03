package analyzer

import (
	"fmt"
	"slices"
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
	// Store is the primary persistence backend: the first signal in the
	// historical first-wins detection order (stack preset > engine import >
	// storage/ custom > sqlite-driver fallback > metaengine memory
	// inference). Kept as a scalar for single-store consumers and rule
	// suggestion text; multi-backend systems should consult Stores.
	Store StoreKind `json:"store"`
	// Stores lists every distinct detected backend signal (deduplicated,
	// detection order) — mixed pools such as a postgres journal plus
	// sqlite projection engines. Empty when no backend signal was found.
	Stores []StoreKind `json:"stores,omitempty"`
	// CommandFlow classifies how (or if) the consumer dispatches commands.
	CommandFlow CommandFlowKind `json:"commandFlow"`
	// HasServer is true when a network listener (HTTP or gRPC) is present.
	HasServer bool `json:"hasServer"`
	// HasSoftDelete is true when the domain emits tombstone-like events.
	HasSoftDelete bool `json:"hasSoftDelete"`
	// Tracing indicates whether OpenTelemetry middleware is wired.
	Tracing TracingKind `json:"tracing"`
	// Snapshot indicates whether a snapshot store or strategy is configured.
	Snapshot SnapshotKind `json:"snapshot"`
	// Domain classifies the business domain, enabling severity calibration.
	// Financial domains escalate security and money-handling rules to error.
	Domain DomainKind `json:"domain"`
	// HasAsyncBus is true when a distributed event bus (Watermill-backed)
	// is wired. In-memory buses don't need dedup; distributed buses do.
	HasAsyncBus bool `json:"hasAsyncBus"`
	// HasTransport is true when an external-delivery layer is wired: the
	// watermill/ bridge (any broker backend), go-sse, cqrs-htmx, or a legacy
	// deprecated transport/http / transport/grpc import. When true, adoption
	// rules that suggest adopting a transport are suppressed — the project
	// already has one.
	HasTransport bool `json:"hasTransport"`
	// ServerLocal is true when HasServer is detected but the server lacks
	// production signals (no TLS, no graceful Shutdown, no health endpoint).
	// This classifies CLI tools with embedded dashboards correctly, suppressing
	// server-only rules (health checks, Prometheus, transport suggestions).
	ServerLocal bool `json:"serverLocal"`
	// HasMetaengine is true when the project imports the metaengine module.
	// Adoption rules (F022-F025) use this to gate pushdown suggestions.
	HasMetaengine bool `json:"hasMetaengine"`
	// MetaengineEngines lists the engine backends wired by the project
	// (e.g. "sqlite", "pebble", "duckdb", "postgres", "memory").
	// Detected from imports of metaengine/<engine>engine subpackages.
	MetaengineEngines []string `json:"metaengineEngines"`
	// MetaenginePushdown is true when the project uses FilterOnField or
	// SortOnField — indicating it has adopted declarative pushdown.
	MetaenginePushdown bool `json:"metaenginePushdown"`
	// MetaengineQueryCount is the number of metaengine.Query declarations
	// found; MetaengineDeclarativeQueries counts those carrying
	// FilterOnField or SortOnField. They make the pushdown profile line
	// actionable instead of a bare boolean (nsfw-classifier feedback,
	// 2026-10-03).
	MetaengineQueryCount        int `json:"metaengineQueryCount,omitempty"`
	MetaengineDeclarativeQueries int `json:"metaengineDeclarativeQueries,omitempty"`
	// Monetary declares whether the project handles monetary values.
	// Unknown (the default) lets money rules infer the signal from source
	// heuristics; "on"/"off" are explicit user declarations that override
	// the inference — e.g. C008 downgrades to Info when "off" is declared
	// for a project whose struct names merely look monetary.
	Monetary MonetaryKind `json:"monetary"`
}

func (fp FeatureProfile) String() string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "store:         %s\n", fp.Store)
	if effective := fp.EffectiveStores(); len(effective) > 1 {
		names := make([]string, len(effective))
		for i, kind := range effective {
			names[i] = string(kind)
		}
		_, _ = fmt.Fprintf(&b, "stores:        %s\n", strings.Join(names, ", "))
	}
	_, _ = fmt.Fprintf(&b, "command-flow:  %s\n", fp.CommandFlow)
	_, _ = fmt.Fprintf(&b, "server:        %t\n", fp.HasServer)
	_, _ = fmt.Fprintf(&b, "soft-delete:   %t\n", fp.HasSoftDelete)
	_, _ = fmt.Fprintf(&b, "tracing:       %s\n", fp.Tracing)
	_, _ = fmt.Fprintf(&b, "snapshot:      %s\n", fp.Snapshot)
	if fp.Domain == DomainUnknown {
		// Surface the pinnable vocabulary — detection only ever returns
		// financial or unknown, so internal/security are otherwise
		// undiscoverable (nsfw-classifier feedback, 2026-10-03).
		_, _ = fmt.Fprintf(&b, "domain:        unknown (pinnable: internal, security, financial)\n")
	} else {
		_, _ = fmt.Fprintf(&b, "domain:        %s\n", fp.Domain)
	}
	_, _ = fmt.Fprintf(&b, "transport:     %t\n", fp.HasTransport)
	_, _ = fmt.Fprintf(&b, "server-local:  %t\n", fp.ServerLocal)
	_, _ = fmt.Fprintf(&b, "async-bus:     %t\n", fp.HasAsyncBus)
	_, _ = fmt.Fprintf(&b, "metaengine:    %t\n", fp.HasMetaengine)
	if len(fp.MetaengineEngines) > 0 {
		_, _ = fmt.Fprintf(&b, "  engines:     %s\n", strings.Join(fp.MetaengineEngines, ", "))
		if len(fp.MetaengineEngines) == 1 && fp.MetaengineEngines[0] == "memory" {
			_, _ = fmt.Fprintln(
				&b,
				"               (built-in driver, inferred — or an in-app custom engine)",
			)
		}
	}
	if fp.MetaengineQueryCount > 0 {
		if fp.MetaenginePushdown {
			_, _ = fmt.Fprintf(&b, "  pushdown:    %t (%d/%d queries declarative)\n",
				fp.MetaenginePushdown, fp.MetaengineDeclarativeQueries, fp.MetaengineQueryCount)
		} else {
			_, _ = fmt.Fprintf(&b, "  pushdown:    false (%d queries, 0 declarative — declare "+
				"FilterOnField/SortOnField; free on the memory engine)\n", fp.MetaengineQueryCount)
		}
	} else {
		_, _ = fmt.Fprintf(&b, "  pushdown:    %t\n", fp.MetaenginePushdown)
	}
	_, _ = fmt.Fprintf(&b, "monetary:      %s\n", fp.Monetary)
	return b.String()
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
		result.Store = merged.Store.Primary()
		result.Stores = slices.Clone(merged.Store.Kinds)
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
	if merged.Monetary != nil {
		result.Monetary = *merged.Monetary
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
	if src.Monetary != nil {
		dst.Monetary = src.Monetary
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
//
// Server and SoftDelete are suggested only as POSITIVE evidence (true): a
// false detection is absence-of-evidence within analyzed packages, and
// packages that don't import go-cqrs-lite are outside the scan scope —
// pinning "server": false as project truth silenced real server rules for a
// consumer whose server lived in a non-importer package (nsfw-classifier
// feedback, 2026-10-03).
func (fp FeatureProfile) ToConfigFeatures() ConfigFeatures {
	cf := ConfigFeatures{}
	if fp.HasServer {
		cf.Server = &fp.HasServer
	}
	if fp.HasSoftDelete {
		cf.SoftDelete = &fp.HasSoftDelete
	}
	if effective := fp.EffectiveStores(); len(effective) > 0 {
		spec := StoreSpec{Kinds: effective}
		cf.Store = &spec
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
	if fp.Monetary != "" && fp.Monetary != MonetaryUnknown {
		cf.Monetary = &fp.Monetary
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
