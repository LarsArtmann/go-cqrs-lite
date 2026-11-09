package system_test

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// JSON-surface fuzz + Standard preset properties (ADR-0165 P12):
// arbitrary .json bytes must never panic LoadConfig, and the preset must be
// idempotent and invariant over engine-name permutations.

// FuzzLoadConfigJSON: any input yields a config or an error, never a panic.
// Seeds cover the shapes that historically break loaders: valid minimal
// configs, truncated JSON, non-JSON text, unicode identifiers, deep nesting,
// and the NixOS module's rendered artifact (the cross-layer input).
func FuzzLoadConfigJSON(f *testing.F) {
	f.Add([]byte(`{"engines":{"primary":{"driver":"sqlite"}}}`))
	f.Add([]byte(`{"engines":`))
	f.Add([]byte(`not json at all`))
	f.Add([]byte(``))
	f.Add([]byte(`{"engines":{"caf\u00e9":{"driver":"sqlite","dsn":"file:caf\u00e9.db"}}}`))
	f.Add(bytes.Repeat([]byte(`{"a":{"b":`), 200))
	f.Add([]byte(`{"instances":[{"role":"bogus","engine":"x"}]}`))

	if artifact, err := os.ReadFile(
		filepath.Join("testdata", "standard-topology.nixos-rendered.json"),
	); err == nil {
		f.Add(artifact)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), "deployment.json")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write corpus file: %v", err)
		}

		_, _ = system.LoadConfig(path)
	})
}

// TestProperty_Standard_Idempotent: filling an engines-only config twice
// yields the identical deployment — Standard is a data filler, never a
// stacker.
func TestProperty_Standard_Idempotent(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		deploy := randomEnginesConfig(rt)
		once, twice := system.Standard(deploy), system.Standard(system.Standard(deploy))

		onceJSON, twiceJSON := mustMarshal(rt, once), mustMarshal(rt, twice)
		if !bytes.Equal(onceJSON, twiceJSON) {
			rt.Fatalf("Standard not idempotent:\n%s\n%s", onceJSON, twiceJSON)
		}
	})
}

// TestProperty_Standard_PermutationInvariant: the filled instances are
// identical no matter the map insertion order of the engines — WAL pick
// ("primary" or sorted-first) and the sorted projections pool are
// deterministic across boots.
func TestProperty_Standard_PermutationInvariant(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		names := rapid.SliceOfNDistinct(
			cfgNameGen, 2, 5, func(s string) string { return s },
		).Draw(rt, "names")

		canon := system.Standard(configFor(names))
		canonJSON := mustMarshal(rt, canon)

		for range 5 {
			shuffled := shuffledNames(rt, names)

			got := system.Standard(configFor(shuffled))
			if !bytes.Equal(canonJSON, mustMarshal(rt, got)) {
				rt.Fatalf("instances differ under permutation %v:\ncanon %s\ngot   %s",
					shuffled, canonJSON, mustMarshal(rt, got))
			}
		}
	})
}

// shuffledNames permutes names via a rapid-drawn permutation so every
// iteration is a fresh, reproducible order.
func shuffledNames(rt *rapid.T, names []string) []string {
	rt.Helper()

	shuffled := slices.Clone(names)
	perm := rapid.IntRange(0, len(shuffled)-1).Draw(rt, "perm")
	for i := range shuffled {
		j := (i + perm) % len(shuffled)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled
}

func randomEnginesConfig(rt *rapid.T) system.DeploymentConfig {
	return configFor(rapid.SliceOfDistinct(cfgNameGen, func(s string) string { return s }).Draw(rt, "names"))
}

func configFor(names []string) system.DeploymentConfig {
	engines := make(map[string]system.EngineConfig, len(names))
	for _, name := range names {
		engines[name] = system.EngineConfig{Driver: "sqlite", DSN: "file:" + name + ".db"}
	}

	return system.DeploymentConfig{Engines: engines}
}

func mustMarshal(rt *rapid.T, d system.DeploymentConfig) []byte {
	rt.Helper()

	// Deterministic(true) sorts map keys: encoding/json/v2 emits maps in
	// iteration order by default, so a byte-compare of two marshals of the
	// same Engines map is randomly ordered without it (the repeat-leg
	// failure class of 2026-11-09). The adapters already marshal this way.
	data, err := json.Marshal(d, json.Deterministic(true))
	if err != nil {
		rt.Fatalf("marshal deployment: %v", err)
	}

	return data
}
