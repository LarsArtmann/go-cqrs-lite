package adoption

import (
	"context"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// deletedModuleImport describes a module whose SOURCE no longer exists at HEAD
// but whose git tags still resolve through the module proxy, plus the removal
// record and the sanctioned replacement.
//
// "The tag exists" and "the module exists" are different facts: a consumer
// pinned to a deleted module builds fine today and only discovers the problem
// when it upgrades. That is exactly the failure this rule reports.
type deletedModuleImport struct {
	fragment    string // import-path fragment, go-cqrs-lite/<module> form
	removal     string // what was removed, when, and under which ADR
	adr         string // repo-relative ADR path carrying the migration table
	replacement string // the sanctioned path forward
}

// deletedModuleImports is the registry of modules deleted from this repo while
// their tags stay proxy-served (ADR-0152 forbids tag deletion). Each entry is
// checked against every consumer import; the LONGEST matching fragment wins,
// so a specific entry is never attributed to a deleted ancestor.
//
//nolint:gochecknoglobals // static lookup table for deleted-module detection
var deletedModuleImports = []deletedModuleImport{
	{
		fragment:    "go-cqrs-lite/transport/http",
		removal: "the transport/* modules were removed 2026-10-17 (ADR-0127, delete-at-zero)",
		adr:     "docs/adr/0127-deprecate-transport-modules.md",
		replacement: "Serve SSE with github.com/larsartmann/go-sse (or metaengine.ServeSSE " +
			"for read-model push), use the watermill/ bridge " +
			"(NewEventPublisher/NewCommandPublisher) for broker fanout, or cqrs-htmx " +
			"for server-rendered UIs",
	},
	{
		fragment:    "go-cqrs-lite/transport/grpc",
		removal:     "the transport/* modules were removed 2026-10-17 (ADR-0127, delete-at-zero)",
		adr:         "docs/adr/0127-deprecate-transport-modules.md",
		replacement: "Use the watermill/ bridge over any broker backend " + "(WithBackend/WithCommandBackend), or call grpc-go directly",
	},
	{
		fragment:    "go-cqrs-lite/idempotency/kvstore",
		removal:     "the idempotency/kvstore module was deleted 2026-10-17 at zero fleet consumers (its tags stay proxy-served)",
		adr:         "docs/adr/0167-idempotency-retirement-system-dedup-facade.md",
		replacement: "Dedup through system.Dedup() over a metaengine.DedupStore engine (ADR-0142/0158/0167)",
	},
	{
		fragment:    "go-cqrs-lite/idempotency",
		removal:     "the idempotency/v4 module was a re-export shim deleted 2026-08-15 (ADR-0128); only its v4.4.0 tag still resolves",
		adr:         "docs/adr/0128-extract-codec-and-remove-shim-modules.md",
		replacement: "Import github.com/larsartmann/go-idempotency directly (MVS already resolves it), or use system.Dedup() for engine-backed durability",
	},
	{
		fragment:    "go-cqrs-lite/codec",
		removal:     "the codec re-export shim was deleted 2026-08-15 (ADR-0128)",
		adr:         "docs/adr/0128-extract-codec-and-remove-shim-modules.md",
		replacement: "Import github.com/larsartmann/go-codec directly",
	},
	{
		fragment:    "go-cqrs-lite/retry",
		removal:     "the retry re-export shim was deleted 2026-08-15 (ADR-0128)",
		adr:         "docs/adr/0128-extract-codec-and-remove-shim-modules.md",
		replacement: "Import github.com/larsartmann/go-retry directly",
	},
	{
		fragment:    "go-cqrs-lite/flightrecorder",
		removal:     "the flightrecorder re-export shim was deleted 2026-08-15 (ADR-0128)",
		adr:         "docs/adr/0128-extract-codec-and-remove-shim-modules.md",
		replacement: "Import github.com/larsartmann/go-flightrecorder directly",
	},
}

// presentModuleFragments lists modules that still exist at HEAD even though a
// deleted ancestor fragment also matches their import path. Without this guard
// the deleted parent idempotency entry would fire on
// …/idempotency/sqlstore imports, which are deprecated but very much present.
//
//nolint:gochecknoglobals // static lookup table
var presentModuleFragments = []string{
	"go-cqrs-lite/idempotency/sqlstore",
}

// F030 detects imports of modules whose source was deleted while their tags
// stay proxy-served: transport/* (ADR-0127), idempotency/* (ADR-0128/0152),
// and the extracted shims removed in ADR-0128. Projects pinned to the old tags
// build today and get a concrete migration path here instead of a surprise at
// upgrade time.
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
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				out = append(out, deletedModuleFindings(ctx, sc.files)...)
			}

			return out, nil
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
		if gf.IsTest {
			continue
		}

		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}

			path := strings.Trim(imp.Path.Value, `"`)

			dep, ok := longestDeletedModuleMatch(path)
			if !ok {
				continue
			}

			out = append(out, singleWarningFinding(
				ctx,
				"F030",
				"Import of deleted module "+path+" — "+dep.removal,
				dep.replacement+". See "+dep.adr+" for the migration path.",
				ctx.Fset.Position(imp.Path.Pos()),
				finding.ConfidenceHigh,
			)...)
		}
	}

	return out
}

// longestDeletedModuleMatch returns the deleted-module entry whose fragment is
// the longest match for importPath, unless a still-present module matches more
// specifically. Longest-match keeps a specific entry from ever being shadowed
// by a deleted ancestor (idempotency vs idempotency/kvstore).
func longestDeletedModuleMatch(importPath string) (deletedModuleImport, bool) {
	presentLen := longestPresentFragmentLen(importPath)

	var best deletedModuleImport

	found := false

	for _, dep := range deletedModuleImports {
		if !fragmentMatchesImport(importPath, dep.fragment) {
			continue
		}

		if len(dep.fragment) <= presentLen {
			continue // a still-present submodule owns this import
		}

		if !found || len(dep.fragment) > len(best.fragment) {
			best, found = dep, true
		}
	}

	return best, found
}

func longestPresentFragmentLen(importPath string) int {
	longest := 0

	for _, fragment := range presentModuleFragments {
		if fragmentMatchesImport(importPath, fragment) && len(fragment) > longest {
			longest = len(fragment)
		}
	}

	return longest
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
