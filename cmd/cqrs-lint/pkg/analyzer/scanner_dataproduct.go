package analyzer

import (
	"go/ast"
)

// scanDataProductDeclaration records one catalog.AddDataProduct call site:
// the product's ID and how many of its outputs carry an explicit Contract.
// The composite literal is inspected structurally — key-value elements for
// ID/Outputs, and within each output element the presence of a Contract
// key. A single-variable indirection in the SAME file
// (`dp := catalog.DataProduct{...}; reg.AddDataProduct(dp)`) is resolved to
// its literal; cross-file variables and helper-built products remain
// invisible to this scanner and are simply not recorded.
func scanDataProductDeclaration(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) {
	lit := dataProductLiteralFor(gf, call)
	if lit == nil {
		return
	}

	pos := ctx.Fset.Position(call.Pos())

	info := DataProductInfo{
		File:   gf.Path,
		Line:   pos.Line,
		Column: pos.Column,
	}

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}

		switch key.Name {
		case "ID":
			info.Name = StringLit(kv.Value)
		case "Outputs":
			info.OutputCount, info.Contracted = countContractedOutputs(kv.Value)
		}
	}

	ctx.Registry.DataProducts = append(ctx.Registry.DataProducts, info)
}

// firstCompositeLitArg returns the first argument that is a composite
// literal (AddDataProduct's single payload argument).
func firstCompositeLitArg(call *ast.CallExpr) *ast.CompositeLit {
	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.CompositeLit); ok {
			return lit
		}
	}

	return nil
}

// dataProductLiteralFor returns the payload literal of an AddDataProduct
// call, resolving ONE level of same-file variable indirection: when the
// argument is an identifier, its `:=`/`var` declaration in the same file is
// consulted for the composite literal (13-32 §f21: variable-passed products
// were invisible to E019). Later re-assignments are NOT tracked — the first
// declared literal wins, matching the single-declaration style the catalog
// recipes teach.
func dataProductLiteralFor(gf *GoFile, call *ast.CallExpr) *ast.CompositeLit {
	if lit := firstCompositeLitArg(call); lit != nil {
		return lit
	}

	for _, arg := range call.Args {
		ident, ok := arg.(*ast.Ident)
		if !ok {
			continue
		}

		var found *ast.CompositeLit

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found != nil {
				return false
			}

			switch stmt := n.(type) {
			case *ast.AssignStmt:
				for i, lhs := range stmt.Lhs {
					if id, ok := lhs.(*ast.Ident); ok && id.Name == ident.Name &&
						i < len(stmt.Rhs) {
						if cl, ok := stmt.Rhs[i].(*ast.CompositeLit); ok {
							found = cl

							return false
						}
					}
				}
			case *ast.ValueSpec:
				for i, name := range stmt.Names {
					if name.Name != ident.Name || i >= len(stmt.Values) {
						continue
					}

					if cl, ok := stmt.Values[i].(*ast.CompositeLit); ok {
						found = cl

						return false
					}
				}
			}

			return true
		})

		if found != nil {
			return found
		}
	}

	return nil
}

// argIsCatalogDataProduct reports whether the call's payload argument is a
// composite literal explicitly typed catalog.DataProduct — the fallback
// signal when the receiver cannot be resolved to the catalog package (a
// Registry method value whose qualifier is a plain variable ident).
func argIsCatalogDataProduct(call *ast.CallExpr) bool {
	lit := firstCompositeLitArg(call)
	if lit == nil {
		return false
	}

	sel, ok := lit.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	pkg, ok := sel.X.(*ast.Ident)

	return ok && pkg.Name == "catalog" && sel.Sel.Name == "DataProduct"
}

// countContractedOutputs counts the elements of an Outputs literal and how
// many carry a non-nil Contract key.
func countContractedOutputs(expr ast.Expr) (total, contracted int) {
	outputs, ok := expr.(*ast.CompositeLit)
	if !ok {
		return 0, 0
	}

	for _, elt := range outputs.Elts {
		total++

		output, ok := elt.(*ast.CompositeLit)
		if !ok {
			continue
		}

		for _, field := range output.Elts {
			kv, ok := field.(*ast.KeyValueExpr)
			if !ok {
				continue
			}

			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Contract" {
				if _, isNil := kv.Value.(*ast.Ident); !isNil {
					contracted++
				}

				break
			}
		}
	}

	return total, contracted
}
