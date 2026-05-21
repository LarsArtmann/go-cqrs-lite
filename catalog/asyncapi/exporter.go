package asyncapi

const (
	asyncAPIVersion = "3.0.0"
	contentType     = "application/json"

	actionSend    = "send"
	actionReceive = "receive"
)

type messageKind string

const (
	kindCommand messageKind = "commands"
	kindEvent   messageKind = "events"
	kindQuery   messageKind = "queries"
)

// Exporter generates an AsyncAPI 3.0 document from a catalog.
type Exporter struct {
	serviceName string
	version     string
	description string
	protocol    string
	host        string
	serverName  string
}

// Option configures an Exporter.
type Option func(*Exporter)

// WithServer sets the server name, host, and protocol for the AsyncAPI document.
func WithServer(name, host, protocol string) Option {
	return func(e *Exporter) {
		e.serverName = name
		e.host = host
		e.protocol = protocol
	}
}

// WithDescription sets the description for the AsyncAPI document.
func WithDescription(desc string) Option {
	return func(e *Exporter) {
		e.description = desc
	}
}

// NewExporter creates an AsyncAPI exporter with the given service name and version.
func NewExporter(serviceName, version string, opts ...Option) *Exporter {
	e := &Exporter{
		serviceName: serviceName,
		version:     version,
		description: "",
		protocol:    "kafka",
		host:        "localhost:9092",
		serverName:  "production",
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

type messageOption func(*messageConfig)

type messageConfig struct {
	action string
}

func withAction(action string) messageOption {
	return func(c *messageConfig) {
		c.action = action
	}
}
