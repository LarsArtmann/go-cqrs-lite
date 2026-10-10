package architecture

import (
	"context"
	"fmt"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// E022: a declared event's version ladder has a missing rung. Ops match on
// (event type, schema version) exactly; the chain upcasts an event only when
// a rung exists for the version it was stored at. A declaration at current
// version N therefore needs an op FROM every version 1..N-1 — a gap means
// stored events at the missing version silently never upcast (they decode as
// their old shape forever, the exact staleness the ladder exists to prevent).
//
// Scope: literal forms only. Const-referenced types or versions stay
// untracked (the ladder's Current stays 0 → silent) — guessing a constant's
// value would fire on renamed identifiers, not real drift. Declarations at
// current version 1 (or with unknown version) never fire: there is nothing
// below to migrate. Policy reference: docs/SCHEMA-COMPATIBILITY.md
// (version-bump triggers + the continuous-ladder rule).
//
//nolint:ireturn // factory returns public interface
func NewE022Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E022-schema-ladder-gap",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for evtType, ladder := range ctx.Registry.SchemaLadders {
				if ladder.Current < 2 {
					continue
				}

				var missing []int

				for version := 1; version < ladder.Current; version++ {
					if _, ok := ladder.OpVersions[version]; !ok {
						missing = append(missing, version)
					}
				}

				if len(missing) == 0 {
					continue
				}

				versions := make([]string, 0, len(missing))
				for _, version := range missing {
					versions = append(versions, fmt.Sprintf("v%d", version))
				}

				file := ladder.Decl.File
				if file == "" {
					file = ctx.ProjectRoot + "/go.mod"
				}

				f, err := findingTemplate.Builder(
					"E022",
					fmt.Sprintf(
						"schema declaration for %q is at version %d but no op migrates FROM %s — stored events at that version never upcast",
						evtType, ladder.Current, strings.Join(versions, ", "),
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(file), ladder.Decl.Line, 1),
				).
					WithCategory(finding.CategoryStructure).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion(
						"Declare an op for each missing version (schema.RenameField/AddField/RemoveField/Transform with that source version), " +
							"or lower the current version — see docs/SCHEMA-COMPATIBILITY.md (the ladder must be continuous)",
					).
					WithSnippet(ctx.SourceLine(ladder.Decl.File, ladder.Decl.Line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}
