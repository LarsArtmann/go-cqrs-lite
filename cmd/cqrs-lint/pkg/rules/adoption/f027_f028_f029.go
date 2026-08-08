package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F027 detects server-mode projects that import OTel but never call
// otel.Setup or SetTracerProvider, leaving tracing configured but inactive.
//
//nolint:ireturn // factory returns public interface
func NewF027Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F027-missing-otel-sdk-init",
		func(_ context.Context) ([]finding.Finding, error) {
			if !ctx.FeatureProfile.HasServer {
				return nil, nil
			}

			if !importsPath(ctx, "go-cqrs-lite/otel") &&
				!importsPath(ctx, "go.opentelemetry.io/otel") {
				return nil, nil
			}

			if projectHasCall(ctx, "otel", "Setup") ||
				projectHasCall(ctx, "cqrsotel", "Setup") ||
				projectHasCallAny(ctx, "otel", "SetTracerProvider", "SetMeterProvider") {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F027",
				"Project imports OTel but never calls Setup() — "+
					"tracing is configured but inactive",
				"Call cqrsotel.Setup(WithService(...)) at startup to "+
					"initialize the tracer and meter providers",
				pos, finding.ConfidenceMedium,
			), nil
		},
	)
}

// F028 detects server-mode projects that use slog but never call
// slog.SetDefault, leaving structured logging unconfigured.
//
//nolint:ireturn // factory returns public interface
func NewF028Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F028-missing-slog-setdefault",
		func(_ context.Context) ([]finding.Finding, error) {
			if !ctx.FeatureProfile.HasServer {
				return nil, nil
			}

			if !importsPath(ctx, "log/slog") {
				return nil, nil
			}

			if projectHasCall(ctx, "slog", "SetDefault") {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F028",
				"Project uses slog but never calls slog.SetDefault — "+
					"the default logger is unconfigured (Text to stderr)",
				"Call slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil))) "+
					"at startup for structured JSON logging to stdout",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// F029 detects server-mode projects with bus/dispatcher but no tracing
// middleware, missing automatic span creation for distributed tracing.
//
//nolint:ireturn // factory returns public interface
func NewF029Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F029-missing-span-creation",
		func(_ context.Context) ([]finding.Finding, error) {
			if !ctx.FeatureProfile.HasServer {
				return nil, nil
			}

			if !importsPath(ctx, "go-cqrs-lite/otel") &&
				!importsPath(ctx, "go.opentelemetry.io/otel") {
				return nil, nil
			}

			if projectHasCallAny(ctx, "middleware",
				"EventTracing", "EventPublishTracing",
				"CommandTracing", "QueryTracing") ||
				projectHasSelector(ctx, "middleware", "NewOTelBundle") {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F029",
				"Project has OTel but no tracing middleware — "+
					"command/event/query handlers are not instrumented",
				"Add middleware.NewOTelBundle(tracer, meter) to your bus and "+
					"dispatcher Use() chains for automatic span creation on every operation",
				pos, finding.ConfidenceMedium,
			), nil
		},
	)
}
