package correctness

import (
	"context"
	"go/ast"
	"go/types"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// C044: branded-ID .String() outside display contexts. `cbid.ID.String()`
// (and the id-package display overrides) render the PREFIXED display form
// ("StreamMarker:<raw>"); the wire/storage form is the bare value (.Get(),
// MarshalText, driver.Valuer). Every display-form leak shipped as a bug was
// a `.String()` call on a branded receiver in a storage, key, cursor, or
// SQL-argument position (scheduling/sqlstore queries, system typed-decoder
// lookups, the getting-started example, the meta_stream_log stream-key path,
// PG snapshot binds — 2026-08..10). This rule flags those calls WARN-first:
// the display intent is legitimate in logging and message-formatting
// positions, so those are whitelisted; hard severity waits for v5.
//
//nolint:ireturn // factory returns public interface
func NewC044Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C044-branded-id-display-string",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if gf.Pkg == nil || gf.Pkg.TypesInfo == nil {
					continue
				}

				findings = append(findings, detectBrandedStringCalls(ctx, gf)...)
			}

			return findings, nil
		},
	)
}

// displayContextCalleeNames are the enclosing-call method names under which
// a branded .String() is display intent (logging, structured attributes,
// message formatting) rather than a wire/storage leak.
var displayContextCalleeNames = map[string]bool{
	// log/slog family
	"Info": true, "Infof": true, "Warn": true, "Warnf": true,
	"Error": true, "Errorf": true, "Debug": true, "Debugf": true,
	"Log": true, "Logf": true,
	// stdlib log family
	"Printf": true, "Print": true, "Println": true, "Fatalf": true, "Panicf": true,
	// fmt display formatting (Sprintf/Errorf render for humans)
	"Sprintf": true, "Fprintf": true,
	// slog attribute construction
	"Attr": true, "String": true, "Any": true,
}

func detectBrandedStringCalls(ctx *analyzer.AnalysisContext, gf *analyzer.GoFile) []finding.Finding {
	var findings []finding.Finding

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 0 {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "String" {
			return true
		}

		if !brandedStringMethod(gf.Pkg.TypesInfo, sel) {
			return true
		}

		if enclosedInDisplayCall(gf.AST, call) {
			return true
		}

		pos := ctx.Fset.Position(call.Pos())

		b := findingTemplate.Builder(
			"C044",
			"branded-ID .String() renders the PREFIXED display form — use .Get()/the typed value for keys, SQL args, cursors, and wire payloads",
			finding.SeverityWarning,
			finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
		).
			WithConfidence(finding.ConfidenceMedium).
			WithSnippet(ctx.SourceLine(pos.Filename, pos.Line))

		if f, err := b.Build(); err == nil {
			findings = append(findings, f)
		}

		return true
	})

	return findings
}

// brandedStringMethod reports whether sel resolves to the String method
// DECLARED by the branded-ID packages (go-branded-id's generic ID, or the
// id package's display overrides) — i.e. .String() on a branded receiver.
func brandedStringMethod(info *types.Info, sel *ast.SelectorExpr) bool {
	obj, ok := info.ObjectOf(sel.Sel).(*types.Func)
	if !ok {
		return false
	}

	sig, ok := obj.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return false
	}

	named, ok := resolveNamed(sig.Recv().Type())
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}

	switch named.Obj().Pkg().Path() {
	case "github.com/larsartmann/go-branded-id",
		"github.com/larsartmann/go-cqrs-lite/id/v4",
		"github.com/larsartmann/go-cqrs-lite/core/v5/id":
		return true
	default:
		return false
	}
}

func resolveNamed(t types.Type) (*types.Named, bool) {
	if alias, ok := t.(*types.Alias); ok {
		t = types.Unalias(alias)
	}

	named, ok := t.(*types.Named)

	return named, ok
}

// enclosedInDisplayCall reports whether the .String() call appears as a
// direct argument of a logging/formatting call (the display whitelist).
func enclosedInDisplayCall(file *ast.File, target *ast.CallExpr) bool {
	enclosing := findEnclosingCallArg(file, target)
	if enclosing == nil {
		return false
	}

	callee, ok := enclosing.Fun.(*ast.SelectorExpr)
	if !ok {
		if ident, ok := enclosing.Fun.(*ast.Ident); ok {
			return displayContextCalleeNames[ident.Name]
		}

		return false
	}

	name := callee.Sel.Name
	if !displayContextCalleeNames[name] {
		return false
	}

	// The whitelisted name must belong to a logging/formatting package or
	// receiver: the import path (or the receiver's type) mentions log/slog,
	// fmt, or a logging-ish name. Conservative: also accept when the X
	// expression text carries no package (method on a logger receiver).
	if path := importPathOf(file, callee); path != "" {
		return isDisplayImportPath(path)
	}

	return true
}

// isDisplayImportPath matches logging and formatting packages: stdlib log,
// log/slog (and vendored slog variants), fmt.
func isDisplayImportPath(path string) bool {
	return path == "log" || path == "fmt" ||
		strings.HasPrefix(path, "log/") ||
		strings.Contains(path, "/slog") ||
		strings.Contains(path, "/log") ||
		strings.Contains(path, "/fmt")
}

// findEnclosingCallArg returns the CallExpr, if any, that has target as a
// direct argument.
func findEnclosingCallArg(file *ast.File, target *ast.CallExpr) *ast.CallExpr {
	var found *ast.CallExpr

	ast.Inspect(file, func(n ast.Node) bool {
		if found != nil {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok || call == target {
			return true
		}

		for _, arg := range call.Args {
			if arg == ast.Node(target) {
				found = call
				return false
			}
		}

		return true
	})

	return found
}

// importPathOf resolves the import path behind a selector's X identifier
// when it is a package name.
func importPathOf(file *ast.File, sel *ast.SelectorExpr) string {
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}

	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)

		if imp.Name != nil {
			if imp.Name.Name == ident.Name {
				return path
			}

			continue
		}

		if path[strings.LastIndex(path, "/")+1:] == ident.Name {
			return path
		}
	}

	return ""
}
