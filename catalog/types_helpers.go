package catalog

import (
	"encoding/json/jsontext"
	"time"
)

type Change struct {
	Version Version    `json:"version"`
	Date    *time.Time `json:"date,omitempty"`
	Summary Summary    `json:"summary"`
}

func (m Message) IsSend() bool { return m.Direction == Sends }

type Badge struct {
	Content         string `json:"content"`
	BackgroundColor Color  `json:"backgroundColor,omitempty"`
	TextColor       Color  `json:"textColor,omitempty"`
	Icon            Icon   `json:"icon,omitempty"`
	URL             URL    `json:"url,omitempty"`
}

type Repository struct {
	Language Language `json:"language,omitempty"`
	URL      URL      `json:"url,omitempty"`
}

type Operation struct {
	Method      Method   `json:"method"`
	Path        string   `json:"path"`
	StatusCodes []string `json:"statusCodes,omitempty"`
}

// ResponseSpec defines a typed HTTP response for a message.
// Each response has a status code, optional description, optional body schema,
// and optional examples. When Responses are present on a Message, exporters
// use them instead of fabricated default responses.
type ResponseSpec struct {
	StatusCode  string           `json:"statusCode"`
	Description string           `json:"description,omitempty"`
	Schema      *Schema          `json:"schema,omitempty"`
	Examples    []jsontext.Value `json:"examples,omitempty"`
}

// SecurityScheme defines an authentication method for the API.
// Mirrors the OpenAPI 3.0 Security Scheme Object subset.
type SecurityScheme struct {
	ID           string `json:"id"`
	Type         string `json:"type"`                   // "http", "apiKey"
	Scheme       string `json:"scheme,omitempty"`       // "bearer" for http type
	BearerFormat string `json:"bearerFormat,omitempty"` // "JWT" etc.
	In           string `json:"in,omitempty"`           // "header", "query", "cookie" for apiKey
	Name         string `json:"name,omitempty"`         // header/param name for apiKey
	Description  string `json:"description,omitempty"`
}

type Specification struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name Name   `json:"name,omitempty"`
}

type Attachment struct {
	URL         URL         `json:"url"`
	Title       Title       `json:"title,omitempty"`
	Description Description `json:"description,omitempty"`
	Type        string      `json:"type,omitempty"`
	Icon        Icon        `json:"icon,omitempty"`
}

type Ref struct {
	ID      MessageID `json:"id"`
	Version Version   `json:"version,omitempty"`
}

type ChannelParam struct {
	Enum        []string    `json:"enum,omitempty"`
	Default     string      `json:"default,omitempty"`
	Description Description `json:"description,omitempty"`
}

type ChannelRoute struct {
	ID ChannelID   `json:"id"`
	To []ChannelID `json:"to,omitempty"`
}
