package architecture

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// E003: Missing module boundary.
// Detects packages that mix commands, events, folds, and projections,
// suggesting a missing module boundary.
//
// Each finding is anchored at the first contributing construct of its package
// rather than at go.mod:1:1. A shared position made every E003 finding share
// one finding ID, so ID-keyed consumers collapsed N packages into one
// (nondeterministically, since the package map iterates randomly).
//
//nolint:ireturn // factory returns public interface
func NewE003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E003-missing-module-boundary",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			boundaries := newPackageBoundaries()

			for _, cmd := range ctx.Registry.Commands {
				boundaries.record(cmd.Package, "command", cmd.File, cmd.Pos)
			}

			for _, evt := range ctx.Registry.Events {
				boundaries.record(evt.Package, "event", evt.File, evt.Pos)
			}

			for _, fold := range ctx.Registry.Folds {
				boundaries.record(fold.Package, "fold", fold.File, fold.Pos)
			}

			for _, proj := range ctx.Registry.Projections {
				boundaries.record(proj.Package, "projection", proj.File, proj.Pos)
			}

			for _, pkg := range boundaries.sortedPackages() {
				constructs := boundaries.packages[pkg]
				if len(constructs) < 3 {
					continue
				}

				f, err := findingTemplate.Builder(
					"E003",
					fmt.Sprintf("Package %s mixes %d CQRS concerns (%s) — split into domain/infrastructure boundaries",
						pkg, len(constructs), strings.Join(sortedKinds(constructs), ", ")),
					finding.SeverityWarning,
					boundaries.pos(pkg, ctx.ProjectRoot),
				).
					WithCategory(finding.CategoryStructure).
					WithConfidence(finding.ConfidenceLow).
					WithSuggestion("Separate commands/events (domain) from projections/handlers (infrastructure) into distinct " +
						"packages").
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

type constructAnchor struct {
	file string
	pos  token.Position
}

// packageBoundaries accumulates the CQRS construct kinds per package plus the
// earliest source position that contributed one, so each package's finding has
// a unique, jumpable location.
type packageBoundaries struct {
	packages map[string]map[string]bool
	anchors  map[string]constructAnchor
}

func newPackageBoundaries() *packageBoundaries {
	return &packageBoundaries{
		packages: make(map[string]map[string]bool),
		anchors:  make(map[string]constructAnchor),
	}
}

func (b *packageBoundaries) record(pkg, kind, file string, pos token.Position) {
	if b.packages[pkg] == nil {
		b.packages[pkg] = make(map[string]bool)
	}

	b.packages[pkg][kind] = true

	anchor, ok := b.anchors[pkg]
	if !ok || precedes(anchor, constructAnchor{file: file, pos: pos}) {
		b.anchors[pkg] = constructAnchor{file: file, pos: pos}
	}
}

func (b *packageBoundaries) sortedPackages() []string {
	names := make([]string, 0, len(b.packages))
	for pkg := range b.packages {
		names = append(names, pkg)
	}

	slices.Sort(names)

	return names
}

// pos returns the package's anchor location, falling back to go.mod:1:1 only
// when no construct recorded a usable position.
func (b *packageBoundaries) pos(pkg, projectRoot string) finding.Position {
	anchor := b.anchors[pkg]
	if anchor.pos.IsValid() {
		return finding.Pos(
			finding.FilePath(anchor.pos.Filename),
			anchor.pos.Line,
			anchor.pos.Column,
		)
	}

	if anchor.file != "" {
		return finding.Pos(finding.FilePath(anchor.file), 1, 1)
	}

	return finding.Pos(finding.FilePath(projectRoot+"/go.mod"), 1, 1)
}

// precedes reports whether a sorts before b by file then position.
func precedes(a, b constructAnchor) bool {
	if a.file != b.file {
		return a.file < b.file
	}

	if a.pos.Line != b.pos.Line {
		return a.pos.Line < b.pos.Line
	}

	return a.pos.Column < b.pos.Column
}

func sortedKinds(constructs map[string]bool) []string {
	kinds := make([]string, 0, len(constructs))
	for kind := range constructs {
		kinds = append(kinds, kind)
	}

	slices.Sort(kinds)

	return kinds
}

// E007: Query without handler.
// Detects query types (struct names ending in "Query") defined but never
// registered with a dispatcher via RegisterTyped or RegisterQuery.
//
// "Request" suffix was removed from the heuristic because it matches HTTP
// request DTOs (LoginRequest, RegisterRequest) that are not CQRS queries.
// Confidence is Low because query registration can happen via patterns the
// analyzer doesn't track (e.g., direct dispatcher.Register with a string type).
//
//nolint:ireturn // factory returns public interface
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

					// Require a Type() method — real CQRS query types implement
					// query.Query's Type() method. DTOs whose name happens to end
					// in "Query" (form-binding structs, filter objects) do not.
					if !ctx.Registry.TypesWithTypeMethod[ts.Name.Name] {
						return true
					}

					if ctx.Registry.IsCommandRegistered(ts.Name.Name) {
						return true
					}

					pos := ctx.Fset.Position(ts.Pos())

					f, err := findingTemplate.Builder(
						"E007",
						fmt.Sprintf("Query type %q has no registered handler — dispatching it will fail", ts.Name.Name),
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryStructure).
						WithConfidence(finding.ConfidenceLow).
						WithSuggestion("Register the query via query.RegisterTyped or dispatcher.RegisterTyped, or suppress if " +
							"registration is runtime/generic (E007 cannot trace through helpers)").
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
