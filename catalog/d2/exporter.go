package d2

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

const (
	shapeRectangle = "rectangle"
	shapeCylinder  = "cylinder"
	shapeQueue     = "queue"
	shapeHexagon   = "hexagon"
)

type styleConfig struct {
	fill          string
	stroke        string
	strokeWidth   int
	borderRadius  int
	fontColor     string
	bold          bool
	shape         string
	animated      bool
	strokeDash    int
	label         string
	tooltip       string
	connectionDir string
}

type nodeConfig struct {
	id    string
	label string
	shape string
	class string
	tooltip string
	style styleConfig
}

type connectionConfig struct {
	from       string
	to         string
	label      string
	arrowColor string
	animated   bool
	dashed     bool
	strokeWidth int
}

type serviceStyle struct {
	cmdShape string
	evtShape string
	queryShape string
	evtDirection string
}

var defaultServiceStyles = map[string]serviceStyle{
	"command": {cmdShape: shapeRectangle, evtShape: shapeQueue, queryShape: shapeRectangle, evtDirection: "send"},
}

type Exporter struct {
	Title       string
	Version     string
	Description string
	Direction   string
}

type Option func(*Exporter)

func WithDescription(desc string) Option {
	return func(e *Exporter) {
		e.Description = desc
	}
}

func WithDirection(dir string) Option {
	return func(e *Exporter) {
		e.Direction = dir
	}
}

func NewExporter(title, version string, opts ...Option) *Exporter {
	e := &Exporter{
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
	var b strings.Builder

	e.writeHeader(&b, cat)
	e.writeClasses(&b)
	e.writeServices(&b, cat)

	if len(cat.Domains) > 0 {
		e.writeDomains(&b, cat)
	}

	return b.String()
}

func (e *Exporter) writeHeader(b *strings.Builder, cat *catalog.Catalog) {
	if e.Title != "" {
		fmt.Fprintf(b, "title: {\n  label: %q\n  near: top-center\n  shape: text\n", e.Title)
		b.WriteString("  style: {\n    font-size: 28\n    bold: true\n    underline: true\n  }\n}\n\n")
	}

	if e.Description != "" {
		fmt.Fprintf(b, "subtitle: {\n  label: %q\n  near: top-center\n  shape: text\n", e.Description)
		b.WriteString("  style: {\n    font-size: 13\n    italic: true\n    font-color: \"#555555\"\n  }\n}\n\n")
	}
}

func (e *Exporter) writeClasses(b *strings.Builder) {
	b.WriteString("classes: {\n")
	b.WriteString("  service: {\n    style: {\n")
	b.WriteString("      fill: \"#e8f5e9\"\n      stroke: \"#2e7d32\"\n      stroke-width: 2\n")
	b.WriteString("      border-radius: 8\n      font-color: \"#1b5e20\"\n      bold: true\n    }\n  }\n")
	b.WriteString("  command: {\n    style: {\n")
	b.WriteString("      fill: \"#e3f2fd\"\n      stroke: \"#1565c0\"\n      stroke-width: 2\n")
	b.WriteString("      border-radius: 6\n      font-color: \"#0d47a1\"\n    }\n  }\n")
	b.WriteString("  event: {\n    style: {\n")
	b.WriteString("      fill: \"#fce4ec\"\n      stroke: \"#c62828\"\n      stroke-width: 2\n")
	b.WriteString("      border-radius: 6\n      font-color: \"#b71c1c\"\n    }\n  }\n")
	b.WriteString("  query: {\n    style: {\n")
	b.WriteString("      fill: \"#f3e5f5\"\n      stroke: \"#6a1b9a\"\n      stroke-width: 2\n")
	b.WriteString("      border-radius: 6\n      font-color: \"#4a148c\"\n    }\n  }\n")
	b.WriteString("}\n\n")
}

func (e *Exporter) writeServices(b *strings.Builder, cat *catalog.Catalog) {
	for _, svc := range cat.Services {
		svcID := sanitizeID(svc.ID)
		fmt.Fprintf(b, "%s: {\n", svcID)
		fmt.Fprintf(b, "  class: service\n  label: %q\n", svc.Name)
		b.WriteString("  direction: down\n\n")

		for _, cmd := range svc.Commands {
			e.writeMessageNode(b, cmd, "command", shapeRectangle)
		}

		for _, evt := range svc.Events {
			e.writeMessageNode(b, evt, "event", shapeQueue)
		}

		for _, q := range svc.Queries {
			e.writeMessageNode(b, q, "query", shapeRectangle)
		}

		e.writeInternalConnections(b, svc)

		b.WriteString("}\n\n")
	}
}

func (e *Exporter) writeMessageNode(b *strings.Builder, msg catalog.Message, class, shape string) {
	msgID := sanitizeID(catalog.MessageID(msg))

	fmt.Fprintf(b, "  %s: {\n", msgID)
	fmt.Fprintf(b, "    class: %s\n", class)

	if shape != shapeRectangle {
		fmt.Fprintf(b, "    shape: %s\n", shape)
	}

	label := msg.Name
	if msg.Version != "" {
		label += fmt.Sprintf(" (v%s)", msg.Version)
	}

	fmt.Fprintf(b, "    label: %q\n", label)

	if msg.Summary != "" {
		fmt.Fprintf(b, "    tooltip: %q\n", msg.Summary)
	}

	b.WriteString("  }\n")
}

func (e *Exporter) writeInternalConnections(b *strings.Builder, svc catalog.Service) {
	if len(svc.Commands) == 0 && len(svc.Events) == 0 && len(svc.Queries) == 0 {
		return
	}

	svcID := sanitizeID(svc.ID)

	for _, cmd := range svc.Commands {
		cmdID := sanitizeID(catalog.MessageID(cmd))
		fmt.Fprintf(b, "  %s -> %s.%s: \"receives\"\n", svcID, svcID, cmdID)
	}

	for _, evt := range svc.Events {
		evtID := sanitizeID(catalog.MessageID(evt))
		action := "publishes"
		if evt.Direction == catalog.Receives {
			action = "receives"
		}

		fmt.Fprintf(b, "  %s.%s -> %s: %q\n", svcID, evtID, svcID, action)
	}

	for _, q := range svc.Queries {
		qID := sanitizeID(catalog.MessageID(q))
		fmt.Fprintf(b, "  %s -> %s.%s: \"handles\"\n", svcID, svcID, qID)
	}

	b.WriteString("\n")
}

func (e *Exporter) writeDomains(b *strings.Builder, cat *catalog.Catalog) {
	for _, domain := range cat.Domains {
		if len(domain.Services) == 0 {
			continue
		}

		domainID := sanitizeID(domain.ID)

		fmt.Fprintf(b, "domain_%s: {\n", domainID)
		fmt.Fprintf(b, "  label: %q\n  shape: text\n", domain.Name)
		b.WriteString("  style: {\n    font-size: 16\n    bold: true\n    font-color: \"#424242\"\n  }\n}\n\n")

		for _, svcRef := range domain.Services {
			svcID := sanitizeID(svcRef)
			fmt.Fprintf(b, "domain_%s -> %s: \"contains\" {\n", domainID, svcID)
			b.WriteString("  style: {\n    stroke: \"#bdbdbd\"\n    stroke-dash: 3\n  }\n}\n\n")
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
