package asyncapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/cattest"
)

func basicCommand(id string) catalog.Message {
	return catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      id,
		Name:    id,
		Version: "1.0.0",
	}
}

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

	msg, ok := doc.Components.Messages["command.CreateOrder"]
	if !ok {
		t.Fatal("missing CreateOrder message component")
	}

	if msg.ContentType != "application/json" {
		t.Errorf("message content type = %q, want %q", msg.ContentType, "application/json")
	}

	if msg.Payload.Ref != "#/components/schemas/command.CreateOrder" {
		t.Errorf(
			"payload ref = %q, want %q",
			msg.Payload.Ref,
			"#/components/schemas/command.CreateOrder",
		)
	}

	schema, ok := doc.Components.Schemas["command.CreateOrder"]
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

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "payment-svc", "Payment Service", "1.0.0")
	cattest.AddEventWithSummary(
		t,
		reg,
		"payment-svc",
		"PaymentProcessed",
		"PaymentProcessed",
		"1.0.0",
		"Payment was processed",
		catalog.Sends,
	)

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

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "svc", "Service", "1.0.0")
	cattest.AddEventSimple(t, reg, "svc", "OrderCreated", "OrderCreated", "1.0.0", catalog.Receives)

	cat := cattest.Build(t, reg)
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

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "catalog-svc", "Catalog Service", "1.0.0")
	cattest.AddMessageSimple(
		t,
		reg,
		"catalog-svc",
		"GetProduct",
		"GetProduct",
		"1.0.0",
		"Get product details",
		catalog.QueryMessage,
		reg.AddQuery,
	)

	cat := cattest.Build(t, reg)
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

func TestExporter_WithOptions(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	exp := NewExporter(
		"Test", "1.0.0",
		WithServer("staging", "kafka.staging:9092", "kafka"),
		WithDescription("Staging API"),
	)

	doc := exp.Export(reg.Build())

	srv, ok := doc.Servers["staging"]
	if !ok {
		t.Fatal("missing staging server")
	}

	if srv.Host != "kafka.staging:9092" {
		t.Errorf("server host = %q", srv.Host)
	}

	if doc.Info.Description != "Staging API" {
		t.Errorf("description = %q", doc.Info.Description)
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

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "svc-a", "Service A", "1.0.0")
	cattest.AddService(t, reg, "svc-b", "Service B", "1.0.0")
	cattest.AddMessageSimple(
		t,
		reg,
		"svc-a",
		"DoA",
		"DoA",
		"1.0.0",
		"",
		catalog.CommandMessage,
		reg.AddCommand,
	)
	cattest.AddEventSimple(t, reg, "svc-b", "DoneB", "DoneB", "1.0.0", catalog.Sends)

	cat := cattest.Build(t, reg)
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
		OrderID string  `doc:"Unique order identifier" json:"orderId"`
		Amount  float64 `doc:"Total amount"            json:"amount"`
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

	s, ok := doc.Components.Schemas["command.CreateOrder"]
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
	reg.AddCommand("svc", basicCommand("DoStuff"))

	cat := reg.Build()
	doc := NewExporter("Test", "1.0.0").Export(cat)

	b, err := doc.MarshalYAML()
	if err != nil {
		t.Fatal(err)
	}

	output := string(b)
	if !strings.Contains(output, "asyncapi: \"3.0.0\"") &&
		!strings.Contains(output, "asyncapi: 3.0.0") {
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

	err = json.Unmarshal(b, &parsed)
	if err != nil {
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
	reg.AddCommand("svc", basicCommand("NoSchema"))

	cat := reg.Build()
	doc := NewExporter("Test", "1.0.0").Export(cat)

	s, ok := doc.Components.Schemas["command.NoSchema"]
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
		{"GetProductByID", "get.product.by.id"},
		{"XMLParser", "xml.parser"},
		{"Get3DView", "get.3d.view"},
		{"HTTPSConnection", "https.connection"},
		{"lower", "lower"},
		{"", ""},
	}
	for _, tt := range tests {
		got := toDotAddress(tt.input)
		if got != tt.want {
			t.Errorf("toDotAddress(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExporter_Export_Examples(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "svc", "Svc", "1.0.0")
	cattest.AddCommandWithExamples(
		t, reg, "svc", "CreateOrder", "CreateOrder", "1.0.0",
		json.RawMessage(`{"orderId":"abc","amount":42.5}`),
	)

	cat := reg.Build()
	doc := NewExporter("Test", "1.0.0").Export(cat)

	msg, ok := doc.Components.Messages["command.CreateOrder"]
	if !ok {
		t.Fatal("missing CreateOrder message")
	}

	if len(msg.Examples) != 1 {
		t.Fatalf("expected 1 example, got %d", len(msg.Examples))
	}

	exampleJSON, err := json.Marshal(msg.Examples[0].Payload)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(exampleJSON), "orderId") {
		t.Errorf("example payload = %q, want orderId", string(exampleJSON))
	}
}
