package architecture

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// sqlImportSignals are import paths whose presence in a package means raw
// SQL I/O: the stdlib surface plus the first-party driver set the fleet
// actually uses. Blank side-effect imports count too — registering a driver
// is still owning SQL.
var sqlImportSignals = map[string]bool{
	"database/sql":                   true,
	"modernc.org/sqlite":             true,
	"github.com/jackc/pgx/v5":        true,
	"github.com/jackc/pgx/v5/stdlib": true,
	"github.com/go-sql-driver/mysql": true,
	"github.com/mattn/go-sqlite3":    true,
	"github.com/lib/pq":              true,
}

// engineTierPrefixes are package-path prefixes that OWN their SQL legitimately
// (ADR-0142): the metaengine substrate and its engines, the claim cores, the
// queue engines, and the legacy storage/stack tiers that are already
// sentenced to v5 removal (flagging them would be noise about a known,
// dated decision — the deletion waves remove them).
var engineTierPrefixes = []string{
	"github.com/larsartmann/go-cqrs-lite/metaengine/",
	"github.com/larsartmann/go-cqrs-lite/claiming/",
	"github.com/larsartmann/go-cqrs-lite/queue/sqlite/",
	"github.com/larsartmann/go-cqrs-lite/queue/postgres/",
	"github.com/larsartmann/go-cqrs-lite/queue/mysql/",
	"github.com/larsartmann/go-cqrs-lite/storage/",
	"github.com/larsartmann/go-cqrs-lite/stack/",
	"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/",
	"github.com/larsartmann/go-cqrs-lite/testutil/",
	"github.com/larsartmann/go-cqrs-lite/example/",
}

// E023: satellite-owned SQL beside go-cqrs-lite usage.
// The all-durable-I/O-through-metaengine rule (ADR-0142 amendment 2026-10-24,
// AGENTS contract 30) makes the engines the only storage layer: a package
// that imports go-cqrs-lite modules AND database/sql (or a SQL driver) is
// carrying a second, satellite persistence path — its own schema, its own
// claim protocol, its own drift surface. Advisory by design: consumers may
// own non-CQRS tables; the finding asks them to route DURABLE STATE through
// metaengine engines (or the system composition root), not to delete every
// database/sql import. Engine-tier modules are exempt by prefix.
//
//nolint:ireturn // factory returns public interface
func NewE023Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E023-satellite-owned-sql",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest || isEngineTierPackage(packagePathOf(gf)) {
					continue
				}

				if !packageMixesLibraryAndSQL(ctx, gf) {
					continue
				}

				for _, imp := range sqlImportsOf(gf) {
					if f, err := e023Finding(ctx, gf, imp); err == nil {
						findings = append(findings, f)
					}
				}
			}

			return findings, nil
		},
	)
}

// packagePathOf returns the file's package import path, empty when unknown
// (syntax-only loads, unit-test source contexts).
func packagePathOf(gf *analyzer.GoFile) string {
	if gf == nil || gf.Pkg == nil {
		return ""
	}

	return gf.Pkg.PkgPath
}

// isEngineTierPackage reports whether pkgPath belongs to a module that owns
// SQL legitimately (substrate, engines, or the v5-sentenced legacy tiers).
func isEngineTierPackage(pkgPath string) bool {
	if pkgPath == "" {
		return false
	}

	for _, prefix := range engineTierPrefixes {
		if strings.HasPrefix(pkgPath, prefix) {
			return true
		}
	}

	return false
}

// sqlImportsOf returns the file's SQL-signal import specs.
func sqlImportsOf(gf *analyzer.GoFile) []*ast.ImportSpec {
	if gf == nil || gf.AST == nil {
		return nil
	}

	var specs []*ast.ImportSpec

	for _, imp := range gf.AST.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if sqlImportSignals[path] {
			specs = append(specs, imp)
		}
	}

	return specs
}

// packageMixesLibraryAndSQL reports whether the package owning gf imports at
// least one go-cqrs-lite module somewhere in its non-test files. The union is
// computed across files of the same package (gf.Pkg); when the package is
// unknown (syntax-only context), the whole context serves as the union so the
// mixed-usage signal still fires in unit-test source contexts.
func packageMixesLibraryAndSQL(ctx *analyzer.AnalysisContext, gf *analyzer.GoFile) bool {
	files := ctx.GoFiles

	if gf != nil && gf.Pkg != nil {
		files = nil
		for _, candidate := range ctx.GoFiles {
			if candidate.Pkg == gf.Pkg {
				files = append(files, candidate)
			}
		}
	}

	for _, file := range files {
		if file.IsTest || file.AST == nil {
			continue
		}

		for _, imp := range file.AST.Imports {
			if analyzer.IsCQRSModulePath(strings.Trim(imp.Path.Value, `"`)) {
				return true
			}
		}
	}

	return false
}

// e023Finding builds the finding for one satellite SQL import, anchored at
// the import spec.
func e023Finding(
	ctx *analyzer.AnalysisContext,
	gf *analyzer.GoFile,
	imp *ast.ImportSpec,
) (finding.Finding, error) {
	pos := ctx.Fset.Position(imp.Pos())

	return findingTemplate.Builder(
		"E023",
		"Package mixes go-cqrs-lite usage with satellite-owned database/sql",
		finding.SeverityInfo,
		finding.Pos(finding.FilePath(gf.Path), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryStructure).
		WithConfidence(finding.ConfidenceHigh).
		WithSuggestion(
			"Route durable state through metaengine engines (ADR-0142 I/O rule): " +
				"collections/planned tables for read models, SnapshotBackend for snapshots, " +
				"DueClaimer/DedupStore/FactSink for timers/dedup/queues — or compose via " +
				"system.New so operators pick engines at deployment time. Non-CQRS tables " +
				"may dismiss this advisory; engine-tier modules are exempt by design",
		).
		WithSnippet(ctx.SourceLine(gf.Path, pos.Line)).
		Build()
}
