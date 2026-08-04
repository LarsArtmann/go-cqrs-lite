package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// C035: Unprotected map field in read model/handler struct.
// Detects structs with map fields that lack sync.Mutex or sync.RWMutex
// protection. In concurrent event handlers (SubscribeAll), unprotected map
// access causes data races — the #1 concurrency bug in read model projections.
//
//nolint:ireturn // factory returns public interface
func NewC035Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C035-unprotected-map-field",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

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
						typeSpec, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}

						structType, ok := typeSpec.Type.(*ast.StructType)
						if !ok {
							continue
						}

						structName := typeSpec.Name.Name
						if !looksLikeReadModel(structName, gf.Path) {
							continue
						}

						mapFields := findMapFields(structType.Fields)
						if len(mapFields) == 0 {
							continue
						}

						if hasMutexField(structType.Fields) {
							continue
						}

						pos := ctx.Fset.Position(typeSpec.Pos())

						names := make([]string, len(mapFields))
						copy(names, mapFields)

						f, err := finding.NewBuilder(
							"C035", toolName,
							fmt.Sprintf(
								"Struct %s has map field(s) [%s] without sync.Mutex/sync.RWMutex — data race risk in concurrent handlers",
								structName, strings.Join(names, ", "),
							),
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceMedium).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Add a sync.RWMutex field and guard map access with Lock/Unlock, or use sync.Map for concurrent read-heavy workloads").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						lintutil.AppendBuild(&findings, f, err)
					}
				}
			}

			return findings, nil
		},
	)
}

// looksLikeReadModel returns true if the struct name or file path suggests
// it is a read model, projection, or event handler that processes events
// concurrently.
func looksLikeReadModel(structName, filePath string) bool {
	upper := strings.ToUpper(structName)

	for _, suffix := range []string{
		"VIEW", "READMODEL", "READMODELSTATE", "PROJECTION",
		"HANDLER", "PROJECTOR", "STORE", "CACHE",
	} {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}

	base := lintutil.BaseFileName(filePath)

	return base == "views" || base == "projection" || base == "readmodel" ||
		base == "handler" || base == "handlers"
}

// findMapFields returns the names of map-type fields in a struct.
func findMapFields(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}

	var names []string

	for _, field := range fields.List {
		if _, ok := field.Type.(*ast.MapType); !ok {
			continue
		}

		if len(field.Names) == 0 {
			names = append(names, "anonymous")
			continue
		}

		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}

	return names
}

// hasMutexField returns true if the struct has a sync.Mutex, sync.RWMutex,
// or sync.Map field.
func hasMutexField(fields *ast.FieldList) bool {
	if fields == nil {
		return false
	}

	for _, field := range fields.List {
		sel, ok := field.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			continue
		}

		if ident.Name == "sync" {
			switch sel.Sel.Name {
			case "Mutex", "RWMutex", "Map":
				return true
			}
		}
	}

	return false
}
