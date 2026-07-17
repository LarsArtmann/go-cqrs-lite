package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects time.Time fields in event payload structs, which lose timezone
// information through CBOR's epoch encoding. Suggests using event.Instant
// (for instants) or event.WallTime (for wall-clock times) instead.
//
// Heuristic: flags time.Time or *time.Time fields in structs whose names
// suggest they are event payloads (ending in Event, Payload, or EventData),
// or in structs defined in files named events.go/payloads.go.
func NewC013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C013-time-time-in-event-payload",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					genDecl, ok := decl.(*ast.GenDecl)
					if !ok {
						continue
					}

					for _, spec := range genDecl.Specs {
						typeSpec, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}

						structType, ok := typeSpec.Type.(*ast.StructType)
						if !ok {
							continue
						}

						structName := typeSpec.Name.Name
						if !looksLikeEventPayload(structName, gf.Path) {
							continue
						}

						for _, field := range structType.Fields.List {
							if isTimeType(field.Type) && !hasAllowPragma(ctx, gf.Path, field) {
								pos := ctx.Fset.Position(field.Pos())
								fieldName := getFieldNames(field)

								f, err := finding.NewBuilder(
									"C013",
									toolName,
									fmt.Sprintf(
										"Struct %s has %s of type time.Time — timezone info is lost via CBOR epoch encoding",
										structName,
										fieldName,
									),
									finding.SeverityWarning,
									finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
								).
									WithCategory(finding.CategoryCorrectness).
									WithConfidence(finding.ConfidenceMedium).
									WithSuggestion(
										"Use event.Instant for unique moments (created_at, occurred_at) " +
											"or event.WallTime for local times (schedules, reminders). " +
											"See docs/TIMEZONE_HANDLING.md for guidance.",
									).
									WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
									Build()
								if err != nil {
									continue
								}

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

// looksLikeEventPayload checks if a struct name or file path suggests
// the struct is an event payload.
func looksLikeEventPayload(structName, filePath string) bool {
	upper := strings.ToUpper(structName)

	for _, suffix := range []string{"EVENT", "PAYLOAD", "EVENTDATA"} {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}

	// Also match files named events.go or payloads.go
	base := filePath
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}

	base = strings.TrimSuffix(base, ".go")

	return base == "events" || base == "payloads"
}

// isTimeType checks if an AST type expression is time.Time or *time.Time.
func isTimeType(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return isTimeType(t.X)
	case *ast.SelectorExpr:
		ident, ok := t.X.(*ast.Ident)
		if !ok {
			return false
		}

		return ident.Name == "time" && t.Sel.Name == "Time"
	default:
		return false
	}
}

// getFieldNames returns a human-readable description of field names.
func getFieldNames(field *ast.Field) string {
	if len(field.Names) == 0 {
		return "anonymous field"
	}

	names := make([]string, len(field.Names))
	for i, name := range field.Names {
		names[i] = name.Name
	}

	return strings.Join(names, ", ")
}

// hasAllowPragma checks if the field has a //cqrs-lint:allow-time-time
// comment in its doc or line comment.
func hasAllowPragma(ctx *analyzer.AnalysisContext, filePath string, field *ast.Field) bool {
	// Check doc comment (comment group above the field)
	if field.Doc != nil {
		for _, comment := range field.Doc.List {
			if strings.Contains(comment.Text, "cqrs-lint:allow-time-time") {
				return true
			}
		}
	}

	// Check inline comment (comment on the same line)
	if field.Comment != nil {
		for _, comment := range field.Comment.List {
			if strings.Contains(comment.Text, "cqrs-lint:allow-time-time") {
				return true
			}
		}
	}

	// Also check the source line itself and the line before
	pos := ctx.Fset.Position(field.Pos())
	if line := ctx.SourceLine(
		filePath,
		pos.Line,
	); strings.Contains(
		line,
		"cqrs-lint:allow-time-time",
	) {
		return true
	}

	if pos.Line > 1 {
		if line := ctx.SourceLine(
			filePath,
			pos.Line-1,
		); strings.Contains(
			line,
			"cqrs-lint:allow-time-time",
		) {
			return true
		}
	}

	return false
}
