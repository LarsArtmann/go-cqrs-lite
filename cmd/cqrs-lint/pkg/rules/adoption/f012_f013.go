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
			if importsPath(ctx, "go-cqrs-lite/deriver") {
				return nil, nil
			}

			// Need SubscribeAll + command dispatch to suggest deriver.
			if !projectHasCallAny(ctx, "bus", "SubscribeAll") {
				return nil, nil
			}

			if ctx.FeatureProfile.CommandFlow != analyzer.CommandFlowCommands {
				return nil, nil
			}

			pos, ok := firstCallPos(ctx, "bus", "SubscribeAll")
			if !ok {
				pos, ok = firstFilePos(ctx)
				if !ok {
					return nil, nil
				}
			}

			return singleInfoFinding(
				ctx,
				"F012",
				"bus.SubscribeAll with command dispatch detected but deriver "+
					"module is not used — saga-like patterns benefit from "+
					"type-safe event→command derivation",
				"Import the deriver module and use deriver.New() with Then() "+
					"chains for type-safe event→command derivation. Provides "+
					"Filter, Idempotent, and AsHandler for clean saga composition.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// F013 detects server-mode projects with manual HTTP handlers for command/query
// dispatch that do not use the transport modules. The transport/http module
// provides SSE delivery, and transport/grpc provides remote dispatch.
//
//nolint:ireturn // factory returns public interface
func NewF013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F013-no-transport-module",
		func(_ context.Context) ([]finding.Finding, error) {
			if !ctx.FeatureProfile.HasServer {
				return nil, nil
			}

			// CLI tools with embedded dashboards don't need transport modules.
			if ctx.FeatureProfile.ServerLocal {
				return nil, nil
			}

			// The project already has a transport layer (transport/http,
			// transport/grpc, or an external module like cqrs-htmx). F013
			// suggests adopting a transport module — no point if one is present.
			if ctx.FeatureProfile.HasTransport {
				return nil, nil
			}

			// Check for manual HTTP handlers via stdlib, gorilla/mux, or
			// third-party web frameworks (chi, gin, echo, fiber).
			hasHTTPHandlers := projectHasCallAny(ctx, "http", "HandleFunc", "Handle") ||
				projectHasCallAny(ctx, "mux", "HandleFunc", "Handle") ||
				hasWebFrameworkHandlers(ctx)

			if !hasHTTPHandlers {
				return nil, nil
			}

			pos, ok := firstCallPos(ctx, "http", "HandleFunc")
			if !ok {
				pos, ok = firstCallPos(ctx, "mux", "HandleFunc")
			}

			if !ok {
				pos, ok = firstFilePos(ctx)
				if !ok {
					return nil, nil
				}
			}

			return singleInfoFinding(
				ctx,
				"F013",
				"Manual HTTP handlers for dispatch but transport module is not "+
					"used — hand-rolled HTTP/gRPC lacks SSE delivery and typed "+
					"remote dispatch",
				"Import transport/http for SSE event delivery (SSEBroker, "+
					"BackfillHandler) or transport/grpc for typed remote "+
					"command/query dispatch (CommandClient, QueryClient).",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}
