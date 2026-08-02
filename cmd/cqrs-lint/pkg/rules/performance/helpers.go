package performance

import (
	"go/ast"
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// directlyOpensSQLite returns true if the file contains a direct
// database/sql.Open (or sql.OpenDB) call with a SQLite driver. Only direct
// opens are flagged by P012/P013 — constructor calls like sqlite.New,
// NewSQLiteBackend, and NewSQLiteEventStore are NOT flagged because they
// either apply PRAGMAs internally (stack preset) or receive an already-opened
// *sql.DB (PRAGMA responsibility is in the caller file, which we do flag).
//
// This fixes the cross-file false positive where sqlite.New is called from a
// CLI file but PRAGMAs are applied in a storage wrapper file: the CLI file
// only calls a constructor, so it is no longer flagged.
func directlyOpensSQLite(root ast.Node) bool {
	direct := false

	ast.Inspect(root, func(n ast.Node) bool {
		if direct {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok {
			return true
		}

		// Only sql.Open and sql.OpenDB are "direct opens".
		if sel.Sel.Name != "Open" && sel.Sel.Name != "OpenDB" {
			return true
		}

		pkg := analyzer.SelectorPackage(sel)
		if pkg != "sql" {
			return true
		}

		// For sql.Open, the first argument is the driver name string.
		if sel.Sel.Name == "Open" && len(call.Args) > 0 {
			if lit, ok := call.Args[0].(*ast.BasicLit); ok {
				driver := strings.Trim(lit.Value, `"`)
				if strings.Contains(strings.ToLower(driver), "sqlite") {
					direct = true
					return false
				}
			}
		}

		// For sql.OpenDB we can't statically determine the driver from the
		// connector argument. Fall back to checking whether the file imports
		// a sqlite driver path — handled by the caller via usesSQLiteDriverImport.

		return true
	})

	return direct
}

// callHasOption returns true if any argument to call is a function call
// whose selector name matches optionName (e.g. "WithBatchSize").
// Handles generic wrappers: decider.WithSnapshotStore[State](store).
func callHasOption(call *ast.CallExpr, optionName string) bool {
	for _, arg := range call.Args {
		argCall, ok := arg.(*ast.CallExpr)
		if !ok {
			continue
		}

		sel, ok := analyzer.SelectorFromExpr(argCall.Fun)
		if !ok {
			continue
		}

		if sel.Sel.Name == optionName {
			return true
		}
	}

	return false
}

// findStructType searches all GoFiles for a top-level type spec with the
// given name that is a struct type. Returns nil if not found.
func findStructType(ctx *analyzer.AnalysisContext, name string) *ast.StructType {
	for _, gf := range ctx.GoFiles {
		for _, decl := range gf.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name == nil || ts.Name.Name != name {
					continue
				}

				if st, ok := ts.Type.(*ast.StructType); ok {
					return st
				}
			}
		}
	}

	return nil
}

// structFieldCount returns the number of named fields in a struct type,
// counting each name in multi-name declarations separately. Embedded
// fields (no names) count as one.
func structFieldCount(st *ast.StructType) int {
	if st.Fields == nil {
		return 0
	}

	count := 0

	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			count++
		} else {
			count += len(field.Names)
		}
	}

	return count
}

// hasCollectionField returns true if the struct has at least one slice
// or map field, indicating unbounded growth potential in an aggregate state.
func hasCollectionField(st *ast.StructType) bool {
	if st.Fields == nil {
		return false
	}

	for _, field := range st.Fields.List {
		switch t := field.Type.(type) {
		case *ast.ArrayType:
			if t.Len == nil { // slice — unbounded growth
				return true
			}
		case *ast.MapType:
			return true
		}
	}

	return false
}

// hasByteSliceField returns true if the struct has a []byte or []uint8 field.
func hasByteSliceField(st *ast.StructType) bool {
	if st.Fields == nil {
		return false
	}

	for _, field := range st.Fields.List {
		arr, ok := field.Type.(*ast.ArrayType)
		if !ok || arr.Len != nil { // only slices
			continue
		}

		elt, ok := arr.Elt.(*ast.Ident)
		if !ok {
			continue
		}

		if elt.Name == "byte" || elt.Name == "uint8" {
			return true
		}
	}

	return false
}

// projectUsesJSONEventCodec detects event-path-specific JSON codec usage.
// Unlike the broader projectUsesJSONCodec, this only returns true when
// JSONCodec is used in an event-encoding context:
//   - event.DefaultCodec = codec.JSONCodec{}
//   - event.WithCodec(codec.JSONCodec{})
//   - stack.WithEventCodec(codec.JSONCodec{})
func projectUsesJSONEventCodec(ctx *analyzer.AnalysisContext) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			// Check for JSONCodec in event/stack codec-related calls.
			call, ok := n.(*ast.CallExpr)
			if ok {
				sel, ok := analyzer.SelectorFromExpr(call.Fun)
				if ok {
					switch sel.Sel.Name {
					case "WithCodec", "WithEventCodec", "WithDefaultCodec":
						if callContainsJSONCodec(call) {
							found = true
							return false
						}
					}
				}
			}

			// Check for DefaultCodec assignment.
			assign, ok := n.(*ast.AssignStmt)
			if ok {
				for _, lhs := range assign.Lhs {
					sel, ok := lhs.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "DefaultCodec" {
						continue
					}
					if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "event" {
						if slices.ContainsFunc(assign.Rhs, exprReferencesJSONCodec) {
							found = true
							return false
						}
					}
				}
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

func callContainsJSONCodec(call *ast.CallExpr) bool {
	return slices.ContainsFunc(call.Args, exprReferencesJSONCodec)
}

func exprReferencesJSONCodec(expr ast.Expr) bool {
	// Composite literal: codec.JSONCodec{}
	if lit, ok := expr.(*ast.CompositeLit); ok {
		sel, ok := lit.Type.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == "JSONCodec" {
			return true
		}
	}

	return false
}

// typeName extracts the bare type name from an AST expression, unwrapping
// package qualifiers (e.g. models.UserState → UserState).
func typeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	}

	return ""
}

// extractStateTypeFromCall tries to determine the state type name from a
// decider.NewRepository or decider.NewTypedRepository call.
// It first checks explicit type parameters, then falls back to the decider
// argument if it is a composite literal (e.g. decider.Decider[State]{...}).
func extractStateTypeFromCall(call *ast.CallExpr) string {
	// 1. Explicit type parameters on the function itself.
	if stateType := extractStateType(call.Fun); stateType != "" {
		return stateType
	}

	// 2. Try the decider argument (3rd positional arg: store, bus, decider).
	if len(call.Args) >= 3 {
		if composite, ok := call.Args[2].(*ast.CompositeLit); ok {
			if stateType := extractStateType(composite.Type); stateType != "" {
				return stateType
			}
		}
	}

	return ""
}

// extractStateType extracts the first type parameter from a generic
// instantiation expression (IndexExpr or IndexListExpr).
func extractStateType(expr ast.Expr) string {
	switch f := expr.(type) {
	case *ast.IndexExpr: // NewRepository[State]
		return typeName(f.Index)
	case *ast.IndexListExpr: // NewTypedRepository[State, Cmd]
		if len(f.Indices) > 0 {
			return typeName(f.Indices[0])
		}
	}

	return ""
}
