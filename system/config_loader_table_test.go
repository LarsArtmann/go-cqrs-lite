package system_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// Table tests over the LoadConfig koanf/YAML/env surfaces (Feedback #6 (a) /
// M17.1). The scenario tests in config_loader_test.go pin individual paths;
// this file pins the full YAML shape field-by-field and the error paths.

func writeConfig(t *testing.T, yml string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}

func TestLoadConfig_YAMLTable(t *testing.T) {
	t.Parallel()

	t.Run("priority block all three levels plus inline engine priority", func(t *testing.T) {
		t.Parallel()

		cfg, err := system.LoadConfig(writeConfig(t, `
engines:
  primary:
    driver: sqlite
    priority: StorageSpace
  cache:
    driver: memory
priority:
  global: WriteSpeed
  perEngine:
    primary: ReadSpeed
  perQuery:
    find_tasks: StorageSpace
`))
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}

		if cfg.Priority == nil {
			t.Fatal("priority block not parsed")
		}

		if cfg.Priority.Global != metaengine.PriorityWriteSpeed {
			t.Fatalf("global = %q, want WriteSpeed", cfg.Priority.Global)
		}

		if cfg.Priority.PerEngine["primary"] != metaengine.PriorityReadSpeed {
			t.Fatalf("perEngine[primary] = %q, want ReadSpeed", cfg.Priority.PerEngine["primary"])
		}

		if cfg.Priority.PerQuery["find_tasks"] != metaengine.PriorityStorageSpace {
			t.Fatalf("perQuery[find_tasks] = %q, want StorageSpace", cfg.Priority.PerQuery["find_tasks"])
		}

		if cfg.Engines["primary"].Priority != metaengine.PriorityStorageSpace {
			t.Fatalf("inline engine priority = %q, want StorageSpace", cfg.Engines["primary"].Priority)
		}
	})

	t.Run("materialized views and manifest path", func(t *testing.T) {
		t.Parallel()

		cfg, err := system.LoadConfig(writeConfig(t, `
engines:
  primary:
    driver: turso
    dsn: libsql://db
    materialized_views:
      - collection: tasks
        fn: COUNT
      - collection: orders
        fn: SUM
        column: amount
        group_by: customer
manifest_path: /var/lib/cqrs/plan.json
`))
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}

		if cfg.ManifestPath != "/var/lib/cqrs/plan.json" {
			t.Fatalf("manifest_path = %q", cfg.ManifestPath)
		}

		views := cfg.Engines["primary"].MaterializedViews
		if len(views) != 2 {
			t.Fatalf("materialized_views = %d entries, want 2", len(views))
		}

		if views[0].Collection != "tasks" || views[0].Fn != "COUNT" || views[0].Column != "" {
			t.Fatalf("views[0] = %+v, want tasks COUNT without column", views[0])
		}

		if views[1].Column != "amount" || views[1].GroupBy != "customer" {
			t.Fatalf("views[1] = %+v, want column amount group_by customer", views[1])
		}
	})

	t.Run("mixed-pool instance with collections publish and cache", func(t *testing.T) {
		t.Parallel()

		cfg, err := system.LoadConfig(writeConfig(t, `
instances:
  - role: projections
    engines: [primary, cache]
    collections: [tasks, orders]
    publish: [local]
    cache:
      capacity: 512
`))
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}

		inst := cfg.Instances[0]
		if inst.Role != system.RoleProjections {
			t.Fatalf("role = %q, want projections", inst.Role)
		}

		if len(inst.Engines) != 2 || inst.Engines[0] != "primary" || inst.Engines[1] != "cache" {
			t.Fatalf("engines = %v, want [primary cache]", inst.Engines)
		}

		if len(inst.Collections) != 2 || inst.Publish[0] != "local" {
			t.Fatalf("collections/publish not parsed: %+v", inst)
		}

		if inst.Cache == nil || inst.Cache.Capacity != 512 {
			t.Fatalf("cache = %+v, want capacity 512", inst.Cache)
		}
	})

	t.Run("bus url and mode", func(t *testing.T) {
		t.Parallel()

		cfg, err := system.LoadConfig(writeConfig(t, `
buses:
  edge:
    driver: nats
    url: nats://edge:4222
    mode: async
`))
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}

		bus := cfg.Buses["edge"]
		if bus.Driver != "nats" || bus.URL != "nats://edge:4222" || bus.Mode != "async" {
			t.Fatalf("bus = %+v", bus)
		}
	})
}

func TestLoadConfig_NilMapInitialization(t *testing.T) {
	t.Parallel()

	cfg, err := system.LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig empty: %v", err)
	}

	if cfg.Engines == nil || cfg.Buses == nil || cfg.Instances == nil {
		t.Fatalf("nil maps/slices after load: engines=%v buses=%v instances=%v",
			cfg.Engines, cfg.Buses, cfg.Instances)
	}

	if len(cfg.Instances) != 0 {
		t.Fatalf("instances = %d, want empty-but-non-nil", len(cfg.Instances))
	}

	if cfg.Priority != nil {
		t.Fatalf("priority = %+v, want nil when absent", cfg.Priority)
	}
}

func TestLoadConfig_Errors(t *testing.T) {
	t.Parallel()

	t.Run("malformed YAML", func(t *testing.T) {
		t.Parallel()

		_, err := system.LoadConfig(writeConfig(t, "engines: [unclosed"))
		if err == nil {
			t.Fatal("malformed YAML: want error, got nil")
		}

		if !strings.Contains(err.Error(), "load config") {
			t.Fatalf("error = %v, want load-config wrapping", err)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()

		_, err := system.LoadConfig(filepath.Join(t.TempDir(), "absent.yaml"))
		if err == nil {
			t.Fatal("missing file: want error, got nil")
		}
	})
}

func TestLoadConfig_EnvIndexedInstances(t *testing.T) {
	yml := `
engines:
  primary:
    driver: memory
instances:
  - role: source-of-truth
    engine: primary
`
	t.Setenv("CQRS_INSTANCES__0__DURABILITY", "strict")

	cfg, err := system.LoadConfig(writeConfig(t, yml))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Instances[0].Durability != system.DurabilityStrict {
		t.Fatalf("instances[0].durability = %q, want strict from env index", cfg.Instances[0].Durability)
	}
}

func TestLoadConfig_EnvIndexedInstancesErrors(t *testing.T) {
	t.Run("index out of range is loud", func(t *testing.T) {
		t.Setenv("CQRS_INSTANCES__5__DURABILITY", "strict")

		_, err := system.LoadConfig("")
		if err == nil || !strings.Contains(err.Error(), "no instances[5]") {
			t.Fatalf("err = %v, want no-instances[5] error", err)
		}
	})

	t.Run("unknown field is loud", func(t *testing.T) {
		yml := "instances:\n  - role: events\n"
		t.Setenv("CQRS_INSTANCES__0__ENGINE_NAME", "x")

		_, err := system.LoadConfig(writeConfig(t, yml))
		if err == nil || !strings.Contains(err.Error(), "unknown instance field") {
			t.Fatalf("err = %v, want unknown-field error", err)
		}
	})
}

func TestLoadConfig_LegacyEnvIgnoredWhenPrimaryExists(t *testing.T) {
	yml := `
engines:
  primary:
    driver: memory
    dsn: "yaml-set"
`
	t.Setenv("CQRS_DEFAULT_DRIVER", "sqlite")
	t.Setenv("CQRS_DEFAULT_DSN", "file:legacy.db")

	cfg, err := system.LoadConfig(writeConfig(t, yml))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Engines["primary"].Driver != "memory" || cfg.Engines["primary"].DSN != "yaml-set" {
		t.Fatalf("legacy env overrode YAML primary: %+v", cfg.Engines["primary"])
	}
}
