package api

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A009: Missing stack preset.
// Detects projects that don't use any stack/ preset (stack/sqlite, stack/pebble, etc.).
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

			suggestion := "Use stack/sqlite.New(dsn) or stack/pebble.New(dir) for one-call setup with sane defaults"
			switch ctx.FeatureProfile.Store {
			case analyzer.StoreSQLite:
				suggestion = "Use stack/sqlite.New(dsn) for one-call setup with sane defaults"
			case analyzer.StorePostgres:
				suggestion = "Use stack/postgres.New(dsn) for one-call setup with sane defaults"
			case analyzer.StorePebble:
				suggestion = "Use stack/pebble.New(dir) for one-call setup with sane defaults"
			case analyzer.StoreCustom:
				suggestion = "Consider using a stack/ preset for boilerplate-free setup, or keep custom wiring if you need full control"
			}

			f, err := finding.NewBuilder(
				"A009",
				toolName,
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

					f, err := finding.NewBuilder(
						"A010", toolName,
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
// Only flags when the project's event types include tombstone-like names
// (Deleted, Removed, Archived) — domains without soft-delete don't need it.
// Detection now consults ctx.FeatureProfile.HasSoftDelete (centralized).
//nolint:ireturn // factory returns public interface
func NewA012Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A012-missing-tombstone-handling",
		func(_ context.Context) ([]finding.Finding, error) {
			if !ctx.FeatureProfile.HasSoftDelete {
				return nil, nil
			}

			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				if !fold.HasSwitch {
					continue
				}

				f, err := finding.NewBuilder(
					"A012",
					toolName,
					fmt.Sprintf(
						"Fold %s does not check for tombstone events — deleted aggregates may resurrect",
						fold.FuncName,
					),
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(fold.File), fold.Pos.Line, fold.Pos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceLow).
					WithSuggestion("Use event.DetectTombstone(events) to handle soft-delete in your fold function").
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

// A013: Pointer vs value BasicCommand embedding.
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
							if se, ok := field.Type.(*ast.StarExpr); ok {
								if id, ok := se.X.(*ast.Ident); ok && id.Name == "BasicCommand" {
									pos := ctx.Fset.Position(ts.Pos())

									f, err := finding.NewBuilder(
										"A013", toolName,
										fmt.Sprintf("Command %s embeds *BasicCommand (pointer) — value embedding is recommended for stack allocation", cmd.Name),
										finding.SeverityInfo,
										finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
									).
										WithCategory(finding.CategoryBestPractice).
										WithConfidence(finding.ConfidenceHigh).
										WithSuggestion("Embed BasicCommand by value (not pointer) for better cache locality and simpler nil-safety").
										WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
										Build()
									if err != nil {
										return true
									}

									findings = append(findings, f)
								}
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
