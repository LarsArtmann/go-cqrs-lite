package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// writeLintConfig writes a .cqrs-lint.json into dir.
func writeLintConfig(t *testing.T, dir, body string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, analyzer.ConfigFileName), []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// foreignCwdConfig builds the AppConfig shape cmdguard would have produced
// after loading the CWD config: every project key set from the file.
func foreignCwdConfig() *AppConfig {
	return &AppConfig{
		Preset: "library",
		Rules: analyzer.RulesConfig{
			Disable: []string{"F030", "V007"},
		},
		Health: HealthConfig{InfoCap: 7},
	}
}

func TestPathOutsideDir(t *testing.T) {
	t.Parallel()

	if pathOutsideDir("/a/b", "/a/b") {
		t.Error("path == dir must not count as outside")
	}

	if pathOutsideDir("/a/b/c", "/a/b") {
		t.Error("path under dir must not count as outside (monorepo descendant)")
	}

	if !pathOutsideDir("/a/b", "/a/c") {
		t.Error("sibling directory must count as outside")
	}

	if !pathOutsideDir("/a/b/c", "/a/b/d") {
		t.Error("overlapping prefixes without the separator must count as outside")
	}
}

func TestRebindForeignCwdConfig_SameDirKeepsConfig(t *testing.T) {
	dir := t.TempDir()
	writeLintConfig(t, dir, `{"preset":"library"}`)

	t.Chdir(dir)

	cfg := foreignCwdConfig()
	cfg.Path = dir

	var buf bytes.Buffer
	rebindForeignCwdConfig(&buf, cfg)

	if cfg.Preset != "library" || len(cfg.Rules.Disable) != 2 {
		t.Errorf("same-dir run must keep the config: preset=%q disables=%v", cfg.Preset, cfg.Rules.Disable)
	}

	if buf.Len() != 0 {
		t.Errorf("same-dir run must not warn, got:\n%s", buf.String())
	}
}

func TestRebindForeignCwdConfig_DescendantKeepsAncestorConfig(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "services", "api")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeLintConfig(t, root, `{"preset":"library"}`)

	t.Chdir(root)

	cfg := foreignCwdConfig()
	cfg.Path = sub

	var buf bytes.Buffer
	rebindForeignCwdConfig(&buf, cfg)

	if cfg.Preset != "library" {
		t.Errorf("monorepo descendant must inherit the ancestor config, got preset=%q", cfg.Preset)
	}

	if buf.Len() != 0 {
		t.Errorf("monorepo descendant must not warn, got:\n%s", buf.String())
	}
}

func TestRebindForeignCwdConfig_TargetConfigWins(t *testing.T) {
	cwdDir := t.TempDir()
	targetDir := t.TempDir()
	writeLintConfig(t, cwdDir, `{"preset":"library","rules":{"disable":["F030"]}}`)
	writeLintConfig(t, targetDir, `{"preset":"production","rules":{"disable":["D002"]},"health":{"info-cap":3}}`)

	t.Chdir(cwdDir)

	cfg := foreignCwdConfig()
	cfg.Path = targetDir

	var buf bytes.Buffer
	rebindForeignCwdConfig(&buf, cfg)

	if cfg.Preset != "production" {
		t.Errorf("target's preset must win, got %q", cfg.Preset)
	}

	if len(cfg.Rules.Disable) != 1 || cfg.Rules.Disable[0] != "D002" {
		t.Errorf("target's rules must replace the cwd rules, got %v", cfg.Rules.Disable)
	}

	if cfg.Health.InfoCap != 3 {
		t.Errorf("target's health must apply, got %d", cfg.Health.InfoCap)
	}

	out := buf.String()
	if !strings.Contains(out, targetDir) || !strings.Contains(out, cwdDir) {
		t.Errorf("warning must name both directories, got:\n%s", out)
	}

	if !strings.Contains(out, "library") {
		t.Errorf("warning must name the foreign preset, got:\n%s", out)
	}
}

func TestRebindForeignCwdConfig_TargetWithoutConfigFallsBackToDefaults(t *testing.T) {
	cwdDir := t.TempDir()
	targetDir := t.TempDir()
	writeLintConfig(t, cwdDir, `{"preset":"library","rules":{"disable":["F030"]}}`)

	t.Chdir(cwdDir)

	cfg := foreignCwdConfig()
	cfg.Path = targetDir

	var buf bytes.Buffer
	rebindForeignCwdConfig(&buf, cfg)

	if cfg.Preset != "" || len(cfg.Rules.Disable) != 0 || cfg.Health.InfoCap != 0 {
		t.Errorf(
			"config-less target must run on tag defaults, got preset=%q disables=%v infoCap=%d",
			cfg.Preset, cfg.Rules.Disable, cfg.Health.InfoCap,
		)
	}

	if !strings.Contains(buf.String(), "tag defaults") {
		t.Errorf("warning must say the fallback, got:\n%s", buf.String())
	}
}

func TestRebindForeignCwdConfig_NoCwdConfigIsNoop(t *testing.T) {
	cwdDir := t.TempDir()
	targetDir := t.TempDir()
	writeLintConfig(t, targetDir, `{"preset":"production"}`)

	t.Chdir(cwdDir)

	cfg := &AppConfig{Path: targetDir, Preset: ""}

	var buf bytes.Buffer
	rebindForeignCwdConfig(&buf, cfg)

	if cfg.Preset != "" {
		t.Errorf("nothing was bound from cwd, target config must not be force-applied, got %q", cfg.Preset)
	}

	if buf.Len() != 0 {
		t.Errorf("no cwd config means no hazard, must not warn, got:\n%s", buf.String())
	}
}

func TestRebindForeignCwdConfig_MalformedTargetConfigDropsForeignKeys(t *testing.T) {
	cwdDir := t.TempDir()
	targetDir := t.TempDir()
	writeLintConfig(t, cwdDir, `{"preset":"library"}`)
	writeLintConfig(t, targetDir, `{"preset":`)

	t.Chdir(cwdDir)

	cfg := foreignCwdConfig()
	cfg.Path = targetDir

	var buf bytes.Buffer
	rebindForeignCwdConfig(&buf, cfg)

	if cfg.Preset != "" || len(cfg.Rules.Disable) != 0 {
		t.Errorf("foreign keys must be dropped even when the target config is broken, got %+v", cfg)
	}

	out := buf.String()
	if !strings.Contains(out, "could not be parsed") {
		t.Errorf("warning must surface the parse failure, got:\n%s", out)
	}
}
