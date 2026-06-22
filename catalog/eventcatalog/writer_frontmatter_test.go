package eventcatalog

import (
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func TestBuildMessageFrontmatter_FullFields(t *testing.T) {
	t.Parallel()

	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	msg := catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "CreateOrder",
		Name:    "Create Order",
		Version: "1.0.0",
		Summary: "Creates a new order",
		Labels: map[string]string{
			"domain": "ordering",
			"team":   "order-team",
		},
		Deprecated: true,
		Changelog: []catalog.Change{
			{Version: "1.0.0", Summary: "Initial version", Date: &date},
			{Version: "0.9.0", Summary: "Beta release"},
		},
		Producers: []catalog.ServiceID{"order-svc"},
		Consumers: []catalog.ServiceID{"payment-svc", "inventory-svc"},
		Operation: &catalog.Operation{
			Method:      "POST",
			Path:        "/orders",
			StatusCodes: []string{"201", "400", "409"},
		},
		Badges: []catalog.Badge{
			{Content: "stable", BackgroundColor: "green", TextColor: "white", Icon: "check"},
		},
		Repository: &catalog.Repository{
			Language: "go",
			URL:      "https://github.com/example/order-svc",
		},
		Schema: &catalog.Schema{Type: catalog.TypeObject},
	}

	md := buildMessageFrontmatter("CreateOrder", msg)
	out := md.String()

	frontmatterAssertContains(t, out, "deprecated: true")
	frontmatterAssertContains(t, out, "labels:")
	frontmatterAssertContains(t, out, "domain: \"ordering\"")
	frontmatterAssertContains(t, out, "team: \"order-team\"")
	frontmatterAssertContains(t, out, "changelog:")
	frontmatterAssertContains(t, out, "version: \"1.0.0\"")
	frontmatterAssertContains(t, out, "date: \"2026-01-15\"")
	frontmatterAssertContains(t, out, "version: \"0.9.0\"")
	frontmatterAssertContains(t, out, "producers:")
	frontmatterAssertContains(t, out, "- id: order-svc")
	frontmatterAssertContains(t, out, "consumers:")
	frontmatterAssertContains(t, out, "- id: payment-svc")
	frontmatterAssertContains(t, out, "- id: inventory-svc")
	frontmatterAssertContains(t, out, "operation:")
	frontmatterAssertContains(t, out, "method: POST")
	frontmatterAssertContains(t, out, "path: \"/orders\"")
	frontmatterAssertContains(t, out, "statusCodes:")
	frontmatterAssertContains(t, out, "- \"201\"")
	frontmatterAssertContains(t, out, "badges:")
	frontmatterAssertContains(t, out, "content: \"stable\"")
	frontmatterAssertContains(t, out, "backgroundColor: green")
	frontmatterAssertContains(t, out, "textColor: white")
	frontmatterAssertContains(t, out, "icon: check")
	frontmatterAssertContains(t, out, "repository:")
	frontmatterAssertContains(t, out, "language: \"go\"")
	frontmatterAssertContains(t, out, "url: \"https://github.com/example/order-svc\"")
	frontmatterAssertContains(t, out, "schemaPath: schemas/schema.json")
}

func TestBuildMessageFrontmatter_Minimal(t *testing.T) {
	t.Parallel()

	msg := catalog.Message{
		Kind:    catalog.EventMessage,
		ID:      "OrderCreated",
		Name:    "Order Created",
		Version: "1.0.0",
	}

	md := buildMessageFrontmatter("OrderCreated", msg)
	out := md.String()

	frontmatterAssertNotContains(t, out, "deprecated")
	frontmatterAssertNotContains(t, out, "labels:")
	frontmatterAssertNotContains(t, out, "changelog:")
	frontmatterAssertNotContains(t, out, "operation:")
	frontmatterAssertNotContains(t, out, "badges:")
	frontmatterAssertNotContains(t, out, "repository:")
	frontmatterAssertNotContains(t, out, "schemaPath")
}

func TestWriteServiceFrontmatter_FullFields(t *testing.T) {
	t.Parallel()

	svc := catalog.Service{
		ID:      "order-svc",
		Name:    "Order Service",
		Version: "1.0.0",
		Summary: "Manages orders",
		Owners:  []string{"order-team"},
		Specifications: []catalog.Specification{
			{Type: "openapi", Path: "/openapi.yaml", Name: "Public API"},
			{Type: "asyncapi", Path: "/asyncapi.yaml"},
		},
		Attachments: []catalog.Attachment{
			{URL: "https://wiki.example/adr-001", Title: "ADR-001", Type: "adr"},
			{URL: "https://wiki.example/runbook", Title: "Runbook"},
			{URL: "https://wiki.example/diagram"},
		},
		Flows: []catalog.FlowID{"create-order"},
	}

	md := newFrontmatterWriter()
	md.addField("id", string(svc.ID))
	writeSpecifications(md, svc.Specifications)
	writeAttachments(md, svc.Attachments)
	writeIDListField(md, "flows", svc.Flows)
	md.finish(string(svc.Name), string(svc.Summary))

	out := md.String()

	frontmatterAssertContains(t, out, "specifications:")
	frontmatterAssertContains(t, out, "type: openapi")
	frontmatterAssertContains(t, out, "path: \"/openapi.yaml\"")
	frontmatterAssertContains(t, out, "name: \"Public API\"")
	frontmatterAssertContains(t, out, "type: asyncapi")
	frontmatterAssertContains(t, out, "attachments:")
	frontmatterAssertContains(t, out, "url: \"https://wiki.example/adr-001\"")
	frontmatterAssertContains(t, out, "title: \"ADR-001\"")
	frontmatterAssertContains(t, out, "type: \"adr\"")
	frontmatterAssertContains(t, out, "url: \"https://wiki.example/runbook\"")
	frontmatterAssertContains(t, out, "url: \"https://wiki.example/diagram\"")
	frontmatterAssertContains(t, out, "flows:")
	frontmatterAssertContains(t, out, "- id: create-order")
}

func TestWriteMessagePointers(t *testing.T) {
	t.Parallel()

	md := newFrontmatterWriter()
	writeMessagePointers(md, "sends", []catalog.Ref{
		{ID: "OrderCreated", Version: "1.0.0"},
		{ID: "OrderShipped"},
	})
	out := md.String()

	frontmatterAssertContains(t, out, "sends:")
	frontmatterAssertContains(t, out, "- id: OrderCreated")
	frontmatterAssertContains(t, out, "version: \"1.0.0\"")
	frontmatterAssertContains(t, out, "- id: OrderShipped")
	frontmatterAssertNotContains(t, out, "version: \"\"")
}

func TestWriteMessagePointers_Empty(t *testing.T) {
	t.Parallel()

	md := newFrontmatterWriter()
	writeMessagePointers(md, "receives", nil)

	if md.String() != "---\n" {
		t.Errorf("expected empty output, got %q", md.String())
	}
}

func frontmatterAssertContains(tb testing.TB, s, substr string) {
	tb.Helper()

	if !strings.Contains(s, substr) {
		tb.Errorf("expected output to contain %q, got:\n%s", substr, s)
	}
}

func frontmatterAssertNotContains(tb testing.TB, s, substr string) {
	tb.Helper()

	if strings.Contains(s, substr) {
		tb.Errorf("expected output NOT to contain %q, got:\n%s", substr, s)
	}
}
