package architecture

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// E008: Stack preset bypass.
// Detects projects that import go-cqrs-lite/stack (stack.Bundle is available)
// but call decider.NewRepository directly instead of using stack presets.
// Stack presets provide opinionated defaults (codec, snapshot, tracing) that
// the manual wiring path misses.
//
//nolint:ireturn // factory returns public interface
func NewE008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E008-stack-preset-bypass",
		func(_ context.Context) ([]finding.Finding, error) {
			if !importsPathSuffix(ctx, "go-cqrs-lite/stack") {
				return nil, nil
			}

			if !projectCalls(ctx, "decider", "NewRepository") {
				return nil, nil
			}

			pos, ok := firstCallPos(ctx, "decider", "NewRepository")
			if !ok {
				pos, _ = firstFilePos(ctx)
			}

			return singleFinding(
				ctx,
				"E008",
				"Project imports stack presets but calls decider.NewRepository directly — "+
					"stack presets provide opinionated defaults (codec, snapshot, tracing) "+
					"that manual wiring misses",
				"Use stack.Bundle methods (EventStore, ReadModel, Materialize) or "+
					"stack/sqlite.New, stack/postgres.New instead of manual decider.NewRepository wiring",
				pos,
				finding.SeverityInfo,
				finding.ConfidenceMedium,
			), nil
		},
	)
}

// E009: No HTTP integration for CQRS.
// Detects projects that have both command and query dispatching but no HTTP
// transport layer. Commands and queries can only be dispatched programmatically,
// making the system inaccessible via REST/SSE. This is a coaching rule — CLI
// tools and background workers legitimately don't need HTTP.
//
//nolint:ireturn // factory returns public interface
func NewE009Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E009-no-http-integration",
		func(_ context.Context) ([]finding.Finding, error) {
			hasCommand := importsPathSuffix(ctx, "go-cqrs-lite/command")
			hasQuery := importsPathSuffix(ctx, "go-cqrs-lite/query")
			if !hasCommand || !hasQuery {
				return nil, nil
			}

			hasHTTP := importsPathSuffix(ctx, "go-cqrs-lite/transport/http") ||
				importsPathSuffix(ctx, "go-cqrs-lite/transport/grpc")
			if hasHTTP {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleFinding(
				ctx,
				"E009",
				"Project has command and query dispatchers but no HTTP/gRPC transport layer — "+
					"commands and queries can only be dispatched programmatically",
				"Add transport/http (SSE event delivery, HTTP handlers) or transport/grpc "+
					"to expose the CQRS API to external clients",
				pos,
				finding.SeverityInfo,
				finding.ConfidenceLow,
			), nil
		},
	)
}

// E010: Event capture without domain validation.
// Detects projects that write events directly to the store (store.Save,
// store.AppendBatch) without going through a decider. This bypasses domain
// rule enforcement — the decider validates invariants before persisting events.
// This is a valid pattern for external event ingestion (Discord sync, webhook
// receivers), but the finding suggests wrapping in a command for validation.
//
//nolint:ireturn // factory returns public interface
func NewE010Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E010-capture-without-validation",
		func(_ context.Context) ([]finding.Finding, error) {
			if !projectCallsAny(ctx, "store", "Save", "AppendBatch", "Append") {
				return nil, nil
			}

			if projectCalls(ctx, "decider", "Execute") ||
				projectCalls(ctx, "repo", "Execute") {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleFinding(
				ctx,
				"E010",
				"Project writes events directly to the store without decider validation — "+
					"domain rules (invariants) are not enforced before persistence",
				"Wrap store writes in a command handler that routes through "+
					"decider.Repository.Execute — or document this as intentional external ingestion",
				pos,
				finding.SeverityInfo,
				finding.ConfidenceLow,
			), nil
		},
	)
}

// E011: Excessive adapter layers.
// Detects projects with 3+ types named *Adapter that bridge between command
// handlers and decider repositories. Excessive adapter layers add indirection
// and make the data flow hard to trace. The threshold is deliberately high
// (3) to avoid flagging a single legitimate adapter.
//
//nolint:ireturn // factory returns public interface
func NewE011Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E011-excessive-adapter-layers",
		func(_ context.Context) ([]finding.Finding, error) {
			adapterCount := countTypesWithSuffix(ctx, "Adapter")
			if adapterCount < 3 {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleFinding(
				ctx,
				"E011",
				fmt.Sprintf(
					"Project has %d *Adapter types — excessive adapter layers between "+
						"command handlers and decider add indirection and obscure data flow",
					adapterCount,
				),
				"Consolidate adapters or inline the conversion logic — each adapter layer "+
					"is a transformation that could be eliminated with a cleaner decider/handler boundary",
				pos,
				finding.SeverityInfo,
				finding.ConfidenceLow,
			), nil
		},
	)
}
