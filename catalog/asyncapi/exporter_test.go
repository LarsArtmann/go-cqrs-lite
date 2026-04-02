package asyncapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func TestExporter_Export_BasicCommand(t *testing.T) {
	t.Parallel()
	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "order-svc", Name: "Order Service", Version: "1.0.0"})
	reg.AddCommand("order-svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "CreateOrder",
		Name:    "CreateOrder",
		Version: "1.0.0",
		Summary: "Create a new order",
		Schema: &catalog.Schema{
			Type: "object",
			Properties: map[string]catalog.Property{
				"orderID": {Type: "string", Description: "Unique order ID"},
			},
		},
	})

	cat := reg.Build()
	exp := NewExporter("Order Service", "1.0.0")
	doc := exp.Export(cat)

	if doc.AsyncAPI != "3.0.0" {
		t.Errorf("AsyncAPI version = %q, want %q", doc.AsyncAPI, "3.0.0")
	}
	if doc.Info.Title != "Order Service" {
		t.Errorf("Info.Title = %q, want %q", doc.Info.Title, "Order Service")
	}
	if len(doc.Channels) == 0 {
		t.Error("expected at least one channel")
	}
	if len(doc.Operations) == 0 {
		t.Error("expected at least one operation")
	}

	ch, ok := doc.Channels["commands.CreateOrder"]
	if !ok {
		t.Fatal("missing commands.CreateOrder channel")
	}
	if ch.Address != "order-svc.commands.create.order" {
		t.Errorf("channel address = %q, want %q", ch.Address, "order-svc.commands.create.order")
	}

	op, ok := doc.Operations["receiveCreateOrder"]
	if !ok {
		t.Fatal("missing receiveCreateOrder operation")
	}
	if op.Action != "receive" {
		t.Errorf("operation action = %q, want %q", op.Action, "receive")
	}

	msg, ok := doc.Components.Messages["CreateOrder"]
	if !ok {
		t.Fatal("missing CreateOrder message component")
	}
	if msg.ContentType != "application/json" {
		t.Errorf("message content type = %q, want %q", msg.ContentType, "application/json")
	}
	if msg.Payload.Ref != "#/components/schemas/CreateOrder" {
		t.Errorf("payload ref = %q, want %q", msg.Payload.Ref, "#/components/schemas/CreateOrder")
	}

	schema, ok := doc.Components.Schemas["CreateOrder"]
	if !ok {
		t.Fatal("missing CreateOrder schema component")
	}
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("schema is not a map: %T", schema)
	}
	if schemaMap["type"] != "object" {
		t.Errorf("schema type = %v, want %q", schemaMap["type"], "object")
	}
}

func TestExporter_Export_Event(t *testing.T) {
	t.Parallel()
	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "payment-svc", Name: "Payment Service", Version: "1.0.0"})
	reg.AddEvent("payment-svc", catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        "PaymentProcessed",
		Name:      "PaymentProcessed",
		Version:   "1.0.0",
		Summary:   "Payment was processed",
		Direction: catalog.Sends,
	})

	cat := reg.Build()
	exp := NewExporter("Payment Service", "1.0.0")
	doc := exp.Export(cat)

	ch, ok := doc.Channels["events.PaymentProcessed"]
	if !ok {
		t.Fatal("missing events.PaymentProcessed channel")
	}
	if ch.Address != "payment-svc.events.payment.processed" {
		t.Errorf("channel address = %q", ch.Address)
	}

	op, ok := doc.Operations["publishPaymentProcessed"]
	if !ok {
		t.Fatal("missing publishPaymentProcessed operation")
	}
	if op.Action != "send" {
		t.Errorf("event action = %q, want %q", op.Action, "send")
	}
}

func TestExporter_Export_EventReceive(t *testing.T) {
	t.Parallel()
	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
	reg.AddEvent("svc", catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        "OrderCreated",
		Name:      "OrderCreated",
		Version:   "1.0.0",
		Direction: catalog.Receives,
	})

	cat := reg.Build()
	exp := NewExporter("Service", "1.0.0")
	doc := exp.Export(cat)

	op, ok := doc.Operations["receiveOrderCreated"]
	if !ok {
		t.Fatal("missing receiveOrderCreated operation")
	}
	if op.Action != "receive" {
		t.Errorf("action = %q, want %q", op.Action, "receive")
	}
}

func TestExporter_Export_Query(t *testing.T) {
	t.Parallel()
	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "catalog-svc", Name: "Catalog Service", Version: "1.0.0"})
	reg.AddQuery("catalog-svc", catalog.Message{
		Kind:    catalog.QueryMessage,
		ID:      "GetProduct",
		Name:    "GetProduct",
		Version: "1.0.0",
		Summary: "Get product details",
	})

	cat := reg.Build()
	exp := NewExporter("Catalog Service", "1.0.0")
	doc := exp.Export(cat)

	_, ok := doc.Channels["queries.GetProduct"]
	if !ok {
		t.Fatal("missing queries.GetProduct channel")
	}

	op, ok := doc.Operations["handleGetProduct"]
	if !ok {
		t.Fatal("missing handleGetProduct operation")
	}
	if op.Action != "receive" {
		t.Errorf("query action = %q, want %q", op.Action, "receive")
	}
}

func TestExporter_Export_Servers(t *testing.T) {
	t.Parallel()
	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	exp := NewExporter("Test", "1.0.0")
	exp.Protocol = "amqp"
	exp.Host = "rabbitmq.example.com:5672"

	doc := exp.Export(reg.Build())

	srv, ok := doc.Servers["production"]
	if !ok {
		t.Fatal("missing production server")
	}
	if srv.Host != "rabbitmq.example.com:5672" {
		t.Errorf("server host = %q", srv.Host)
	}
	if srv.Protocol != "amqp" {
		t.Errorf("server protocol = %q", srv.Protocol)
	}
}

func TestExporter_Export_NoHost(t *testing.T) {
	t.Parallel()
	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	exp := NewExporter("Test", "1.0.0")
	exp.Host = ""

	doc := exp.Export(reg.Build())

	if len(doc.Servers) != 0 {
		t.Errorf("expected no servers, got %d", len(doc.Servers))
	}
}

func TestExporter_Export_MultipleServices(t *testing.T) {
	t.Parallel()
	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc-a", Name: "Service A", Version: "1.0.0"})
	reg.AddService(catalog.Service{ID: "svc-b", Name: "Service B", Version: "1.0.0"})
	reg.AddCommand("svc-a", catalog.Message{
		Kind: catalog.CommandMessage, ID: "DoA", Name: "DoA", Version: "1.0.0",
	})
	reg.AddEvent("svc-b", catalog.Message{
		Kind: catalog.EventMessage, ID: "DoneB", Name: "DoneB", Version: "1.0.0", Direction: catalog.Sends,
	})

	cat := reg.Build()
	doc := NewExporter("Multi", "1.0.0").Export(cat)

	if len(doc.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(doc.Channels))
	}
	if len(doc.Operations) != 2 {
		t.Errorf("expected 2 operations, got %d", len(doc.Operations))
	}
}

func TestExporter_Export_SchemaFromReflection(t *testing.T) {
	t.Parallel()
	type CreateOrder struct {
		OrderID string  `json:"orderId" doc:"Unique order identifier"`
		Amount  float64 `json:"amount" doc:"Total amount"`
	}

	schema := catalog.SchemaFromType[CreateOrder]()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "order-svc", Name: "Order Service", Version: "1.0.0"})
	reg.AddCommand("order-svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "CreateOrder",
		Name:    "CreateOrder",
		Version: "1.0.0",
		Schema:  schema,
	})

	cat := reg.Build()
	doc := NewExporter("Order Service", "1.0.0").Export(cat)

	s, ok := doc.Components.Schemas["CreateOrder"]
	if !ok {
		t.Fatal("missing schema")
	}
	schemaMap, ok := s.(map[string]any)
	if !ok {
		t.Fatalf("schema type = %T", s)
	}
	props, ok := schemaMap["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties type = %T", schemaMap["properties"])
	}
	if _, ok := props["orderId"]; !ok {
		t.Error("missing orderId property in schema")
	}
	if _, ok := props["amount"]; !ok {
		t.Error("missing amount property in schema")
	}
}

func TestDocument_MarshalYAML(t *testing.T) {
	t.Parallel()
	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "DoStuff", Name: "DoStuff", Version: "1.0.0",
	})

	cat := reg.Build()
	doc := NewExporter("Test", "1.0.0").Export(cat)

	b, err := doc.MarshalYAML()
	if err != nil {
		t.Fatal(err)
	}
	output := string(b)
	if !strings.Contains(output, "asyncapi: \"3.0.0\"") && !strings.Contains(output, "asyncapi: 3.0.0") {
		t.Errorf("YAML missing asyncapi version:\n%s", output)
	}
	if !strings.Contains(output, "channels:") {
		t.Errorf("YAML missing channels:\n%s", output)
	}
}

func TestDocument_MarshalJSON(t *testing.T) {
	t.Parallel()
	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	doc := NewExporter("Test", "1.0.0").Export(reg.Build())

	b, err := doc.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	if parsed["asyncapi"] != "3.0.0" {
		t.Errorf("asyncapi = %v, want 3.0.0", parsed["asyncapi"])
	}
}

func TestExporter_Export_NoSchema(t *testing.T) {
	t.Parallel()
	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "NoSchema", Name: "NoSchema", Version: "1.0.0",
	})

	cat := reg.Build()
	doc := NewExporter("Test", "1.0.0").Export(cat)

	s, ok := doc.Components.Schemas["NoSchema"]
	if !ok {
		t.Fatal("missing schema for NoSchema")
	}
	switch v := s.(type) {
	case map[string]any:
		if v["type"] != "object" {
			t.Errorf("fallback schema type = %v, want object", v["type"])
		}
	case map[string]string:
		if v["type"] != "object" {
			t.Errorf("fallback schema type = %v, want object", v["type"])
		}
	default:
		t.Fatalf("schema is unexpected type: %T", s)
	}
}

func TestToDotAddress(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"CreateOrder", "create.order"},
		{"PaymentProcessed", "payment.processed"},
		{"simple", "simple"},
		{"GetProductByID", "get.product.by.i.d"},
	}
	for _, tt := range tests {
		got := toDotAddress(tt.input)
		if got != tt.want {
			t.Errorf("toDotAddress(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
