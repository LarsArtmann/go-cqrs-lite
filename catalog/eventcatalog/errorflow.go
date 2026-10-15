package eventcatalog

import (
	"encoding/json/v2"
	"os"

	errorfamily "github.com/larsartmann/go-error-family"
)

// errorflowSchemaVersion is the erraudit errorflow export schema this reader
// supports. The schema is a cross-repo contract (erraudit
// pkg/errorflow.Export); bump-mismatch rejects loudly rather than emitting
// silently-wrong frontmatter.
const errorflowSchemaVersion = 1

// errorflowExport mirrors erraudit's errorflow JSON document (schemaVersion
// 1): every named error of the analyzed module with kind, go-error-family
// code/family, declaration site, declared parent, reference sites, and a
// referenced/unreferenced status. Only the fields the catalog surfaces are
// decoded; unknown fields are ignored for forward compatibility within the
// same schema version.
type errorflowExport struct {
	SchemaVersion int              `json:"schemaVersion"`
	Package       string           `json:"package"`
	FilesScanned  int              `json:"filesScanned"`
	Errors        []errorflowError `json:"errors"`
}

type errorflowError struct {
	Name      string          `json:"name"`
	Kind      string          `json:"kind"`
	Code      string          `json:"code"`
	Family    string          `json:"family"`
	Message   string          `json:"message"`
	Declared  string          `json:"declared"`
	Wraps     string          `json:"wraps"`
	Status    string          `json:"status"`
	WrappedBy []errorflowSite `json:"wrappedBy"`
	CheckedBy []errorflowSite `json:"checkedBy"`
}

type errorflowSite struct {
	Position string `json:"position"`
	Function string `json:"function"`
}

// readErrorflowFile parses an erraudit errorflow export from disk and
// validates the schema version. The site lists (wrappedBy/checkedBy) are
// intentionally NOT carried into frontmatter: catalogs need the surface
// (what errors exist and their status), not per-site call positions — those
// live in the source repo and the hub's error-insights pipeline.
func readErrorflowFile(path string) (*errorflowExport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errorfamily.Newf(errorfamily.Rejection, "catalog.exporter.errorflow.read.1",
			"read errorflow file %s: %v", path, err)
	}

	var export errorflowExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, errorfamily.Newf(errorfamily.Rejection, "catalog.exporter.errorflow.parse.1",
			"parse errorflow file %s as JSON: %v", path, err)
	}

	if export.SchemaVersion != errorflowSchemaVersion {
		return nil, errorfamily.Newf(errorfamily.Rejection, "catalog.exporter.errorflow.schema.1",
			"errorflow file %s: schemaVersion %d is not the supported %d — regenerate it with a "+
				"matching erraudit version (erraudit errorflow ./... --type-aware)",
			path, export.SchemaVersion, errorflowSchemaVersion)
	}

	return &export, nil
}

// xErrorsFM is the `x-errors` frontmatter block emitted on service pages
// when an errorflow export is provided. The `x-` prefix is EventCatalog's
// extension namespace: unknown x-* keys pass schema validation (contract
// tested), so the error surface rides the service page without forking the
// core format.
//
// Scope is honest about attribution: erraudit analyzes a MODULE, and the
// catalog does not map errors to individual services — every service page
// of the export carries the module-level surface, labeled as such.
type xErrorsFM struct {
	Scope         string     `yaml:"scope"`
	Tool          string     `yaml:"tool"`
	SchemaVersion int        `yaml:"schemaVersion"`
	Errors        []xErrorFM `yaml:"errors"`
	FilesScanned  int        `yaml:"filesScanned"`
	Source        string     `yaml:"source,omitempty"`
}

type xErrorFM struct {
	Name    string `yaml:"name"`
	Kind    string `yaml:"kind"`
	Code    string `yaml:"code,omitempty"`
	Family  string `yaml:"family,omitempty"`
	Status  string `yaml:"status"`
	Wraps   string `yaml:"wraps,omitempty"`
	Message string `yaml:"message,omitempty"`
}

// toXErrors converts the errorflow export into frontmatter shape. The error
// list is already name-sorted by erraudit (deterministic exports); this is
// a straight projection, not a re-sort.
func (e *errorflowExport) toXErrors() xErrorsFM {
	errors := make([]xErrorFM, 0, len(e.Errors))
	for _, entry := range e.Errors {
		errors = append(errors, xErrorFM{
			Name:    entry.Name,
			Kind:    entry.Kind,
			Code:    entry.Code,
			Family:  entry.Family,
			Status:  entry.Status,
			Wraps:   entry.Wraps,
			Message: entry.Message,
		})
	}

	return xErrorsFM{
		Scope:         "module",
		Tool:          "erraudit",
		SchemaVersion: e.SchemaVersion,
		Errors:        errors,
		FilesScanned:  e.FilesScanned,
		Source:        e.Package,
	}
}

// WithErrorflowFile attaches an erraudit errorflow export (JSON,
// schemaVersion 1 — `erraudit errorflow ./... --type-aware`) to the export:
// every service page carries the module's error surface as `x-errors`
// frontmatter, and catalog.index.json gains an errorSurface section
// (schemaVersion 2) so federation gates can diff error-surface changes
// (error_surface_changed) like any other breaking change.
//
// The file is read once at Export time; a missing or schema-mismatched file
// fails the export loudly — a silently absent error surface would drift the
// manifest unnoticed.
func WithErrorflowFile(path string) Option {
	return func(e *Exporter) {
		e.errorflowPath = path
	}
}

// ErrorflowError is the consumer-facing projection of one error in an
// erraudit errorflow export: the surface fields (what the error is and
// whether anything references it), without the per-site call positions.
type ErrorflowError struct {
	Name    string
	Kind    string
	Code    string
	Family  string
	Message string
	Wraps   string
	Status  string
}

// Errorflow is the consumer-facing view of an errorflow export, for
// consumers beyond the exporter (the docserver's error-surface page).
type Errorflow struct {
	SchemaVersion int
	Package       string
	FilesScanned  int
	Errors        []ErrorflowError
}

// ReadErrorflow reads and validates an erraudit errorflow export for
// non-exporter consumers. Same loud schema contract as WithErrorflowFile:
// a missing or future-version file is an error, never a silently empty
// surface.
func ReadErrorflow(path string) (*Errorflow, error) {
	export, err := readErrorflowFile(path)
	if err != nil {
		return nil, err
	}

	errors := make([]ErrorflowError, 0, len(export.Errors))
	for _, entry := range export.Errors {
		errors = append(errors, ErrorflowError{
			Name:    entry.Name,
			Kind:    entry.Kind,
			Code:    entry.Code,
			Family:  entry.Family,
			Message: entry.Message,
			Wraps:   entry.Wraps,
			Status:  entry.Status,
		})
	}

	return &Errorflow{
		SchemaVersion: export.SchemaVersion,
		Package:       export.Package,
		FilesScanned:  export.FilesScanned,
		Errors:        errors,
	}, nil
}
