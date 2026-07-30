package security

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// S005: Event signing available but disabled.
//
// Detects when the signing module is imported but signer construction and
// middleware wiring are guarded behind a boolean flag that defaults to false.
// The signing import signals "this project signs events" when in reality the
// flag is never set to true — a false sense of tamper protection.
//
// Detection requires three signals:
//  1. The signing package is imported somewhere in the project.
//  2. A signing constructor or middleware call exists inside an `if` block
//     whose condition is a positive reference to a bool field with an
//     enable-related name (SigningEnabled, EnableSign, VerifyEnabled, …).
//  3. That bool field defaults to false — it is a plain `bool` field with no
//     explicit `true` initialization found anywhere in the project.
//
// The rule is suppressed when signing middleware is also applied
// unconditionally (outside any enable-flag guard) — in that case signing is
// genuinely active and the guarded path is just a secondary code path.
//
//nolint:ireturn // factory returns public interface
func NewS005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S005-signing-available-but-disabled",
		func(_ context.Context) ([]finding.Finding, error) {
			if !moduleHasSigning(ctx) {
				return nil, nil
			}

			enableFields := collectEnableBoolFields(ctx)
			trueDefaults := collectExplicitTrueDefaults(ctx)

			var allGuards []signingGuardSite
			hasUnguardedSigning := false

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				fileGuards := findSigningGuardsInFile(ctx, gf.AST, enableFields)
				allGuards = append(allGuards, fileGuards...)

				if hasUnguardedSigningCall(gf.AST, fileGuards) {
					hasUnguardedSigning = true
				}
			}

			if hasUnguardedSigning {
				return nil, nil
			}

			if len(allGuards) == 0 {
				return nil, nil
			}

			var findings []finding.Finding

			for _, g := range allGuards {
				if trueDefaults[g.fieldName] {
					continue
				}

				f, err := finding.NewBuilder(
					"S005", toolName,
					fmt.Sprintf(
						"Signing imported but disabled — signer construction is guarded by %q "+
							"which defaults to false, so events are never actually signed",
						g.fieldName,
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(g.filename), g.line, g.column),
				).
					WithCategory(finding.CategorySecurity).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion(
						"Either set the signing flag to true in your config defaults, " +
							"or remove the signing import if signing is not intended",
					).
					WithSnippet(ctx.SourceLine(g.filename, g.line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// --- Types ---

type signingGuardSite struct {
	body      *ast.BlockStmt
	fieldName string
	filename  string
	line      int
	column    int
}

// --- Module-scope signing import check (mirrors moduleHasEncryption) ---

func moduleHasSigning(ctx *analyzer.AnalysisContext) bool {
	for _, pkg := range ctx.Packages {
		for _, imp := range pkg.Imports {
			if imp == nil {
				continue
			}
			if strings.Contains(imp.PkgPath, "go-cqrs-lite/signing") {
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
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, "go-cqrs-lite/signing") {
				return true
			}
		}
	}

	return false
}

// --- Enable-field collection ---

// enableNamePatterns matches struct field names that gate signing/verification.
var enableNamePatterns = []string{
	"enabled", "sign", "signing", "verify", "verification", "signature",
}

func collectEnableBoolFields(ctx *analyzer.AnalysisContext) map[string]bool {
	result := map[string]bool{}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}
		ast.Inspect(gf.AST, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				return true
			}
			for _, field := range st.Fields.List {
				if !isBoolType(field.Type) {
					continue
				}
				for _, name := range field.Names {
					lname := strings.ToLower(name.Name)
					for _, pat := range enableNamePatterns {
						if strings.Contains(lname, pat) {
							result[name.Name] = true
							break
						}
					}
				}
			}
			return true
		})
	}

	return result
}

func isBoolType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "bool"
}

// --- Explicit true defaults ---

func collectExplicitTrueDefaults(ctx *analyzer.AnalysisContext) map[string]bool {
	result := map[string]bool{}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}
		ast.Inspect(gf.AST, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				if val, ok := kv.Value.(*ast.Ident); ok && val.Name == "true" {
					result[key.Name] = true
				}
			}
			return true
		})
	}

	return result
}

// --- Guard detection ---

func enableFieldName(cond ast.Expr, enableFields map[string]bool) string {
	switch e := cond.(type) {
	case *ast.Ident:
		if enableFields[e.Name] {
			return e.Name
		}
	case *ast.SelectorExpr:
		if enableFields[e.Sel.Name] {
			return e.Sel.Name
		}
	}
	return ""
}

func findSigningGuardsInFile(
	ctx *analyzer.AnalysisContext,
	fileAST *ast.File,
	enableFields map[string]bool,
) []signingGuardSite {
	var guards []signingGuardSite

	ast.Inspect(fileAST, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		fieldName := enableFieldName(ifStmt.Cond, enableFields)
		if fieldName == "" {
			return true
		}

		if !blockContainsSigningCall(ifStmt.Body) {
			return true
		}

		pos := ctx.Fset.Position(ifStmt.Pos())
		guards = append(guards, signingGuardSite{
			body:      ifStmt.Body,
			fieldName: fieldName,
			filename:  pos.Filename,
			line:      pos.Line,
			column:    pos.Column,
		})

		return true
	})

	return guards
}

func hasUnguardedSigningCall(fileAST *ast.File, fileGuards []signingGuardSite) bool {
	found := false

	ast.Inspect(fileAST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isSigningConstruction(call) {
			return true
		}

		for _, g := range fileGuards {
			if g.body != nil &&
				call.Pos() >= g.body.Lbrace && call.End() <= g.body.Rbrace {
				return true
			}
		}

		found = true
		return false
	})

	return found
}

// --- Signing call detection ---

var signingConstructionNames = map[string]bool{ //nolint:gochecknoglobals // static lookup table
	"NewHMAC":                    true,
	"NewEd25519":                 true,
	"NewEd25519Verifier":         true,
	"GenerateEd25519KeyPair":     true,
	"NewCOSEHMAC":                true,
	"NewCOSEEd25519Signer":       true,
	"NewCOSEEd25519Verifier":     true,
	"SignMiddleware":             true,
	"VerifyMiddleware":           true,
	"RequireSignatureMiddleware": true,
}

func isSigningConstruction(call *ast.CallExpr) bool {
	sel, ok := analyzer.SelectorFromExpr(call.Fun)
	if !ok {
		return false
	}
	return signingConstructionNames[sel.Sel.Name]
}

func blockContainsSigningCall(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && isSigningConstruction(call) {
			found = true
			return false
		}
		return true
	})
	return found
}

