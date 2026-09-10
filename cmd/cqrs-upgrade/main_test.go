package main

import (
	"encoding/json"
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
	}, "")

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
		{
			Module:   "github.com/larsartmann/go-cqrs-lite/system/v4",
			From:     "v4.5.0",
			To:       "v4.5.0",
			upToDate: true,
		},
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
		t.Errorf(
			"system/v4 unexpectedly changed: %s",
			got["github.com/larsartmann/go-cqrs-lite/system/v4"],
		)
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

func TestPlanUpgrades_CeilingClampsWithoutDowngrade(t *testing.T) {
	t.Parallel()

	orig := versionResolver

	t.Cleanup(func() { versionResolver = orig })

	versionResolver = func(string) (string, error) { return "v4.20.0", nil }

	bumps := planUpgrades([]pin{
		{Module: "a", Current: "v4.10.0"}, // clamped down to ceiling
		{Module: "b", Current: "v4.30.0"}, // above ceiling: held, never downgraded
		{Module: "c", Current: "v4.12.0"}, // ceiling == current: up-to-date
	}, "v4.12.0")

	if bumps[0].To != "v4.12.0" || bumps[0].upToDate || bumps[0].held {
		t.Errorf("expected a clamped bump to v4.12.0, got %+v", bumps[0])
	}

	if bumps[1].To != "v4.30.0" || !bumps[1].held {
		t.Errorf("expected b held at v4.30.0, got %+v", bumps[1])
	}

	if !bumps[2].upToDate || bumps[2].To != "v4.12.0" {
		t.Errorf("expected c up-to-date at v4.12.0, got %+v", bumps[2])
	}
}

func TestParseFlags_InvalidToVersion(t *testing.T) {
	t.Parallel()

	if _, err := parseFlags([]string{"--to", "4.13.0"}); err == nil {
		t.Fatal("expected error for non-semver --to value")
	}

	cfg, err := parseFlags([]string{"--to", "v4.13.0", "/tmp"})
	if err != nil {
		t.Fatalf("valid --to rejected: %v", err)
	}

	if cfg.to != "v4.13.0" || cfg.dir != "/tmp" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestFindGoMods_SkipsVendorAndTestdata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	for _, m := range []string{
		"go.mod",
		"svc/go.mod",
		"svc/vendor/lib/go.mod",
		"svc/testdata/fixture/go.mod",
		".git/ignored/go.mod",
	} {
		path := filepath.Join(root, filepath.FromSlash(m))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}

		if err := os.WriteFile(
			path,
			[]byte("module example.com/m\n\ngo 1.26\n"),
			0o600,
		); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	mods, err := findGoMods(root)
	if err != nil {
		t.Fatalf("findGoMods: %v", err)
	}

	wantDirs := []string{root, filepath.Join(root, "svc")}

	got := map[string]bool{}
	for _, m := range mods {
		got[m] = true
	}

	for _, want := range wantDirs {
		if !got[want] {
			t.Errorf("expected %s in results, got %v", want, mods)
		}
	}

	if len(mods) != len(wantDirs) {
		t.Errorf("expected exactly %d modules, got %v", len(wantDirs), mods)
	}
}

func TestStrictViolationCounting(t *testing.T) {
	t.Parallel()

	reports := []moduleReport{
		{Dir: "a", Deprecations: []findingJSON{{Position: "x.go:1:1"}}},
		{Dir: "b"},
		{Dir: "c", Deprecations: []findingJSON{{Position: "y.go:2:2"}, {Position: "z.go:3:3"}}},
	}

	if !hasStrictViolation(reports) || countStrictViolations(reports) != 2 {
		t.Errorf("expected 2 violating modules, got %d", countStrictViolations(reports))
	}

	if hasStrictViolation([]moduleReport{{Dir: "clean"}}) {
		t.Error("clean report set must not violate --strict")
	}
}

func TestEmitJSON_WireShape(t *testing.T) {
	t.Parallel()

	reports := []moduleReport{{
		Dir: "/tmp/consumer",
		Bumps: []bump{
			{Module: "example.com/a", From: "v1.0.0", To: "v1.1.0"},
			{Module: "example.com/b", From: "v1.0.0", To: "v1.0.0", upToDate: true},
		},
	}}

	var out strings.Builder

	if err := emitJSON(&out, reports); err != nil {
		t.Fatalf("emitJSON: %v", err)
	}

	var decoded []moduleJSON
	if err := json.Unmarshal([]byte(out.String()), &decoded); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out.String())
	}

	if len(decoded) != 1 || decoded[0].Dir != "/tmp/consumer" {
		t.Fatalf("unexpected decoded report: %+v", decoded)
	}

	if len(decoded[0].Bumps) != 2 {
		t.Fatalf("expected 2 bumps, got %+v", decoded[0].Bumps)
	}

	if decoded[0].Bumps[0].Status != "bump" || decoded[0].Bumps[1].Status != "up-to-date" {
		t.Errorf("statuses not recomputed on the wire: %+v", decoded[0].Bumps)
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
