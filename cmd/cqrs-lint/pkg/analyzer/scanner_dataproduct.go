package analyzer

import (
	"go/ast"
)

// scanDataProductDeclaration records one catalog.AddDataProduct call site:
// the product's ID and how many of its outputs carry an explicit Contract.
// The composite literal is inspected structurally — key-value elements for
// ID/Outputs, and within each output element the presence of a Contract
// key. Products built through helpers (a variable passed instead of a
// literal) are invisible to this scanner and simply not recorded.
func scanDataProductDeclaration(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) {
	lit := firstCompositeLitArg(call)
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
