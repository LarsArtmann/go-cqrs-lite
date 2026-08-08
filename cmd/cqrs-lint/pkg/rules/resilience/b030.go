package resilience

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B030: Circuit breaker absence.
// Detects a bus/dispatcher that lacks circuit breaker middleware. Without a
// circuit breaker, cascading failures from downstream services can overwhelm
// the system.
//
//nolint:ireturn // factory returns public interface
func NewB030Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B030-missing-circuit-breaker",
		func(_ context.Context) ([]finding.Finding, error) {
			if ctx.IsLibrarySelfLint() {
				return nil, nil
			}
			if !ctx.FeatureProfile.HasServer {
				return nil, nil
			}

			buses := findBusVariables(ctx)

			var findings []finding.Finding

			for name, pos := range buses {
				if hasMiddlewareKeyword(ctx, name, "circuit") ||
					hasMiddlewareKeyword(ctx, name, "breaker") {
					continue
				}

				fs := singleInfoFinding(
					ctx,
					"B030",
					"Bus/dispatcher "+name+" has no circuit breaker middleware — "+
						"cascading failures from downstream services are not isolated",
					"Add middleware.CircuitBreaker() to "+name+".Use() chain to "+
						"isolate downstream failures and prevent cascade",
					pos,
					finding.ConfidenceLow,
				)
				findings = append(findings, fs...)
			}

			return findings, nil
		},
	)
}
