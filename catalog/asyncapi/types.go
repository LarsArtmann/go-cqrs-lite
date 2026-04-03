package asyncapi

type Document struct {
	AsyncAPI           string               `yaml:"asyncapi"                     json:"asyncapi"`
	ID                 string               `yaml:"id,omitempty"                 json:"id,omitempty"`
	Info               Info                 `yaml:"info"                         json:"info"`
	DefaultContentType string               `yaml:"defaultContentType,omitempty" json:"defaultContentType,omitempty"`
	Servers            map[string]Server    `yaml:"servers,omitempty"            json:"servers,omitempty"`
	Channels           map[string]Channel   `yaml:"channels"                     json:"channels"`
	Operations         map[string]Operation `yaml:"operations"                   json:"operations"`
	Components         Components           `yaml:"components"                   json:"components"`
}

type Info struct {
	Title       string `yaml:"title"                 json:"title"`
	Version     string `yaml:"version"               json:"version"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

type Server struct {
	Host            string `yaml:"host"                      json:"host"`
	Protocol        string `yaml:"protocol"                  json:"protocol"`
	ProtocolVersion string `yaml:"protocolVersion,omitempty" json:"protocolVersion,omitempty"`
	Description     string `yaml:"description,omitempty"     json:"description,omitempty"`
	Tags            []Tag  `yaml:"tags,omitempty"            json:"tags,omitempty"`
}

type Tag struct {
	Name string `yaml:"name" json:"name"`
}

type Channel struct {
	Address     string         `yaml:"address"               json:"address"`
	Title       string         `yaml:"title,omitempty"       json:"title,omitempty"`
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	Messages    map[string]Ref `yaml:"messages"              json:"messages"`
}

type Ref struct {
	Ref string `yaml:"$ref" json:"$ref"`
}

type Operation struct {
	Title    string `yaml:"title,omitempty"   json:"title,omitempty"`
	Summary  string `yaml:"summary,omitempty" json:"summary,omitempty"`
	Action   string `yaml:"action"            json:"action"`
	Channel  Ref    `yaml:"channel"           json:"channel"`
	Messages []Ref  `yaml:"messages"          json:"messages"`
	Tags     []Tag  `yaml:"tags,omitempty"    json:"tags,omitempty"`
	Reply    *Reply `yaml:"reply,omitempty"   json:"reply,omitempty"`
}

type Reply struct {
	Address  *ReplyAddress `yaml:"address,omitempty" json:"address,omitempty"`
	Channel  Ref           `yaml:"channel"           json:"channel"`
	Messages []Ref         `yaml:"messages"          json:"messages"`
}

type ReplyAddress struct {
	Location string `yaml:"location" json:"location"`
}

type Components struct {
	Schemas  map[string]any     `yaml:"schemas"  json:"schemas"`
	Messages map[string]Message `yaml:"messages" json:"messages"`
}

type Message struct {
	Name        string `yaml:"name"              json:"name"`
	Title       string `yaml:"title"             json:"title"`
	Summary     string `yaml:"summary,omitempty" json:"summary,omitempty"`
	ContentType string `yaml:"contentType"       json:"contentType"`
	Payload     Ref    `yaml:"payload"           json:"payload"`
	Tags        []Tag  `yaml:"tags,omitempty"    json:"tags,omitempty"`
}
