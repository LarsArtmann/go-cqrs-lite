package architecture

import (
	"context"
	"fmt"
	"go/token"

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

			// Import-alias-aware check: matches d.NewRepository() even when
			// decider is imported as `import d "go-cqrs-lite/decider"`.
			pos, found := projectCallsImportPath(ctx, "go-cqrs-lite/decider", "NewRepository")
			if !found {
				return nil, nil
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
// Detects modules that have both command and query dispatching but no HTTP
// transport layer. Commands and queries can only be dispatched programmatically,
// making the system inaccessible via REST/SSE. This is a coaching rule — CLI
// tools and background workers legitimately don't need HTTP.
// Evaluated per-module via ProfileForFile so a library sub-module is not
// suppressed when an example sub-module has transport.
//
//nolint:ireturn // factory returns public interface
func NewE009Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E009-no-http-integration",
		func(_ context.Context) ([]finding.Finding, error) {
			// Group files by module directory, tracking which modules
			// import both command and query. This lets us evaluate
			// transport per-module instead of workspace-wide.
			type moduleState struct {
				hasCommand bool
				hasQuery   bool
				firstPos   token.Position
				hasPos     bool
			}

			modules := make(map[string]*moduleState)

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				state := modules[gf.ModuleDir]
				if state == nil {
					state = &moduleState{}
					modules[gf.ModuleDir] = state
				}

				if fileImportsPath(gf, "go-cqrs-lite/command") {
					state.hasCommand = true
				}

				if fileImportsPath(gf, "go-cqrs-lite/query") {
					state.hasQuery = true
				}

				if !state.hasPos && gf.AST.Package != token.NoPos {
					state.firstPos = ctx.Fset.Position(gf.AST.Package)
					state.hasPos = true
				}
			}

			var findings []finding.Finding

			for _, state := range modules {
				if !state.hasCommand || !state.hasQuery || !state.hasPos {
					continue
				}

				// Evaluate transport per-module: a module with command+query
				// but no transport should be flagged, even if another module
				// in the workspace has transport.
				if ctx.ProfileForFile(state.firstPos.Filename).HasTransport {
					continue
				}

				findings = append(findings, singleFinding(
					ctx,
					"E009",
					"Module has command and query dispatchers but no HTTP/gRPC transport layer — "+
						"commands and queries can only be dispatched programmatically",
					"Add transport/http (SSE event delivery, HTTP handlers), transport/grpc, "+
						"or cqrs-htmx to expose the CQRS API to external clients",
					state.firstPos,
					finding.SeverityInfo,
					finding.ConfidenceLow,
				)...)
			}

			return findings, nil
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
// Uses type-aware receiver matching when type info is available: checks that
// the receiver of Save/Append/AppendBatch is a type from go-cqrs-lite event or
// storage packages. Falls back to variable-name heuristic ("store") in unit
// tests where type info is absent.
//
//nolint:ireturn // factory returns public interface
func NewE010Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E010-capture-without-validation",
		func(_ context.Context) ([]finding.Finding, error) {
			if !importsPathSuffix(ctx, "go-cqrs-lite/event") {
				return nil, nil
			}

			// Type-aware check: any Save/Append/AppendBatch on a CQRS store type.
			pos, found := projectCallsMethodOnType(
				ctx,
				[]string{"Save", "AppendBatch", "Append"},
				[]string{"go-cqrs-lite/event", "go-cqrs-lite/storage", "cqrs-lite/event"},
			)
			if !found {
				return nil, nil
			}

			// Suppress if any decider.Execute call is present.
			if projectHasMethodCallContaining(ctx, "Execute") {
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
// Detects CQRS projects with 3+ types named *Adapter. Excessive adapter
// layers add indirection and make the data flow hard to trace. The threshold
// is deliberately high (3) to avoid flagging a single legitimate adapter.
//
// Gated on the project importing both command and decider/event modules — the
// adapter layers this rule targets bridge between command handlers and the
// decider/event store. Non-CQRS projects (e.g., HTTP API gateways) may
// legitimately have many adapter types for unrelated reasons.
//
//nolint:ireturn // factory returns public interface
func NewE011Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E011-excessive-adapter-layers",
		func(_ context.Context) ([]finding.Finding, error) {
			hasCommand := importsPathSuffix(ctx, "go-cqrs-lite/command")
			hasDeciderOrEvent := importsPathSuffix(ctx, "go-cqrs-lite/decider") ||
				importsPathSuffix(ctx, "go-cqrs-lite/event")
			if !hasCommand || !hasDeciderOrEvent {
				return nil, nil
			}

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
