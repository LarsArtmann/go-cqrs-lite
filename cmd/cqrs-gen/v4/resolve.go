package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

type symbolTable struct {
	names  map[string]bool
	values map[string]string
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

	// A constructor is derivable only from the envelope embed, and a
	// hand-written New<Struct> keeps its place: it may carry validation.
	m.constructor = m.byValue && !symbols.names[constructorName(m)]

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
