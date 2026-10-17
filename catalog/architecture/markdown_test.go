package architecture

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func TestRenderMarkdownBlocks(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "heading", in: "## Value chain", want: "<h2>Value chain</h2>"},
		{name: "paragraph", in: "Two words.", want: "<p>Two words.</p>"},
		{name: "bold", in: "**events**", want: "<p><strong>events</strong></p>"},
		{name: "italic", in: "*events*", want: "<p><em>events</em></p>"},
		{name: "code span", in: "`order-id`", want: "<p><code>order-id</code></p>"},
		{
			name: "bullet list",
			in:   "- replayable\n- auditable",
			want: "<ul><li>replayable</li><li>auditable</li></ul>",
		},
		{
			name: "ordered list",
			in:   "1. first\n2. second",
			want: "<ol><li>first</li><li>second</li></ol>",
		},
		{
			name: "fence keeps its body verbatim",
			in:   "```go\nif a < b {\n}\n```",
			want: `<pre><code class="lang-go">if a &lt; b {` + "\n" + `}</code></pre>`,
		},
		{
			name: "link",
			in:   "[docs](https://example.com/x)",
			want: `<a href="https://example.com/x" rel="noreferrer noopener" target="_blank">docs</a>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := string(renderMarkdown(tc.in, nil))
			if !strings.Contains(got, tc.want) {
				t.Errorf("renderMarkdown(%q) = %q, want it to contain %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRenderMarkdownEscapesAndRejectsRawHTML(t *testing.T) {
	got := string(renderMarkdown("<img src=x onerror=alert(1)>", nil))

	if strings.Contains(got, "<img") {
		t.Errorf("raw HTML survived markdown rendering: %q", got)
	}

	if !strings.Contains(got, "&lt;img") {
		t.Errorf("expected the HTML to be escaped, got %q", got)
	}
}

func TestRenderMarkdownResourceReference(t *testing.T) {
	link := func(kind, id string) resourceRef {
		return resourceRef{anchor: kind + "-" + id, name: "Order", tag: "EVT", class: "evt"}
	}

	got := string(renderMarkdown("see [[event|order-created]]", link))

	if !strings.Contains(got, `href="#event-order-created"`) {
		t.Errorf("resource reference was not resolved: %q", got)
	}
}

func TestRenderMarkdownWithoutLinkerKeepsTheText(t *testing.T) {
	got := string(renderMarkdown("see [[event|order-created]]", nil))

	if !strings.Contains(got, "event|order-created") {
		t.Errorf("expected the reference to degrade to text, got %q", got)
	}
}

func TestTypeScriptInterface(t *testing.T) {
	schema := catalog.SchemaFromType[struct {
		OrderID string `json:"order_id" doc:"The order"`
		Amount  int    `json:"amount,omitempty" doc:"Total"`
	}]()

	got := typeScriptInterface("order.created", schema)

	for _, want := range []string{
		"export interface OrderCreated {",
		"order_id: string;",
		"amount?: number;",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("interface is missing %q:\n%s", want, got)
		}
	}
}

func TestExamplePayloadFallsBackToSynthesis(t *testing.T) {
	schema := catalog.SchemaFromType[struct {
		OrderID string `json:"order_id" doc:"The order"`
	}]()

	got := examplePayload(schema, nil)

	if !strings.Contains(got, `"order_id": "string"`) {
		t.Errorf("synthesised payload = %q", got)
	}
}

func TestSafeHrefBlocksScriptURLs(t *testing.T) {
	cases := map[string]string{
		"https://example.com": "https://example.com",
		"mailto:a@b.c":        "mailto:a@b.c",
		"javascript:alert(1)": "#",
		"  JavaScript:alert(1)": "#",
		"data:text/html,x":    "#",
	}

	for in, want := range cases {
		if got := safeHref(in); got != want {
			t.Errorf("safeHref(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugKeepsIdentifiersAddressable(t *testing.T) {
	cases := map[string]string{
		"order.created": "order-created",
		"Order Service": "order-service",
		"a_b-c":         "a_b-c",
	}

	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAnchorDistinguishesKindsWithTheSameID(t *testing.T) {
	infos := kindInfos()

	command := infos[kindCommand].anchor("review")
	event := infos[kindEvent].anchor("review")

	if command == event {
		t.Fatalf("command and event anchors collide: %q", command)
	}
}
