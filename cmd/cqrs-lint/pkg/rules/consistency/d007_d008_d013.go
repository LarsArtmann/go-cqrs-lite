package consistency

import (
	"context"
	"fmt"
	"go/ast"
	"path/filepath"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// D007: Inconsistent event creation API.
// Detects projects that use both event.New and event.NewEvent. Standardizing
// on event.New (the shorter alias) reduces cognitive load and prevents the
// split-brain pattern where different files use different constructors.
//
//nolint:ireturn // factory returns public interface
func NewD007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D007-inconsistent-event-creation-api",
		func(_ context.Context) ([]finding.Finding, error) {
			hasNew := false
			hasNewEvent := false
			firstFile := ""
			firstLine := 0

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					pkg, name, ok := selectorPkgAndName(call.Fun)
					if !ok || !isEventPkg(gf.AST, pkg) {
						return true
					}

					switch name {
					case "New":
						hasNew = true
						if firstFile == "" {
							pos := ctx.Fset.Position(call.Pos())
							firstFile = pos.Filename
							firstLine = pos.Line
						}
					case "NewEvent":
						hasNewEvent = true
						if firstFile == "" {
							pos := ctx.Fset.Position(call.Pos())
							firstFile = pos.Filename
							firstLine = pos.Line
						}
					}

					return true
				})
			}

			if !hasNew || !hasNewEvent {
				return nil, nil
			}

			pos := anchorPos(ctx, firstFile, firstLine)

			f, err := finding.NewBuilder(
				"D007", toolName,
				"Project uses both event.New and event.NewEvent — standardize on event.New",
				finding.SeverityInfo,
				pos,
			).
				WithCategory(finding.CategoryStyle).
				WithConfidence(finding.ConfidenceMedium).
				WithSuggestion("Replace all event.NewEvent calls with event.New — they are aliases").
				WithSnippet(ctx.SourceLine(firstFile, firstLine)).
				Build()
			if err != nil {
				return nil, nil //nolint:nilerr // best-effort: drop malformed finding
			}

			return []finding.Finding{f}, nil
		},
	)
}

// D008: Inconsistent codec usage.
// Detects projects mixing event.DecodePayload (explicit codec) with
// event.DecodePayloadAuto (auto-detect). Mixing them is inconsistent: pick
// one decode strategy per project.
//
//nolint:ireturn // factory returns public interface
func NewD008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D008-inconsistent-codec-usage",
		func(_ context.Context) ([]finding.Finding, error) {
			hasExplicit := false
			hasAuto := false
			firstFile := ""
			firstLine := 0

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					pkg, name, ok := selectorPkgAndName(call.Fun)
					if !ok || !isEventPkg(gf.AST, pkg) {
						return true
					}

					switch name {
					case "DecodePayload":
						hasExplicit = true
						if firstFile == "" {
							pos := ctx.Fset.Position(call.Pos())
							firstFile = pos.Filename
							firstLine = pos.Line
						}
					case "DecodePayloadAuto":
						hasAuto = true
						if firstFile == "" {
							pos := ctx.Fset.Position(call.Pos())
							firstFile = pos.Filename
							firstLine = pos.Line
						}
					}

					return true
				})
			}

			if !hasExplicit || !hasAuto {
				return nil, nil
			}

			pos := anchorPos(ctx, firstFile, firstLine)

			f, err := finding.NewBuilder(
				"D008",
				toolName,
				"Project mixes event.DecodePayload (explicit codec) and event.DecodePayloadAuto — pick one decode strategy",
				finding.SeverityInfo,
				pos,
			).
				WithCategory(finding.CategoryStyle).
				WithConfidence(finding.ConfidenceMedium).
				WithSuggestion(
					"Standardize on event.DecodePayloadAuto (auto-detects codec from event stamp) " +
						"unless you need explicit codec control in a specific path",
				).
				WithSnippet(ctx.SourceLine(firstFile, firstLine)).
				Build()
			if err != nil {
				return nil, nil //nolint:nilerr // best-effort: drop malformed finding
			}

			return []finding.Finding{f}, nil
		},
	)
}

// D013: Schema version not stamped on events.
// Detects projects that create events (event.New/event.NewEvent) without ever
// using event.WithSchemaVersion. Without schema versioning, upcasting is
// impossible to implement retroactively. This is a coaching rule — it fires
// once per project when there are event creation calls but zero schema-version
// options used.
//
//nolint:ireturn // factory returns public interface
func NewD013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D013-missing-schema-version",
		func(_ context.Context) ([]finding.Finding, error) {
			eventCreateCount := 0
			hasSchemaVersion := false
			firstFile := ""
			firstLine := 0

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					pkg, name, ok := selectorPkgAndName(call.Fun)
					if !ok {
						return true
					}

					if isEventPkg(gf.AST, pkg) && (name == "New" || name == "NewEvent") {
						eventCreateCount++
						if firstFile == "" {
							pos := ctx.Fset.Position(call.Pos())
							firstFile = pos.Filename
							firstLine = pos.Line
						}
					}

					if isEventPkg(gf.AST, pkg) && name == "WithSchemaVersion" {
						hasSchemaVersion = true
					}

					return true
				})
			}

			if eventCreateCount == 0 || hasSchemaVersion {
				return nil, nil
			}

			pos := anchorPos(ctx, firstFile, firstLine)

			f, err := finding.NewBuilder(
				"D013", toolName,
				fmt.Sprintf(
					"Project creates %d events without event.WithSchemaVersion — schema evolution (upcasting) is impossible to add retroactively",
					eventCreateCount,
				),
				finding.SeverityInfo,
				pos,
			).
				WithCategory(finding.CategoryStyle).
				WithConfidence(finding.ConfidenceLow).
				WithSuggestion(
					"Add event.WithSchemaVersion(1) to new event constructors so future schema " +
						"changes can use upcasters without breaking stored events",
				).
				WithSnippet(ctx.SourceLine(firstFile, firstLine)).
				Build()
			if err != nil {
				return nil, nil //nolint:nilerr // best-effort: drop malformed finding
			}

			return []finding.Finding{f}, nil
		},
	)
}

// selectorPkgAndName extracts (pkgName, methodName) from a selector expression
// like event.New or fmt.Errorf. Returns ok=false if the expression is not a
// pkgName.method selector.
func selectorPkgAndName(expr ast.Expr) (pkg, name string, ok bool) {
	sel, ok := analyzer.SelectorFromExpr(expr)
	if !ok {
		return "", "", false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", "", false
	}

	return ident.Name, sel.Sel.Name, true
}

// isEventPkg checks whether a qualifier (e.g. the "event" in event.New) refers
// to the go-cqrs-lite/event module, accounting for import aliases. The raw-name
// fast path handles the common case without file scanning.
func isEventPkg(file *ast.File, qualifier string) bool {
	if qualifier == "event" {
		return true
	}

	return lintutil.QualifierResolvesTo(file, qualifier, "go-cqrs-lite/event")
}

// isErrorFamilyPkg checks whether a qualifier refers to the go-error-family
// module, accounting for import aliases.
func isErrorFamilyPkg(file *ast.File, qualifier string) bool {
	if qualifier == "errorfamily" {
		return true
	}

	return lintutil.QualifierResolvesTo(file, qualifier, "go-error-family")
}

// anchorPos returns a finding.Position anchored at the given file/line, or
// falls back to the project's go.mod position when no specific call site
// was captured (project-level findings with no matching call).
func anchorPos(ctx *analyzer.AnalysisContext, file string, line int) finding.Position {
	if file != "" {
		return finding.Pos(finding.FilePath(file), line, 1)
	}

	return finding.Pos(
		finding.FilePath(filepath.Join(ctx.ProjectRoot, "go.mod")),
		1,
		1,
	)
}
