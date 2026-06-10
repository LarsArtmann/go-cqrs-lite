package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Loader loads configuration from JSON files with environment-specific overlays.
type Loader struct {
	basePath string
}

// NewLoader creates a new config loader for the given base configuration directory.
func NewLoader(basePath string) *Loader {
	return &Loader{basePath: basePath}
}

// Load reads the base configuration file and overlays environment-specific values.
// It looks for `{name}.json` as the base and `{name}.{env}.json` as the overlay.
func (l *Loader) Load(name string, dest any) error {
	baseFile := filepath.Join(l.basePath, name+".json")

	data, err := os.ReadFile(filepath.Clean(baseFile))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %w", err)
		}

		return fmt.Errorf("read config: %w", err)
	}

	err = json.Unmarshal(data, dest)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	overlayFile := filepath.Join(l.basePath, name+"."+env+".json")

	overlayData, err := os.ReadFile(filepath.Clean(overlayFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no overlay, base config is sufficient
		}

		return fmt.Errorf("read overlay: %w", err)
	}

	err = json.Unmarshal(overlayData, dest)
	if err != nil {
		return fmt.Errorf("parse overlay: %w", err)
	}

	return nil
}
