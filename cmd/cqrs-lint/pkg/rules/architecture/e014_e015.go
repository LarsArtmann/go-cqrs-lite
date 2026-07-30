package architecture

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// E014: No read-your-writes consistency.
// Detects projects using projectionhost that don't wait for projection drain
// before responding to commands. The read model may be stale when the command
// handler returns, leading to "I just created it but it's not there" bugs.
// The finding checks for absence of Drain/Wait/Sync calls that would block
// until the projection catches up.
//
//nolint:ireturn // factory returns public interface
func NewE014Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E014-no-read-your-writes",
		func(_ context.Context) ([]finding.Finding, error) {
			if !importsPathSuffix(ctx, "go-cqrs-lite/projectionhost") {
				return nil, nil
			}

			// Check for drain/wait patterns: host.Stop(), host.Sync(), or
			// any call containing "Drain"/"Wait"/"Sync" in the project.
			if projectCalls(ctx, "host", "Stop") ||
				projectCalls(ctx, "host", "Sync") ||
				projectHasCallContaining(ctx, "Drain") ||
				projectHasCallContaining(ctx, "WaitFor") {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleFinding(
				ctx,
				"E014",
				"Project uses projectionhost but has no projection drain/wait call — "+
					"the read model may be stale when the command handler returns",
				"Call host.Stop() or a sync/drain method before responding to commands, "+
					"or use a read-your-writes strategy (synchronous projection for the "+
					"command's own stream)",
				pos,
				finding.SeverityInfo,
				finding.ConfidenceLow,
			), nil
		},
	)
}

// E015: Watermill EventBus without ordered delivery.
// Detects projects that configure a Watermill EventBus with
// BlockPublishUntilSubscriberAck set to false. This breaks ordered event
// delivery — projections that depend on event ordering (e.g., building a
// sequence of state changes) will receive events out of order, leading to
// inconsistent read models. The library defaults BlockPublishUntilSubscriberAck
// to true, so an explicit false is a deliberate override.
//
//nolint:ireturn // factory returns public interface
func NewE015Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E015-watermill-no-ordered-delivery",
		func(_ context.Context) ([]finding.Finding, error) {
			if !importsPathSuffix(ctx, "go-cqrs-lite/watermill") {
				return nil, nil
			}

			if !findKeyBoolLit(ctx, "BlockPublishUntilSubscriberAck", false) {
				return nil, nil
			}

			pos, ok := firstKeyBoolPos(ctx, "BlockPublishUntilSubscriberAck", false)
			if !ok {
				pos, _ = firstFilePos(ctx)
			}

			return singleFinding(
				ctx,
				"E015",
				"BlockPublishUntilSubscriberAck: false breaks ordered event delivery — "+
					"projections that depend on event ordering will receive events out of order",
				"Remove the explicit false (the library defaults to true) or ensure all "+
					"projections are order-independent before disabling this guarantee",
				pos,
				finding.SeverityWarning,
				finding.ConfidenceHigh,
			), nil
		},
	)
}

// projectHasCallContaining reports whether any non-test file calls a function
// whose name contains substring, regardless of package qualifier.
func projectHasCallContaining(ctx *analyzer.AnalysisContext, substring string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			if containsSubstring(sel.Sel.Name, substring) {
				found = true
				return false
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// firstKeyBoolPos returns the position of the first composite-literal
// key-value pair where key == keyName and value is a bool matching wantBool.
func firstKeyBoolPos(
	ctx *analyzer.AnalysisContext,
	keyName string,
	wantBool bool,
) (tokenPosition, bool) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var hit ast.Node

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				ident, ok := kv.Key.(*ast.Ident)
				if !ok || ident.Name != keyName {
					continue
				}

				bid, ok := kv.Value.(*ast.Ident)
				if !ok {
					continue
				}

				if (wantBool && bid.Name == "true") || (!wantBool && bid.Name == "false") {
					hit = kv
					return false
				}
			}

			return true
		})

		if hit != nil {
			p := ctx.Fset.Position(hit.Pos())
			return tokenPosition{Filename: p.Filename, Line: p.Line, Column: p.Column}, true
		}
	}

	return tokenPosition{}, false
}

// tokenPosition is a local alias to avoid importing go/token in the return
// signature of firstKeyBoolPos (keeps the file import list lean).
type tokenPosition struct {
	Filename string
	Line     int
	Column   int
}

// containsSubstring reports whether s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(s, substr)
}
