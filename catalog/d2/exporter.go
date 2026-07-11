package d2

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

const (
	shapeRectangle = "rectangle"
	shapeQueue     = "queue"
)

// Exporter generates a D2 diagram from a catalog.
type Exporter struct {
	title       string
	version     string
	description string
	direction   string
}

// Option configures an Exporter.
type Option = catalog.Option[Exporter]

// WithDescription sets the diagram description.
func WithDescription(desc string) Option {
	return func(e *Exporter) {
		e.description = desc
	}
}

// WithDirection sets the diagram direction ("up" or "down").
func WithDirection(dir string) Option {
	return func(e *Exporter) {
		e.direction = dir
	}
}

// NewExporter creates a D2 exporter with the given title and version.
func NewExporter(title, version string, opts ...Option) *Exporter {
	e := &Exporter{ //nolint:exhaustruct // Description is optional, filled by WithDescription
		title:     title,
		version:   version,
		direction: "down",
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}
