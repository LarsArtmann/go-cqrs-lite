package boilerplate

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B006: Duplicate foreign-key stub SQL.
// Detects repeated string literals containing SQL REFERENCES clauses
// that are copy-pasted instead of centralized.
//
//nolint:ireturn // factory returns public interface
func NewB006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B006-duplicate-fk-stub-sql",
		func(_ context.Context) ([]finding.Finding, error) {
			fkCounts := make(map[string]int)
			fkPositions := make(map[string]token.Position)

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					lit, ok := n.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						return true
					}

					val := strings.ToUpper(strings.Trim(lit.Value, "\"`'"))
					if !strings.Contains(val, "REFERENCES") {
						return true
					}

					fkCounts[val]++
					if fkCounts[val] == 1 {
						fkPositions[val] = ctx.Fset.Position(lit.Pos())
					}

					return true
				})
			}

			var findings []finding.Finding

			for sql, count := range fkCounts {
				if count < 2 {
					continue
				}

				pos := fkPositions[sql]

				preview := sql
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}

				f, err := finding.NewBuilder(
					"B006",
					toolName,
					fmt.Sprintf(
						"Foreign-key SQL duplicated %d times: %q — centralize as a shared constant",
						count,
						preview,
					),
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion("Extract FK SQL into a shared constant or migration helper").
					WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
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

// B007: Repeated handler registration.
// Detects 3+ consecutive Register/RegisterTyped calls that could be table-driven.
//
// Scope: this rule covers CQRS handler registration (Register, RegisterTyped,
// Handle, Subscribe) ONLY. It does NOT fire on stdlib http.ServeMux.HandleFunc
// chains — Go 1.22+ pattern routing with per-route middleware is idiomatic and
// not a boilerplate smell. See the DiscordSync feedback (B007 was a non-issue).
//
// To avoid false positives on third-party frameworks whose API collides with
// CQRS naming (e.g. huma.Register, grpc.Server.Register), a denylist of
// non-CQRS package qualifiers is consulted. Registration calls qualified by a
// denylisted package are not counted. Variable qualifiers (d.Register,
// cmdDisp.Register) are always counted — they are the idiomatic CQRS pattern.
// See the browser-history feedback (B007 fired on 12 huma.Register calls).
var nonCQRSRegisterPackages = map[string]bool{
	"huma":  true, // Huma v2 HTTP framework: huma.Register[I,O,Body]
	"http":  true, // net/http
	"mux":   true, // gorilla/mux
	"chi":   true, // go-chi/chi
	"gin":   true, // gin-gonic/gin
	"echo":  true, // labstack/echo
	"fiber": true, // gofiber/fiber
	"grpc":  true, // grpc-go Server.Register (proto service registration)
}

//nolint:ireturn // factory returns public interface
func NewB007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B007-repeated-handler-registration",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					fn, ok := n.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						return true
					}

					registerCount := 0

					var firstPos token.Position

					for _, stmt := range fn.Body.List {
						exprStmt, ok := stmt.(*ast.ExprStmt)
						if !ok {
							if registerCount >= 3 {
								reportRepeatedRegistration(
									ctx,
									&findings,
									fn.Name.Name,
									registerCount,
									firstPos,
								)
							}

							registerCount = 0

							continue
						}

						call, ok := exprStmt.X.(*ast.CallExpr)
						if !ok {
							continue
						}

						sel, ok := analyzer.SelectorFromExpr(call.Fun)
						if !ok {
							continue
						}

						if sel.Sel.Name == "Register" || sel.Sel.Name == "RegisterTyped" ||
							sel.Sel.Name == "Handle" || sel.Sel.Name == "Subscribe" {
							// Skip third-party Register APIs (huma, grpc, etc.) whose
							// method name collides with CQRS but serves a different
							// purpose. Variable qualifiers (d, cmdDisp) are never
							// denylisted — they are the idiomatic CQRS pattern.
							if nonCQRSRegisterPackages[analyzer.SelectorPackage(sel)] {
								continue
							}

							if registerCount == 0 {
								firstPos = ctx.Fset.Position(call.Pos())
							}

							registerCount++
						}
					}

					if registerCount >= 3 {
						reportRepeatedRegistration(
							ctx,
							&findings,
							fn.Name.Name,
							registerCount,
							firstPos,
						)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

func reportRepeatedRegistration(
	ctx *analyzer.AnalysisContext,
	findings *[]finding.Finding,
	funcName string,
	count int,
	pos token.Position,
) {
	f, err := finding.NewBuilder(
		"B007",
		toolName,
		fmt.Sprintf(
			"%d consecutive handler registrations in %s — use a table-driven or variadic approach",
			count,
			funcName,
		),
		finding.SeverityInfo,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryBestPractice).
		WithConfidence(finding.ConfidenceHigh).
		WithSuggestion("Collect handlers into a slice and register them in a loop, or use a variadic helper").
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	if err != nil {
		return
	}

	*findings = append(*findings, f)
}
