package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F003 detects server-mode projects that have no OpenTelemetry tracing.
// OTel provides distributed tracing for request flows across services.
//
//nolint:ireturn // factory returns public interface
func NewF003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F003-no-otel-tracing",
		func(_ context.Context) ([]finding.Finding, error) {
			if !ctx.FeatureProfile.HasServer {
				return nil, nil
			}

			if importsPath(ctx, "go.opentelemetry.io") ||
				importsPath(ctx, "go-cqrs-lite/otel") {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(ctx,
				"F003",
				"Server-mode project has no OpenTelemetry tracing — "+
					"distributed tracing is unavailable",
				"Import the otel module and call cqrsotel.Setup() to enable "+
					"tracing. Add middleware.NewOTelBundle() to your bus and "+
					"dispatcher Use() chains for automatic span creation.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// F004 detects server-mode projects that have no Prometheus metrics.
// Prometheus provides runtime observability for production services.
//
//nolint:ireturn // factory returns public interface
func NewF004Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F004-no-prometheus-metrics",
		func(_ context.Context) ([]finding.Finding, error) {
			if !ctx.FeatureProfile.HasServer {
				return nil, nil
			}

			if importsPath(ctx, "go-cqrs-lite/prometheus") ||
				importsPath(ctx, "prometheus/client_golang") {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(ctx,
				"F004",
				"Server-mode project has no Prometheus metrics — "+
					"runtime observability is unavailable",
				"Import the prometheus module and call cqrsprometheus.Setup() "+
					"to expose a /metrics endpoint with CQRS-specific histograms "+
					"and counters.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}
