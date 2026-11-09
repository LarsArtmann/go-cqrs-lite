package pgengine_test

import (
	"context"
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept dep-isolated engine modules each need their own registration guard test
func TestRegisterDriver_InvalidDurabilityTier(t *testing.T) {
	t.Parallel()

	factory, err := metaengine.LookupDriver("postgres")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	_, err = factory(context.Background(), metaengine.DriverConfig{
		Durability: "bogus",
	})
	if !errors.Is(err, metaengine.ErrUnsupportedDurability) {
		t.Fatalf("durability tier error = %v, want ErrUnsupportedDurability", err)
	}
}

// TestRegisterDriver_DurabilityTiers pins the tier → DSN → EffectiveDurability
// round trip: the factory applies synchronous_commit for the named tier and
// the engine reports it back through DurabilityReporter. Requires live PG
// (skips without a testcontainer runtime).
func TestRegisterDriver_DurabilityTiers(
	t *testing.T,
) { //nolint:dupl // dep-isolated engine modules each need their own registration durability test
	t.Parallel()

	factory, err := metaengine.LookupDriver("postgres")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	expected := map[metaengine.DurabilityTier]metaengine.DurabilityTier{
		"":                           "",
		metaengine.DurabilityStrict:  metaengine.DurabilityStrict,
		metaengine.DurabilityNormal:  metaengine.DurabilityNormal,
		metaengine.DurabilityRelaxed: metaengine.DurabilityRelaxed,
	}

	for tier, want := range expected {
		eng, err := factory(context.Background(), metaengine.DriverConfig{
			DSN:        pgDSN(t),
			Durability: tier,
		})
		if err != nil {
			t.Fatalf("tier %q: %v", tier, err)
		}

		dr, ok := eng.(metaengine.DurabilityReporter)
		if !ok {
			t.Fatalf("tier %q: engine does not implement DurabilityReporter", tier)
		}

		if got := dr.EffectiveDurability(); got != want {
			t.Errorf("tier %q: EffectiveDurability() = %q, want %q", tier, got, want)
		}

		eng.Close()
	}
}
