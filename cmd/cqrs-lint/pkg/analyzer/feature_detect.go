package analyzer

import (
	"go/ast"
	"os"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// DetectFeatures scans the analyzed project and returns a FeatureProfile
// describing which go-cqrs-lite features the consumer uses. This replaces the
// per-detector heuristics (isLocalOnlyProject, hasTombstoneLikeEvents,
// hasDispatch) with one centralized declaration that all detectors consult.
func DetectFeatures(ctx *AnalysisContext) FeatureProfile {
	return detectFeatureSignals(ctx.Packages, ctx.GoFiles, ctx.Registry)
}

// DetectFeaturesPerModule partitions the analyzed files/packages by their go.mod
// directory and runs feature detection once per module. The result is keyed by
// module directory path. This prevents a multi-module workspace from getting a
// single merged profile that is wrong for every individual module (e.g. an
// examples/ app's ListenAndServe flipping server=true for the library module).
//
// packagesByModule maps each module dir to the packages loaded from it; only
// BuildContext has this mapping (*packages.Package does not carry its module
// dir). The registry-derived signals (soft-delete, domain) use the global
// registry since those are weak, project-wide heuristics; the strong import/AST
// signals (store, server, command-flow, tracing, snapshot, transport,
// async-bus) are derived from each module's own files.
func DetectFeaturesPerModule(
	ctx *AnalysisContext,
	packagesByModule map[string][]*packages.Package,
) map[string]FeatureProfile {
	filesByModule := groupGoFilesByModule(ctx)

	// Union of module dirs from both partitions, ordered shallowest-first.
	dirSet := map[string]struct{}{}
	for dir := range packagesByModule {
		dirSet[dir] = struct{}{}
	}
	for dir := range filesByModule {
		dirSet[dir] = struct{}{}
	}
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

	profiles := make(map[string]FeatureProfile, len(dirs))
	for _, dir := range dirs {
		profiles[dir] = detectFeatureSignals(packagesByModule[dir], filesByModule[dir], ctx.Registry)
	}

	return profiles
}

// detectFeatureSignals runs the import + AST based detection passes over an
// explicit set of packages and files, then overlays the registry-derived
// soft-delete and domain signals from the (global) registry. This is the shared
// core used by both the workspace-wide DetectFeatures and per-module detection.
func detectFeatureSignals(
	pkgs []*packages.Package,
	gofiles []*GoFile,
	registry *CQRSRegistry,
) FeatureProfile {
	fp := FeatureProfile{
		Store:       StoreUnknown,
		CommandFlow: CommandFlowUnknown,
		Tracing:     TracingUnknown,
		Snapshot:    SnapshotUnknown,
	}

	hasSQLiteImport := false
	hasDispatcher := false
	hasDispatch := false
	hasOTelImport := false
	hasSnapshotImport := false
	hasSnapshotUsage := false
	hasTLS := false
	hasShutdown := false
	hasHealthRoute := false

	// Pass 1: import-based detection (store, tracing, snapshot presence).
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			continue
		}
		for _, imp := range pkg.Imports {
			if imp == nil {
				continue
			}
			path := imp.PkgPath

			if strings.Contains(path, "go-cqrs-lite/stack/sqlite") {
				fp.Store = StoreSQLite
			} else if strings.Contains(path, "go-cqrs-lite/stack/postgres") {
				fp.Store = StorePostgres
			} else if strings.Contains(path, "go-cqrs-lite/stack/mysql") {
				fp.Store = StoreMySQL
			} else if strings.Contains(path, "go-cqrs-lite/stack/pebble") {
				fp.Store = StorePebble
			} else if strings.Contains(path, "go-cqrs-lite/stack/memory") {
				fp.Store = StoreMemory
			} else if strings.Contains(path, "go-cqrs-lite/stack/turso") {
				fp.Store = StoreTurso
			} else if strings.Contains(path, "go-cqrs-lite/storage/") &&
				fp.Store == StoreUnknown {
				fp.Store = StoreCustom
			}

			if strings.Contains(path, "mattn/go-sqlite3") ||
				strings.Contains(path, "modernc.org/sqlite") {
				hasSQLiteImport = true
			}

			if strings.Contains(path, "go.opentelemetry.io") ||
				strings.Contains(path, "go-cqrs-lite/otel") {
				hasOTelImport = true
			}

			if strings.Contains(path, "go-cqrs-lite/snapshot") {
				hasSnapshotImport = true
			}

			if strings.Contains(path, "go-cqrs-lite/watermill") {
				fp.HasAsyncBus = true
			}

			if strings.Contains(path, "go-cqrs-lite/transport") ||
				strings.Contains(path, "cqrs-htmx") {
				fp.HasTransport = true
			}
		}
	}

	// If no stack preset was found but SQLite driver is imported, infer SQLite.
	if fp.Store == StoreUnknown && hasSQLiteImport {
		fp.Store = StoreSQLite
	}
	if fp.Store == StoreUnknown {
		fp.Store = StoreNone
	}

	// Pass 2: AST-based detection (server, command-flow, snapshot usage, tracing wiring).
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
			// http.Server.Serve, grpc.NewServer.
			if method == "ListenAndServe" || method == "ListenAndServeTLS" ||
				method == "Serve" {
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
				hasDispatcher = true
			}
			if method == "Dispatch" {
				hasDispatch = true
			}

			// Snapshot usage detection.
			if method == "WithSnapshotStore" || method == "WithSnapshotStrategy" ||
				method == "NewSnapshotStore" {
				hasSnapshotUsage = true
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
			// determine the ACTUAL backend. storage.NewSQLiteEventStore,
			// storage.NewSQLiteSnapshotStore, etc. are library-provided SQLite
			// stores — recognizing them prevents C036 cascades and produces an
			// accurate `doctor` profile.
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

			// ServerLocal signals: detect production server indicators so
			// we can classify embedded-dashboards (ListenAndServe without
			// TLS/Shutdown/health) as ServerLocal.
			if method == "ListenAndServeTLS" {
				hasTLS = true
			}
			if method == "NewListener" || method == "Listen" {
				if strings.Contains(SelectorPackage(sel), "tls") {
					hasTLS = true
				}
			}
			if method == "Shutdown" || method == "GracefulClose" {
				hasShutdown = true
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
				hasHealthRoute = true
			}

			return true
		})
	}

	// Resolve ServerLocal: HasServer without ANY production signals (TLS,
	// graceful Shutdown, or health endpoint) means this is a CLI tool with an
	// embedded dashboard, not a deployed service. Suppress server-only rules.
	if fp.HasServer && !hasTLS && !hasShutdown && !hasHealthRoute {
		fp.ServerLocal = true
	}

	// Resolve command-flow from dispatcher/dispatch signals.
	switch {
	case hasDispatch:
		fp.CommandFlow = CommandFlowCommands
	case hasDispatcher:
		fp.CommandFlow = CommandFlowSync
	default:
		fp.CommandFlow = CommandFlowReadOnly
	}

	// Resolve soft-delete and domain from the registry. These are weak,
	// project-wide heuristics and intentionally use the global registry rather
	// than a per-module partition (which would require splitting scanFile).
	fp.HasSoftDelete = detectSoftDeleteRegistry(registry)
	fp.Domain = detectDomainRegistry(registry)

	// Resolve tracing.
	if fp.Tracing == TracingUnknown {
		if hasOTelImport {
			fp.Tracing = TracingOn
		} else {
			fp.Tracing = TracingOff
		}
	}

	// Resolve snapshot.
	switch {
	case hasSnapshotUsage:
		fp.Snapshot = SnapshotOn
	case hasSnapshotImport:
		fp.Snapshot = SnapshotOn
	default:
		fp.Snapshot = SnapshotOff
	}

	return fp
}

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
	for i := 0; i < len(dir); i++ {
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
