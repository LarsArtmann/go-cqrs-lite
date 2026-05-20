package aggregate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestNewCore_Errors(t *testing.T) {
	t.Parallel()

	_, err := aggregate.NewCore(id.AggregateID{}, "User")
	if err == nil {
		t.Fatal("expected error for zero ID")
	}

	_, err = aggregate.NewCore(id.NewAggregateID(), "")
	if err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestMustNewCore_Panics(t *testing.T) {
	t.Parallel()

	defer func() { _ = recover() }()

	aggregate.MustNewCore(id.AggregateID{}, "User")
	t.Fatal("expected panic for zero ID")
}
