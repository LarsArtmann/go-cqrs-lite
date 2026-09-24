package docserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// dataProductTestProvider builds a catalog with one data product carrying
// input/output ports and an output contract.
func dataProductTestProvider() *catalog.Catalog {
	reg := catalog.NewRegistry("Mesh API", "1.0.0")
	reg.AddService(catalog.Service{ID: "orders", Name: "Orders", Version: "1.0.0"})
	reg.AddEvent("orders", catalog.Message{
		Kind: catalog.EventMessage, ID: "order.placed", Name: "Order Placed",
		Version: "1.0.0", Summary: "placed", Direction: catalog.Sends,
	})
	reg.AddEvent("orders", catalog.Message{
		Kind: catalog.EventMessage, ID: "invoice.issued", Name: "Invoice Issued",
		Version: "1.0.0", Summary: "issued", Direction: catalog.Receives,
		Producers: []catalog.ServiceID{"billing"},
	})
	reg.AddDataProduct(catalog.DataProduct{
		ID: "order-lifecycle", Name: "Order Lifecycle", Version: "1.0.0",
		Summary: "Order facts for analytics", Owners: []string{"orders-team"},
		Inputs: []catalog.Ref{{ID: "invoice.issued", Version: "1.0.0"}},
		Outputs: []catalog.DataProductOutput{
			{
				Ref: catalog.Ref{ID: "order.placed", Version: "1.0.0"},
				Contract: &catalog.DataContract{
					Path: "contracts/order-placed.yaml",
					Name: "order-placed",
				},
			},
		},
	})

	return reg.Build()
}

func dataProductTestServer(t *testing.T) *DocsServer {
	t.Helper()

	return NewDocsServer(
		dataProductTestProvider,
		Config{ServiceName: "Mesh Service", Version: "1.0.0"},
	)
}

func TestDocsServer_EventCatalog_DataProductOverviewSection(t *testing.T) {
	srv := dataProductTestServer(t)

	req := newTestRequest("/docs/eventcatalog")
	recorder := httptest.NewRecorder()
	srv.serveEventCatalog(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		"Data Products",
		"Order Lifecycle",
		"orders-team",
		`href="/docs/eventcatalog/data-products/order-lifecycle"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected overview page to contain %q", expected)
		}
	}
}

func TestDocsServer_EventCatalog_DataProductDetail(t *testing.T) {
	srv := dataProductTestServer(t)

	req := newTestRequest("/docs/eventcatalog/data-products/order-lifecycle")
	req.SetPathValue("id", "order-lifecycle")
	recorder := httptest.NewRecorder()
	srv.serveEventCatalogDataProduct(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		"Order Lifecycle",
		"order-lifecycle",
		"orders-team",
		"Input ports",
		"invoice.issued",
		"Output ports",
		"order.placed",
		"contracts/order-placed.yaml",
		`href="/docs/eventcatalog/messages/order.placed"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected data product page to contain %q", expected)
		}
	}
}

func TestDocsServer_EventCatalog_DataProductNotFound(t *testing.T) {
	srv := dataProductTestServer(t)

	req := newTestRequest("/docs/eventcatalog/data-products/nope")
	req.SetPathValue("id", "nope")
	recorder := httptest.NewRecorder()
	srv.serveEventCatalogDataProduct(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("not-found page should render 200 with a message, got %d", recorder.Code)
	}

	if body := recorder.Body.String(); !strings.Contains(body, "nope") {
		t.Errorf("not-found page should name the missing data product")
	}
}
