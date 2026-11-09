package correctness

import (
	"context"
	"go/ast"
	"go/types"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// C043: named []byte payload passed to event.New.
// event.New's direct-use fast path (`case []byte:`) matches ONLY the
// unnamed slice type (plus jsontext.Value). A NAMED type with underlying
// []byte — message.Payload, RawEvent, wire.Bytes — falls through to the
// codec default branch and is silently CBOR/JSON-ENCODED (the pre-encoded
// bytes get wrapped, not passed through). Consumers decoding the payload
// into a struct get garbage or errors, and the bug is invisible at the
// call site because the doc says "[]byte is used directly".
//
// Type-gated: without TypesInfo the rule stays silent — it cannot
// distinguish named from unnamed slices syntactically, and a heuristic
// would false-positive on every plain []byte conversion.
//
//nolint:ireturn // factory returns public interface
func NewC043Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C043-named-byte-slice-payload",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if gf.Pkg == nil || gf.Pkg.TypesInfo == nil || gf.Pkg.TypesInfo.Types == nil {
					continue
				}

				for _, call := range eventNewCalls(gf) {
					payloadArg := call.Args[4]

					named, ok := namedByteSlice(gf, payloadArg)
					if !ok {
						continue
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := findingTemplate.Builder(
						"C043",
						named+" (named []byte) passed as event.New payload — "+
							"it is codec-encoded, not used directly",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("event.New's direct-use fast path matches only the " +
							"unnamed []byte (and jsontext.Value). Convert at the boundary: " +
							"[]byte(x), or pass the typed struct payload instead.").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err != nil {
						continue
					}

					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// eventNewCalls returns every event.New call in the file with the payload
// argument present (5th positional, index 4).
func eventNewCalls(gf *analyzer.GoFile) []*ast.CallExpr {
	var calls []*ast.CallExpr

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "New" {
			return true
		}

		if !analyzer.IsQualifierFor(gf, sel, "go-cqrs-lite/event") {
			return true
		}

		if len(call.Args) < 5 {
			return true
		}

		calls = append(calls, call)

		return true
	})

	return calls
}

// namedByteSlice reports whether expr's static type is a NAMED type whose
// underlying type is []byte (excluding jsontext.Value, the one named
// []byte the fast path accepts). Returns the type name when it is.
func namedByteSlice(gf *analyzer.GoFile, expr ast.Expr) (string, bool) {
	tv, ok := gf.Pkg.TypesInfo.Types[expr]
	if !ok {
		return "", false
	}

	named, ok := tv.Type.(*types.Named)
	if !ok {
		return "", false
	}

	slice, ok := named.Underlying().(*types.Slice)
	if !ok {
		return "", false
	}

	elem, ok := slice.Elem().(*types.Basic)
	if !ok || elem.Kind() != types.Byte {
		return "", false
	}

	if obj := named.Obj(); obj != nil && obj.Pkg() != nil &&
		obj.Pkg().Path() == "encoding/json/jsontext" && obj.Name() == "Value" {
		return "", false
	}

	if obj := named.Obj(); obj != nil {
		return obj.Name(), true
	}

	return named.String(), true
}
