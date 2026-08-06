package adoption

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// eventCount returns the number of distinct event types emitted by the project.
func eventCount(ctx *analyzer.AnalysisContext) int {
	return len(ctx.Registry.EventTypesEmitted)
}

// distinctAggregateCount returns the number of distinct aggregate types
// inferred from event type prefixes (the segment before the first dot).
// Event types without dots each count as a separate aggregate.
func distinctAggregateCount(ctx *analyzer.AnalysisContext) int {
	aggregates := make(map[string]bool)

	for eventType := range ctx.Registry.EventTypesEmitted {
		prefix := eventType
		if idx := strings.Index(eventType, "."); idx > 0 {
			prefix = eventType[:idx]
		}

		aggregates[prefix] = true
	}

	return len(aggregates)
}

// hasPIIInEventPayloads scans event payload structs for PII-like field names.
// Returns the position of the first PII field found.
func hasPIIInEventPayloads(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	piiFields := []string{
		"email", "phone", "ssn", "password", "address",
		"creditcard", "credit_card", "passport", "iban",
		"national_id", "dob", "birthdate",
	}

	for _, gf := range ctx.GoFiles {
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
								return ctx.Fset.Position(field.Pos()), true
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
func hasTimeBasedPatterns(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	timeFns := []string{"AfterFunc", "NewTimer", "Tick", "After", "NewTicker"}

	for _, fn := range timeFns {
		if pos, ok := firstCallPos(ctx, "time", fn); ok {
			return pos, true
		}
	}

	for _, prefix := range []string{"Expire", "Timeout", "Deadline", "Schedule", "Cancel"} {
		if pos, ok := firstFuncDeclPos(ctx, prefix); ok {
			return pos, true
		}
	}

	return token.Position{}, false
}

// hasTraversalPatterns detects signals that the domain needs graph-like
// traversal (recursive queries, ancestry, path-finding).
func hasTraversalPatterns(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	keywords := []string{
		"Traverse", "Ancestor", "Descendant", "ShortestPath",
		"Path", "Neighbor", "Adjacency", "Hierarchy",
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			for _, kw := range keywords {
				if strings.Contains(fn.Name.Name, kw) {
					return ctx.Fset.Position(fn.Pos()), true
				}
			}
		}
	}

	for _, gf := range ctx.GoFiles {
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
			return ctx.Fset.Position(gf.AST.Pos()), true
		}
	}

	return token.Position{}, false
}

// manualSortPatterns are the function calls that indicate in-memory sorting.
// Used by F022 to detect sort.Slice/slices.SortFunc without metaengine pushdown.
var manualSortPatterns = []struct {
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
// frameworks are used for manual HTTP handler registration, which is an
// alternative to go-cqrs-lite transport/http + transport/grpc.
func hasWebFrameworkHandlers(ctx *analyzer.AnalysisContext) bool {
	for _, prefix := range webFrameworkImportPaths {
		if importsPath(ctx, prefix) {
			return true
		}
	}

	return false
}
