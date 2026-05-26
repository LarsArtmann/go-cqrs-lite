package catalog

import (
	"time"
)

type Change struct {
	Version string     `json:"version"`
	Date    *time.Time `json:"date,omitempty"`
	Summary string     `json:"summary"`
}

func (m Message) IsSend() bool { return m.Direction == Sends }

type Badge struct {
	Content         string `json:"content"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
	TextColor       string `json:"textColor,omitempty"`
	Icon            string `json:"icon,omitempty"`
	URL             string `json:"url,omitempty"`
}

type Repository struct {
	Language string `json:"language,omitempty"`
	URL      string `json:"url,omitempty"`
}

type Operation struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	StatusCodes []string `json:"statusCodes,omitempty"`
}

type Specification struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

type Attachment struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

type Ref struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

type ChannelParam struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
}

type ChannelRoute struct {
	ID ChannelID   `json:"id"`
	To []ChannelID `json:"to,omitempty"`
}
