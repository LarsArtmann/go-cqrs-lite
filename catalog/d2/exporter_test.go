package d2

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/cattest"
)

func TestExporter_Export_EmptyCatalog(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	exp := NewExporter("Empty", "1.0.0")
	output := exp.Export(reg.Build())

	if !strings.Contains(output, "title:") {
		t.Error("expected title in output")
	}

	if !strings.Contains(output, "classes:") {
		t.Error("expected classes in output")
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
	})

	cat := reg.Build()
	exp := NewExporter("Order Service", "1.0.0")
	output := exp.Export(cat)

	if !strings.Contains(output, "order_svc: {") {
		t.Error("expected service node")
	}

	if !strings.Contains(output, "createorder:") {
		t.Error("expected command node")
	}

	if !strings.Contains(output, "class: command") {
		t.Error("expected command class")
	}

	if !strings.Contains(output, "CreateOrder") {
		t.Error("expected command name in label")
	}

	if !strings.Contains(output, "receives") {
		t.Error("expected receives connection")
	}
}

func TestExporter_Export_Event(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "payment-svc", "Payment Service", "1.0.0")
	cattest.AddEventWithSummary(
		t, reg, "payment-svc",
		"PaymentProcessed", "PaymentProcessed", "1.0.0",
		"Payment was processed", catalog.Sends,
	)

	cat := reg.Build()
	exp := NewExporter("Payment Service", "1.0.0")
	output := exp.Export(cat)

	if !strings.Contains(output, "paymentprocessed:") {
		t.Error("expected event node")
	}

	if !strings.Contains(output, "class: event") {
		t.Error("expected event class")
	}

	if !strings.Contains(output, "publishes") {
		t.Error("expected publishes connection for Sends event")
	}

	if !strings.Contains(output, "shape: queue") {
		t.Error("expected queue shape for events")
	}
}

func TestExporter_Export_EventReceive(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "svc", "Service", "1.0.0")
	cattest.AddEventSimple(t, reg, "svc", "OrderCreated", "OrderCreated", "1.0.0", catalog.Receives)

	cat := cattest.Build(t, reg)
	output := NewExporter("Svc", "1.0.0").Export(cat)

	if !strings.Contains(output, "receives") {
		t.Error("expected receives connection for Receives event")
	}
}

func TestExporter_Export_Query(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "catalog-svc", "Catalog Service", "1.0.0")
	cattest.AddMessageSimple(
		t, reg, "catalog-svc",
		"GetProduct", "GetProduct", "1.0.0", "Get product details",
		catalog.QueryMessage, reg.AddQuery,
	)

	cat := cattest.Build(t, reg)
	output := NewExporter("Catalog Service", "1.0.0").Export(cat)

	if !strings.Contains(output, "getproduct:") {
		t.Error("expected query node")
	}

	if !strings.Contains(output, "class: query") {
		t.Error("expected query class")
	}

	if !strings.Contains(output, "handles") {
		t.Error("expected handles connection")
	}
}

func TestExporter_Export_MultipleServices(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "svc-a", "Service A", "1.0.0")
	cattest.AddService(t, reg, "svc-b", "Service B", "1.0.0")
	cattest.AddMessageSimple(
		t, reg, "svc-a", "DoA", "DoA", "1.0.0", "",
		catalog.CommandMessage, reg.AddCommand,
	)
	cattest.AddEventSimple(t, reg, "svc-b", "DoneB", "DoneB", "1.0.0", catalog.Sends)

	cat := cattest.Build(t, reg)
	output := NewExporter("Multi", "1.0.0").Export(cat)

	if !strings.Contains(output, "svc_a: {") {
		t.Error("expected svc_a node")
	}

	if !strings.Contains(output, "svc_b: {") {
		t.Error("expected svc_b node")
	}
}

func TestExporter_Export_DomainGrouping(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "order-svc", "Order Service", "1.0.0")
	cattest.AddDomain(t, reg, "ecommerce", "E-Commerce", "1.0.0", "Online store", []string{"order-svc"})

	cat := reg.Build()
	output := NewExporter("Test", "1.0.0").Export(cat)

	if !strings.Contains(output, "domain_ecommerce:") {
		t.Error("expected domain node")
	}

	if !strings.Contains(output, "E-Commerce") {
		t.Error("expected domain name")
	}

	if !strings.Contains(output, "contains") {
		t.Error("expected domain-service connection")
	}
}

func TestExporter_Export_WithDescription(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	exp := NewExporter("Test", "1.0.0", WithDescription("A test description"))
	output := exp.Export(reg.Build())

	if !strings.Contains(output, "A test description") {
		t.Error("expected description in subtitle")
	}
}

func TestExporter_Export_SchemaTooltip(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "DoStuff",
		Name:    "DoStuff",
		Version: "1.0.0",
		Summary: "Does important stuff",
	})

	cat := reg.Build()
	output := NewExporter("Test", "1.0.0").Export(cat)

	if !strings.Contains(output, "Does important stuff") {
		t.Error("expected summary as tooltip")
	}
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

	if !strings.Contains(output, "v2.0.0") {
		t.Error("expected version in message label")
	}
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

func TestExporter_Export_ValidD2(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "svc", "Service", "1.0.0")
	cattest.AddMessageSimple(
		t, reg, "svc", "DoWork", "DoWork", "1.0.0", "Work command",
		catalog.CommandMessage, reg.AddCommand,
	)
	cattest.AddEventSimple(t, reg, "svc", "WorkDone", "WorkDone", "1.0.0", catalog.Sends)
	cattest.AddQuerySimple(t, reg, "svc", "GetStatus", "GetStatus", "1.0.0", "Get status")

	cat := cattest.Build(t, reg)
	output := NewExporter("Test Service", "1.0.0").Export(cat)

	if strings.Contains(output, "nil") {
		t.Error("output should not contain 'nil'")
	}

	for _, bad := range []string{"[]", "{}", "func"} {
		if strings.Contains(output, bad) {
			t.Errorf("output should not contain Go literal %q", bad)
		}
	}
}
