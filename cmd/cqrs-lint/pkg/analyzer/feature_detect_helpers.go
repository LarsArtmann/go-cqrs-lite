package analyzer

import (
	"go/ast"
	"os"
	"sort"
	"strings"
)

func groupGoFilesByModule(ctx *AnalysisContext) map[string][]*GoFile {
	out := map[string][]*GoFile{}
	for _, gf := range ctx.GoFiles {
		if gf.ModuleDir == "" {
			continue
		}
		out[gf.ModuleDir] = append(out[gf.ModuleDir], gf)
	}
	return out
}

// pathDepth counts path separators in dir, used to order modules shallowest-
// first so the project root module sorts before nested example/demo modules.
func pathDepth(dir string) int {
	if dir == "" {
		return 0
	}
	count := 1
	for i := range len(dir) {
		if dir[i] == os.PathSeparator {
			count++
		}
	}
	return count
}

// financialKeywords are event/command type name fragments that indicate a
// financial domain. When any of these appear, the domain is classified as
// financial, which escalates security and money-handling rule severities.
var financialKeywords = []string{ //nolint:gochecknoglobals // constant lookup table, package-level is correct
	"amount",
	"balance",
	"payment",
	"invoice",
	"salary",
	"transaction",
	"transfer",
	"deposit",
	"withdraw",
	"refund",
	"price",
	"cost",
	"fee",
	"tax",
	"currency",
	"bank",
	"wallet",
	"ledger",
	"billing",
	"payroll",
}

// detectDomainRegistry scans event and command type names for domain-specific
// keywords. Returns DomainFinancial when financial keywords are found,
// DomainUnknown otherwise.
func detectDomainRegistry(registry *CQRSRegistry) DomainKind {
	check := func(name string) bool {
		lower := strings.ToLower(name)
		for _, kw := range financialKeywords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
		return false
	}

	for eventType := range registry.EventTypesEmitted {
		if check(eventType) {
			return DomainFinancial
		}
	}

	for cmdType := range registry.CommandTypesRegistered {
		if check(cmdType) {
			return DomainFinancial
		}
	}

	return DomainUnknown
}

// detectSoftDeleteRegistry returns true if any emitted event type name contains
// words associated with soft-delete (Deleted, Removed, Archived, Tombstoned).
func detectSoftDeleteRegistry(registry *CQRSRegistry) bool {
	for eventType := range registry.EventTypesEmitted {
		lower := strings.ToLower(eventType)
		for _, keyword := range []string{"deleted", "removed", "archived", "tombstoned"} {
			if strings.Contains(lower, keyword) {
				return true
			}
		}
	}

	return false
}

// httpFrameworkImports lists Go HTTP framework module paths whose presence in
// go.mod / imports is a strong server signal. A project importing Gin/Echo/
// Fiber/Chi is serving HTTP.
var httpFrameworkImports = []string{
	"gin-gonic/gin",
	"labstack/echo",
	"gofiber/fiber",
	"go-chi/chi",
}

func isHTTPFrameworkImport(path string) bool {
	for _, fw := range httpFrameworkImports {
		if strings.Contains(path, fw) {
			return true
		}
	}
	return false
}

// astCallSignals collects the signals detected during AST call-expression
// scanning (Pass 2 of detectFeatureSignals).
type astCallSignals struct {
	hasDispatcher    bool
	hasDispatch      bool
	hasSnapshotUsage bool
	hasTLS           bool
	hasShutdown      bool
	hasHealthRoute   bool
}

// scanASTCalls inspects AST call expressions for server detection, command-flow
// signals, snapshot usage, tracing wiring, store-backend refinement, and
// ServerLocal signals. Also scans for health endpoint string literals.
func scanASTCalls(
	gofiles []*GoFile,
	fp *FeatureProfile,
	hasOTelImport, hasHTTPFramework bool,
) astCallSignals {
	var sig astCallSignals

	for _, gf := range gofiles {
		if gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			method := sel.Sel.Name

			// Server detection: http.ListenAndServe, http.ListenAndServeTLS,
			// http.Server.Serve, grpc.NewServer, gin engine.Run.
			if method == "ListenAndServe" || method == "ListenAndServeTLS" ||
				method == "Serve" {
				fp.HasServer = true
			}
			// Gin/Echo/Fiber engine.Run() starts an HTTP listener.
			if method == "Run" && hasHTTPFramework {
				fp.HasServer = true
			}
			if method == "Listen" || method == "NewListener" {
				if strings.Contains(SelectorPackage(sel), "tls") ||
					strings.Contains(SelectorPackage(sel), "net") {
					fp.HasServer = true
				}
			}
			if method == "NewServer" {
				pkgName := SelectorPackage(sel)
				if strings.Contains(pkgName, "grpc") || strings.Contains(pkgName, "http") {
					fp.HasServer = true
				}
			}

			// Command-flow detection.
			if method == "NewDispatcher" {
				sig.hasDispatcher = true
			}
			if method == "Dispatch" {
				sig.hasDispatch = true
			}

			// Snapshot usage detection.
			if method == "WithSnapshotStore" || method == "WithSnapshotStrategy" ||
				method == "NewSnapshotStore" {
				sig.hasSnapshotUsage = true
			}

			// Tracing wiring detection.
			if method == "EventTracing" || method == "CommandTracing" ||
				method == "NewOTelBundle" || method == "EventPublishTracing" {
				if hasOTelImport {
					fp.Tracing = TracingOn
				}
			}

			// Store-backend refinement: when the feature profile classified the
			// store as "custom" (no stack bundle), inspect constructor calls to
			// determine the ACTUAL backend.
			if fp.Store == StoreCustom {
				pkgName := SelectorPackage(sel)
				if pkgName == "storage" {
					switch {
					case strings.Contains(method, "SQLite"):
						fp.Store = StoreSQLite
					case strings.Contains(method, "Postgres"):
						fp.Store = StorePostgres
					case strings.Contains(method, "Pebble"):
						fp.Store = StorePebble
					}
				}
			}

			// Metaengine pushdown detection: FilterOnField / SortOnField calls
			// indicate declarative pushdown adoption.
			if method == "FilterOnField" || method == "SortOnField" {
				fp.MetaenginePushdown = true
			}

			// ServerLocal signals: detect production server indicators so
			// we can classify embedded-dashboards (ListenAndServe without
			// TLS/Shutdown/health) as ServerLocal.
			if method == "ListenAndServeTLS" {
				sig.hasTLS = true
			}
			if method == "NewListener" || method == "Listen" {
				if strings.Contains(SelectorPackage(sel), "tls") {
					sig.hasTLS = true
				}
			}
			if method == "Shutdown" || method == "GracefulClose" {
				sig.hasShutdown = true
			}

			return true
		})

		// Scan for health endpoint string literals (production signal).
		ast.Inspect(gf.AST, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok {
				return true
			}

			val := strings.Trim(lit.Value, `"`)
			switch val {
			case "/health", "/healthz", "/ready", "/readyz", "/livez":
				sig.hasHealthRoute = true
			}

			return true
		})
	}

	return sig
}

// sortModuleDirs returns a shallowest-first sorted list of module dirs.
func sortModuleDirs(dirSet map[string]struct{}) []string {
	dirs := make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}
	sort.Slice(dirs, func(i, j int) bool {
		di, dj := pathDepth(dirs[i]), pathDepth(dirs[j])
		if di != dj {
			return di < dj
		}
		return dirs[i] < dirs[j]
	})
	return dirs
}
