package querytest

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func New(tb testing.TB, queryType query.Type) *query.BasicQuery {
	tb.Helper()

	q, err := query.New(queryType)
	if err != nil {
		tb.Fatalf("querytest: new query %q: %v", queryType, err)
	}

	return q
}
