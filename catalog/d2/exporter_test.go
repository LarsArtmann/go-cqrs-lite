package d2

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/internal/cattest"
)

func assertContains(t *testing.T, output, substr, msg string) {
	t.Helper()

	if !strings.Contains(output, substr) {
		t.Error(msg)
	}
}

func assertNotContains(t *testing.T, output, substr, msg string) {
	t.Helper()

	if strings.Contains(output, substr) {
		t.Error(msg)
	}
}

func TestExporter_Export_EmptyCatalog(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	exp := NewExporter("Empty", "1.0.0")
	output := exp.Export(reg.Build())

	assertContains(t, output, "title:", "expected title in output")
	assertContains(t, output, "classes:", "expected classes in output")
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
	})

	cat := reg.Build()
	exp := NewExporter("Order Service", "1.0.0")
	output := exp.Export(cat)

	assertContains(t, output, "order_svc: {", "expected service node")
	assertContains(t, output, "createorder:", "expected command node")
	assertContains(t, output, "class: command", "expected command class")
	assertContains(t, output, "CreateOrder", "expected command name in label")
	assertContains(t, output, "receives", "expected receives connection")
}

func TestExporter_Export_Event(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("payment-svc"), "Payment Service", "1.0.0")
	cattest.AddEventWithSummary(
		t, reg, catalog.ServiceID("payment-svc"),
		catalog.MessageID("PaymentProcessed"), "PaymentProcessed", "1.0.0",
		"Payment was processed", catalog.Sends,
	)

	cat := reg.Build()
	exp := NewExporter("Payment Service", "1.0.0")
	output := exp.Export(cat)

	assertContains(t, output, "paymentprocessed:", "expected event node")
	assertContains(t, output, "class: event", "expected event class")
	assertContains(t, output, "publishes", "expected publishes connection for Sends event")
	assertContains(t, output, "shape: queue", "expected queue shape for events")
}

func TestExporter_Export_EventReceive(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("svc"), "Service", "1.0.0")
	cattest.AddEvent(t, reg, catalog.ServiceID("svc"), catalog.MessageID("OrderCreated"), "OrderCreated", "1.0.0", catalog.Receives)

	cat := cattest.Build(t, reg)
	output := NewExporter("Svc", "1.0.0").Export(cat)

	assertContains(t, output, "receives", "expected receives connection for Receives event")
}

func TestExporter_Export_Query(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddServiceWithQuery(
		t,
		reg,
		catalog.ServiceID("catalog-svc"),
		catalog.MessageID("GetProduct"),
		"GetProduct",
		"1.0.0",
		"Get product details",
	)

	cat := cattest.Build(t, reg)
	output := NewExporter("Catalog Service", "1.0.0").Export(cat)

	assertContains(t, output, "getproduct:", "expected query node")
	assertContains(t, output, "class: query", "expected query class")
	assertContains(t, output, "handles", "expected handles connection")
}

func TestExporter_Export_MultipleServices(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("svc-a"), "Service A", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("svc-b"), "Service B", "1.0.0")
	cattest.AddServiceWithCommand(
		t,
		reg,
		catalog.ServiceID("svc-a"),
		catalog.MessageID("DoA"),
		"DoA",
		"1.0.0",
		"",
	)
	cattest.AddEvent(t, reg, catalog.ServiceID("svc-b"), catalog.MessageID("DoneB"), "DoneB", "1.0.0", catalog.Sends)

	cat := cattest.Build(t, reg)
	output := NewExporter("Multi", "1.0.0").Export(cat)

	assertContains(t, output, "svc_a: {", "expected svc_a node")
	assertContains(t, output, "svc_b: {", "expected svc_b node")
}

func TestExporter_Export_DomainGrouping(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("order-svc"), "Order Service", "1.0.0")
	cattest.AddDomain(
		t,
		reg,
		catalog.DomainID("ecommerce"),
		"E-Commerce",
		"1.0.0",
		"Online store",
		[]catalog.ServiceID{"order-svc"},
	)

	cat := reg.Build()
	output := NewExporter("Test", "1.0.0").Export(cat)

	assertContains(t, output, "domain_ecommerce:", "expected domain node")
	assertContains(t, output, "E-Commerce", "expected domain name")
	assertContains(t, output, "contains", "expected domain-service connection")
}

func TestExporter_Export_WithDescription(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	exp := NewExporter("Test", "1.0.0", WithDescription("A test description"))
	output := exp.Export(reg.Build())

	assertContains(t, output, "A test description", "expected description in subtitle")
}

func TestExporter_Export_SchemaTooltip(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "CreateOrder",
		Name:    "CreateOrder",
		Version: "1.0.0",
		Summary: "Create a new order",
		Schema: &catalog.Schema{
			Type: "object",
			Properties: map[string]catalog.Property{
				"orderId": {Type: "string", Description: "Unique order ID"},
				"amount":  {Type: "number"},
			},
		},
	})

	cat := reg.Build()
	output := NewExporter("Test", "1.0.0").Export(cat)

	assertContains(t, output, "Create a new order", "expected summary in tooltip")
	assertContains(t, output, "orderId: string", "expected schema field in tooltip")
	assertContains(t, output, "Unique order ID", "expected field description in tooltip")
}

func TestExporter_Export_CrossServiceEventFlow(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("order-svc"), "Order Service", "1.0.0")
	cattest.AddService(
		t,
		reg,
		catalog.ServiceID("notification-svc"),
		"Notification Service",
		"1.0.0",
	)

	cattest.AddEvent(t, reg, catalog.ServiceID("order-svc"), catalog.MessageID("OrderCreated"), "OrderCreated", "1.0.0", catalog.Sends)
	cattest.AddEvent(
		t,
		reg,
		catalog.ServiceID("notification-svc"),
		catalog.MessageID("OrderCreated"),
		"OrderCreated",
		"1.0.0",
		catalog.Receives,
	)

	cat := cattest.Build(t, reg)
	output := NewExporter("E-Commerce", "1.0.0").Export(cat)

	assertContains(
		t,
		output,
		"order_svc.ordercreated -> notification_svc.ordercreated",
		"expected cross-service event connection from publisher to receiver",
	)
	assertContains(t, output, "OrderCreated", "expected event name as connection label")
	assertContains(
		t,
		output,
		"animated: true",
		"expected animated connection for cross-service events",
	)
}

func TestExporter_Export_CrossService_NoSelfConnection(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("svc"), "Service", "1.0.0")
	cattest.AddEvent(t, reg, catalog.ServiceID("svc"), catalog.MessageID("SelfEvent"), "SelfEvent", "1.0.0", catalog.Sends)
	cattest.AddEvent(t, reg, catalog.ServiceID("svc"), catalog.MessageID("SelfEvent"), "SelfEvent", "1.0.0", catalog.Receives)

	cat := cattest.Build(t, reg)
	output := NewExporter("Test", "1.0.0").Export(cat)

	assertNotContains(
		t,
		output,
		"svc.selfevent -> svc.selfevent",
		"should not create cross-service connection for same service",
	)
}

func TestExporter_Export_CrossService_NoConnectionWithoutMatch(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("svc-a"), "Service A", "1.0.0")
	cattest.AddEvent(t, reg, catalog.ServiceID("svc-a"), catalog.MessageID("OrphanEvent"), "OrphanEvent", "1.0.0", catalog.Sends)

	cat := cattest.Build(t, reg)
	output := NewExporter("Test", "1.0.0").Export(cat)

	assertNotContains(
		t,
		output,
		"animated: true",
		"should not create cross-service connection when no receiver",
	)
}

func TestExporter_Export_VersionInLabel(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "Create",
		Name:    "Create",
		Version: "2.0.0",
	})

	cat := reg.Build()
	output := NewExporter("Test", "1.0.0").Export(cat)

	assertContains(t, output, "v2.0.0", "expected version in message label")
}

func TestSanitizeID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"CreateOrder", "createorder"},
		{"order-svc", "order_svc"},
		{"my.service", "my_service"},
		{"some/path", "some_path"},
		{"with spaces", "with_spaces"},
		{"mixed-thing.here/now", "mixed_thing_here_now"},
	}

	for _, tt := range tests {
		got := sanitizeID(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExporter_Export_WithDirection(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddServiceWithCommand(
		t,
		reg,
		catalog.ServiceID("svc"),
		catalog.MessageID("DoWork"),
		"DoWork",
		"1.0.0",
		"",
	)

	cat := cattest.Build(t, reg)
	output := NewExporter("Test", "1.0.0", WithDirection("up")).Export(cat)

	if !strings.Contains(output, "direction: up") {
		t.Error("expected 'direction: up' in output")
	}

	if strings.Contains(output, "direction: down") {
		t.Error("should not contain 'direction: down' when WithDirection(\"up\") is set")
	}
}

func TestExporter_Export_ValidD2(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddServiceWithCommand(
		t,
		reg,
		catalog.ServiceID("svc"),
		catalog.MessageID("DoWork"),
		"DoWork",
		"1.0.0",
		"Work command",
	)
	cattest.AddEvent(t, reg, catalog.ServiceID("svc"), catalog.MessageID("WorkDone"), "WorkDone", "1.0.0", catalog.Sends)
	cattest.AddMessageSimple(
		t,
		reg,
		catalog.ServiceID("svc"),
		catalog.MessageID("GetStatus"),
		"GetStatus",
		"1.0.0",
		"Get status",
		catalog.QueryMessage, reg.AddQuery,
	)

	cat := cattest.Build(t, reg)
	output := NewExporter("Test Service", "1.0.0").Export(cat)

	assertNotContains(t, output, "nil", "output should not contain 'nil'")

	for _, bad := range []string{"[]", "{}", "func"} {
		if strings.Contains(output, bad) {
			t.Errorf("output should not contain Go literal %q", bad)
		}
	}
}
