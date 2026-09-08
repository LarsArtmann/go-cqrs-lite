package version

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// V007: v5-removed API usage.
//
// Detects consumer code referencing go-cqrs-lite symbols that are removed at
// v5, so projects get a migration window instead of a broken build at the
// major bump. Two granularities:
//
//  1. Wholly-removed modules (ADR-0123): every stack/* preset, and the
//     storage/relational and storage/view tiers. Any import-qualified
//     reference into these modules fires.
//  2. Deprecated symbols inside partially-deprecated modules (ADR-0114,
//     ADR-0123, ADR-0126): e.g. stack.Materialize, schema.VersionedStore,
//     signing.RejectingPublishMiddleware, event.DetectTombstone.
//
// The transport/* modules are intentionally NOT covered here — F030
// (deprecated-transport-import) owns that surface at import granularity.
//
// Coverage contract: the removal surface (deprecatedV5Modules plus
// deprecatedV5Symbols) is held against the repo's actual `Deprecated:` …v5
// markers by the drift meta-tests in v007_drift_test.go. A new v5 removal
// without a table (or allowlist) entry fails the suite, and a stale table
// entry outliving its symbol fails the reverse check. Fragments are
// normalized with stripVersionSuffix, so subpackages of versioned modules
// (storage/v4/relational) map to the same fragment space as module roots.
//
// Skipped in library self-lint mode: the library legitimately references its
// own deprecated surfaces while they exist (shims, tests, forwarders).

// adrForFragment maps a module fragment to the ADR documenting the removal.
func adrForFragment(fragment string) string {
	switch {
	case strings.HasPrefix(fragment, "stack"),
		strings.HasPrefix(fragment, "storage/"),
		fragment == "graph":
		return "ADR-0123"
	case fragment == "event" || fragment == "metadata":
		return "ADR-0114/0126"
	default:
		return "ADR-0126"
	}
}

//nolint:ireturn // factory returns public interface
func NewV007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"V007-v5-removed-api-usage",
		func(_ context.Context) ([]finding.Finding, error) {
			if ctx.IsLibrarySelfLint() {
				return nil, nil
			}

			var out []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				checkDotImports(ctx, gf, &out)
				checkDotImportedRemovedSymbols(ctx, gf, &out)

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					sel, ok := n.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					ident, ok := sel.X.(*ast.Ident)
					if !ok {
						return true // method call on a value, not a package ref
					}

					// F091 Tier 1: resolve the qualifier through the type checker
					// first (exact, shadow-proof); fall back to the import-table
					// scan ONLY when type info is unavailable (broken builds) —
					// a typed "not a package" answer is final.
					path, known := analyzer.ResolveQualifierTyped(gf, ident)
					if !known {
						path, _ = resolveQualifier(gf.AST, ident.Name)
					}
					if path == "" {
						return true
					}

					module, ok := cqrsModuleOf(path)
					if !ok {
						return true
					}

					symbol := sel.Sel.Name

					if entry, hit := matchModule(module); hit {
						out = append(out, v007Finding(ctx, sel, ident.Name,
							fmt.Sprintf("%s.%s (whole module)", module, symbol),
							entry.replacement, adrForFragment(module)))
						return true
					}

					if entry, hit := matchSymbol(module, symbol); hit {
						out = append(out, v007Finding(ctx, sel, ident.Name,
							module+"."+symbol, entry.replacement, adrForFragment(module)))
					}

					return true
				})
			}

			return out, nil
		},
	)
}

// checkDotImports flags dot-imports of go-cqrs-lite modules (F090). A
// dot-import leaves v5-removed-API references qualifier-less, so the
// selector-based checks below cannot attribute them — flagging the import
// itself closes that hole with a message that teaches the fix. The finding
// fires on any go-cqrs-lite module, not just removed ones: the hole exists
// the moment the import is dot-shaped, and dot-importing library internals
// is rare and already discouraged.
func checkDotImports(ctx *analyzer.AnalysisContext, gf *analyzer.GoFile, out *[]finding.Finding) {
	for _, imp := range gf.AST.Imports {
		if imp == nil || imp.Name == nil || imp.Name.Name != "." || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)
		module, ok := cqrsModuleOf(path)
		if !ok {
			continue
		}

		*out = append(*out, v007DotImportFinding(ctx, imp, module, path))
	}
}

// v007DotImportFinding builds the F090 finding for one dot-import spec.
func v007DotImportFinding(
	ctx *analyzer.AnalysisContext,
	imp *ast.ImportSpec,
	module, path string,
) finding.Finding {
	pos := ctx.Fset.Position(imp.Pos())

	f, _ := finding.NewBuilder(
		"V007",
		"cqrs-lint",
		fmt.Sprintf(
			"dot-import of go-cqrs-lite module %s hides v5-removed-API usage from this linter — name the import",
			module,
		),
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryBestPractice).
		WithConfidence(finding.ConfidenceHigh).
		WithSuggestion(fmt.Sprintf(
			"Replace the dot-import of %s with a named or default import so V007 can attribute removed-API usage",
			path,
		)).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()

	return f
}

// checkDotImportedRemovedSymbols is F090(b): with type information
// available, bare identifiers that resolve into a dot-imported
// go-cqrs-lite module are attributed to that module, so removed symbols
// referenced WITHOUT a qualifier (the hole F090(a) warns about) still fire
// with a precise position. Silent when typed confirmations are off or the
// file has no type info: a bare identifier's origin cannot be attributed by
// name alone, and a false attribution is worse than none.
func checkDotImportedRemovedSymbols(ctx *analyzer.AnalysisContext, gf *analyzer.GoFile, out *[]finding.Finding) {
	if !ctx.TypedConfirmations() || gf.Pkg == nil || gf.Pkg.TypesInfo == nil {
		return
	}

	// import path → module fragment, for this file's cqrs dot-imports only.
	dotted := make(map[string]string)
	for _, imp := range gf.AST.Imports {
		if imp == nil || imp.Name == nil || imp.Name.Name != "." || imp.Path == nil {
			continue
		}
		path := strings.Trim(imp.Path.Value, `"`)
		if module, ok := cqrsModuleOf(path); ok {
			dotted[path] = module
		}
	}
	if len(dotted) == 0 {
		return
	}

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		obj, ok := gf.Pkg.TypesInfo.Uses[ident]
		if !ok || obj == nil || !obj.Exported() {
			return true
		}
		pkg := obj.Pkg()
		if pkg == nil {
			return true // builtin or universe scope
		}

		module, ok := dotted[pkg.Path()]
		if !ok {
			return true // identifier belongs to another package/import
		}

		if m, removed := matchModule(module); removed {
			*out = append(*out, v007BareIdentFinding(ctx, ident, module, ident.Name, m.replacement))
			return true
		}
		if s, hit := matchSymbol(module, ident.Name); hit {
			*out = append(*out, v007BareIdentFinding(ctx, ident, module, ident.Name, s.replacement))
		}

		return true
	})
}

// v007BareIdentFinding builds the F090(b) finding for one dot-imported
// removed-symbol reference.
func v007BareIdentFinding(
	ctx *analyzer.AnalysisContext,
	ident *ast.Ident,
	module, symbol, replacement string,
) finding.Finding {
	pos := ctx.Fset.Position(ident.Pos())

	f, _ := finding.NewBuilder(
		"V007",
		"cqrs-lint",
		fmt.Sprintf(
			"%s (dot-imported from %s) is removed at v5 — replace with %s; name the import so V007 can attribute usage",
			symbol, module, replacement,
		),
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryBestPractice).
		WithConfidence(finding.ConfidenceHigh).
		WithSuggestion(fmt.Sprintf(
			"Import %s by name and migrate off %s before the v5 cut",
			module, symbol,
		)).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()

	return f
}

// v007Finding builds one V007 finding for the given selector position.
func v007Finding(
	ctx *analyzer.AnalysisContext,
	sel *ast.SelectorExpr,
	qualifier, target, replacement, adr string,
) finding.Finding {
	pos := ctx.Fset.Position(sel.Pos())

	f, _ := finding.NewBuilder(
		"V007",
		"cqrs-lint",
		fmt.Sprintf("%s is removed at v5 (%s) — replace with %s", target, adr, replacement),
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryBestPractice).
		WithConfidence(finding.ConfidenceHigh).
		WithSuggestion(fmt.Sprintf(
			"Migrate off %s.%s before the v5 cut; see docs/adr (%s)",
			qualifier, sel.Sel.Name, adr,
		)).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()

	return f
}

// matchModule looks up a wholly-removed module by fragment.
func matchModule(module string) (deprecatedV5Module, bool) {
	for _, m := range deprecatedV5Modules {
		if m.fragment == module {
			return m, true
		}
	}

	return deprecatedV5Module{}, false
}

func matchSymbol(module, symbol string) (deprecatedV5Symbol, bool) {
	for _, s := range deprecatedV5Symbols {
		if s.fragment == module && s.symbol == symbol {
			return s, true
		}
	}

	return deprecatedV5Symbol{}, false
}

// resolveQualifier maps a package qualifier to its import path using the
// file's import declarations (alias-aware). Blank and dot imports are
// skipped: a dot-imported package has no qualifier, and matching one would
// falsely attribute unrelated selectors.
func resolveQualifier(file *ast.File, qualifier string) (string, bool) {
	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)

		if imp.Name == nil {
			if defaultQualifier(path) == qualifier {
				return path, true
			}

			continue
		}

		switch imp.Name.Name {
		case "_", ".":
			continue
		case qualifier:
			return path, true
		}
	}

	return "", false
}

// cqrsModuleOf strips the go-cqrs-lite prefix and major-version suffix from
// an import path, returning the module fragment (e.g. "stack/sqlite").
// Non-go-cqrs-lite paths return ok=false.
func cqrsModuleOf(importPath string) (string, bool) {
	rest, ok := strings.CutPrefix(importPath, cqrsModulePrefix)
	if !ok {
		return "", false
	}

	return stripVersionSuffix(rest), true
}

// stripVersionSuffix removes every major-version segment ("/v2".."/v9")
// from a module path. Both module-root imports ("stack/sqlite/v4") and
// subpackages of versioned modules ("storage/v4/relational") must normalize
// to the table fragment ("stack/sqlite", "storage/relational"): the version
// segment sits mid-path whenever a module carries more than one package.
func stripVersionSuffix(path string) string {
	segs := strings.Split(path, "/")
	kept := segs[:0]
	for _, seg := range segs {
		if len(seg) == 2 && seg[0] == 'v' && seg[1] >= '2' && seg[1] <= '9' {
			continue
		}

		kept = append(kept, seg)
	}

	return strings.Join(kept, "/")
}

// defaultQualifier returns the package qualifier an unaliased import binds:
// the last path segment, with a major-version suffix ("/v2".."/v9") stripped.
func defaultQualifier(path string) string {
	return lastPathSegment(stripVersionSuffix(path))
}

// lastPathSegment returns the final slash-separated segment of an import
// path (the default package qualifier).
func lastPathSegment(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}

	return path
}
