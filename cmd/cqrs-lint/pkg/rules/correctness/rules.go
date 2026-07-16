// Package correctness implements bug-detection rules for go-cqrs-lite consumers.
package correctness

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

// C006: Manual Version Arithmetic.
// Detects event.Version(x.Int()+1) instead of x.Increment().
func NewC006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C006-manual-version-arithmetic",
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
					if !ok || pkgIdent.Name != "event" || sel.Sel.Name != "Version" {
						return true
					}

					if len(call.Args) != 1 {
						return true
					}

					binOp, ok := call.Args[0].(*ast.BinaryExpr)
					if !ok || binOp.Op != token.ADD {
						return true
					}

					leftCall, ok := binOp.X.(*ast.CallExpr)
					if !ok {
						return true
					}

					leftSel, ok := leftCall.Fun.(*ast.SelectorExpr)
					if !ok || leftSel.Sel.Name != "Int" {
						return true
					}

					rightLit, ok := binOp.Y.(*ast.BasicLit)
					if !ok || rightLit.Value != "1" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())
					versionVar := analyzer.ExprString(leftSel.X)
					oldExpr := fmt.Sprintf("event.Version(%s.Int()+1)", versionVar)
					newExpr := versionVar + ".Increment()"

					f, err := finding.NewBuilder(
						"C006", toolName,
						"Manual version arithmetic — use Version.Increment() instead",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategyDirect).
						WithSuggestion(fmt.Sprintf("Replace %s with %s", oldExpr, newExpr)).
						WithBeforeCode(oldExpr).
						WithAfterCode(newExpr).
						WithMetadata(map[string]string{
							"oldExpr": oldExpr,
							"newExpr": newExpr,
						}).
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

// C003: Silent Unknown Event in Fold.
// Detects fold functions whose switch default case returns nil error.
func NewC003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C003-silent-unknown-event-fold",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				if !fold.HasSwitch || !fold.HasDefault || !fold.DefaultNil {
					continue
				}

				f, err := finding.NewBuilder(
					"C003", toolName,
					fmt.Sprintf("Fold %s silently ignores unknown event types in default case", fold.FuncName),
					finding.SeverityError,
					finding.Pos(finding.FilePath(fold.File), fold.Pos.Line, fold.Pos.Column),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceHigh).
					WithFixStrategy(finding.FixStrategyDirect).
					WithSuggestion("Return an error in the default case: return state, fmt.Errorf(\"fold: unknown event type: %s\", evt.Type())").
					WithBeforeCode("return state, nil").
					WithAfterCode(`return state, fmt.Errorf("fold: unknown event type: %s", evt.Type())`).
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

// C002: Broken Command ID.
// Detects command ID() methods that return a zero-value composite literal.
func NewC002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C002-broken-command-id",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, cmd := range ctx.Registry.Commands {
				if !cmd.IDReturnsZero {
					continue
				}

				f, err := finding.NewBuilder(
					"C002", toolName,
					fmt.Sprintf("Command %s ID() returns zero value — breaks idempotency and tracing", cmd.Name),
					finding.SeverityCritical,
					finding.Pos(finding.FilePath(cmd.File), cmd.Pos.Line, cmd.Pos.Column),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceHigh).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Generate a unique CommandID per instance, or embed *command.BasicCommand which provides ID() automatically").
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

// C001: Missing Transaction Commit.
// Detects withTx-like helpers that call BeginTx but return nil instead of tx.Commit().
func NewC001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C001-missing-tx-commit",
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

					if !returnsError(fn) {
						continue
					}

					txVar := findBeginTxVar(fn)
					if txVar == "" {
						continue
					}

					if hasDeferCommit(fn, txVar) {
						continue
					}

					if hasCommitCall(fn, txVar) {
						continue
					}

					if !hasReturnNil(fn) {
						continue
					}

					pos := ctx.Fset.Position(fn.Pos())

					f, err := finding.NewBuilder(
						"C001",
						toolName,
						fmt.Sprintf(
							"Function %s calls BeginTx but never commits — data silently lost on success path",
							fn.Name.Name,
						),
						finding.SeverityCritical,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategyDirect).
						WithSuggestion(fmt.Sprintf("Change `return nil` to `return %s.Commit()`", txVar)).
						WithBeforeCode("return nil").
						WithAfterCode(fmt.Sprintf("return %s.Commit()", txVar)).
						WithMetadata(map[string]string{"txVar": txVar}).
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

// C005: Raw json.Unmarshal for Event Payload.
// Detects json.Unmarshal(evt.Payload(), ...) instead of event.DecodePayloadAuto[T](evt).
func NewC005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C005-raw-json-unmarshal-payload",
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
					if !ok || pkgIdent.Name != "json" {
						return true
					}

					if sel.Sel.Name != "Unmarshal" && sel.Sel.Name != "NewDecoder" {
						return true
					}

					var payloadArg ast.Expr
					if sel.Sel.Name == "Unmarshal" && len(call.Args) > 0 {
						payloadArg = call.Args[0]
					}

					if sel.Sel.Name == "NewDecoder" && len(call.Args) > 0 {
						payloadArg = call.Args[0]
					}

					if payloadArg == nil {
						return true
					}

					if !isPayloadCall(payloadArg) {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"C005", toolName,
						"Raw json.Unmarshal on event payload — use event.DecodePayloadAuto[T] instead",
						finding.SeverityError,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Use event.DecodePayloadAuto[YourPayload](evt) for automatic codec detection and schema versioning").
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

// C007: time.Now() in decider.
// Detects time.Now() calls inside decider decide functions.
func NewC007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C007-time-now-in-decider",
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

					if !isLikelyDecider(fn) {
						continue
					}

					ast.Inspect(fn.Body, func(n ast.Node) bool {
						call, ok := n.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						pkgIdent, ok := sel.X.(*ast.Ident)
						if !ok || pkgIdent.Name != "time" || sel.Sel.Name != "Now" {
							return true
						}

						pos := ctx.Fset.Position(call.Pos())

						f, err := finding.NewBuilder(
							"C007", toolName,
							"time.Now() inside decider — non-deterministic, makes testing impossible",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceMedium).
							WithSuggestion("Pass time as a parameter or inject a clock interface for deterministic testing").
							Build()
						if err != nil {
							return true
						}

						findings = append(findings, f)

						return true
					})
				}
			}

			return findings, nil
		},
	)
}

// C009: panic() in production code.
// Detects panic() calls in non-test, non-init files.
func NewC009Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C009-panic-in-production",
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

					ident, ok := call.Fun.(*ast.Ident)
					if !ok || ident.Name != "panic" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"C009", toolName,
						"panic() in production code — use error returns instead",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Return an error instead of panicking. Panics crash the process and bypass error handling middleware.").
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

// C010: Swallowed error in fold.
// Detects `_, _ = decode(evt)` or `_, := decode(evt); _ = err` in fold functions.
func NewC010Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C010-swallowed-error-in-fold",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				// We need to find the function in the AST to check for swallowed errors.
				for _, gf := range ctx.GoFiles {
					if gf.Path != fold.File || gf.IsTest {
						continue
					}

					ast.Inspect(gf.AST, func(n ast.Node) bool {
						fn, ok := n.(*ast.FuncDecl)
						if !ok || fn.Name == nil || fn.Name.Name != fold.FuncName {
							if strings.Contains(fold.FuncName, ".") {
								return true
							}

							return true
						}

						return inspectForSwallowedError(ctx, fn, &findings)
					})
				}
			}

			return findings, nil
		},
	)
}

// C008: float64 for money.
// Detects float64 fields with monetary names in struct types.
func NewC008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C008-float64-for-money",
		func(_ context.Context) ([]finding.Finding, error) {
			moneyFields := []string{
				"amount",
				"price",
				"cost",
				"balance",
				"total",
				"fee",
				"charge",
				"payment",
				"salary",
				"value",
			}

			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					st, ok := n.(*ast.StructType)
					if !ok || st.Fields == nil {
						return true
					}

					for _, field := range st.Fields.List {
						if !isFloat64(field.Type) {
							continue
						}

						for _, name := range field.Names {
							lowerName := strings.ToLower(name.Name)
							for _, mf := range moneyFields {
								if strings.Contains(lowerName, mf) {
									pos := ctx.Fset.Position(name.Pos())

									f, err := finding.NewBuilder(
										"C008", toolName,
										fmt.Sprintf("Field %s is float64 — use decimal or integer cents for money to avoid rounding errors", name.Name),
										finding.SeverityWarning,
										finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
									).
										WithCategory(finding.CategoryCorrectness).
										WithConfidence(finding.ConfidenceMedium).
										WithSuggestion("Use shopspring/decimal or int64 cents instead of float64 for monetary values").
										Build()
									if err != nil {
										return true
									}

									findings = append(findings, f)

									break
								}
							}
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// C012: Missing error return in withTx.
// Detects withTx-like functions that don't return the body's error.
func NewC012Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C012-missing-error-return-in-with-tx",
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

					if !returnsError(fn) {
						continue
					}

					txVar := findBeginTxVar(fn)
					if txVar == "" {
						continue
					}
					// Check if the function body calls body(tx) and ignores the error.
					bodyVar := findBodyParam(fn)
					if bodyVar == "" {
						continue
					}

					if !ignoresBodyError(fn, bodyVar) {
						continue
					}

					pos := ctx.Fset.Position(fn.Pos())

					f, err := finding.NewBuilder(
						"C012",
						toolName,
						fmt.Sprintf(
							"Function %s ignores error from body callback — failures silently lost",
							fn.Name.Name,
						),
						finding.SeverityCritical,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion(fmt.Sprintf("Check the error from %s(tx) and return it if non-nil", bodyVar)).
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

// --- Helper functions ---

func returnsError(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Results == nil {
		return false
	}

	for _, field := range fn.Type.Results.List {
		if id, ok := field.Type.(*ast.Ident); ok && id.Name == "error" {
			return true
		}
	}

	return false
}

func findBeginTxVar(fn *ast.FuncDecl) string {
	var txVar string

	ast.Inspect(fn, func(n ast.Node) bool {
		if txVar != "" {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name != "BeginTx" && sel.Sel.Name != "Begin" {
			return true
		}
		// Find the assignment that contains this call.
		if assignStmt := findContainingAssignStmt(fn, call); assignStmt != nil {
			if len(assignStmt.Lhs) > 0 {
				if id, ok := assignStmt.Lhs[0].(*ast.Ident); ok {
					txVar = id.Name
				}
			}
		}

		return true
	})

	return txVar
}

func findContainingAssignStmt(fn *ast.FuncDecl, target ast.Node) *ast.AssignStmt {
	var result *ast.AssignStmt

	ast.Inspect(fn, func(n ast.Node) bool {
		if result != nil {
			return false
		}

		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		for _, rhs := range assign.Rhs {
			if containsNode(rhs, target) {
				result = assign

				return false
			}
		}

		return true
	})

	return result
}

func containsNode(parent, target ast.Node) bool {
	if parent == target {
		return true
	}

	found := false

	ast.Inspect(parent, func(n ast.Node) bool {
		if n == target {
			found = true

			return false
		}

		return !found
	})

	return found
}

func hasCommitCall(fn *ast.FuncDecl, txVar string) bool {
	found := false

	ast.Inspect(fn, func(n ast.Node) bool {
		if found {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if id, ok := sel.X.(*ast.Ident); ok && id.Name == txVar && sel.Sel.Name == "Commit" {
			found = true
		}

		return true
	})

	return found
}

func hasDeferCommit(fn *ast.FuncDecl, txVar string) bool {
	found := false

	ast.Inspect(fn, func(n ast.Node) bool {
		if found {
			return false
		}

		deferStmt, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}

		call := deferStmt.Call
		if call == nil {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if id, ok := sel.X.(*ast.Ident); ok && id.Name == txVar && sel.Sel.Name == "Commit" {
			found = true
		}

		return true
	})

	return found
}

func hasReturnNil(fn *ast.FuncDecl) bool {
	found := false

	ast.Inspect(fn, func(n ast.Node) bool {
		if found {
			return false
		}

		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}

		for _, result := range ret.Results {
			if id, ok := result.(*ast.Ident); ok && id.Name == "nil" {
				found = true

				return false
			}
		}

		return true
	})

	return found
}

func isPayloadCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	return sel.Sel.Name == "Payload"
}

func isLikelyDecider(fn *ast.FuncDecl) bool {
	name := fn.Name.Name

	return name == "decide" || name == "Decide" ||
		strings.HasPrefix(name, "decide") ||
		strings.HasPrefix(name, "Decide") ||
		strings.Contains(name, "decide") ||
		strings.Contains(name, "Decide")
}

func isFloat64(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)

	return ok && id.Name == "float64"
}

func inspectForSwallowedError(
	ctx *analyzer.AnalysisContext,
	fn *ast.FuncDecl,
	findings *[]finding.Finding,
) bool {
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		// Check for patterns like: _, err := decode()  followed by no error check.
		// Or: _ = err
		if len(assign.Lhs) == 0 {
			return true
		}

		for _, lhs := range assign.Lhs {
			id, ok := lhs.(*ast.Ident)
			if !ok || id.Name != "_" {
				continue
			}
			// Check if the RHS is a function call that returns error.
			for _, rhs := range assign.Rhs {
				call, ok := rhs.(*ast.CallExpr)
				if !ok {
					continue
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					continue
				}

				callStr := analyzer.ExprString(call.Fun)
				if strings.Contains(callStr, "Decode") || strings.Contains(callStr, "Unmarshal") {
					pos := ctx.Fset.Position(assign.Pos())

					f, err := finding.NewBuilder(
						"C010", toolName,
						"Error from decode/unmarshal call is discarded — use the error or handle it",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Check the error return: `if err != nil { return state, fmt.Errorf(\"decode: %w\", err) }`").
						Build()
					if err == nil {
						*findings = append(*findings, f)
					}
				}

				_ = sel
			}
		}

		return true
	})

	return true
}

func findBodyParam(fn *ast.FuncDecl) string {
	if fn.Type == nil || fn.Type.Params == nil {
		return ""
	}

	for _, param := range fn.Type.Params.List {
		if ft, ok := param.Type.(*ast.FuncType); ok {
			if ft.Params != nil && len(ft.Params.List) > 0 {
				if len(param.Names) > 0 {
					return param.Names[0].Name
				}
			}
		}
	}

	return ""
}

func ignoresBodyError(fn *ast.FuncDecl, bodyVar string) bool {
	// Check if body is called without checking the error.
	calledWithoutCheck := false

	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if id, ok := call.Fun.(*ast.Ident); ok && id.Name == bodyVar {
			// Check if the result is checked.
			assign := findContainingAssignStmt(fn, call)
			if assign == nil {
				calledWithoutCheck = true

				return false
			}
			// Check if error is ignored.
			for _, lhs := range assign.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && (id.Name == "_" || id.Name == "") {
					calledWithoutCheck = true
				}
			}
		}

		return true
	})

	return calledWithoutCheck
}
