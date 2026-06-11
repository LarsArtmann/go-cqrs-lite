package openapi

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
)

// Document represents an OpenAPI 3.0 specification document.
type Document struct {
	OpenAPI    string                 `json:"openapi"`
	Info       catalog.DocumentInfo   `json:"info"`
	Servers    []Server               `json:"servers,omitempty"`
	Paths      map[string]*PathItem   `json:"paths"`
	Components Components             `json:"components"`
	Tags       []Tag                  `json:"tags,omitempty"`
}

// Server describes a server URL.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Tag is used for grouping operations.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PathItem describes operations available on a single path.
type PathItem struct {
	Summary     string      `json:"summary,omitempty"`
	Description string      `json:"description,omitempty"`
	Post        *Operation  `json:"post,omitempty"`
	Get         *Operation  `json:"get,omitempty"`
	Put         *Operation  `json:"put,omitempty"`
	Delete      *Operation  `json:"delete,omitempty"`
	Patch       *Operation  `json:"patch,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty"`
}

// Operation describes a single API operation.
type Operation struct {
	Tags        []string             `json:"tags,omitempty"`
	Summary     string               `json:"summary,omitempty"`
	Description string               `json:"description,omitempty"`
	OperationID string               `json:"operationId,omitempty"`
	Deprecated  bool                 `json:"deprecated,omitempty"`
	RequestBody *RequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]*Response `json:"responses"`
	Parameters  []Parameter          `json:"parameters,omitempty"`
}

// RequestBody describes a request body.
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required,omitempty"`
}

// Response describes a response.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// MediaType describes the media type of a request or response body.
type MediaType struct {
	Schema  any `json:"schema,omitempty"`
	Example any `json:"example,omitempty"`
}

// Parameter describes an operation parameter.
type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Schema      any    `json:"schema,omitempty"`
}

// Components holds reusable schemas and responses.
type Components struct {
	Schemas map[string]any `json:"schemas"`
}

// SchemaRef is a reference to a schema in components.
type SchemaRef struct {
	Ref string `json:"$ref"`
}
