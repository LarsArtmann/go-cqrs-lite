package api

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A015: Global mutable state.
// Detects package-level var declarations that are mutated at runtime.
// Only flags globals that are actually written to after initialization
// (not read-only lookup tables initialized at package load).
// Suppressed for non-server modules — race conditions require concurrent
// access, which only happens in server deployments. Evaluated per-module via
// ProfileForFile so a library module is not flagged when an example sub-module
// happens to run a server.
//
//nolint:ireturn // factory returns public interface
func NewA015Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A015-global-mutable-state",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			candidates := collectGlobalMutables(ctx)

			for _, name := range candidates {
				if !isGlobalWrittenAfterInit(ctx, name.origName) {
					continue
				}

				f, err := finding.NewBuilder(
					"A015", toolName,
					fmt.Sprintf("Global mutable variable %s — may cause race conditions in concurrent handlers", name.origName),
					finding.SeverityError,
					finding.Pos(finding.FilePath(name.file), name.line, name.col),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion("Inject dependencies via constructor parameters or use sync.OnceValue for lazy init").
					WithSnippet(ctx.SourceLine(name.file, name.line)).
					Build()
				if err != nil {
					continue
				}

				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

type globalCandidate struct {
	origName string
	file     string
	line     int
	col      int
}

// collectGlobalMutables finds package-level var declarations whose names
// contain "cache", "registry", or "instance" (excluding Err/Sentinel errors).
func collectGlobalMutables(ctx *analyzer.AnalysisContext) []globalCandidate {
	var candidates []globalCandidate

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		// Evaluate per-module: race conditions require concurrent access,
		// which only happens in server deployments. Using the primary profile
		// would flag library files when an example sub-module runs a server.
		if !ctx.ProfileForFile(gf.Path).HasServer {
			continue
		}

		for _, decl := range gf.AST.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok.String() != "var" {
				continue
			}

			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) == 0 {
					continue
				}

				name := vs.Names[0].Name
				lower := strings.ToLower(name)

				if strings.HasPrefix(name, "Err") || strings.HasPrefix(name, "Sentinel") {
					continue
				}

				if !strings.Contains(lower, "cache") &&
					!strings.Contains(lower, "registry") &&
					!strings.Contains(lower, "instance") {
					continue
				}

				pos := ctx.Fset.Position(vs.Pos())
				candidates = append(candidates, globalCandidate{
					origName: name,
					file:     pos.Filename,
					line:     pos.Line,
					col:      pos.Column,
				})
			}
		}
	}

	return candidates
}

// isGlobalWrittenAfterInit checks if a package-level variable is assigned to
// anywhere in the codebase inside a function body (excluding the declaration).
// Read-only globals (lookup tables initialized at package load and never
// written again) return false.
func isGlobalWrittenAfterInit(ctx *analyzer.AnalysisContext, varName string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var found bool

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			for _, lhs := range assign.Lhs {
				// Direct assignment: globalVar = newVal
				if id, ok := lhs.(*ast.Ident); ok && id.Name == varName {
					found = true
					return false
				}

				// Index assignment: globalVar[key] = val
				if idx, ok := lhs.(*ast.IndexExpr); ok {
					if ident, ok := idx.X.(*ast.Ident); ok && ident.Name == varName {
						found = true
						return false
					}
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
