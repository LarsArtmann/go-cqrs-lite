package watermill

import (
	"github.com/ThreeDotsLabs/watermill/message"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// streamIDFromMessage reads the stream ID from message metadata, honoring the
// dual-read window: messages written before the stream vocabulary rename carry
// only the legacy aggregate spelling (v6: drop the fallback).
func streamIDFromMessage(md message.Metadata) (id.StreamID, error) {
	streamIDStr := md.Get(metaStreamID)
	if streamIDStr == "" {
		streamIDStr = md.Get(metaLegacyAggregateID)
	}
	streamID, err := id.ParseStreamID(streamIDStr)
	if err != nil {
		return id.StreamID{}, errorfamily.WrapRejection(err,
			"watermill.parse_stream_id_failed", "parse stream_id")
	}
	return streamID, nil
}
