package performance

import (
	"context"
	"fmt"
	"go/ast"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// eventPayloadSuffixes are struct name suffixes that identify event payloads.
var eventPayloadSuffixes = []string{
	"Created", "Updated", "Deleted", "Removed", "Added", "Changed", "Event",
}

// largePayloadFieldThreshold is the minimum number of fields in an event
// payload struct to trigger P009 (>10 fields suggests a large payload).
const largePayloadFieldThreshold = 10

// P009: JSON codec for large payloads.
// Detects event payload structs with many fields (>10) or []byte fields
// in projects that use codec.JSONCodec. CBOR produces ~35% smaller payloads
// for the same data, which reduces storage and network transfer costs.
//
//nolint:ireturn // factory returns public interface
func NewP009Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	usesJSON := projectUsesJSONCodec(ctx)

	return finding.NamedDetectorFunc(
		"P009-json-codec-large-payloads",
		func(_ context.Context) ([]finding.Finding, error) {
			// Only flag if the project actually uses JSON codec.
			if !usesJSON {
				return nil, nil
			}

			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					ts, ok := n.(*ast.TypeSpec)
					if !ok || ts.Name == nil {
						return true
					}

					name := ts.Name.Name
					if !isEventPayloadName(name) {
						return true
					}

					st, ok := ts.Type.(*ast.StructType)
					if !ok || st.Fields == nil {
						return true
					}

					fieldCount := structFieldCount(st)
					hasBytes := hasByteSliceField(st)

					if fieldCount <= largePayloadFieldThreshold && !hasBytes {
						return true
					}

					pos := ctx.Fset.Position(ts.Pos())

					var reason string
					switch {
					case hasBytes && fieldCount > largePayloadFieldThreshold:
						reason = fmt.Sprintf(
							"%d fields and []byte field — base64 inflation on JSON",
							fieldCount,
						)
					case hasBytes:
						reason = "[]byte field — base64 inflation on JSON (33% overhead)"
					default:
						reason = fmt.Sprintf("%d fields — large payload", fieldCount)
					}

					f, err := finding.NewBuilder(
						"P009", toolName,
						fmt.Sprintf(
							"Event payload %s uses JSON codec — %s; CBOR is ~35%% smaller",
							name, reason,
						),
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryPerformance).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Use codec.CBORCodec{} via event.WithCodec or " +
							"stack.WithEventCodec — events are self-describing, " +
							"so mixed JSON+CBOR streams decode correctly").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					lintutil.AppendBuild(&findings, f, err)

					return true
				})
			}

			return findings, nil
		},
	)
}

// isEventPayloadName returns true if the struct name ends with a common
// event-payload suffix (Created, Updated, Deleted, etc.).
func isEventPayloadName(name string) bool {
	return slices.ContainsFunc(eventPayloadSuffixes, func(s string) bool {
		return strings.HasSuffix(name, s)
	})
}
