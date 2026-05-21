package id_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestDeriveAggregateID_Deterministic(t *testing.T) {
	t.Parallel()

	a := id.DeriveAggregateID("lock", "user1", "resource1")
	b := id.DeriveAggregateID("lock", "user1", "resource1")

	if a != b {
		t.Error("same inputs should produce same ID")
	}
}

func TestDeriveAggregateID_DifferentInputs(t *testing.T) {
	t.Parallel()

	a := id.DeriveAggregateID("lock", "user1", "resource1")
	b := id.DeriveAggregateID("lock", "user2", "resource1")

	if a == b {
		t.Error("different inputs should produce different IDs")
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
