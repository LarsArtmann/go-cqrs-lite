package metaengine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestValidateDurabilityTier(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		tier string
		want error
	}{
		{tier: "", want: nil},
		{tier: "strict", want: nil},
		{tier: "normal", want: nil},
		{tier: "relaxed", want: nil},
		{tier: "bogus", want: metaengine.ErrUnsupportedDurability},
		{tier: "STRICT", want: metaengine.ErrUnsupportedDurability},
	} {
		err := metaengine.ValidateDurabilityTier(metaengine.DurabilityTier(tc.tier))
		if !errors.Is(err, tc.want) {
			t.Fatalf("ValidateDurabilityTier(%q) = %v, want %v", tc.tier, err, tc.want)
		}
	}
}

func TestRejectDurabilityTier(t *testing.T) {
	t.Parallel()

	if err := metaengine.RejectDurabilityTier("pebble", metaengine.DriverConfig{}); err != nil {
		t.Fatalf("empty tier must pass, got %v", err)
	}

	err := metaengine.RejectDurabilityTier("pebble", metaengine.DriverConfig{
		Durability: metaengine.DurabilityNormal,
	})
	if !errors.Is(err, metaengine.ErrUnsupportedDurability) {
		t.Fatalf("normal tier error = %v, want ErrUnsupportedDurability", err)
	}

	if got := err.Error(); !strings.Contains(got, "pebble") {
		t.Fatalf("error must name the driver, got %q", got)
	}

	err = metaengine.RejectDurabilityTier("pebble", metaengine.DriverConfig{
		Durability: "bogus",
	})
	if !errors.Is(err, metaengine.ErrUnsupportedDurability) {
		t.Fatalf("bogus tier error = %v, want ErrUnsupportedDurability", err)
	}
}

// TestMemoryDriverDurability locks the built-in memory driver's tier
// contract: strict is a durability lie for process memory and must fail
// construction; normal/relaxed are advisory and pass.
func TestMemoryDriverDurability(t *testing.T) {
	t.Parallel()

	factory, err := metaengine.LookupDriver("memory")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	ctx := context.Background()

	_, err = factory(ctx, metaengine.DriverConfig{Durability: metaengine.DurabilityStrict})
	if !errors.Is(err, metaengine.ErrUnsupportedDurability) {
		t.Fatalf("strict error = %v, want ErrUnsupportedDurability", err)
	}

	_, err = factory(ctx, metaengine.DriverConfig{Durability: "bogus"})
	if !errors.Is(err, metaengine.ErrUnsupportedDurability) {
		t.Fatalf("bogus error = %v, want ErrUnsupportedDurability", err)
	}

	for _, tier := range []metaengine.DurabilityTier{
		"", metaengine.DurabilityNormal, metaengine.DurabilityRelaxed,
	} {
		eng, err := factory(ctx, metaengine.DriverConfig{Durability: tier})
		if err != nil {
			t.Fatalf("tier %q: %v", tier, err)
		}

		eng.Close()
	}
}
