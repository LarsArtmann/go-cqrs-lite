package eventcatalog

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func TestAutoDerive_MultipleProducers(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc-a", Name: "Service A", Version: "1.0.0"})
	reg.AddEvent("svc-a", newEvent("OrderPlaced", "Order Placed", catalog.Sends))
	reg.AddService(catalog.Service{ID: "svc-b", Name: "Service B", Version: "1.0.0"})
	reg.AddEvent("svc-b", newEvent("OrderPlaced", "Order Placed", catalog.Sends))

	cat := reg.Build()
	enriched := autoDeriveProducersConsumers(cat)

	evt := enriched.Services[0].Events[0]
	if len(evt.Producers) != 2 {
		t.Errorf("expected 2 producers, got %d", len(evt.Producers))
	}
}

func TestAutoDerive_CommandsGetConsumers(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: "Create Order",
		Version: "1.0.0",
	})

	cat := reg.Build()
	enriched := autoDeriveProducersConsumers(cat)

	cmd := enriched.Services[0].Commands[0]
	if len(cmd.Consumers) != 1 || cmd.Consumers[0] != "svc" {
		t.Errorf("expected consumers [svc], got %v", cmd.Consumers)
	}
}

func TestAutoDerive_DoesNotOverrideExistingProducers(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
	reg.AddEvent("svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "OrderPlaced", Name: "Order Placed",
		Version: "1.0.0", Direction: catalog.Sends,
		Producers: []catalog.ServiceID{"explicit-producer"},
	})

	cat := reg.Build()
	enriched := autoDeriveProducersConsumers(cat)

	evt := enriched.Services[0].Events[0]
	if len(evt.Producers) != 1 || evt.Producers[0] != "explicit-producer" {
		t.Errorf("expected explicit producers preserved, got %v", evt.Producers)
	}
}

func TestAutoDerive_EmptyCatalog(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	cat := reg.Build()
	enriched := autoDeriveProducersConsumers(cat)

	if len(enriched.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(enriched.Services))
	}
}

func TestAutoDerive_QueryGetsConsumers(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
	reg.AddQuery("svc", catalog.Message{
		Kind: catalog.QueryMessage, ID: "GetOrder", Name: "Get Order",
		Version: "1.0.0",
	})

	cat := reg.Build()
	enriched := autoDeriveProducersConsumers(cat)

	q := enriched.Services[0].Queries[0]
	if len(q.Consumers) != 1 || q.Consumers[0] != "svc" {
		t.Errorf("expected consumers [svc], got %v", q.Consumers)
	}
}
