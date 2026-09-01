package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// F027 detects server-mode projects that import OTel but never call
// otel.Setup or SetTracerProvider, leaving tracing configured but inactive.
//
//nolint:ireturn // factory returns public interface
func NewF027Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F027-missing-otel-sdk-init",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if !sc.profile.HasServer {
					continue
				}

				if !importsPathIn(sc.files, "go-cqrs-lite/otel") &&
					!importsPathIn(sc.files, "go.opentelemetry.io/otel") {
					continue
				}

				if hasCallIn(sc.files, "otel", "Setup") ||
					hasCallIn(sc.files, "cqrsotel", "Setup") ||
					hasCallIn(sc.files, "otel", "SetTracerProvider", "SetMeterProvider") {
					continue
				}

				pos, ok := firstFilePosIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F027",
					"Project imports OTel but never calls Setup() — "+
						"tracing is configured but inactive",
					"Call cqrsotel.Setup(WithService(...)) at startup to "+
						"initialize the tracer and meter providers",
					pos, finding.ConfidenceMedium,
				)...)
			}

			return out, nil
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
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if !sc.profile.HasServer {
					continue
				}

				if !importsPathIn(sc.files, "log/slog") {
					continue
				}

				if hasCallIn(sc.files, "slog", "SetDefault") {
					continue
				}

				pos, ok := firstFilePosIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F028",
					"Project uses slog but never calls slog.SetDefault — "+
						"the default logger is unconfigured (Text to stderr)",
					"Call slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil))) "+
						"at startup for structured JSON logging to stdout",
					pos, finding.ConfidenceLow,
				)...)
			}

			return out, nil
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
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if !sc.profile.HasServer {
					continue
				}

				if !importsPathIn(sc.files, "go-cqrs-lite/otel") &&
					!importsPathIn(sc.files, "go.opentelemetry.io/otel") {
					continue
				}

				if hasCallIn(sc.files, "middleware",
					"EventTracing", "EventPublishTracing",
					"CommandTracing", "QueryTracing") ||
					hasSelectorIn(sc.files, "middleware", "NewOTelBundle") {
					continue
				}

				pos, ok := firstFilePosIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F029",
					"Project has OTel but no tracing middleware — "+
						"command/event/query handlers are not instrumented",
					"Add middleware.NewOTelBundle(tracer, meter) to your bus and "+
						"dispatcher Use() chains for automatic span creation on every operation",
					pos, finding.ConfidenceMedium,
				)...)
			}

			return out, nil
		},
	)
}
