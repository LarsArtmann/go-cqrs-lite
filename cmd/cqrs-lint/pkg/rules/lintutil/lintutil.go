// Package lintutil holds helpers shared across the cqrs-lint rule
// subpackages (correctness, consistency, boilerplate, etc.).
package lintutil

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

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

	ident, ok := sel.X.(*ast.Ident)

	return ok && ident.Name == "fmt" && sel.Sel.Name == "Errorf"
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
