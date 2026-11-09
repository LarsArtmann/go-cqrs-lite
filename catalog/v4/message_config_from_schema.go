package catalog

import "strings"

// CommandFromSchema creates a command message configuration for
// catalog.Builder.AddService from an optional pre-built payload schema.
//
// It is the schema-parameterized sibling of [Command]: use it when the payload
// type is not statically known at the call site (derivation from runtime
// registrations, generators, bridges from other tooling). sch may be nil —
// the message then documents identity (id, name, version) without a payload
// schema. The name is humanized from the id
// ("record_invoice_state" → "Record Invoice State").
func CommandFromSchema(id MessageID, sch *Schema, opts ...MessageOption) MessageConfig {
	return newMessageBuilderFromSchema(CommandMessage, id, Receives, sch, opts)
}

// EventFromSchema creates an event message configuration for
// catalog.Builder.AddService from an optional pre-built payload schema.
//
// It is the schema-parameterized sibling of [Event]; direction must be
// explicit (Sends or Receives). sch may be nil — DomainConfig-derived
// catalogs know event identities and versions but not payload field shapes
// (those live in schema.EventSchema wire declarations, not Go types).
func EventFromSchema(
	id MessageID,
	direction Direction,
	sch *Schema,
	opts ...MessageOption,
) MessageConfig {
	return newMessageBuilderFromSchema(EventMessage, id, direction, sch, opts)
}

// QueryFromSchema creates a query message configuration for
// catalog.Builder.AddService from an optional pre-built payload schema.
//
// It is the schema-parameterized sibling of [Query]. sch may be nil.
func QueryFromSchema(id MessageID, sch *Schema, opts ...MessageOption) MessageConfig {
	return newMessageBuilderFromSchema(QueryMessage, id, Receives, sch, opts)
}

// newMessageBuilderFromSchema assembles a messageBuilder without reflection:
// the schema (if any) arrives pre-built, and the display name is derived from
// the message id instead of a Go type name.
func newMessageBuilderFromSchema(
	kind MessageKind,
	id MessageID,
	direction Direction,
	sch *Schema,
	opts []MessageOption,
) MessageConfig {
	msgBuilder := &messageBuilder{
		kind:      kind,
		id:        id,
		name:      humanizeMessageID(string(id)),
		version:   defaultVersion,
		schema:    sch,
		direction: direction,
	}

	for _, opt := range opts {
		opt(msgBuilder)
	}

	return msgBuilder
}

// humanizeMessageID turns a wire-style message id into a display name by
// splitting on '_', '-', and '.' and title-casing each word. Ids with no
// separators are returned title-cased verbatim ("invoices" → "Invoices").
func humanizeMessageID(id string) string {
	words := strings.FieldsFunc(id, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})

	for i, word := range words {
		runes := []rune(word)
		if len(runes) > 0 && runes[0] >= 'a' && runes[0] <= 'z' {
			runes[0] = runes[0] - 'a' + 'A'
		}

		words[i] = string(runes)
	}

	return strings.Join(words, " ")
}
