// Package httptyped provides generic typed request/response envelopes for
// consumers who want type-safe HTTP contracts without adopting a framework.
//
// This package is optional. The core catalog package does not depend on it.
// It is a thin convenience layer over catalog.Response[T] and catalog.Command[T].
package httptyped

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

// RequestSchema derives a catalog schema from a Go struct type T.
// This is useful when you need the schema outside of a message builder.
type RequestSchema[In any] struct {
	schema *catalog.Schema
}

func NewRequestSchema[In any]() RequestSchema[In] {
	return RequestSchema[In]{schema: schema.FromType[In]()}
}

func (r RequestSchema[In]) Schema() *catalog.Schema { return r.schema }

// ResponseSchema derives a catalog schema from a Go struct type T.
type ResponseSchema[Out any] struct {
	schema *catalog.Schema
}

func NewResponseSchema[Out any]() ResponseSchema[Out] {
	return ResponseSchema[Out]{schema: schema.FromType[Out]()}
}

func (r ResponseSchema[Out]) Schema() *catalog.Schema { return r.schema }

func (r ResponseSchema[Out]) ToResponseSpec(statusCode, description string) catalog.ResponseSpec {
	return catalog.ResponseSpec{
		StatusCode:  statusCode,
		Description: description,
		Schema:      r.schema,
	}
}

// Command creates a command message with a typed request body.
// Additional options (responses, operation, security) can be appended.
// This is equivalent to catalog.Command[In] but re-exported here for
// consumers who import only httptyped.
func Command[In any](id catalog.MessageID, opts ...catalog.MessageOption) catalog.MessageConfig {
	return catalog.Command[In](id, opts...)
}

// Query creates a query message with a typed request body.
func Query[In any](id catalog.MessageID, opts ...catalog.MessageOption) catalog.MessageConfig {
	return catalog.Query[In](id, opts...)
}

// OKResponse adds a typed 200 response with a schema derived from Out.
func OKResponse[Out any](description string) catalog.MessageOption {
	return catalog.Response[Out]("200", description)
}

// CreatedResponse adds a typed 201 response with a schema derived from Out.
func CreatedResponse[Out any](description string) catalog.MessageOption {
	return catalog.Response[Out]("201", description)
}

// ErrorResponse adds a typed error response (default 400) with a schema derived from Out.
func ErrorResponse[Out any](statusCode, description string) catalog.MessageOption {
	return catalog.Response[Out](statusCode, description)
}
