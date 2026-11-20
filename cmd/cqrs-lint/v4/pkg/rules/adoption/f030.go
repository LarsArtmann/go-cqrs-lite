package adoption

import (
	"context"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// modulePresence records whether a module's SOURCE exists at HEAD.
//
// "The tag exists" and "the module exists" are different facts: ADR-0152
// forbids deleting tags, so a deleted module stays proxy-served forever.
// Presence is therefore tested against go.mod at HEAD, never against tags.
type modulePresence int

const (
	// modulePresent is a module whose go.mod still exists at HEAD. Deprecated
	// modules are still present — V007 owns their v5 removal.
	modulePresent modulePresence = iota

	// moduleDeleted is a module whose source was deleted while its tags stay
	// proxy-served. Importing one builds today and only breaks at upgrade.
	moduleDeleted
)

// modulePresenceEntry is one row of the module-presence registry: the
// import-path fragment, whether the module exists today, and the record a
// finding needs to explain the migration.
type modulePresenceEntry struct {
	fragment    string         // import-path fragment, go-cqrs-lite/<module> form
	state       modulePresence // present at HEAD, or deleted with tags surviving
	removal     string         // what was removed, when, and under which ADR
	adr         string         // repo-relative ADR path carrying the migration table
	replacement string         // the sanctioned path forward
}

// modulePresenceRegistry is the single table behind F030. It holds BOTH states
// on purpose: a deleted ancestor fragment (go-cqrs-lite/idempotency) also
// matches the imports of a still-present submodule (idempotency/sqlstore), and
// only a shared longest-match resolver can rank them against each other.
// Splitting present rows into a side list is how that ranking drifts.
//
//nolint:gochecknoglobals // static lookup table for module-presence detection
var modulePresenceRegistry = []modulePresenceEntry{
	{
		fragment: "go-cqrs-lite/transport/http",
		state:    moduleDeleted,
		removal:  "the transport/* modules were removed 2026-10-17 (ADR-0127, delete-at-zero)",
		adr:      "docs/adr/0127-deprecate-transport-modules.md",
		replacement: "Serve SSE with github.com/larsartmann/go-sse (or metaengine.ServeSSE " +
			"for read-model push), use the watermill/ bridge " +
			"(NewEventPublisher/NewCommandPublisher) for broker fanout, or cqrs-htmx " +
			"for server-rendered UIs",
	},
	{
		fragment:    "go-cqrs-lite/transport/grpc",
		state:       moduleDeleted,
		removal:     "the transport/* modules were removed 2026-10-17 (ADR-0127, delete-at-zero)",
		adr:         "docs/adr/0127-deprecate-transport-modules.md",
		replacement: "Use the watermill/ bridge over any broker backend " + "(WithBackend/WithCommandBackend), or call grpc-go directly",
	},
	{
		fragment:    "go-cqrs-lite/idempotency/kvstore",
		state:       moduleDeleted,
		removal:     "the idempotency/kvstore module was deleted 2026-10-17 at zero fleet consumers (its tags stay proxy-served)",
		adr:         "docs/adr/0167-idempotency-retirement-system-dedup-facade.md",
		replacement: "Dedup through system.Dedup() over a metaengine.DedupStore engine (ADR-0142/0158/0167)",
	},
	{
		fragment:    "go-cqrs-lite/idempotency",
		state:       moduleDeleted,
		removal:     "the idempotency/v4 module was a re-export shim deleted 2026-08-15 (ADR-0128); only its v4.4.0 tag still resolves",
		adr:         "docs/adr/0128-extract-codec-and-remove-shim-modules.md",
		replacement: "Import github.com/larsartmann/go-idempotency directly (MVS already resolves it), or use system.Dedup() for engine-backed durability",
	},
	{
		fragment:    "go-cqrs-lite/codec",
		state:       moduleDeleted,
		removal:     "the codec re-export shim was deleted 2026-08-15 (ADR-0128)",
		adr:         "docs/adr/0128-extract-codec-and-remove-shim-modules.md",
		replacement: "Import github.com/larsartmann/go-codec directly",
	},
	{
		fragment:    "go-cqrs-lite/retry",
		state:       moduleDeleted,
		removal:     "the retry re-export shim was deleted 2026-08-15 (ADR-0128)",
		adr:         "docs/adr/0128-extract-codec-and-remove-shim-modules.md",
		replacement: "Import github.com/larsartmann/go-retry directly",
	},
	{
		fragment:    "go-cqrs-lite/flightrecorder",
		state:       moduleDeleted,
		removal:     "the flightrecorder re-export shim was deleted 2026-08-15 (ADR-0128)",
		adr:         "docs/adr/0128-extract-codec-and-remove-shim-modules.md",
		replacement: "Import github.com/larsartmann/go-flightrecorder directly",
	},
	{
		// Present, deprecated: listed so the deleted idempotency ancestor
		// fragment never claims this submodule's imports.
		fragment:    "go-cqrs-lite/idempotency/sqlstore",
		state:       modulePresent,
		removal:     "the idempotency/sqlstore module is deprecated (ADR-0158) and removed at v5.0.0",
		adr:         "docs/adr/0158-idempotency-retirement-engine-dedupstore.md",
		replacement: "Dedup through system.Dedup() over a metaengine.DedupStore engine (ADR-0142/0158/0167), or keep github.com/larsartmann/go-idempotency for in-process keys",
	},
}

// F030 detects imports of modules whose source was deleted while their tags
// stay proxy-served: transport/* (ADR-0127), idempotency/* (ADR-0128/0152),
// and the extracted shims removed in ADR-0128. Projects pinned to the old tags
// build today and get a concrete migration path here instead of a surprise at
// upgrade time.
//
// Test files are deliberately INCLUDED: a _test.go import pins the same
// frozen tag in go.mod/go.sum and blocks upgrades exactly as hard as a
// production import. F030 is purely import-path driven — it consults no
// feature profile — so it walks every parsed file instead of the
// profile-scoped coachingScopes (which exist to stop per-module signal
// leakage, a concern F030 does not share).
//
// This is the companion to F013: F013 coaches projects WITHOUT a delivery
// module toward the sanctioned paths, F030 coaches projects OFF the deleted
// ones. Deprecated-but-present modules (idempotency/sqlstore, scheduling/
// sqlstore, the stack presets) belong to V007, which owns v5 removals.
//
//nolint:ireturn // factory returns public interface
func NewF030Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F030-deleted-module-import",
		func(_ context.Context) ([]finding.Finding, error) {
			return deletedModuleFindings(ctx, ctx.GoFiles), nil
		},
	)
}

// deletedModuleFindings reports one finding per deleted-module import in files.
func deletedModuleFindings(
	ctx *analyzer.AnalysisContext,
	files []*analyzer.GoFile,
) []finding.Finding {
	var out []finding.Finding

	for _, gf := range files {
		if gf.AST == nil {
			continue
		}

		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}

			path := strings.Trim(imp.Path.Value, `"`)

			entry, ok := resolveModulePresence(path)
			if !ok || entry.state != moduleDeleted {
				continue
			}

			out = append(out, singleWarningFinding(
				ctx,
				"F030",
				"Import of deleted module "+path+" — "+entry.removal,
				entry.replacement+". See "+entry.adr+" for the migration path.",
				ctx.Fset.Position(imp.Path.Pos()),
				finding.ConfidenceHigh,
			)...)
		}
	}

	return out
}

// resolveModulePresence returns the registry entry that most specifically
// matches importPath, whatever its state. Longest-fragment-wins is the whole
// point of one table: a specific entry can never be shadowed by an ancestor
// (idempotency/sqlstore must not resolve to the deleted idempotency shim), and
// that ranking is only expressible when both states share one resolver.
func resolveModulePresence(importPath string) (modulePresenceEntry, bool) {
	var best modulePresenceEntry

	found := false

	for _, entry := range modulePresenceRegistry {
		if !fragmentMatchesImport(importPath, entry.fragment) {
			continue
		}

		if !found || len(entry.fragment) > len(best.fragment) {
			best, found = entry, true
		}
	}

	return best, found
}

// fragmentMatchesImport reports whether fragment appears in importPath as a
// whole path segment sequence: the character before it must be '/' (or absent)
// and the character after it must be '/' (or absent).
func fragmentMatchesImport(importPath, fragment string) bool {
	idx := strings.Index(importPath, fragment)
	if idx < 0 {
		return false
	}

	if idx > 0 && importPath[idx-1] != '/' {
		return false
	}

	end := idx + len(fragment)

	return end == len(importPath) || importPath[end] == '/'
}
