package analyzer

import (
	"maps"
	"slices"
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
	// Iterate in sorted order (packages and imports): pkg.Imports is a Go map,
	// and first-wins store resolution below is only deterministic when the
	// visit order is (T20-3).
	sortedPkgs := slices.Clone(pkgs)
	slices.SortFunc(sortedPkgs, func(a, b *packages.Package) int {
		return strings.Compare(a.PkgPath, b.PkgPath)
	})
	for _, pkg := range sortedPkgs {
		if len(pkg.Errors) > 0 {
			continue
		}
		for _, path := range slices.Sorted(maps.Keys(pkg.Imports)) {
			if pkg.Imports[path] == nil {
				continue
			}
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
	// Stack presets are explicit deployment choices; first-wins (the caller
	// iterates imports in sorted order) so a package importing two presets
	// resolves deterministically instead of overwriting in map order (T20-3).
	// StoreNone is unreachable mid-pass (assigned after all imports), kept in
	// the guard for symmetry with the metaengine branch below.
	if fp.Store == StoreUnknown || fp.Store == StoreNone {
		switch {
		case strings.Contains(path, "go-cqrs-lite/stack/sqlite"):
			fp.Store = StoreSQLite
		case strings.Contains(path, "go-cqrs-lite/stack/postgres"):
			fp.Store = StorePostgres
		case strings.Contains(path, "go-cqrs-lite/stack/mysql"):
			fp.Store = StoreMySQL
		case strings.Contains(path, "go-cqrs-lite/stack/pebble"):
			fp.Store = StorePebble
		case strings.Contains(path, "go-cqrs-lite/stack/memory"):
			fp.Store = StoreMemory
		case strings.Contains(path, "go-cqrs-lite/stack/turso"):
			fp.Store = StoreTurso
		case strings.Contains(path, "go-cqrs-lite/stack/duckdb"):
			fp.Store = StoreDuckDB
		case strings.Contains(path, "go-cqrs-lite/stack/bbolt"):
			fp.Store = StoreBolt
		case strings.Contains(path, "go-cqrs-lite/storage/"):
			fp.Store = StoreCustom
		}
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

	// HasTransport covers every sanctioned external-delivery path: the
	// watermill/ bridge (broker transports), go-sse (SSE delivery), cqrs-htmx,
	// and the deprecated transport/* modules (kept so legacy projects do not
	// get coached while migrating).
	if strings.Contains(path, "go-cqrs-lite/transport") ||
		strings.Contains(path, "go-cqrs-lite/watermill") ||
		strings.Contains(path, "larsartmann/go-sse") ||
		strings.Contains(path, "cqrs-htmx") {
		fp.HasTransport = true
	}

	// Metaengine detection (P3: engine imports also serve as store hints).
	if strings.Contains(path, "go-cqrs-lite/metaengine") {
		fp.HasMetaengine = true

		engine := metaengineEngineFromImport(path)
		if engine != "" && !slices.Contains(fp.MetaengineEngines, engine) {
			fp.MetaengineEngines = append(fp.MetaengineEngines, engine)
		}

		// Engine subpackages imply a store backend.
		if fp.Store == StoreUnknown || fp.Store == StoreNone {
			switch engine {
			case "sqlite":
				fp.Store = StoreSQLite
			case "pebble":
				fp.Store = StorePebble
			case "duckdb":
				fp.Store = StoreDuckDB
			case "postgres":
				fp.Store = StorePostgres
			case "mysql":
				fp.Store = StoreMySQL
			case "turso":
				fp.Store = StoreTurso
			case "bbolt":
				fp.Store = StoreBolt
			case "badger":
				fp.Store = StoreBadger
			case "dgraph":
				fp.Store = StoreDgraph
			case "iroh":
				fp.Store = StoreIroh
			}
		}
	}
}

// metaengineEngineFromImport maps an import path to a short engine name.
// Returns "" for the core metaengine module (no specific engine) and for
// non-engine subpackages (projectionadapter, keycodec, …). The full
// shipped-engine list lives in metaengine/*engine — every engine module
// MUST appear here and in the engine→StoreKind switch above (T20-1);
// TestMetaengineEngineFromImport_CoversShippedEngines pins both.
func metaengineEngineFromImport(path string) string {
	switch {
	case strings.Contains(path, "metaengine/pebbleengine"):
		return "pebble"
	case strings.Contains(path, "metaengine/duckdbengine"):
		return "duckdb"
	case strings.Contains(path, "metaengine/pgengine"):
		return "postgres"
	case strings.Contains(path, "metaengine/sqliteengine"):
		return "sqlite"
	case strings.Contains(path, "metaengine/mysqlengine"):
		return "mysql"
	case strings.Contains(path, "metaengine/badgerengine"):
		return "badger"
	case strings.Contains(path, "metaengine/dgraphengine"):
		return "dgraph"
	case strings.Contains(path, "metaengine/tursoengine"):
		return "turso"
	case strings.Contains(path, "metaengine/bboltengine"):
		return "bbolt"
	case strings.Contains(path, "metaengine/irohengine"):
		return "iroh"
	default:
		return ""
	}
}
