package docserver

import (
	"net/url"
	"strconv"

	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/display"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// Display glue for the event catalog pages: URL builders and the
// view-model-to-display-table row builders.

// eventCatalogHref returns the event catalog landing page URL.
func eventCatalogHref(docsPath string) string {
	return docsPath + "/eventcatalog"
}

func eventCatalogMessageHref(docsPath string, id catalog.MessageID) string {
	return docsPath + "/eventcatalog/messages/" + url.PathEscape(string(id))
}

func eventCatalogChannelHref(docsPath, id string) string {
	return docsPath + "/eventcatalog/channels/" + url.PathEscape(id)
}

func eventCatalogServiceHref(docsPath, id string) string {
	return docsPath + "/eventcatalog/services/" + url.PathEscape(id)
}

// cmpOr returns a if non-empty, else b (local helper to avoid importing cmp
// for two call sites in the view builders).
func cmpOr(a, b string) string {
	if a != "" {
		return a
	}

	return b
}

// countKind counts message rows of one kind for the overview stat cards.
func countKind(rows []eventCatalogMessageRow, kind string) int {
	count := 0
	for _, row := range rows {
		if row.Kind == kind {
			count++
		}
	}

	return count
}

// messageKindBadgeTypes maps catalog message kinds to badge colors.
var messageKindBadgeTypes = map[string]display.BadgeType{ //nolint:gochecknoglobals // static style lookup table
	"event":   display.BadgeSuccess,
	"command": display.BadgePrimary,
	"query":   display.BadgeInfo,
}

// tableRow assembles a display table row from per-cell builders.
func tableRow(cells ...display.TableCell) display.TableRow {
	return display.TableRow{Cells: cells}
}

func textCell(text string) display.TableCell {
	return display.TableCell{Text: text}
}

func componentCell(c templ.Component) display.TableCell {
	return display.TableCell{Content: c}
}

// messageRows builds the messages table rows.
func messageRows(rows []eventCatalogMessageRow) []display.TableRow {
	out := make([]display.TableRow, 0, len(rows))
	for _, msg := range rows {
		out = append(out, tableRow(
			componentCell(messageKindBadge(msg.Kind)),
			componentCell(messageLinkCell(msg.Name, msg.ID, msg.Href)),
			textCell(msg.Summary),
			textCell(msg.Producer),
			textCell(msg.Consumers),
		))
	}

	return out
}

// channelRows builds the channels table rows.
func channelRows(rows []eventCatalogChannelRow) []display.TableRow {
	out := make([]display.TableRow, 0, len(rows))
	for _, ch := range rows {
		out = append(out, tableRow(
			componentCell(messageLinkCell(ch.Name, "", ch.Href)),
			textCell(ch.Address),
			textCell(ch.Protocols),
			textCell(ch.Guarantee),
			textCell(strconv.Itoa(ch.MessageCnt)),
		))
	}

	return out
}

// serviceRows builds the services table rows.
func serviceRows(rows []eventCatalogServiceRow) []display.TableRow {
	out := make([]display.TableRow, 0, len(rows))
	for _, svc := range rows {
		out = append(out, tableRow(
			componentCell(messageLinkCell(svc.Name, "", svc.Href)),
			textCell(svc.Version),
			textCell(svc.Summary),
			textCell(strconv.Itoa(svc.Commands+svc.Events+svc.Queries)),
		))
	}

	return out
}

// propertyRows builds the schema property table rows of a message detail
// page.
func propertyRows(props []eventCatalogProperty) []display.TableRow {
	out := make([]display.TableRow, 0, len(props))
	for _, prop := range props {
		required := "optional"
		if prop.Required {
			required = "required"
		}

		out = append(out, tableRow(
			textCell(prop.Name),
			textCell(prop.Type),
			textCell(required),
			textCell(prop.Description),
			textCell(prop.Details),
		))
	}

	return out
}
