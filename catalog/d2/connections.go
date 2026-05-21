package d2

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

type eventOwner struct {
	svcID string
	evtID string
}

// Export generates a D2 diagram string from the given catalog.
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

func (e *Exporter) writeInternalConnections(buf *strings.Builder, svc catalog.Service) {
	if len(svc.Commands) == 0 && len(svc.Events) == 0 && len(svc.Queries) == 0 {
		return
	}

	svcID := sanitizeID(string(svc.ID))

	writeInternalEdges(buf, svcID, svc)

	buf.WriteString("\n")
}

func writeInternalEdges(buf *strings.Builder, svcID string, svc catalog.Service) {
	for _, cmd := range svc.Commands {
		cmdID := sanitizeID(string(catalog.GetID(cmd)))
		fmt.Fprintf(buf, "  %s -> %s.%s: \"receives\"\n", svcID, svcID, cmdID)
	}

	for _, evt := range svc.Events {
		evtID := sanitizeID(string(catalog.GetID(evt)))
		action := eventAction(evt)
		fmt.Fprintf(buf, "  %s.%s -> %s: %q\n", svcID, evtID, svcID, action)
	}

	for _, q := range svc.Queries {
		qID := sanitizeID(string(catalog.GetID(q)))
		fmt.Fprintf(buf, "  %s -> %s.%s: \"handles\"\n", svcID, svcID, qID)
	}
}

func eventAction(evt catalog.Message) string {
	if evt.Direction == catalog.Receives {
		return "receives"
	}

	return "publishes"
}

func buildPublisherReceiverMaps(
	cat *catalog.Catalog,
) (map[string][]eventOwner, map[string][]eventOwner) {
	publishers := make(map[string][]eventOwner)
	receivers := make(map[string][]eventOwner)

	for _, svc := range cat.Services {
		svcID := sanitizeID(string(svc.ID))

		for _, evt := range svc.Events {
			evtID := catalog.GetID(evt)
			owner := eventOwner{svcID: svcID, evtID: sanitizeID(string(evtID))}

			switch evt.Direction {
			case catalog.Sends:
				publishers[string(evtID)] = append(publishers[string(evtID)], owner)
			case catalog.Receives:
				receivers[string(evtID)] = append(receivers[string(evtID)], owner)
			}
		}
	}

	return publishers, receivers
}

func writeCrossServiceEdge(b *strings.Builder, pub, recv eventOwner, evtID string) {
	fmt.Fprintf(
		b, "%s.%s -> %s.%s: %q {\n",
		pub.svcID, pub.evtID, recv.svcID, recv.evtID, evtID,
	)
	b.WriteString("  style: {\n")
	b.WriteString("    stroke: \"#c62828\"\n")
	b.WriteString("    stroke-width: 2\n")
	b.WriteString("    animated: true\n")
	b.WriteString("  }\n}\n\n")
}

func (e *Exporter) writeCrossServiceConnections(b *strings.Builder, cat *catalog.Catalog) {
	publishers, receivers := buildPublisherReceiverMaps(cat)

	var hasCrossService bool

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

				writeCrossServiceEdge(b, pub, recv, evtID)

				hasCrossService = true
			}
		}
	}

	if hasCrossService {
		b.WriteString("\n")
	}
}

func (e *Exporter) writeDomains(buf *strings.Builder, cat *catalog.Catalog) {
	for _, domain := range cat.Domains {
		if len(domain.Services) == 0 {
			continue
		}

		domainID := sanitizeID(string(domain.ID))

		fmt.Fprintf(buf, "domain_%s: {\n", domainID)
		fmt.Fprintf(buf, "  label: %q\n  shape: text\n", domain.Name)

		buf.WriteString(
			"  style: {\n    font-size: 16\n    bold: true\n    font-color: \"#424242\"\n  }\n}\n\n",
		)

		for _, svcRef := range domain.Services {
			svcID := sanitizeID(string(svcRef))

			fmt.Fprintf(buf, "domain_%s -> %s: \"contains\" {\n", domainID, svcID)

			buf.WriteString(
				"  style: {\n    stroke: \"#bdbdbd\"\n    stroke-dash: 3\n  }\n}\n\n",
			)
		}
	}
}

func sanitizeID(s string) string {
	return strings.ToLower(strings.Map(func(r rune) rune {
		switch r {
		case ' ', '-', '.', '/':
			return '_'
		default:
			return r
		}
	}, s))
}
