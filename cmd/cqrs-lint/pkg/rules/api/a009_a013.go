package api

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-finding"
)

// A009: Missing stack preset.
// Detects projects that don't use any stack/ preset (stack/sqlite, stack/pebble, etc.).
//
//nolint:ireturn // factory returns public interface
func NewA009Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A009-missing-stack-preset",
		func(_ context.Context) ([]finding.Finding, error) {
			hasStackPreset := false
			hasStorageFacade := false

			for _, pkg := range ctx.Packages {
				for _, imp := range pkg.Imports {
					if imp == nil {
						continue
					}

					if strings.Contains(imp.PkgPath, "go-cqrs-lite/stack/") {
						hasStackPreset = true
					}

					// The system/ composition root (ADR-0123: system.New auto-wires
					// store, bus, projections, queries) supersedes stack/ presets —
					// the sanctioned composition path. Importing it is adoption,
					// not a missing preset (nsfw-classifier feedback, 2026-10-03;
					// presets are legacy and removed at v5).
					if strings.Contains(imp.PkgPath, "go-cqrs-lite/system/") {
						hasStackPreset = true
					}

					// Using the storage/ facade directly (NewSQLBackend, RelationalProjection,
					// SQLViewStore, ...) signals an intentional custom-wiring architecture
					// (shared *sql.DB across CQRS + relational reads) that stack/ presets
					// don't support. Suppress A009 in that case — it's not a missing-preset
					// mistake, it's a deliberate design choice.
					if strings.Contains(imp.PkgPath, "go-cqrs-lite/storage") {
						hasStorageFacade = true
					}
				}
			}

			if hasStackPreset || hasStorageFacade {
				return nil, nil
			}

			var findings []finding.Finding

			// Evaluate per-module: use the primary module's store backend
			// for the suggestion text, resolved via ProfileForFile for
			// consistency with other per-module detectors.
			primaryStore := ctx.ProfileForFile(ctx.ProjectRoot + "/go.mod").Store
			suggestion := "Use stack/sqlite.New(dsn) or stack/pebble.New(dir) for one-call setup with sane defaults"
			switch primaryStore {
			case analyzer.StoreSQLite:
				suggestion = "Use stack/sqlite.New(dsn) for one-call setup with sane defaults"
			case analyzer.StorePostgres:
				suggestion = "Use stack/postgres.New(dsn) for one-call setup with sane defaults"
			case analyzer.StoreMySQL:
				suggestion = "Use stack/mysql.New(dsn) for one-call setup with sane defaults"
			case analyzer.StorePebble:
				suggestion = "Use stack/pebble.New(dir) for one-call setup with sane defaults"
			case analyzer.StoreCustom:
				suggestion = "Consider using a stack/ preset for boilerplate-free setup, or keep custom wiring if you need full control"
			case analyzer.StoreUnknown,
				analyzer.StoreMemory,
				analyzer.StoreTurso,
				analyzer.StoreNone,
				analyzer.StoreDuckDB,
				analyzer.StoreBolt,
				analyzer.StoreBadger,
				analyzer.StoreDgraph,
				analyzer.StoreIroh,
				analyzer.StoreBigTable:
				// Keep generic suggestion for these cases (no stack/ preset
				// exists for them).
			}

			f, err := findingTemplate.Builder(
				"A009",
				"Project does not use a stack/ preset — manual wiring is error-prone and misses defaults",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceMedium).
				WithSuggestion(suggestion).
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// A010: Custom error types duplicating go-error-family.
//
//nolint:ireturn // factory returns public interface
func NewA010Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A010-custom-error-types",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					ts, ok := n.(*ast.TypeSpec)
					if !ok {
						return true
					}

					name := ts.Name.Name
					if !strings.HasSuffix(name, "Error") && !strings.HasSuffix(name, "Err") {
						return true
					}

					if strings.HasSuffix(name, "FamilyError") {
						return true
					}

					it, ok := ts.Type.(*ast.InterfaceType)
					if !ok || it.Methods == nil || len(it.Methods.List) == 0 {
						return true
					}

					hasErrorMethod := false

					for _, m := range it.Methods.List {
						if len(m.Names) > 0 && m.Names[0].Name == "Error" {
							hasErrorMethod = true

							break
						}
					}

					if !hasErrorMethod {
						return true
					}

					pos := ctx.Fset.Position(ts.Pos())

					f, err := findingTemplate.Builder(
						"A010",
						fmt.Sprintf("Custom error interface %s — consider using go-error-family taxonomy instead", name),
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceLow).
						WithSuggestion("Use errorfamily.NewRejection, errorfamily.WrapConflict, etc. for classified errors").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err != nil {
						return true
					}

					findings = append(findings, f)

					return true
				})
			}

			return findings, nil
		},
	)
}

// A012: Missing tombstone handling.
// Detects fold/apply functions that don't check for tombstone events.
// Only flags when the fold's module includes tombstone-like event types
// (Deleted, Removed, Archived) — domains without soft-delete don't need it.
// Evaluated per-module via ProfileForFile so a library sub-module is not
// flagged when an example sub-module happens to emit tombstone events.
//
//nolint:ireturn // factory returns public interface
func NewA012Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A012-missing-tombstone-handling",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				if !fold.HasSwitch {
					continue
				}

				// The fold's switch already handles a tombstone-like event type
				// (case values resolved against event.Type constants in
				// ResolveFoldTombstoneCases) — ADR-0114's event-type-based
				// deletion handling is present; coaching would be a false
				// positive (nsfw-classifier feedback, 2026-10-03).
				if fold.HandlesTombstoneEvent {
					continue
				}

				// Evaluate per-module: only flag folds in modules that emit
				// tombstone-like events. Using the primary profile would flag
				// library folds when an example sub-module has soft-delete.
				if !ctx.ProfileForFile(fold.File).HasSoftDelete {
					continue
				}

				f, err := findingTemplate.Builder(
					"A012",
					fmt.Sprintf(
						"Fold %s does not check for tombstone events — deleted aggregates may resurrect",
						fold.FuncName,
					),
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(fold.File), fold.Pos.Line, fold.Pos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceLow).
					WithSuggestion("Handle deletion events (e.g., \"user.deleted\") in your fold function via event-type-based detection (ADR-0114)").
					WithSnippet(ctx.SourceLine(fold.File, fold.Pos.Line)).
					Build()
				if err != nil {
					continue
				}

				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// A013: Value-embedded BasicCommand (inverted 2026-10-03, GitHub #51).
//
// Every BasicCommand method has a pointer receiver, so promoted methods
// only enter a struct's method set through a pointer: a value embed cannot
// satisfy command.Command and fails to compile when dispatched. Pointer
// embedding is additionally load-bearing for ApplyOptions, which mutates
// through the embedded pointer so pipeline enrichment reaches the command.
//
//nolint:ireturn // factory returns public interface
func NewA013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A013-pointer-vs-value-basic-command",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, cmd := range ctx.Registry.Commands {
				if !cmd.HasBasicCmd {
					continue
				}

				for _, gf := range ctx.GoFiles {
					if gf.Path != cmd.File || gf.IsTest {
						continue
					}

					ast.Inspect(gf.AST, func(n ast.Node) bool {
						ts, ok := n.(*ast.TypeSpec)
						if !ok || ts.Name.Name != cmd.Name {
							return true
						}

						st, ok := ts.Type.(*ast.StructType)
						if !ok || st.Fields == nil {
							return true
						}

						for _, field := range st.Fields.List {
							// Pointer embedding is the sanctioned form — silent.
							if _, ok := field.Type.(*ast.StarExpr); ok {
								continue
							}

							// Accept both `BasicCommand` (bare ident) and the
							// qualified real-world embed `command.BasicCommand`.
							basic := false

							switch inner := field.Type.(type) {
							case *ast.Ident:
								basic = inner.Name == "BasicCommand"
							case *ast.SelectorExpr:
								basic = inner.Sel.Name == "BasicCommand"
							}

							if basic {
								pos := ctx.Fset.Position(ts.Pos())

								f, err := findingTemplate.Builder(
									"A013",
									fmt.Sprintf("Command %s embeds BasicCommand by value — its methods are pointer-receiver, so %s does not satisfy command.Command", cmd.Name, cmd.Name),
									finding.SeverityWarning,
									finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
								).
									WithCategory(finding.CategoryBestPractice).
									WithConfidence(finding.ConfidenceHigh).
									WithSuggestion("Embed *command.BasicCommand (pointer): required while every BasicCommand method has a pointer receiver, and ApplyOptions mutates through the embedded pointer so pipeline enrichment reaches the dispatched command").
									WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
									Build()
								if err != nil {
									return true
								}

								findings = append(findings, f)
							}
						}

						return true
					})
				}
			}

			return findings, nil
		},
	)
}
