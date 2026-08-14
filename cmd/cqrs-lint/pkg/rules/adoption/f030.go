package adoption

import (
	"context"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// deprecatedTransportImport pairs a deprecated transport/* import fragment
// (ADR-0127) with the sanctioned replacement summary used in the suggestion.
type deprecatedTransportImport struct {
	fragment    string
	replacement string
}

var deprecatedTransportImports = []deprecatedTransportImport{ //nolint:gochecknoglobals // static lookup table for deprecated transport detection
	{
		fragment: "go-cqrs-lite/transport/http",
		replacement: "Serve SSE with github.com/larsartmann/go-sse (or " +
			"metaengine.ServeSSE for read-model push), use the watermill/ " +
			"bridge (NewEventPublisher/NewCommandPublisher) for broker fanout, " +
			"or cqrs-htmx for server-rendered UIs",
	},
	{
		fragment: "go-cqrs-lite/transport/grpc",
		replacement: "Use the watermill/ bridge over any broker backend " +
			"(WithBackend/WithCommandBackend), or call grpc-go directly",
	},
}

// F030 detects imports of the deprecated transport/* modules (ADR-0127).
// The modules are frozen and removed at v5; projects importing them today
// get a concrete migration path toward the sanctioned delivery modules.
// This is the companion to F013: F013 coaches projects WITHOUT a delivery
// module toward the sanctioned paths, F030 coaches projects OFF the
// deprecated ones.
//
//nolint:ireturn // factory returns public interface
func NewF030Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F030-deprecated-transport-import",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				for _, dep := range deprecatedTransportImports {
					pos, ok := firstImportPosIn(ctx.Fset, sc.files, dep.fragment)
					if !ok {
						continue
					}

					out = append(out, singleWarningFinding(
						ctx,
						"F030",
						"Import of deprecated module "+dep.fragment+"/v4 — the "+
							"transport/* modules are removed at v5 (ADR-0127)",
						dep.replacement+
							". See docs/adr/0127-deprecate-transport-modules.md "+
							"for the symbol migration table.",
						pos, finding.ConfidenceHigh,
					)...)
				}
			}

			return out, nil
		},
	)
}

// firstImportPosIn returns the position of the first non-test import whose
// path contains fragment.
func firstImportPosIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
	fragment string,
) (token.Position, bool) {
	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}

			if strings.Contains(strings.Trim(imp.Path.Value, `"`), fragment) {
				return fset.Position(imp.Path.Pos()), true
			}
		}
	}

	return token.Position{}, false
}
