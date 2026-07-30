package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F007 detects projects with command dispatchers that do not use idempotency
// middleware. For at-least-once delivery, duplicate commands cause duplicate
// side effects without idempotency.
//
//nolint:ireturn // factory returns public interface
func NewF007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F007-no-idempotency-middleware",
		func(_ context.Context) ([]finding.Finding, error) {
			if ctx.FeatureProfile.CommandFlow != analyzer.CommandFlowCommands {
				return nil, nil
			}

			if projectHasCallAny(ctx, "middleware",
				"CommandIdempotency", "EventIdempotency", "QueryIdempotency") {
				return nil, nil
			}

			if importsPath(ctx, "go-cqrs-lite/idempotency") {
				return nil, nil
			}

			pos, ok := firstCallByName(ctx, "NewDispatcher")
			if !ok {
				pos, ok = firstFilePos(ctx)
				if !ok {
					return nil, nil
				}
			}

			return singleInfoFinding(ctx,
				"F007",
				"Command dispatcher has no idempotency middleware — "+
					"duplicate commands cause duplicate side effects under "+
					"at-least-once delivery",
				"Add middleware.CommandIdempotency(store, ttl, nil) to your "+
					"command dispatcher's Use() chain. Requires an idempotency.Store "+
					"(MemoryStore for single-process, KVStore/SQLStore for distributed).",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// F008 detects projects with many event types that use the JSON codec instead
// of CBOR. CBOR produces ~35% smaller payloads for event-heavy systems.
//
//nolint:ireturn // factory returns public interface
func NewF008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F008-no-cbor-codec",
		func(_ context.Context) ([]finding.Finding, error) {
			if eventCount(ctx) < 5 {
				return nil, nil
			}

			if !projectHasSelector(ctx, "codec", "JSONCodec") {
				return nil, nil
			}

			if projectHasSelector(ctx, "codec", "CBORCodec") {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(ctx,
				"F008",
				"Project has "+itoa(eventCount(ctx))+
					" event types and uses JSON codec — CBOR is ~35% smaller "+
					"for event-heavy systems",
				"Switch to codec.CBORCodec{} via event.DefaultCodec or the stack "+
					"bundle's WithEventCodec option. Events are self-describing "+
					"(encoding stamped per event), so mixed JSON+CBOR streams "+
					"decode correctly.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}
