package adapters_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/cattest"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func TestBuilder_AddEventFromType(t *testing.T) {
	t.Parallel()

	type userDeleted struct {
		*event.CatalogCore

		Reason string `doc:"Deletion reason" json:"reason"`
	}

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "")

	adapters.AddEventFromType[userDeleted](
		builder,
		"user-svc",
		"user.deleted",
		event.CatalogMeta{
			Name:    "UserDeleted",
			Version: "1.0.0",
			Summary: "User was deleted",
		},
		catalog.Sends,
	)

	cat := builder.Build()
	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Events", svc.Events, 1)

	evtMsg := svc.Events[0]
	if evtMsg.Name != "UserDeleted" {
		t.Errorf("name = %q, want UserDeleted", evtMsg.Name)
	}

	if evtMsg.Direction != catalog.Sends {
		t.Errorf("direction = %v, want sends", evtMsg.Direction)
	}

	if evtMsg.Schema == nil {
		t.Fatal("schema should not be nil")
	}

	cattest.AssertSchemaProperty(t, evtMsg.Schema, "reason")

	if _, ok := evtMsg.Schema.Properties["CatalogCore"]; ok {
		t.Error("schema should NOT contain embedded CatalogCore")
	}
}

func TestBuilder_AddServiceToDomain(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("order-svc", "Order Service", "1.0.0", "")
	builder.AddService("payment-svc", "Payment Service", "1.0.0", "")
	builder.AddDomain("ecommerce", "E-Commerce", "Online store", []string{"order-svc"})

	err := builder.AddServiceToDomain("payment-svc", "ecommerce")
	if err != nil {
		t.Fatalf("add service to domain: %v", err)
	}

	cat := builder.Build()
	cattest.AssertSliceLen(t, "cat.Domains", cat.Domains, 1)

	d := cat.Domains[0]
	if len(d.Services) != 2 {
		t.Errorf("expected 2 services in domain, got %d", len(d.Services))
	}

	found := map[string]bool{}
	for _, sid := range d.Services {
		found[sid] = true
	}

	if !found["order-svc"] || !found["payment-svc"] {
		t.Errorf("domain services = %v, want both order-svc and payment-svc", d.Services)
	}
}

func TestBuilder_AddServiceToDomain_NonexistentDomain(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("svc", "Service", "1.0.0", "")

	err := builder.AddServiceToDomain("svc", "nonexistent")
	if err == nil {
		t.Fatal("expected error when adding service to nonexistent domain")
	}
}

func TestBuilder_AddChannel(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")

	ch := catalog.Channel{
		ID:      "order-events",
		Name:    "Order Events Channel",
		Version: "1.0.0",
		Address: "orders.events",
	}
	builder.AddChannel(ch)

	cat := builder.Build()
	cattest.AssertSliceLen(t, "cat.Channels", cat.Channels, 1)

	got := cat.Channels[0]
	if got.ID != "order-events" {
		t.Errorf("channel ID = %q, want order-events", got.ID)
	}

	if got.Address != "orders.events" {
		t.Errorf("channel address = %q, want orders.events", got.Address)
	}
}

func TestBuilder_FromQueryDispatcher(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()
	d.RegisterCatalogEntry("user.get", query.CatalogMeta{
		Name:    "GetUser",
		Version: "1.0.0",
		Summary: "Gets a user by ID",
	})

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "")
	adapters.FromQueryDispatcher(builder, "user-svc", d)

	cat := builder.Build()
	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Queries", svc.Queries, 1)

	qry := svc.Queries[0]
	if qry.ID != "user.get" {
		t.Errorf("ID = %q, want user.get", qry.ID)
	}

	if qry.Name != "GetUser" {
		t.Errorf("name = %q, want GetUser", qry.Name)
	}
}
