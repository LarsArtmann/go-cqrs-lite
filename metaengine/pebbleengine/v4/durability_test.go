package pebbleengine

import (
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept dep-isolated engine modules each need their own durability translation table test
func TestTierToOptions(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		tier           metaengine.DurabilityTier
		wantSync       bool
		wantDisableWAL bool
	}{
		{tier: "", wantSync: true},
		{tier: metaengine.DurabilityStrict, wantSync: true},
		{tier: metaengine.DurabilityNormal, wantSync: false},
		{tier: metaengine.DurabilityRelaxed, wantSync: false, wantDisableWAL: true},
	} {
		opts, err := tierToOptions(tc.tier)
		if err != nil {
			t.Fatalf("tierToOptions(%q): %v", tc.tier, err)
		}

		cfg := engineConfig{syncWrites: true} // constructor default
		for _, opt := range opts {
			opt(&cfg)
		}

		if cfg.syncWrites != tc.wantSync || cfg.disableWAL != tc.wantDisableWAL {
			t.Fatalf(
				"tierToOptions(%q) config = sync=%t disableWAL=%t, want sync=%t disableWAL=%t",
				tc.tier, cfg.syncWrites, cfg.disableWAL, tc.wantSync, tc.wantDisableWAL,
			)
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
