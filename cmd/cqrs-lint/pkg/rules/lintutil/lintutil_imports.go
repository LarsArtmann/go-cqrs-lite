package lintutil

import (
	"go/ast"
	"strconv"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// QualifierToImportPath resolves a package qualifier (the identifier before
// the dot in a selector expression like "event.NewEvent") to its full import
// path using the file's import declarations. This handles import aliases:
//
//	import event "github.com/larsartmann/go-cqrs-lite/event/v4"
//	// qualifier "event" → "github.com/larsartmann/go-cqrs-lite/event/v4"
//
//	import cqrs "github.com/larsartmann/go-cqrs-lite/event/v4"
//	// qualifier "cqrs" → "github.com/larsartmann/go-cqrs-lite/event/v4"
//
// Rules that previously hardcoded the expected package name (e.g., matching
// selector X == "event") should use this helper to resolve the actual import
// path, making them resilient to import aliases. Returns "" and false if the
// qualifier does not match any import in the file.
func QualifierToImportPath(file *ast.File, qualifier string) (string, bool) {
	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)

		if imp.Name != nil && imp.Name.Name == "_" {
			continue
		}

		// Note: dot imports (".") bind no qualifier, so they never match one
		// here — returning their path for an arbitrary qualifier would
		// false-attribute another package's symbols.

		if imp.Name != nil {
			if imp.Name.Name == qualifier {
				return path, true
			}

			continue
		}

		if lastSegment(path) == qualifier {
			return path, true
		}
	}

	return "", false
}

// QualifierResolvesTo checks whether a qualifier in the given file resolves to
// an import path that contains the expected suffix.
func QualifierResolvesTo(file *ast.File, qualifier, expectedPathSuffix string) bool {
	path, ok := QualifierToImportPath(file, qualifier)
	if !ok {
		return false
	}

	return strings.Contains(path, expectedPathSuffix)
}

// QualifierTargetsModule reports whether a selector qualifier refers to the
// module whose import path contains pathFragment — the alias-safe successor
// of the bare `pkgIdent.Name != "event"` comparisons (the A014 bug class).
// Resolution order: the type checker when available (exact, shadow-proof),
// then the file's import table (alias-aware), then the segment-name fallback
// so partial results survive broken or import-less loads.
func QualifierTargetsModule(gf *analyzer.GoFile, ident *ast.Ident, pathFragment string) bool {
	if gf == nil || ident == nil {
		return false
	}

	if path, resolved := analyzer.ResolveQualifierTyped(gf, ident); resolved {
		return strings.Contains(path, pathFragment)
	}

	if gf.AST != nil {
		if path, ok := QualifierToImportPath(gf.AST, ident.Name); ok {
			return strings.Contains(path, pathFragment)
		}
	}

	segment := pathFragment
	if i := strings.LastIndex(pathFragment, "/"); i >= 0 {
		segment = pathFragment[i+1:]
	}

	return ident.Name == segment
}

// lastSegment returns the likely package name from an import path.
// Go convention: the package name matches the last path segment, EXCEPT for
// major-version suffixes (/v2, /v3, ...) which are stripped. For example,
// "github.com/foo/event/v4" → "event", not "v4".
// isMajorVersionSegment reports whether seg is a Go major-version suffix
// ("v2" through "v99"). Only single-digit majors were handled before, so a
// v10+ import yielded the bogus package name "v10".
func isMajorVersionSegment(seg string) bool {
	if len(seg) < 2 || seg[0] != 'v' {
		return false
	}

	for _, r := range seg[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}

	n, err := strconv.Atoi(seg[1:])

	return err == nil && n >= 2
}

func lastSegment(importPath string) string {
	if idx := strings.LastIndex(importPath, "/"); idx >= 0 {
		seg := importPath[idx+1:]
		// Strip major-version suffix (v2, v3, ... v99) — the package name is
		// the segment before it.
		if isMajorVersionSegment(seg) {
			rest := importPath[:idx]
			if idx2 := strings.LastIndex(rest, "/"); idx2 >= 0 {
				return rest[idx2+1:]
			}
			return rest
		}

		return seg
	}

	return importPath
}

// FileImportsSubstr reports whether the AST file imports a path containing
// substr. Shared by the architecture and testrules detectors.
func FileImportsSubstr(file *ast.File, substr string) bool {
	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, substr) {
			return true
		}
	}

	return false
}
