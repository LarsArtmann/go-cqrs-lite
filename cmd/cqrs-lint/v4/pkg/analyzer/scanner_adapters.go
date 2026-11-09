package analyzer

import (
	"go/ast"
	"strings"
)

// ResolveTransportAdapters scans all files for methods that convert transport
// adapter types to domain types (e.g. toDomain, ToDomain, toCommand). When a
// command type has such a conversion method, it is flagged as a transport
// adapter — it implements command.Command for compile-time type safety but is
// never dispatched directly. Detectors C002/A001/E005 skip transport adapters.
//
// This must run as a post-pass after all files have been scanned so that
// cross-file references (struct in one file, method in another) resolve.
func ResolveTransportAdapters(ctx *AnalysisContext) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil {
				continue
			}

			if !isConversionMethod(fn.Name.Name) {
				continue
			}

			recvType := recvTypeName(fn)
			if recvType == "" {
				continue
			}

			if cmd := ctx.Registry.CommandByName(recvType); cmd != nil {
				cmd.TransportAdapter = true
			}
		}
	}
}

// isConversionMethod reports whether a method name looks like a transport-to-
// domain conversion (toDomain, ToDomain, toCommand, ToCommand, asDomainCmd).
func isConversionMethod(name string) bool {
	lower := strings.ToLower(name)

	return lower == "todomain" ||
		lower == "tocommand" ||
		lower == "asdomaincmd" ||
		lower == "asdomaincommand" ||
		lower == "todomaincmd" ||
		lower == "todomaincommand"
}
