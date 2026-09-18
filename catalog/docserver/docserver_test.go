package docserver

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func testProvider() *catalog.Catalog {
	reg := catalog.NewRegistry("Test API", "1.0.0")
	reg.AddService(catalog.Service{
		ID:      "test-svc",
		Name:    "Test Service",
		Version: "1.0.0",
		Commands: []catalog.Message{
			{
				Kind:      catalog.CommandMessage,
				ID:        "create-item",
				Name:      "CreateItem",
				Version:   "1.0.0",
				Summary:   "Creates a new item",
				Direction: catalog.Receives,
				Schema:    cattest.CreateItemSchema(),
			},
		},
		Queries: []catalog.Message{
			{
				Kind:      catalog.QueryMessage,
				ID:        "get-item",
				Name:      "GetItem",
				Version:   "1.0.0",
				Summary:   "Gets an item by ID",
				Direction: catalog.Receives,
			},
		},
		Events: []catalog.Message{
			{
				Kind:      catalog.EventMessage,
				ID:        "item-created",
				Name:      "ItemCreated",
				Version:   "1.0.0",
				Summary:   "An item was created",
				Direction: catalog.Sends,
			},
		},
	})

	return reg.Build()
}

func testServer(t *testing.T) *DocsServer {
	t.Helper()

	return NewDocsServer(testProvider, Config{
		ServiceName: "Test Service",
		Version:     "1.0.0",
		Description: "A test service",
	})
}

func decodeJSON(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.UnmarshalRead(recorder.Body, &doc); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	return doc
}

func newTestRequest(path string) *http.Request {
	return httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, path, nil,
	)
}

func TestDocsServer_OpenAPISpecJSON(t *testing.T) {
	srv := testServer(t)
	handler := srv.OpenAPISpec()

	req := newTestRequest("/docs/openapi.json")
	recorder := httptest.NewRecorder()
	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}

	ct := recorder.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content type, got %s", ct)
	}

	doc := decodeJSON(t, recorder)
	if doc["openapi"] != "3.0.3" {
		t.Errorf("expected openapi 3.0.3, got %v", doc["openapi"])
	}

	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatal("expected info to be a map")
	}

	if info["title"] != "Test Service" {
		t.Errorf("expected title 'Test Service', got %v", info["title"])
	}
}

// TestDocsServer_OpenAPISpecRequestScopedServers pins the Try-It console
// contract: the served document names the REQUEST's host as the server, so
// "Try it" calls hit the deployment host instead of resolving the exporter's
// relative default against the document URL, and behind a reverse proxy the
// forwarded scheme wins over the local one.
func TestDocsServer_OpenAPISpecRequestScopedServers(t *testing.T) {
	srv := testServer(t)
	handler := srv.OpenAPISpec()

	req := newTestRequest("/docs/openapi.json")
	req.Host = "docs.example.test"
	req.Header.Set("X-Forwarded-Proto", "https")
	recorder := httptest.NewRecorder()
	handler(recorder, req)

	servers, ok := decodeJSON(t, recorder)["servers"].([]any)
	if !ok || len(servers) != 1 {
		t.Fatalf(
			"expected exactly one request-derived server, got %v",
			decodeJSON(t, recorder)["servers"],
		)
	}

	server, ok := servers[0].(map[string]any)
	if !ok || server["url"] != "https://docs.example.test" {
		t.Errorf("expected server url https://docs.example.test, got %v", servers)
	}

	plainReq := newTestRequest("/docs/openapi.json")
	plainReq.Host = ""
	plainRecorder := httptest.NewRecorder()
	handler(plainRecorder, plainReq)

	defaultDoc := decodeJSON(t, plainRecorder)
	baseRaw, err := json.Marshal(srv.buildOpenAPI().Servers)
	if err != nil {
		t.Fatal(err)
	}
	var baseDecoded any
	if err := json.Unmarshal(baseRaw, &baseDecoded); err != nil {
		t.Fatal(err)
	}
	// Compare decoded values: encoding/json/v2 does not sort map keys, so a
	// byte-wise comparison against the typed struct marshal is order-flaky.
	if !reflect.DeepEqual(defaultDoc["servers"], baseDecoded) {
		t.Errorf(
			"empty Host must preserve exporter default servers %s, got %v",
			baseRaw,
			defaultDoc["servers"],
		)
	}
}

func TestDocsServer_OpenAPISpecYAML(t *testing.T) {
	srv := testServer(t)
	handler := srv.OpenAPISpecYAML()

	req := newTestRequest("/docs/openapi.yaml")
	recorder := httptest.NewRecorder()
	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}

	ct := recorder.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/yaml") {
		t.Errorf("expected text/yaml content type, got %s", ct)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "openapi:") || !strings.Contains(body, "3.0.3") {
		t.Errorf("expected YAML to contain openapi version, got:\n%s", body)
	}
}

func TestDocsServer_OpenAPIUI(t *testing.T) {
	srv := testServer(t)
	handler := srv.OpenAPIUI()

	req := newTestRequest("/docs/openapi")
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
	if !strings.Contains(body, "Scalar.createApiReference") {
		t.Error("expected Scalar JS initialization in HTML")
	}

	if !strings.Contains(body, "/docs/openapi.json") {
		t.Error("expected spec URL reference in HTML")
	}
}

func TestDocsServer_AsyncAPISpecJSON(t *testing.T) {
	srv := testServer(t)
	handler := srv.AsyncAPISpec()

	req := newTestRequest("/docs/asyncapi.json")
	recorder := httptest.NewRecorder()
	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}

	doc := decodeJSON(t, recorder)
	if doc["asyncapi"] != "3.0.0" {
		t.Errorf("expected asyncapi 3.0.0, got %v", doc["asyncapi"])
	}
}

func TestDocsServer_AsyncAPISpecYAML(t *testing.T) {
	srv := testServer(t)
	handler := srv.AsyncAPISpecYAML()

	req := newTestRequest("/docs/asyncapi.yaml")
	recorder := httptest.NewRecorder()
	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}

	ct := recorder.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/yaml") {
		t.Errorf("expected text/yaml content type, got %s", ct)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "asyncapi:") || !strings.Contains(body, "3.0.0") {
		t.Errorf("expected YAML to contain asyncapi version, got:\n%s", body)
	}
}

func TestDocsServer_AsyncAPIUI(t *testing.T) {
	srv := testServer(t)
	handler := srv.AsyncAPIUI()

	req := newTestRequest("/docs/asyncapi")
	recorder := httptest.NewRecorder()
	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "AsyncApiStandalone.render") {
		t.Error("expected AsyncApiStandalone JS initialization in HTML")
	}

	if !strings.Contains(body, "/docs/asyncapi.json") {
		t.Error("expected spec URL reference in HTML")
	}
}

func TestDocsServer_CatalogJSON(t *testing.T) {
	srv := testServer(t)
	handler := srv.CatalogJSON()

	req := newTestRequest("/docs/catalog.json")
	recorder := httptest.NewRecorder()
	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}

	var cat map[string]any
	if err := json.UnmarshalRead(recorder.Body, &cat); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if cat["title"] != "Test API" {
		t.Errorf("expected title 'Test API', got %v", cat["title"])
	}
}

func TestDocsServer_RegisterRoutes(t *testing.T) {
	srv := testServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	routes := []struct {
		path string
	}{
		{"/docs/openapi"},
		{"/docs/openapi.json"},
		{"/docs/openapi.yaml"},
		{"/docs/asyncapi"},
		{"/docs/asyncapi.json"},
		{"/docs/asyncapi.yaml"},
		{"/docs/catalog.json"},
	}

	for _, tc := range routes {
		req := newTestRequest(tc.path)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("route %s: expected 200, got %d", tc.path, recorder.Code)
		}
	}
}

func TestDocsServer_StaticFS(t *testing.T) {
	srv := testServer(t)
	fsys := srv.StaticFS()

	if fsys == nil {
		t.Fatal("expected non-nil FileSystem")
	}

	for _, name := range []string{
		"scalar.js",
		"asyncapi-react.js",
		"asyncapi-react.css",
	} {
		f, err := fsys.Open(name)
		if err != nil {
			t.Errorf("expected to open %s: %v", name, err)

			continue
		}

		_ = f.Close()
	}
}

func TestDocsServer_DefaultConfig(t *testing.T) {
	srv := NewDocsServer(testProvider, Config{
		ServiceName: "Test",
		Version:     "1.0.0",
	})

	if srv.config.DocsPath != "/docs" {
		t.Errorf("expected default DocsPath /docs, got %s", srv.config.DocsPath)
	}

	if srv.config.BasePath != "/api" {
		t.Errorf("expected default BasePath /api, got %s", srv.config.BasePath)
	}

	if srv.config.AsyncAPIServer.Protocol != "http" {
		t.Errorf(
			"expected default asyncapi protocol http, got %s",
			srv.config.AsyncAPIServer.Protocol,
		)
	}
}

func TestDocsServer_CustomDocsPath(t *testing.T) {
	srv := NewDocsServer(testProvider, Config{
		ServiceName: "Test",
		Version:     "1.0.0",
		DocsPath:    "/api/v1/docs",
	})

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	req := newTestRequest("/api/v1/docs/openapi.json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}
}

func TestDocsServer_RegisterRoutes_StaticFiles(t *testing.T) {
	srv := NewDocsServer(testProvider, Config{
		ServiceName: "Test",
		Version:     "1.0.0",
	})
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	for _, tc := range []struct {
		path        string
		contentType string
	}{
		{"/docs/static/asyncapi-react.js", "text/javascript"},
		{"/docs/static/asyncapi-react.css", "text/css"},
		{"/docs/static/scalar.js", "text/javascript"},
		{"/docs/static/docs-ui.css", "text/css"},
	} {
		req := newTestRequest(tc.path)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d", tc.path, recorder.Code)

			continue
		}

		ct := recorder.Header().Get("Content-Type")
		if !strings.Contains(ct, tc.contentType) {
			t.Errorf("GET %s: expected %s content type, got %s", tc.path, tc.contentType, ct)
		}
	}
}
