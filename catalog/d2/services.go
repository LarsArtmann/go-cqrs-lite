package d2

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func (e *Exporter) writeHeader(buf *strings.Builder) {
	if e.title != "" {
		fmt.Fprintf(buf, "title: {\n  label: %q\n  near: top-center\n  shape: text\n", e.title)

		buf.WriteString(
			"  style: {\n    font-size: 28\n    bold: true\n    underline: true\n  }\n}\n\n",
		)
	}

	if e.description != "" {
		fmt.Fprintf(
			buf,
			"subtitle: {\n  label: %q\n  near: top-center\n  shape: text\n",
			e.description,
		)

		buf.WriteString(
			"  style: {\n    font-size: 13\n    italic: true\n    font-color: \"#555555\"\n  }\n}\n\n",
		)
	}
}

func (e *Exporter) writeClasses(buf *strings.Builder) {
	buf.WriteString("classes: {\n")

	buf.WriteString(
		"  service: {\n    style: {\n" +
			"      fill: \"#e8f5e9\"\n      stroke: \"#2e7d32\"\n      stroke-width: 2\n" +
			"      border-radius: 8\n      font-color: \"#1b5e20\"\n      bold: true\n    }\n  }\n",
	)

	buf.WriteString(
		"  command: {\n    style: {\n" +
			"      fill: \"#e3f2fd\"\n      stroke: \"#1565c0\"\n      stroke-width: 2\n" +
			"      border-radius: 6\n      font-color: \"#0d47a1\"\n    }\n  }\n",
	)

	buf.WriteString(
		"  event: {\n    style: {\n" +
			"      fill: \"#fce4ec\"\n      stroke: \"#c62828\"\n      stroke-width: 2\n" +
			"      border-radius: 6\n      font-color: \"#b71c1c\"\n    }\n  }\n",
	)

	buf.WriteString(
		"  query: {\n    style: {\n" +
			"      fill: \"#f3e5f5\"\n      stroke: \"#6a1b9a\"\n      stroke-width: 2\n" +
			"      border-radius: 6\n      font-color: \"#4a148c\"\n    }\n  }\n",
	)

	buf.WriteString("}\n\n")
}

func (e *Exporter) writeServices(buf *strings.Builder, cat *catalog.Catalog) {
	for _, svc := range cat.Services {
		svcID := sanitizeID(string(svc.ID))

		fmt.Fprintf(buf, "%s: {\n", svcID)
		fmt.Fprintf(buf, "  class: service\n  label: %q\n", svc.Name)
		buf.WriteString("  direction: down\n\n")

		for _, cmd := range svc.Commands {
			e.writeMessageNode(buf, cmd, "command", shapeRectangle)
		}

		for _, evt := range svc.Events {
			e.writeMessageNode(buf, evt, "event", shapeQueue)
		}

		for _, q := range svc.Queries {
			e.writeMessageNode(buf, q, "query", shapeRectangle)
		}

		e.writeInternalConnections(buf, svc)

		buf.WriteString("}\n\n")
	}
}

func (e *Exporter) writeMessageNode(
	buf *strings.Builder,
	msg catalog.Message,
	class, shape string,
) {
	msgID := sanitizeID(string(catalog.GetID(msg)))

	fmt.Fprintf(buf, "  %s: {\n", msgID)
	fmt.Fprintf(buf, "    class: %s\n", class)

	if shape != shapeRectangle {
		fmt.Fprintf(buf, "    shape: %s\n", shape)
	}

	label := msg.Name

	if msg.Version != "" {
		label += fmt.Sprintf(" (v%s)", msg.Version)
	}

	fmt.Fprintf(buf, "    label: %q\n", label)

	tooltip := e.buildTooltip(msg)

	if tooltip != "" {
		fmt.Fprintf(buf, "    tooltip: %q\n", tooltip)
	}

	buf.WriteString("  }\n")
}

func (e *Exporter) buildTooltip(msg catalog.Message) string {
	var parts []string

	if msg.Summary != "" {
		parts = append(parts, msg.Summary)
	}

	if msg.Schema != nil && len(msg.Schema.Properties) > 0 {
		props := make([]string, 0, len(msg.Schema.Properties))

		for name, p := range msg.Schema.Properties {
			propStr := name + ": " + p.Type

			if p.Description != "" {
				propStr += " — " + p.Description
			}

			props = append(props, propStr)
		}

		parts = append(parts, "Fields: "+strings.Join(props, ", "))
	}

	return strings.Join(parts, "\n")
}
