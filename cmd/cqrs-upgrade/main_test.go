package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectPins_FiltersIndirectAndNonCQRS(t *testing.T) {
	t.Parallel()

	pins, err := collectPins(filepath.Join("testdata", "go.mod.txt"))
	if err != nil {
		t.Fatalf("collectPins: %v", err)
	}

	if len(pins) != 3 {
		t.Fatalf("expected 3 direct cqrs pins, got %d: %+v", len(pins), pins)
	}

	want := map[string]string{
		"github.com/larsartmann/go-cqrs-lite/event/v4":  "v4.8.0",
		"github.com/larsartmann/go-cqrs-lite/id/v4":     "v4.5.0",
		"github.com/larsartmann/go-cqrs-lite/system/v4": "v4.5.0",
	}

	for _, p := range pins {
		if want[p.Module] != p.Current {
			t.Errorf("pin %s: got %s, want %s", p.Module, p.Current, want[p.Module])
		}
	}
}

func TestCollectPins_MissingFile(t *testing.T) {
	t.Parallel()

	_, err := collectPins(filepath.Join(t.TempDir(), "missing", "go.mod"))
	if err == nil {
		t.Fatal("expected error for missing go.mod")
	}
}

func TestPickLatest_SemverOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fields []string
		want   string
	}{
		{[]string{"mod", "v4.1.0", "v4.2.0", "v4.10.0"}, "v4.10.0"},
		{[]string{"mod", "v4.2.0", "v4.1.0"}, "v4.2.0"},
		{[]string{"mod", "v4.9.0"}, "v4.9.0"},
		{[]string{"mod"}, ""},
	}

	for _, tt := range tests {
		if got := pickLatest(tt.fields); got != tt.want {
			t.Errorf("pickLatest(%v) = %q, want %q", tt.fields, got, tt.want)
		}
	}
}

func TestPlanUpgrades_StubbedResolver(t *testing.T) {
	t.Parallel()

	orig := versionResolver

	t.Cleanup(func() { versionResolver = orig })

	versionResolver = func(module string) (string, error) {
		if module == "github.com/larsartmann/go-cqrs-lite/event/v4" {
			return "", errors.New("proxy unreachable")
		}

		if strings.Contains(module, "system") {
			return "v4.5.0", nil // same as current
		}

		return "v4.99.0", nil
	}

	bumps := planUpgrades([]pin{
		{Module: "github.com/larsartmann/go-cqrs-lite/event/v4", Current: "v4.8.0"},
		{Module: "github.com/larsartmann/go-cqrs-lite/system/v4", Current: "v4.5.0"},
		{Module: "github.com/larsartmann/go-cqrs-lite/id/v4", Current: "v4.5.0"},
	})

	if bumps[0].resolveErr == nil {
		t.Error("expected resolve error recorded for event/v4")
	}

	if !bumps[1].upToDate {
		t.Error("expected system/v4 up-to-date")
	}

	if bumps[2].To != "v4.99.0" || bumps[2].upToDate {
		t.Errorf("expected id/v4 bump to v4.99.0, got %+v", bumps[2])
	}
}

func TestEditGoMod_UpdatesOnlyChangedPins(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modPath := filepath.Join(dir, "go.mod")

	src, err := os.ReadFile(filepath.Join("testdata", "go.mod.txt"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	if err := os.WriteFile(modPath, src, 0o600); err != nil {
		t.Fatalf("write fixture copy: %v", err)
	}

	bumps := []bump{
		{Module: "github.com/larsartmann/go-cqrs-lite/event/v4", From: "v4.8.0", To: "v4.9.0"},
		{Module: "github.com/larsartmann/go-cqrs-lite/system/v4", From: "v4.5.0", To: "v4.5.0", upToDate: true},
		{Module: "github.com/larsartmann/go-cqrs-lite/id/v4", From: "v4.5.0", To: "v4.99.0"},
	}

	if err := editGoMod(modPath, bumps); err != nil {
		t.Fatalf("editGoMod: %v", err)
	}

	pins, err := collectPins(modPath)
	if err != nil {
		t.Fatalf("re-collect: %v", err)
	}

	got := map[string]string{}

	for _, p := range pins {
		got[p.Module] = p.Current
	}

	if got["github.com/larsartmann/go-cqrs-lite/event/v4"] != "v4.9.0" {
		t.Errorf("event/v4 not bumped: %s", got["github.com/larsartmann/go-cqrs-lite/event/v4"])
	}

	if got["github.com/larsartmann/go-cqrs-lite/id/v4"] != "v4.99.0" {
		t.Errorf("id/v4 not bumped: %s", got["github.com/larsartmann/go-cqrs-lite/id/v4"])
	}

	if got["github.com/larsartmann/go-cqrs-lite/system/v4"] != "v4.5.0" {
		t.Errorf("system/v4 unexpectedly changed: %s", got["github.com/larsartmann/go-cqrs-lite/system/v4"])
	}

	if !strings.Contains(readAll(t, modPath), "// indirect") {
		t.Error("indirect markers lost during rewrite")
	}
}

func TestFormatBumps_Statuses(t *testing.T) {
	t.Parallel()

	out := formatBumps([]bump{
		{Module: "a", From: "v1.0.0", To: "v2.0.0"},
		{Module: "b", From: "v1.0.0", To: "v1.0.0", upToDate: true},
		{Module: "c", From: "v1.0.0", resolveErr: errors.New("boom")},
	})

	for _, want := range []string{"bump", "up-to-date", "resolve-error"} {
		if !strings.Contains(out, want) {
			t.Errorf("formatBumps output missing %q:\n%s", want, out)
		}
	}
}

func readAll(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(data)
}
