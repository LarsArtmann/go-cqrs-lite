// Package api implements API-misuse detection rules.
package api

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

const toolName finding.ToolName = "cqrs-lint"

// A001: Manual command interface.
// Detects command structs with manual Type()/AggregateID()/ID() methods instead of embedding *command.BasicCommand.
func NewA001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A001-manual-command-interface",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, cmd := range ctx.Registry.Commands {
				if cmd.HasBasicCmd {
					continue
				}

				manualCount := 0
				if cmd.ManualID {
					manualCount++
				}
				// Check for manual Type() and AggregateID() methods.
				if hasMethod(ctx, cmd, "Type") {
					manualCount++
				}

				if hasMethod(ctx, cmd, "AggregateID") {
					manualCount++
				}

				if manualCount >= 2 {
					f, err := finding.NewBuilder(
						"A001", toolName,
						fmt.Sprintf("Command %s manually implements Type()/ID()/AggregateID() — embed *command.BasicCommand instead", cmd.Name),
						finding.SeverityError,
						finding.Pos(finding.FilePath(cmd.File), cmd.Pos.Line, cmd.Pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Embed *command.BasicCommand to get Type(), ID(), and AggregateID() for free, constructed via command.New(type, aggregateID)").
						Build()
					if err != nil {
						continue
					}

					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// A002: event.NewEvent with json.Marshal argument.
// Detects event.NewEvent(type, id, aggType, ver, json.Marshal(payload)) instead of event.New.
func NewA002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A002-newevent-manual-marshal",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "event" || sel.Sel.Name != "NewEvent" {
						return true
					}
					// Check if the 5th argument (payload) is a json.Marshal call.
					if len(call.Args) < 5 {
						return true
					}

					payloadArg := call.Args[4]

					argCall, ok := payloadArg.(*ast.CallExpr)
					if !ok {
						return true
					}

					argSel, ok := argCall.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					argPkg, ok := argSel.X.(*ast.Ident)
					if !ok || argPkg.Name != "json" || argSel.Sel.Name != "Marshal" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"A002", toolName,
						"event.NewEvent with json.Marshal payload — use event.New which auto-marshals typed payloads",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Replace event.NewEvent(type, id, aggType, ver, json.Marshal(payload)) with event.New(type, id, aggType, ver, payload)").
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

// A003: Explicit codec in decode.
// Detects event.DecodePayload[T](evt, codec.JSONCodec{}) — should use DecodePayloadAuto.
func NewA003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A003-explicit-codec-in-decode",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "event" {
						return true
					}

					if sel.Sel.Name != "DecodePayload" {
						return true
					}
					// Check if there are more than 1 argument (the event) — extra args are codec.
					if len(call.Args) <= 1 {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"A003", toolName,
						"Explicit codec passed to DecodePayload — use DecodePayloadAuto[T] for automatic codec detection",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceMedium).
						WithSuggestion("Use event.DecodePayloadAuto[T](evt) — it auto-detects JSON/CBOR from the event's encoding stamp").
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

// A004: Untyped dispatch register.
// Detects dispatcher.Register with type assertion inside the handler.
func NewA004Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A004-untyped-dispatch-register",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					if sel.Sel.Name != "Register" && sel.Sel.Name != "Handle" {
						return true
					}
					// Check if the handler function literal contains a type assertion.
					for _, arg := range call.Args {
						funcLit, ok := arg.(*ast.FuncLit)
						if !ok {
							continue
						}

						hasTypeAssert := false

						ast.Inspect(funcLit.Body, func(nn ast.Node) bool {
							_, ok := nn.(*ast.TypeAssertExpr)
							if ok {
								hasTypeAssert = true

								return false
							}

							return true
						})

						if hasTypeAssert {
							pos := ctx.Fset.Position(call.Pos())

							f, err := finding.NewBuilder(
								"A004", toolName,
								"Untyped handler registration with type assertion — use RegisterTyped for compile-time type safety",
								finding.SeverityWarning,
								finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
							).
								WithCategory(finding.CategoryBestPractice).
								WithConfidence(finding.ConfidenceMedium).
								WithSuggestion("Use command.RegisterTyped or query.RegisterTyped to register typed handlers without runtime type assertions").
								Build()
							if err != nil {
								return true
							}

							findings = append(findings, f)
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// A007: Dual model (OO aggregate + functional decider).
// Detects projects that use both OO-style aggregates and functional deciders.
func NewA007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A007-dual-model-oo-functional",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			hasOO := false
			hasFunctional := false

			var firstOO analyzer.DeciderInfo

			for _, d := range ctx.Registry.Deciders {
				if d.IsOO {
					hasOO = true

					if firstOO.File == "" {
						firstOO = d
					}
				}
			}
			// Check for functional decider usage (decider.Decider[State]{...}).
			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					lit, ok := n.(*ast.CompositeLit)
					if !ok {
						return true
					}

					typeStr := analyzer.ExprString(lit.Type)
					if strings.Contains(typeStr, "decider.Decider") ||
						strings.Contains(typeStr, "Decider[") {
						hasFunctional = true
					}

					return true
				})
			}

			if hasOO && hasFunctional {
				f, err := finding.NewBuilder(
					"A007", toolName,
					"Project uses both OO-style aggregates and functional deciders — pick one model for consistency",
					finding.SeverityError,
					finding.Pos(finding.FilePath(firstOO.File), firstOO.Pos.Line, firstOO.Pos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion("Use the functional decider.Decider[State] pattern (Initial + Apply) consistently — it's the recommended approach").
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// A005: Custom projection runner.
// Detects bus.SubscribeAll + manual switch without projectionhost.
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

					sel, ok := call.Fun.(*ast.SelectorExpr)
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
						f, err := finding.NewBuilder(
							"A005", toolName,
							"Manual projection via bus.SubscribeAll — use projectionhost.Host for checkpoint persistence, dead-letter queues, and crash recovery",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(subscribePos.Filename), subscribePos.Line, subscribePos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceMedium).
							WithSuggestion("Register projections with projectionhost.New(journal, checkpointStore) instead of manual bus.SubscribeAll + switch").
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

// A006: Adapter layer wrapping.
// Detects WrapEvent/UnwrapEvent/ToEvent adapter methods that add unnecessary indirection.
func NewA006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A006-adapter-layer-wrapping",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			adapterNames := []string{
				"WrapEvent",
				"UnwrapEvent",
				"ToEvent",
				"FromEvent",
				"ToCQRSEvent",
				"FromCQRSEvent",
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Name == nil {
						continue
					}

					for _, adapter := range adapterNames {
						if fn.Name.Name == adapter {
							pos := ctx.Fset.Position(fn.Pos())

							f, err := finding.NewBuilder(
								"A006", toolName,
								fmt.Sprintf("Adapter method %s — go-cqrs-lite events are concrete types, no wrapping needed", adapter),
								finding.SeverityInfo,
								finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
							).
								WithCategory(finding.CategoryBestPractice).
								WithConfidence(finding.ConfidenceLow).
								WithSuggestion("Work with event.Event directly — it's a type alias for *ImmutableEvent, no adapter layer needed").
								Build()
							if err == nil {
								findings = append(findings, f)
							}
						}
					}
				}
			}

			return findings, nil
		},
	)
}

// A008: Parallel type system.
// Detects custom AggregateID/Version/CommandType types duplicating go-cqrs-lite.
func NewA008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A008-parallel-type-system",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			duplicateTypes := map[string]bool{
				"AggregateID": true,
				"CommandType": true,
				"EventType":   true,
				"Version":     true,
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					gd, ok := decl.(*ast.GenDecl)
					if !ok || gd.Tok != token.TYPE {
						continue
					}

					for _, spec := range gd.Specs {
						ts, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}

						if duplicateTypes[ts.Name.Name] {
							// Check if this is in a types package (not the actual event package).
							if strings.Contains(gf.Path, "/event/") || strings.HasPrefix(gf.Path, "event/") ||
								strings.Contains(gf.Path, "/command/") ||
								strings.HasPrefix(gf.Path, "command/") {
								continue
							}

							pos := ctx.Fset.Position(ts.Pos())

							f, err := finding.NewBuilder(
								"A008", toolName,
								fmt.Sprintf("Custom type %s duplicates go-cqrs-lite's type system — use id.%s or event.%s instead", ts.Name.Name, ts.Name.Name, ts.Name.Name),
								finding.SeverityError,
								finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
							).
								WithCategory(finding.CategoryBestPractice).
								WithConfidence(finding.ConfidenceHigh).
								WithSuggestion(fmt.Sprintf("Replace custom %s with the library's type from id/ or event/ package", ts.Name.Name)).
								Build()
							if err == nil {
								findings = append(findings, f)
							}
						}
					}
				}
			}

			return findings, nil
		},
	)
}

// hasMethod checks if a command type has a method with the given name.
func hasMethod(ctx *analyzer.AnalysisContext, cmd analyzer.CommandInfo, methodName string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.Path != cmd.File || gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil {
				continue
			}

			if fn.Name.Name != methodName {
				continue
			}

			recvType := baseTypeName(fn.Recv.List[0].Type)
			if recvType == cmd.Name {
				return true
			}
		}
	}

	return false
}

func baseTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return baseTypeName(t.X)
	case *ast.IndexExpr:
		return baseTypeName(t.X)
	}

	return ""
}
