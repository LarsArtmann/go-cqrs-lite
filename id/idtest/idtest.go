package idtest

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func parse[T any](tb testing.TB, s string, fn func(string) (T, error)) T {
	tb.Helper()

	v, err := fn(s)
	if err != nil {
		tb.Fatalf("idtest: parse %q: %v", s, err)
	}

	return v
}

func ParseStreamID(tb testing.TB, s string) id.StreamID {
	tb.Helper()

	return parse(tb, s, id.ParseStreamID)
}

// Deprecated: use ParseStreamID.
func ParseAggregateID(tb testing.TB, s string) id.StreamID {
	tb.Helper()

	return ParseStreamID(tb, s)
}

func ParseEventID(tb testing.TB, s string) id.EventID {
	tb.Helper()

	return parse(tb, s, id.ParseEventID)
}

func ParseCorrelationID(tb testing.TB, s string) id.CorrelationID {
	tb.Helper()

	return parse(tb, s, id.ParseCorrelationID)
}

func ParseCausationID(tb testing.TB, s string) id.CausationID {
	tb.Helper()

	return parse(tb, s, id.ParseCausationID)
}

func ParseUserID(tb testing.TB, s string) id.UserID {
	tb.Helper()

	return parse(tb, s, id.ParseUserID)
}

func ParseRequestID(tb testing.TB, s string) id.RequestID {
	tb.Helper()

	return parse(tb, s, id.ParseRequestID)
}
