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

// C037: Typed store codec mismatch.
//
// Detects a typed store (snapshot, command, query, or kv) configured with a
// different codec than the repository/event store. A store serialized with
// CBOR cannot be decoded by a repository using JSON (or vice versa): the data
// either fails to load or silently deserializes into a zero/corrupt state.
//
// Patterns tracked:
//   - Repository codec:  decider.WithCodec(codec.XXXCodec{})
//   - Snapshot codec:    snapshot.NewTypedStore(store, codec.XXXCodec{})
//   - Command codec:     command.NewTypedCommandStore(store, codec.XXXCodec{})
//   - Query codec:       query.NewTypedQueryStore(store, codec.XXXCodec{})
//   - KV store codec:    kv.WithTypedCodec(codec.XXXCodec{})  (passed to kv.NewTypedStore)
//
// Events are self-describing (each carries its encoding stamp), but typed
// stores are blind — the codec must match exactly between write and read,
// which is why this mismatch is a correctness bug rather than a style note.
//
//nolint:ireturn // factory returns public interface
func NewC037Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C037-typed-store-codec-mismatch",
		func(_ context.Context) ([]finding.Finding, error) {
			var (
				repoCodec string
				sites     []codecSite
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

					if desc, name, ok := codecFromTypedStore(call); ok {
						sites = append(sites, codecSite{
							storeDesc: desc,
							name:      name,
							pos:       ctx.Fset.Position(call.Pos()),
						})
					}

					return true
				})
			}

			if repoCodec == "" || len(sites) == 0 {
				return nil, nil
			}

			var findings []finding.Finding

			for _, s := range sites {
				if s.name == "" || s.name == repoCodec {
					continue
				}

				f, err := finding.NewBuilder(
					"C037", toolName,
					fmt.Sprintf(
						"%s uses %s codec but repository uses %s — "+
							"store cannot be decoded, loads as corrupt/zero state",
						s.storeDesc, s.name, repoCodec,
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(s.pos.Filename), s.pos.Line, s.pos.Column),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceHigh).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion(fmt.Sprintf(
						"Use %s to match the repository codec",
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

type codecSite struct {
	storeDesc string
	name      string
	pos       token.Position
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

// codecFromTypedStore extracts the codec type name and a human-readable store
// description from any typed store constructor that accepts a codec. Supports:
//   - snapshot.NewTypedStore(store, codec)       — 2nd positional arg
//   - command.NewTypedCommandStore(store, codec) — 2nd positional arg
//   - query.NewTypedQueryStore(store, codec)     — 2nd positional arg
//   - kv.WithTypedCodec(codec)                   — 1st arg (option for kv.NewTypedStore)
func codecFromTypedStore(call *ast.CallExpr) (storeDesc, codecName string, ok bool) {
	sel, ok := unwrapIndex(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}

	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", "", false
	}

	switch {
	case pkg.Name == "snapshot" && sel.Sel.Name == "NewTypedStore":
		if len(call.Args) < 2 {
			return "", "", false
		}
		name, ok := codecTypeName(call.Args[1])
		return "Snapshot store", name, ok

	case pkg.Name == "command" && sel.Sel.Name == "NewTypedCommandStore":
		if len(call.Args) < 2 {
			return "", "", false
		}
		name, ok := codecTypeName(call.Args[1])
		return "Command store", name, ok

	case pkg.Name == "query" && sel.Sel.Name == "NewTypedQueryStore":
		if len(call.Args) < 2 {
			return "", "", false
		}
		name, ok := codecTypeName(call.Args[1])
		return "Query store", name, ok

	case pkg.Name == "kv" && sel.Sel.Name == "WithTypedCodec":
		if len(call.Args) == 0 {
			return "", "", false
		}
		name, ok := codecTypeName(call.Args[0])
		return "KV store", name, ok
	}

	return "", "", false
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
