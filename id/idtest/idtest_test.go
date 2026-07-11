package idtest_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
)

const validULID = "01HK1540X0841Y0A6BSX1VKR95"

func TestParse_HappyPath(t *testing.T) {
	t.Parallel()

	t.Run("AggregateID", func(t *testing.T) {
		t.Parallel()

		got := idtest.ParseAggregateID(t, validULID)
		if got.String() != validULID {
			t.Fatalf("got %q, want %q", got, validULID)
		}
	})

	t.Run("EventID", func(t *testing.T) {
		t.Parallel()

		got := idtest.ParseEventID(t, validULID)
		if got.String() != validULID {
			t.Fatalf("got %q, want %q", got, validULID)
		}
	})

	t.Run("CorrelationID", func(t *testing.T) {
		t.Parallel()

		got := idtest.ParseCorrelationID(t, validULID)
		if got.String() != validULID {
			t.Fatalf("got %q, want %q", got, validULID)
		}
	})

	t.Run("CausationID", func(t *testing.T) {
		t.Parallel()

		got := idtest.ParseCausationID(t, validULID)
		if got.String() != validULID {
			t.Fatalf("got %q, want %q", got, validULID)
		}
	})

	t.Run("UserID", func(t *testing.T) {
		t.Parallel()

		got := idtest.ParseUserID(t, validULID)
		if got.String() != validULID {
			t.Fatalf("got %q, want %q", got, validULID)
		}
	})

	t.Run("RequestID", func(t *testing.T) {
		t.Parallel()

		got := idtest.ParseRequestID(t, validULID)
		if got.String() != validULID {
			t.Fatalf("got %q, want %q", got, validULID)
		}
	})
}

// AggregateID is string-backed: any non-empty string is valid.
func TestParseAggregateID_AcceptsNonULIDString(t *testing.T) {
	t.Parallel()

	got := idtest.ParseAggregateID(t, "lock_user1_user2")
	if got.String() != "lock_user1_user2" {
		t.Fatalf("got %q, want %q", got, "lock_user1_user2")
	}
}
