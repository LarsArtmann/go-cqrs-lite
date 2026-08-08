package consistency

import (
	"context"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// D018: Stale catalog entries.
// Detects catalog.NewBuilder calls that reference event type strings not
// present in any event.NewEvent call. These entries are potentially stale —
// the event type may have been renamed or removed.
//
//nolint:ireturn // factory returns public interface
func NewD018Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D018-stale-catalog-entries",
		func(_ context.Context) ([]finding.Finding, error) {
			eventTypes := collectEventNewTypes(ctx)
			if len(eventTypes) == 0 {
				return nil, nil
			}

			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					if !isCatalogBuilder(call) {
						return true
					}

					typeName := extractStringArg(call, 0)
					if typeName == "" {
						return true
					}

					if eventTypes[typeName] {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"D018", toolName,
						"Catalog entry for "+typeName+" has no matching event.NewEvent — "+
							"this event type may have been renamed or removed",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryDocumentation).
						WithConfidence(finding.ConfidenceLow).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Remove the stale catalog entry for " + typeName +
							" or update it to match the current event type").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err != nil {
						return true
					}

					findings = append(findings, f)

					return true
				})
			}

			return findings, nil
		},
	)
}

// D019: AsyncAPI/OpenAPI spec freshness.
// Detects projects that export AsyncAPI or OpenAPI specs but have event types
// not registered in the catalog. These missing types make the exported specs
// incomplete and stale.
//
//nolint:ireturn // factory returns public interface
func NewD019Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D019-stale-spec-freshness",
		func(_ context.Context) ([]finding.Finding, error) {
			if !projectExportsSpecs(ctx) {
				return nil, nil
			}

			eventTypes := collectEventNewTypes(ctx)
			catalogTypes := collectCatalogTypes(ctx)

			var missing []string

			for et := range eventTypes {
				if !catalogTypes[et] {
					missing = append(missing, et)
				}
			}

			if len(missing) == 0 {
				return nil, nil
			}

			pos, ok := lintutil.FirstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			f, err := finding.NewBuilder(
				"D019", toolName,
				"Exported specs are stale: "+fmtMissing(missing)+
					" exist as events but are not in the catalog",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
			).
				WithCategory(finding.CategoryDocumentation).
				WithConfidence(finding.ConfidenceLow).
				WithFixStrategy(finding.FixStrategySuggest).
				WithSuggestion("Register the missing event types in the catalog " +
					"and re-export the AsyncAPI/OpenAPI specs").
				WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
				Build()
			if err != nil {
				return nil, err
			}

			return []finding.Finding{f}, nil
		},
	)
}

// --- helpers ---

// collectEventNewTypes returns a set of event type strings from event.NewEvent
// and event.WithType calls in non-test files.
func collectEventNewTypes(ctx *analyzer.AnalysisContext) map[string]bool {
	types := make(map[string]bool)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			isNewEvent := pkg.Name == "event" && sel.Sel.Name == "NewEvent"
			isNewEventAlt := sel.Sel.Name == "NewEvent"

			if !isNewEvent && !isNewEventAlt {
				return true
			}

			typeName := extractStringArg(call, 0)
			if typeName != "" {
				types[typeName] = true
			}

			return true
		})
	}

	return types
}

// collectCatalogTypes returns a set of event type strings registered in the catalog.
func collectCatalogTypes(ctx *analyzer.AnalysisContext) map[string]bool {
	types := make(map[string]bool)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			if !isCatalogBuilder(call) {
				return true
			}

			typeName := extractStringArg(call, 0)
			if typeName != "" {
				types[typeName] = true
			}

			return true
		})
	}

	return types
}

// projectExportsSpecs reports whether any non-test file calls catalog export functions.
func projectExportsSpecs(ctx *analyzer.AnalysisContext) bool {
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

			name := sel.Sel.Name
			if name == "ExportAsyncAPI" || name == "ExportOpenAPI" ||
				name == "ExportEventCatalog" || name == "ExportD2" {
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

func isCatalogBuilder(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return pkg.Name == "catalog" && (sel.Sel.Name == "NewBuilder" || sel.Sel.Name == "Register")
}

func extractStringArg(call *ast.CallExpr, index int) string {
	if index >= len(call.Args) {
		return ""
	}

	lit, ok := call.Args[index].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}

	return strings.Trim(lit.Value, `"`)
}

func fmtMissing(types []string) string {
	if len(types) <= 3 {
		return strings.Join(types, ", ")
	}

	return strings.Join(types[:3], ", ") + " and others"
}
