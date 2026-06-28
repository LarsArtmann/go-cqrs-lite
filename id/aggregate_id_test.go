package id_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestParseAggregateIDStrict_ValidULID(t *testing.T) {
	t.Parallel()

	original := id.NewAggregateID()
	parsed, err := id.ParseAggregateIDStrict(original.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed != original {
		t.Errorf("parsed %q != original %q", parsed, original)
	}
}

func TestParseAggregateIDStrict_InvalidULID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"domain-specific ID", "lock_user1_user2"},
		{"SHA-256 hash", "a1b2c3d4e5f6"},
		{"random garbage", "not-a-ulid"},
		{"too short", "01H"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := id.ParseAggregateIDStrict(tt.input)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tt.input)
			}
		})
	}
}

func TestParseAggregateIDStrict_Empty(t *testing.T) {
	t.Parallel()

	_, err := id.ParseAggregateIDStrict("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestParseAggregateID_LenientAcceptsNonULID(t *testing.T) {
	t.Parallel()

	got, err := id.ParseAggregateID("lock_user1_user2")
	if err != nil {
		t.Fatalf("lenient parse should accept non-ULID: %v", err)
	}

	if got.String() != "lock_user1_user2" {
		t.Errorf("got %q, want lock_user1_user2", got)
	}
}

func TestIsAggregateIDULID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   id.AggregateID
		want bool
	}{
		{"NewAggregateID", id.NewAggregateID(), true},
		{"parsed ULID", mustParseAgg(t, id.NewAggregateID().String()), true},
		{"derived SHA-256", id.DeriveAggregateID("ns", "key"), false},
		{"domain string", mustParseAgg(t, "lock_user1"), false},
		{"empty", id.AggregateID{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := id.IsAggregateIDULID(tt.id)
			if got != tt.want {
				t.Errorf("IsAggregateIDULID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestAggregateTimestamp_ValidULID(t *testing.T) {
	t.Parallel()

	before := time.Now()
	aggID := id.NewAggregateID()
	after := time.Now()

	ts, err := id.AggregateTimestamp(aggID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ts.Before(before.Add(-time.Millisecond)) || ts.After(after.Add(time.Millisecond)) {
		t.Errorf("timestamp %v not in expected range [%v, %v]", ts, before, after)
	}
}

func TestAggregateTimestamp_NotULID(t *testing.T) {
	t.Parallel()

	derived := id.DeriveAggregateID("lock", "user1")
	_, err := id.AggregateTimestamp(derived)
	if err == nil {
		t.Fatal("expected error for non-ULID AggregateID")
	}
}

func TestAggregateTimestamp_Empty(t *testing.T) {
	t.Parallel()

	_, err := id.AggregateTimestamp(id.AggregateID{})
	if err == nil {
		t.Fatal("expected error for empty AggregateID")
	}
}

func mustParseAgg(t *testing.T, s string) id.AggregateID {
	t.Helper()

	id, err := id.ParseAggregateID(s)
	if err != nil {
		t.Fatalf("ParseAggregateID(%q): %v", s, err)
	}

	return id
}
