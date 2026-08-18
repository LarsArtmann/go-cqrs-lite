package badgerengine

import (
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept dep-isolated engine modules each need their own durability translation table test
func TestTierToOptions(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		tier      metaengine.DurabilityTier
		wantSync  bool
		wantAsync bool
	}{
		{tier: "", wantSync: true},
		{tier: metaengine.DurabilityStrict, wantSync: true},
		{tier: metaengine.DurabilityNormal, wantAsync: true},
		{tier: metaengine.DurabilityRelaxed, wantAsync: true},
	} {
		opts, err := tierToOptions(tc.tier)
		if err != nil {
			t.Fatalf("tierToOptions(%q): %v", tc.tier, err)
		}

		cfg := engineConfig{syncWrites: true} // constructor default
		for _, opt := range opts {
			opt(&cfg)
		}

		if tc.wantSync && cfg.syncWrites != true {
			t.Fatalf("tierToOptions(%q): syncWrites = false, want true", tc.tier)
		}

		if tc.wantAsync && cfg.syncWrites != false {
			t.Fatalf("tierToOptions(%q): syncWrites = true, want false", tc.tier)
		}
	}
}

func TestTierToOptions_InvalidTier(t *testing.T) {
	t.Parallel()

	_, err := tierToOptions("bogus")
	if !errors.Is(err, metaengine.ErrUnsupportedDurability) {
		t.Fatalf("error = %v, want ErrUnsupportedDurability", err)
	}
}
