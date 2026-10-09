package querytest

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/v5/query"
)

func New(tb testing.TB, queryType query.Type) *query.BasicQuery {
	//art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
	tb.Helper()

	q, err := query.New(queryType)
	if err != nil {
		tb.Fatalf("querytest: new query %q: %v", queryType, err)
	}

	return q
}
