package d2

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

type eventOwner struct {
	serviceDisplayID string
	eventDisplayID   string
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

	serviceDisplayID := sanitizeID(string(svc.ID))

	writeInternalEdges(buf, serviceDisplayID, svc)

	buf.WriteString("\n")
}

func writeInternalEdges(buf *strings.Builder, serviceDisplayID string, svc catalog.Service) {
	for _, cmd := range svc.Commands {
		commandDisplayID := sanitizeID(string(catalog.GetID(cmd)))
		fmt.Fprintf(
			buf,
			"  %s -> %s.%s: \"receives\"\n",
			serviceDisplayID,
			serviceDisplayID,
			commandDisplayID,
		)
	}

	for _, evt := range svc.Events {
		eventDisplayID := sanitizeID(string(catalog.GetID(evt)))
		action := eventAction(evt)
		fmt.Fprintf(
			buf,
			"  %s.%s -> %s: %q\n",
			serviceDisplayID,
			eventDisplayID,
			serviceDisplayID,
			action,
		)
	}

	for _, q := range svc.Queries {
		queryDisplayID := sanitizeID(string(catalog.GetID(q)))
		fmt.Fprintf(
			buf,
			"  %s -> %s.%s: \"handles\"\n",
			serviceDisplayID,
			serviceDisplayID,
			queryDisplayID,
		)
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
) (map[catalog.MessageID][]eventOwner, map[catalog.MessageID][]eventOwner) {
	publishers := make(map[catalog.MessageID][]eventOwner)
	receivers := make(map[catalog.MessageID][]eventOwner)

	for _, svc := range cat.Services {
		serviceDisplayID := sanitizeID(string(svc.ID))

		for _, evt := range svc.Events {
			messageID := catalog.GetID(evt)
			owner := eventOwner{
				serviceDisplayID: serviceDisplayID,
				eventDisplayID:   sanitizeID(string(messageID)),
			}

			switch evt.Direction {
			case catalog.Sends:
				publishers[messageID] = append(publishers[messageID], owner)
			case catalog.Receives:
				receivers[messageID] = append(receivers[messageID], owner)
			}
		}
	}

	return publishers, receivers
}

func writeCrossServiceEdge(b *strings.Builder, pub, recv eventOwner, messageID catalog.MessageID) {
	fmt.Fprintf(
		b, "%s.%s -> %s.%s: %q {\n",
		pub.serviceDisplayID, pub.eventDisplayID,
		recv.serviceDisplayID, recv.eventDisplayID,
		string(messageID),
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

	for messageID, pubs := range publishers {
		recvs, ok := receivers[messageID]
		if !ok {
			continue
		}

		for _, pub := range pubs {
			for _, recv := range recvs {
				if pub.serviceDisplayID == recv.serviceDisplayID {
					continue
				}

				writeCrossServiceEdge(b, pub, recv, messageID)

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

		domainDisplayID := sanitizeID(string(domain.ID))

		fmt.Fprintf(buf, "domain_%s: {\n", domainDisplayID)
		fmt.Fprintf(buf, "  label: %q\n  shape: text\n", domain.Name)

		buf.WriteString(
			"  style: {\n    font-size: 16\n    bold: true\n    font-color: \"#424242\"\n  }\n}\n\n",
		)

		for _, svcRef := range domain.Services {
			serviceDisplayID := sanitizeID(string(svcRef))

			fmt.Fprintf(buf, "domain_%s -> %s: \"contains\" {\n", domainDisplayID, serviceDisplayID)

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
