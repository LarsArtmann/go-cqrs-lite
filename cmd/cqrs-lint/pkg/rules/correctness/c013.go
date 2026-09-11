package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// Detects time.Time fields in event payload structs, which lose timezone
// information through CBOR's epoch encoding. Suggests using event.Instant
// (for instants) or event.WallTime (for wall-clock times) instead.
//
// Heuristic: flags time.Time or *time.Time fields in structs whose names
// suggest they are event payloads (ending in Event, Payload, or EventData),
// or in structs defined in files named events.go/payloads.go.
//
// Also detects time.Time fields in anonymous nested structs within event payloads.
//
//nolint:ireturn // factory returns public interface
func NewC013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C013-time-time-in-event-payload",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			typedTier := ctx.TypedConfirmations()
			var ev *typedEvidence
			if typedTier {
				ev = buildTypedEvidence(ctx)
			}

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
						checkC013Struct(ctx, typedTier, ev, gf, structName, structType, &findings)
					}
				}
			}

			return findings, nil
		},
	)
}

// checkC013Struct runs the payload/view field checks for one classified
// struct, gating file-name-only candidates behind the F091 Tier-3 typed
// confirmation: an ambient payload candidate needs payload evidence (a real
// event.New payload flow or a Type() string method); an ambient view candidate
// needs a serialization point (any json tag). Strong name-suffix candidates
// and the typed-off fallback run the historical heuristic unchanged.
func checkC013Struct(
	ctx *analyzer.AnalysisContext,
	typedTier bool,
	ev *typedEvidence,
	gf *analyzer.GoFile,
	structName string,
	st *ast.StructType,
	findings *[]finding.Finding,
) {
	switch classifyC013Candidate(structName, gf.Path) {
	case c013PayloadByName:
		checkStructFields(ctx, gf, structName, st.Fields, findings)
	case c013PayloadByFile:
		if !typedTier || ev.payloadConfirmed(structName) {
			checkStructFields(ctx, gf, structName, st.Fields, findings)
		}
	case c013ViewByName:
		checkProjectionTimeFields(ctx, gf, structName, st.Fields, findings)
	case c013ViewByFile:
		if !typedTier || viewSerialized(st) {
			checkProjectionTimeFields(ctx, gf, structName, st.Fields, findings)
		}
	default:
	}
}

// checkStructFields checks all fields in a struct for time.Time usage,
// recursing into anonymous nested structs.
func checkStructFields(
	ctx *analyzer.AnalysisContext,
	gf *analyzer.GoFile,
	structName string,
	fields *ast.FieldList,
	findings *[]finding.Finding,
) {
	if fields == nil {
		return
	}

	for _, field := range fields.List {
		if isTimeType(field.Type) && !hasAllowPragma(ctx, gf.Path, field) &&
			!hasJSONDashTag(field) {
			reportTimeField(ctx, gf, structName, field, findings)
		}

		if nested, ok := field.Type.(*ast.StructType); ok {
			nestedName := structName + "."
			if len(field.Names) > 0 {
				nestedName += field.Names[0].Name
			} else {
				nestedName += "anonymous"
			}
			checkStructFields(ctx, gf, nestedName, nested.Fields, findings)
		}
	}
}

// reportTimeField creates a finding for a time.Time field with a specific suggestion.
func reportTimeField(
	ctx *analyzer.AnalysisContext,
	_ *analyzer.GoFile,
	structName string,
	field *ast.Field,
	findings *[]finding.Finding,
) {
	pos := ctx.Fset.Position(field.Pos())
	fieldName := getFieldNames(field)
	suggestion := suggestReplacement(fieldName)

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
		WithSuggestion(suggestion).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	lintutil.AppendBuild(findings, f, err)
}

// suggestReplacement returns a specific replacement recommendation
// based on the field name semantics.
func suggestReplacement(fieldName string) string {
	lower := strings.ToLower(fieldName)

	if strings.Contains(lower, "schedule") ||
		strings.Contains(lower, "reminder") ||
		strings.Contains(lower, "business_hour") ||
		strings.Contains(lower, "meeting") ||
		strings.Contains(lower, "alarm") {
		return "Use event.WallTime for this local time-of-day field. " +
			"Example: WallTime{Hour: 9, Minute: 0, Location: \"America/New_York\"}. " +
			"See docs/TIMEZONE_HANDLING.md."
	}

	timestampFields := []string{
		"created", "updated", "occurred", "deleted", "expires", "expired",
		"timestamp", "at_time", "processed", "completed", "started",
		"ended", "locked", "unlocked", "revoked", "sent", "received",
		"discovered", "last_login", "submitted", "accepted", "rejected",
	}

	for _, tf := range timestampFields {
		if strings.Contains(lower, tf) {
			return fmt.Sprintf(
				"Use event.Instant for this timestamp field. "+
					"Example: replace `%s time.Time` with `%s event.Instant`, "+
					"and use event.NewInstant(time.Now()) at construction. "+
					"See docs/TIMEZONE_HANDLING.md.",
				fieldName, fieldName,
			)
		}
	}

	return "Use event.Instant for unique moments (created_at, occurred_at) " +
		"or event.WallTime for local times (schedules, reminders). " +
		"For calendar dates (birth dates, employment dates), use event.Date. " +
		"See docs/TIMEZONE_HANDLING.md for guidance."
}

// checkProjectionTimeFields checks for time.Time fields in projection view
// structs. Unlike event payloads (which lose timezone via CBOR), projection
// views lose timezone when stored in SQL columns without timezone info
// (TIMESTAMP without TZ). The message and suggestion are tailored to this.
func checkProjectionTimeFields(
	ctx *analyzer.AnalysisContext,
	gf *analyzer.GoFile,
	structName string,
	fields *ast.FieldList,
	findings *[]finding.Finding,
) {
	if fields == nil {
		return
	}

	for _, field := range fields.List {
		if isTimeType(field.Type) && !hasAllowPragma(ctx, gf.Path, field) &&
			!hasJSONDashTag(field) {
			pos := ctx.Fset.Position(field.Pos())
			fieldName := getFieldNames(field)

			f, err := finding.NewBuilder(
				"C013", toolName,
				fmt.Sprintf(
					"Projection view %s has %s of type time.Time — timezone may be lost in SQL storage",
					structName, fieldName,
				),
				finding.SeverityWarning,
				finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
			).
				WithCategory(finding.CategoryCorrectness).
				WithConfidence(finding.ConfidenceMedium).
				WithSuggestion("Store timestamps as UTC ISO-8601 strings (timestamptz) or use event.Instant to preserve timezone info across SQL round-trips").
				WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
				Build()
			lintutil.AppendBuild(findings, f, err)
		}
	}
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
	if field.Doc != nil {
		for _, comment := range field.Doc.List {
			if strings.Contains(comment.Text, "cqrs-lint:allow-time-time") {
				return true
			}
		}
	}

	if field.Comment != nil {
		for _, comment := range field.Comment.List {
			if strings.Contains(comment.Text, "cqrs-lint:allow-time-time") {
				return true
			}
		}
	}

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

// hasJSONDashTag reports whether the field has a `json:"-"` tag, meaning it is
// explicitly excluded from serialization and should not be flagged for timezone
// concerns.
func hasJSONDashTag(field *ast.Field) bool {
	if field.Tag == nil {
		return false
	}

	return strings.Contains(field.Tag.Value, `json:"-"`)
}
