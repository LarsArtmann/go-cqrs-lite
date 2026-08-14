package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F012 detects saga-like patterns (bus.SubscribeAll handlers that dispatch
// commands based on events) without using the deriver module. The deriver
// provides Deriver, Then, Filter, and Idempotent for type-safe event→command
// derivation.
//
//nolint:ireturn // factory returns public interface
func NewF012Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F012-no-deriver-module",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if importsPathIn(sc.files, "go-cqrs-lite/deriver") {
					continue
				}

				if !hasCallIn(sc.files, "bus", "SubscribeAll") {
					continue
				}

				if sc.profile.CommandFlow != analyzer.CommandFlowCommands {
					continue
				}

				pos, ok := firstCallPosIn(ctx.Fset, sc.files, "bus", "SubscribeAll")
				if !ok {
					pos, ok = firstFilePosIn(ctx.Fset, sc.files)
					if !ok {
						continue
					}
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F012",
					"bus.SubscribeAll with command dispatch detected but deriver "+
						"module is not used — saga-like patterns benefit from "+
						"type-safe event→command derivation",
					"Import the deriver module and use deriver.New() with Then() "+
						"chains for type-safe event→command derivation. Provides "+
						"Filter, Idempotent, and AsHandler for clean saga composition.",
					pos, finding.ConfidenceLow,
				)...)
			}

			return out, nil
		},
	)
}

// F013 detects server-mode projects with manual HTTP handlers for command/query
// dispatch that do not use any sanctioned delivery module: the watermill/
// bridge (broker backends), go-sse (SSE), or cqrs-htmx. The transport/*
// modules are deprecated and no longer coached.
//
//nolint:ireturn // factory returns public interface
func NewF013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F013-no-transport-module",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if !sc.profile.HasServer || sc.profile.ServerLocal ||
					sc.profile.HasTransport {
					continue
				}

				hasHTTPHandlers := hasCallIn(sc.files, "http", "HandleFunc", "Handle") ||
					hasCallIn(sc.files, "mux", "HandleFunc", "Handle") ||
					hasWebFrameworkHandlersIn(sc.files)

				if !hasHTTPHandlers {
					continue
				}

				pos, ok := firstCallPosIn(ctx.Fset, sc.files, "http", "HandleFunc")
				if !ok {
					pos, ok = firstCallPosIn(ctx.Fset, sc.files, "mux", "HandleFunc")
				}

				if !ok {
					pos, ok = firstFilePosIn(ctx.Fset, sc.files)
					if !ok {
						continue
					}
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F013",
					"Manual HTTP handlers for dispatch but no delivery module is used — "+
						"hand-rolled HTTP lacks broker fanout, SSE delivery, and "+
						"trace-context propagation",
					"Use the watermill/ bridge (NewEventPublisher/WithBackend) for "+
						"broker-backed dispatch, go-sse for SSE streams, or cqrs-htmx. "+
						"The transport/* modules are deprecated.",
					pos, finding.ConfidenceLow,
				)...)
			}

			return out, nil
		},
	)
}
