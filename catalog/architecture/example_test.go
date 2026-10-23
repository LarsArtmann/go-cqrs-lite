package architecture_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

// Example_exportAndServe shows the one-call capability fleet serving apps
// use: render the reference once, embed the bytes, serve them from an
// existing route.
func Example_exportAndServe() {
	out, err := architecture.NewExporter(
		architecture.WithTitle("Ordering"),
	).Export(fixtureCatalog())
	if err != nil {
		fmt.Println("render:", err)

		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/docs/architecture", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(out)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/docs/architecture")
	if err != nil {
		fmt.Println("serve:", err)

		return
	}

	defer resp.Body.Close()

	fmt.Println(resp.StatusCode, resp.Header.Get("Content-Type"))

	// Output: 200 text/html; charset=utf-8
}

// TestServedReferenceIsSelfContained pins that what an app serves is the
// same offline artifact the CLI writes: no external URLs, no external
// assets, and the embedded bytes survive a render/serve round trip.
func TestServedReferenceIsSelfContained(t *testing.T) {
	out, err := architecture.NewExporter(
		architecture.WithTitle("Ordering"),
		architecture.WithGeneratedAt(excludeFrozenAt()),
	).Export(fixtureCatalog())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(out)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("serve: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	buf := new(bytes.Buffer)

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read: %v", err)
	}

	served := buf.String()

	for _, forbidden := range []string{
		`src="http`,
		`href="http`,
		"<link ",
		"@import",
	} {
		if strings.Contains(served, forbidden) {
			t.Errorf("served reference is not self-contained: %q", forbidden)
		}
	}

	// The write path shares the same bytes: ExportFile must equal Export.
	path := filepath.Join(t.TempDir(), "reference.html")

	if err := architecture.NewExporter(
		architecture.WithTitle("Ordering"),
		architecture.WithGeneratedAt(excludeFrozenAt()),
	).ExportFile(fixtureCatalog(), path); err != nil {
		t.Fatalf("ExportFile: %v", err)
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if !bytes.Equal(out, written) {
		t.Error("ExportFile bytes differ from Export bytes")
	}
}
