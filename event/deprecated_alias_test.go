//go:build goexperiment.jsonv2

package event_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestDeprecatedAliasesHaveComments verifies that every known deprecated alias
// in the event/ package has a proper "Deprecated:" comment.
//
// v4-removal: When these aliases are deleted, this test should also be removed.
func TestDeprecatedAliasesHaveComments(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse event/: %v", err)
	}

	pkg, ok := pkgs["event"]
	if !ok {
		t.Fatal("package event not found")
	}

	deprecatedNames := map[string]bool{
		"AggregateRef":       true,
		"NewAggregateRef":    true,
		"AggregateType":      true,
		"ParseAggregateType": true,
		"Tracing":            true,
		"CustomData":         true,
	}

	found := make(map[string]bool)

	checkDoc := func(name, shortName, doc string) {
		found[name] = true
		if !strings.Contains(doc, "Deprecated:") {
			t.Errorf("%s: %s should have Deprecated: comment", shortName, name)
		}
	}

	for _, file := range pkg.Files {
		shortName := fset.Position(file.Pos()).Filename
		if idx := strings.LastIndex(shortName, "/"); idx >= 0 {
			shortName = shortName[idx+1:]
		}

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if deprecatedNames[s.Name.Name] {
							doc := ""
							if d.Doc != nil {
								doc = d.Doc.Text()
							}
							checkDoc(s.Name.Name, shortName, doc)
						}

					case *ast.ValueSpec:
						if deprecatedNames[s.Names[0].Name] {
							doc := ""
							if d.Doc != nil {
								doc = d.Doc.Text()
							}
							checkDoc(s.Names[0].Name, shortName, doc)
						}
					}
				}
			}
		}
	}

	for name := range deprecatedNames {
		if !found[name] {
			t.Errorf("deprecated alias %q not found in event/ package", name)
		}
	}
}
