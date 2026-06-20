package idtest_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v2/idtest"
)

// AggregateID is string-backed: only the empty string is invalid.
// The ULID-backed types additionally reject malformed ULID strings.
func TestMustParse_PanicsOnInvalidInput(t *testing.T) {
	t.Parallel()

	ulidOnly := []struct {
		name string
		fn   func(string)
	}{
		{"EventID", func(s string) { _ = idtest.MustParseEventID(s) }},
		{"CorrelationID", func(s string) { _ = idtest.MustParseCorrelationID(s) }},
		{"CausationID", func(s string) { _ = idtest.MustParseCausationID(s) }},
		{"UserID", func(s string) { _ = idtest.MustParseUserID(s) }},
		{"RequestID", func(s string) { _ = idtest.MustParseRequestID(s) }},
	}

	for _, tc := range ulidOnly {
		t.Run(tc.name+"/panics", func(t *testing.T) {
			t.Parallel()

			for _, bad := range []string{"", "not-a-ulid", "tooShort"} {
				if !didPanic(func() { tc.fn(bad) }) {
					t.Fatalf("expected panic for input %q", bad)
				}
			}
		})
	}

	t.Run("AggregateID/panics_on_empty", func(t *testing.T) {
		t.Parallel()

		if !didPanic(func() { _ = idtest.MustParseAggregateID("") }) {
			t.Fatal("expected panic for empty AggregateID")
		}
	})

	t.Run("AggregateID/accepts_non_ulid_string", func(t *testing.T) {
		t.Parallel()

		// AggregateID is string-backed: any non-empty string is valid.
		if didPanic(func() { _ = idtest.MustParseAggregateID("lock_user1_user2") }) {
			t.Fatal("must not panic for non-empty string AggregateID")
		}
	})
}

const validULID = "01HK1540X0841Y0A6BSX1VKR95"

func TestMustParse_HappyPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		fn   func(string)
	}{
		{"AggregateID", func(s string) { _ = idtest.MustParseAggregateID(s) }},
		{"EventID", func(s string) { _ = idtest.MustParseEventID(s) }},
		{"CorrelationID", func(s string) { _ = idtest.MustParseCorrelationID(s) }},
		{"CausationID", func(s string) { _ = idtest.MustParseCausationID(s) }},
		{"UserID", func(s string) { _ = idtest.MustParseUserID(s) }},
		{"RequestID", func(s string) { _ = idtest.MustParseRequestID(s) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.fn(validULID) // must not panic on a valid ULID
		})
	}
}

func didPanic(fn func()) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	fn()

	return panicked
}
