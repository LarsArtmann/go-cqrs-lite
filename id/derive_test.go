package id_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestDeriveAggregateID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		a, b   id.AggregateID
		wantEq bool
	}{
		{
			"deterministic",
			id.DeriveAggregateID("lock", "user1", "resource1"),
			id.DeriveAggregateID("lock", "user1", "resource1"),
			true,
		},
		{
			"different inputs",
			id.DeriveAggregateID("lock", "user1", "resource1"),
			id.DeriveAggregateID("lock", "user2", "resource1"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.a == tt.b; got != tt.wantEq {
				t.Errorf("a == b = %v, want %v", got, tt.wantEq)
			}
		})
	}
}

func TestDeriveAggregateID_DifferentNamespace(t *testing.T) {
	t.Parallel()

	a := id.DeriveAggregateID("lock", "user1")
	b := id.DeriveAggregateID("unlock", "user1")

	if a == b {
		t.Error("different namespaces should produce different IDs")
	}
}

func TestDeriveAggregateID_NotZero(t *testing.T) {
	t.Parallel()

	got := id.DeriveAggregateID("test")

	if got.IsZero() {
		t.Error("derived ID should not be zero")
	}

	if got.String() == "" {
		t.Error("derived ID should not be empty string")
	}
}

type mockStringer struct{ s string }

func (m mockStringer) String() string { return m.s }

func TestAggregateIDFrom(t *testing.T) {
	t.Parallel()

	s := mockStringer{s: "custom-domain-id-123"}
	got := id.AggregateIDFrom(s)

	if got.String() != "custom-domain-id-123" {
		t.Errorf("got %q, want custom-domain-id-123", got.String())
	}
}

func TestDeriveCommandID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		a, b   id.CommandID
		wantEq bool
	}{
		{
			"deterministic",
			id.DeriveCommandID("deriver", "evt-123", "0"),
			id.DeriveCommandID("deriver", "evt-123", "0"),
			true,
		},
		{
			"different index",
			id.DeriveCommandID("deriver", "evt-123", "0"),
			id.DeriveCommandID("deriver", "evt-123", "1"),
			false,
		},
		{
			"different event",
			id.DeriveCommandID("deriver", "evt-123", "0"),
			id.DeriveCommandID("deriver", "evt-456", "0"),
			false,
		},
		{
			"different namespace",
			id.DeriveCommandID("deriver", "evt-123"),
			id.DeriveCommandID("saga", "evt-123"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.a == tt.b; got != tt.wantEq {
				t.Errorf("a == b = %v, want %v", got, tt.wantEq)
			}
		})
	}
}

func TestDeriveCommandID_IsValidULID(t *testing.T) {
	t.Parallel()

	got := id.DeriveCommandID("deriver", "evt-123", "0")

	if got.IsZero() {
		t.Fatal("derived command ID should not be zero")
	}

	// Derived CommandIDs are ULID-encoded (first 16 bytes of SHA-256 packed
	// into a ulid.ULID), so the string form must be a parseable ULID.
	parsed, err := id.ParseCommandID(got.String())
	if err != nil {
		t.Errorf(
			"derived command ID %q should round-trip through ParseCommandID: %v",
			got.String(),
			err,
		)
	}

	if parsed != got {
		t.Errorf("round-trip mismatch: %v != %v", parsed, got)
	}
}

func TestDeriveCommandID_ZeroTimestamp(t *testing.T) {
	t.Parallel()

	derived := id.DeriveCommandID("deriver", "evt-123", "0")
	fresh := id.NewCommandID()

	// Derived IDs must have a zero timestamp (epoch sentinel).
	if !id.IsDerivedCommandID(derived) {
		t.Error("DeriveCommandID result should be detected as derived")
	}

	if id.IsDerivedCommandID(fresh) {
		t.Error("NewCommandID result should NOT be detected as derived")
	}

	// id.ULID on a derived ID must return Unix epoch (1970-01-01), not garbage.
	ts := id.ULID[id.CommandMarker](derived)
	epoch := time.Unix(0, 0)
	if !ts.Equal(epoch) {
		t.Errorf("derived CommandID timestamp should be Unix epoch (1970), got %v", ts)
	}
}
