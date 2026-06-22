package catalog

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3/schema"
)

// MessageConfig is implemented by message builders produced by Command[T](),
// Event[T](), and Query[T]().
type MessageConfig interface {
	apply(serviceID ServiceID, reg *Registry)
}

const defaultVersion = "1.0.0"

type messageBuilder struct {
	kind      MessageKind
	id        MessageID
	name      string
	version   string
	summary   string
	schema    *Schema
	direction Direction
	owners    []string
	labels    map[string]string
	producers []ServiceID
	consumers []ServiceID
	op        *Operation
	badges    []Badge
	repo      *Repository
}

func (m *messageBuilder) apply(serviceID ServiceID, reg *Registry) {
	msg := Message{ //nolint:exhaustruct // optional fields omitted by design
		Kind:       m.kind,
		ID:         m.id,
		Name:       Name(m.name),
		Version:    Version(m.version),
		Summary:    Summary(m.summary),
		Schema:     m.schema,
		Direction:  m.direction,
		Owners:     m.owners,
		Labels:     m.labels,
		Producers:  m.producers,
		Consumers:  m.consumers,
		Operation:  m.op,
		Badges:     m.badges,
		Repository: m.repo,
	}

	switch m.kind {
	case CommandMessage:
		reg.AddCommand(serviceID, msg)
	case EventMessage:
		reg.AddEvent(serviceID, msg)
	case QueryMessage:
		reg.AddQuery(serviceID, msg)
	}
}

// MessageOption configures a message builder.
type MessageOption func(*messageBuilder)

// WithName overrides the auto-derived message name.
func WithName(name string) MessageOption {
	return func(m *messageBuilder) {
		m.name = name
	}
}

// WithSummary sets the message summary/description.
func WithSummary(summary string) MessageOption {
	return func(m *messageBuilder) {
		m.summary = summary
	}
}

// WithVersion overrides the default message version ("1.0.0").
func WithVersion(version string) MessageOption {
	return func(m *messageBuilder) {
		m.version = version
	}
}

// Owners sets the list of owners (teams or individuals) for the message.
func Owners(owners ...string) MessageOption {
	return func(m *messageBuilder) {
		m.owners = owners
	}
}

// Labels sets key-value labels for cross-cutting grouping.
func Labels(labels map[string]string) MessageOption {
	return func(m *messageBuilder) {
		m.labels = labels
	}
}

// Producers sets the list of service IDs that produce this message.
func Producers(ids ...ServiceID) MessageOption {
	return func(m *messageBuilder) {
		m.producers = ids
	}
}

// Consumers sets the list of service IDs that consume this message.
func Consumers(ids ...ServiceID) MessageOption {
	return func(m *messageBuilder) {
		m.consumers = ids
	}
}

// MsgOperation maps this message to an HTTP endpoint.
func MsgOperation(method, path string, statusCodes ...string) MessageOption {
	return func(m *messageBuilder) {
		m.op = &Operation{Method: Method(method), Path: path, StatusCodes: statusCodes}
	}
}

// MsgBadges adds visual badges to this message.
func MsgBadges(badges ...Badge) MessageOption {
	return func(m *messageBuilder) {
		m.badges = badges
	}
}

// MsgRepository attaches code repository metadata to this message.
func MsgRepository(language, url string) MessageOption {
	return func(m *messageBuilder) {
		m.repo = &Repository{Language: Language(language), URL: URL(url)}
	}
}

// Command creates a command message configuration for catalog.Builder.AddService.
// The schema is auto-derived from T using reflection on its struct fields and tags.
// The name is auto-derived from T's type name (e.g. CreateUserCmd → "Create User").
// Direction defaults to Receives.
func Command[T any](id MessageID, opts ...MessageOption) MessageConfig {
	return newMessageBuilder[T](CommandMessage, id, Receives, opts)
}

// Event creates an event message configuration for catalog.Builder.AddService.
// The schema is auto-derived from T using reflection on its struct fields and tags.
// The name is auto-derived from T's type name (e.g. UserCreatedEvent → "User Created").
// Direction must be explicit (Sends or Receives).
func Event[T any](id MessageID, direction Direction, opts ...MessageOption) MessageConfig {
	return newMessageBuilder[T](EventMessage, id, direction, opts)
}

// Query creates a query message configuration for catalog.Builder.AddService.
// The schema is auto-derived from T using reflection on its struct fields and tags.
// The name is auto-derived from T's type name (e.g. GetUserQuery → "Get User").
// Direction defaults to Receives.
func Query[T any](id MessageID, opts ...MessageOption) MessageConfig {
	return newMessageBuilder[T](QueryMessage, id, Receives, opts)
}

func newMessageBuilder[T any](
	kind MessageKind,
	id MessageID,
	direction Direction,
	opts []MessageOption,
) MessageConfig {
	rt := reflect.TypeFor[T]()

	name := camelCaseToHuman(rt.Name())
	sch := schema.FromReflect(rt)

	msgBuilder := &messageBuilder{ //nolint:exhaustruct // optional fields omitted by design
		kind:      kind,
		id:        id,
		name:      name,
		version:   defaultVersion,
		summary:   "",
		schema:    sch,
		direction: direction,
	}

	for _, opt := range opts {
		opt(msgBuilder)
	}

	return msgBuilder
}

// cmdSuffix is extracted to a constant because it appears 3+ times across the codebase.
const cmdSuffix = "Cmd"

func camelCaseToHuman(s string) string {
	knownSuffixes := []string{
		"Command",
		cmdSuffix,
		"Event",
		"Evt",
		"Query",
		"Qry",
	}

	for _, suffix := range knownSuffixes {
		if stripped, ok := strings.CutSuffix(s, suffix); ok && stripped != "" {
			s = stripped

			break
		}
	}

	var result strings.Builder

	for i, r := range s {
		switch {
		case i == 0:
			result.WriteRune(unicode.ToUpper(r))
		case unicode.IsUpper(r):
			result.WriteRune(' ')
			result.WriteRune(r)
		default:
			result.WriteRune(r)
		}
	}

	return result.String()
}
