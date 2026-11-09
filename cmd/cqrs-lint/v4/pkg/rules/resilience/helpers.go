package resilience

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/larsartmann/go-finding"
	"golang.org/x/tools/go/packages"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// singleInfoFinding builds a single info-level finding with the common
// resilience defaults: CategoryBestPractice, FixStrategySuggest.
func singleInfoFinding(
	ctx *analyzer.AnalysisContext,
	ruleID, message, suggestion string,
	pos token.Position,
	confidence finding.Confidence,
) []finding.Finding {
	f, err := findingTemplate.Builder(
		finding.RuleName(ruleID),
		message,
		finding.SeverityInfo,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
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

// isBusName reports whether name looks like a bus or dispatcher variable.
func isBusName(name string) bool {
	name = strings.ToLower(name)

	return strings.HasSuffix(name, "bus") ||
		strings.HasSuffix(name, "dispatcher") ||
		strings.HasSuffix(name, "disp")
}

// receiverIsCQRSBus checks whether a method call's receiver is a go-cqrs-lite
// bus type (event, command, query, or dispatcher). Returns true when type info
// is unavailable (conservative — assumes yes to avoid false negatives).
func receiverIsCQRSBus(pkg *packages.Package, sel *ast.SelectorExpr) bool {
	if pkg == nil || pkg.TypesInfo == nil {
		return true
	}

	tv, ok := pkg.TypesInfo.Types[sel.X]
	if !ok || tv.Type == nil {
		return true
	}

	typeStr := tv.Type.String()

	return strings.Contains(typeStr, "cqrs-lite/event/") ||
		strings.Contains(typeStr, "cqrs-lite/command/") ||
		strings.Contains(typeStr, "cqrs-lite/query/") ||
		strings.Contains(typeStr, "cqrs-lite/dispatcher/")
}

// busVariable is one tracked bus/dispatcher variable: its name, its
// declaration position, and — when type info resolves — the variable's
// types.Object identity. Identity is what separates two same-named `bus`
// variables in different scopes or files: name-keyed tracking blended
// them into one entry (a journal-tail accessor in one file masked a
// dispatch bus in another; CV feedback, 2026-10-05). When obj is nil
// (syntax-only loads), identity degrades to the legacy name match.
type busVariable struct {
	name string
	pos  token.Position
	obj  types.Object
}

// selectorOnVariable reports whether sel's receiver identifier refers to
// variable v: exact types.Object identity when both sides resolve, else
// the conservative name match (the legacy semantics).
func selectorOnVariable(pkg *packages.Package, sel *ast.SelectorExpr, v busVariable) bool {
	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != v.name {
		return false
	}

	if v.obj == nil || pkg == nil || pkg.TypesInfo == nil {
		return true
	}

	if obj := pkg.TypesInfo.ObjectOf(ident); obj != nil {
		return obj == v.obj
	}

	return true
}

// identIsVariable is selectorOnVariable's assignment-LHS twin.
func identIsVariable(pkg *packages.Package, ident *ast.Ident, v busVariable) bool {
	if ident.Name != v.name {
		return false
	}

	if v.obj == nil || pkg == nil || pkg.TypesInfo == nil {
		return true
	}

	if obj := pkg.TypesInfo.ObjectOf(ident); obj != nil {
		return obj == v.obj
	}

	return true
}

func hasBusMethodCall(ctx *analyzer.AnalysisContext, v busVariable) bool {
	busMethodNames := map[string]bool{
		"Use":           true,
		"UsePublish":    true,
		"Publish":       true,
		"Subscribe":     true,
		"SubscribeAll":  true,
		"Handle":        true,
		"Dispatch":      true,
		"RegisterTyped": true,
		"RegisterQuery": true,
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

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if !selectorOnVariable(gf.Pkg, sel, v) {
				return true
			}

			if busMethodNames[sel.Sel.Name] {
				if receiverIsCQRSBus(gf.Pkg, sel) {
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

// subscribeOnlyBusMethods are the read-side subscription calls: a project
// variable whose ENTIRE bus-method surface is these is consuming an
// in-process journal/notification tail, not dispatching through the bus.
//
//nolint:gochecknoglobals // read-only lookup table
var subscribeOnlyBusMethods = map[string]bool{
	"Subscribe":    true,
	"SubscribeAll": true,
}

// busIsReadOnlySubscriber reports whether every CQRS bus method the project
// calls on varName is a subscription (Subscribe/SubscribeAll). Such a
// variable is an in-process journal/notification TAIL the project only
// READS — the engine publishes internally after appends; no dispatch to
// downstream services flows through the variable — so retry/circuit-breaker
// middleware advice is category confusion for it (CV feedback, 2026-10-03).
// Any dispatch-side call (Publish, Dispatch, Handle, Use, Register*) makes
// it a dispatch pipeline and returns false.
func busIsReadOnlySubscriber(ctx *analyzer.AnalysisContext, v busVariable) bool {
	observed := 0

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		onlySubscribe := true

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if !selectorOnVariable(gf.Pkg, sel, v) {
				return true
			}

			if !receiverIsCQRSBus(gf.Pkg, sel) {
				return true
			}

			observed++

			if !subscribeOnlyBusMethods[sel.Sel.Name] {
				onlySubscribe = false
				return false
			}

			return true
		})

		if !onlySubscribe {
			return false
		}
	}

	return observed > 0
}
