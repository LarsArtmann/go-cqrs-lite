package version

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// This file implements the V007 drift contract's repo scanner: it walks the
// go-cqrs-lite checkout and collects every line-initial `Deprecated:` doc
// marker that cites v5 removal (ADR-0114/0123/0126/0127 or "removed … v5").
// The meta-tests in v007_drift_test.go hold the V007 tables against this
// inventory in both directions, so a new v5 removal without a table (or
// allowlist) entry fails CI instead of rotting silently.

// v5DriftDecl is one discovered v5 deprecation marker in the repo.
type v5DriftDecl struct {
	kind   string // func | type | var | const | method | package
	frag   string // detector fragment space: decl dir, /vN segments stripped
	symbol string
	recv   string // receiver type for methods
	pos    string // repo-relative file:line
}

// v5MarkerKey identifies a package-level declaration (or a package doc).
type v5MarkerKey struct {
	frag   string
	symbol string
}

// v5MethodKey identifies a deprecated method on a receiver type.
type v5MethodKey struct {
	frag   string
	recv   string
	symbol string
}

// v5DriftAllowlist exempts forward-contract markers from needing table
// entries, each with the reason it is intentionally not flagged by V007.
// (Test fixture data — test files are exempt from gochecknoglobals.)
var v5DriftAllowlist = map[v5MarkerKey]string{
	{"transport/http", "(package)"}: "F030 (deprecated-transport-import) owns transport/* at import granularity",
	{"transport/grpc", "(package)"}: "F030 (deprecated-transport-import) owns transport/* at import granularity",
	{"stack", "(package)"}:          "package-doc deprecation; V007 flags the individual symbols instead",
}

// v5DriftMethodAllowlist lists deprecated METHODS on receiver types that
// survive v5. Methods are undetectable by V007 (it matches qualifier.Symbol
// selectors only; a method call's receiver is a value, not a package).
var v5DriftMethodAllowlist = map[v5MethodKey]string{
	{"decider", "Repository", "Execute"}:             "pair-form forwarder removed at v5; use ExecuteRef",
	{"decider", "Repository", "Load"}:                "pair-form forwarder removed at v5; use LoadRef",
	{"decider", "Repository", "LoadAtVersion"}:       "pair-form forwarder removed at v5; use LoadAtVersionRef",
	{"decider", "Repository", "LoadAtTime"}:          "pair-form forwarder removed at v5; use LoadAtTimeRef",
	{"decider", "Repository", "WaitForVersion"}:      "pair-form forwarder removed at v5; use WaitForVersionRef",
	{"decider", "TypedRepository", "ExecuteCommand"}: "pair-form forwarder removed at v5; use ExecuteCommandRef",
	{"decider", "TypedRepository", "Load"}:           "pair-form forwarder removed at v5; use LoadRef",
	{"metadata", "Metadata", "EnsureCustom"}:         "in-place mutation removed at v5 (ADR-0126); use WithCustom",
}

// v5StaleEntryAllowlist exempts table entries that have no live package-level
// v5 marker, each with the reason the entry is still correct.
var v5StaleEntryAllowlist = map[v5MarkerKey]string{
	{"event", "TombstoneActive"}:       "enum member of deprecated TombstoneStatus; dies with the type (const carries no own marker)",
	{"event", "TombstoneTombstoned"}:   "enum member of deprecated TombstoneStatus; dies with the type (const carries no own marker)",
	{"event", "TombstoneUndetermined"}: "enum member of deprecated TombstoneStatus; dies with the type (const carries no own marker)",
}

var (
	v5DriftScanOnce   sync.Once
	v5DriftScanAll    []v5DriftDecl
	v5DriftScanFailed []string
	v5DriftScanErr    error
)

// scanV5DriftMarkers returns the repo-wide v5 deprecation inventory, walking
// the checkout once per process and caching the result across meta-tests.
func scanV5DriftMarkers(t *testing.T) []v5DriftDecl {
	t.Helper()
	v5DriftScanOnce.Do(func() {
		root, err := repoRootFromCwd()
		if err != nil {
			v5DriftScanErr = err
			return
		}
		v5DriftScanAll, v5DriftScanFailed, v5DriftScanErr = walkV5Markers(root)
	})
	if v5DriftScanErr != nil {
		t.Fatalf("v5 drift scan failed: %v", v5DriftScanErr)
	}
	for _, f := range v5DriftScanFailed {
		t.Logf("v5 drift scan: skipped unparseable file: %s", f)
	}
	return v5DriftScanAll
}

func repoRootFromCwd() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if isRepoRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func isRepoRoot(dir string) bool {
	for _, marker := range []string{"go.work", "AGENTS.md", "flake.nix"} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err != nil {
			return false
		}
	}
	return true
}

func walkV5Markers(root string) ([]v5DriftDecl, []string, error) {
	var out []v5DriftDecl
	var failed []string
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "node_modules" || name == "testdata" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}

		src, perr := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if perr != nil {
			failed = append(failed, path+": "+perr.Error())
			return nil //nolint:nilerr // skip unparseable file, recorded in failed
		}

		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			failed = append(failed, path+": "+rerr.Error())
			return nil //nolint:nilerr // skip unrelativizable path, recorded in failed
		}
		frag := stripVersionSuffix(filepath.ToSlash(filepath.Dir(rel)))
		posPrefix := rel

		if _, v5 := v5DocSignal(src.Doc); v5 {
			out = append(out, v5DriftDecl{kind: "package", frag: frag, symbol: "(package)", pos: posPrefix + ":1"})
		}

		for _, decl := range src.Decls {
			collectV5Marker(fset, decl, frag, posPrefix, &out)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return out, failed, nil
}

func collectV5Marker(fset *token.FileSet, decl ast.Decl, frag, posPrefix string, out *[]v5DriftDecl) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if _, v5 := v5DocSignal(d.Doc); !v5 {
			return
		}
		line := strconv.Itoa(fset.Position(d.Pos()).Line)
		if d.Recv == nil {
			*out = append(*out, v5DriftDecl{kind: "func", frag: frag, symbol: d.Name.Name, pos: posPrefix + ":" + line})
			return
		}
		*out = append(*out, v5DriftDecl{
			kind:   "method",
			frag:   frag,
			symbol: d.Name.Name,
			recv:   recvTypeName(d.Recv),
			pos:    posPrefix + ":" + line,
		})
	case *ast.GenDecl:
		if d.Tok != token.TYPE && d.Tok != token.VAR && d.Tok != token.CONST {
			return
		}
		for _, spec := range d.Specs {
			collectV5Spec(fset, d, spec, frag, posPrefix, out)
		}
	}
}

func collectV5Spec(
	fset *token.FileSet,
	decl *ast.GenDecl,
	spec ast.Spec,
	frag, posPrefix string,
	out *[]v5DriftDecl,
) {
	var names []string
	var doc *ast.CommentGroup
	switch s := spec.(type) {
	case *ast.TypeSpec:
		names, doc = make([]string, 0, 1), s.Doc
		names = append(names, s.Name.Name)
	case *ast.ValueSpec:
		names, doc = make([]string, 0, len(s.Names)), s.Doc
		for _, id := range s.Names {
			names = append(names, id.Name)
		}
	default:
		return
	}
	if doc == nil && len(decl.Specs) == 1 {
		doc = decl.Doc
	}
	if _, v5 := v5DocSignal(doc); !v5 {
		return
	}
	line := strconv.Itoa(fset.Position(spec.Pos()).Line)
	for _, n := range names {
		*out = append(*out, v5DriftDecl{kind: decl.Tok.String(), frag: frag, symbol: n, pos: posPrefix + ":" + line})
	}
}

// v5DocSignal reports whether the comment group carries a line-initial
// "Deprecated:" marker and references v5 removal somewhere in its text.
func v5DocSignal(doc *ast.CommentGroup) (string, bool) {
	if doc == nil {
		return "", false
	}
	hasMarker := false
	var lines []string
	for _, c := range doc.List {
		text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		lines = append(lines, text)
		if strings.HasPrefix(text, "Deprecated:") {
			hasMarker = true
		}
	}
	if !hasMarker {
		return "", false
	}
	joined := strings.ToLower(strings.Join(lines, " "))
	for _, sig := range []string{
		"removed in v5", "removed at v5", "removed v5", "removal at v5",
		"adr-0114", "adr-0123", "adr-0126", "adr-0127",
	} {
		if strings.Contains(joined, sig) {
			return strings.Join(lines, "\n"), true
		}
	}
	return "", false
}

func recvTypeName(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return "?"
	}
	t := fl.List[0].Type
	for {
		switch tt := t.(type) {
		case *ast.StarExpr:
			t = tt.X
		case *ast.IndexExpr:
			t = tt.X
		case *ast.IndexListExpr:
			t = tt.X
		case *ast.Ident:
			return tt.Name
		default:
			return "?"
		}
	}
}
