package api

import (
	"go/ast"

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

			recvType := analyzer.BaseTypeName(fn.Recv.List[0].Type)
			if recvType == cmd.Name {
				return true
			}
		}
	}

	return false
}
