package system_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestLoadConfig_YAML(t *testing.T) {
	t.Parallel()

	yaml := `
engines:
  primary:
    driver: sqlite
    dsn: "file:test.db"
    pragmas:
      - journal_mode=wal
      - foreign_keys=on
  cache:
    driver: memory
buses:
  local:
    driver: gochannel
    mode: sync
instances:
  - role: source-of-truth
    engine: primary
    durability: strict
  - role: projections
    engine: cache
    durability: relaxed
acknowledge_warnings:
  - "volatile-source-of-truth:source-of-truth"
`

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := system.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Engines["primary"].Driver != "sqlite" {
		t.Fatalf("expected sqlite driver, got %s", cfg.Engines["primary"].Driver)
	}

	if cfg.Engines["primary"].DSN != "file:test.db" {
		t.Fatalf("expected file:test.db, got %s", cfg.Engines["primary"].DSN)
	}

	if len(cfg.Engines["primary"].Pragmas) != 2 {
		t.Fatalf("expected 2 pragmas, got %d", len(cfg.Engines["primary"].Pragmas))
	}

	if cfg.Engines["cache"].Driver != "memory" {
		t.Fatalf("expected memory driver for cache, got %s", cfg.Engines["cache"].Driver)
	}

	if cfg.Buses["local"].Driver != "gochannel" {
		t.Fatalf("expected gochannel, got %s", cfg.Buses["local"].Driver)
	}

	if len(cfg.Instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(cfg.Instances))
	}

	if cfg.Instances[0].Role != system.RoleSourceOfTruth {
		t.Fatalf("expected source-of-truth, got %s", cfg.Instances[0].Role)
	}

	if cfg.Instances[0].Durability != system.DurabilityStrict {
		t.Fatalf("expected strict durability, got %s", cfg.Instances[0].Durability)
	}

	if len(cfg.AcknowledgeWarnings) != 1 {
		t.Fatalf("expected 1 acknowledged warning, got %d", len(cfg.AcknowledgeWarnings))
	}
}

func TestLoadConfig_EmptyPath(t *testing.T) {
	t.Parallel()

	cfg, err := system.LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig with empty path: %v", err)
	}

	if len(cfg.Engines) != 0 {
		t.Fatalf("expected 0 engines, got %d", len(cfg.Engines))
	}
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	t.Parallel()

	t.Setenv("CQRS_DEFAULT_DRIVER", "sqlite")
	t.Setenv("CQRS_DEFAULT_DSN", "file:env.db")

	cfg, err := system.LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Engines["primary"].Driver != "sqlite" {
		t.Fatalf("expected sqlite from env, got %s", cfg.Engines["primary"].Driver)
	}

	if cfg.Engines["primary"].DSN != "file:env.db" {
		t.Fatalf("expected file:env.db from env, got %s", cfg.Engines["primary"].DSN)
	}
}
