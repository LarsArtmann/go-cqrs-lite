package adoption

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// F032 detects projects that import any go-cqrs-lite module but never import
// the catalog module. Owner policy (2026-11-03): using the library means
// documenting the domain — every consumer generates its catalog (AsyncAPI,
// OpenAPI, D2, EventCatalog exports) from declared truth instead of leaving
// the vocabulary undocumented. Error severity by design: the finding blocks
// CI, not just reports; import detection is deterministic, hence confidence
// high.
//
// Satisfied by ANY catalog-family import: the catalog module itself (any
// package) or systemcatalog, which derives a full catalog from a
// system.DomainConfig and feeds a catalog.Builder — a system-first app has
// catalog documentation without importing catalog directly. F002 remains the
// complementary second step (imported catalog but never calls NewBuilder
// with 3+ event types). Test-only usage does not count: a project that only
// exercises cqrs test utilities owns no production domain to document.
//
//nolint:ireturn // factory returns public interface
func NewF032Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F032-no-catalog-import",
		func(_ context.Context) ([]finding.Finding, error) {
			gf, imp, ok := firstLibraryImport(ctx)
			if !ok {
				return nil, nil
			}

			if importsPathIn(ctx.GoFiles, "go-cqrs-lite/catalog", "go-cqrs-lite/systemcatalog") {
				return nil, nil
			}

			pos := ctx.Fset.Position(imp.Pos())

			f, err := findingTemplate.Builder(
				"F032",
				"Project uses go-cqrs-lite but never imports go-cqrs-lite/catalog — "+
					"catalog-driven documentation is not wired",
				finding.SeverityError,
				finding.Pos(finding.FilePath(gf.Path), pos.Line, pos.Column),
			).
				WithConfidence(finding.ConfidenceHigh).
				WithFixStrategy(finding.FixStrategySuggest).
				WithSuggestion(
					"Import github.com/larsartmann/go-cqrs-lite/catalog/v4 and build your " +
						"documentation: catalog.NewBuilder() (or the catalog/simple.New wrapper) " +
						"declares commands/events/queries and exports AsyncAPI, OpenAPI, D2, and " +
						"EventCatalog docs. System-first apps derive the whole catalog from declared " +
						"truth via systemcatalog.FromDomainConfig / DeriveInto.",
				).
				WithSnippet(ctx.SourceLine(gf.Path, pos.Line)).
				Build()
			if err != nil {
				return nil, nil
			}

			return []finding.Finding{f}, nil
		},
	)
}

// firstLibraryImport returns the first non-test import of a go-cqrs-lite
// module that is NOT part of the catalog family, anchored at its import
// spec — the line where the project's library usage starts. Catalog-family
// imports alone are not usage: a project importing only catalog owns no
// library usage the rule may coach.
func firstLibraryImport(
	ctx *analyzer.AnalysisContext,
) (*analyzer.GoFile, *ast.ImportSpec, bool) {
	for _, gf := range ctx.GoFiles {
		if gf == nil || gf.IsTest || gf.AST == nil {
			continue
		}

		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}

			path := strings.Trim(imp.Path.Value, `"`)
			if analyzer.IsCQRSModulePath(path) && !isCatalogFamilyPath(path) {
				return gf, imp, true
			}
		}
	}

	return nil, nil, false
}

// isCatalogFamilyPath reports whether path belongs to the catalog
// documentation family: the catalog module (any package — the fragment
// matches catalog/v4 and its subpackages) or the systemcatalog derivation
// module (systemcatalog never contains "go-cqrs-lite/catalog" as a
// substring, so it needs its own check).
func isCatalogFamilyPath(path string) bool {
	return analyzer.PathContainsFragment(path, "go-cqrs-lite/catalog") ||
		analyzer.PathContainsFragment(path, "go-cqrs-lite/systemcatalog")
}
