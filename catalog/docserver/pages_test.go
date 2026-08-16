package docserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func TestDocsServer_Index(t *testing.T) {
	srv := testServer(t)
	handler := srv.Index()

	req := newTestRequest("/docs")
	recorder := httptest.NewRecorder()
	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}

	ct := recorder.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content type, got %s", ct)
	}

	body := recorder.Body.String()

	if !strings.Contains(body, "Test API — API Documentation") {
		t.Error("expected page title in HTML")
	}

	for _, link := range []string{
		`href="/docs/openapi"`,
		`href="/docs/openapi.json"`,
		`href="/docs/openapi.yaml"`,
		`href="/docs/asyncapi"`,
		`href="/docs/asyncapi.json"`,
		`href="/docs/asyncapi.yaml"`,
		`href="/docs/d2"`,
		`href="/docs/d2.txt"`,
		`href="/docs/catalog.json"`,
	} {
		if !strings.Contains(body, link) {
			t.Errorf("expected artifact link %s on index page", link)
		}
	}

	if !strings.Contains(body, "Test Service") {
		t.Error("expected service card for test-svc on index page")
	}
}

func TestIndexPageData_DeduplicatesSharedMessages(t *testing.T) {
	shared := catalog.Message{
		Kind:      catalog.CommandMessage,
		ID:        "shared-cmd",
		Name:      "SharedCmd",
		Version:   "1.0.0",
		Direction: catalog.Receives,
	}
	reg := catalog.NewRegistry("Shared Catalog", "2.0.0")
	reg.AddService(catalog.Service{
		ID:       "svc-a",
		Name:     "Service A",
		Version:  "1.0.0",
		Commands: []catalog.Message{shared},
	})
	reg.AddService(catalog.Service{
		ID:       "svc-b",
		Name:     "Service B",
		Version:  "1.0.0",
		Commands: []catalog.Message{shared},
	})

	data := newIndexPageData(Config{ServiceName: "Shared", DocsPath: "/docs"}, reg.Build())

	if data.Stats.Services != 2 {
		t.Errorf("expected 2 services, got %d", data.Stats.Services)
	}

	if data.Stats.Commands != 1 {
		t.Errorf("expected 1 deduplicated command, got %d", data.Stats.Commands)
	}

	for _, svc := range data.Services {
		if svc.Commands != 1 {
			t.Errorf("service %s: expected per-service count 1, got %d", svc.ID, svc.Commands)
		}
	}
}

func TestDocsServer_Index_AbsoluteAssetsWithCustomDocsPath(t *testing.T) {
	srv := NewDocsServer(testProvider, Config{
		ServiceName: "Test Service",
		Version:     "1.0.0",
		DocsPath:    "/api/v1/docs",
	})

	req := newTestRequest("/api/v1/docs")
	recorder := httptest.NewRecorder()
	srv.Index()(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, asset := range []string{
		`href="/api/v1/docs/static/docs-ui.css"`,
		`href="/api/v1/docs/static/favicon.svg"`,
	} {
		if !strings.Contains(body, asset) {
			t.Errorf("expected absolute asset reference %s", asset)
		}
	}
}

func TestDocsServer_D2View(t *testing.T) {
	srv := testServer(t)

	req := newTestRequest("/docs/d2")
	recorder := httptest.NewRecorder()
	srv.D2View()(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}

	ct := recorder.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content type, got %s", ct)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, `href="/docs/d2.txt"`) {
		t.Error("expected download link to /docs/d2.txt")
	}

	if !strings.Contains(body, "Test API") {
		t.Error("expected D2 source content in view")
	}
}

func TestDocsServer_D2Diagram_MatchesStandaloneD2Handler(t *testing.T) {
	srv := testServer(t)

	req := newTestRequest("/docs/d2.txt")
	recorder := httptest.NewRecorder()
	srv.D2Diagram()(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}

	ct := recorder.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain content type, got %s", ct)
	}

	standalone := httptest.NewRecorder()
	D2Handler(testProvider())(standalone, newTestRequest("/d2"))

	if recorder.Body.String() != standalone.Body.String() {
		t.Error("expected D2Diagram handler output to equal standalone D2Handler output")
	}
}

func TestDocsServer_RegisterRoutes_IndexRedirectAndNotFound(t *testing.T) {
	srv := testServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	t.Run("index", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, newTestRequest("/docs"))

		if recorder.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recorder.Code)
		}
	})

	t.Run("trailing slash redirects", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, newTestRequest("/docs/"))

		if recorder.Code != http.StatusPermanentRedirect {
			t.Errorf("expected 308, got %d", recorder.Code)
		}

		if loc := recorder.Header().Get("Location"); loc != "/docs" {
			t.Errorf("expected redirect to /docs, got %s", loc)
		}
	})

	t.Run("unknown path is 404", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, newTestRequest("/docs/does-not-exist"))

		if recorder.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recorder.Code)
		}
	})
}

func TestDocsServer_SPAPages_ReferenceAbsoluteAssets(t *testing.T) {
	srv := testServer(t)

	t.Run("openapi", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		srv.OpenAPIUI()(recorder, newTestRequest("/docs/openapi"))
		body := recorder.Body.String()

		for _, expected := range []string{"/docs/static/scalar.js", `data-spec-url="/docs/openapi.json"`, "<noscript"} {
			if !strings.Contains(body, expected) {
				t.Errorf("openapi page: expected %s in HTML", expected)
			}
		}
	})

	t.Run("asyncapi", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		srv.AsyncAPIUI()(recorder, newTestRequest("/docs/asyncapi"))
		body := recorder.Body.String()

		expected := []string{
			"/docs/static/asyncapi-react.js",
			"/docs/static/asyncapi-react.css",
			`data-spec-url="/docs/asyncapi.json"`,
			"<noscript",
		}
		for _, exp := range expected {
			if !strings.Contains(body, exp) {
				t.Errorf("asyncapi page: expected %s in HTML", exp)
			}
		}
	})
}
