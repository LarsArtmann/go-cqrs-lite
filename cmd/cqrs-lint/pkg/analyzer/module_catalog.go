package analyzer

import "slices"

// ModuleCategory groups adoptable modules for display grouping and
// recommendation priority ordering.
type ModuleCategory string

const (
	CategoryCore          ModuleCategory = "Core"
	CategoryPersistence   ModuleCategory = "Persistence"
	CategoryObservability ModuleCategory = "Observability"
	CategorySecurity      ModuleCategory = "Security"
	CategoryReliability   ModuleCategory = "Reliability"
	CategorySchema        ModuleCategory = "Schema"
	CategoryProjections   ModuleCategory = "Projections"
	CategoryMessaging     ModuleCategory = "Messaging"
	CategoryWorkflow      ModuleCategory = "Workflow"
	CategoryDocumentation ModuleCategory = "Documentation"
	CategoryOptimization  ModuleCategory = "Optimization"
)

// categoryPriority orders categories for recommendation sorting. Lower number
// = higher priority. Missing categories default to 99 (lowest priority).
var categoryPriority = map[ModuleCategory]int{
	CategorySecurity:      1,
	CategoryReliability:   2,
	CategoryObservability: 3,
	CategoryPersistence:   4,
	CategorySchema:        5,
	CategoryProjections:   6,
	CategoryWorkflow:      7,
	CategoryMessaging:     8,
	CategoryOptimization:  9,
	CategoryDocumentation: 10,
	CategoryCore:          99,
}

// ModuleKey uniquely identifies a catalog entry (e.g. "scheduling", "stack/sqlite").
type ModuleKey string

// ModuleEntry describes one adoptable go-cqrs-lite module in the catalog.
type ModuleEntry struct {
	Key         ModuleKey
	DisplayName string
	Category    ModuleCategory
	// ImportHints are substrings matched against consumer import paths.
	// A package import path containing any hint marks this module as used.
	ImportHints []string
	// Description is a one-line summary of what the module provides.
	Description string
	// Suggestion is the "Consider:" text shown when the module is missing
	// and relevant for the detected profile.
	Suggestion string
	// Profiles restricts relevance to specific presets. Empty = all profiles.
	// Non-empty = only relevant when the project's effective preset matches.
	Profiles []ConfigPreset
	// Core marks foundational modules (event, command, etc.) that are
	// always used. Core modules are excluded from the scorecard denominator
	// and numerator — they are infrastructure, not adoption decisions.
	Core bool
}

// Catalog is the canonical universe of adoptable go-cqrs-lite modules.
type Catalog struct {
	entries []ModuleEntry
}

// All returns all catalog entries in canonical order.
func (c Catalog) All() []ModuleEntry {
	return c.entries
}

// Scored returns only non-core entries (the ones counted in the scorecard).
func (c Catalog) Scored() []ModuleEntry {
	result := make([]ModuleEntry, 0, len(c.entries))
	for _, e := range c.entries {
		if !e.Core {
			result = append(result, e)
		}
	}
	return result
}

// Core returns only core entries.
func (c Catalog) Core() []ModuleEntry {
	result := make([]ModuleEntry, 0, 6)
	for _, e := range c.entries {
		if e.Core {
			result = append(result, e)
		}
	}
	return result
}

// Get returns the entry for the given key, or false.
func (c Catalog) Get(key ModuleKey) (ModuleEntry, bool) {
	for _, e := range c.entries {
		if e.Key == key {
			return e, true
		}
	}
	return ModuleEntry{}, false
}

// Keys returns all entry keys in canonical order.
func (c Catalog) Keys() []ModuleKey {
	keys := make([]ModuleKey, len(c.entries))
	for i, e := range c.entries {
		keys[i] = e.Key
	}
	return keys
}

// RelevantFor returns entries relevant to the given feature profile and
// explicit preset. Core entries are excluded. Profile-restricted entries
// (transport, server infra) are excluded when the profile indicates they
// are irrelevant (e.g. a local CLI without a server).
func (c Catalog) RelevantFor(fp FeatureProfile, preset ConfigPreset) []ModuleEntry {
	result := make([]ModuleEntry, 0, len(c.entries))
	for _, e := range c.entries {
		if e.RelevantForProfile(fp, preset) {
			result = append(result, e)
		}
	}
	return result
}

// ByCategory groups entries by category, returning categories in priority order.
func (c Catalog) ByCategory() map[ModuleCategory][]ModuleEntry {
	result := make(map[ModuleCategory][]ModuleEntry)
	for _, e := range c.entries {
		result[e.Category] = append(result[e.Category], e)
	}
	return result
}

// RelevantForProfile reports whether this module should appear in the
// scorecard denominator for the given feature profile and explicit preset.
//
// Core modules are never scored (they are infrastructure, not adoption decisions).
// Modules with no Profiles restriction are always relevant. Modules with a
// Profiles restriction are relevant only when the project's effective preset
// matches — a local-CLI project is not penalized for missing server-only modules.
func (e ModuleEntry) RelevantForProfile(fp FeatureProfile, preset ConfigPreset) bool {
	if e.Core {
		return false
	}
	if len(e.Profiles) == 0 {
		return true
	}
	// Check explicit preset match.
	if slices.Contains(e.Profiles, preset) {
		return true
	}
	// Derive effective preset from FeatureProfile signals.
	// A production server (HasServer && !ServerLocal) implies the production preset.
	if fp.HasServer && !fp.ServerLocal {
		if slices.Contains(e.Profiles, PresetProduction) {
			return true
		}
	}
	return false
}

// CategoryPriority returns the recommendation priority for this entry's category.
// Lower number = higher priority (Security missing > Documentation missing).
func (e ModuleEntry) CategoryPriority() int {
	if p, ok := categoryPriority[e.Category]; ok {
		return p
	}
	return 99
}

// DefaultCatalog is the canonical universe of adoptable go-cqrs-lite modules.
// It contains 6 core (infrastructure) entries and 28 scored (adoptable) entries.
//
//nolint:gochecknoglobals // read-only catalog data
var DefaultCatalog = Catalog{entries: buildDefaultCatalog()}
