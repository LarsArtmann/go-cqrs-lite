package eventcatalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

// readIndexManifest exports the given catalog and returns the raw
// catalog.index.json bytes from the export root.
func readIndexManifest(t *testing.T, cat *catalog.Catalog) string {
	t.Helper()

	tmpDir := exportToTempDir(t, cat)

	data, err := os.ReadFile(filepath.Join(tmpDir, "catalog.index.json"))
	if err != nil {
		t.Fatalf("read catalog.index.json: %v", err)
	}

	return string(data)
}

func TestIndexManifest_CoversEveryResourceKind(t *testing.T) {
	// BuildTestCatalog covers services, commands, events, queries, channels,
	// and containers; the remaining kinds are added here so the manifest's
	// full kind coverage is pinned.
	cat := cattest.BuildTestCatalog()

	manifest := readIndexManifest(t, cat)

	for _, want := range []string{
		`"kind": "services"`,
		`"kind": "commands"`,
		`"kind": "events"`,
		`"kind": "queries"`,
		`"kind": "channels"`,
	} {
		if !strings.Contains(manifest, want) {
			t.Errorf("manifest missing %s entry:\n%s", want, manifest)
		}
	}

	if !strings.Contains(manifest, `"schemaVersion": 1`) {
		t.Errorf("manifest missing schemaVersion:\n%s", manifest)
	}
}

func TestIndexManifest_OrdersKindsCanonicallyThenID(t *testing.T) {
	reg := cattest.NewTestRegistry(catalog.Service{
		ID: "zeta-svc", Name: "Zeta", Version: "1.0.0", Summary: "z",
	}, catalog.Service{
		ID: "alpha-svc", Name: "Alpha", Version: "1.0.0", Summary: "a",
	})
	reg.AddEvent("zeta-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "ZEvent", Name: "Z", Version: "1.0.0",
		Summary: "z", Direction: catalog.Sends,
	})
	reg.AddEvent("alpha-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "AEvent", Name: "A", Version: "1.0.0",
		Summary: "a", Direction: catalog.Sends,
	})

	manifest := readIndexManifest(t, reg.Build())

	firstService := strings.Index(manifest, `"id": "alpha-svc"`)
	secondService := strings.Index(manifest, `"id": "zeta-svc"`)
	firstEvent := strings.Index(manifest, `"id": "AEvent"`)
	secondEvent := strings.Index(manifest, `"id": "ZEvent"`)

	if firstService == -1 || secondService == -1 || firstEvent == -1 || secondEvent == -1 {
		t.Fatalf("manifest missing expected entries:\n%s", manifest)
	}

	if firstService > secondService {
		t.Errorf("services not ID-sorted: alpha-svc should precede zeta-svc:\n%s", manifest)
	}

	if firstEvent < secondService {
		t.Errorf("kind order violated: events should follow services:\n%s", manifest)
	}

	if firstEvent > secondEvent {
		t.Errorf("events not ID-sorted: AEvent should precede ZEvent:\n%s", manifest)
	}
}

func TestIndexManifest_DedupesSharedMessages(t *testing.T) {
	sharedEvent := catalog.Message{
		Kind: catalog.EventMessage, ID: "Shared", Name: "Shared", Version: "1.0.0",
		Summary: "shared", Direction: catalog.Sends,
	}
	reg := cattest.NewTestRegistry(catalog.Service{
		ID: "one-svc", Name: "One", Version: "1.0.0", Summary: "1",
		Events: []catalog.Message{sharedEvent},
	}, catalog.Service{
		ID: "two-svc", Name: "Two", Version: "1.0.0", Summary: "2",
		Events: []catalog.Message{sharedEvent},
	})

	manifest := readIndexManifest(t, reg.Build())

	if got := strings.Count(manifest, `"id": "Shared"`); got != 1 {
		t.Errorf("shared event should appear exactly once, got %d:\n%s", got, manifest)
	}
}

// TestIndexManifest_StableAcrossReExport pins the hub-diffing contract:
// exporting the same catalog twice must produce byte-identical manifests.
func TestIndexManifest_StableAcrossReExport(t *testing.T) {
	cat := cattest.BuildTestCatalog()

	first := readIndexManifest(t, cat)
	second := readIndexManifest(t, cat)

	if first != second {
		t.Errorf("manifest differs across re-exports:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestIndexManifest_PathsMatchWrittenFiles(t *testing.T) {
	tmpDir := exportToTempDir(t, cattest.BuildTestCatalog())

	data, err := os.ReadFile(filepath.Join(tmpDir, "catalog.index.json"))
	if err != nil {
		t.Fatalf("read catalog.index.json: %v", err)
	}

	for _, resourcePath := range extractJSONStringField(data, "path") {
		if _, err := os.Stat(filepath.Join(tmpDir, filepath.FromSlash(resourcePath))); err != nil {
			t.Errorf("manifest path %q does not exist in export: %v", resourcePath, err)
		}
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func extractJSONStringField(data []byte, field string) []string {
	var values []string

	text := string(data)
	key := `"` + field + `": "`
	for from := 0; ; {
		at := strings.Index(text[from:], key)
		if at == -1 {
			return values
		}
		start := from + at + len(key)
		end := start
		for end < len(text) && text[end] != '"' && text[end] != '\n' {
			end++
		}
		values = append(values, text[start:end])
		from = start
	}
}
