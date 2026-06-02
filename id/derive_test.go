package id_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
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
