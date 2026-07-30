package adoption

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// importsPath reports whether any non-test file imports a path containing
// suffix. Works with AST-level import declarations so it is testable via
// analyzer.BuildContextFromSource.
func importsPath(ctx *analyzer.AnalysisContext, suffix string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}

			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, suffix) {
				return true
			}
		}
	}

	return false
}

// projectHasCall reports whether any non-test file calls pkgName.funcName.
func projectHasCall(ctx *analyzer.AnalysisContext, pkgName, funcName string) bool {
	return projectHasCallAny(ctx, pkgName, funcName)
}

// projectHasCallAny reports whether any non-test file calls any of funcNames
// on pkgName.
func projectHasCallAny(ctx *analyzer.AnalysisContext, pkgName string, funcNames ...string) bool {
	nameSet := make(map[string]bool, len(funcNames))
	for _, n := range funcNames {
		nameSet[n] = true
	}

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

			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != pkgName {
				return true
			}

			if nameSet[sel.Sel.Name] {
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

// projectHasSelector reports whether any non-test file references pkgName.selName
// in any selector expression (covers type usage, composite literals, calls).
func projectHasSelector(ctx *analyzer.AnalysisContext, pkgName, selName string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			if pkg.Name == pkgName && sel.Sel.Name == selName {
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

// firstCallPos returns the position of the first call to pkgName.funcName
// in any non-test file.
func firstCallPos(
	ctx *analyzer.AnalysisContext,
	pkgName, funcName string,
) (token.Position, bool) {
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

			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != pkgName {
				return true
			}

			if sel.Sel.Name == funcName {
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

// firstFilePos returns the package declaration position of the first non-test
// file. Used as an anchor for project-level findings without a specific call site.
func firstFilePos(ctx *analyzer.AnalysisContext) (token.Position, bool) {
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

// firstFuncDeclPos returns the position of the first function or method
// declaration whose name starts with prefix.
func firstFuncDeclPos(
	ctx *analyzer.AnalysisContext,
	prefix string,
) (token.Position, bool) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if strings.HasPrefix(fn.Name.Name, prefix) {
				return ctx.Fset.Position(fn.Pos()), true
			}
		}
	}

	return token.Position{}, false
}

// eventCount returns the number of distinct event types emitted by the project.
func eventCount(ctx *analyzer.AnalysisContext) int {
	return len(ctx.Registry.EventTypesEmitted)
}

// distinctAggregateCount returns the number of distinct aggregate types
// inferred from event type prefixes (the segment before the first dot).
// Event types without dots each count as a separate aggregate.
func distinctAggregateCount(ctx *analyzer.AnalysisContext) int {
	aggregates := make(map[string]bool)

	for eventType := range ctx.Registry.EventTypesEmitted {
		prefix := eventType
		if idx := strings.Index(eventType, "."); idx > 0 {
			prefix = eventType[:idx]
		}

		aggregates[prefix] = true
	}

	return len(aggregates)
}

// hasPIIInEventPayloads scans event payload structs for PII-like field names.
// Returns the position of the first PII field found.
func hasPIIInEventPayloads(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	piiFields := []string{
		"email", "phone", "ssn", "password", "address",
		"creditcard", "credit_card", "passport", "iban",
		"national_id", "dob", "birthdate",
	}

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

				st, ok := ts.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					continue
				}

				if !looksLikeEventPayload(ts.Name.Name, gf.Path) {
					continue
				}

				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						lower := strings.ToLower(name.Name)
						for _, pii := range piiFields {
							if strings.Contains(lower, pii) {
								return ctx.Fset.Position(field.Pos()), true
							}
						}
					}
				}
			}
		}
	}

	return token.Position{}, false
}

// looksLikeEventPayload checks if a struct name or file path suggests the
// struct is an event payload.
func looksLikeEventPayload(structName, filePath string) bool {
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

// hasTimeBasedPatterns detects signals that the domain has time-based business
// rules (deadlines, expirations, timeouts) by scanning for time.AfterFunc,
// time.NewTimer, or function names containing deadline/expire/timeout/schedule.
func hasTimeBasedPatterns(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	if pos, ok := firstCallPos(ctx, "time", "AfterFunc"); ok {
		return pos, true
	}

	if pos, ok := firstCallPos(ctx, "time", "NewTimer"); ok {
		return pos, true
	}

	for _, prefix := range []string{"Expire", "Timeout", "Deadline", "Schedule", "Cancel"} {
		if pos, ok := firstFuncDeclPos(ctx, prefix); ok {
			return pos, true
		}
	}

	return token.Position{}, false
}

// hasTraversalPatterns detects signals that the domain needs graph-like
// traversal (recursive queries, ancestry, path-finding).
func hasTraversalPatterns(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	keywords := []string{
		"Traverse", "Ancestor", "Descendant", "ShortestPath",
		"Path", "Neighbor", "Adjacency", "Hierarchy",
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			for _, kw := range keywords {
				if strings.Contains(fn.Name.Name, kw) {
					return ctx.Fset.Position(fn.Pos()), true
				}
			}
		}
	}

	// Also check for SQL recursive CTE patterns in string literals.
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}

			val := strings.ToUpper(strings.Trim(lit.Value, `"`))
			if strings.Contains(val, "WITH RECURSIVE") {
				found = true
				return false
			}

			return true
		})

		if found {
			return ctx.Fset.Position(gf.AST.Pos()), true
		}
	}

	return token.Position{}, false
}

// astInspectCalls walks the AST and calls fn for every *ast.CallExpr.
// Stops early if fn returns false.
func astInspectCalls(root ast.Node, fn func(*ast.CallExpr) bool) {
	ast.Inspect(root, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		return fn(call)
	})
}

// firstCallByName returns the position of the first call to a function named
// funcName, regardless of package qualifier. Useful for detecting calls like
// NewDispatcher where the package qualifier varies.
func firstCallByName(
	ctx *analyzer.AnalysisContext,
	funcName string,
) (token.Position, bool) {
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

			if sel.Sel.Name == funcName {
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

// singleInfoFinding builds and returns a single info-level finding with the
// common F-series defaults: CategoryBestPractice, FixStrategySuggest.
func singleInfoFinding(
	ctx *analyzer.AnalysisContext,
	ruleID, message, suggestion string,
	pos token.Position,
	confidence finding.Confidence,
) []finding.Finding {
	f, err := finding.NewBuilder(
		finding.RuleName(ruleID), toolName,
		message,
		finding.SeverityInfo,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryBestPractice).
		WithConfidence(confidence).
		WithFixStrategy(finding.FixStrategySuggest).
		WithSuggestion(suggestion).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	if err != nil {
		return nil
	}

	return []finding.Finding{f}
}
