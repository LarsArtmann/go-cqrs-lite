package docserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/internal/cattest"
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
	if err := json.NewDecoder(recorder.Body).Decode(&doc); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	return doc
}

func TestDocsServer_OpenAPISpecJSON(t *testing.T) {
	srv := testServer(t)
	handler := srv.OpenAPISpec()

	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/docs/openapi.json", nil,
	)
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

func TestDocsServer_OpenAPISpecYAML(t *testing.T) {
	srv := testServer(t)
	handler := srv.OpenAPISpecYAML()

	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/docs/openapi.yaml", nil,
	)
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

	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/docs/openapi", nil,
	)
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

	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/docs/asyncapi.json", nil,
	)
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

	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/docs/asyncapi.yaml", nil,
	)
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

	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/docs/asyncapi", nil,
	)
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

	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/docs/catalog.json", nil,
	)
	recorder := httptest.NewRecorder()
	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}

	var cat map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&cat); err != nil {
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
		req := httptest.NewRequestWithContext(
			context.Background(), http.MethodGet, tc.path, nil,
		)
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

	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/api/v1/docs/openapi.json", nil,
	)
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
	} {
		req := httptest.NewRequestWithContext(
			context.Background(), http.MethodGet, tc.path, nil,
		)
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
