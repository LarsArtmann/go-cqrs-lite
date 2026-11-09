package pebble

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func TestTierToSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		tier            stack.DurabilityTier
		wantDisableWAL  bool
		wantAsyncStores bool
	}{
		{
			name:            "strict",
			tier:            stack.DurabilityStrict,
			wantDisableWAL:  false,
			wantAsyncStores: false,
		},
		{
			name:            "normal",
			tier:            stack.DurabilityNormal,
			wantDisableWAL:  false,
			wantAsyncStores: true,
		},
		{
			name:            "relaxed",
			tier:            stack.DurabilityRelaxed,
			wantDisableWAL:  true,
			wantAsyncStores: true,
		},
		{
			name:            "unknown falls back to safest",
			tier:            stack.DurabilityTier("bogus"),
			wantDisableWAL:  false,
			wantAsyncStores: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			disableWAL, asyncStores := tierToSettings(tt.tier)
			if disableWAL != tt.wantDisableWAL {
				t.Errorf(
					"tierToSettings(%q) disableWAL = %v, want %v",
					tt.tier,
					disableWAL,
					tt.wantDisableWAL,
				)
			}

			if asyncStores != tt.wantAsyncStores {
				t.Errorf(
					"tierToSettings(%q) asyncStores = %v, want %v",
					tt.tier,
					asyncStores,
					tt.wantAsyncStores,
				)
			}
		})
	}
}
