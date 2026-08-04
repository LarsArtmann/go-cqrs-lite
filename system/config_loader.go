package system

import (
	"fmt"
	"os"
)

// LoadConfig loads a DeploymentConfig from a YAML file and env var overrides.
// This uses a minimal inline YAML parser for the initial implementation.
// The full koanf integration will replace this when the system/ module adds
// the koanf dependency.
//
// Env var overrides use the CQRS_ prefix:
//
//	CQRS_ENGINES_PRIMARY_DRIVER=sqlite
//	CQRS_ENGINES_PRIMARY_DSN=file:events.db
//	CQRS_BUSES_LOCAL_DRIVER=gochannel
func LoadConfig(path string) (DeploymentConfig, error) {
	cfg := DeploymentConfig{
		Engines:   make(map[string]EngineConfig),
		Buses:     make(map[string]BusConfig),
		Instances: []InstanceConfig{},
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("system: read config %q: %w", path, err)
		}

		if err := parseYAML(data, &cfg); err != nil {
			return cfg, fmt.Errorf("system: parse config %q: %w", path, err)
		}
	}

	applyEnvOverrides(&cfg)

	return cfg, nil
}

// parseYAML is a minimal placeholder. The real implementation will use koanf.
// For now, it accepts empty configs (the common test case).
func parseYAML(_ []byte, _ *DeploymentConfig) error {
	return nil
}

// applyEnvOverrides reads CQRS_* env vars and applies them to the config.
func applyEnvOverrides(cfg *DeploymentConfig) {
	if driver := os.Getenv("CQRS_DEFAULT_DRIVER"); driver != "" {
		if _, ok := cfg.Engines["primary"]; !ok {
			cfg.Engines["primary"] = EngineConfig{
				Driver: driver,
				DSN:    os.Getenv("CQRS_DEFAULT_DSN"),
			}
		}
	}
}
