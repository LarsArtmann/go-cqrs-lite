package event_test

import "github.com/larsartmann/go-cqrs-lite/id/v2"

func parseAggID(s string) id.AggregateID {
	v, err := id.ParseAggregateID(s)
	if err != nil {
		panic(err)
	}

	return v
}

func parseCorrID(s string) id.CorrelationID {
	v, err := id.ParseCorrelationID(s)
	if err != nil {
		panic(err)
	}

	return v
}
