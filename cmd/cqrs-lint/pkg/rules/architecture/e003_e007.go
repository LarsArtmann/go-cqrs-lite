package architecture

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// E003: Missing module boundary.
// Detects packages that mix commands, events, folds, and projections —
// suggesting a missing module boundary.
func NewE003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E003-missing-module-boundary",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			pkgConstructs := make(map[string]map[string]bool)

			for _, cmd := range ctx.Registry.Commands {
				pkg := cmd.Package
				if pkgConstructs[pkg] == nil {
					pkgConstructs[pkg] = make(map[string]bool)
				}

				pkgConstructs[pkg]["command"] = true
			}

			for _, evt := range ctx.Registry.Events {
				pkg := evt.Package
				if pkgConstructs[pkg] == nil {
					pkgConstructs[pkg] = make(map[string]bool)
				}

				pkgConstructs[pkg]["event"] = true
			}

			for _, fold := range ctx.Registry.Folds {
				pkg := fold.Package
				if pkgConstructs[pkg] == nil {
					pkgConstructs[pkg] = make(map[string]bool)
				}

				pkgConstructs[pkg]["fold"] = true
			}

			for _, proj := range ctx.Registry.Projections {
				pkg := proj.Package
				if pkgConstructs[pkg] == nil {
					pkgConstructs[pkg] = make(map[string]bool)
				}

				pkgConstructs[pkg]["projection"] = true
			}

			for pkg, constructs := range pkgConstructs {
				if len(constructs) < 3 {
					continue
				}

				types := make([]string, 0, len(constructs))
				for t := range constructs {
					types = append(types, t)
				}

				f, err := finding.NewBuilder(
					"E003", toolName,
					fmt.Sprintf("Package %s mixes %d CQRS concerns (%s) — split into domain/infrastructure boundaries",
						pkg, len(constructs), strings.Join(types, ", ")),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
				).
					WithCategory(finding.CategoryStructure).
					WithConfidence(finding.ConfidenceLow).
					WithSuggestion("Separate commands/events (domain) from projections/handlers (infrastructure) into distinct packages").
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// E007: Query without handler.
// Detects query types (struct names ending in "Query") defined but never
// registered with a dispatcher via RegisterTyped or RegisterQuery.
//
// "Request" suffix was removed from the heuristic because it matches HTTP
// request DTOs (LoginRequest, RegisterRequest) that are not CQRS queries.
// Confidence is Low because query registration can happen via patterns the
// analyzer doesn't track (e.g., direct dispatcher.Register with a string type).
func NewE007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E007-query-without-handler",
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

					if !strings.HasSuffix(ts.Name.Name, "Query") {
						return true
					}

					if ctx.Registry.IsCommandRegistered(ts.Name.Name) {
						return true
					}

					pos := ctx.Fset.Position(ts.Pos())

					f, err := finding.NewBuilder(
						"E007", toolName,
						fmt.Sprintf("Query type %q has no registered handler — dispatching it will fail", ts.Name.Name),
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryStructure).
						WithConfidence(finding.ConfidenceLow).
						WithSuggestion("Register the query via query.RegisterTyped or dispatcher.RegisterTyped").
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
