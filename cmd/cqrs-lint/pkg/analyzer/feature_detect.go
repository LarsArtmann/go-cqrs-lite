package analyzer

import (
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
func DetectFeaturesPerModule(
	ctx *AnalysisContext,
	packagesByModule map[string][]*packages.Package,
) map[string]FeatureProfile {
	filesByModule := groupGoFilesByModule(ctx)

	dirSet := map[string]struct{}{}
	for dir := range packagesByModule {
		dirSet[dir] = struct{}{}
	}
	for dir := range filesByModule {
		dirSet[dir] = struct{}{}
	}
	dirs := sortModuleDirs(dirSet)

	profiles := make(map[string]FeatureProfile, len(dirs))
	for _, dir := range dirs {
		profiles[dir] = detectFeatureSignals(
			packagesByModule[dir],
			filesByModule[dir],
			ctx.Registry,
		)
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
	hasOTelImport := false
	hasSnapshotImport := false
	hasHTTPFramework := false

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
			detectImports(path, &fp, &hasSQLiteImport, &hasOTelImport, &hasSnapshotImport)
		}
	}

	// Pass 1b: AST import-based detection. Supplements Pass 1 which uses
	// pkg.Imports (populated by go/packages). In test contexts where
	// pkg.Imports is empty, AST import declarations are the only source.
	for _, gf := range gofiles {
		if gf.IsTest {
			continue
		}
		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}
			path := strings.Trim(imp.Path.Value, `"`)
			detectImports(path, &fp, &hasSQLiteImport, &hasOTelImport, &hasSnapshotImport)
			if isHTTPFrameworkImport(path) {
				hasHTTPFramework = true
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
	sig := scanASTCalls(gofiles, &fp, hasOTelImport, hasHTTPFramework)

	// HTTP framework imports (Gin/Echo/Fiber/Chi) are strong server signals.
	if hasHTTPFramework {
		fp.HasServer = true
	}

	// Resolve ServerLocal: HasServer without ANY production signals (TLS,
	// graceful Shutdown, or health endpoint) means this is a CLI tool with an
	// embedded dashboard, not a deployed service. Suppress server-only rules.
	if fp.HasServer && !sig.hasTLS && !sig.hasShutdown && !sig.hasHealthRoute {
		fp.ServerLocal = true
	}

	// Resolve command-flow from dispatcher/dispatch signals.
	switch {
	case sig.hasDispatch:
		fp.CommandFlow = CommandFlowCommands
	case sig.hasDispatcher:
		fp.CommandFlow = CommandFlowSync
	default:
		fp.CommandFlow = CommandFlowReadOnly
	}

	// Resolve soft-delete and domain from the registry.
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
	case sig.hasSnapshotUsage:
		fp.Snapshot = SnapshotOn
	case hasSnapshotImport:
		fp.Snapshot = SnapshotOn
	default:
		fp.Snapshot = SnapshotOff
	}

	return fp
}

// detectImports applies import-path-based feature signals from a single import
// path string. Shared by Pass 1 (packages.Imports) and Pass 1b (AST imports).
func detectImports(
	path string,
	fp *FeatureProfile,
	hasSQLiteImport, hasOTelImport, hasSnapshotImport *bool,
) {
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
	} else if strings.Contains(path, "go-cqrs-lite/stack/duckdb") {
		fp.Store = StoreDuckDB
	} else if strings.Contains(path, "go-cqrs-lite/stack/bbolt") {
		fp.Store = StoreBolt
	} else if strings.Contains(path, "go-cqrs-lite/storage/") &&
		fp.Store == StoreUnknown {
		fp.Store = StoreCustom
	}

	if strings.Contains(path, "mattn/go-sqlite3") ||
		strings.Contains(path, "modernc.org/sqlite") {
		*hasSQLiteImport = true
	}

	if strings.Contains(path, "go.opentelemetry.io") ||
		strings.Contains(path, "go-cqrs-lite/otel") {
		*hasOTelImport = true
	}

	if strings.Contains(path, "go-cqrs-lite/snapshot") {
		*hasSnapshotImport = true
	}

	if strings.Contains(path, "go-cqrs-lite/watermill") {
		fp.HasAsyncBus = true
	}

	if strings.Contains(path, "go-cqrs-lite/transport") ||
		strings.Contains(path, "cqrs-htmx") {
		fp.HasTransport = true
	}
}
