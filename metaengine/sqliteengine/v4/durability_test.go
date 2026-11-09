package sqliteengine

import (
	"context"
	"errors"
	"slices"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestDurabilityPragmas(t *testing.T) {
	t.Parallel()

	user := []string{"foreign_keys=ON"}

	for _, tc := range []struct {
		tier metaengine.DurabilityTier
		want []string
	}{
		{tier: "", want: user},
		{tier: metaengine.DurabilityStrict, want: []string{"synchronous=FULL", "foreign_keys=ON"}},
		{tier: metaengine.DurabilityNormal, want: []string{"synchronous=NORMAL", "foreign_keys=ON"}},
		{tier: metaengine.DurabilityRelaxed, want: []string{"synchronous=OFF", "foreign_keys=ON"}},
	} {
		got, err := durabilityPragmas(tc.tier, user)
		if err != nil {
			t.Fatalf("durabilityPragmas(%q): %v", tc.tier, err)
		}

		if !slices.Equal(got, tc.want) {
			t.Fatalf("durabilityPragmas(%q) = %v, want %v", tc.tier, got, tc.want)
		}
	}
}

func TestDurabilityPragmas_ConflictingSynchronous(t *testing.T) {
	t.Parallel()

	_, err := durabilityPragmas(metaengine.DurabilityStrict, []string{"synchronous=OFF"})
	if err == nil {
		t.Fatal("explicit synchronous pragma + strict tier must conflict")
	}

	_, err = durabilityPragmas("", []string{"synchronous=OFF"})
	if err != nil {
		t.Fatalf("explicit pragma without a tier must pass, got %v", err)
	}
}

func TestDurabilityPragmas_InvalidTier(t *testing.T) {
	t.Parallel()

	_, err := durabilityPragmas("bogus", nil)
	if !errors.Is(err, metaengine.ErrUnsupportedDurability) {
		t.Fatalf("error = %v, want ErrUnsupportedDurability", err)
	}
}

// TestRegisterDriver_DurabilityTiers exercises the driver-factory path: each
// tier must construct successfully on an in-memory database, and a tier
// conflicting with an explicit synchronous pragma must fail construction.
func TestRegisterDriver_DurabilityTiers(t *testing.T) {
	t.Parallel()

	factory, err := metaengine.LookupDriver("sqlite")
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

		// The engine must report back the tier the factory applied: the
		// synchronous pragma it was constructed with resolves to the same
		// tier (FULL→strict, NORMAL→normal, OFF→relaxed); no pragma (empty
		// tier) reports engine-default. In-memory tier echo is exact because
		// the pragma, not the storage medium, decides here.
		if tier != "" {
			dr, ok := eng.(metaengine.DurabilityReporter)
			if !ok {
				t.Fatalf("tier %q: engine does not implement DurabilityReporter", tier)
			}

			if got := dr.EffectiveDurability(); got != tier {
				t.Errorf("tier %q: EffectiveDurability() = %q, want %q", tier, got, tier)
			}
		}

		eng.Close()
	}

	_, err = factory(context.Background(), metaengine.DriverConfig{
		Durability: metaengine.DurabilityStrict,
		Pragmas:    []string{"synchronous=OFF"},
	})
	if err == nil {
		t.Fatal("conflicting synchronous pragma must fail construction")
	}
}
