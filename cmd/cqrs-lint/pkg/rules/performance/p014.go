package performance

import (
	"context"
	"go/ast"
	"go/types"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// P014: ApplyLayout called on an engine that also implements the plan path.
//
// Metaengine engines expose two layout write paths: the legacy
// ApplyLayout(collection, filterFields, sortFields) and the plan path
// (BuildLayoutPlanFromType[R] -> plan -> ApplyLayoutPlan). Calling
// ApplyLayout directly on an engine whose method set also carries
// ApplyLayoutPlan bypasses type-derived planning: no pushdown or aggregate
// cost inference from the read-model type.
//
// Detection is structural (T23 design addendum, 2026-09-07): no metaengine
// import is needed (dep budget, layer rules). A receiver "supports the plan
// path" when its method set contains ApplyLayoutPlan (case-insensitive) —
// the single method of metaengine.LayoutPlanApplier. The addendum's original
// pair (ApplyLayoutPlan + BuildLayoutPlan) cannot work as written:
// BuildLayoutPlan is a PACKAGE-level function in metaengine, never an engine
// method, so the corrected co-occurrence is the ApplyLayout call itself plus
// ApplyLayoutPlan on the same type. Renaming the interface method makes the
// rule go silent — it degrades safely and never false-fires on unrelated
// types because both names must co-occur on one receiver.
//
// Typed-path only: receiver-type attribution requires TypesInfo, so the rule
// never fires on syntax-only loads (the name-only fallback stays silent
// rather than guessing).
//
//nolint:ireturn // factory returns public interface
func NewP014Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P014-applylayout-bypasses-plan-path",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			if !ctx.TypedConfirmations() {
				return findings, nil
			}

			for _, gf := range ctx.GoFiles {
				if gf.Pkg == nil || gf.Pkg.TypesInfo == nil {
					continue
				}

				for _, call := range applyLayoutCalls(gf) {
					typeName, ok := planPathReceiver(gf, call)
					if !ok {
						continue
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"P014", toolName,
						"ApplyLayout on "+typeName+
							" bypasses the LayoutPlan path — derive a plan from the read-model "+
							"type instead (see recipes §2.27)",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryPerformance).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategyNone).
						WithSuggestion(
							"Build a plan with metaengine.BuildLayoutPlanFromType[R] and apply it " +
								"via ApplyLayoutPlan so filter/sort/aggregate shapes are inferred " +
								"from the read-model type",
						).
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					lintutil.AppendBuild(&findings, f, err)
				}
			}

			return findings, nil
		},
	)
}

// applyLayoutCalls returns every selector call expression named ApplyLayout
// in the file.
func applyLayoutCalls(gf *analyzer.GoFile) []*ast.CallExpr {
	var calls []*ast.CallExpr

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "ApplyLayout" {
			return true
		}

		calls = append(calls, call)

		return true
	})

	return calls
}

// planPathReceiver resolves the call's receiver type through type info and
// reports whether it carries the LayoutPlanApplier method shape
// (ApplyLayoutPlan, case-insensitive). The returned name is the receiver's
// named type for the finding message. Interfaces are skipped: a consumer
// holding a metaengine.LayoutPlanner cannot reach the plan path without the
// concrete type anyway.
func planPathReceiver(gf *analyzer.GoFile, call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	tv, ok := gf.Pkg.TypesInfo.Types[sel.X]
	if !ok || tv.Type == nil {
		return "", false
	}

	named := namedTypeOf(tv.Type)
	if named == nil {
		return "", false
	}

	for method := range types.NewMethodSet(types.NewPointer(named)).Methods() {
		if strings.EqualFold(method.Obj().Name(), "ApplyLayoutPlan") {
			return named.Obj().Name(), true
		}
	}

	return "", false
}

// namedTypeOf strips pointers down to the named type, or returns nil for
// interfaces, type parameters, and anonymous types.
func namedTypeOf(t types.Type) *types.Named {
	for t != nil {
		switch x := t.(type) {
		case *types.Pointer:
			t = x.Elem()
		case *types.Named:
			return x
		default:
			return nil
		}
	}

	return nil
}
