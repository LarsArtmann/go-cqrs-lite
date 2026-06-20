package memory_test

import (
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func parseAggID(s string) id.AggregateID {
	v, err := id.ParseAggregateID(s)
	if err != nil {
		panic(err)
	}

	return v
}
