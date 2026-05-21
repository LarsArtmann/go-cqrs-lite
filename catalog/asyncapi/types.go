package asyncapi

// Document represents an AsyncAPI 3.0 specification document.
type Document struct {
	AsyncAPI           string               `json:"asyncapi"                     yaml:"asyncapi"`
	ID                 string               `json:"id,omitempty"                 yaml:"id,omitempty"`
	Info               Info                 `json:"info"                         yaml:"info"`
	DefaultContentType string               `json:"defaultContentType,omitempty" yaml:"defaultContentType,omitempty"`
	Servers            map[string]Server    `json:"servers,omitempty"            yaml:"servers,omitempty"`
	Channels           map[string]Channel   `json:"channels"                     yaml:"channels"`
	Operations         map[string]Operation `json:"operations"                   yaml:"operations"`
	Components         Components           `json:"components"                   yaml:"components"`
}

// Info contains metadata about the AsyncAPI document.
type Info struct {
	Title       string `json:"title"                 yaml:"title"`
	Version     string `json:"version"               yaml:"version"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Server describes a server connection in the AsyncAPI specification.
type Server struct {
	Host            string `json:"host"                      yaml:"host"`
	Protocol        string `json:"protocol"                  yaml:"protocol"`
	ProtocolVersion string `json:"protocolVersion,omitempty" yaml:"protocolVersion,omitempty"`
	Description     string `json:"description,omitempty"     yaml:"description,omitempty"`
	Tags            []Tag  `json:"tags,omitempty"            yaml:"tags,omitempty"`
}

// Tag is a label attached to a server or operation for categorization.
type Tag struct {
	Name string `json:"name" yaml:"name"`
}

// Channel represents a messaging channel with an address and associated messages.
type Channel struct {
	Address     string         `json:"address"               yaml:"address"`
	Title       string         `json:"title,omitempty"       yaml:"title,omitempty"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	Messages    map[string]Ref `json:"messages"              yaml:"messages"`
}

// Ref is a JSON Reference pointing to another component in the document.
type Ref struct {
	Ref string `json:"$ref" yaml:"$ref"`
}

// Operation describes a send or receive action on a channel.
type Operation struct {
	Title    string `json:"title,omitempty"   yaml:"title,omitempty"`
	Summary  string `json:"summary,omitempty" yaml:"summary,omitempty"`
	Action   string `json:"action"            yaml:"action"`
	Channel  Ref    `json:"channel"           yaml:"channel"`
	Messages []Ref  `json:"messages"          yaml:"messages"`
	Tags     []Tag  `json:"tags,omitempty"    yaml:"tags,omitempty"`
	Reply    *Reply `json:"reply,omitempty"   yaml:"reply,omitempty"`
}

// Reply describes the reply configuration for a request-reply operation.
type Reply struct {
	Address  *ReplyAddress `json:"address,omitempty" yaml:"address,omitempty"`
	Channel  Ref           `json:"channel"           yaml:"channel"`
	Messages []Ref         `json:"messages"          yaml:"messages"`
}

// ReplyAddress specifies where reply messages should be sent.
type ReplyAddress struct {
	Location string `json:"location" yaml:"location"`
}

// Components holds reusable schemas and message definitions.
type Components struct {
	Schemas  map[string]any     `json:"schemas"  yaml:"schemas"`
	Messages map[string]Message `json:"messages" yaml:"messages"`
}

// Message describes a message type including its schema reference and metadata.
type Message struct {
	Name        string    `json:"name"                 yaml:"name"`
	Title       string    `json:"title"                yaml:"title"`
	Summary     string    `json:"summary,omitempty"    yaml:"summary,omitempty"`
	ContentType string    `json:"contentType"          yaml:"contentType"`
	Payload     Ref       `json:"payload"              yaml:"payload"`
	Tags        []Tag     `json:"tags,omitempty"       yaml:"tags,omitempty"`
	Examples    []Example `json:"examples,omitempty"   yaml:"examples,omitempty"`
	Deprecated  bool      `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}

// Example provides an example payload for a message.
type Example struct {
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`
	Payload any    `json:"payload,omitempty" yaml:"payload,omitempty"`
}
