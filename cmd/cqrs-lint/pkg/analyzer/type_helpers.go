package analyzer

import (
	"go/ast"
	"go/types"
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

// ResolveQualifierTyped resolves a package-qualifier identifier to its import
// path through the type checker (F091 Tier 1). Where the import-table scan
// string-matches a qualifier against import declarations, this asks the
// compiler's own resolution: Uses[ident] must be the PkgName that the import
// bound. Shadowed qualifiers (a local variable named like the package) stop
// matching by construction, and the answer is exact for aliased and
// versioned paths alike.
//
// The second result is authoritative-resolution, not success:
//
//   - resolved=false  → type info is unavailable (non-compiling project,
//     syntax-only load). The caller MUST fall back to its string-based scan
//     so partial results survive broken builds.
//   - resolved=true   → the answer is final. path is empty when ident does
//     not reference a package (it is a value/type shadowing the name) — the
//     caller must NOT fall back in this case, or the shadow dies.
func ResolveQualifierTyped(gf *GoFile, ident *ast.Ident) (path string, resolved bool) {
	if gf == nil || gf.Pkg == nil || gf.Pkg.TypesInfo == nil || ident == nil {
		return "", false
	}

	obj, ok := gf.Pkg.TypesInfo.Uses[ident]
	if !ok || obj == nil {
		return "", false
	}

	pkgName, isPkg := obj.(*types.PkgName)
	if !isPkg {
		return "", true
	}

	if pkgName.Imported() == nil {
		return "", false
	}

	return pkgName.Imported().Path(), true
}
