package systemtest_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // registers "sqlite"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// The operator enables ADR-0141 temporal cells in deployment YAML; the
// chain YAML → koanf → EngineConfig → metaengine.DriverConfig → driver
// must carry it (issue #131). A green boot with cell_versioning declared
// proves the config parsed and the sqlite driver accepted it.
func TestDeployment_CellVersioningFlowsToDriver(t *testing.T) {
	t.Parallel()

	yaml := `
engines:
  orders-db:
    driver: sqlite
    dsn: ":memory:"
    cell_versioning:
      max_versions: 10
      max_age: 720h
instances:
  - role: source-of-truth
    collections: ["orders"]
    engine: orders-db`

	path := filepath.Join(t.TempDir(), "deployment.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := system.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	versioning := cfg.Engines["orders-db"].CellVersioning
	if versioning == nil {
		t.Fatal("cell_versioning must parse into EngineConfig.CellVersioning")
	}

	if versioning.MaxVersions != 10 || versioning.MaxAge != "720h" {
		t.Fatalf("cell_versioning = %+v, want MaxVersions 10 / MaxAge 720h", versioning)
	}

	sys, err := system.New(context.Background(), system.DomainConfig{}, cfg)
	if err != nil {
		t.Fatalf("system.New with cell_versioning: %v", err)
	}

	defer sys.Close() //nolint:errcheck // test teardown
}

// A malformed cell_versioning must fail construction with a clear error
// naming the engine and the offending knob — never a silently unversioned
// engine.
func TestDeployment_CellVersioningInvalidAge(t *testing.T) {
	t.Parallel()

	yaml := `
engines:
  orders-db:
    driver: sqlite
    dsn: ":memory:"
    cell_versioning:
      max_age: sometime
instances:
  - role: source-of-truth
    collections: ["orders"]
    engine: orders-db`

	path := filepath.Join(t.TempDir(), "deployment.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := system.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	_, err = system.New(context.Background(), system.DomainConfig{}, cfg)
	if err == nil {
		t.Fatal("expected construction error for malformed max_age")
	}

	if !strings.Contains(err.Error(), "cell_versioning") || !strings.Contains(err.Error(), "max_age") {
		t.Fatalf("error should name cell_versioning/max_age, got: %v", err)
	}
}
