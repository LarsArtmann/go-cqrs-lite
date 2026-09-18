package docserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestProbeScalarMountDOM(t *testing.T) {
	browser := os.Getenv("CQRS_BROWSER")
	if browser == "" {
		t.Skip("probe")
	}
	ds := NewDocsServer(testProvider, Config{ServiceName: "Test Service", Version: "1.0.0", Description: "A test service", EnableCSP: true})
	mux := http.NewServeMux()
	ds.RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	dom, console := renderWithBrowser(t, browser, srv.URL+"/docs/asyncapi")
	if err := os.WriteFile("/tmp/asyncapi-dom.html", []byte(dom), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/scalar-console.log", []byte(console), 0o600); err != nil {
		t.Fatal(err)
	}
}
