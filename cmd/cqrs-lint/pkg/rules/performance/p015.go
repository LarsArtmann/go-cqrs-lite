package performance

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// P015: journal.ReadAll reachable from a query handler.
//
// A query handler that folds the ENTIRE journal (event.Journal.ReadAll)
// re-reads and re-decodes every event on the request path — O(journal) per
// query, growing linearly with history. Read models exist so queries never
// touch the whole journal: fold from the projection, an aggregate-bounded
// stream (Store.Load), or a counter/stat read model instead.
//
// Detection is multi-signal to keep the false-positive budget at zero:
// the ReadAll must be reachable from a system.RegisterQuery handler closure
// (directly or via one same-file helper hop), and the receiver must look
// like a journal (journal-ish identifier or an EventStore() call chain).
// Package-qualified calls (io.ReadAll) and event bus handlers (B019/P001's
// territory) never fire. One-shot Open/startup hydration that no handler
// reaches stays silent by construction.
//
//nolint:ireturn // factory returns public interface
func NewP015Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P015-journal-readall-in-query-handler",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				findings = append(findings, p015ScanFile(ctx, gf)...)
			}

			return findings, nil
		},
	)
}

// p015ScanFile finds RegisterQuery handler closures in one file and reports
// journal ReadAll calls reachable from each (directly or via a same-file
// helper — the ReadAll almost never sits inside the closure itself).
func p015ScanFile(ctx *analyzer.AnalysisContext, gf *analyzer.GoFile) []finding.Finding {
	helpers := p015JournalHelpers(gf)

	var findings []finding.Finding

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok || !strings.HasSuffix(sel.Sel.Name, "RegisterQuery") {
			return true
		}

		for _, arg := range call.Args {
			lit, ok := arg.(*ast.FuncLit)
			if !ok {
				continue
			}

			findings = append(findings, p015ScanHandler(ctx, lit.Body, helpers)...)
		}

		return true
	})

	return findings
}

// p015ScanHandler reports direct ReadAll sites inside the handler body plus
// one-hop sites: helpers called by name from the body whose own body folds
// the journal (the statsFromJournal production shape).
func p015ScanHandler(
	ctx *analyzer.AnalysisContext,
	body *ast.BlockStmt,
	helpers map[string]ast.Node,
) []finding.Finding {
	var findings []finding.Finding

	for _, site := range p015ReadAllSites(body) {
		if f, err := p015Finding(ctx, site); err == nil {
			findings = append(findings, f)
		}
	}

	for _, callee := range p015Callees(body) {
		site, ok := helpers[callee]
		if !ok {
			continue
		}

		if f, err := p015Finding(ctx, site); err == nil {
			findings = append(findings, f)
		}
	}

	return findings
}

// p015JournalHelpers maps same-file function names to their ReadAll call
// node when that function folds the journal — the one-hop reachable shape.
func p015JournalHelpers(gf *analyzer.GoFile) map[string]ast.Node {
	helpers := map[string]ast.Node{}

	for _, decl := range gf.AST.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Body == nil {
			continue
		}

		for _, site := range p015ReadAllSites(fn.Body) {
			helpers[fn.Name.Name] = site
		}
	}

	return helpers
}

// p015ReadAllSites collects ReadAll calls whose receiver plausibly is an
// event journal: a journal-ish local identifier or an EventStore() chain.
// Package-qualified calls (io.ReadAll) and unrelated receivers never match.
func p015ReadAllSites(body ast.Node) []ast.Node {
	var sites []ast.Node

	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "ReadAll" {
			return true
		}

		if p015JournalishRecv(sel.X) {
			sites = append(sites, call)
		}

		return true
	})

	return sites
}

// p015JournalishRecv reports whether the receiver expression names a journal
// (case-insensitive identifier match) or is a direct EventStore() result.
func p015JournalishRecv(x ast.Expr) bool {
	switch recv := x.(type) {
	case *ast.Ident:
		return p015JournalishName(recv.Name)
	case *ast.CallExpr:
		inner, ok := recv.Fun.(*ast.SelectorExpr)

		return ok && inner.Sel.Name == "EventStore"
	default:
		return false
	}
}

// p015JournalishName matches the identifier names a journal receiver carries
// in practice. Deliberately a closed set: `r`, `reader`, and friends must
// stay silent (csv.Reader.ReadAll is a real method with a real use).
func p015JournalishName(name string) bool {
	switch strings.ToLower(name) {
	case "journal", "jrn", "events", "evts", "store", "eventstore":
		return true
	default:
		return false
	}
}

// p015Callees collects plain-identifier function calls in the body — the
// one-hop candidates resolved against p015JournalHelpers.
func p015Callees(body ast.Node) []string {
	var names []string

	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if ident, ok := call.Fun.(*ast.Ident); ok {
			names = append(names, ident.Name)
		}

		return true
	})

	return names
}

// p015Finding builds the P015 finding at the offending ReadAll call site.
func p015Finding(ctx *analyzer.AnalysisContext, site ast.Node) (finding.Finding, error) {
	pos := ctx.Fset.Position(site.Pos())

	return findingTemplate.Builder(
		"P015",
		"journal.ReadAll reachable from a query handler — folds the entire "+
			"journal per query (O(journal) on the request path)",
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithConfidence(finding.ConfidenceMedium).
		WithFixStrategy(finding.FixStrategySuggest).
		WithSuggestion("Serve the query from the read model (metaengine projection) " +
			"or an aggregate-bounded stream (Store.Load) — not from a full journal fold").
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
}
