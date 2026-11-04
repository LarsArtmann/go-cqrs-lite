package analyzer

import (
	"errors"
	"strings"
	"testing"
)

func TestApplyModulePins_PinsAbsentModuleUsed(t *testing.T) {
	t.Parallel()

	usage := map[ModuleKey]ModuleUsage{
		"projectionhost": {Key: "projectionhost", Status: UsageAbsent},
		"kv":             {Key: "kv", Status: UsageImported, Evidence: "go-cqrs-lite/kv"},
	}

	if err := ApplyModulePins(usage, map[string]bool{"projectionhost": true}); err != nil {
		t.Fatalf("ApplyModulePins: %v", err)
	}

	pinned := usage["projectionhost"]
	if pinned.Status != UsageImported {
		t.Fatalf("pinned module status = %v, want used", pinned.Status)
	}

	if !strings.Contains(pinned.Evidence, "scorecard.modules") {
		t.Errorf("pin evidence = %q, want the pin provenance marker", pinned.Evidence)
	}

	if imported := usage["kv"]; imported.Evidence != "go-cqrs-lite/kv" {
		t.Errorf("detected module must keep its import evidence, got %q", imported.Evidence)
	}
}

func TestApplyModulePins_RejectsUnknownKey(t *testing.T) {
	t.Parallel()

	usage := map[ModuleKey]ModuleUsage{}

	err := ApplyModulePins(usage, map[string]bool{"no-such-module": true})
	if err == nil || !errors.Is(err, ErrScorecardWaiver) {
		t.Fatalf("unknown pin key must fail wrapped in ErrScorecardWaiver, got %v", err)
	}
}

func TestApplyModulePins_RejectsFalsePin(t *testing.T) {
	t.Parallel()

	usage := map[ModuleKey]ModuleUsage{
		"kv": {Key: "kv", Status: UsageAbsent},
	}

	err := ApplyModulePins(usage, map[string]bool{"kv": false})
	if err == nil || !errors.Is(err, ErrScorecardWaiver) {
		t.Fatalf("false pin must fail (refusals belong to waivers), got %v", err)
	}

	if usage["kv"].Status != UsageAbsent {
		t.Error("rejected pin must not mutate usage")
	}
}

func TestValidateScorecardModulePins_KnownKeysPass(t *testing.T) {
	t.Parallel()

	if err := ValidateScorecardModulePins(map[string]bool{
		"projectionhost": true,
		"graph":          true,
	}); err != nil {
		t.Fatalf("known scored keys must validate: %v", err)
	}
}

func TestLoadProjectConfig_ModulesBlockParses(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfig(t, dir, `{
		"scorecard": {
			"modules": {"projectionhost": true}
		}
	}`)

	cfg, found, err := LoadProjectConfig(dir)
	if err != nil || !found {
		t.Fatalf("LoadProjectConfig: %v, found=%v", err, found)
	}

	if !cfg.Scorecard.Modules["projectionhost"] {
		t.Fatal("scorecard.modules pin did not parse")
	}
}

func TestLoadProjectConfig_FalsePinRejected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfig(t, dir, `{
		"scorecard": {
			"modules": {"kv": false}
		}
	}`)

	_, _, err := LoadProjectConfig(dir)
	if err == nil || !errors.Is(err, ErrScorecardWaiver) {
		t.Fatalf("false pin must fail config load, got %v", err)
	}
}
