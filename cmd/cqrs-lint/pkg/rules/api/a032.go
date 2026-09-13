package api

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
	"github.com/larsartmann/go-finding"
)

// A032: String IDs instead of branded IDs.
// Detects struct fields named "*ID" or "*Id" with type `string` or `int`
// instead of `id.Of[T]`. Plain string/int IDs lose type safety — the
// compiler can't prevent mixing UserID with OrderID.
//
// Only fires in files that import the `id` package from go-cqrs-lite.
//
//nolint:ireturn // factory returns public interface
func NewA032Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A032-string-id-instead-of-branded",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if !fileImportsIDPackage(gf.AST) {
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

					if structHasFormTag(st) {
						return true
					}

					if isDisplayPackage(gf.Path) {
						return true
					}

					for _, field := range st.Fields.List {
						if len(field.Names) == 0 {
							continue
						}

						name := field.Names[0].Name
						if !looksLikeIDField(name) {
							continue
						}

						if !isPlainIDType(field.Type) {
							continue
						}

						pos := ctx.Fset.Position(field.Pos())

						f, err := finding.NewBuilder(
							"A032", toolName,
							"Field "+ts.Name.Name+"."+name+
								" is string/int — use id.Of[T] branded ID for type safety",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceHigh).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Use `type UserID = id.Of[id.UserMarker]` and type the field as UserID").
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

// looksLikeIDField reports whether the field name ends in "ID" or "Id".
func looksLikeIDField(name string) bool {
	return strings.HasSuffix(name, "ID") || strings.HasSuffix(name, "Id")
}

// isPlainIDType reports whether the field type is plain string or int.
func isPlainIDType(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}

	return id.Name == "string" || id.Name == "int"
}

// fileImportsIDPackage reports whether the file imports the go-cqrs-lite id package.
func fileImportsIDPackage(root ast.Node) bool {
	hasID := false

	ast.Inspect(root, func(n ast.Node) bool {
		imp, ok := n.(*ast.ImportSpec)
		if !ok {
			return true
		}

		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, "go-cqrs-lite") &&
			(strings.Contains(path, "/id/") || strings.HasSuffix(path, "/id")) {
			hasID = true
			return false
		}

		return true
	})

	return hasID
}

// structHasFormTag reports whether any field in the struct has a `form:` tag,
// indicating it's an HTTP binding DTO rather than a domain type.
func structHasFormTag(st *ast.StructType) bool {
	if st.Fields == nil {
		return false
	}

	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}

		if strings.Contains(field.Tag.Value, "form:") {
			return true
		}
	}

	return false
}

// isDisplayPackage reports whether the file path suggests a display/view/DTO
// package where branded IDs add no value.
func isDisplayPackage(path string) bool {
	lower := strings.ToLower(path)

	return strings.Contains(lower, "dashboard") ||
		strings.Contains(lower, "/ui/") ||
		strings.Contains(lower, "/views/") ||
		strings.Contains(lower, "/view/") ||
		strings.Contains(lower, "/display/") ||
		strings.Contains(lower, "/dto/") ||
		strings.Contains(lower, "/frontend/") ||
		strings.Contains(lower, "/webui/")
}
