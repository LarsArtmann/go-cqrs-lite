package query_test

import (
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func TestQueryErrors_Classification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want errorfamily.Family
	}{
		{"ErrHandlerNotFound", query.ErrHandlerNotFound, errorfamily.Rejection},
		{"ErrDispatcherClosed", query.ErrDispatcherClosed, errorfamily.Infrastructure},
		{"ErrEmptyQueryType", query.ErrEmptyQueryType, errorfamily.Rejection},
		{"ErrTypeAssertion", query.ErrTypeAssertion, errorfamily.Rejection},
		{"ErrStoreClosed", query.ErrStoreClosed, errorfamily.Infrastructure},
		{"ErrQueryNotFound", query.ErrQueryNotFound, errorfamily.Rejection},
		{"ErrDuplicateQuery", query.ErrDuplicateQuery, errorfamily.Conflict},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := errorfamily.Classify(tc.err); got != tc.want {
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
