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
}

// DetectUsedModules scans the analyzed packages and Go files against the
// catalog's import hints, returning a usage map keyed by ModuleKey. Every
// scored (non-core) catalog entry gets an entry in the result map — either
// UsageImported (if detected) or UsageAbsent (if not).
//
// The detection uses two passes mirroring feature_detect.go:
//
//	Pass 1:  pkg.Imports scan (populated by go/packages)
//	Pass 1b: AST import declarations (fallback for test contexts)
func DetectUsedModules(
	pkgs []*packages.Package,
	gofiles []*GoFile,
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
			matchModule(result, catalog, imp.PkgPath)
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
			matchModule(result, catalog, path)
		}
	}

	return result
}

// matchModule checks if an import path matches any catalog entry's import
// hints at a path boundary, and if so, upgrades the usage status to
// UsageImported. Uses path-boundary matching to prevent false matches
// (e.g. "go-cqrs-lite/id" must not match "go-cqrs-lite/idempotency").
func matchModule(usage map[ModuleKey]ModuleUsage, catalog Catalog, importPath string) {
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
				}
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
