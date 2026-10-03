package adoption

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// Pushdown utilization coaching (nsfw-classifier feedback, 2026-10-03):
// adopting metaengine and adopting PUSHDOWN are different steps. The F022/F023
// adoption rules skip metaengine importers entirely, so the population that
// needs pushdown coaching most — metaengine users Go-side filtering Query
// results at scale — was never coached. These detectors fire for importers:
// manual sort/filter sites over a registered Query's R type whose declaration
// lacks SortOnField/FilterOnField, gated on the declaration's Volume (below
// pushdownVolumeThreshold, Go-side filtering is legitimately fine and stays
// noise-free).

// pushdownVolumeThreshold is the Volume(n) magnitude above which Go-side
// sort/filter over Query results is coached (~1k rows, per the feedback).
const pushdownVolumeThreshold = 1000

// memoryEngineNote explains why declaring pushdown is worthwhile even when
// the memory engine serves the collection today.
const memoryEngineNote = "Declare it even on the memory engine — the planner " +
	"ignores pushdown there, so the declaration is free and goes live the " +
	"moment a SQL DSN is configured."

type pushdownScope struct {
	ctx *analyzer.AnalysisContext
}

func newPushdownScope(ctx *analyzer.AnalysisContext) pushdownScope {
	return pushdownScope{ctx: ctx}
}

// detectPushdownSortMisses reports manual sort call sites (sort.Slice,
// slices.SortFunc, slices.SortStableFunc) whose operand is a slice of a
// registered Query's R type where the declaration lacks SortOnField.
func (p pushdownScope) detectPushdownSortMisses() []finding.Finding {
	var out []finding.Finding

	forEachTypedFile(p.ctx, func(gf *analyzer.GoFile) {
		ast.Inspect(gf.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			matched := slices.ContainsFunc(manualSortPatterns, func(pt struct {
				pkg  string
				name string
			},
			) bool {
				return pkg.Name == pt.pkg && sel.Sel.Name == pt.name
			})
			if !matched || len(call.Args) == 0 {
				return true
			}

			query, ok := p.queryForElement(gf, call.Args[0])
			if !ok || query.HasSortOnField {
				return true
			}

			out = append(out, p.sortFinding(call, query))

			return true
		})
	})

	return out
}

// detectPushdownFilterMisses reports range loops over a registered Query's R
// slice whose body compares fields of the loop variable, when the declaration
// lacks FilterOnField. The compared field names become the suggestion's
// columns.
func (p pushdownScope) detectPushdownFilterMisses() []finding.Finding {
	var out []finding.Finding

	forEachTypedFile(p.ctx, func(gf *analyzer.GoFile) {
		ast.Inspect(gf.AST, func(n ast.Node) bool {
			rng, ok := n.(*ast.RangeStmt)
			if !ok {
				return true
			}

			query, ok := p.queryForElement(gf, rng.X)
			if !ok || query.HasFilterOnField {
				return true
			}

			fields := comparedFieldsOfRangeVar(rng)
			if len(fields) == 0 {
				return true
			}

			out = append(out, p.filterFinding(rng, query, fields))

			return true
		})
	})

	return out
}

// queryForElement resolves the named element type of expr (slice of R, or R)
// to a registered Query declaration that passes the Volume gate. Attribution
// is by R type name: when several declarations share the same R the site
// cannot be attributed to one collection, so coaching stays silent (a missed
// hint, never a wrong-collection finding).
func (p pushdownScope) queryForElement(
	gf *analyzer.GoFile,
	expr ast.Expr,
) (analyzer.QueryDeclInfo, bool) {
	typeName := elementTypeName(gf, expr)
	if typeName == "" {
		return analyzer.QueryDeclInfo{}, false
	}

	var match analyzer.QueryDeclInfo

	found := false

	for _, q := range p.ctx.Registry.MetaengineQueries {
		if q.ResultType != typeName {
			continue
		}

		if found {
			return analyzer.QueryDeclInfo{}, false
		}

		if !q.HasVolume || q.Volume < pushdownVolumeThreshold {
			return analyzer.QueryDeclInfo{}, false
		}

		match = q
		found = true
	}

	return match, found
}

func (p pushdownScope) sortFinding(
	call *ast.CallExpr,
	query analyzer.QueryDeclInfo,
) finding.Finding {
	pos := p.ctx.Fset.Position(call.Pos())
	collection := query.Collection
	if collection == "" {
		collection = query.ResultType
	}

	f, err := findingTemplate.Builder(
		finding.RuleName("F022"),
		"Query results for "+collection+" ("+query.ResultType+", Volume "+itoa(query.Volume)+") "+
			"are sorted in Go — declare SortOnField for engine-side ORDER BY",
		finding.SeverityInfo,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryBestPractice).
		WithConfidence(finding.ConfidenceMedium).
		WithFixStrategy(finding.FixStrategySuggest).
		WithSuggestion(
			"Add metaengine.SortOnField[" + query.ResultType + "](\"column\", false) to the " +
				collection + " Query declaration — the declaration allow-lists the sort " +
				"column; the planner pushes ORDER BY to the engine. " + memoryEngineNote,
		).
		WithSnippet(p.ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	if err != nil {
		return finding.Finding{}
	}

	return f
}

func (p pushdownScope) filterFinding(
	rng *ast.RangeStmt,
	query analyzer.QueryDeclInfo,
	fields []string,
) finding.Finding {
	pos := p.ctx.Fset.Position(rng.Pos())
	collection := query.Collection
	if collection == "" {
		collection = query.ResultType
	}

	primary := fields[0]

	f, err := findingTemplate.Builder(
		finding.RuleName("F023"),
		"Query results for "+collection+" ("+query.ResultType+", Volume "+itoa(query.Volume)+") "+
			"are filtered in Go on "+strings.Join(fields, ", ")+" — declare FilterOnField "+
			"for engine-side WHERE",
		finding.SeverityInfo,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryBestPractice).
		WithConfidence(finding.ConfidenceMedium).
		WithFixStrategy(finding.FixStrategySuggest).
		WithSuggestion(
			"Two layers — the declaration allow-lists columns, read-time binds values: " +
				"metaengine.FilterOnField[" + query.ResultType + "](\"" + primary + "\", metaengine.FilterGe) " +
				"in the declaration, then reader.Scan(ctx, metaengine.WithFilter(\"" + primary +
				"\", metaengine.FilterGe, value), metaengine.WithLimit(pageSize)) at the call site. " +
				"Filter values are runtime data — never put them in the declaration. " + memoryEngineNote,
		).
		WithSnippet(p.ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	if err != nil {
		return finding.Finding{}
	}

	return f
}

// comparedFieldsOfRangeVar returns the deduplicated, sorted field names the
// range body compares on the loop variable (`if row.Score >= x` → "Score").
func comparedFieldsOfRangeVar(rng *ast.RangeStmt) []string {
	varNames := make(map[string]bool)
	for _, ident := range []*ast.Ident{rangeKeyIdent(rng), rangeValueIdent(rng)} {
		if ident != nil {
			varNames[ident.Name] = true
		}
	}

	seen := make(map[string]bool)
	var fields []string

	ast.Inspect(rng.Body, func(n ast.Node) bool {
		bin, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}

		switch bin.Op {
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		default:
			return true
		}

		for _, side := range []ast.Expr{bin.X, bin.Y} {
			sel, ok := side.(*ast.SelectorExpr)
			if !ok {
				continue
			}

			ident, ok := sel.X.(*ast.Ident)
			if !ok || !varNames[ident.Name] || sel.Sel == nil {
				continue
			}

			if !seen[sel.Sel.Name] {
				seen[sel.Sel.Name] = true
				fields = append(fields, sel.Sel.Name)
			}
		}

		return true
	})

	slices.Sort(fields)

	return fields
}

func rangeKeyIdent(rng *ast.RangeStmt) *ast.Ident {
	if id, ok := rng.Key.(*ast.Ident); ok {
		return id
	}
	return nil
}

func rangeValueIdent(rng *ast.RangeStmt) *ast.Ident {
	if id, ok := rng.Value.(*ast.Ident); ok {
		return id
	}
	return nil
}

// forEachTypedFile invokes fn for every non-test file with usable type info;
// utilization coaching is type-linked and cannot run on syntax-only loads.
func forEachTypedFile(ctx *analyzer.AnalysisContext, fn func(gf *analyzer.GoFile)) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest || gf.Pkg == nil || gf.Pkg.TypesInfo == nil {
			continue
		}

		fn(gf)
	}
}

// elementTypeName resolves the named type behind expr — unwrapping slices and
// pointers — for R-type linkage against Query declarations. Empty when the
// expression does not resolve to a named type.
func elementTypeName(gf *analyzer.GoFile, expr ast.Expr) string {
	t := gf.Pkg.TypesInfo.TypeOf(expr)
	if t == nil {
		return ""
	}

	if sl, ok := t.(*types.Slice); ok {
		t = sl.Elem()
	}

	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	named, ok := t.(*types.Named)
	if !ok {
		return ""
	}

	return named.Obj().Name()
}
