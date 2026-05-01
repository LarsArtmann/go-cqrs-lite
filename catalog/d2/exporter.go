package d2

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

const (
	shapeRectangle = "rectangle"
	shapeQueue     = "queue"
)

// Exporter generates D2 diagram output from a Catalog.
type Exporter struct {
	Title       string
	Version     string
	Description string
	Direction   string
}

// Option configures an Exporter.
type Option func(*Exporter)

// WithDescription sets the diagram subtitle.
func WithDescription(desc string) Option {
	return func(e *Exporter) {
		e.Description = desc
	}
}

// WithDirection sets the diagram layout direction.
func WithDirection(dir string) Option {
	return func(e *Exporter) {
		e.Direction = dir
	}
}

func NewExporter(title, version string, opts ...Option) *Exporter {
	e := &Exporter{ //nolint:exhaustruct // Description is optional, filled by WithDescription
		Title:     title,
		Version:   version,
		Direction: "down",
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

func (e *Exporter) Export(cat *catalog.Catalog) string {
	var buf strings.Builder

	e.writeHeader(&buf)
	e.writeClasses(&buf)
	e.writeServices(&buf, cat)
	e.writeCrossServiceConnections(&buf, cat)

	if len(cat.Domains) > 0 {
		e.writeDomains(&buf, cat)
	}

	return buf.String()
}

func (e *Exporter) writeHeader(buf *strings.Builder) {
	if e.Title != "" {
		fmt.Fprintf(buf, "title: {\n  label: %q\n  near: top-center\n  shape: text\n", e.Title)

		buf.WriteString(
			"  style: {\n    font-size: 28\n    bold: true\n    underline: true\n  }\n}\n\n",
		)
	}

	if e.Description != "" {
		fmt.Fprintf(
			buf,
			"subtitle: {\n  label: %q\n  near: top-center\n  shape: text\n",
			e.Description,
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
		svcID := sanitizeID(svc.ID)

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
	msgID := sanitizeID(catalog.MessageID(msg))

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

func (e *Exporter) writeInternalConnections(buf *strings.Builder, svc catalog.Service) {
	if len(svc.Commands) == 0 && len(svc.Events) == 0 && len(svc.Queries) == 0 {
		return
	}

	svcID := sanitizeID(svc.ID)

	for _, cmd := range svc.Commands {
		cmdID := sanitizeID(catalog.MessageID(cmd))

		fmt.Fprintf(buf, "  %s -> %s.%s: \"receives\"\n", svcID, svcID, cmdID)
	}

	for _, evt := range svc.Events {
		evtID := sanitizeID(catalog.MessageID(evt))

		action := "publishes"

		switch evt.Direction {
		case catalog.Receives:
			action = "receives"
		case catalog.Sends:
			action = "publishes"
		}

		fmt.Fprintf(buf, "  %s.%s -> %s: %q\n", svcID, evtID, svcID, action)
	}

	for _, q := range svc.Queries {
		qID := sanitizeID(catalog.MessageID(q))

		fmt.Fprintf(buf, "  %s -> %s.%s: \"handles\"\n", svcID, svcID, qID)
	}

	buf.WriteString("\n")
}

func (e *Exporter) writeCrossServiceConnections(b *strings.Builder, cat *catalog.Catalog) {
	type eventOwner struct {
		svcID string
		evtID string
	}

	publishers := make(map[string][]eventOwner)
	receivers := make(map[string][]eventOwner)

	for _, svc := range cat.Services {
		svcID := sanitizeID(svc.ID)
		for _, evt := range svc.Events {
			evtID := catalog.MessageID(evt)
			owner := eventOwner{svcID: svcID, evtID: sanitizeID(evtID)}

			switch evt.Direction {
			case catalog.Sends:
				publishers[evtID] = append(publishers[evtID], owner)
			case catalog.Receives:
				receivers[evtID] = append(receivers[evtID], owner)
			}
		}
	}

	drawn := 0

	for evtID, pubs := range publishers {
		recvs, ok := receivers[evtID]
		if !ok {
			continue
		}

		for _, pub := range pubs {
			for _, recv := range recvs {
				if pub.svcID == recv.svcID {
					continue
				}

				fmt.Fprintf(b, "%s.%s -> %s.%s: %q {\n",
					pub.svcID, pub.evtID,
					recv.svcID, recv.evtID,
					evtID,
				)
				b.WriteString("  style: {\n")
				b.WriteString("    stroke: \"#c62828\"\n")
				b.WriteString("    stroke-width: 2\n")
				b.WriteString("    animated: true\n")
				b.WriteString("  }\n}\n\n")

				drawn++
			}
		}
	}

	if drawn > 0 {
		b.WriteString("\n")
	}
}

func (e *Exporter) writeDomains(buf *strings.Builder, cat *catalog.Catalog) {
	for _, domain := range cat.Domains {
		if len(domain.Services) == 0 {
			continue
		}

		domainID := sanitizeID(domain.ID)

		fmt.Fprintf(buf, "domain_%s: {\n", domainID)
		fmt.Fprintf(buf, "  label: %q\n  shape: text\n", domain.Name)

		buf.WriteString(
			"  style: {\n    font-size: 16\n    bold: true\n    font-color: \"#424242\"\n  }\n}\n\n",
		)

		for _, svcRef := range domain.Services {
			svcID := sanitizeID(svcRef)

			fmt.Fprintf(buf, "domain_%s -> %s: \"contains\" {\n", domainID, svcID)

			buf.WriteString(
				"  style: {\n    stroke: \"#bdbdbd\"\n    stroke-dash: 3\n  }\n}\n\n",
			)
		}
	}
}

func sanitizeID(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "/", "_")

	return strings.ToLower(s)
}
