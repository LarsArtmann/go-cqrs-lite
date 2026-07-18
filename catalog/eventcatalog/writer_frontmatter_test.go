package eventcatalog

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func TestRenderMDX_MessageFullFields(t *testing.T) {
	t.Parallel()

	msg := catalog.Message{
		Kind:      catalog.CommandMessage,
		ID:        "CreateOrder",
		Name:      "Create Order",
		Version:   "1.0.0",
		Summary:   "Creates a new order",
		Direction: catalog.Receives,
		Labels:    map[string]string{"domain": "ordering", "team": "order-team"},
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

	fm := messageFM{
		ID:         string(catalog.Key(msg)),
		Name:       string(msg.Name),
		Version:    string(msg.Version),
		Summary:    string(msg.Summary),
		Owners:     msg.Owners,
		Labels:     msg.Labels,
		Changelog:  toChangelog(msg.Changelog),
		Producers:  toPointers(msg.Producers),
		Consumers:  toPointers(msg.Consumers),
		Operation:  toOperation(msg.Operation),
		Badges:     toBadges(msg.Badges),
		Repository: toRepository(msg.Repository),
	}
	if msg.Schema != nil {
		fm.SchemaPath = "schemas/schema.json"
	}

	out, err := renderMDX(fm, string(msg.Name), string(msg.Summary), false)
	if err != nil {
		t.Fatal(err)
	}

	frontmatterAssertContains(t, out, "id: CreateOrder")
	frontmatterAssertContains(t, out, "name: Create Order")
	frontmatterAssertContains(t, out, "version: 1.0.0")
	frontmatterAssertContains(t, out, "summary: Creates a new order")
	frontmatterAssertContains(t, out, "producers:")
	frontmatterAssertContains(t, out, "id: order-svc")
	frontmatterAssertContains(t, out, "consumers:")
	frontmatterAssertContains(t, out, "id: payment-svc")
	frontmatterAssertContains(t, out, "id: inventory-svc")
	frontmatterAssertContains(t, out, "operation:")
	frontmatterAssertContains(t, out, "method: POST")
	frontmatterAssertContains(t, out, "path: /orders")
	frontmatterAssertContains(t, out, "statusCodes:")
	frontmatterAssertContains(t, out, "badges:")
	frontmatterAssertContains(t, out, "content: stable")
	frontmatterAssertContains(t, out, "backgroundColor: green")
	frontmatterAssertContains(t, out, "textColor: white")
	frontmatterAssertContains(t, out, "icon: check")
	frontmatterAssertContains(t, out, "repository:")
	frontmatterAssertContains(t, out, "schemaPath: schemas/schema.json")

	if !strings.HasPrefix(out, "---\n") {
		t.Error("MDX should start with ---")
	}

	if !strings.Contains(out, "---\n\n# Create Order") {
		t.Error("MDX should have heading after frontmatter")
	}
}

func TestRenderMDX_Minimal(t *testing.T) {
	t.Parallel()

	fm := messageFM{
		ID:      "PlainEvent",
		Name:    "PlainEvent",
		Version: "1.0.0",
	}

	out, err := renderMDX(fm, "PlainEvent", "", false)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(out, "summary:") {
		t.Errorf("message without summary should not have summary field")
	}

	if !strings.Contains(out, "# PlainEvent") {
		t.Errorf("message should have title heading")
	}
}

func frontmatterAssertContains(t *testing.T, content string, expected ...string) {
	t.Helper()

	for _, exp := range expected {
		if !strings.Contains(content, exp) {
			t.Errorf("expected output to contain %q, got:\n%s", exp, content)
		}
	}
}

func TestRenderMDX_WithResponses(t *testing.T) {
	t.Parallel()

	msg := catalog.Message{
		Kind: catalog.CommandMessage,
		ID:   "CreateUser",
		Name: "Create User",
		Responses: []catalog.ResponseSpec{
			{
				StatusCode:  "201",
				Description: "User created",
				Schema:      &catalog.Schema{Type: catalog.TypeObject},
			},
			{StatusCode: "400", Description: "Bad request"},
		},
	}

	fm := messageFM{
		ID:        string(catalog.Key(msg)),
		Name:      string(msg.Name),
		Version:   "1.0.0",
		Responses: toResponses(msg.Responses),
	}

	out, err := renderMDX(fm, string(msg.Name), "", false)
	if err != nil {
		t.Fatal(err)
	}

	frontmatterAssertContains(
		t, out,
		"responses:",
		"statusCode: \"201\"",
		"description: User created",
		"statusCode: \"400\"",
		"description: Bad request",
	)
}
