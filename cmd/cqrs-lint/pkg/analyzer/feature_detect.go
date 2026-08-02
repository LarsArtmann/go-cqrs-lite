package analyzer

import (
	"go/ast"
	"strings"
)

// DetectFeatures scans the analyzed project and returns a FeatureProfile
// describing which go-cqrs-lite features the consumer uses. This replaces the
// per-detector heuristics (isLocalOnlyProject, hasTombstoneLikeEvents,
// hasDispatch) with one centralized declaration that all detectors consult.
func DetectFeatures(ctx *AnalysisContext) FeatureProfile {
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
	// Skip packages with errors — their import metadata may be unreliable.
	for _, pkg := range ctx.Packages {
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
	for _, gf := range ctx.GoFiles {
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

			// Server detection: http.ListenAndServe, http.Server, grpc.NewServer.
			if method == "ListenAndServe" || method == "Serve" {
				fp.HasServer = true
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
			if method == "ListenAndServeTLS" || method == "NewListener" {
				hasTLS = true
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

	// Resolve soft-delete from event registry.
	fp.HasSoftDelete = detectSoftDelete(ctx)

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

	// Resolve domain from event/command type names.
	fp.Domain = detectDomain(ctx)

	return fp
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

// detectDomain scans event and command type names for domain-specific
// keywords. Returns DomainFinancial when financial keywords are found,
// DomainUnknown otherwise.
func detectDomain(ctx *AnalysisContext) DomainKind {
	check := func(name string) bool {
		lower := strings.ToLower(name)
		for _, kw := range financialKeywords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
		return false
	}

	for eventType := range ctx.Registry.EventTypesEmitted {
		if check(eventType) {
			return DomainFinancial
		}
	}

	for cmdType := range ctx.Registry.CommandTypesRegistered {
		if check(cmdType) {
			return DomainFinancial
		}
	}

	return DomainUnknown
}

// detectSoftDelete returns true if any emitted event type name contains words
// associated with soft-delete (Deleted, Removed, Archived, Tombstoned).
func detectSoftDelete(ctx *AnalysisContext) bool {
	for eventType := range ctx.Registry.EventTypesEmitted {
		lower := strings.ToLower(eventType)
		for _, keyword := range []string{"deleted", "removed", "archived", "tombstoned"} {
			if strings.Contains(lower, keyword) {
				return true
			}
		}
	}

	return false
}
