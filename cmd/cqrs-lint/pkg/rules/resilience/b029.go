package resilience

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-finding"
)

// B029: Missing retry middleware.
// Detects a bus/dispatcher that is created and used without any retry
// middleware registered. B008 detects manual retry; this rule detects
// the absence of middleware-based retry entirely.
//
//nolint:ireturn // factory returns public interface
func NewB029Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B029-missing-retry-middleware",
		func(_ context.Context) ([]finding.Finding, error) {
			if ctx.IsLibrarySelfLint() {
				return nil, nil
			}

			buses := findBusVariables(ctx)

			var findings []finding.Finding

			for name, pos := range buses {
				if !ctx.ProfileForFile(pos.Filename).HasServer {
					continue
				}

				if hasMiddlewareKeyword(ctx, name, "retry") {
					continue
				}

				fs := singleInfoFinding(
					ctx,
					"B029",
					"Bus/dispatcher "+name+" has no retry middleware — "+
						"transient failures will propagate to callers",
					"Add middleware.Retry() to "+name+".Use() chain for "+
						"automatic transient failure recovery",
					pos,
					finding.ConfidenceLow,
				)
				findings = append(findings, fs...)
			}

			return findings, nil
		},
	)
}
