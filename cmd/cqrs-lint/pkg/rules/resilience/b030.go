package resilience

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-finding"
)

// B030: Circuit breaker absence.
// Detects a bus/dispatcher that lacks circuit breaker middleware. Without a
// circuit breaker, cascading failures from downstream services can overwhelm
// the system. Journal-tail buses are skipped (busIsJournalTail): read-only
// subscribers (a variable whose only bus calls are Subscribe/SubscribeAll —
// an in-process journal tail) and variables assigned from an engine Bus()
// accessor (subscribing or feeding the in-process fan-out are both
// journal-tail acts) never call downstream services, so circuit breaker
// middleware is category confusion for them (CV feedback, 2026-10-03).
//
//nolint:ireturn // factory returns public interface
func NewB030Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B030-missing-circuit-breaker",
		func(_ context.Context) ([]finding.Finding, error) {
			if ctx.IsLibrarySelfLint() {
				return nil, nil
			}

			buses := findBusVariables(ctx)

			var findings []finding.Finding

			for _, v := range buses {
				if !ctx.ProfileForFile(v.pos.Filename).HasServer {
					continue
				}

				if busIsJournalTail(ctx, v) {
					continue
				}

				if hasMiddlewareKeyword(ctx, v, "circuit") ||
					hasMiddlewareKeyword(ctx, v, "breaker") {
					continue
				}

				fs := singleInfoFinding(
					ctx,
					"B030",
					"Bus/dispatcher "+v.name+" has no circuit breaker middleware — "+
						"cascading failures from downstream services are not isolated",
					"Add middleware.CircuitBreaker() to "+v.name+".Use() chain to "+
						"isolate downstream failures and prevent cascade",
					v.pos,
					finding.ConfidenceLow,
				)
				findings = append(findings, fs...)
			}

			return findings, nil
		},
	)
}
