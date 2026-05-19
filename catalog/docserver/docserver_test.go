package docserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
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
				Schema: &catalog.Schema{
					Type: "object",
					Properties: map[string]catalog.Property{
						"name": {Type: "string", Description: "Item name"},
					},
					Required: []string{"name"},
				},
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

func TestDocsServer_OpenAPISpecJSON(t *testing.T) {
	ds := testServer(t)
	handler := ds.OpenAPISpec()

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content type, got %s", ct)
	}

	var doc map[string]any
	if err := json.NewDecoder(w.Body).Decode(&doc); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

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

func TestDocsServer_OpenAPISpecYAML(t *testing.T) {
	ds := testServer(t)
	handler := ds.OpenAPISpecYAML()

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/yaml") {
		t.Errorf("expected text/yaml content type, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "openapi:") || !strings.Contains(body, "3.0.3") {
		t.Errorf("expected YAML to contain openapi version, got:\n%s", body)
	}
}

func TestDocsServer_OpenAPIUI(t *testing.T) {
	ds := testServer(t)
	handler := ds.OpenAPIUI()

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content type, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Scalar.createApiReference") {
		t.Error("expected Scalar JS initialization in HTML")
	}

	if !strings.Contains(body, "/docs/openapi.json") {
		t.Error("expected spec URL reference in HTML")
	}
}

func TestDocsServer_AsyncAPISpecJSON(t *testing.T) {
	ds := testServer(t)
	handler := ds.AsyncAPISpec()

	req := httptest.NewRequest(http.MethodGet, "/docs/asyncapi.json", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var doc map[string]any
	if err := json.NewDecoder(w.Body).Decode(&doc); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if doc["asyncapi"] != "3.0.0" {
		t.Errorf("expected asyncapi 3.0.0, got %v", doc["asyncapi"])
	}
}

func TestDocsServer_AsyncAPISpecYAML(t *testing.T) {
	ds := testServer(t)
	handler := ds.AsyncAPISpecYAML()

	req := httptest.NewRequest(http.MethodGet, "/docs/asyncapi.yaml", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/yaml") {
		t.Errorf("expected text/yaml content type, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "asyncapi:") || !strings.Contains(body, "3.0.0") {
		t.Errorf("expected YAML to contain asyncapi version, got:\n%s", body)
	}
}

func TestDocsServer_AsyncAPIUI(t *testing.T) {
	ds := testServer(t)
	handler := ds.AsyncAPIUI()

	req := httptest.NewRequest(http.MethodGet, "/docs/asyncapi", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "AsyncApiStandalone.render") {
		t.Error("expected AsyncApiStandalone JS initialization in HTML")
	}

	if !strings.Contains(body, "/docs/asyncapi.json") {
		t.Error("expected spec URL reference in HTML")
	}
}

func TestDocsServer_CatalogJSON(t *testing.T) {
	ds := testServer(t)
	handler := ds.CatalogJSON()

	req := httptest.NewRequest(http.MethodGet, "/docs/catalog.json", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var cat map[string]any
	if err := json.NewDecoder(w.Body).Decode(&cat); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if cat["title"] != "Test API" {
		t.Errorf("expected title 'Test API', got %v", cat["title"])
	}
}

func TestDocsServer_RegisterRoutes(t *testing.T) {
	ds := testServer(t)
	mux := http.NewServeMux()
	ds.RegisterRoutes(mux)

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
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("route %s: expected 200, got %d", tc.path, w.Code)
		}
	}
}

func TestDocsServer_DefaultConfig(t *testing.T) {
	ds := NewDocsServer(testProvider, Config{
		ServiceName: "Test",
		Version:     "1.0.0",
	})

	if ds.config.DocsPath != "/docs" {
		t.Errorf("expected default DocsPath /docs, got %s", ds.config.DocsPath)
	}

	if ds.config.BasePath != "/api" {
		t.Errorf("expected default BasePath /api, got %s", ds.config.BasePath)
	}

	if ds.config.AsyncAPIServer.Protocol != "http" {
		t.Errorf("expected default asyncapi protocol http, got %s", ds.config.AsyncAPIServer.Protocol)
	}
}

func TestDocsServer_CustomDocsPath(t *testing.T) {
	ds := NewDocsServer(testProvider, Config{
		ServiceName: "Test",
		Version:     "1.0.0",
		DocsPath:    "/api/v1/docs",
	})

	mux := http.NewServeMux()
	ds.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs/openapi.json", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
