package adoption

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// eventCount returns the number of distinct event types emitted by the project.
func eventCount(ctx *analyzer.AnalysisContext) int {
	return len(ctx.Registry.EventTypesEmitted)
}

// eventCountIn returns the number of distinct event types emitted from files
// in the given slice. Used by per-module coaching rules to count events per
// module instead of workspace-wide.
func eventCountIn(ctx *analyzer.AnalysisContext, files []*analyzer.GoFile) int {
	paths := make(map[string]bool, len(files))
	for _, gf := range files {
		paths[gf.Path] = true
	}

	count := 0
	for _, emission := range ctx.Registry.EventTypesEmitted {
		if paths[emission.File] {
			count++
		}
	}

	return count
}

// distinctAggregateCountIn returns the number of distinct aggregate types
// emitted from files in the given slice. Used by per-module coaching rules.
func distinctAggregateCountIn(ctx *analyzer.AnalysisContext, files []*analyzer.GoFile) int {
	paths := make(map[string]bool, len(files))
	for _, gf := range files {
		paths[gf.Path] = true
	}

	aggregates := make(map[string]bool)
	for eventType, emission := range ctx.Registry.EventTypesEmitted {
		if !paths[emission.File] {
			continue
		}

		prefix := eventType
		if idx := strings.Index(eventType, "."); idx > 0 {
			prefix = eventType[:idx]
		}

		aggregates[prefix] = true
	}

	return len(aggregates)
}

func hasPIIInEventPayloadsIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
) (token.Position, bool) {
	piiFields := []string{
		"email", "phone", "ssn", "password", "address",
		"creditcard", "credit_card", "passport", "iban",
		"national_id", "dob", "birthdate",
	}

	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				st, ok := ts.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					continue
				}

				if !lintutil.LooksLikeEventPayload(ts.Name.Name, gf.Path) {
					continue
				}

				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						lower := strings.ToLower(name.Name)
						for _, pii := range piiFields {
							if strings.Contains(lower, pii) {
								return fset.Position(field.Pos()), true
							}
						}
					}
				}
			}
		}
	}

	return token.Position{}, false
}

// hasTimeBasedPatterns detects signals that the domain has time-based business
// rules (deadlines, expirations, timeouts) by scanning for time.AfterFunc,
// time.NewTimer, or function names containing deadline/expire/timeout/schedule.
func hasTimeBasedPatternsIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
) (token.Position, bool) {
	timeFns := []string{"AfterFunc", "NewTimer", "Tick", "After", "NewTicker"}

	for _, fn := range timeFns {
		if pos, ok := firstCallPosIn(fset, files, "time", fn); ok {
			return pos, true
		}
	}

	for _, prefix := range []string{"Expire", "Timeout", "Deadline", "Schedule", "Cancel"} {
		if pos, ok := firstFuncDeclPosIn(fset, files, prefix); ok {
			return pos, true
		}
	}

	return token.Position{}, false
}

func hasTraversalPatternsIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
) (token.Position, bool) {
	// No bare "Path": it contains-matches unrelated identifiers such as
	// ChromePath/file-path setters (CV feedback, 2026-10-03); the graph
	// path-finding case stays covered by the dedicated "ShortestPath" entry.
	// Keywords match at identifier-part EDGES (the leading or trailing
	// CamelCase word run, case-insensitively) instead of as raw substrings,
	// so a keyword buried mid-name (gitHierarchyDump) no longer coaches
	// graph adoption. A keyword that LEADS the name still fires with a
	// trailing noun (ShortestPathHelper): no name-only model can separate
	// that shape from TraverseGraph, and F010 is a low-confidence hint
	// while the WITH RECURSIVE branch below carries the strong signal.
	keywords := []string{
		"Traverse", "Ancestor", "Descendant", "ShortestPath",
		"Neighbor", "Adjacency", "Hierarchy",
	}

	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			for _, kw := range keywords {
				if keywordAtIdentifierEdge(fn.Name.Name, kw) {
					return fset.Position(fn.Pos()), true
				}
			}
		}
	}

	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}

			val := strings.ToUpper(strings.Trim(lit.Value, `"`))
			if strings.Contains(val, "WITH RECURSIVE") {
				found = true
				return false
			}

			return true
		})

		if found {
			return fset.Position(gf.AST.Pos()), true
		}
	}

	return token.Position{}, false
}

// keywordAtIdentifierEdge reports whether keyword's CamelCase word parts
// align with the name's leading or trailing word parts, case-insensitively.
// A keyword part may extend into its word ("Ancestor" matches "Ancestors"),
// keeping the plural/variant recall that exact part equality would drop.
func keywordAtIdentifierEdge(name, keyword string) bool {
	nameParts := identifierParts(name)
	kwParts := identifierParts(keyword)
	if len(nameParts) < len(kwParts) {
		return false
	}

	partHasPrefix := func(part, kw string) bool {
		return strings.HasPrefix(strings.ToLower(part), strings.ToLower(kw))
	}

	lead := true
	for i, kw := range kwParts {
		if !partHasPrefix(nameParts[i], kw) {
			lead = false

			break
		}
	}
	if lead {
		return true
	}

	base := len(nameParts) - len(kwParts)
	for i, kw := range kwParts {
		if !partHasPrefix(nameParts[base+i], kw) {
			return false
		}
	}

	return true
}

// identifierParts splits a Go identifier into CamelCase word parts:
// lower→upper humps, acronym-run ends ("URLToken" → URL, Token),
// letter↔digit boundaries, and non-alphanumeric separators.
func identifierParts(name string) []string {
	parts := make([]string, 0, 2)

	for _, tok := range strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		parts = append(parts, splitCamelToken(tok)...)
	}

	return parts
}

// splitCamelToken splits one non-empty alphanumeric token at its CamelCase
// humps.
func splitCamelToken(tok string) []string {
	rs := []rune(tok)
	parts := make([]string, 0, 2)
	start := 0

	for i := 1; i < len(rs); i++ {
		hump := unicode.IsLower(rs[i-1]) && unicode.IsUpper(rs[i])
		acronymEnd := unicode.IsUpper(rs[i-1]) && unicode.IsUpper(rs[i]) &&
			i+1 < len(rs) && unicode.IsLower(rs[i+1])
		digitEdge := unicode.IsDigit(rs[i-1]) != unicode.IsDigit(rs[i])
		if hump || acronymEnd || digitEdge {
			parts = append(parts, string(rs[start:i]))
			start = i
		}
	}

	return append(parts, string(rs[start:]))
}

// manualSortPatterns are the function calls that indicate in-memory sorting.
// Used by F022 to detect sort.Slice/slices.SortFunc without metaengine pushdown.
var manualSortPatterns = []struct { //nolint:gochecknoglobals // static lookup table for in-memory sort detection
	pkg  string
	name string
}{
	{"sort", "Slice"},
	{"sort", "SliceStable"},
	{"slices", "SortFunc"},
	{"slices", "SortStableFunc"},
	{"slices", "Sort"},
}

// webFrameworkImportPaths are the import path prefixes of popular Go web
// frameworks whose presence signals manual HTTP handler registration.
var webFrameworkImportPaths = []string{ //nolint:gochecknoglobals // static lookup table
	"github.com/go-chi/chi",
	"github.com/gin-gonic/gin",
	"github.com/labstack/echo",
	"github.com/gofiber/fiber",
	"github.com/gorilla/mux",
	"github.com/julienschmidt/httprouter",
	"github.com/labstack/echo/v4",
}

// hasWebFrameworkHandlers reports whether the project imports any third-party
// web framework (chi, gin, echo, fiber, gorilla/mux, httprouter). These
// frameworks are used for manual HTTP handler registration, an alternative to
// the sanctioned delivery modules (watermill/, go-sse, cqrs-htmx).
func hasWebFrameworkHandlersIn(files []*analyzer.GoFile) bool {
	for _, prefix := range webFrameworkImportPaths {
		if importsPathIn(files, prefix) {
			return true
		}
	}

	return false
}
