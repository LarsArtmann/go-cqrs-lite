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

// Detects fmt.Errorf without %w in files that import go-cqrs-lite modules.
// While D006 catches this globally at info severity, C025 escalates to warning
// for CQRS consumer code — bare fmt.Errorf in event/command/query handlers
// loses error classification and breaks the 6-family error taxonomy.
// D006 defers fmt.Errorf in CQRS files to C025 to avoid double-reporting.
//
// C025: fmt.Errorf without %w in CQRS error paths.
//
//nolint:ireturn // factory returns public interface
func NewC025Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C025-bare-errorf-in-cqrs",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			sentinels := lintutil.CollectPkgLevelVarCalls(ctx)

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if !fileImportsCQRS(gf.AST) {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					if !lintutil.IsFmtErrorf(call) || lintutil.HasWrapVerb(call) {
						return true
					}

					if sentinels[call.Pos()] {
						return true
					}

					reportBareErrorf(ctx, &findings, call)

					return true
				})
			}

			return findings, nil
		},
	)
}

// fileImportsCQRS returns true if the file's import declarations include
// any go-cqrs-lite module path.
func fileImportsCQRS(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)
		if analyzer.IsCQRSModulePath(path) {
			return true
		}
	}

	return false
}

func reportBareErrorf(
	ctx *analyzer.AnalysisContext,
	findings *[]finding.Finding,
	call *ast.CallExpr,
) {
	pos := ctx.Fset.Position(call.Pos())

	f, err := finding.NewBuilder(
		"C025", toolName,
		fmt.Sprintf(
			"fmt.Errorf without %%w in CQRS code at %s — "+
				"loses error classification, breaks the 6-family taxonomy",
			pos.String(),
		),
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceMedium).
		WithSuggestion("Use fmt.Errorf(\"...: %w\", err) to wrap, or " +
			"errorfamily.WrapConflict/WrapTransient/etc. for classified errors").
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	lintutil.AppendBuild(findings, f, err)
}
