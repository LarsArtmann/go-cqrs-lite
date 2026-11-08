package system_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// Cross-layer round-trip proofs (ADR-0165): the artifacts other layers render
// must load back through LoadConfig unchanged. Each testdata file records its
// provenance so drift is auditable.

func loadConfigFile(t *testing.T, name string) system.DeploymentConfig {
	t.Helper()

	path := filepath.Join("testdata", name)
	cfg, err := system.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%s): %v", name, err)
	}

	return cfg
}

func writeAndLoad(t *testing.T, name string, data []byte) system.DeploymentConfig {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := system.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%s): %v", name, err)
	}

	return cfg
}

// TestRoundTrip_NixOSRenderedJSON pins that the NixOS module's rendered
// deployment.json (nixosModules.cqrs, standardTopology preset, peer-socket
// postgres DSN) loads verbatim. The artifact mirrors
// nix eval --raw '...environment.etc."cqrs/demo/deployment.json".text' for
// engines = { primary = postgres; proj = sqlite; }; regenerate it whenever the
// module's resolveDSN or standardInstances changes.
func TestRoundTrip_NixOSRenderedJSON(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("testdata", "standard-topology.nixos-rendered.json"))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}

	cfg := writeAndLoad(t, "deployment.json", data)

	if cfg.Engines["primary"].Driver != "postgres" {
		t.Fatalf("expected postgres driver, got %s", cfg.Engines["primary"].Driver)
	}

	const wantDSN = "postgres:///cqrs_demo_primary?host=/run/postgresql&user=cqrs_demo_primary"
	if cfg.Engines["primary"].DSN != wantDSN {
		t.Fatalf("expected peer-socket DSN %q, got %q", wantDSN, cfg.Engines["primary"].DSN)
	}

	if cfg.Engines["proj"].Driver != "sqlite" || cfg.Engines["proj"].DSN != "file:/var/lib/cqrs/demo/proj.db" {
		t.Fatalf("unexpected proj engine: %+v", cfg.Engines["proj"])
	}

	if len(cfg.Instances) != 3 {
		t.Fatalf("expected 3 standard instances, got %d", len(cfg.Instances))
	}

	if cfg.Instances[0].Role != system.RoleSourceOfTruth || cfg.Instances[0].Engine != "primary" {
		t.Fatalf("unexpected source-of-truth instance: %+v", cfg.Instances[0])
	}

	if cfg.Instances[1].Role != system.RoleQueries || cfg.Instances[1].Engine != "primary" {
		t.Fatalf("unexpected queries instance: %+v", cfg.Instances[1])
	}

	proj := cfg.Instances[2]
	if proj.Role != system.RoleProjections || len(proj.Engines) != 2 ||
		proj.Engines[0] != "primary" || proj.Engines[1] != "proj" {
		t.Fatalf("unexpected projections instance: %+v", proj)
	}

	if want := []string{
		"events",
		"commands",
		"queries",
		"snapshots",
		"checkpoints",
	}; !slices.Equal(
		cfg.Instances[0].Collections,
		want,
	) {
		t.Fatalf("NixOS twin source-of-truth collections: want %v, got %v", want, cfg.Instances[0].Collections)
	}
}

// TestStandard_InstanceTripleMatchesSharedFixture is the Go leg of the
// ADR-0165 cross-layer equivalence pin: system.Standard must produce exactly
// the instance triple in testdata/standard-instance-triple.json — the same
// bytes the CUE gate (scripts/check-cue-config.sh) and the NixOS eval test
// (nix/nixos/eval-test.nix) assert against. Engine names primary+proj are
// load-bearing: the fixture pins the projections pool order.
func TestStandard_InstanceTripleMatchesSharedFixture(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("testdata", "standard-instance-triple.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var want []system.InstanceConfig
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatalf("fixture is not a valid instance triple: %v", err)
	}

	got := system.Standard(system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "sqlite", DSN: "file:app.db"},
			"proj":    {Driver: "memory"},
		},
	}).Instances

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Standard drifted from the shared fixture:\ngot:  %#v\nwant: %#v", got, want)
	}
}

// TestRoundTrip_CueGoldenJSON pins that the CUE `standard` preset export
// (system/config.cue evaluated against testdata/standard-fixture.cue, the same
// render scripts/check-cue-config.sh gates in YAML form) loads verbatim,
// collections included.
func TestRoundTrip_CueGoldenJSON(t *testing.T) {
	t.Parallel()

	cfg := loadConfigFile(t, "standard-topology.golden.json")

	if cfg.Engines["primary"].Driver != "sqlite" || cfg.Engines["primary"].DSN != "file:app.db" {
		t.Fatalf("unexpected primary engine: %+v", cfg.Engines["primary"])
	}

	if len(cfg.Instances) != 3 {
		t.Fatalf("expected 3 standard instances, got %d", len(cfg.Instances))
	}

	wantCollections := [][]string{
		{"events", "commands", "queries", "snapshots", "checkpoints"},
		{"queries"},
		{"projections"},
	}

	for i, want := range wantCollections {
		got := cfg.Instances[i].Collections
		if len(got) != len(want) {
			t.Fatalf("instance %d collections: want %v, got %v", i, want, got)
		}

		for j, name := range want {
			if got[j] != name {
				t.Fatalf("instance %d collections: want %v, got %v", i, want, got)
			}
		}
	}

	if cfg.Instances[0].Engine != "primary" || cfg.Instances[1].Engine != "primary" {
		t.Fatalf("expected WAL instances on primary, got %+v", cfg.Instances[:2])
	}
}

// TestRoundTrip_CueDeploymentExplicitJSON pins the schema-only `deployment`
// entry's explicit passthrough: the JSON twin of
// deployment-explicit.golden.yaml (same cue eval, gated by
// check-cue-config.sh) loads through LoadConfig as EXACTLY the declared
// shape — no standard filler, no pool expansion (ADR-0165: explicit is
// explicit; the filler is opt-in via system.Standard or the `standard`
// entry, never implicit).
func TestRoundTrip_CueDeploymentExplicitJSON(t *testing.T) {
	t.Parallel()

	cfg := loadConfigFile(t, "deployment-explicit.golden.json")

	if len(cfg.Engines) != 1 || cfg.Engines["primary"].Driver != "sqlite" ||
		cfg.Engines["primary"].DSN != "file:app.db" {
		t.Fatalf("unexpected engines: %+v", cfg.Engines)
	}

	if len(cfg.Instances) != 2 {
		t.Fatalf("explicit passthrough must not fill: want 2 declared instances, got %d", len(cfg.Instances))
	}

	sot := cfg.Instances[0]
	if sot.Role != system.RoleSourceOfTruth || sot.Engine != "primary" {
		t.Fatalf("unexpected source-of-truth instance: %+v", sot)
	}
	if sot.Durability != system.DurabilityStrict {
		t.Fatalf("durability must pass through verbatim: got %q", sot.Durability)
	}

	proj := cfg.Instances[1]
	if proj.Role != system.RoleProjections || len(proj.Engines) != 1 || proj.Engines[0] != "primary" {
		t.Fatalf("unexpected projections instance: %+v", proj)
	}
}
