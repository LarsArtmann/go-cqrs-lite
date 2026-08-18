package bboltengine_test

import (
	"context"
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept dep-isolated engine modules each need their own registration durability test
func TestRegisterDriver_DurabilityTiers(t *testing.T) {
	t.Parallel()

	factory, err := metaengine.LookupDriver("bbolt")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	for _, tier := range []metaengine.DurabilityTier{
		"", metaengine.DurabilityStrict, metaengine.DurabilityNormal, metaengine.DurabilityRelaxed,
	} {
		eng, err := factory(context.Background(), metaengine.DriverConfig{
			Durability: tier,
		})
		if err != nil {
			t.Fatalf("tier %q: %v", tier, err)
		}

		eng.Close()
	}
}

func TestRegisterDriver_InvalidDurabilityTier(t *testing.T) {
	t.Parallel()

	factory, err := metaengine.LookupDriver("bbolt")
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
