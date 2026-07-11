package d2

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
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

	buf.WriteString(
		"  entity: {\n    style: {\n" +
			"      fill: \"#e0f7fa\"\n      stroke: \"#00838f\"\n      stroke-width: 2\n" +
			"      border-radius: 6\n      font-color: \"#006064\"\n    }\n  }\n",
	)

	buf.WriteString(
		"  dataProduct: {\n    style: {\n" +
			"      fill: \"#fff8e1\"\n      stroke: \"#f57f17\"\n      stroke-width: 2\n" +
			"      border-radius: 6\n      font-color: \"#e65100\"\n    }\n  }\n",
	)

	buf.WriteString(
		"  agent: {\n    style: {\n" +
			"      fill: \"#f3e5f5\"\n      stroke: \"#7b1fa2\"\n      stroke-width: 2\n" +
			"      border-radius: 6\n      font-color: \"#4a148c\"\n    }\n  }\n",
	)

	buf.WriteString("}\n\n")
}

func (e *Exporter) writeServices(buf *strings.Builder, cat *catalog.Catalog) {
	for _, svc := range cat.Services {
		serviceDisplayID := sanitizeID(string(svc.ID))

		fmt.Fprintf(buf, "%s: {\n", serviceDisplayID)
		fmt.Fprintf(buf, "  class: service\n")
		fmt.Fprintf(buf, "  label: %q\n  direction: %s\n\n", svc.Name, e.direction)

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
	messageDisplayID := sanitizeID(string(catalog.Key(msg)))

	fmt.Fprintf(buf, "  %s: {\n", messageDisplayID)
	fmt.Fprintf(buf, "    class: %s\n", class)

	if shape != shapeRectangle {
		fmt.Fprintf(buf, "    shape: %s\n", shape)
	}

	label := string(msg.Name)

	if msg.Deprecated {
		label += " [DEPRECATED]"
	}

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
		parts = append(parts, string(msg.Summary))
	}

	if len(msg.Owners) > 0 {
		parts = append(parts, "Owners: "+strings.Join(msg.Owners, ", "))
	}

	if len(msg.Labels) > 0 {
		labelParts := make([]string, 0, len(msg.Labels))

		for k, v := range msg.Labels {
			labelParts = append(labelParts, k+"="+v)
		}

		parts = append(parts, "Labels: "+strings.Join(labelParts, ", "))
	}

	if msg.Schema != nil && len(msg.Schema.Properties) > 0 {
		props := make([]string, 0, len(msg.Schema.Properties))

		for name, p := range msg.Schema.Properties {
			propStr := name + ": " + string(p.Type)

			if p.Description != "" {
				propStr += " — " + p.Description
			}

			props = append(props, propStr)
		}

		parts = append(parts, "Fields: "+strings.Join(props, ", "))
	}

	return strings.Join(parts, "\n")
}

func (e *Exporter) writeEntities(buf *strings.Builder, cat *catalog.Catalog) {
	for _, entity := range cat.Entities {
		id := "entity_" + sanitizeID(string(entity.ID))

		fmt.Fprintf(buf, "%s: {\n", id)
		fmt.Fprintf(buf, "  class: entity\n")

		label := string(entity.Name)
		if entity.AggregateRoot {
			label += " [Aggregate Root]"
		}

		fmt.Fprintf(buf, "  label: %q\n", label)
		buf.WriteString("  shape: cylinder\n")

		tooltip := e.buildEntityTooltip(entity)
		if tooltip != "" {
			fmt.Fprintf(buf, "  tooltip: %q\n", tooltip)
		}

		buf.WriteString("}\n\n")
	}

	e.writeEntityRelationships(buf, cat)
}

func (e *Exporter) buildEntityTooltip(entity catalog.Entity) string {
	var parts []string

	if entity.Summary != "" {
		parts = append(parts, string(entity.Summary))
	}

	if entity.Identifier != "" {
		parts = append(parts, "Identifier: "+entity.Identifier)
	}

	if len(entity.Properties) > 0 {
		propParts := make([]string, 0, len(entity.Properties))
		for _, p := range entity.Properties {
			propStr := string(p.Name) + ": " + p.Type
			if p.Required {
				propStr += " (required)"
			}

			propParts = append(propParts, propStr)
		}

		parts = append(parts, "Properties: "+strings.Join(propParts, ", "))
	}

	return strings.Join(parts, "\n")
}

func (e *Exporter) writeEntityRelationships(buf *strings.Builder, cat *catalog.Catalog) {
	for _, entity := range cat.Entities {
		fromID := "entity_" + sanitizeID(string(entity.ID))

		for _, prop := range entity.Properties {
			if prop.References == "" {
				continue
			}

			toID := "entity_" + sanitizeID(string(prop.References))

			label := prop.RelationType
			if label == "" {
				label = "references"
			}

			fmt.Fprintf(buf, "%s -> %s: %q {\n", fromID, toID, label)
			buf.WriteString("  style: {\n")
			buf.WriteString("    stroke: \"#00838f\"\n")
			buf.WriteString("    stroke-dash: 3\n")
			buf.WriteString("  }\n}\n\n")
		}
	}
}

func (e *Exporter) writeDataProducts(buf *strings.Builder, cat *catalog.Catalog) {
	for _, product := range cat.DataProducts {
		id := "dp_" + sanitizeID(string(product.ID))

		fmt.Fprintf(buf, "%s: {\n", id)
		fmt.Fprintf(buf, "  class: dataProduct\n")
		fmt.Fprintf(buf, "  label: %q\n", product.Name)
		buf.WriteString("  shape: package\n")
		buf.WriteString("}\n\n")

		for _, input := range product.Inputs {
			inputID := sanitizeID(string(input.ID))
			fmt.Fprintf(buf, "%s -> %s: \"feeds\"\n\n", inputID, id)
		}

		for _, output := range product.Outputs {
			outputID := sanitizeID(string(output.ID))
			fmt.Fprintf(buf, "%s -> %s: \"produces\"\n\n", id, outputID)
		}
	}
}

func (e *Exporter) writeAgents(buf *strings.Builder, cat *catalog.Catalog) {
	for _, agent := range cat.Agents {
		id := "agent_" + sanitizeID(string(agent.ID))

		fmt.Fprintf(buf, "%s: {\n", id)
		fmt.Fprintf(buf, "  class: agent\n")

		label := string(agent.Name)
		if agent.Model != nil {
			label += fmt.Sprintf(" (%s/%s)", agent.Model.Provider, agent.Model.Name)
		}

		fmt.Fprintf(buf, "  label: %q\n", label)

		if agent.Summary != "" {
			fmt.Fprintf(buf, "  tooltip: %q\n", agent.Summary)
		}

		buf.WriteString("  shape: step\n")
		buf.WriteString("}\n\n")

		for _, msg := range agent.Sends {
			msgID := sanitizeID(string(msg.ID))
			fmt.Fprintf(buf, "%s -> %s: \"sends\"\n\n", id, msgID)
		}

		for _, msg := range agent.Receives {
			msgID := sanitizeID(string(msg.ID))
			fmt.Fprintf(buf, "%s -> %s: \"receives\"\n\n", msgID, id)
		}
	}
}
