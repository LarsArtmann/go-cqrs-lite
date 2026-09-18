package docserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// catalogTestProvider builds a catalog rich enough to exercise every event
// catalog page: producers/consumers, channels, and a data store.
func catalogTestProvider() *catalog.Catalog {
	reg := catalog.NewRegistry("Test API", "1.0.0")
	reg.AddService(catalog.Service{
		ID:      "orders",
		Name:    "Order Service",
		Version: "1.0.0",
		Summary: "Handles orders",
		Events: []catalog.Message{
			{
				Kind:      catalog.EventMessage,
				ID:        "order-created",
				Name:      "OrderCreated",
				Version:   "1.2.0",
				Summary:   "An order was created",
				Direction: catalog.Sends,
				Schema: &catalog.Schema{
					Type: "object",
					Properties: map[string]catalog.Property{
						"orderID": {
							Type:        "string",
							Description: "Order identifier",
							Format:      "uuid",
						},
						"total": {Type: "integer", Description: "Order total in cents"},
						"tags": {
							Type:        "array",
							Items:       &catalog.Property{Type: "string"},
							Description: "Free-form tags",
						},
					},
					Required: []string{"orderID"},
				},
				Producers: []catalog.ServiceID{"orders"},
				Consumers: []catalog.ServiceID{"billing"},
				Channels:  []catalog.ChannelID{"orders-channel"},
				Examples:  nil,
			},
		},
		WritesTo: []catalog.DataStoreID{"orders-db"},
	})
	reg.AddService(catalog.Service{
		ID:      "billing",
		Name:    "Billing Service",
		Version: "0.9.0",
		Queries: []catalog.Message{
			{
				Kind:      catalog.QueryMessage,
				ID:        "get-invoice",
				Name:      "GetInvoice",
				Version:   "1.0.0",
				Summary:   "Fetches an invoice",
				Direction: catalog.Receives,
			},
		},
	})
	reg.AddChannel(catalog.Channel{
		ID:                "orders-channel",
		Name:              "Orders Channel",
		Version:           "1.0.0",
		Address:           "orders",
		Protocols:         []catalog.Protocol{"kafka"},
		Messages:          []catalog.MessageID{"order-created"},
		DeliveryGuarantee: catalog.DeliveryExactlyOnce,
	})

	return reg.Build()
}

func catalogTestServer(t *testing.T) *DocsServer {
	t.Helper()

	return NewDocsServer(catalogTestProvider, Config{
		ServiceName: "Test Service",
		Version:     "1.0.0",
	})
}

func TestDocsServer_EventCatalog_Overview(t *testing.T) {
	srv := catalogTestServer(t)

	req := newTestRequest("/docs/eventcatalog")
	recorder := httptest.NewRecorder()
	srv.serveEventCatalog(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		"Event Catalog",
		`href="/docs/eventcatalog/messages/order-created"`,
		`href="/docs/eventcatalog/channels/orders-channel"`,
		`href="/docs/eventcatalog/services/orders"`,
		"OrderCreated",
		"kafka",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected overview page to contain %q", expected)
		}
	}
}

func TestDocsServer_EventCatalog_MessageDetail(t *testing.T) {
	srv := catalogTestServer(t)

	req := newTestRequest("/docs/eventcatalog/messages/order-created")
	req.SetPathValue("id", "order-created")
	recorder := httptest.NewRecorder()
	srv.serveEventCatalogMessage(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		"OrderCreated",
		"An order was created",
		"Order Service",
		"Billing Service",
		"orderID",
		"array of",
		"Raw JSON schema",
		`href="/docs/eventcatalog/channels/orders-channel"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected message detail page to contain %q", expected)
		}
	}
}

func TestDocsServer_EventCatalog_MessageDetail_NotFound(t *testing.T) {
	srv := catalogTestServer(t)

	req := newTestRequest("/docs/eventcatalog/messages/does-not-exist")
	recorder := httptest.NewRecorder()
	srv.serveEventCatalogMessage(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected rendered not-found page with 200, got %d", recorder.Code)
	}

	if body := recorder.Body.String(); !strings.Contains(body, "Unknown message") {
		t.Error("expected not-found page for unknown message")
	}
}

func TestDocsServer_EventCatalog_ChannelAndServiceDetail(t *testing.T) {
	srv := catalogTestServer(t)

	t.Run("channel", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := newTestRequest("/docs/eventcatalog/channels/orders-channel")
		req.SetPathValue("id", "orders-channel")
		srv.serveEventCatalogChannel(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}

		body := recorder.Body.String()
		for _, expected := range []string{"Orders Channel", "orders", "kafka", "OrderCreated"} {
			if !strings.Contains(body, expected) {
				t.Errorf("expected channel page to contain %q", expected)
			}
		}
	})

	t.Run("service", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := newTestRequest("/docs/eventcatalog/services/orders")
		req.SetPathValue("id", "orders")
		srv.serveEventCatalogService(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}

		body := recorder.Body.String()
		for _, expected := range []string{"Order Service", "orders-db", "OrderCreated"} {
			if !strings.Contains(body, expected) {
				t.Errorf("expected service page to contain %q", expected)
			}
		}
	})

	t.Run("unknown channel is not-found page", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := newTestRequest("/docs/eventcatalog/channels/nope")
		req.SetPathValue("id", "nope")
		srv.serveEventCatalogChannel(recorder, req)

		if !strings.Contains(recorder.Body.String(), "Unknown channel") {
			t.Error("expected not-found page for unknown channel")
		}
	})
}

func TestDocsServer_EventCatalog_RegisterRoutes(t *testing.T) {
	srv := catalogTestServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	for _, path := range []string{
		"/docs/eventcatalog",
		"/docs/eventcatalog/messages/order-created",
		"/docs/eventcatalog/channels/orders-channel",
		"/docs/eventcatalog/services/orders",
	} {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, newTestRequest(path))

		if recorder.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", path, recorder.Code)
		}
	}
}

func TestDocsServer_OpenAPIServers_UsesRequestHost(t *testing.T) {
	srv := testServer(t)

	req := newTestRequest("/docs/openapi.json")
	req.Host = "api.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	recorder := httptest.NewRecorder()
	srv.OpenAPISpec()(recorder, req)

	doc := decodeJSON(t, recorder)
	servers, ok := doc["servers"].([]any)
	if !ok || len(servers) == 0 {
		t.Fatal("expected servers array in OpenAPI document")
	}

	first, _ := servers[0].(map[string]any)
	if first["url"] != "https://api.example.com" {
		t.Errorf("expected server url derived from request, got %v", first["url"])
	}
}

func TestDocsServer_OpenAPIServers_KeepsRelativeWithoutHost(t *testing.T) {
	srv := testServer(t)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/docs/openapi.json",
		nil,
	)
	req.Host = ""

	recorder := httptest.NewRecorder()
	srv.OpenAPISpec()(recorder, req)

	doc := decodeJSON(t, recorder)
	servers, _ := doc["servers"].([]any)
	if len(servers) == 0 {
		t.Fatal("expected fallback servers entry")
	}

	first, _ := servers[0].(map[string]any)
	if first["url"] != "." {
		t.Errorf("expected exporter default \".\" server url, got %v", first["url"])
	}
}

func TestDocsServer_D2View_WithRenderHook(t *testing.T) {
	srv := testServer(t)
	srv.config.D2SVG = func(d2Source string) (string, error) {
		return "<svg><g>" + d2Source + "</g></svg>", nil
	}

	recorder := httptest.NewRecorder()
	srv.D2View()(recorder, newTestRequest("/docs/d2"))

	body := recorder.Body.String()
	if !strings.Contains(body, "<svg><g>") {
		t.Error("expected rendered diagram SVG on the D2 page")
	}

	if !strings.Contains(body, "D2 source") {
		t.Error("expected collapsible D2 source section")
	}
}

func TestDocsServer_D2View_RenderHookErrorFallsBackToSource(t *testing.T) {
	srv := testServer(t)
	srv.config.D2SVG = func(string) (string, error) {
		return "", errors.New("d2 binary not found")
	}

	recorder := httptest.NewRecorder()
	srv.D2View()(recorder, newTestRequest("/docs/d2"))

	body := recorder.Body.String()
	if strings.Contains(body, "<svg><g>") {
		t.Error("expected no SVG when the hook errors")
	}

	if !strings.Contains(body, "Diagram rendering not configured") {
		t.Error("expected fallback hint when the hook errors")
	}
}
