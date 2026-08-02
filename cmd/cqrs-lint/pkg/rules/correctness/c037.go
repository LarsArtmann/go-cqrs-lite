package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// C037: Snapshot/event codec mismatch.
//
// Detects a snapshot store configured with a different codec than the
// repository/event store. A snapshot serialized with CBOR cannot be decoded by
// a repository using JSON (or vice versa): the snapshot either fails to load or
// silently deserializes into a zero/corrupt state, defeating the snapshot
// optimization and potentially producing wrong aggregate state.
//
// Patterns tracked:
//   - Repository codec:  decider.WithCodec(codec.XXXCodec{})
//   - Snapshot codec:    snapshot.NewTypedStore(store, codec.XXXCodec{})
//
// Events are self-describing (each carries its encoding stamp), but snapshots
// and typed stores are blind — the codec must match exactly between write and
// read, which is why this mismatch is a correctness bug rather than a style note.
//
//nolint:ireturn // factory returns public interface
func NewC037Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C037-snapshot-codec-mismatch",
		func(_ context.Context) ([]finding.Finding, error) {
			var (
				repoCodec string
				snaps     []snapshotSite
			)

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					if name, ok := codecFromWithCodec(call); ok && repoCodec == "" {
						repoCodec = name
					}

					if name, ok := codecFromSnapshotStore(call); ok {
						snaps = append(snaps, snapshotSite{name: name, pos: ctx.Fset.Position(call.Pos())})
					}

					return true
				})
			}

			if repoCodec == "" || len(snaps) == 0 {
				return nil, nil
			}

			var findings []finding.Finding

			for _, s := range snaps {
				if s.name == "" || s.name == repoCodec {
					continue
				}

				f, err := finding.NewBuilder(
					"C037", toolName,
					fmt.Sprintf(
						"Snapshot store uses %s codec but repository uses %s — "+
							"snapshot cannot be decoded, loads as corrupt/zero state",
						s.name, repoCodec,
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(s.pos.Filename), s.pos.Line, s.pos.Column),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceHigh).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion(fmt.Sprintf(
						"Pass the same codec to snapshot.NewTypedStore: use %s to match the repository",
						repoCodec,
					)).
					WithSnippet(ctx.SourceLine(s.pos.Filename, s.pos.Line)).
					Build()
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}

type snapshotSite struct {
	name string
	pos  token.Position
}

// codecFromWithCodec extracts the codec type name from a decider.WithCodec(...)
// call (generic or inferred). Returns "" when not applicable.
func codecFromWithCodec(call *ast.CallExpr) (string, bool) {
	sel, ok := unwrapIndex(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	if sel.Sel.Name != "WithCodec" {
		return "", false
	}

	if len(call.Args) == 0 {
		return "", false
	}

	return codecTypeName(call.Args[0])
}

// codecFromSnapshotStore extracts the codec type name from
// snapshot.NewTypedStore(store, codec). The codec is the second positional arg.
func codecFromSnapshotStore(call *ast.CallExpr) (string, bool) {
	sel, ok := unwrapIndex(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	if sel.Sel.Name != "NewTypedStore" {
		return "", false
	}

	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "snapshot" {
		return "", false
	}

	if len(call.Args) < 2 {
		return "", false
	}

	return codecTypeName(call.Args[1])
}

// unwrapIndex strips a single type-argument bracket from a generic call Fun,
// handling both IndexExpr (one type arg: f[T]) and IndexListExpr (several:
// f[T, U]). Non-generic expressions pass through unchanged.
func unwrapIndex(fun ast.Expr) ast.Expr {
	if idx, ok := fun.(*ast.IndexExpr); ok {
		return idx.X
	}

	if il, ok := fun.(*ast.IndexListExpr); ok {
		return il.X
	}

	return fun
}

// codecTypeName extracts the codec name from an expression like
// codec.JSONCodec{} (composite literal) or codec.JSONCodec (selector). It
// returns the trailing type identifier, e.g. "JSONCodec".
func codecTypeName(expr ast.Expr) (string, bool) {
	if cl, ok := expr.(*ast.CompositeLit); ok {
		return codecTypeName(cl.Type)
	}

	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	name := sel.Sel.Name

	// Only treat it as a codec when the name looks like one. This avoids
	// false positives from unrelated NewTypedStore/store helpers.
	if !strings.HasSuffix(name, "Codec") {
		return "", false
	}

	return name, true
}
