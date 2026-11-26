package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

const structTagKey = "cqrs"

type symbolTable struct {
	names  map[string]bool
	values map[string]string
}

func resolvePackage(
	dir string,
	files []*ast.File,
	fset *token.FileSet,
	kinds map[kind]bool,
) ([]pkg, error) {
	name, err := packageName(dir, files, fset)
	if err != nil {
		return nil, err
	}

	symbols := collectSymbols(files)

	markers, err := collectMarkers(files, fset, kinds)
	if err != nil {
		return nil, err
	}

	if len(markers) == 0 {
		return nil, nil
	}

	if err := resolveMarkers(markers, symbols); err != nil {
		return nil, err
	}

	slices.SortFunc(markers, compareMarkers)

	return []pkg{{dir: dir, name: name, markers: markers}}, nil
}

func packageName(dir string, files []*ast.File, fset *token.FileSet) (string, error) {
	first := files[0]

	var foreign []string

	for _, file := range files[1:] {
		if file.Name.Name != first.Name.Name {
			foreign = append(foreign, fset.Position(file.Package).Filename)
		}
	}

	if len(foreign) > 0 {
		return "", fmt.Errorf("%s mixes package %q with other packages (%s): "+
			"generated code needs one package per directory",
			dir, first.Name.Name, strings.Join(foreign, ", "))
	}

	return first.Name.Name, nil
}

func collectMarkers(files []*ast.File, fset *token.FileSet, kinds map[kind]bool) ([]marker, error) {
	var markers []marker

	for _, file := range files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				found, marked, err := markerOf(genDecl, ts, fset, kinds)
				if err != nil {
					return nil, err
				}

				if marked {
					markers = append(markers, found)
				}
			}
		}
	}

	return markers, nil
}

func markerOf(
	decl *ast.GenDecl,
	spec *ast.TypeSpec,
	fset *token.FileSet,
	kinds map[kind]bool,
) (marker, bool, error) {
	markedKind, value, marked := commentMarker(kinds, decl.Doc, spec.Doc)
	if !marked {
		markedKind, value, marked = tagMarker(spec.Type, kinds)
	}

	if !marked {
		return marker{}, false, nil
	}

	pos := fset.Position(spec.Pos()).String()

	st, isStruct := spec.Type.(*ast.StructType)
	if !isStruct || spec.TypeParams != nil || spec.Assign.IsValid() {
		return marker{}, false, fmt.Errorf("%s: the cqrs:%s marker marks %q, "+
			"which is not a plain struct type declaration", pos, markedKind, spec.Name.Name)
	}

	wire, receives, err := parseMarkerValue(markedKind, value)
	if err != nil {
		return marker{}, false, fmt.Errorf("%s: %w", pos, err)
	}

	return marker{
		kind:       markedKind,
		structName: spec.Name.Name,
		wire:       wire,
		receives:   receives,
		byValue:    embedsEnvelope(st, markedKind),
		pos:        pos,
	}, true, nil
}

// embedsEnvelope: a struct embedding the library's command or query envelope
// by pointer satisfies the interface as a value; every other shape needs a
// pointer.
func embedsEnvelope(st *ast.StructType, k kind) bool {
	if st.Fields == nil {
		return false
	}

	want := "BasicCommand"
	if k == kindQuery {
		want = "BasicQuery"
	}

	for _, field := range st.Fields.List {
		star, ok := field.Type.(*ast.StarExpr)
		if !ok || len(field.Names) > 0 {
			continue
		}

		if sel, ok := star.X.(*ast.SelectorExpr); ok && sel.Sel.Name == want {
			return true
		}
	}

	return false
}

func commentMarker(kinds map[kind]bool, docs ...*ast.CommentGroup) (kind, string, bool) {
	for _, doc := range docs {
		if doc == nil {
			continue
		}

		for _, comment := range doc.List {
			k, value, marked := parseMarkerComment(comment.Text)
			if marked && kinds[k] {
				return k, value, true
			}
		}
	}

	return "", "", false
}

func parseMarkerComment(text string) (kind, string, bool) {
	rest, ok := strings.CutPrefix(text, markerPrefix)
	if !ok {
		return "", "", false
	}

	name, value := splitMarker(rest)

	k := kind(name)

	return k, value, isKind(k)
}

func splitMarker(rest string) (string, string) {
	for i, r := range rest {
		if r == ' ' || r == '\t' {
			return rest[:i], rest[i+1:]
		}
	}

	return rest, ""
}

func tagMarker(expr ast.Expr, kinds map[kind]bool) (kind, string, bool) {
	st, ok := expr.(*ast.StructType)
	if !ok || st.Fields == nil {
		return "", "", false
	}

	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}

		raw := strings.TrimSuffix(strings.TrimPrefix(field.Tag.Value, "`"), "`")

		tag, ok := reflect.StructTag(raw).Lookup(structTagKey)
		if !ok {
			continue
		}

		name, value, _ := strings.Cut(tag, ":")

		k := kind(name)
		if isKind(k) && kinds[k] {
			return k, value, true
		}
	}

	return "", "", false
}

func parseMarkerValue(k kind, value string) (string, bool, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "", false, fmt.Errorf("the cqrs:%s marker carries no type string", k)
	}

	if k != kindEvent {
		return fields[0], false, rejectOptions(k, fields[1:])
	}

	receives := false

	for _, option := range fields[1:] {
		switch option {
		case optionSends:
		case optionReceives:
			receives = true
		default:
			return "", false, fmt.Errorf("the cqrs:event marker takes %q or %q, got %q",
				optionSends, optionReceives, option)
		}
	}

	return fields[0], receives, nil
}

func rejectOptions(k kind, options []string) error {
	if len(options) == 0 {
		return nil
	}

	return fmt.Errorf("the cqrs:%s marker takes no options, got %q", k, options[0])
}

func collectSymbols(files []*ast.File) symbolTable {
	t := symbolTable{names: map[string]bool{}, values: map[string]string{}}

	for _, file := range files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				t.names[d.Name.Name] = true
			case *ast.GenDecl:
				collectValueSpecs(d, &t)
			}
		}
	}

	return t
}

func collectValueSpecs(decl *ast.GenDecl, t *symbolTable) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			t.names[s.Name.Name] = true
		case *ast.ValueSpec:
			for i, name := range s.Names {
				t.names[name.Name] = true

				if i < len(s.Values) {
					if value, ok := stringValue(s.Values[i]); ok {
						t.values[name.Name] = value
					}
				}
			}
		}
	}
}

func stringValue(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}

		value, err := strconv.Unquote(e.Value)

		return value, err == nil
	case *ast.CallExpr:
		if len(e.Args) != 1 || !isConversion(e.Fun) {
			return "", false
		}

		return stringValue(e.Args[0])
	case *ast.ParenExpr:
		return stringValue(e.X)
	}

	return "", false
}

func isConversion(fun ast.Expr) bool {
	switch fun.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		return true
	}

	return false
}

func resolveMarkers(markers []marker, symbols symbolTable) error {
	claimed := map[kind]map[string]string{}

	for i := range markers {
		m := &markers[i]

		if other := claimed[m.kind][m.wire]; other != "" {
			return fmt.Errorf("%s: %s and %s both claim the %s type string %q",
				m.pos, other, m.structName, m.kind, m.wire)
		}

		if err := m.resolve(symbols); err != nil {
			return fmt.Errorf("%s: %w", m.pos, err)
		}

		if claimed[m.kind] == nil {
			claimed[m.kind] = map[string]string{}
		}

		claimed[m.kind][m.wire] = m.structName
	}

	return nil
}

// resolve names the constant the generated code uses for the wire name. An
// identifier that names a package-level declaration IS the reference, so a
// package never states a wire name twice.
func (m *marker) resolve(symbols symbolTable) error {
	existing := symbols.nameOfValue(m.wire)

	switch {
	case isIdentifier(m.wire) && symbols.names[m.wire]:
		m.constName = m.wire
	case existing != "":
		m.constName = existing
	default:
		m.constName = constantName(m.kind, m.structName)
		m.constExpr = strconv.Quote(m.wire)
	}

	return m.rejectTaken(symbols)
}

func (t symbolTable) nameOfValue(wire string) string {
	best := ""

	for name, value := range t.values {
		if value == wire && (best == "" || name < best) {
			best = name
		}
	}

	return best
}

func (m *marker) rejectTaken(symbols symbolTable) error {
	if m.constExpr != "" && symbols.names[m.constName] {
		return fmt.Errorf("generated constant %s collides with an existing package-level "+
			"declaration; reference that declaration in the marker instead", m.constName)
	}

	for _, name := range m.reservedNames() {
		if symbols.names[name] {
			return fmt.Errorf("generated declaration %s collides with an existing package-level "+
				"declaration of the same name", name)
		}
	}

	return nil
}
