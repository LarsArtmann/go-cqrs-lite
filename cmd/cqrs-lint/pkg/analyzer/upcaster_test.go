package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/packages"
)

// findCallBySelName locates the first call named symbol, matching both bare
// identifier calls and selector calls by their function name. Used to pick
// specific calls in upcaster fixtures.
func findCallBySelName(t *testing.T, gf *GoFile, symbol string) *ast.CallExpr {
	t.Helper()

	var found *ast.CallExpr

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			if fn.Sel.Name == symbol {
				found = call
				return false
			}
		case *ast.Ident:
			if fn.Name == symbol {
				found = call
				return false
			}
		}
		return true
	})

	if found == nil {
		t.Fatalf("call %q not found in fixture", symbol)
	}

	return found
}

func upcasterFixtureGoFile(t *testing.T) *GoFile {
	t.Helper()

	src := `package consumer

import "schema"

func upcast(raw []byte) error {
	return schema.NewUpcaster("user", func(b []byte) error {
		_, err := jsonInClosure(b)
		return err
	})(raw)
}

func elsewhere(raw []byte) error {
	_, err := jsonOutside(raw)
	return err
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "consumer.go", src, 0)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	return &GoFile{
		Path: "consumer.go",
		AST:  file,
		Pkg:  &packages.Package{PkgPath: "example.com/consumer"},
	}
}

func TestIsInsideUpcasterClosure_CallPositions(t *testing.T) {
	t.Parallel()

	gf := upcasterFixtureGoFile(t)

	if !IsInsideUpcasterClosure(gf, findCallBySelName(t, gf, "jsonInClosure")) {
		t.Error("a call inside the NewUpcaster closure must be detected")
	}

	if IsInsideUpcasterClosure(gf, findCallBySelName(t, gf, "jsonOutside")) {
		t.Error("a call outside the closure must not be detected")
	}
}

// The cache must serve identical answers on repeated queries (the memoized
// path is what production hits after the first candidate call per file).
func TestIsInsideUpcasterClosure_CachedResultsStable(t *testing.T) {
	t.Parallel()

	gf := upcasterFixtureGoFile(t)
	inCall := findCallBySelName(t, gf, "jsonInClosure")
	outCall := findCallBySelName(t, gf, "jsonOutside")

	for range 5 {
		if !IsInsideUpcasterClosure(gf, inCall) {
			t.Fatal("cached answer flipped for the in-closure call")
		}
		if IsInsideUpcasterClosure(gf, outCall) {
			t.Fatal("cached answer flipped for the out-of-closure call")
		}
	}
}
