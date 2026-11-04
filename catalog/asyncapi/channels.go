package asyncapi

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// channelIndex maps each message to the catalog-declared channels that carry
// it. Both directions of the association feed the index: a Channel listing
// the message in Channel.Messages, and a Message referencing the channel via
// Message.Channels (settable with [catalog.Channels]). Only one direction is
// needed for a link to exist; the config API and the resource model cannot
// disagree invisibly.
type channelIndex map[catalog.MessageID][]catalog.Channel

// buildChannelIndex indexes declared channels for export. Returns nil when
// the catalog declares no channels, leaving every message on the synthetic
// derivation path (output unchanged for channel-less catalogs).
func buildChannelIndex(cat *catalog.Catalog) channelIndex {
	if len(cat.Channels) == 0 {
		return nil
	}

	byID := make(map[catalog.ChannelID]catalog.Channel, len(cat.Channels))
	for _, ch := range cat.Channels {
		byID[ch.ID] = ch
	}

	idx := make(channelIndex)

	for _, ch := range cat.Channels {
		for _, msgID := range ch.Messages {
			idx[msgID] = appendDeclaredChannel(idx[msgID], ch)
		}
	}

	for _, svc := range cat.Services {
		messages := make([]catalog.Message, 0, len(svc.Commands)+len(svc.Events)+len(svc.Queries))
		messages = append(messages, svc.Commands...)
		messages = append(messages, svc.Events...)
		messages = append(messages, svc.Queries...)

		for _, msg := range messages {
			for _, chID := range msg.Channels {
				if ch, ok := byID[chID]; ok {
					idx[catalog.Key(msg)] = appendDeclaredChannel(idx[catalog.Key(msg)], ch)
				}
			}
		}
	}

	return idx
}

func appendDeclaredChannel(channels []catalog.Channel, ch catalog.Channel) []catalog.Channel {
	for _, existing := range channels {
		if existing.ID == ch.ID {
			return channels
		}
	}

	return append(channels, ch)
}

// declaredChannelFor returns the first declared channel carrying the message.
// Channel declaration order decides; nil index or no link falls back to the
// synthetic channel derivation.
func declaredChannelFor(idx channelIndex, messageID catalog.MessageID) (catalog.Channel, bool) {
	if len(idx) == 0 {
		return catalog.Channel{}, false
	}

	channels := idx[messageID]
	if len(channels) == 0 {
		return catalog.Channel{}, false
	}

	return channels[0], true
}

// ensureDeclaredChannel emits (or extends) the document channel for a
// catalog-declared channel, carrying the declared address, name, summary, and
// delivery guarantee instead of the synthetic per-message derivation. A
// declared channel may carry several messages, so the message ref is appended
// to whatever the channel already holds.
func ensureDeclaredChannel(doc *Document, ch catalog.Channel, componentKey, ref string) {
	key := string(ch.ID)

	existing, ok := doc.Channels[key]
	if !ok {
		description := string(ch.Summary)

		if guarantee := string(ch.DeliveryGuarantee); guarantee != "" {
			description = strings.TrimSpace(
				description + " Delivery guarantee: " + guarantee + ".",
			)
		}

		existing = Channel{
			Address:     cmp.Or(string(ch.Address), key),
			Title:       cmp.Or(string(ch.Name), key),
			Description: description,
			Messages:    map[string]Ref{},
		}
	}

	if existing.Messages == nil {
		existing.Messages = map[string]Ref{}
	}

	existing.Messages[componentKey] = Ref{Ref: ref}
	doc.Channels[key] = existing
}

// channelFor resolves the channel key and emission strategy for a message:
// a declared channel wins (its ID becomes the document channel key), else the
// synthetic kind.message key with the synthetic channel shape.
func channelFor(
	idx channelIndex,
	doc *Document,
	owner string,
	msg catalog.Message,
	kind messageKind,
	messageID catalog.MessageID,
) (string, func(componentKey, ref string)) {
	if ch, ok := declaredChannelFor(idx, messageID); ok {
		return string(ch.ID), func(componentKey, ref string) {
			ensureDeclaredChannel(doc, ch, componentKey, ref)
		}
	}

	channelKey := string(kind) + "." + string(messageID)

	return channelKey, func(componentKey, ref string) {
		ensureSyntheticChannel(doc, owner, msg, kind, channelKey, componentKey, ref)
	}
}
