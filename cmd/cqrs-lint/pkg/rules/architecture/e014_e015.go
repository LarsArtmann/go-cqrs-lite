package architecture

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// E014: No read-your-writes consistency.
// Detects projects using projectionhost that don't drain or sync projections
// before responding to commands. The read model may be stale when the command
// handler returns, leading to "I just created it but it's not there" bugs.
//
// Uses type-aware receiver matching: checks that Drain/Sync/Flush/WaitFor is
// called on a projectionhost.Host variable. Falls back to variable-name
// heuristic ("host", "proj") in unit tests without type info.
//
//nolint:ireturn // factory returns public interface
func NewE014Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E014-no-read-your-writes",
		func(_ context.Context) ([]finding.Finding, error) {
			if !importsPathSuffix(ctx, "go-cqrs-lite/projectionhost") {
				return nil, nil
			}

			// Type-aware check: drain/sync/flush/wait on a projectionhost type.
			if _, found := projectCallsMethodOnType(
				ctx,
				[]string{"Drain", "Sync", "Flush", "WaitFor"},
				[]string{"go-cqrs-lite/projectionhost", "cqrs-lite/projectionhost"},
			); found {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleFinding(
				ctx,
				"E014",
				"Project uses projectionhost but has no projection drain/sync/flush call — "+
					"the read model may be stale when the command handler returns",
				"Call a drain/sync method (e.g., host.Sync() or host.Drain()) before "+
					"responding to commands, or use a read-your-writes strategy "+
					"(synchronous projection for the command's own stream)",
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

// firstKeyBoolPos returns the position of the first composite-literal
// key-value pair where key == keyName and value is a bool matching wantBool.
func firstKeyBoolPos(
	ctx *analyzer.AnalysisContext,
	keyName string,
	wantBool bool,
) (token.Position, bool) {
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
			return ctx.Fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}
