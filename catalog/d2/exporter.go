package d2

const (
	shapeRectangle = "rectangle"
	shapeQueue     = "queue"
)

type Exporter struct {
	title       string
	version     string
	description string
	direction   string
}

type Option func(*Exporter)

func WithDescription(desc string) Option {
	return func(e *Exporter) {
		e.description = desc
	}
}

func WithDirection(dir string) Option {
	return func(e *Exporter) {
		e.direction = dir
	}
}

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
