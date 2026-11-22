package main

// rewrite_v5_surface.go extends --to-v5 with the new-surface rewrites (M14,
// plan 2026-11-19): the mechanical transformations consumer code needs after
// the core/v5 type finalization — the validating StreamRef constructors
// (ADR-0166), the string-form record constructors that died with the slash
// shape, and manual-review reporting for the collapsed Record fields that
// only type information can rewrite safely.
//
// Rules:
//
//	R1 single-value id/command/event NewStreamRef + NewAggregateRef calls
//	   -> id.NewStreamRefOrZero (the v5 single-value door; two-value
//	   assignment contexts are left alone — their shape is unchanged).
//	R2 record.NewStreamRef / record.NewStreamRefOrZero (string,string) calls
//	   -> cqrsV5StreamRef(...) — a per-file injected helper that adapts the
//	   old string form onto the validating struct (zero ref on invalid,
//	   never malformed).
//	R3 record-literal keys StreamID/StreamType/Encoding and .Split() calls
//	   -> REPORTED as manual sites (parser-only pass cannot type-check; the
//	   fields collapsed into Stream / Payload.Enc in M04/M05).

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strconv"
	"strings"
)

const (
	v5HelperName      = "cqrsV5StreamRef"
	v5HelperMarker    = "cqrs-upgrade v5 helper"
	v5IDImportPath    = "github.com/larsartmann/go-cqrs-lite/core/v5/id"
	v5RecordQualifier = "record"
)

// v5SurfaceChange is one applied (or planned) new-surface rewrite.
type v5SurfaceChange struct {
	Kind string
	From string
	To   string
}

// singleValueStreamRefCtors maps package qualifier + function to the v5
// single-value door. Two-value contexts (ref, err := ...) keep the original
// name — v5 preserved that shape.
var singleValueStreamRefCtors = map[string]string{
	"id.NewStreamRef":         "id.NewStreamRefOrZero",
	"id.NewAggregateRef":      "id.NewStreamRefOrZero",
	"command.NewStreamRef":    "id.NewStreamRefOrZero",
	"command.NewAggregateRef": "id.NewStreamRefOrZero",
}

// stringFormRecordCtors are the v4 record-package constructors that took two
// strings and died with the slash shape (ADR-0166).
var stringFormRecordCtors = map[string]bool{
	"record.NewStreamRef":       true,
	"record.NewStreamRefOrZero": true,
}

// v5HelperSrc is injected once per file that still uses the string form.
const v5HelperSrc = `

// cqrs-upgrade v5 helper: the v4 record.NewStreamRef(string, string) form
// built a slash-joined ref; ADR-0166 made refs a validated struct. This
// adapts the old string form — a zero ref on invalid input, never a
// malformed one.
func cqrsV5StreamRef(streamType, entityID string) record.StreamRef {
	sid, err := id.ParseStreamID(entityID)
	if err != nil {
		return record.StreamRef{}
	}
	return id.NewStreamRefOrZero(id.StreamType(streamType), sid)
}
`

// edit is one byte-span replacement (offsets into the file's source).
type edit struct {
	start, end int
	text       string
}

// rewriteV5SurfaceInFile applies the M14 surface rules to one file. It
// returns the applied changes, the manual-review sites, and whether the file
// was modified. Parse failures are returned — a file the codemod cannot read
// is never silently skipped.
func rewriteV5SurfaceInFile(path string, dryRun bool) ([]v5SurfaceChange, []string, bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, false, err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, nil, false, fmt.Errorf("parse: %w", err)
	}

	if fileForeignHelperCollision(f) {
		return nil, []string{
			path + ":1: file declares a foreign " + v5HelperName +
				" — resolve the name collision by hand (rename yours or the injected helper), then re-run",
		}, false, nil
	}

	var edits []edit
	var changes []v5SurfaceChange
	var manual []string

	// parents maps each node to its parent for context checks.
	parents := recordParents(f)

	hasIDImport := false
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err == nil && p == v5IDImportPath {
			hasIDImport = true
		}
	}
	needIDImport := !hasIDImport

	sites := collectSurfaceSites(f, fset, parents, path)
	edits = sites.edits
	changes = sites.changes
	manual = sites.manual
	if sites.anyCtorRewrite {
		needIDImport = true
	}

	needHelper := sites.sawStringCtor

	if len(edits) == 0 && !needHelper {
		return nil, manual, false, nil
	}

	out, changedFile := applySurfaceEdits(string(src), fset, f, edits, needIDImport, needHelper)
	if !changedFile {
		return changes, manual, false, nil
	}

	if !dryRun {
		if err := osWriteFile(path, []byte(out)); err != nil {
			return nil, nil, false, err
		}
	}

	return changes, manual, true, nil
}

// applySurfaceEdits sorts the byte-span replacements, applies them back to
// front (descending offsets keep later spans' positions valid), then splices
// in the id import and the adapting helper when the file lacks them.
func applySurfaceEdits(
	src string,
	fset *token.FileSet,
	f *ast.File,
	edits []edit,
	needIDImport, needHelper bool,
) (string, bool) {
	slices.SortStableFunc(edits, func(a, b edit) int {
		switch {
		case a.start < b.start:
			return -1
		case a.start > b.start:
			return 1
		}

		return 0
	})

	out := src
	for _, e := range slices.Backward(edits) {
		out = out[:e.start] + e.text + out[e.end:]
	}

	if needIDImport && len(f.Decls) > 0 {
		out = injectIDImport(out, fset, f)
	}

	if needHelper && !fileDeclaresHelper(f) {
		pkgEnd := fset.Position(f.Package).Offset + len("package "+f.Name.Name)
		out = out[:pkgEnd] + v5HelperSrc + out[pkgEnd:]
	}

	return out, out != src
}

// fileDeclaresHelper reports whether the file already carries the injected
// helper (idempotent re-runs).
func fileDeclaresHelper(f *ast.File) bool {
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if ok && fd.Name.Name == v5HelperName {
			return true
		}
	}

	return false
}

// fileForeignHelperCollision reports whether the file declares a
// cqrsV5StreamRef that is NOT the codemod's own helper — a user function
// squatting on the injected name. Rewriting call sites to that name would
// compile-break (different signature) or silently call the wrong code, so
// such files route to manual review instead of any rewrite.
func fileForeignHelperCollision(f *ast.File) bool {
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != v5HelperName {
			continue
		}

		if fd.Doc == nil || !strings.Contains(fd.Doc.Text(), v5HelperMarker) {
			return true
		}

		if !helperShapeMatch(fd.Type) {
			return true
		}
	}

	return false
}

// helperShapeMatch reports whether a FuncType matches the injected helper's
// (streamType, entityID string) record.StreamRef signature.
func helperShapeMatch(ft *ast.FuncType) bool {
	if ft == nil || ft.Params == nil || len(ft.Params.List) != 1 || len(ft.Params.List[0].Names) != 2 {
		return false
	}

	if id, ok := ft.Params.List[0].Type.(*ast.Ident); !ok || id.Name != "string" {
		return false
	}

	if ft.Results == nil || len(ft.Results.List) != 1 {
		return false
	}

	sel, ok := ft.Results.List[0].Type.(*ast.SelectorExpr)

	return ok && sel.Sel != nil && sel.Sel.Name == "StreamRef"
}

// injectIDImport adds the core/v5/id import after the package clause when
// the file lacks it. The insertion point is right after the package name —
// gofmt/goimports consolidates block placement later (the sweep step owns
// formatting).
func injectIDImport(src string, fset *token.FileSet, f *ast.File) string {
	pkgEnd := fset.Position(f.Package).Offset + len("package "+f.Name.Name)
	imp := "\n\nimport " + strconv.Quote(v5IDImportPath)

	return src[:pkgEnd] + imp + src[pkgEnd:]
}

// recordParents maps every node in the file to its parent node (go/ast has
// no Children accessor; the manual-recursion Inspect below visits each
// direct child once and stops, handing subtrees to the recursive walk).
func recordParents(f *ast.File) map[ast.Node]ast.Node {
	parents := make(map[ast.Node]ast.Node)

	var walk func(parent, node ast.Node)
	walk = func(parent, node ast.Node) {
		if node == nil {
			return
		}

		parents[node] = parent
		ast.Inspect(node, func(c ast.Node) bool {
			if c == nil || c == node {
				return c != nil
			}
			walk(node, c)

			return false
		})
	}
	walk(nil, f)

	return parents
}
