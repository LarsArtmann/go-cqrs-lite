package correctness

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// Detects dual-write read models that mutate in-memory state before
// persisting to SQL/DB without a transaction. If the DB write fails, in-memory
// and persisted state diverge with no rollback.
//
// C024: Dual-write read model without rollback.
//
//nolint:ireturn // factory returns public interface
func NewC024Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C024-dual-write-without-rollback",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						continue
					}

					findings = append(findings, scanDualWrite(ctx, fn.Body)...)
				}
			}

			return findings, nil
		},
	)
}

// scanDualWrite checks a function body for the dual-write pattern:
// in-memory mutation + DB/SQL write call without a transaction.
func scanDualWrite(
	ctx *analyzer.AnalysisContext,
	body *ast.BlockStmt,
) []finding.Finding {
	var dbWriteCalls []*ast.CallExpr
	hasMutation := false
	hasTransaction := false

	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}

		if assign, ok := n.(*ast.AssignStmt); ok && isInMemoryMutation(assign) {
			hasMutation = true
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isTransactionCall(call) {
			hasTransaction = true
		}

		if isDualWriteCall(call) {
			dbWriteCalls = append(dbWriteCalls, call)
		}

		return true
	})

	if !hasMutation || hasTransaction || len(dbWriteCalls) == 0 {
		return nil
	}

	var findings []finding.Finding
	for _, call := range dbWriteCalls {
		reportDualWrite(ctx, &findings, call)
	}

	return findings
}

// isInMemoryMutation returns true for assignments to struct fields or map keys
// (e.g., s.field = ..., m[key] = ...), which represent in-memory state changes.
func isInMemoryMutation(assign *ast.AssignStmt) bool {
	for _, lhs := range assign.Lhs {
		switch lhs.(type) {
		case *ast.SelectorExpr, *ast.IndexExpr:
			return true
		}
	}

	return false
}

// isDualWriteCall returns true for method calls that persist to SQL/DB,
// indicating a dual-write pattern (e.g., syncToSQL, writeToDB, saveToSQL).
func isDualWriteCall(call *ast.CallExpr) bool {
	sel, ok := analyzer.SelectorFromExpr(call.Fun)
	if !ok {
		return false
	}

	name := strings.ToLower(sel.Sel.Name)
	hasStorage := strings.Contains(name, "sql") || strings.Contains(name, "db")
	hasWriteVerb := strings.Contains(name, "sync") ||
		strings.Contains(name, "write") ||
		strings.Contains(name, "save") ||
		strings.Contains(name, "persist")

	return hasStorage && hasWriteVerb
}

// isTransactionCall returns true for calls that start a transaction
// (Begin, BeginTx, RunInTx).
func isTransactionCall(call *ast.CallExpr) bool {
	return lintutil.CallSelectorMatches(call, "Begin", "BeginTx", "RunInTx")
}

func reportDualWrite(
	ctx *analyzer.AnalysisContext,
	findings *[]finding.Finding,
	call *ast.CallExpr,
) {
	pos := ctx.Fset.Position(call.Pos())
	sel, _ := analyzer.SelectorFromExpr(call.Fun)
	methodName := ""
	if sel != nil {
		methodName = sel.Sel.Name
	}

	f, err := finding.NewBuilder(
		"C024", toolName,
		methodName+"() — dual-write without transaction: "+
			"if DB write fails, in-memory and SQL state diverge",
		finding.SeverityError,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceMedium).
		WithFixStrategy(finding.FixStrategySuggest).
		WithSuggestion("Wrap both the in-memory mutation and the DB write in a " +
			"single transaction, or write to SQL first then update in-memory on success").
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	lintutil.AppendBuild(findings, f, err)
}
