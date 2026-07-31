package stack_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func TestDurability_Default(t *testing.T) {
	t.Parallel()

	b, err := stack.New(
		stack.WithEventStore(eventtest.NewFakeStore()),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	if got := b.Durability(); got != stack.DurabilityNormal {
		t.Fatalf("default Durability() = %q, want %q", got, stack.DurabilityNormal)
	}
}

func TestDurability_WithDurability(t *testing.T) {
	t.Parallel()

	for _, tier := range []stack.DurabilityTier{
		stack.DurabilityStrict,
		stack.DurabilityNormal,
		stack.DurabilityRelaxed,
	} {
		t.Run(string(tier), func(t *testing.T) {
			t.Parallel()

			b, err := stack.New(
				stack.WithEventStore(eventtest.NewFakeStore()),
				stack.WithDurability(tier),
			)
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			defer func() { _ = b.Close() }()

			if got := b.Durability(); got != tier {
				t.Fatalf("Durability() = %q, want %q", got, tier)
			}
		})
	}
}
