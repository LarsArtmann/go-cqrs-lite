package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateScorecardWaivers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		waivers []ScorecardWaiver
		wantErr string
	}{
		{name: "nil is valid", waivers: nil},
		{
			name:    "valid waiver",
			waivers: []ScorecardWaiver{{Key: "graph", Reason: "no traversal models", Trigger: "t"}},
		},
		{
			name:    "empty key",
			waivers: []ScorecardWaiver{{Key: "", Reason: "r"}},
			wantErr: "key must not be empty",
		},
		{
			name: "unknown key",
			waivers: []ScorecardWaiver{{
				Key: "not-a-module", Reason: "r",
			}},
			wantErr: `unknown module key "not-a-module"`,
		},
		{
			name:    "empty reason",
			waivers: []ScorecardWaiver{{Key: "graph"}},
			wantErr: "reason must not be empty",
		},
		{
			name: "duplicate key",
			waivers: []ScorecardWaiver{
				{Key: "graph", Reason: "r1"},
				{Key: "graph", Reason: "r2"},
			},
			wantErr: `duplicate key "graph"`,
		},
		{
			name: "core-only key is not waivable",
			waivers: []ScorecardWaiver{{
				Key: "event", Reason: "core modules are never scored",
			}},
			wantErr: `unknown module key "event"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateScorecardWaivers(tt.waivers)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestLoadProjectConfig_ScorecardWaivers pins the config-file surface: the
// scorecard block parses (JSONC, comments allowed) and invalid waivers fail
// at load time — a typo must never silently lint unconfigured.
func TestLoadProjectConfig_ScorecardWaivers(t *testing.T) {
	t.Parallel()

	t.Run("valid waivers parse", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		content := `{
			// graph projections have no use case in this domain
			"scorecard": {
				"waivers": [
					{
						"key": "graph",
						"reason": "no traversal-heavy read models",
						"trigger": "variable-depth queries appear"
					}
				]
			}
		}`
		if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, found, err := LoadProjectConfig(dir)
		if err != nil || !found {
			t.Fatalf("LoadProjectConfig = (%v, %t, %v)", cfg, found, err)
		}
		if len(cfg.Scorecard.Waivers) != 1 || cfg.Scorecard.Waivers[0].Key != "graph" {
			t.Fatalf("waivers = %+v", cfg.Scorecard.Waivers)
		}
		if cfg.Scorecard.Waivers[0].Trigger == "" {
			t.Error("trigger must parse")
		}
	})

	t.Run("invalid waiver key fails load", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		content := `{"scorecard": {"waivers": [{"key": "bogus", "reason": "r"}]}}`
		if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, err := LoadProjectConfig(dir)
		if err == nil || !strings.Contains(err.Error(), "bogus") {
			t.Fatalf("want load error naming the bogus key, got %v", err)
		}
	})
}
