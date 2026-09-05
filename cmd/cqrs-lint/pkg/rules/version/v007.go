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
// Skipped in library self-lint mode: the library legitimately references its
// own deprecated surfaces while they exist (shims, tests, forwarders).

// cqrsModulePrefix is the go-cqrs-lite module path prefix every consumer
// import shares.
const cqrsModulePrefix = "github.com/larsartmann/go-cqrs-lite/"

// deprecatedV5Module describes a module removed entirely at v5.
type deprecatedV5Module struct {
	fragment    string // module path relative to go-cqrs-lite/ (no /vN suffix)
	replacement string
}

var deprecatedV5Modules = []deprecatedV5Module{ //nolint:gochecknoglobals // static table
	// ADR-0123: stack presets replaced by system.New.
	{fragment: "stack/memory", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/sqlite", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/pebble", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/bbolt", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/duckdb", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/postgres", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/mysql", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/turso", replacement: "system.New with DomainConfig + DeploymentConfig"},
	// ADR-0123: relational + view tiers absorbed into metaengine engines.
	{fragment: "storage/relational", replacement: "metaengine engines with layout planning"},
	{fragment: "storage/view", replacement: "metaengine engines with layout planning"},
}

// deprecatedV5Symbol describes one deprecated symbol inside a module that
// otherwise survives v5.
type deprecatedV5Symbol struct {
	fragment    string // module path relative to go-cqrs-lite/
	symbol      string // exported identifier
	replacement string
}

var deprecatedV5Symbols = []deprecatedV5Symbol{ //nolint:gochecknoglobals // static table
	// ADR-0123: stack root — Bundle + Materialize + projection runner.
	{fragment: "stack", symbol: "Bundle", replacement: "system.New composition"},
	{fragment: "stack", symbol: "New", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack", symbol: "Materialize", replacement: "metaengine auto-projection"},
	{fragment: "stack", symbol: "NewMaterialize", replacement: "metaengine auto-projection"},
	{fragment: "stack", symbol: "RunProjections", replacement: "projectionhost.Host"},
	{fragment: "stack", symbol: "TombstonePolicy", replacement: "event-type-driven deletion (ADR-0114)"},
	{fragment: "stack", symbol: "IncludeTombstoned", replacement: "event-type-driven deletion (ADR-0114)"},
	{fragment: "stack", symbol: "ExcludeTombstoned", replacement: "event-type-driven deletion (ADR-0114)"},
	{fragment: "stack", symbol: "OnlyTombstoned", replacement: "event-type-driven deletion (ADR-0114)"},
	// ADR-0123: graph projection tier (GraphDriver/GraphSink survive via graphadapter).
	{fragment: "graph", symbol: "GraphProjection", replacement: "metaengine auto-projection + graphadapter"},
	{fragment: "graph", symbol: "NewGraphProjection", replacement: "metaengine auto-projection + graphadapter"},
	// ADR-0126: deprecated store/journal shells.
	{fragment: "schema", symbol: "VersionedStore", replacement: "schema.UpcastSourceTransform + event.DecorateStore"},
	{fragment: "schema", symbol: "NewVersionedStore", replacement: "schema.UpcastSourceTransform + event.DecorateStore"},
	{fragment: "schema", symbol: "VersionedSeekableJournal", replacement: "schema.UpcastSourceTransform + event.DecorateJournal"},
	{fragment: "schema", symbol: "NewVersionedSeekableJournal", replacement: "schema.UpcastSourceTransform + event.DecorateJournal"},
	// ADR-0126: signing middleware shells.
	{fragment: "signing", symbol: "RejectingPublishMiddleware", replacement: "event.RejectingPublishMiddleware"},
	{fragment: "signing", symbol: "RejectingHandlerMiddleware", replacement: "event.RejectingHandlerMiddleware"},
	// ADR-0126: encryption error shells.
	{fragment: "encryption", symbol: "ErrInnerStoreNotJournal", replacement: "event.ErrInnerStoreNotJournal"},
	{fragment: "encryption", symbol: "ErrInnerStoreNotSeekable", replacement: "event.ErrInnerStoreNotSeekable"},
	{fragment: "encryption", symbol: "ErrInnerStoreNotBackwards", replacement: "event.ErrInnerStoreNotBackwards"},
	// ADR-0126: metadata CustomData alias.
	{fragment: "metadata", symbol: "CustomData", replacement: "metadata.Metadata[K]"},
	// ADR-0114: tombstones replaced by domain events.
	{fragment: "event", symbol: "DetectTombstone", replacement: "domain events for deletion (docs/migration/tombstone-to-domain-events.md)"},
	{fragment: "event", symbol: "MarkTombstone", replacement: "domain events for deletion (docs/migration/tombstone-to-domain-events.md)"},
	{fragment: "event", symbol: "MarkRebirth", replacement: "domain events for restore (docs/migration/tombstone-to-domain-events.md)"},
	{fragment: "event", symbol: "MetadataKeyTombstone", replacement: "domain events for deletion"},
	{fragment: "event", symbol: "MetadataKeyRebirth", replacement: "domain events for restore"},
	{fragment: "event", symbol: "TombstoneMark", replacement: "domain events for deletion"},
	{fragment: "event", symbol: "TombstoneStatus", replacement: "listing.StreamStatus or domain events"},
	{fragment: "event", symbol: "TombstoneActive", replacement: "listing.StreamStatus or domain events"},
	{fragment: "event", symbol: "TombstoneTombstoned", replacement: "listing.StreamStatus or domain events"},
	{fragment: "event", symbol: "TombstoneUndetermined", replacement: "listing.StreamStatus or domain events"},
	{fragment: "event", symbol: "EnsureCustom", replacement: "event.Metadata.WithCustom"},
	// ADR-0126: metadata in-place mutation.
	{fragment: "metadata", symbol: "EnsureCustom", replacement: "metadata.WithCustom"},
}

// adrForFragment maps a module fragment to the ADR documenting the removal.
func adrForFragment(fragment string) string {
	switch {
	case strings.HasPrefix(fragment, "stack"), strings.HasPrefix(fragment, "storage/"), fragment == "graph":
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

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					sel, ok := n.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					ident, ok := sel.X.(*ast.Ident)
					if !ok {
						return true // method call on a value, not a package ref
					}

					path, ok := resolveQualifier(gf.AST, ident.Name)
					if !ok {
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
			if lastPathSegment(path) == qualifier {
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

// stripVersionSuffix removes a trailing single-digit major-version suffix
// ("/v2" .. "/v9") from a module path.
func stripVersionSuffix(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return path
	}

	seg := path[idx+1:]
	if len(seg) == 2 && seg[0] == 'v' && seg[1] >= '2' && seg[1] <= '9' {
		return path[:idx]
	}

	return path
}

// lastPathSegment returns the final slash-separated segment of an import
// path (the default package qualifier).
func lastPathSegment(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}

	return path
}
