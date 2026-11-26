package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// trackingSite is a store site whose locations are only recorded, so a test
// can assert which role every factory call received.
func trackingSite(calls storeCalls) (*storeSite, *[]storeRole) {
	roles := []storeRole{}
	site := &storeSite{
		calls: calls,
		location: func(role storeRole) (string, error) {
			roles = append(roles, role)

			return "store", nil
		},
	}

	return site, &roles
}

// TestStoreSite_ClassifiesCallsByRunPosition pins the role table: benchkit
// calls the factory once per run, once more for warmup, and once more for
// the recovery reopen, and the site must map each position to the store that
// call needs.
func TestStoreSite_ClassifiesCallsByRunPosition(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		calls storeCalls
		want  []storeRole
	}{
		{
			name:  "repeats start a fresh store each",
			calls: storeCalls{},
			want:  []storeRole{storeMeasured, storeMeasured, storeMeasured},
		},
		{
			name:  "warmup writes to a sibling between runs",
			calls: storeCalls{warmup: true},
			want:  []storeRole{storeMeasured, storeWarmup, storeMeasured, storeWarmup},
		},
		{
			name:  "recovery reopens the store it just measured",
			calls: storeCalls{recovery: true},
			want:  []storeRole{storeMeasured, storeReopened, storeMeasured, storeReopened},
		},
		{
			name:  "warmup then recovery in one run",
			calls: storeCalls{warmup: true, recovery: true},
			want: []storeRole{
				storeMeasured, storeWarmup, storeReopened,
				storeMeasured, storeWarmup, storeReopened,
			},
		},
		{
			name:  "replay reads the store, warmup still writes elsewhere",
			calls: storeCalls{warmup: true, replay: true},
			want:  []storeRole{storeReopened, storeWarmup, storeReopened, storeWarmup},
		},
		{
			name:  "replay with recovery only reads",
			calls: storeCalls{recovery: true, replay: true},
			want:  []storeRole{storeReopened, storeReopened, storeReopened, storeReopened},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			site, roles := trackingSite(tc.calls)

			for range tc.want {
				if _, err := site.claim(); err != nil {
					t.Fatalf("claim: %v", err)
				}
			}

			if !slices.Equal(*roles, tc.want) {
				t.Errorf("roles = %v, want %v", *roles, tc.want)
			}
		})
	}
}

// TestDirSite_ResetsEachGenerationAndKeepsWarmupSeparate: a new generation
// starts from an empty store, the warmup sibling lives outside the measured
// directory (so it never counts as disk footprint), the recovery reopen
// reads the store as is, and close keeps the measured store.
func TestDirSite_ResetsEachGenerationAndKeepsWarmupSeparate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	site, err := newDirSite(root, "sqlite", storeCalls{warmup: true, recovery: true})
	if err != nil {
		t.Fatalf("newDirSite: %v", err)
	}

	measured, err := site.claim()
	if err != nil {
		t.Fatalf("claim measured: %v", err)
	}

	marker := filepath.Join(measured, "written-by-the-run")
	if err := os.WriteFile(marker, []byte("x"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	warmup, err := site.claim()
	if err != nil {
		t.Fatalf("claim warmup: %v", err)
	}

	reopened, err := site.claim()
	if err != nil {
		t.Fatalf("claim reopened: %v", err)
	}

	next, err := site.claim()
	if err != nil {
		t.Fatalf("claim next generation: %v", err)
	}

	home := dirHome(root, "sqlite")
	if want := filepath.Join(home, storeDirName); measured != want {
		t.Errorf("measured store = %q, want %q", measured, want)
	}

	if want := filepath.Join(home, warmupDirName); warmup != want {
		t.Errorf("warmup store = %q, want the sibling %q", warmup, want)
	}

	if reopened != measured {
		t.Errorf("recovery reopened %q, want the measured store %q", reopened, measured)
	}

	if next != measured {
		t.Errorf("next generation opened %q, want the measured store %q", next, measured)
	}

	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("a new generation must start from an empty store, marker still present (%v)", err)
	}

	if err := site.close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := os.Stat(warmup); !os.IsNotExist(err) {
		t.Errorf("close must remove the warmup store, still present (%v)", err)
	}

	if info, err := os.Stat(measured); err != nil || !info.IsDir() {
		t.Errorf("close must keep the measured store for inspection and --replay: %v", err)
	}
}

// TestDirSite_CreatesMissingRoot: --dir may name a directory that does not
// exist yet (an older cqrs-bench failed deep inside the driver instead).
func TestDirSite_CreatesMissingRoot(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "does", "not", "exist", "yet")

	site, err := newDirSite(root, "pebble", storeCalls{})
	if err != nil {
		t.Fatalf("newDirSite on a missing root: %v", err)
	}

	defer func() { _ = site.close() }()

	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Errorf("--dir root was not created: %v", err)
	}

	if want := filepath.Join(root, toolDataDir, "pebble", storeDirName); site.measuredDir != want {
		t.Errorf("measuredDir = %q, want %q", site.measuredDir, want)
	}
}

// TestTempSite_CloseRemovesEverything: a temp site leaves nothing behind.
func TestTempSite_CloseRemovesEverything(t *testing.T) {
	t.Parallel()

	site, err := newTempSite(storeCalls{})
	if err != nil {
		t.Fatalf("newTempSite: %v", err)
	}

	if _, err := site.claim(); err != nil {
		t.Fatalf("claim: %v", err)
	}

	root := filepath.Dir(site.measuredDir)

	if _, err := os.Stat(root); err != nil {
		t.Fatalf("temp root missing before close: %v", err)
	}

	if err := site.close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Errorf("temp root still present after close: %v", err)
	}
}

// TestReplaySite_FindsLegacyLayout: a store an older cqrs-bench wrote
// straight into --dir is still found by --replay.
func TestReplaySite_FindsLegacyLayout(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, storeFileName), []byte("legacy"), 0o600); err != nil {
		t.Fatalf("plant legacy store: %v", err)
	}

	site, err := newReplaySite(root, "sqlite", legacyFileStore)
	if err != nil {
		t.Fatalf("newReplaySite on a legacy store: %v", err)
	}

	if site.measuredDir != root {
		t.Errorf("legacy replay measures %q, want the root %q", site.measuredDir, root)
	}

	if _, err := newReplaySite(t.TempDir(), "sqlite", legacyFileStore); !errors.Is(err, errStorePrepare) {
		t.Errorf("replay without any store must fail with errStorePrepare, got %v", err)
	}
}
