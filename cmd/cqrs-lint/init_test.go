package main

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestGenerateInitConfigDefaultProducesValidJSON verifies the default (no
// preset) config loads into AppConfig without error and contains the expected
// core knobs. This replaces the old TestPresetConfigsLoadIntoAppConfig which
// iterated over hardcoded JSON string templates.
func TestGenerateInitConfigDefaultProducesValidJSON(t *testing.T) {
	t.Parallel()

	content, err := generateInitConfig("")
	if err != nil {
		t.Fatalf("generateInitConfig(\"\") failed: %v", err)
	}

	var cfg AppConfig
	err = json.Unmarshal([]byte(content), &cfg,
		json.MatchCaseInsensitiveNames(true))
	if err != nil {
		t.Fatalf("default config does not load into AppConfig: %v\nconfig:\n%s",
			err, content)
	}

	if cfg.MinSeverity != "info" {
		t.Errorf("expected min-severity=info, got %q", cfg.MinSeverity)
	}
	if cfg.MinConfidence != "low" {
		t.Errorf("expected min-confidence=low, got %q", cfg.MinConfidence)
	}
	if cfg.Format != "text" {
		t.Errorf("expected format=text, got %q", cfg.Format)
	}
}

// TestGenerateInitConfigAllValidPresets verifies every named preset produces
// JSON that loads into AppConfig and carries the preset name.
func TestGenerateInitConfigAllValidPresets(t *testing.T) {
	t.Parallel()

	for _, name := range analyzer.ValidPresetNames() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			content, err := generateInitConfig(name)
			if err != nil {
				t.Fatalf("generateInitConfig(%q) failed: %v", name, err)
			}

			var cfg AppConfig
			err = json.Unmarshal([]byte(content), &cfg,
				json.MatchCaseInsensitiveNames(true))
			if err != nil {
				t.Fatalf("preset %q config does not load into AppConfig: %v\nconfig:\n%s",
					name, err, content)
			}

			if string(cfg.Preset) != name {
				t.Errorf("preset %q: expected preset=%q in loaded config, got %q",
					name, name, cfg.Preset)
			}
		})
	}
}

// TestGenerateInitConfigUnknownPresetErrors verifies that an unknown preset
// name returns a helpful error listing the available presets.
func TestGenerateInitConfigUnknownPresetErrors(t *testing.T) {
	t.Parallel()

	_, err := generateInitConfig("server")
	if err == nil {
		t.Fatal("expected error for unknown preset 'server', got nil")
	}
	if !strings.Contains(err.Error(), "unknown preset") {
		t.Errorf("expected 'unknown preset' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "local-cli") {
		t.Errorf("expected available presets in error, got: %v", err)
	}
}
