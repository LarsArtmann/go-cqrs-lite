package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// renderDoctorSuggestedConfig outputs a copy-pasteable .cqrs-lint.json features
// section based on the detected profile. In multi-module workspaces, the most
// permissive profile across all modules is used so pinning does not silence
// findings in sub-modules with richer feature sets.
func renderDoctorSuggestedConfig(w io.Writer, cfg *AppConfig, actx *analyzer.AnalysisContext) {
	profile := actx.FeatureProfile
	isMultiModule := len(actx.FeatureProfiles) > 1

	if isMultiModule {
		profile = mergeMostPermissiveProfile(actx.FeatureProfiles)
	}

	features := profile.ToConfigFeatures()

	raw, err := json.Marshal(
		map[string]analyzer.ConfigFeatures{"features": features},
		jsontext.WithIndentPrefix(""),
		jsontext.WithIndent("  "),
	)
	if err != nil {
		return
	}

	_, _ = fmt.Fprintln(w, "SUGGESTED .cqrs-lint.json")
	_, _ = fmt.Fprintln(w, "─────────────────────────")
	_, _ = fmt.Fprintln(w)
	if isMultiModule {
		_, _ = fmt.Fprintln(
			w,
			"  Multi-module workspace: using the MOST PERMISSIVE profile across all modules",
		)
		_, _ = fmt.Fprintln(
			w,
			"  so pinning does not silence findings in sub-modules with richer feature sets.",
		)
		_, _ = fmt.Fprintln(w)
	} else {
		_, _ = fmt.Fprintln(
			w,
			"  Copy-paste to pin the detected profile (prevents auto-detection drift):",
		)
		_, _ = fmt.Fprintln(w)
	}
	_, _ = fmt.Fprintln(w, string(raw))

	// Show rules overrides if loaded
	if len(cfg.Rules.ExternalAPIStructPrefixes) > 0 {
		rulesRaw, err := json.Marshal(
			map[string]analyzer.RulesConfig{"rules": cfg.Rules},
			jsontext.WithIndentPrefix(""),
			jsontext.WithIndent("  "),
		)
		if err == nil {
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, "  Loaded rules overrides:")
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, string(rulesRaw))
		}
	}

	_, _ = fmt.Fprintln(w)
}

// mergeMostPermissiveProfile computes the union of all per-module profiles,
// enabling every feature that ANY module uses. This prevents pinning the
// workspace-level config from silencing findings in sub-modules with richer
// feature sets (e.g. an api module with server=true in a workspace where the
// root module is server=false).
func mergeMostPermissiveProfile(
	profiles map[string]analyzer.FeatureProfile,
) analyzer.FeatureProfile {
	var result analyzer.FeatureProfile

	for _, p := range profiles {
		result.HasServer = result.HasServer || p.HasServer
		result.HasSoftDelete = result.HasSoftDelete || p.HasSoftDelete
		result.HasAsyncBus = result.HasAsyncBus || p.HasAsyncBus
		result.HasTransport = result.HasTransport || p.HasTransport
		result.HasMetaengine = result.HasMetaengine || p.HasMetaengine
		result.MetaenginePushdown = result.MetaenginePushdown || p.MetaenginePushdown

		if len(p.MetaengineEngines) > 0 {
			result.MetaengineEngines = append(result.MetaengineEngines, p.MetaengineEngines...)
		}

		result.CommandFlow = mostPermissiveCommandFlow(result.CommandFlow, p.CommandFlow)
		result.Tracing = mostPermissiveTracing(result.Tracing, p.Tracing)
		result.Snapshot = mostPermissiveSnapshot(result.Snapshot, p.Snapshot)
		result.Store = mostPermissiveStore(result.Store, p.Store)
		result.Domain = mostPermissiveDomain(result.Domain, p.Domain)
	}

	return result
}

func mostPermissiveCommandFlow(a, b analyzer.CommandFlowKind) analyzer.CommandFlowKind {
	order := map[analyzer.CommandFlowKind]int{
		analyzer.CommandFlowUnknown:  0,
		analyzer.CommandFlowReadOnly: 1,
		analyzer.CommandFlowSync:     2,
		analyzer.CommandFlowCommands: 3,
	}
	if order[b] > order[a] {
		return b
	}
	return a
}

func mostPermissiveTracing(a, b analyzer.TracingKind) analyzer.TracingKind {
	if b == analyzer.TracingOn {
		return b
	}
	return a
}

func mostPermissiveSnapshot(a, b analyzer.SnapshotKind) analyzer.SnapshotKind {
	if b == analyzer.SnapshotOn {
		return b
	}
	return a
}

func mostPermissiveStore(a, b analyzer.StoreKind) analyzer.StoreKind {
	if a == analyzer.StoreUnknown || a == analyzer.StoreNone || a == "" {
		return b
	}
	return a
}

func mostPermissiveDomain(a, b analyzer.DomainKind) analyzer.DomainKind {
	order := map[analyzer.DomainKind]int{
		analyzer.DomainUnknown:   0,
		analyzer.DomainInternal:  1,
		analyzer.DomainSecurity:  2,
		analyzer.DomainFinancial: 3,
	}
	if order[b] > order[a] {
		return b
	}
	return a
}
