package correctness

import (
	"context"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// Detects idempotency TTL mismatches: a named TTL constant is defined but a
// different literal value is passed to idempotency.NewMemoryStore or
// middleware.CommandIdempotency/EventIdempotency/QueryIdempotency. The
// constant is dead/misleading code — the actual TTL used at runtime differs
// from what the constant suggests.
//
// C026: Idempotency TTL mismatch.
//
//nolint:ireturn // factory returns public interface
func NewC026Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C026-idempotency-ttl-mismatch",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			ttlConsts := collectTTLConsts(ctx)
			if len(ttlConsts) == 0 {
				return nil, nil
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					ttlArg := extractTTLArg(call)
					if ttlArg == nil {
						return true
					}

					if isTTLConstRef(ttlArg, ttlConsts) {
						return true
					}

					reportTTLMismatch(ctx, &findings, call, ttlConsts)

					return true
				})
			}

			return findings, nil
		},
	)
}

// collectTTLConsts returns the set of constant names containing "TTL"
// (case-insensitive) declared at package level across all non-test files.
func collectTTLConsts(ctx *analyzer.AnalysisContext) map[string]bool {
	consts := make(map[string]bool)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.CONST {
				continue
			}

			for _, spec := range genDecl.Specs {
				valSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}

				for _, name := range valSpec.Names {
					if strings.Contains(strings.ToUpper(name.Name), "TTL") {
						consts[name.Name] = true
					}
				}
			}
		}
	}

	return consts
}

// extractTTLArg returns the TTL argument from idempotency/middleware calls:
// - idempotency.NewMemoryStore(ttl) → arg[0]
// - middleware.CommandIdempotency(store, ttl, ...) → arg[1]
// - middleware.EventIdempotency(store, ttl, ...) → arg[1]
// - middleware.QueryIdempotency(store, ttl, ...) → arg[1]
func extractTTLArg(call *ast.CallExpr) ast.Expr {
	sel, ok := analyzer.SelectorFromExpr(call.Fun)
	if !ok {
		return nil
	}

	method := sel.Sel.Name
	pkg := analyzer.SelectorPackage(sel)

	switch {
	case method == "NewMemoryStore" && pkg == "idempotency":
		if len(call.Args) >= 1 {
			return call.Args[0]
		}
	case (method == "CommandIdempotency" ||
		method == "EventIdempotency" ||
		method == "QueryIdempotency") && pkg == "middleware":
		if len(call.Args) >= 2 {
			return call.Args[1]
		}
	}

	return nil
}

// isTTLConstRef returns true if the expression is an identifier referencing
// one of the named TTL constants.
func isTTLConstRef(expr ast.Expr, ttlConsts map[string]bool) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}

	return ttlConsts[ident.Name]
}

func reportTTLMismatch(
	ctx *analyzer.AnalysisContext,
	findings *[]finding.Finding,
	call *ast.CallExpr,
	ttlConsts map[string]bool,
) {
	pos := ctx.Fset.Position(call.Pos())

	constNames := make([]string, 0, len(ttlConsts))
	for name := range ttlConsts {
		constNames = append(constNames, name)
	}

	f, err := finding.NewBuilder(
		"C026", toolName,
		"Idempotency TTL mismatch — literal TTL passed but a TTL constant ("+
			strings.Join(constNames, ", ")+
			") is defined and not used here",
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceHigh).
		WithFixStrategy(finding.FixStrategySuggest).
		WithSuggestion("Use the defined TTL constant instead of a bare literal, " +
			"or remove the dead constant if the literal is intentional").
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	lintutil.AppendBuild(findings, f, err)
}
