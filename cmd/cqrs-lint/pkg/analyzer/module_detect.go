package analyzer

import (
	"strings"

	"golang.org/x/tools/go/packages"
)

// UsageStatus classifies how a consumer project uses a go-cqrs-lite module.
type UsageStatus int

const (
	// UsageAbsent means the module is not imported at all.
	UsageAbsent UsageStatus = iota
	// UsageImported means the module's import path appears in the consumer's
	// code. v1 counts this as "used" — importing IS adoption.
	UsageImported
	// UsageActive means a constructor call from the module was found at the
	// AST level. This is a stronger signal than UsageImported and is reserved
	// for future "stale import" detection. v1 does not require it.
	UsageActive
)

// String returns a human-readable status label for table rendering.
func (s UsageStatus) String() string {
	switch s {
	case UsageAbsent:
		return "missing"
	case UsageImported:
		return "used"
	case UsageActive:
		return "active"
	default:
		return "missing"
	}
}

// ModuleUsage records the detected usage status of one catalog module.
type ModuleUsage struct {
	Key      ModuleKey
	Status   UsageStatus
	Evidence string // e.g. "import in main.go" — empty when absent
	// TestOnly is true when the module reached UsageImported exclusively
	// through _test.go imports (Pass 2): production code never imports it.
	// Rows flagged this way render a "test-only import" provenance marker.
	TestOnly bool
}

// DetectUsedModules scans the analyzed packages and Go files against the
// catalog's import hints, returning a usage map keyed by ModuleKey. Every
// scored (non-core) catalog entry gets an entry in the result map — either
// UsageImported (if detected) or UsageAbsent (if not).
//
// The detection uses three passes mirroring feature_detect.go:
//
//	Pass 1:  pkg.Imports scan (populated by go/packages)
//	Pass 1b: AST import declarations (fallback for test contexts)
//	Pass 2:  import paths collected from _test.go files
//
// Pass 2 exists because the typed load runs with Tests: false: test files
// reach neither pkg.Imports nor GoFiles, so test-only modules
// (systemscenario, scenario) are detectable ONLY through their _test.go
// imports. Importing IS adoption — in tests as anywhere else.
func DetectUsedModules(
	pkgs []*packages.Package,
	gofiles []*GoFile,
	testImportPaths []string,
	catalog Catalog,
) map[ModuleKey]ModuleUsage {
	result := make(map[ModuleKey]ModuleUsage, len(catalog.Scored()))
	for _, e := range catalog.Scored() {
		result[e.Key] = ModuleUsage{Key: e.Key, Status: UsageAbsent}
	}

	// Pass 1: import-based detection via pkg.Imports.
	for _, pkg := range pkgs {
		if pkg == nil || len(pkg.Errors) > 0 {
			continue
		}
		for _, imp := range pkg.Imports {
			if imp == nil {
				continue
			}
			matchModule(result, catalog, imp.PkgPath, false)
		}
	}

	// Pass 1b: AST import-path fallback. Supplements Pass 1 which uses
	// pkg.Imports (populated by go/packages). In test contexts or when
	// pkg.Imports is incomplete, AST import declarations are the only source.
	for _, gf := range gofiles {
		if gf == nil || gf.IsTest || gf.AST == nil {
			continue
		}
		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}
			path := strings.Trim(imp.Path.Value, `"`)
			matchModule(result, catalog, path, false)
		}
	}

	// Pass 2: test-file imports. The typed load (Tests: false) never surfaces
	// _test.go files, so test-only modules are invisible to Passes 1/1b.
	// Callers collect these via CollectTestImportPaths (go/parser, imports
	// only); matching reuses matchModule/pathBoundaryMatch unchanged. A
	// module first matched HERE (Passes 1/1b left it absent) carries
	// TestOnly=true — the scorecard renders it as test-only adoption.
	for _, path := range testImportPaths {
		matchModule(result, catalog, path, true)
	}

	return result
}

// matchModule checks if an import path matches any catalog entry's import
// hints at a path boundary, and if so, upgrades the usage status to
// UsageImported. testOnly marks upgrades driven by _test.go imports (Pass 2)
// so provenance survives into the scorecard. Uses path-boundary matching to
// prevent false matches (e.g. "go-cqrs-lite/id" must not match
// "go-cqrs-lite/idempotency").
//
// Several distinct imports can match the same entry (module root plus
// subpackages, e.g. catalog/v4, catalog/v4/simple, catalog/v4/docserver).
// Callers iterate maps with randomized order, so the evidence records the
// lexicographically smallest matched path — the scorecard output stays
// deterministic no matter which import the scan sees first.
func matchModule(usage map[ModuleKey]ModuleUsage, catalog Catalog, importPath string, testOnly bool) {
	for _, e := range catalog.Scored() {
		for _, hint := range e.ImportHints {
			if !pathBoundaryMatch(importPath, hint) {
				continue
			}
			entry := usage[e.Key]
			if entry.Status == UsageAbsent {
				usage[e.Key] = ModuleUsage{
					Key:      e.Key,
					Status:   UsageImported,
					Evidence: importPath,
					TestOnly: testOnly,
				}
			} else if importPath < entry.Evidence {
				entry.Evidence = importPath
				usage[e.Key] = entry
			}
			break // one match per module is enough
		}
	}
}

// pathBoundaryMatch checks whether importPath contains hint at a path
// boundary: the hint must end at a '/' delimiter or at the end of the path.
// This prevents "go-cqrs-lite/id" from matching "go-cqrs-lite/idempotency".
func pathBoundaryMatch(importPath, hint string) bool {
	idx := strings.Index(importPath, hint)
	if idx < 0 {
		return false
	}
	end := idx + len(hint)
	if end >= len(importPath) {
		return true // hint matches at end of path
	}
	// Path boundary: next char must be '/' for a clean module-path match.
	return importPath[end] == '/'
}

// UsedKeys extracts the set of module keys with status >= UsageImported
// from a usage map. Used by ComputeScorecard to determine the numerator.
func UsedKeys(usage map[ModuleKey]ModuleUsage) []ModuleKey {
	keys := make([]ModuleKey, 0, len(usage))
	for key, u := range usage {
		if u.Status >= UsageImported {
			keys = append(keys, key)
		}
	}
	return keys
}
