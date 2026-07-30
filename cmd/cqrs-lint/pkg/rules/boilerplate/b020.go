package boilerplate

import (
	"context"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B020: Manual legacy field upcasting.
// Detects field-renaming or field-defaulting logic in a decode/unmarshal
// function that is NOT inside a schema.NewUpcaster callback. Manual upcasting
// is error-prone and should use schema.Upcaster / schema.VersionedStore.
//
//nolint:ireturn // factory returns public interface
func NewB020Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B020-manual-legacy-upcasting",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			// Collect position ranges of schema.NewUpcaster callback bodies so
			// we can skip functions that are inside them.
			type posRange struct {
				start, end token.Pos
			}

			var upcasterRanges []posRange

			for _, gf := range ctx.GoFiles {
				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					if sel.Sel.Name != "NewUpcaster" {
						return true
					}

					pkg := analyzer.SelectorPackage(sel)
					if pkg != "schema" {
						return true
					}

					upcasterRanges = append(upcasterRanges, posRange{
						start: call.Pos(),
						end:   call.End(),
					})

					return true
				})
			}

			isInsideUpcaster := func(pos token.Pos) bool {
				for _, r := range upcasterRanges {
					if pos >= r.start && pos <= r.end {
						return true
					}
				}

				return false
			}

			decodeNames := []string{"decode", "unmarshal", "adapt", "migrate", "fromlegacy", "fromrow"}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Name == nil || fn.Body == nil {
						continue
					}

					funcNameLower := strings.ToLower(fn.Name.Name)

					isDecodeFunc := false
					for _, prefix := range decodeNames {
						if strings.Contains(funcNameLower, prefix) {
							isDecodeFunc = true
							break
						}
					}

					if !isDecodeFunc {
						continue
					}

					// Skip if this function is inside a schema.NewUpcaster callback.
					if isInsideUpcaster(fn.Pos()) {
						continue
					}

					// Check if the function does field-level manipulation:
					// - calls Unmarshal/Decode
					// - AND has map index expressions with string literal keys
					//   (field renaming/defaulting pattern)
					hasUnmarshal := false
					hasFieldManipulation := false

					ast.Inspect(fn.Body, func(n ast.Node) bool {
						// Detect Unmarshal / Decode calls.
						if call, ok := n.(*ast.CallExpr); ok {
							sel, ok := analyzer.SelectorFromExpr(call.Fun)
							if ok {
								name := sel.Sel.Name
								if name == "Unmarshal" || name == "Decode" || name == "UnmarshalJSON" {
									hasUnmarshal = true
								}
							}

							return true
						}

						// Detect map index with string literal key (field manipulation).
						// Pattern: raw["fieldName"] = value  OR  if _, ok := raw["field"]; ok
						if idx, ok := n.(*ast.IndexExpr); ok {
							if _, ok := idx.Index.(*ast.BasicLit); ok {
								hasFieldManipulation = true
							}
						}

						return true
					})

					if !hasUnmarshal || !hasFieldManipulation {
						continue
					}

					pos := ctx.Fset.Position(fn.Pos())

					f, err := finding.NewBuilder(
						"B020", toolName,
						"Manual field upcasting in "+fn.Name.Name+
							" — use schema.NewUpcaster for versioned schema evolution",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Use schema.NewUpcaster(eventName, fromVersion, func(evt) (*ImmutableEvent, error) {...}) "+
							"with schema.VersionedStore for type-safe, versioned upcasting").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}
				}
			}

			return findings, nil
		},
	)
}
