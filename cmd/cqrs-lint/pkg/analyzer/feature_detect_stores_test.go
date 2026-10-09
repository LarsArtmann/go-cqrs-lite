package analyzer

import (
	"slices"
	"testing"
)

// TestDetectFeatures_MetaengineCoreOnlyInfersMemory pins the engine-less
// metaengine inference: the core package init-registers the built-in memory
// driver, so importing it without any engine module implies the memory
// backend — the only import-invisible persistence that exists.
func TestDetectFeatures_MetaengineCoreOnlyInfersMemory(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func main() { _ = metaengine.Store{} }
`,
	})

	fp := ctx.FeatureProfile
	if !fp.HasMetaengine {
		t.Fatal("metaengine core import should set HasMetaengine")
	}
	if !slices.Equal(fp.MetaengineEngines, []string{"memory"}) {
		t.Fatalf("engines = %v, want [memory] (inferred built-in driver)", fp.MetaengineEngines)
	}
	if fp.Store != StoreMemory {
		t.Fatalf("store = %s, want memory (no other backend signal)", fp.Store)
	}
	if !slices.Contains(fp.Stores, StoreMemory) {
		t.Fatalf("stores = %v, want to contain memory", fp.Stores)
	}
	if fp.AnyStoreSQL() || fp.AnyStorePersistent() {
		t.Fatal("memory-only profile must not count as SQL-backed or persistent")
	}
}

// TestDetectFeatures_MixedPoolRecordsAllStores pins the mixed-pool model:
// a stack preset journal plus a projection engine keeps BOTH backends in
// Stores while the primary is the first-seen import signal (here: the
// stack preset, imported first).
func TestDetectFeatures_MixedPoolRecordsAllStores(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/stack/postgres/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
)

func main() { _, _ = postgres.Name, sqliteengine.Name }
`,
	})

	fp := ctx.FeatureProfile
	if fp.Store != StorePostgres {
		t.Fatalf("primary store = %s, want postgres (first-seen import signal)", fp.Store)
	}
	if !slices.Contains(fp.Stores, StorePostgres) || !slices.Contains(fp.Stores, StoreSQLite) {
		t.Fatalf("stores = %v, want postgres+sqlite recorded (mixed pool)", fp.Stores)
	}
	if !slices.Equal(fp.MetaengineEngines, []string{"sqlite"}) {
		t.Fatalf("engines = %v, want [sqlite]", fp.MetaengineEngines)
	}
	if !fp.AnyStoreSQL() {
		t.Fatal("mixed postgres+sqlite pool must count as SQL-backed")
	}
	if !fp.AnyStorePersistent() {
		t.Fatal("mixed postgres+sqlite pool must count as persistent")
	}
}

// TestDetectFeatures_SystemCompositionImport pins the system composition
// root detection with its boundary rule: "go-cqrs-lite/system/" must match
// the versioned system import but never "go-cqrs-lite/systemtest".
func TestDetectFeatures_SystemCompositionImport(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func main() { _ = system.New }
`,
	})

	if !ctx.FeatureProfile.HasSystemComposition {
		t.Fatal("system/v4 import should set HasSystemComposition")
	}

	sibling := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/systemtest/v4"
)

func main() { _ = systemtest.Run }
`,
	})

	if sibling.FeatureProfile.HasSystemComposition {
		t.Fatal("systemtest import must NOT set HasSystemComposition (path-boundary rule)")
	}
}

// TestDetectFeatures_EngineImportSetsStoreAndList pins single-engine
// wiring: the engine import implies both the primary store and the list.
func TestDetectFeatures_EngineImportSetsStoreAndList(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
)

func main() { _ = pebbleengine.Name }
`,
	})

	fp := ctx.FeatureProfile
	if fp.Store != StorePebble {
		t.Fatalf("store = %s, want pebble", fp.Store)
	}
	if !slices.Equal(fp.Stores, []StoreKind{StorePebble}) {
		t.Fatalf("stores = %v, want [pebble]", fp.Stores)
	}
	if fp.AnyStoreSQL() {
		t.Fatal("pebble is a KV store and must not count as SQL-backed")
	}
	if !fp.AnyStorePersistent() {
		t.Fatal("pebble must count as persistent")
	}
}

// TestDetectFeatures_NoBackendStaysNone guards the regression where the
// memory inference could fire without a metaengine import.
func TestDetectFeatures_NoBackendStaysNone(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func main() { _ = event.Type("x") }
`,
	})

	fp := ctx.FeatureProfile
	if fp.Store != StoreNone {
		t.Fatalf("store = %s, want none", fp.Store)
	}
	if len(fp.Stores) != 0 {
		t.Fatalf("stores = %v, want empty", fp.Stores)
	}
	if len(fp.MetaengineEngines) != 0 {
		t.Fatalf("engines = %v, want empty (no metaengine import)", fp.MetaengineEngines)
	}
}

// TestDetectFeatures_StackMemoryWithMetaengine pins the overlap edge: a
// stack/memory preset (explicit primary) plus engine-less metaengine — the
// inferred memory engine is recorded, no duplicate store appears.
func TestDetectFeatures_StackMemoryWithMetaengine(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
)

func main() { _ = memory.Name }
`,
	})

	fp := ctx.FeatureProfile
	if fp.Store != StoreMemory {
		t.Fatalf("store = %s, want memory (stack preset)", fp.Store)
	}
	if !slices.Equal(fp.Stores, []StoreKind{StoreMemory}) {
		t.Fatalf("stores = %v, want [memory] deduped", fp.Stores)
	}
	if !slices.Equal(fp.MetaengineEngines, []string{"memory"}) {
		t.Fatalf("engines = %v, want [memory]", fp.MetaengineEngines)
	}
}

// TestFeatureProfileStorePredicates exercises the exists-quantified
// predicates over explicit profiles: rules must never consult the primary
// alone when a mixed pool is declared.
func TestFeatureProfileStorePredicates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		profile             FeatureProfile
		wantSQL             bool
		wantPersistent      bool
		wantDistributed     bool
		wantEffectiveStores []StoreKind
	}{
		{
			name:    "empty profile",
			profile: FeatureProfile{},
		},
		{
			name:                "primary only (legacy single-store)",
			profile:             FeatureProfile{Store: StoreSQLite},
			wantSQL:             true,
			wantPersistent:      true,
			wantEffectiveStores: []StoreKind{StoreSQLite},
		},
		{
			name:                "memory primary",
			profile:             FeatureProfile{Store: StoreMemory},
			wantEffectiveStores: []StoreKind{StoreMemory},
		},
		{
			name:                "none primary filtered",
			profile:             FeatureProfile{Store: StoreNone},
			wantEffectiveStores: []StoreKind{},
		},
		{
			name: "memory primary plus postgres in list",
			profile: FeatureProfile{
				Store:  StoreMemory,
				Stores: []StoreKind{StoreMemory, StorePostgres},
			},
			wantSQL:         true,
			wantPersistent:  true,
			wantDistributed: true,
			wantEffectiveStores: []StoreKind{
				StoreMemory, StorePostgres,
			},
		},
		{
			name: "list filters none and unknown entries",
			profile: FeatureProfile{
				Stores: []StoreKind{StoreUnknown, StoreNone, StoreDuckDB},
			},
			wantSQL:             true,
			wantPersistent:      true,
			wantEffectiveStores: []StoreKind{StoreDuckDB},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.profile.EffectiveStores()
			if !slices.Equal(got, tc.wantEffectiveStores) {
				t.Fatalf("EffectiveStores() = %v, want %v", got, tc.wantEffectiveStores)
			}
			if got := tc.profile.AnyStoreSQL(); got != tc.wantSQL {
				t.Errorf("AnyStoreSQL() = %t, want %t", got, tc.wantSQL)
			}
			if got := tc.profile.AnyStorePersistent(); got != tc.wantPersistent {
				t.Errorf("AnyStorePersistent() = %t, want %t", got, tc.wantPersistent)
			}
			if got := tc.profile.AnyStoreDistributed(); got != tc.wantDistributed {
				t.Errorf("AnyStoreDistributed() = %t, want %t", got, tc.wantDistributed)
			}
		})
	}
}

// TestResolveFeatureProfile_StoreSpecOverride pins config-driven mixed
// pools: the array form sets both the primary (first entry) and the list.
func TestResolveFeatureProfile_StoreSpecOverride(t *testing.T) {
	t.Parallel()

	detected := FeatureProfile{Store: StoreSQLite, Stores: []StoreKind{StoreSQLite}}
	cfg := ConfigFeatures{
		Store: &StoreSpec{Kinds: []StoreKind{StorePostgres, StoreSQLite}},
	}

	resolved := ResolveFeatureProfile(cfg, PresetNone, detected)

	if resolved.Store != StorePostgres {
		t.Fatalf("primary = %s, want postgres (first array entry)", resolved.Store)
	}
	if !slices.Equal(resolved.Stores, []StoreKind{StorePostgres, StoreSQLite}) {
		t.Fatalf("stores = %v, want [postgres sqlite] (config replaces detection)", resolved.Stores)
	}
	if !resolved.AnyStoreSQL() {
		t.Fatal("declared mixed pool must count as SQL-backed")
	}
}
