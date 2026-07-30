package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// Detects mutex held during payload decode. Lock()/RLock() followed by
// DecodePayloadAuto or json.Unmarshal before Unlock()/RUnlock() serializes
// all event processing unnecessarily — decode is CPU-bound and doesn't need
// lock protection.
//
// C021: Mutex held during payload decode.
//
//nolint:ireturn // factory returns public interface
func NewC021Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C021-mutex-held-during-decode",
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

					findings = append(findings, scanMutexDecode(ctx, fn.Body)...)
				}
			}

			return findings, nil
		},
	)
}

// scanMutexDecode walks a function body in source order using an ancestor
// stack. It tracks a "locked" state: Lock/RLock sets it true, Unlock/RUnlock
// clears it. Deferred unlock calls are ignored (they don't release the lock
// at their source position). Function literals are not entered — they have
// their own lock context. While locked, any decode call (DecodePayloadAuto,
// DecodePayload, json.Unmarshal) is flagged.
func scanMutexDecode(
	ctx *analyzer.AnalysisContext,
	body *ast.BlockStmt,
) []finding.Finding {
	var findings []finding.Finding
	var ancestors []ast.Node
	locked := false

	ast.Inspect(body, func(n ast.Node) bool {
		if n == nil {
			if len(ancestors) > 0 {
				ancestors = ancestors[:len(ancestors)-1]
			}

			return false
		}

		// Don't enter function literals — separate lock context.
		if _, ok := n.(*ast.FuncLit); ok && len(ancestors) > 0 {
			return false
		}

		ancestors = append(ancestors, n)

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		inDefer := hasDeferAncestor(ancestors)

		switch {
		case isLockCall(call) && !inDefer:
			locked = true
		case isUnlockCall(call) && !inDefer:
			locked = false
		case locked && isDecodeCall(call):
			reportMutexDecode(ctx, &findings, call)
		}

		return true
	})

	return findings
}

func hasDeferAncestor(ancestors []ast.Node) bool {
	for _, a := range ancestors {
		if _, ok := a.(*ast.DeferStmt); ok {
			return true
		}
	}

	return false
}

func isLockCall(call *ast.CallExpr) bool {
	sel, ok := analyzer.SelectorFromExpr(call.Fun)
	if !ok {
		return false
	}

	return sel.Sel.Name == "Lock" || sel.Sel.Name == "RLock"
}

func isUnlockCall(call *ast.CallExpr) bool {
	sel, ok := analyzer.SelectorFromExpr(call.Fun)
	if !ok {
		return false
	}

	return sel.Sel.Name == "Unlock" || sel.Sel.Name == "RUnlock"
}

func isDecodeCall(call *ast.CallExpr) bool {
	sel, ok := analyzer.SelectorFromExpr(call.Fun)
	if !ok {
		return false
	}

	method := sel.Sel.Name
	if method == "DecodePayloadAuto" || method == "DecodePayload" {
		return true
	}

	pkg := analyzer.SelectorPackage(sel)

	return pkg == "json" && method == "Unmarshal"
}

func reportMutexDecode(
	ctx *analyzer.AnalysisContext,
	findings *[]finding.Finding,
	call *ast.CallExpr,
) {
	pos := ctx.Fset.Position(call.Pos())

	f, err := finding.NewBuilder(
		"C021", toolName,
		"Payload decode while mutex is held — serializes all event processing",
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceHigh).
		WithFixStrategy(finding.FixStrategySuggest).
		WithSuggestion("Decode outside the lock, then acquire the lock only for the map mutation").
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	lintutil.AppendBuild(findings, f, err)
}
