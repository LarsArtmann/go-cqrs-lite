package docserver

import (
	"cmp"

	"github.com/larsartmann/templ-components/display"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
)

// This file builds the view model for the error-surface page
// (/docs/eventcatalog/errors): every named error of the analyzed module
// from an erraudit errorflow export, with referenced/unreferenced status —
// the same surface the exporter emits as x-errors frontmatter. The page is
// explicit about its three states: attached (table + stats), not attached
// (Config.EventCatalogErrorsFile unset), and broken (unreadable or
// schema-mismatched file) — never a silently empty table.

// errorStatusReferenced / errorStatusUnreferenced mirror erraudit's
// errorflow status vocabulary (pkg/errorflow.Status*).
const (
	errorStatusReferenced   = "referenced"
	errorStatusUnreferenced = "unreferenced"
)

// eventCatalogErrorsDetail is the view model of the error-surface page.
type eventCatalogErrorsDetail struct {
	Brand    string
	DocsPath string
	// Rows is nil unless a valid errorflow export was read.
	Rows []eventcatalog.ErrorflowError
	// ReadError carries the error from reading/parsing the errorflow file
	// (Config.EventCatalogErrorsFile set but broken).
	ReadError string
	// Attached is false when no errorflow file is configured at all.
	Attached  bool
	FileName  string
	Package   string
	FilesScan int
	Total     int
	Referenced   int
	Unreferenced int
}

// newEventCatalogErrorsDetail projects an errorflow export (or its
// absence/failure) into the page view model. Attached is derived: a nil
// flow with no read error means no file was configured at all.
func newEventCatalogErrorsDetail(cfg Config, flow *eventcatalog.Errorflow, readErr error) eventCatalogErrorsDetail {
	detail := eventCatalogErrorsDetail{
		Brand:    cfg.ServiceName,
		DocsPath: cfg.DocsPath,
		Attached: flow != nil || readErr != nil,
		FileName: cfg.EventCatalogErrorsFile,
	}
	if readErr != nil {
		detail.ReadError = readErr.Error()

		return detail
	}
	if flow == nil {
		return detail
	}

	detail.Rows = flow.Errors
	detail.Package = flow.Package
	detail.FilesScan = flow.FilesScanned
	detail.Total = len(flow.Errors)

	for _, entry := range flow.Errors {
		if entry.Status == errorStatusUnreferenced {
			detail.Unreferenced++
		}
	}

	detail.Referenced = detail.Total - detail.Unreferenced

	return detail
}

// errorStatusBadgeTypes maps errorflow status vocabulary to badge tones.
var errorStatusBadgeTypes = map[string]display.BadgeType{ //nolint:gochecknoglobals // static style lookup table
	errorStatusReferenced:   display.BadgeSuccess,
	errorStatusUnreferenced: display.BadgeWarning,
}

// errorRows builds the error-surface table rows in export order (already
// name-sorted by erraudit — deterministic, straight projection).
func errorRows(rows []eventcatalog.ErrorflowError) []display.TableRow {
	tableRows := make([]display.TableRow, 0, len(rows))
	for _, entry := range rows {
		tableRows = append(tableRows, tableRow(
			textCell(entry.Name),
			textCell(entry.Kind),
			textCell(cmp.Or(entry.Code, "—")),
			componentCell(errorStatusBadge(entry.Status)),
			textCell(entry.Message),
			textCell(cmp.Or(entry.Wraps, "—")),
		))
	}

	return tableRows
}
