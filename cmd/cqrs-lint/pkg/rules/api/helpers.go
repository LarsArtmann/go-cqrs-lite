package api

import (
	"go/ast"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// hasMethod checks if a command type has a method with the given name.
func hasMethod(ctx *analyzer.AnalysisContext, cmd analyzer.CommandInfo, methodName string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.Path != cmd.File || gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil {
				continue
			}

			if fn.Name.Name != methodName {
				continue
			}

			recvType := baseTypeName(fn.Recv.List[0].Type)
			if recvType == cmd.Name {
				return true
			}
		}
	}

	return false
}

func baseTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return baseTypeName(t.X)
	case *ast.IndexExpr:
		return baseTypeName(t.X)
	}

	return ""
}

func selectorPackage(sel *ast.SelectorExpr) string {
	if ident, ok := sel.X.(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}

func extractJSONTag(tag string) string {
	idx := strings.Index(tag, `json:"`)
	if idx < 0 {
		return ""
	}

	start := idx + len(`json:"`)

	end := strings.Index(tag[start:], `"`)
	if end < 0 {
		return ""
	}

	value := tag[start : start+end]
	if comma := strings.Index(value, ","); comma >= 0 {
		value = value[:comma]
	}

	return value
}
