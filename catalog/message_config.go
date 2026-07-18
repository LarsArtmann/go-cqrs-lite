package catalog

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"strings"
	"unicode"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
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
	responses []ResponseSpec
	examples  []jsontext.Value
	badges    []Badge
	repo      *Repository
	security  []string
}

func (m *messageBuilder) apply(serviceID ServiceID, reg *Registry) {
	msg := Message{
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
		Responses:  m.responses,
		Examples:   m.examples,
		Badges:     m.badges,
		Repository: m.repo,
		Security:   m.security,
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

// Response adds a typed response with a body schema auto-derived from T.
func Response[T any](statusCode, description string) MessageOption {
	return func(m *messageBuilder) {
		m.responses = append(m.responses, ResponseSpec{
			StatusCode:  statusCode,
			Description: description,
			Schema:      schema.FromType[T](),
		})
	}
}

// WithResponse adds a typed response without a body schema (e.g. 204 No Content).
func WithResponse(statusCode, description string) MessageOption {
	return func(m *messageBuilder) {
		m.responses = append(m.responses, ResponseSpec{
			StatusCode:  statusCode,
			Description: description,
		})
	}
}

// WithParam adds an HTTP parameter to this message.
func WithParam(name, location, description string, required bool) MessageOption {
	return func(m *messageBuilder) {
		if m.schema == nil {
			m.schema = &Schema{Type: TypeObject, Properties: map[string]Property{}}
		}

		if m.schema.Parameters == nil {
			m.schema.Parameters = []Parameter{}
		}

		m.schema.Parameters = append(m.schema.Parameters, Parameter{
			Name: name, In: location, Description: description, Required: required,
		})
	}
}

// WithExample adds an example payload to this message.
func WithExample(value any) MessageOption {
	return func(m *messageBuilder) {
		raw, err := json.Marshal(value, json.Deterministic(true))
		if err != nil {
			return
		}

		m.examples = append(m.examples, jsontext.Value(raw))
	}
}

// MsgSecurity attaches security scheme IDs to this message.
func MsgSecurity(schemeIDs ...string) MessageOption {
	return func(m *messageBuilder) {
		m.security = schemeIDs
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

// DELETE creates a command message pre-tagged with an HTTP DELETE operation.
// The schema is auto-derived from T. Equivalent to:
//
//	catalog.Command[T](id, catalog.MsgOperation("DELETE", path), opts...)
func DELETE[T any](id MessageID, path string, opts ...MessageOption) MessageConfig {
	return Command[T](id, append([]MessageOption{MsgOperation("DELETE", path)}, opts...)...)
}

// PUT creates a command message pre-tagged with an HTTP PUT operation.
func PUT[T any](id MessageID, path string, opts ...MessageOption) MessageConfig {
	return Command[T](id, append([]MessageOption{MsgOperation("PUT", path)}, opts...)...)
}

// PATCH creates a command message pre-tagged with an HTTP PATCH operation.
func PATCH[T any](id MessageID, path string, opts ...MessageOption) MessageConfig {
	return Command[T](id, append([]MessageOption{MsgOperation("PATCH", path)}, opts...)...)
}

// WithOperation is a composite MessageOption that sets the HTTP operation
// (method + path) and a typed success response in a single call, reducing
// boilerplate when registering REST endpoints:
//
//	catalog.Command[CreateUserCmd]("user.create",
//	    catalog.WithOperation[UserDTO]("POST", "/api/users", "201"),
//	)
func WithOperation[T any](method Method, path, successCode string) MessageOption {
	return func(m *messageBuilder) {
		MsgOperation(string(method), path)(m)
		Response[T](successCode, "")(m)
	}
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

	msgBuilder := &messageBuilder{
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
