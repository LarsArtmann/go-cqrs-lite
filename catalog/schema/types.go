package schema

import "encoding/json"

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
	Examples   []json.RawMessage   `json:"examples,omitempty"`
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
	Examples    []json.RawMessage   `json:"examples,omitempty"`
}
