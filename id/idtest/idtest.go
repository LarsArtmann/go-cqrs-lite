package idtest

import "github.com/larsartmann/go-cqrs-lite/id/v2"

// must unwraps a (value, error) pair, panicking on error. It backs every
// MustParse* helper so the public API stays typed without repeating boilerplate.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

// MustParseAggregateID converts a string to an id.AggregateID, panicking on invalid input.
func MustParseAggregateID(s string) id.AggregateID { return must(id.ParseAggregateID(s)) }

// MustParseEventID converts a string to an id.EventID, panicking on invalid input.
func MustParseEventID(s string) id.EventID { return must(id.ParseEventID(s)) }

// MustParseCorrelationID converts a string to an id.CorrelationID, panicking on invalid input.
func MustParseCorrelationID(s string) id.CorrelationID { return must(id.ParseCorrelationID(s)) }

// MustParseCausationID converts a string to an id.CausationID, panicking on invalid input.
func MustParseCausationID(s string) id.CausationID { return must(id.ParseCausationID(s)) }

// MustParseUserID converts a string to an id.UserID, panicking on invalid input.
func MustParseUserID(s string) id.UserID { return must(id.ParseUserID(s)) }

// MustParseRequestID converts a string to an id.RequestID, panicking on invalid input.
func MustParseRequestID(s string) id.RequestID { return must(id.ParseRequestID(s)) }
