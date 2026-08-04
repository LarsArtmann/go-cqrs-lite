package analyzer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCatalogNoDuplicateKeys(t *testing.T) {
	t.Parallel()

	seen := make(map[ModuleKey]bool)
	for _, e := range DefaultCatalog.All() {
		if seen[e.Key] {
			t.Fatalf("duplicate catalog key: %s", e.Key)
		}
		seen[e.Key] = true
	}
}

// hintMatchesAtBoundary checks whether hint would match longerHint at a path
// boundary. E.g. "go-cqrs-lite/id" matches "go-cqrs-lite/id/v4" (boundary: /)
// but NOT "go-cqrs-lite/idempotency" (next char: 'e', not '/'). This mirrors
// the path-boundary matching used by DetectUsedModules.
func hintMatchesAtBoundary(hint, candidate string) bool {
	idx := strings.Index(candidate, hint)
	if idx < 0 {
		return false
	}
	end := idx + len(hint)
	if end >= len(candidate) {
		return true // hint matches at end of string
	}
	// Path boundary: next char must be '/' for a clean module-path match.
	return candidate[end] == '/'
}

func TestCatalogImportHintsUnique(t *testing.T) {
	t.Parallel()

	// Build a map of every hint to the module that owns it. A hint that is a
	// path-boundary prefix of another hint causes ambiguous matches at the
	// module level (e.g. "go-cqrs-lite/id" vs "go-cqrs-lite/id/v4/eventtest").
	hintToKey := make(map[string]ModuleKey)
	for _, e := range DefaultCatalog.All() {
		for _, h := range e.ImportHints {
			if existing, ok := hintToKey[h]; ok && existing != e.Key {
				t.Fatalf("import hint %q shared by %s and %s", h, existing, e.Key)
			}
			hintToKey[h] = e.Key
		}
	}

	// Check no hint is a path-boundary prefix of another hint. This catches
	// real detection ambiguity: if hint A matches at a path boundary within
	// hint B, then an import of module B would also be detected as module A.
	for hintA, keyA := range hintToKey {
		for hintB, keyB := range hintToKey {
			if keyA == keyB {
				continue
			}
			if hintA != hintB && hintMatchesAtBoundary(hintA, hintB) {
				t.Errorf(
					"import hint %q (module %s) matches at path boundary within %q (module %s) — ambiguous detection",
					hintA,
					keyA,
					hintB,
					keyB,
				)
			}
		}
	}
}

func TestCatalogHasExpectedCounts(t *testing.T) {
	t.Parallel()

	all := DefaultCatalog.All()
	scored := DefaultCatalog.Scored()
	core := DefaultCatalog.Core()

	if len(core) != 6 {
		t.Fatalf("expected 6 core entries, got %d: %+v", len(core), core)
	}
	if len(scored) != 32 {
		t.Fatalf("expected 32 scored entries, got %d", len(scored))
	}
	if len(all) != 38 {
		t.Fatalf("expected 38 total entries, got %d", len(all))
	}
}

func TestCatalogEveryEntryHasRequiredFields(t *testing.T) {
	t.Parallel()

	for _, e := range DefaultCatalog.All() {
		if e.Key == "" {
			t.Error("entry with empty Key")
		}
		if e.DisplayName == "" {
			t.Errorf("module %s: empty DisplayName", e.Key)
		}
		if e.Category == "" {
			t.Errorf("module %s: empty Category", e.Key)
		}
		if len(e.ImportHints) == 0 {
			t.Errorf("module %s: no ImportHints", e.Key)
		}
		if !e.Core && e.Description == "" {
			t.Errorf("module %s: empty Description", e.Key)
		}
		if !e.Core && e.Suggestion == "" {
			t.Errorf("module %s: empty Suggestion", e.Key)
		}
	}
}

func TestCatalogRelevantForProfile(t *testing.T) {
	t.Parallel()

	// A local-CLI profile (no server) should exclude transport and server-infra modules.
	localCLI := FeatureProfile{
		HasServer:   false,
		ServerLocal: false,
	}
	relevant := DefaultCatalog.RelevantFor(localCLI, PresetLocalCLI)
	relevantKeys := make(map[ModuleKey]bool, len(relevant))
	for _, e := range relevant {
		relevantKeys[e.Key] = true
	}

	// Transport and server-infra modules should NOT be relevant for local-cli.
	serverOnlyKeys := []ModuleKey{
		"transport/http", "transport/grpc",
		"prometheus", "watermill",
		"stack/postgres", "stack/mysql", "stack/turso",
	}
	for _, k := range serverOnlyKeys {
		if relevantKeys[k] {
			t.Errorf("module %s should NOT be relevant for local-cli profile", k)
		}
	}

	// Universally relevant modules SHOULD be present.
	universalKeys := []ModuleKey{
		"otel", "encryption", "signing",
		"scheduling", "snapshot", "schema",
		"kv", "projectionhost", "catalog", "codec",
	}
	for _, k := range universalKeys {
		if !relevantKeys[k] {
			t.Errorf("module %s should be relevant for local-cli profile", k)
		}
	}

	// A production profile (server, not local) should include transport.
	production := FeatureProfile{
		HasServer:   true,
		ServerLocal: false,
	}
	prodRelevant := DefaultCatalog.RelevantFor(production, PresetProduction)
	prodKeys := make(map[ModuleKey]bool, len(prodRelevant))
	for _, e := range prodRelevant {
		prodKeys[e.Key] = true
	}
	for _, k := range serverOnlyKeys {
		if !prodKeys[k] {
			t.Errorf("module %s should be relevant for production profile", k)
		}
	}
}

func TestCatalogGet(t *testing.T) {
	t.Parallel()

	e, ok := DefaultCatalog.Get("scheduling")
	if !ok {
		t.Fatal("expected to find 'scheduling' in catalog")
	}
	if e.DisplayName != "Scheduling" {
		t.Fatalf("expected DisplayName 'Scheduling', got %q", e.DisplayName)
	}

	_, ok = DefaultCatalog.Get("nonexistent")
	if ok {
		t.Fatal("expected Get to return false for nonexistent key")
	}
}

func TestCatalogEveryGoWorkModuleCovered(t *testing.T) {
	t.Parallel()

	// Prevents catalog drift: every library module in go.work must be in the
	// catalog or explicitly excluded. Mirrors TestEveryGoModDirIsInModulesList.

	goWorkPath := findGoWork(t)
	data, err := os.ReadFile(goWorkPath)
	if err != nil {
		t.Skipf("cannot read go.work: %v (skipping drift check)", err)
	}

	goWorkModules := extractUseModules(t, string(data))

	catalogHints := make(map[string]bool)
	for _, e := range DefaultCatalog.All() {
		for _, h := range e.ImportHints {
			catalogHints[h] = true
		}
	}

	// Excluded: tooling, test helpers, sub-packages, low-level internals, root.
	excludedModules := map[string]string{
		".":                            "library root (consumers import sub-modules, not the root)",
		"benchkit":                     "benchmarking utility (niche, not a typical adoption decision)",
		"cmd/api-stability":            "tooling",
		"cmd/cqrs-bench":               "tooling",
		"cmd/cqrs-gen":                 "tooling",
		"cmd/cqrs-lint":                "tooling",
		"cmd/doc-check":                "tooling",
		"dispatcher":                   "internal infrastructure (generic dispatcher used by command/query)",
		"event/v4/eventtest":           "test helper sub-package",
		"example/getting-started":      "example project",
		"example/readme-quickstart":    "example project",
		"example/taskmanager":          "example project",
		"idempotency/kvstore":          "sub-package (covered by idempotency)",
		"idempotency/sqlstore":         "sub-package (covered by idempotency)",
		"integration":                  "cross-module integration tests",
		"metaengine/irohengine":      "sub-engine (covered by metaengine)",
		"metaengine/adttest":         "test helper sub-package",
		"metaengine/duckdbengine":      "sub-engine (covered by metaengine)",
		"metaengine/pebbleengine":      "sub-engine (covered by metaengine)",
		"metaengine/pgengine":          "sub-engine (covered by metaengine)",
		"metaengine/projectionadapter": "sub-package (covered by metaengine)",
		"projection":                   "interface-only module (consumers use projectionhost)",
		"scheduling/sqlstore":          "sub-package (covered by scheduling)",
		"stack":                        "root stack types (consumers import stack/<backend> presets)",
		"stack/bench":                  "benchmarking utility",
		"storage/memory":               "low-level in-memory store (covered by stack/memory)",
		"storage/pebble":               "low-level Pebble store (covered by stack/pebble)",
		"storage/turso":                "low-level Turso connector (covered by stack/turso)",
		"system":                       "system-level utilities (not a domain module)",
		"testutil":                     "test utility package",
	}

	// Also exclude external workspace entries (../go-* paths).
	for _, mod := range goWorkModules {
		if strings.HasPrefix(mod, "../") {
			continue // external repo (go-idempotency, go-retry)
		}

		// Check if the module is covered by a catalog import hint.
		fullPath := "go-cqrs-lite/" + mod
		covered := false
		for _, e := range DefaultCatalog.All() {
			for _, h := range e.ImportHints {
				if h == fullPath {
					covered = true
					break
				}
			}
			if covered {
				break
			}
		}

		if covered {
			continue
		}

		// Not in catalog — must be in the exclusion list.
		if _, excluded := excludedModules[mod]; !excluded {
			t.Errorf("go.work module %q is neither in the catalog nor in the exclusion list.\n"+
				"Either add it to DefaultCatalog or add it to excludedModules with a reason.",
				mod)
		}
	}
}

// findGoWork locates go.work by walking up from the test source file.
func findGoWork(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}

	dir := filepath.Dir(filename)
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "go.work")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Skip("go.work not found (skipping drift check)")
	return ""
}

// extractUseModules parses the "use (...)" block from go.work and returns
// the list of module paths (relative, without the leading "./").
func extractUseModules(t *testing.T, content string) []string {
	t.Helper()

	lines := strings.Split(content, "\n")
	var modules []string
	inUseBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "use (") {
			inUseBlock = true
			continue
		}
		if strings.HasPrefix(trimmed, "use ") && !strings.HasSuffix(trimmed, "(") {
			// Single-line use directive: use ./foo
			mod := strings.TrimSpace(strings.TrimPrefix(trimmed, "use"))
			mod = strings.TrimPrefix(mod, "./")
			if mod != "" {
				modules = append(modules, mod)
			}
			continue
		}
		if inUseBlock {
			if trimmed == ")" {
				inUseBlock = false
				continue
			}
			mod := strings.TrimSpace(strings.TrimPrefix(trimmed, "./"))
			mod = strings.TrimSpace(mod)
			if mod != "" && !strings.HasPrefix(mod, "//") {
				modules = append(modules, mod)
			}
		}
	}

	return modules
}
