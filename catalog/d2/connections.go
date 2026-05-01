package d2

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

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
