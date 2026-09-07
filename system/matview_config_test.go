package system_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // registers "sqlite"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// The operator declares materialized views in deployment YAML; the chain
// YAML → koanf → EngineConfig → metaengine.DriverConfig → driver must carry
// them. Plain SQLite cannot maintain materialized views, so construction must
// FAIL LOUDLY with the feature hint — proving the specs actually reached the
// driver (a silent ignore would yield a green start with zero acceleration).
func TestDeployment_MaterializedViewsFlowToDriver(t *testing.T) {
	t.Parallel()

	yaml := `
engines:
  orders-db:
    driver: sqlite
    dsn: ":memory:"
    materialized_views:
      - collection: orders
        fn: SUM
        column: amount
      - collection: orders
        fn: SUM
        column: amount
        group_by: customer
instances:
  - role: source-of-truth
    collections: ["orders"]
    engine: orders-db
`

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
		t.Fatal("expected construction error: sqlite cannot maintain materialized views")
	}

	if !strings.Contains(err.Error(), "experimental") {
		t.Fatalf("error should hint at the experimental feature, got: %v", err)
	}
}

// A materialized view with an unknown fn must fail config resolution with a
// clear, actionable error before any engine is constructed.
func TestDeployment_MaterializedViewInvalidFn(t *testing.T) {
	t.Parallel()

	yaml := `
engines:
  orders-db:
    driver: sqlite
    materialized_views:
      - collection: orders
        fn: MEDIAN
        column: amount
instances:
  - role: source-of-truth
    collections: ["orders"]
    engine: orders-db
`

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
		t.Fatal("expected error for unknown materialized view fn")
	}

	if !strings.Contains(err.Error(), "MEDIAN") {
		t.Fatalf("error should name the offending fn, got: %v", err)
	}
}
