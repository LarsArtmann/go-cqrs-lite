package version

import (
	"go/ast"
	"strings"
)

// Path-resolution and table-lookup helpers for V007 (v5-removed-api-usage),
// split out of v007.go so the detector, the curated removal surface
// (v007_tables.go), and this resolution layer each stay under the 350-line
// limit.

// matchModule looks up a wholly-removed module by fragment.
func matchModule(module string) (deprecatedV5Module, bool) {
	for _, m := range deprecatedV5Modules {
		if m.fragment == module {
			return m, true
		}
	}

	return deprecatedV5Module{}, false
}

func matchSymbol(module, symbol string) (deprecatedV5Symbol, bool) {
	for _, s := range deprecatedV5Symbols {
		if s.fragment == module && s.symbol == symbol {
			return s, true
		}
	}

	return deprecatedV5Symbol{}, false
}

// resolveQualifier maps a package qualifier to its import path using the
// file's import declarations (alias-aware). Blank and dot imports are
// skipped: a dot-imported package has no qualifier, and matching one would
// falsely attribute unrelated selectors.
func resolveQualifier(file *ast.File, qualifier string) (string, bool) {
	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)

		if imp.Name == nil {
			if defaultQualifier(path) == qualifier {
				return path, true
			}

			continue
		}

		switch imp.Name.Name {
		case "_", ".":
			continue
		case qualifier:
			return path, true
		}
	}

	return "", false
}

// cqrsModuleOf strips the go-cqrs-lite prefix and major-version suffix from
// an import path, returning the module fragment (e.g. "stack/sqlite").
// Non-go-cqrs-lite paths return ok=false.
func cqrsModuleOf(importPath string) (string, bool) {
	rest, ok := strings.CutPrefix(importPath, cqrsModulePrefix)
	if !ok {
		return "", false
	}

	return stripVersionSuffix(rest), true
}

// stripVersionSuffix removes every major-version segment ("/v2".."/v9")
// from a module path. Both module-root imports ("stack/sqlite/v4") and
// subpackages of versioned modules ("storage/v4/relational") must normalize
// to the table fragment ("stack/sqlite", "storage/relational"): the version
// segment sits mid-path whenever a module carries more than one package.
func stripVersionSuffix(path string) string {
	segs := strings.Split(path, "/")
	kept := segs[:0]
	for _, seg := range segs {
		if len(seg) == 2 && seg[0] == 'v' && seg[1] >= '2' && seg[1] <= '9' {
			continue
		}

		kept = append(kept, seg)
	}

	return strings.Join(kept, "/")
}

// defaultQualifier returns the package qualifier an unaliased import binds:
// the last path segment, with a major-version suffix ("/v2".."/v9") stripped.
func defaultQualifier(path string) string {
	return lastPathSegment(stripVersionSuffix(path))
}

// lastPathSegment returns the final slash-separated segment of an import
// path (the default package qualifier).
func lastPathSegment(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}

	return path
}
