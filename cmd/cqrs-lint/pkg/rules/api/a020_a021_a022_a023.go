package api

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// typeMethodMap maps a receiver type name to its method names and the
// position of the first method encountered (for reporting).
type typeMethodMap struct {
	methods map[string]bool
	file    string
	line    int
	col     int
}

// collectMethodsByType scans all non-test Go files and returns a map of
// receiver type name → method names + first-seen position.
func collectMethodsByType(ctx *analyzer.AnalysisContext) map[string]*typeMethodMap {
	result := make(map[string]*typeMethodMap)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil {
				continue
			}

			recvType := receiverTypeName(fn.Recv)
			if recvType == "" {
				continue
			}

			entry := result[recvType]
			if entry == nil {
				pos := ctx.Fset.Position(fn.Pos())
				entry = &typeMethodMap{
					methods: make(map[string]bool),
					file:    pos.Filename,
					line:    pos.Line,
					col:     pos.Column,
				}
				result[recvType] = entry
			}

			entry.methods[fn.Name.Name] = true
		}
	}

	return result
}

func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	expr := recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}

// A020: Custom event.Bus reimplementation.
// Detects a struct implementing Subscribe, SubscribeAll, Use, UsePublish,
// Publish, and/or Close — the event.Bus interface — in a project that imports
// go-cqrs-lite. Consumers should use watermill.NewEventBus() or the library's
// memory bus instead of hand-rolling a bus.
//
// Requires at least 4 of {Subscribe, SubscribeAll, Use, UsePublish, Publish,
// Close} to reduce false positives. The UsePublish method name is very
// distinctive and must be one of the matching methods.
//
//nolint:ireturn // factory returns public interface
func NewA020Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A020-custom-event-bus",
		func(_ context.Context) ([]finding.Finding, error) {
			if !projectImportsCQRS(ctx) {
				return nil, nil
			}

			busMethods := map[string]bool{
				"Subscribe":    true,
				"SubscribeAll": true,
				"Use":          true,
				"UsePublish":   true,
				"Publish":      true,
				"Close":        true,
			}

			methodsByType := collectMethodsByType(ctx)
			var findings []finding.Finding

			for typeName, entry := range methodsByType {
				matchCount := 0
				hasUsePublish := false

				for m := range entry.methods {
					if busMethods[m] {
						matchCount++
					}

					if m == "UsePublish" {
						hasUsePublish = true
					}
				}

				// Need at least 4 of 6 bus methods AND UsePublish (very distinctive).
				if matchCount < 4 || !hasUsePublish {
					continue
				}

				f, err := finding.NewBuilder(
					"A020", toolName,
					fmt.Sprintf(
						"Custom event.Bus implementation %q — use watermill.NewEventBus() or the library's memory bus instead of reimplementing",
						typeName,
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(entry.file), entry.line, entry.col),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion("Replace custom bus with watermill.NewEventBus() or the library's built-in memory bus").
					WithSnippet(ctx.SourceLine(entry.file, entry.line)).
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

// A021: Custom event.Store reimplementation.
// Detects a struct implementing Save, Load, and LoadFromVersion — the core
// event.Store interface — in a project that imports go-cqrs-lite. Consumers
// should use storage/memory.MemoryStore or a SQL/Pebble backend instead.
//
// Requires all three methods: Save, Load, LoadFromVersion. LoadFromVersion
// is very CQRS-specific and eliminates most false positives.
//
//nolint:ireturn // factory returns public interface
func NewA021Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A021-custom-event-store",
		func(_ context.Context) ([]finding.Finding, error) {
			if !projectImportsCQRS(ctx) {
				return nil, nil
			}

			methodsByType := collectMethodsByType(ctx)
			var findings []finding.Finding

			for typeName, entry := range methodsByType {
				if !entry.methods["Save"] || !entry.methods["Load"] ||
					!entry.methods["LoadFromVersion"] {
					continue
				}

				f, err := finding.NewBuilder(
					"A021", toolName,
					fmt.Sprintf(
						"Custom event.Store implementation %q — use storage/memory.MemoryStore or a SQL/Pebble backend instead of reimplementing",
						typeName,
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(entry.file), entry.line, entry.col),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion("Replace custom store with storage/memory.MemoryStore, storage.NewSQLiteBackend, or storage/pebble.NewStore").
					WithSnippet(ctx.SourceLine(entry.file, entry.line)).
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

// A022: Raw otel.Tracer/Meter instead of cqrsotel.
// Detects direct otel.Tracer() or otel.Meter() calls in files that import
// go-cqrs-lite modules. The library's otel/ module provides re-exports with
// CQRS-specific span names and views. Consumers should use cqrsotel.NewTracer
// and cqrsotel.NewMeter instead.
//
//nolint:ireturn // factory returns public interface
func NewA022Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A022-raw-otel-tracer",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if !lintutil.FileImportsCQRS(gf.AST) {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					pkg, ok := sel.X.(*ast.Ident)
					if !ok || pkg.Name != "otel" {
						return true
					}

					if sel.Sel.Name != "Tracer" && sel.Sel.Name != "Meter" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"A022", toolName,
						fmt.Sprintf(
							"Raw otel.%s() — use cqrsotel.New%s instead for CQRS-specific span names and views",
							sel.Sel.Name,
							sel.Sel.Name,
						),
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion(fmt.Sprintf(
							"Replace otel.%s(name) with cqrsotel.New%s(name)",
							sel.Sel.Name, sel.Sel.Name,
						)).
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

// A023: Custom in-memory snapshot store.
// Detects a struct with "Snapshot" in its name that implements Save and Load
// methods — the SnapshotSink/SnapshotSource interfaces — in a project that
// imports go-cqrs-lite. Consumers should use storage/memory.NewMemorySnapshotStore
// instead of reimplementing.
//
//nolint:ireturn // factory returns public interface
func NewA023Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A023-custom-snapshot-store",
		func(_ context.Context) ([]finding.Finding, error) {
			if !projectImportsCQRS(ctx) {
				return nil, nil
			}

			methodsByType := collectMethodsByType(ctx)
			var findings []finding.Finding

			for typeName, entry := range methodsByType {
				if !strings.Contains(typeName, "Snapshot") {
					continue
				}

				if !entry.methods["Save"] || !entry.methods["Load"] {
					continue
				}

				f, err := finding.NewBuilder(
					"A023", toolName,
					fmt.Sprintf(
						"Custom snapshot store %q — use storage/memory.NewMemorySnapshotStore() instead of reimplementing",
						typeName,
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(entry.file), entry.line, entry.col),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion("Replace custom snapshot store with storage/memory.NewMemorySnapshotStore()").
					WithSnippet(ctx.SourceLine(entry.file, entry.line)).
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

// projectImportsCQRS returns true if any non-test file in the project imports
// a go-cqrs-lite module. Used by project-level rules to gate detection.
func projectImportsCQRS(ctx *analyzer.AnalysisContext) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		if lintutil.FileImportsCQRS(gf.AST) {
			return true
		}
	}

	return false
}
