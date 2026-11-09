package asyncapi

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// singularKind renders a message kind as a singular noun for channel titles
// (commands → command, queries → query). strings.TrimSuffix cannot do this:
// "queries" minus "s" is "querie".
func singularKind(kind messageKind) string {
	switch kind {
	case kindCommand:
		return "command"
	case kindEvent:
		return "event"
	case kindQuery:
		return "query"
	default:
		return string(kind)
	}
}

// queryReplyFor returns the request/reply projection for a query operation,
// nil for every other kind. Replies go to the per-request address carried in
// the message header ($message.header#/replyTo) on a dedicated reply channel;
// the reply payload schema is intentionally opaque because the catalog does
// not model query response types. Refs follow the scoping contract documented
// on [Reply]: messages point into the reply channel's own messages map.
func queryReplyFor(
	doc *Document,
	serviceID catalog.ServiceID,
	msg catalog.Message,
	kind messageKind,
	messageID catalog.MessageID,
) *Reply {
	if kind != kindQuery {
		return nil
	}

	replyChannelKey := string(kindQuery) + "." + string(messageID) + ".replies"
	replyComponentKey := string(catalog.QueryMessage) + "." + string(messageID) + ".reply"

	ensureReplyChannel(doc, serviceID, msg, messageID, replyChannelKey, replyComponentKey)
	ensureReplyMessage(doc, msg, messageID, replyComponentKey)

	return &Reply{ //nolint:exhaustruct_v5 // optional fields omitted by design
		Address:  &ReplyAddress{Location: "$message.header#/replyTo"},
		Channel:  Ref{Ref: "#/channels/" + replyChannelKey},
		Messages: []Ref{{Ref: "#/channels/" + replyChannelKey + "/messages/" + replyComponentKey}},
	}
}

func ensureReplyChannel(
	doc *Document,
	serviceID catalog.ServiceID,
	msg catalog.Message,
	messageID catalog.MessageID,
	replyChannelKey, replyComponentKey string,
) {
	if _, exists := doc.Channels[replyChannelKey]; exists {
		return
	}

	doc.Channels[replyChannelKey] = Channel{
		Address: fmt.Sprintf(
			"%s.%s.%s.replies",
			serviceID,
			kindQuery,
			dotSeparated(string(messageID)),
		),
		Title: string(msg.Name) + " Reply Channel",
		Messages: map[string]Ref{
			replyComponentKey: {Ref: "#/components/messages/" + replyComponentKey},
		},
	}
}

func ensureReplyMessage(
	doc *Document,
	msg catalog.Message,
	messageID catalog.MessageID,
	replyComponentKey string,
) {
	if _, exists := doc.Components.Messages[replyComponentKey]; exists {
		return
	}

	doc.Components.Messages[replyComponentKey] = Message{
		Name:        string(messageID),
		Title:       string(msg.Name) + " Response",
		ContentType: contentType,
		Payload:     Ref{Ref: "#/components/schemas/" + replyComponentKey},
		Tags:        []Tag{{Name: kindToTagName(catalog.QueryMessage)}},
	}

	doc.Components.Schemas[replyComponentKey] = SchemaToAny(nil)
}
