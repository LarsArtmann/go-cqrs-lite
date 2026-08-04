package system

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads a DeploymentConfig from a YAML file and env var overrides.
//
// The YAML structure mirrors DeploymentConfig:
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
//
// Env var overrides use the CQRS_ prefix:
//
//	CQRS_DEFAULT_DRIVER=sqlite
//	CQRS_DEFAULT_DSN=file:events.db
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

// yamlConfig is the YAML representation of DeploymentConfig.
type yamlConfig struct {
	Engines   map[string]yamlEngine `yaml:"engines"`
	Buses     map[string]yamlBus    `yaml:"buses"`
	Instances []yamlInstance        `yaml:"instances"`
	AckWarns  []string              `yaml:"acknowledge_warnings"`
}

type yamlEngine struct {
	Driver  string   `yaml:"driver"`
	DSN     string   `yaml:"dsn"`
	Pragmas []string `yaml:"pragmas"`
}

type yamlBus struct {
	Driver string `yaml:"driver"`
	URL    string `yaml:"url"`
	Mode   string `yaml:"mode"`
}

type yamlInstance struct {
	Role       string   `yaml:"role"`
	Engine     string   `yaml:"engine"`
	Engines    []string `yaml:"engines"`
	Durability string   `yaml:"durability"`
	Publish    []string `yaml:"publish"`
	Subscribe  []string `yaml:"subscribe"`
}

func parseYAML(data []byte, cfg *DeploymentConfig) error {
	if len(data) == 0 {
		return nil
	}

	var yc yamlConfig
	if err := yaml.Unmarshal(data, &yc); err != nil {
		return fmt.Errorf("yaml unmarshal: %w", err)
	}

	for name, eng := range yc.Engines {
		cfg.Engines[name] = EngineConfig{
			Driver:  eng.Driver,
			DSN:     eng.DSN,
			Pragmas: eng.Pragmas,
		}
	}

	for name, bus := range yc.Buses {
		cfg.Buses[name] = BusConfig{
			Driver: bus.Driver,
			URL:    bus.URL,
			Mode:   bus.Mode,
		}
	}

	for _, inst := range yc.Instances {
		ic := InstanceConfig{
			Role:       InstanceRole(inst.Role),
			Engine:     inst.Engine,
			Engines:    inst.Engines,
			Durability: DurabilityTier(inst.Durability),
			Publish:    inst.Publish,
			Subscribe:  inst.Subscribe,
		}

		if ic.Durability == "" {
			ic.Durability = DurabilityNormal
		}

		cfg.Instances = append(cfg.Instances, ic)
	}

	cfg.AcknowledgeWarnings = append(cfg.AcknowledgeWarnings, yc.AckWarns...)

	return nil
}

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
