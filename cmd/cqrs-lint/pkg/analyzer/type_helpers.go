package analyzer

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ReceiverTypeName resolves the fully-qualified type name of a method call's
// receiver using type info. Returns "" when type info is unavailable or the
// receiver cannot be resolved.
//
// Example: for `errorBus.SubscribeAll()`, returns the type of `errorBus`,
// e.g. "*github.com/user/repo.ErrorBus".
func ReceiverTypeName(pkg *packages.Package, call *ast.CallExpr) string {
	if pkg == nil || pkg.TypesInfo == nil {
		return ""
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	tv, ok := pkg.TypesInfo.Types[sel.X]
	if !ok || tv.Type == nil {
		return ""
	}

	return tv.Type.String()
}

// IsEventBusType reports whether a fully-qualified type string looks like a
// go-cqrs-lite event bus type (Bus, MemoryBus, etc. from the event module).
// Returns true for empty strings (can't resolve — conservatively assume yes).
func IsEventBusType(typeStr string) bool {
	if typeStr == "" {
		return true
	}

	return strings.Contains(typeStr, "cqrs-lite/event/")
}

// ReceiverIsEventBus checks whether a method call's receiver is a go-cqrs-lite
// event bus. Returns true when type info is unavailable (conservative — assumes
// yes to preserve current behavior when types can't be resolved).
func ReceiverIsEventBus(pkg *packages.Package, call *ast.CallExpr) bool {
	return IsEventBusType(ReceiverTypeName(pkg, call))
}
