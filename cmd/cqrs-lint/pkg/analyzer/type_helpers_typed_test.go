package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

// synthesizedGoFile builds a GoFile whose TypesInfo.Uses maps the given
// idents to objects — the minimal harness for ResolveQualifierTyped, which
// needs REAL type info (BuildContextFromSource produces empty types.Info and
// correctly exercises the fallback path instead).
func synthesizedGoFile(uses map[*ast.Ident]types.Object) *GoFile {
	return &GoFile{
		Pkg: &packages.Package{
			TypesInfo: &types.Info{Uses: uses},
		},
	}
}

func TestResolveQualifierTyped_PackageReference(t *testing.T) {
	t.Parallel()

	imported := types.NewPackage("github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4", "sqlite")
	ident := &ast.Ident{Name: "sqlite", NamePos: token.NoPos}
	pkgObj := types.NewPkgName(token.NoPos, imported, "sqlite", imported)

	gf := synthesizedGoFile(map[*ast.Ident]types.Object{ident: pkgObj})

	path, resolved := ResolveQualifierTyped(gf, ident)
	if !resolved {
		t.Fatal("package reference must resolve authoritatively")
	}
	if path != "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4" {
		t.Errorf("path = %q, want the imported package path", path)
	}
}

func TestResolveQualifierTyped_ShadowedValueIsAuthoritativeEmpty(t *testing.T) {
	t.Parallel()

	// A local variable named like the package: Uses[ident] is a *types.Var.
	// The answer must be (path="", resolved=true) — authoritative "not a
	// package" — so callers do NOT fall back to the string scan and the
	// shadow survives.
	shadow := types.NewVar(token.NoPos, nil, "sqlite", types.NewStruct(nil, nil))
	ident := &ast.Ident{Name: "sqlite", NamePos: token.NoPos}

	gf := synthesizedGoFile(map[*ast.Ident]types.Object{ident: shadow})

	path, resolved := ResolveQualifierTyped(gf, ident)
	if !resolved {
		t.Fatal("a resolved non-package object is still an authoritative answer")
	}
	if path != "" {
		t.Errorf("path = %q, want empty for a shadowed value", path)
	}
}

func TestResolveQualifierTyped_MissingTypesInfoSignalsFallback(t *testing.T) {
	t.Parallel()

	ident := &ast.Ident{Name: "sqlite", NamePos: token.NoPos}
	gf := synthesizedGoFile(nil) // no Uses entries — syntax-only load

	_, resolved := ResolveQualifierTyped(gf, ident)
	if resolved {
		t.Fatal("missing Uses entry must signal the string-scan fallback")
	}
}

func TestResolveQualifierTyped_NilInputs(t *testing.T) {
	t.Parallel()

	if _, resolved := ResolveQualifierTyped(nil, &ast.Ident{}); resolved {
		t.Error("nil GoFile must signal fallback")
	}
	if _, resolved := ResolveQualifierTyped(&GoFile{}, &ast.Ident{}); resolved {
		t.Error("nil Pkg must signal fallback")
	}
	if _, resolved := ResolveQualifierTyped(synthesizedGoFile(nil), nil); resolved {
		t.Error("nil ident must signal fallback")
	}
}
