package event

import (
	"strconv"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func validateEventParams(
	eventType Type,
	streamID id.StreamID,
	streamType id.StreamType,
	version Version,
	payload []byte,
) error {
	if eventType == "" {
		return errorfamily.WrapRejection(
			ErrEmptyEventType,
			"event.empty_event_type",
			"event type is required: got empty for stream "+streamID.String()+" of type "+string(
				streamType,
			),
		)
	}

	if streamID.IsZero() {
		return errorfamily.WrapRejection(
			ErrNilStreamID,
			"event.nil_aggregate_id",
			"stream ID is required: for event type "+string(
				eventType,
			)+", stream type "+string(
				streamType,
			)+", version "+version.String(),
		)
	}

	if streamType == "" {
		return errorfamily.WrapRejection(
			ErrEmptyStreamType,
			"event.empty_aggregate_type",
			"stream type is required: for stream "+streamID.String()+", event type "+string(
				eventType,
			)+", version "+version.String(),
		)
	}

	if version.IsZero() {
		return errorfamily.WrapRejection(
			ErrVersionNotPositive,
			"event.version_not_positive",
			"version must be positive: for stream "+streamID.String()+" of type "+string(
				streamType,
			)+" (event type "+string(
				eventType,
			)+", payload size "+strconv.Itoa(
				len(payload),
			)+")",
		)
	}

	return nil
}
