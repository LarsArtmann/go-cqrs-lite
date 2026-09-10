package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// writeCoeffectSummary renders the coeffect graph — every event with its
// derived producers and consumers and its validation status — to
// coeffects.md at the export root. Dangling rows (consumed, never produced)
// mirror Catalog.ValidateCoeffects violations; unconsumed rows are the
// advisory tier (dead-event visibility), not violations. The file is plain
// Markdown outside EventCatalog's content collections, so the Astro build
// ignores it while ops gets the graph next to the rendered site.
func (e *Exporter) writeCoeffectSummary(cat *catalog.Catalog) error {
	enriched := cat.DeriveProducersConsumers()

	//art-dupl:accept trivial strings.Builder prologue — the write-a-report idiom, not domain logic
	var sb strings.Builder

	sb.WriteString("# Coeffect Validation Summary\n\n")
	sb.WriteString("Producer/consumer graph over declared events ")
	sb.WriteString("(derived from service sends/receives; explicit declarations win).\n\n")

	rows := coeffectRows(enriched)
	if len(rows) == 0 {
		sb.WriteString("No events declared.\n")

		return e.writePlainFile("coeffects.md", sb.String())
	}

	sb.WriteString("| Event | Producers | Consumers | Status |\n")
	sb.WriteString("| ----- | --------- | --------- | ------ |\n")

	for _, row := range rows {
		fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n",
			row.event,
			strings.Join(row.producers, ", "),
			strings.Join(row.consumers, ", "),
			row.status,
		)
	}

	sb.WriteString("\nValidation: ")
	if violations := enriched.ValidateCoeffects(); len(violations) > 0 {
		fmt.Fprintf(
			&sb,
			"%d dangling subscription(s) — see Catalog.ValidateCoeffects\n",
			len(violations),
		)
	} else {
		sb.WriteString("no dangling subscriptions\n")
	}

	return e.writePlainFile("coeffects.md", sb.String())
}

type coeffectRow struct {
	event     string
	producers []string
	consumers []string
	status    string
}

// coeffectRows flattens the enriched catalog into one deduplicated row per
// event message with its validation status.
func coeffectRows(cat *catalog.Catalog) []coeffectRow {
	seen := make(map[catalog.MessageID]bool)

	var rows []coeffectRow

	for _, svc := range cat.Services {
		for _, evt := range svc.Events {
			messageID := catalog.Key(evt)
			if seen[messageID] {
				continue
			}

			seen[messageID] = true

			rows = append(rows, coeffectRow{
				event:     string(messageID),
				producers: serviceIDs(evt.Producers),
				consumers: serviceIDs(evt.Consumers),
				status:    coeffectStatus(evt),
			})
		}
	}

	return rows
}

func coeffectStatus(evt catalog.Message) string {
	switch {
	case len(evt.Consumers) > 0 && len(evt.Producers) == 0:
		return "DANGLING (consumed, never produced)"

	case len(evt.Producers) > 0 && len(evt.Consumers) == 0:
		return "unconsumed (advisory)"

	default:
		return "ok"
	}
}

func serviceIDs(ids []catalog.ServiceID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}

	return out
}

// writePlainFile writes non-MDX companion files (coeffects.md) with the
// exporter's standard permissions.
func (e *Exporter) writePlainFile(relPath, content string) error {
	return os.WriteFile( //nolint:wrapcheck // direct passthrough
		filepath.Join(e.outputDir, relPath),
		[]byte(content),
		filePerm,
	)
}
