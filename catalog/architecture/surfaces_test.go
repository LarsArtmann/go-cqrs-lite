package architecture_test

import (
	"bytes"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

// searchIndexBlock extracts the JSON payload of the embedded search index
// (compact json.Marshal output on a single line).
var searchIndexBlock = regexp.MustCompile(
	`<script type="application/json" id="search-index">(\[.*\])</script>`,
)

// TestSearchIndexKeyContractPinned pins the published key set of the search
// index. assets/reference.js reads these exact one-letter keys; adding,
// renaming or removing one is a breaking change for every generated file
// already in the wild and must be a conscious decision.
func TestSearchIndexKeyContractPinned(t *testing.T) {
	out := renderFixture(t)

	match := searchIndexBlock.FindStringSubmatch(out)
	if match == nil {
		t.Fatal("search index block not found in the rendered document")
	}

	var entries []map[string]any
	if err := json.Unmarshal([]byte(match[1]), &entries); err != nil {
		t.Fatalf("search index is not valid JSON: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("search index carries no entries")
	}

	// Sorted so the comparison against the sorted entry keys is order-safe.
	allowed := []string{"a", "c", "d", "g", "h", "k", "s", "t"}

	for i, entry := range entries {
		keys := make([]string, 0, len(entry))
		for key := range entry {
			keys = append(keys, key)
		}

		slices.Sort(keys)

		if !slices.Equal(keys, allowed) {
			t.Errorf("entry %d carries keys %v; contract is %v", i, keys, allowed)
		}
	}
}

// TestExportToWritesSameBytes pins the io.Writer entry point against Export:
// both must produce byte-identical documents, since CLI, docserver and
// embed-and-serve consumers route through different ones.
func TestExportToWritesSameBytes(t *testing.T) {
	exported, err := fixtureExporter().Export(fixtureCatalog())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	var buf bytes.Buffer
	if err := fixtureExporter().ExportTo(&buf, fixtureCatalog()); err != nil {
		t.Fatalf("ExportTo: %v", err)
	}

	if !bytes.Equal(exported, buf.Bytes()) {
		t.Errorf("ExportTo wrote %d bytes but Export produced %d", buf.Len(), len(exported))
	}
}

// TestLogoRendersWhenAllowed covers the happy path of the brand logo.
func TestLogoRendersWhenAllowed(t *testing.T) {
	exp := architecture.NewExporter(
		architecture.WithLogo("https://example.com/logo.png"),
		architecture.WithGeneratedAt(excludeFrozenAt()),
	)

	out, err := exp.Export(fixtureCatalog())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if !strings.Contains(
		string(out),
		`<img class="brand-logo" src="https://example.com/logo.png" alt="">`,
	) {
		t.Error("allowed logo was not rendered in the top bar")
	}
}

// TestLogoDroppedWhenUnsafe must hold even for a javascript: logo: the file
// is handed to third parties, so a hostile brand image leaves no trace.
func TestLogoDroppedWhenUnsafe(t *testing.T) {
	for _, hostile := range []string{
		`javascript:alert(1)`,
		`data:text/html,<script>alert(2)</script>`,
		`/etc/passwd.png`,
		` JAVASCRIPT:alert(3)`,
	} {
		exp := architecture.NewExporter(
			architecture.WithLogo(hostile),
			architecture.WithGeneratedAt(excludeFrozenAt()),
		)

		out, err := exp.Export(fixtureCatalog())
		if err != nil {
			t.Fatalf("Export(%q): %v", hostile, err)
		}

		if strings.Contains(string(out), hostile) {
			t.Errorf("hostile logo %q reached the output", hostile)
		}

		if strings.Contains(string(out), `<img class="brand-logo"`) {
			t.Errorf("hostile logo %q still rendered an image element", hostile)
		}
	}
}
