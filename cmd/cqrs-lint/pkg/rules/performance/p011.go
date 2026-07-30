package performance

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// P011: Unbounded in-memory growth in read models.
// Detects `map[...]` fields in structs that are used as in-memory read models
// (have Handle/HandleEvent methods or appear in SubscribeAll handler context).
// A map field without eviction (size limit, TTL, LRU) grows unboundedly as
// events accumulate, leading to OOM kills in long-running services.
//
//nolint:ireturn // factory returns public interface
func NewP011Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P011-unbounded-memory-growth",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

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

					if !isReadModelStruct(ts.Name.Name) {
						return true
					}

					for _, field := range st.Fields.List {
						if !hasMapType(field.Type) {
							continue
						}

						if hasSyncMutex(st) {
							continue
						}

						pos := ctx.Fset.Position(field.Pos())
						fieldName := fieldName(field)

						f, err := finding.NewBuilder(
							"P011", toolName,
							"Read model "+ts.Name.Name+"."+fieldName+
								" is a map without size limit — grows unboundedly as events accumulate, risk of OOM",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryPerformance).
							WithConfidence(finding.ConfidenceMedium).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Use a bounded cache (kv.Cache, LRU, or map with max-size eviction)").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						lintutil.AppendBuild(&findings, f, err)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// readModelKeywords identifies struct names that look like in-memory read models.
//
//nolint:gochecknoglobals // read-only keyword list
var readModelKeywords = []string{
	"readmodel", "readmodelstore", "viewstore", "viewcache",
	"projection", "projector", "handlerstore",
}

// isReadModelStruct reports whether the struct name suggests an in-memory
// read model (ViewStore, ReadModel, Projection, etc). We check the struct
// name rather than method signatures because the struct definition may be
// in a different file from the Handle/Apply methods.
func isReadModelStruct(name string) bool {
	lower := strings.ToLower(name)
	for _, kw := range readModelKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// hasMapType reports whether the field type is a map.
func hasMapType(expr ast.Expr) bool {
	_, ok := expr.(*ast.MapType)
	return ok
}

// hasSyncMutex reports whether the struct has a sync.Mutex or sync.RWMutex field,
// which indicates the developer is at least thinking about concurrent access
// (though it doesn't prevent unbounded growth).
func hasSyncMutex(st *ast.StructType) bool {
	if st.Fields == nil {
		return false
	}

	for _, field := range st.Fields.List {
		if sel, ok := field.Type.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "sync" {
				name := sel.Sel.Name
				if name == "Mutex" || name == "RWMutex" || name == "Map" {
					return true
				}
			}
		}
	}

	return false
}

// fieldName extracts the field name for reporting.
func fieldName(field *ast.Field) string {
	if len(field.Names) > 0 {
		return field.Names[0].Name
	}

	return strings.ToLower(analyzer.ExprString(field.Type))
}
