package api

import (
	"context"
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A005: Custom projection runner.
// Detects bus.SubscribeAll + manual switch without projectionhost.
//
// To avoid false positives on fire-and-forget fan-out subscribers (SSE
// broadcasters, stats notifiers), the rule inspects the SubscribeAll callback
// body and suppresses when it contains broadcast/notify signals (Notify,
// Broadcast, Send) and NO persistence signals (Save, Set, Upsert, ...).
// Persistence writes are the defining trait of a real projection; pure
// broadcasts never persist. See feedback: docs/feedback/2026-07-16_DiscordSync.
func NewA005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A005-custom-projection-runner",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				hasSubscribeAll := false

				var subscribePos token.Position

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					if sel.Sel.Name == "SubscribeAll" {
						hasSubscribeAll = true
						subscribePos = ctx.Fset.Position(call.Pos())
					}

					return true
				})

				if hasSubscribeAll {
					// Check if projectionhost is imported.
					usesProjectionHost := false

					for _, imp := range gf.AST.Imports {
						if imp.Path != nil && strings.Contains(imp.Path.Value, "projectionhost") {
							usesProjectionHost = true

							break
						}
					}

					if !usesProjectionHost {
						// Inspect each SubscribeAll callback: if it only
						// broadcasts/notifies without persisting, it's a
						// fire-and-forget fan-out, not a manual projection.
						if !isManualProjection(gf.AST) {
							continue
						}

						f, err := finding.NewBuilder(
							"A005", toolName,
							"Manual projection via bus.SubscribeAll — use projectionhost.Host for checkpoint persistence, dead-letter queues, and crash recovery",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(subscribePos.Filename), subscribePos.Line, subscribePos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceMedium).
							WithSuggestion("Register projections with projectionhost.New(journal, checkpointStore) instead of manual bus.SubscribeAll + switch").
							WithSnippet(ctx.SourceLine(subscribePos.Filename, subscribePos.Line)).
							Build()
						if err == nil {
							findings = append(findings, f)
						}
					}
				}
			}

			return findings, nil
		},
	)
}

// isManualProjection reports whether any bus.SubscribeAll callback in the file
// looks like a genuine manual projection (writes to a store) rather than a
// pure broadcast/notify fan-out.
//
// A callback is treated as a NON-projection (fan-out) only when it contains a
// broadcast/notify call and no persistence call. Everything else — empty
// bodies, delegating helpers, persistence writes — is conservatively reported.
func isManualProjection(fileAST *ast.File) bool {
	anyProjectionCandidate := false

	ast.Inspect(fileAST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok || sel.Sel.Name != "SubscribeAll" {
			return true
		}

		callback := extractCallbackFuncLit(call)
		if callback == nil {
			// Named-function subscriber — can't inspect the body. Treat as
			// projection candidate (conservative).
			anyProjectionCandidate = true

			return true
		}

		hasBroadcast, hasPersist := classifyCallbackBody(callback.Body)
		// Suppress only for the clear fan-out shape: broadcasts without any
		// persistence. Anything else stays a candidate.
		if hasPersist || !hasBroadcast {
			anyProjectionCandidate = true
		}

		return true
	})

	return anyProjectionCandidate
}

// extractCallbackFuncLit returns the FuncLit callback passed to SubscribeAll,
// or nil if the argument is not a function literal.
func extractCallbackFuncLit(call *ast.CallExpr) *ast.FuncLit {
	for _, v := range slices.Backward(call.Args) {
		if fn, ok := v.(*ast.FuncLit); ok {
			return fn
		}
	}

	return nil
}

// classifyCallbackBody inspects a SubscribeAll callback body and reports
// whether it contains broadcast/notify calls and/or persistence calls.
func classifyCallbackBody(body *ast.BlockStmt) (hasBroadcast, hasPersist bool) {
	if body == nil {
		return false, false
	}

	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok {
			return true
		}

		method := sel.Sel.Name

		if !hasBroadcast && slices.Contains([]string{
			"Broadcast", "Dispatch", "Emit", "FanOut", "Fanout", "Flush",
			"Forward", "Multicast", "Notify", "Publish", "Push", "Send", "WriteTo",
		}, method) {
			hasBroadcast = true
		}

		if !hasPersist && slices.Contains([]string{
			"Save", "Set", "Upsert", "Insert", "Update", "Delete", "Remove",
			"Exec", "ExecContext", "Materialize", "Apply", "Persist", "Commit", "Put",
		}, method) {
			hasPersist = true
		}

		return !hasBroadcast || !hasPersist
	})

	return hasBroadcast, hasPersist
}
