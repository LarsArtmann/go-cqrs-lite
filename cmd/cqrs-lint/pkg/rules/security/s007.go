package security

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// S007: In-memory session/token store in a production server.
// Session state held only in memory is lost on every restart, forcing every
// user to re-authenticate. This fires only for server projects (HasServer):
// in CLIs, batch jobs, and tests, an in-memory store is harmless. Detection
// is a conjunction of two lexical signals on a constructor call or composite
// literal — a volatile-storage indicator ("inmemory"/"memory") AND an
// auth-state indicator ("session", or "token"+"store"). Each axis
// independently suppresses false positives (e.g. memory.NewStore, an event
// store, lacks the auth indicator; NewInMemoryTokenBucket, a rate limiter,
// lacks "store").
//
//nolint:ireturn // factory returns public interface
func NewS007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S007-in-memory-session-store",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				// Evaluate per-module: an in-memory session store is only a
				// concern when THIS file's module runs a server. Using the
				// primary profile would flag library files when an example
				// sub-module happens to have a server.
				if !ctx.ProfileForFile(gf.Path).HasServer {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					name, ok := constructionName(n)
					if !ok {
						return true
					}

					if !isInMemorySessionTokenStore(name) {
						return true
					}

					pos := ctx.Fset.Position(n.Pos())

					f, err := finding.NewBuilder(
						"S007", toolName,
						fmt.Sprintf(
							"In-memory session/token store %q — session state is lost on restart, forcing re-authentication",
							name,
						),
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategorySecurity).
						WithConfidence(finding.ConfidenceMedium).
						WithSuggestion(
							"Use a persistent session/token store (Redis, SQL-backed) so sessions survive restarts",
						).
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// constructionName returns the identifier text of a store construction site —
// a function call (NewInMemorySessionStore()) or a composite literal
// (InMemorySessionStore{}) — including any package selector so that
// session.NewInMemoryStore() is recognized via its package name.
func constructionName(n ast.Node) (string, bool) {
	switch node := n.(type) {
	case *ast.CallExpr:
		return exprName(node.Fun), true
	case *ast.CompositeLit:
		return exprName(node.Type), true
	default:
		return "", false
	}
}

// exprName renders an expression as a dotted identifier string. For a
// selector pkg.Foo it yields "pkg.foo"; for a bare ident it yields the name.
func exprName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprName(e.X) + "." + e.Sel.Name
	default:
		return ""
	}
}

// isInMemorySessionTokenStore applies the two-signal conjunction. The token
// case requires an accompanying "store" to avoid matching rate-limiter
// helpers such as NewInMemoryTokenBucket.
func isInMemorySessionTokenStore(name string) bool {
	n := strings.ToLower(name)
	if !strings.Contains(n, "inmemory") && !strings.Contains(n, "memory") {
		return false
	}

	if strings.Contains(n, "session") {
		return true
	}

	return strings.Contains(n, "token") && strings.Contains(n, "store")
}
