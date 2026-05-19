package catalog

import "reflect"

// MessageConfig is implemented by message builders produced by Command[T](),
// Event[T](), and Query[T]().
type MessageConfig interface {
	apply(serviceID ServiceID, reg *Registry)
}

type messageBuilder struct {
	kind      MessageKind
	id        MessageID
	name      string
	version   string
	summary   string
	schema    *Schema
	direction Direction
}

func (m *messageBuilder) apply(serviceID ServiceID, reg *Registry) {
	msg := Message{ //nolint:exhaustruct
		Kind:      m.kind,
		ID:        m.id,
		Name:      m.name,
		Version:   m.version,
		Summary:   m.summary,
		Schema:    m.schema,
		Direction: m.direction,
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

// Name overrides the auto-derived message name.
func Name(name string) MessageOption {
	return func(m *messageBuilder) {
		m.name = name
	}
}

// Summary sets the message summary/description.
func Summary(summary string) MessageOption {
	return func(m *messageBuilder) {
		m.summary = summary
	}
}

// Version overrides the default message version ("1.0.0").
func Version(version string) MessageOption {
	return func(m *messageBuilder) {
		m.version = version
	}
}

// Command creates a command message configuration for catalog.Builder.AddService.
// The schema is auto-derived from T using reflection on its struct fields and tags.
// The name is auto-derived from T's type name (e.g. CreateUserCmd → "Create User").
// Direction defaults to Receives.
func Command[T any](id string, opts ...MessageOption) MessageConfig {
	return newMessageBuilder[T](CommandMessage, MessageID(id), Receives, opts)
}

// Event creates an event message configuration for catalog.Builder.AddService.
// The schema is auto-derived from T using reflection on its struct fields and tags.
// The name is auto-derived from T's type name (e.g. UserCreatedEvent → "User Created").
// Direction must be explicit (Sends or Receives).
func Event[T any](id string, direction Direction, opts ...MessageOption) MessageConfig {
	return newMessageBuilder[T](EventMessage, MessageID(id), direction, opts)
}

// Query creates a query message configuration for catalog.Builder.AddService.
// The schema is auto-derived from T using reflection on its struct fields and tags.
// The name is auto-derived from T's type name (e.g. GetUserQuery → "Get User").
// Direction defaults to Receives.
func Query[T any](id string, opts ...MessageOption) MessageConfig {
	return newMessageBuilder[T](QueryMessage, MessageID(id), Receives, opts)
}

func newMessageBuilder[T any](
	kind MessageKind,
	id MessageID,
	direction Direction,
	opts []MessageOption,
) MessageConfig {
	rt := reflect.TypeFor[T]()

	name := camelCaseToHuman(rt.Name())
	schema := schemaFromReflect(rt)

	msgBuilder := &messageBuilder{
		kind:      kind,
		id:        id,
		name:      name,
		version:   "1.0.0",
		summary:   "",
		schema:    schema,
		direction: direction,
	}

	for _, opt := range opts {
		opt(msgBuilder)
	}

	return msgBuilder
}
