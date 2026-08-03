// Package lintutil holds helpers shared across the cqrs-lint rule
// subpackages (correctness, consistency, boilerplate, etc.).
package lintutil

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// ToolName is the canonical linter name used in every finding. Centralized
// here so the value is defined once instead of copy-pasted across 8 rule
// packages. Each package aliases it via `const toolName = lintutil.ToolName`.
const ToolName finding.ToolName = "cqrs-lint"

// AppendBuild appends a built [finding.Finding] to findings unless err is
// non-nil. Building a finding only fails when the builder was mis-configured
// at construction time (a programming bug in the rule itself), so the error
// is silently dropped — the rule's own unit tests catch builder bugs, and
// surfacing them in the lint output would pollute results for users.
//
// Collapses the repeated
//
//	f, err := finding.NewBuilder(...).Build()
//	if err != nil {
//	    return
//	}
//	*findings = append(*findings, f)
//
// boilerplate found in every rule's report* helper.
func AppendBuild(findings *[]finding.Finding, f finding.Finding, err error) {
	if err != nil {
		return
	}

	*findings = append(*findings, f)
}

// nonCQRSRegisterPackages lists package qualifiers whose Register/Handle
// method name collides with CQRS but serves a different purpose. Rules that
// look for CQRS handler registration must skip these to avoid false positives.
//
//nolint:gochecknoglobals // read-only denylist
var nonCQRSRegisterPackages = map[string]bool{
	"huma":  true, // Huma v2 HTTP framework: huma.Register[I,O,Body]
	"http":  true, // net/http
	"mux":   true, // gorilla/mux
	"chi":   true, // go-chi/chi
	"gin":   true, // gin-gonic/gin
	"echo":  true, // labstack/echo
	"fiber": true, // gofiber/fiber
	"grpc":  true, // grpc-go Server.Register (proto service registration)
}

// IsNonCQRSRegisterPackage reports whether pkgName is a third-party package
// qualifier whose Register/Handle method is unrelated to CQRS dispatching.
// Use to suppress false positives in rules that detect handler registration.
func IsNonCQRSRegisterPackage(pkgName string) bool {
	return nonCQRSRegisterPackages[pkgName]
}

// CollectPkgLevelVarCalls returns the set of CallExpr positions that are
// the initializer of a package-level var declaration (the sentinel-error
// pattern: var ErrXxx = errors.New(...)). Computed once per lint run, over
// non-test files only. Shared by D006 (consistency) and C025 (correctness).
func CollectPkgLevelVarCalls(ctx *analyzer.AnalysisContext) map[token.Pos]bool {
	positions := make(map[token.Pos]bool)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				continue
			}

			for _, spec := range genDecl.Specs {
				valSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}

				for _, val := range valSpec.Values {
					if c, ok := val.(*ast.CallExpr); ok {
						positions[c.Pos()] = true
					}
				}
			}
		}
	}

	return positions
}

// IsFmtErrorf returns true if call is fmt.Errorf(...).
func IsFmtErrorf(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	return SelectorMatches(sel, "fmt", "Errorf")
}

// SelectorMatches reports whether sel is pkgName.selName where selName matches
// any of selNames. Shared by rules that check call targets by package and
// method name (d012 isContextType, c032 isContextCreation, IsFmtErrorf).
func SelectorMatches(sel *ast.SelectorExpr, pkgName string, selNames ...string) bool {
	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != pkgName {
		return false
	}

	return slices.Contains(selNames, sel.Sel.Name)
}

// ExprCallSelector extracts the selector from an expression that is a function
// call: expr → *ast.CallExpr → analyzer.SelectorFromExpr(call.Fun). Returns
// (nil, false) when expr is not a call or its target is not a selector.
// Shared by rules that inspect call expressions for method patterns
// (a002 isDirectJSONMarshal, swallow_helpers isPayloadCall).
func ExprCallSelector(expr ast.Expr) (*ast.SelectorExpr, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}

	return analyzer.SelectorFromExpr(call.Fun)
}

// CallSelectorMatches reports whether call's function target is a selector
// whose method name matches any of names. Convenience for the common
// analyzer.SelectorFromExpr(call.Fun) → check Sel.Name pattern.
// Shared by c021 (isLockCall/isUnlockCall) and c024 (isTransactionCall).
func CallSelectorMatches(call *ast.CallExpr, names ...string) bool {
	sel, ok := analyzer.SelectorFromExpr(call.Fun)
	if !ok {
		return false
	}

	return slices.Contains(names, sel.Sel.Name)
}

// ModuleImportsPath reports whether any non-test file in the analysis context
// imports a path containing the given substring. Checks both the packages
// loader (ctx.Packages) and raw AST import declarations (ctx.GoFiles).
// Shared by s005 (moduleHasSigning) and s006 (moduleHasEncryption).
func ModuleImportsPath(ctx *analyzer.AnalysisContext, path string) bool {
	for _, pkg := range ctx.Packages {
		for _, imp := range pkg.Imports {
			if imp == nil {
				continue
			}

			if strings.Contains(imp.PkgPath, path) {
				return true
			}
		}
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, imp := range gf.AST.Imports {
			if imp.Path == nil {
				continue
			}

			p := strings.Trim(imp.Path.Value, `"`)

			if strings.Contains(p, path) {
				return true
			}
		}
	}

	return false
}

// FirstFilePos returns the package declaration position of the first non-test
// file. Used as an anchor for project-level findings without a specific call
// site. Shared by adoption and architecture rule packages.
func FirstFilePos(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		if gf.AST.Package != token.NoPos {
			return ctx.Fset.Position(gf.AST.Package), true
		}
	}

	return token.Position{}, false
}

// HasWrapVerb returns true if the format string contains %w (error wrapping).
// Non-literal format strings return true (can't analyze statically, assume wrapping).
func HasWrapVerb(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok {
		return true // Non-literal format string — can't analyze statically.
	}

	return strings.Contains(lit.Value, "%w")
}

// StringLit extracts a string literal value from an AST expression.
// Returns "" when the expression is not a string literal.
func StringLit(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}

	return strings.Trim(lit.Value, `"`)
}

// FileImportsCQRS returns true if the file's import declarations include
// any go-cqrs-lite module path. Shared by D006 (consistency) and C025
// (correctness) to gate CQRS-specific error-handling checks.
func FileImportsCQRS(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)
		if analyzer.IsCQRSModulePath(path) {
			return true
		}
	}

	return false
}

// LooksLikeEventPayload checks if a struct name or file path suggests the
// struct is an event payload. Shared by C013 (correctness) and the F-series
// adoption rules to avoid duplicating the heuristic across packages.
func LooksLikeEventPayload(structName, filePath string) bool {
	upper := strings.ToUpper(structName)

	for _, suffix := range []string{"EVENT", "PAYLOAD", "EVENTDATA"} {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}

	base := filePath
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}

	base = strings.TrimSuffix(base, ".go")

	return base == "events" || base == "payloads"
}

// IsEventPayloadName reports whether the struct name follows the CQRS event
// payload naming convention (created/updated/deleted/event suffixes). Shared
// by D014 and D015 (consistency) which gate payload-specific checks on it.
func IsEventPayloadName(name string) bool {
	lower := strings.ToLower(name)

	for _, suffix := range []string{"created", "updated", "deleted", "event"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}

	return false
}

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

		if imp.Name != nil && imp.Name.Name == "." {
			return path, true
		}

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

// ImportQualifierMap builds a complete qualifier to import-path map for a file.
func ImportQualifierMap(file *ast.File) map[string]string {
	result := make(map[string]string)

	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)

		if imp.Name != nil {
			switch imp.Name.Name {
			case "_":
				continue
			case ".":
				result["."] = path
				continue
			default:
				result[imp.Name.Name] = path
				continue
			}
		}

		result[lastSegment(path)] = path
	}

	return result
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

// lastSegment returns the likely package name from an import path.
// Go convention: the package name matches the last path segment, EXCEPT for
// major-version suffixes (/v2, /v3, ...) which are stripped. For example,
// "github.com/foo/event/v4" → "event", not "v4".
func lastSegment(importPath string) string {
	if idx := strings.LastIndex(importPath, "/"); idx >= 0 {
		seg := importPath[idx+1:]
		// Strip major-version suffix (v2, v3, etc.) — the package name is
		// the segment before it.
		if len(seg) == 2 && seg[0] == 'v' && seg[1] >= '2' && seg[1] <= '9' {
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
