package bboltengine

import (
	"context"
	"testing"

	bolt "go.etcd.io/bbolt"
)

func TestBoltOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		cfg                engineConfig
		wantNoSync         bool
		wantNoFreelistSync bool
	}{
		{name: "default sync", cfg: engineConfig{}, wantNoSync: false, wantNoFreelistSync: false},
		{
			name:               "no sync",
			cfg:                engineConfig{noSync: true},
			wantNoSync:         true,
			wantNoFreelistSync: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := boltOptions(tt.cfg)
			if opts.NoSync != tt.wantNoSync {
				t.Errorf("NoSync = %v, want %v", opts.NoSync, tt.wantNoSync)
			}

			if opts.NoFreelistSync != tt.wantNoFreelistSync {
				t.Errorf("NoFreelistSync = %v, want %v", opts.NoFreelistSync, tt.wantNoFreelistSync)
			}
		})
	}
}

func TestBoltOptionsDoesNotMutateDefaultOptions(t *testing.T) {
	t.Parallel()

	before := *bolt.DefaultOptions

	opts := boltOptions(engineConfig{noSync: true})
	if !opts.NoSync {
		t.Fatal("noSync config should set NoSync on the returned options")
	}

	if bolt.DefaultOptions.NoSync != before.NoSync ||
		bolt.DefaultOptions.NoFreelistSync != before.NoFreelistSync {
		t.Fatal("boltOptions must never mutate the shared bolt.DefaultOptions global")
	}
}

func TestNewBboltEngine_WithNoSyncSmoke(t *testing.T) {
	t.Parallel()

	eng, err := NewBboltEngine("", WithNoSync())
	if err != nil {
		t.Fatalf("NewBboltEngine failed: %v", err)
	}

	defer func() { _ = eng.Close() }()

	be := eng.(*bboltEngine)
	if err := be.MapSet(context.Background(), "col", "k", map[string]any{"v": 1}); err != nil {
		t.Fatalf("MapSet with NoSync failed: %v", err)
	}

	val, ok, err := be.MapGet(context.Background(), "col", "k")
	if err != nil || !ok {
		t.Fatalf("MapGet after NoSync MapSet: ok=%v err=%v", ok, err)
	}

	decoded, _ := val.(map[string]any)
	if decoded["v"] != float64(1) {
		t.Fatalf("roundtrip mismatch: got %v", val)
	}
}
