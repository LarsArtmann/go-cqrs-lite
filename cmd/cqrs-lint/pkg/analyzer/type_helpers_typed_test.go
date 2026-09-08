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

// --- T20-8: IsQualifierFor / IsEventTypeParam adopt the Tier-1 typed
// resolution. These tests synthesize real types.Info shapes; the AST-only
// harness (BuildContextFromSource) cannot exercise the typed paths.

func typedSelector(qualifier, symbol string) (*ast.Ident, *ast.SelectorExpr) {
	ident := &ast.Ident{Name: qualifier, NamePos: token.NoPos}
	return ident, &ast.SelectorExpr{X: ident, Sel: &ast.Ident{Name: symbol, NamePos: token.NoPos}}
}

func TestIsQualifierFor_AliasedImportMatchesByPath(t *testing.T) {
	t.Parallel()

	eventPkg := types.NewPackage("github.com/larsartmann/go-cqrs-lite/event/v4", "event")
	ident, sel := typedSelector("ev", "New")
	pkgObj := types.NewPkgName(token.NoPos, eventPkg, "event", eventPkg)

	gf := synthesizedGoFile(map[*ast.Ident]types.Object{ident: pkgObj})

	if !IsQualifierFor(gf, sel, "go-cqrs-lite/event") {
		t.Error("aliased import `ev` must match by its resolved import path, not its local name")
	}
}

func TestIsQualifierFor_ShadowedQualifierRejected(t *testing.T) {
	t.Parallel()

	shadow := types.NewVar(token.NoPos, nil, "event", types.NewStruct(nil, nil))
	ident, sel := typedSelector("event", "New")

	gf := synthesizedGoFile(map[*ast.Ident]types.Object{ident: shadow})

	if IsQualifierFor(gf, sel, "go-cqrs-lite/event") {
		t.Error("a shadowing local whose name equals the package name must NOT match")
	}
}

func TestIsQualifierFor_MissingTypesFallsBackToName(t *testing.T) {
	t.Parallel()

	_, sel := typedSelector("schema", "NewUpcaster")
	gf := synthesizedGoFile(nil)

	if !IsQualifierFor(gf, sel, "go-cqrs-lite/schema") {
		t.Error("without type info the qualifier name is the fallback and must match")
	}
}

// eventPackageFixtures builds the event module package with an
// ImmutableEvent named type and the *ImmutableEvent event type shape.
func eventPackageFixtures() types.Type {
	eventPkg := types.NewPackage("github.com/larsartmann/go-cqrs-lite/event/v4", "event")
	immutable := types.NewNamed(
		types.NewTypeName(token.NoPos, eventPkg, "ImmutableEvent", nil),
		types.NewStruct(nil, nil), nil,
	)

	return types.NewPointer(immutable)
}

func TestIsEventTypeParam_DefinedTypeOverEventMatches(t *testing.T) {
	t.Parallel()

	evtType := eventPackageFixtures()
	consumer := types.NewPackage("example.com/consumer", "consumer")

	// `type OrderEvt event.Event` — defined type whose chain reaches the
	// event module. The old string heuristic saw only "OrderEvt" (blind).
	orderObj := types.NewTypeName(token.NoPos, consumer, "OrderEvt", nil)
	types.NewNamed(orderObj, evtType, nil)

	ident := &ast.Ident{Name: "OrderEvt", NamePos: token.NoPos}
	gf := synthesizedGoFile(map[*ast.Ident]types.Object{ident: orderObj})

	if !IsEventTypeParam(gf, ident) {
		t.Error(
			"a consumer defined type over event.Event must be recognized (alias-fold-blindness fix)",
		)
	}
}

func TestIsEventTypeParam_UnrelatedSameNamedTypeRejected(t *testing.T) {
	t.Parallel()

	// `events.Event` in a CONSUMER package named events — the old string
	// heuristic matched it (contains "event" and "Event"); the typed path
	// must reject it because the defining package is not the event module.
	eventsPkg := types.NewPackage("example.com/myapp/events", "events")
	evtObj := types.NewTypeName(token.NoPos, eventsPkg, "Event", nil)
	types.NewNamed(evtObj, types.NewStruct(nil, nil), nil)

	ident := &ast.Ident{Name: "Event", NamePos: token.NoPos}
	gf := synthesizedGoFile(map[*ast.Ident]types.Object{ident: evtObj})

	expr := &ast.SelectorExpr{X: &ast.Ident{Name: "events", NamePos: token.NoPos}, Sel: ident}

	if IsEventTypeParam(gf, expr) {
		t.Error("a consumer package's own Event type must not be treated as a cqrs event")
	}
}

func TestIsEventTypeParam_MissingTypesFallsBackToStrings(t *testing.T) {
	t.Parallel()

	expr := &ast.SelectorExpr{
		X:   &ast.Ident{Name: "event", NamePos: token.NoPos},
		Sel: &ast.Ident{Name: "Event", NamePos: token.NoPos},
	}

	if !IsEventTypeParam(synthesizedGoFile(nil), expr) {
		t.Error("without type info the string heuristic must still match event.Event")
	}
}
