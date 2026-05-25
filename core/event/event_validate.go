package event

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func validateEventParams(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
	payload []byte,
) error {
	if eventType == "" {
		return fmt.Errorf(
			"%w: got empty for aggregate %q of type %q",
			ErrEmptyEventType,
			aggregateID,
			aggregateType,
		)
	}

	if aggregateID.IsZero() {
		return fmt.Errorf(
			"%w: for event type %q, aggregate type %q, version %d",
			ErrNilAggregateID,
			eventType,
			aggregateType,
			version,
		)
	}

	if aggregateType == "" {
		return fmt.Errorf(
			"%w: for aggregate %q, event type %q, version %d",
			ErrEmptyAggregateType,
			aggregateID,
			eventType,
			version,
		)
	}

	if version.IsZero() {
		return fmt.Errorf(
			"%w: for aggregate %q of type %q (event type %q, payload size %d)",
			ErrVersionNotPositive,
			aggregateID,
			aggregateType,
			eventType,
			len(payload),
		)
	}

	return nil
}
