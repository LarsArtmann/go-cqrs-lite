package query_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

func TestQueryErrors_Classification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want event.Family
	}{
		{"ErrHandlerNotFound", query.ErrHandlerNotFound, event.Rejection},
		{"ErrDispatcherClosed", query.ErrDispatcherClosed, event.Infrastructure},
		{"ErrEmptyQueryType", query.ErrEmptyQueryType, event.Rejection},
		{"ErrTypeAssertion", query.ErrTypeAssertion, event.Rejection},
		{"ErrStoreClosed", query.ErrStoreClosed, event.Infrastructure},
		{"ErrQueryNotFound", query.ErrQueryNotFound, event.Rejection},
		{"ErrDuplicateQuery", query.ErrDuplicateQuery, event.Conflict},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := event.Classify(tc.err); got != tc.want {
				t.Fatalf("Classify(%s) = %s, want %s", tc.name, got, tc.want)
			}
		})
	}
}

func TestQueryErrors_ErrorString(t *testing.T) {
	t.Parallel()

	if query.ErrHandlerNotFound.Error() == "" {
		t.Fatal("ErrHandlerNotFound should have non-empty Error() string")
	}
}
