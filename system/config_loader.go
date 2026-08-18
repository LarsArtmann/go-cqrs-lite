package system

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// LoadConfig loads a DeploymentConfig from a YAML file and env var overrides.
//
// The YAML structure mirrors DeploymentConfig (koanf struct tags):
//
//	engines:
//	  primary:
//	    driver: sqlite
//	    dsn: file:events.db
//	    pragmas: [journal_mode=wal, foreign_keys=on]
//	buses:
//	  local:
//	    driver: gochannel
//	    mode: sync
//	instances:
//	  - role: source-of-truth
//	    engine: primary
//	    durability: normal
//	  - role: projections
//	    engine: primary
//	acknowledge_warnings:
//	  - "volatile-source-of-truth:source-of-truth"
//	priority:
//	  global: WriteSpeed
//	  perEngine:
//	    primary: ReadSpeed
//	  perQuery:
//	    find_tasks: StorageSpace
//
// Per-engine priority can also be set inline via engines.<name>.priority.
// Priority values: WriteSpeed, ReadSpeed, StorageSpace, Balanced (default).
//
// Env var overrides use the CQRS_ prefix with double-underscore as the
// map/nested separator. koanf merges env on top of YAML (env wins):
//
//	CQRS_ENGINES__PRIMARY__DRIVER=sqlite       → engines.primary.driver
//	CQRS_ENGINES__PRIMARY__DSN=file:events.db  → engines.primary.dsn
//	CQRS_ENGINES__PRIMARY__PRIORITY=ReadSpeed  → engines.primary.priority
//	CQRS_PRIORITY__GLOBAL=WriteSpeed           → priority.global
//	CQRS_PRIORITY__PERENGINE__PRIMARY=ReadSpeed → priority.perEngine.primary
//	CQRS_BUSES__LOCAL__DRIVER=gochannel        → buses.local.driver
//	CQRS_INSTANCES__0__DURABILITY=strict       → instances[0].durability
//
// For backward compatibility, CQRS_DEFAULT_DRIVER and CQRS_DEFAULT_DSN
// create a "primary" engine if one does not already exist.
func LoadConfig(path string) (DeploymentConfig, error) {
	k := koanf.New(".")

	// 1. Load YAML file if a path is provided.
	if path != "" {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return DeploymentConfig{}, fmt.Errorf("system: load config %q: %w", path, err)
		}
	}

	// 2. Load env overrides with CQRS_ prefix. Double-underscore maps to
	// the koanf delimiter ("."), enabling structured nested overrides:
	// CQRS_ENGINES__PRIMARY__DRIVER=sqlite → engines.primary.driver
	if err := k.Load(env.Provider("CQRS_", ".", func(key string) string {
		stripped := strings.TrimPrefix(key, "CQRS_")
		return strings.ReplaceAll(strings.ToLower(stripped), "__", ".")
	}), nil); err != nil {
		return DeploymentConfig{}, fmt.Errorf("system: load env overrides: %w", err)
	}

	// 3. Unmarshal into DeploymentConfig using koanf struct tags.
	var cfg DeploymentConfig
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "koanf",
	}); err != nil {
		return DeploymentConfig{}, fmt.Errorf("system: unmarshal config: %w", err)
	}

	// 4. Initialize nil maps/slices (koanf may leave them nil).
	if cfg.Engines == nil {
		cfg.Engines = make(map[string]EngineConfig)
	}

	if cfg.Buses == nil {
		cfg.Buses = make(map[string]BusConfig)
	}

	if cfg.Instances == nil {
		cfg.Instances = []InstanceConfig{}
	}

	// 5. Apply legacy env var overrides for backward compatibility.
	// (No durability defaulting: an unset instance Durability means
	// unspecified — engine defaults. Silently stamping "normal" would push
	// an explicit tier onto every engine, breaking engines without tier
	// support.)
	applyLegacyEnvOverrides(&cfg)

	return cfg, nil
}

// applyLegacyEnvOverrides handles the old CQRS_DEFAULT_DRIVER/CQRS_DEFAULT_DSN
// env vars for backward compatibility. New deployments should use structured
// env vars: CQRS_ENGINES__PRIMARY__DRIVER, CQRS_ENGINES__PRIMARY__DSN.
func applyLegacyEnvOverrides(cfg *DeploymentConfig) {
	driver := os.Getenv("CQRS_DEFAULT_DRIVER")
	if driver == "" {
		return
	}

	if _, ok := cfg.Engines["primary"]; ok {
		return // structured env or YAML already set this
	}

	cfg.Engines["primary"] = EngineConfig{
		Driver: driver,
		DSN:    os.Getenv("CQRS_DEFAULT_DSN"),
	}
}
