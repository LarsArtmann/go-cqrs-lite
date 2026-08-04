package architecture

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// importsPathSuffix reports whether any non-test file imports a path containing
// suffix. Works with AST-level import declarations so it is testable via
// analyzer.BuildContextFromSource.
func importsPathSuffix(ctx *analyzer.AnalysisContext, suffix string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		if fileImportsPath(gf, suffix) {
			return true
		}
	}

	return false
}

// fileImportsPath reports whether a single GoFile imports a path containing
// suffix. Used by per-module detectors (e.g. E009) that need to evaluate
// imports per-module rather than workspace-wide.
func fileImportsPath(gf *analyzer.GoFile, suffix string) bool {
	for _, imp := range gf.AST.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, suffix) {
			return true
		}
	}

	return false
}

// findKeyBoolLit reports whether any non-test file contains a composite literal
// key-value pair where the key is keyName and the value is a boolean literal
// matching wantBool. Used to detect config flags like Enabled: false or
// BlockPublishUntilSubscriberAck: false.
func findKeyBoolLit(ctx *analyzer.AnalysisContext, keyName string, wantBool bool) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				ident, ok := kv.Key.(*ast.Ident)
				if !ok || ident.Name != keyName {
					continue
				}

				bid, ok := kv.Value.(*ast.Ident)
				if !ok {
					continue
				}

				if (wantBool && bid.Name == "true") || (!wantBool && bid.Name == "false") {
					found = true
					return false
				}
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// countTypesWithSuffix counts non-test type declarations whose name ends with
// suffix. Used to detect adapter layers (E011) and dual-write buses (E012).
func countTypesWithSuffix(ctx *analyzer.AnalysisContext, suffix string) int {
	count := 0

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if strings.HasSuffix(ts.Name.Name, suffix) {
					count++
				}
			}
		}
	}

	return count
}

// typeExists reports whether any non-test file declares a type with the given
// name. Used to detect DualWrite types (E012).
func typeExists(ctx *analyzer.AnalysisContext, nameSubstring string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if strings.Contains(ts.Name.Name, nameSubstring) {
					return true
				}
			}
		}
	}

	return false
}

// firstFilePos returns the package declaration position of the first non-test
// file. Used as an anchor for project-level findings without a specific call site.
func firstFilePos(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	return lintutil.FirstFilePos(ctx)
}

// singleFinding builds and returns a single finding with the common architecture
// defaults. Category is CategoryStructure, matching all existing E-series rules.
func singleFinding(
	ctx *analyzer.AnalysisContext,
	ruleID, message, suggestion string,
	pos token.Position,
	severity finding.Severity,
	confidence finding.Confidence,
) []finding.Finding {
	f, err := finding.NewBuilder(
		finding.RuleName(ruleID), toolName,
		message,
		severity,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryStructure).
		WithConfidence(confidence).
		WithSuggestion(suggestion).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	if err != nil {
		return nil
	}

	return []finding.Finding{f}
}

// findKeyBoolLitInTypedComposite reports whether any non-test file contains a
// composite literal key-value pair where the key is keyName, the value is a
// boolean literal matching wantBool, AND the composite literal's type name
// contains one of typeSubstrings. This prevents false positives where an
// unrelated struct happens to have the same key name (e.g., Enabled: false
// in a feature-flag config vs a signing/encryption config).
func findKeyBoolLitInTypedComposite(
	ctx *analyzer.AnalysisContext,
	keyName string,
	wantBool bool,
	typeSubstrings ...string,
) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			typeName := compositeTypeName(cl.Type)
			if typeName == "" {
				return true
			}

			lowerType := strings.ToLower(typeName)
			matchesType := false
			for _, sub := range typeSubstrings {
				if strings.Contains(lowerType, strings.ToLower(sub)) {
					matchesType = true
					break
				}
			}

			if !matchesType {
				return true
			}

			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				ident, ok := kv.Key.(*ast.Ident)
				if !ok || ident.Name != keyName {
					continue
				}

				bid, ok := kv.Value.(*ast.Ident)
				if !ok {
					continue
				}

				if (wantBool && bid.Name == "true") || (!wantBool && bid.Name == "false") {
					found = true
					return false
				}
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// compositeTypeName extracts the type name from a composite literal type
// expression (e.g., "signing.Config" from *ast.CompositeLit.Type).
func compositeTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			return pkg.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	case *ast.StarExpr:
		return compositeTypeName(t.X)
	case *ast.ArrayType:
		return compositeTypeName(t.Elt)
	}
	return ""
}

// firstKeyBoolPosInTypedComposite returns the position of the first composite
// literal key-value pair matching keyName/wantBool within a typed composite
// literal whose type contains one of typeSubstrings.
func firstKeyBoolPosInTypedComposite(
	ctx *analyzer.AnalysisContext,
	keyName string,
	wantBool bool,
	typeSubstrings ...string,
) (token.Position, bool) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var hit ast.Node

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			typeName := compositeTypeName(cl.Type)
			if typeName == "" {
				return true
			}

			lowerType := strings.ToLower(typeName)
			matchesType := false
			for _, sub := range typeSubstrings {
				if strings.Contains(lowerType, strings.ToLower(sub)) {
					matchesType = true
					break
				}
			}

			if !matchesType {
				return true
			}

			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				ident, ok := kv.Key.(*ast.Ident)
				if !ok || ident.Name != keyName {
					continue
				}

				bid, ok := kv.Value.(*ast.Ident)
				if !ok {
					continue
				}

				if (wantBool && bid.Name == "true") || (!wantBool && bid.Name == "false") {
					hit = kv
					return false
				}
			}

			return true
		})

		if hit != nil {
			return ctx.Fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}

// projectCallsMethodOnType scans for calls to any of methodNames on any
// receiver whose type path contains any of typePathSubstrs. Uses type info
// when available (real analysis); falls back to variable-name matching when
// type info is absent (unit tests with AST-only context).
//
// Example: projectCallsMethodOnType(ctx, []string{"Save", "Append"},
// []string{"go-cqrs-lite/event", "go-cqrs-lite/storage"}) matches
// `myStore.Save(...)` when myStore's type path contains "go-cqrs-lite/event"
// or "go-cqrs-lite/storage".
func projectCallsMethodOnType(
	ctx *analyzer.AnalysisContext,
	methodNames []string,
	typePathSubstrs []string,
) (token.Position, bool) {
	nameSet := make(map[string]bool, len(methodNames))
	for _, n := range methodNames {
		nameSet[n] = true
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var hit *ast.CallExpr

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			if !nameSet[sel.Sel.Name] {
				return true
			}

			if receiverTypeMatches(gf, sel.X, typePathSubstrs) {
				hit = call
				return false
			}

			return true
		})

		if hit != nil {
			return ctx.Fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}

// receiverTypeMatches checks whether the expression's type path contains any
// of the substrings. Uses go/types when available, falls back to variable-name
// heuristic otherwise.
func receiverTypeMatches(
	gf *analyzer.GoFile,
	expr ast.Expr,
	typePathSubstrs []string,
) bool {
	// Try type info first (real analysis path).
	if gf.Pkg != nil && gf.Pkg.TypesInfo != nil {
		if tv, ok := gf.Pkg.TypesInfo.Types[expr]; ok {
			typeStr := tv.Type.String()
			for _, sub := range typePathSubstrs {
				if strings.Contains(typeStr, sub) {
					return true
				}
			}
			// Type info available but type doesn't match — trust the type info.
			return false
		}
	}

	// Fallback: no type info (unit tests). Use variable-name heuristic.
	if ident, ok := expr.(*ast.Ident); ok {
		name := strings.ToLower(ident.Name)
		for _, sub := range typePathSubstrs {
			switch {
			case strings.Contains(sub, "event") || strings.Contains(sub, "storage"):
				if slices.Contains([]string{"store", "eventstore", "repo", "es"}, name) {
					return true
				}
			case strings.Contains(sub, "projectionhost"):
				if slices.Contains([]string{"host", "projectionhost", "proj"}, name) {
					return true
				}
			case strings.Contains(sub, "signing") || strings.Contains(sub, "encryption"):
				if slices.Contains([]string{"signer", "encryptor", "enc", "sign"}, name) {
					return true
				}
			}
		}
	}

	return false
}

// projectHasMethodCallContaining reports whether any non-test file has a call
// to a method whose name contains substr. This is a broad check used for
// suppression (e.g., any .Execute() call suppresses E010).
func projectHasMethodCallContaining(
	ctx *analyzer.AnalysisContext,
	substr string,
) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			if strings.Contains(sel.Sel.Name, substr) {
				found = true
				return false
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// projectCallsImportPathBool is a boolean-only wrapper around
// projectCallsImportPath for use in compound boolean expressions.
func projectCallsImportPathBool(
	ctx *analyzer.AnalysisContext,
	importPathSubstr string,
	funcNames ...string,
) bool {
	_, ok := projectCallsImportPath(ctx, importPathSubstr, funcNames...)
	return ok
}

// projectCallsImportPath checks whether any non-test file calls any of funcNames
// on a receiver whose import path contains importPathSubstr. This resolves
// import aliases: `import es "go-cqrs-lite/event"` then `es.NewEvent(...)` is
// correctly matched even though the qualifier is "es", not "event".
func projectCallsImportPath(
	ctx *analyzer.AnalysisContext,
	importPathSubstr string,
	funcNames ...string,
) (token.Position, bool) {
	nameSet := make(map[string]bool, len(funcNames))
	for _, n := range funcNames {
		nameSet[n] = true
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var hit *ast.CallExpr

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			if !nameSet[sel.Sel.Name] {
				return true
			}

			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			impPath, found := lintutil.QualifierToImportPath(gf.AST, pkgIdent.Name)
			if !found {
				return true
			}

			if strings.Contains(impPath, importPathSubstr) {
				hit = call
				return false
			}

			return true
		})

		if hit != nil {
			return ctx.Fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}
