package schema

import (
	"encoding/json/jsontext"
)

type Type string

const (
	TypeString  Type = "string"
	TypeObject  Type = "object"
	TypeInteger Type = "integer"
	TypeNumber  Type = "number"
	TypeBoolean Type = "boolean"
	TypeArray   Type = "array"
	TypeNull    Type = "null"
)

type Schema struct {
	Type       Type                `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
	Items      *Property           `json:"items,omitempty"`
	Examples   []jsontext.Value    `json:"examples,omitempty"`
	Parameters []Parameter         `json:"-"`
}

// Parameter represents an HTTP parameter extracted from struct tags
// (query, path, header, cookie). These are not serialized to JSON schema
// output but are consumed by OpenAPI exporters.
type Parameter struct {
	Name        string
	In          string // "query", "path", "header", "cookie"
	Description string
	Required    bool
	Schema      *Property
}

type Property struct {
	Type        Type                `json:"type"`
	Description string              `json:"description,omitempty"`
	Format      string              `json:"format,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Items       *Property           `json:"items,omitempty"`
	Required    []string            `json:"required,omitempty"`
	Default     string              `json:"default,omitempty"`
	Enum        []string            `json:"enum,omitempty"`
	Nullable    bool                `json:"nullable,omitempty"`
	Deprecated  bool                `json:"deprecated,omitempty"`
	Pattern     string              `json:"pattern,omitempty"`
	Examples    []jsontext.Value    `json:"examples,omitempty"`
}
