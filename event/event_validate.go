package event

import (
	"strconv"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func validateEventParams(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
	payload []byte,
) error {
	if eventType == "" {
		return errorfamily.WrapRejection(
			ErrEmptyEventType,
			"event.empty_event_type",
			"event type is required: got empty for aggregate "+aggregateID.String()+" of type "+string(
				aggregateType,
			),
		)
	}

	if aggregateID.IsZero() {
		return errorfamily.WrapRejection(
			ErrNilAggregateID,
			"event.nil_aggregate_id",
			"aggregate ID is required: for event type "+string(
				eventType,
			)+", aggregate type "+string(
				aggregateType,
			)+", version "+version.String(),
		)
	}

	if aggregateType == "" {
		return errorfamily.WrapRejection(
			ErrEmptyAggregateType,
			"event.empty_aggregate_type",
			"aggregate type is required: for aggregate "+aggregateID.String()+", event type "+string(
				eventType,
			)+", version "+version.String(),
		)
	}

	if version.IsZero() {
		return errorfamily.WrapRejection(
			ErrVersionNotPositive,
			"event.version_not_positive",
			"version must be positive: for aggregate "+aggregateID.String()+" of type "+string(
				aggregateType,
			)+" (event type "+string(
				eventType,
			)+", payload size "+strconv.Itoa(
				len(payload),
			)+")",
		)
	}

	return nil
}
