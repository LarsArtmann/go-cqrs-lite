package main

import (
	"encoding/json/v2"
	"errors"
	"strings"
	"testing"
)

// TestEmitJSON_DeprecationsAlwaysPresent pins the wire contract: every
// module object carries a `deprecations` key (empty array when clean), and a
// failed scan surfaces as `deprecationScanError` instead of silently
// masquerading as clean. Key-absence must never mean "clean".
func TestEmitJSON_DeprecationsAlwaysPresent(t *testing.T) {
	t.Parallel()

	reports := []moduleReport{
		{Dir: "/tmp/clean"},
		{Dir: "/tmp/failed-scan", ScanErr: errors.New("package load failed")},
	}

	var out strings.Builder

	if err := emitJSON(&out, reports); err != nil {
		t.Fatalf("emitJSON: %v", err)
	}

	wire := out.String()

	if !strings.Contains(wire, `"deprecations": []`) {
		t.Errorf("clean module must emit an empty deprecations array:\n%s", wire)
	}

	if strings.Contains(wire, `"deprecations": null`) {
		t.Errorf("deprecations must never serialize as null:\n%s", wire)
	}

	if !strings.Contains(wire, `"deprecationScanError": "package load failed"`) {
		t.Errorf("failed scan must surface deprecationScanError:\n%s", wire)
	}

	var decoded []map[string]any
	if err := json.Unmarshal([]byte(wire), &decoded); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, wire)
	}

	if len(decoded) != 2 {
		t.Fatalf("expected 2 module objects, got %d", len(decoded))
	}

	for i, obj := range decoded {
		if _, ok := obj["deprecations"]; !ok {
			t.Errorf("module %d missing deprecations key:\n%s", i, wire)
		}
	}
}

// TestEmitJSON_BumpsAlwaysPresent pins the symmetry with deprecations:
// every module object carries a `bumps` key (empty array when no pins or
// nothing to bump) — consumers must not treat key-absence as "no bumps".
func TestEmitJSON_BumpsAlwaysPresent(t *testing.T) {
	t.Parallel()

	reports := []moduleReport{
		{Dir: "/tmp/no-pins", NoPins: true},
		{
			Dir:   "/tmp/with-bumps",
			Bumps: []bump{{Module: "example.com/a", From: "v1.0.0", To: "v1.1.0"}},
		},
	}

	var out strings.Builder

	if err := emitJSON(&out, reports); err != nil {
		t.Fatalf("emitJSON: %v", err)
	}

	wire := out.String()

	if !strings.Contains(wire, `"bumps": []`) {
		t.Errorf("pin-free module must emit an empty bumps array:\n%s", wire)
	}

	if strings.Contains(wire, `"bumps": null`) {
		t.Errorf("bumps must never serialize as null:\n%s", wire)
	}

	var decoded []map[string]any
	if err := json.Unmarshal([]byte(wire), &decoded); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, wire)
	}

	for i, obj := range decoded {
		bumps, ok := obj["bumps"]
		if !ok {
			t.Errorf("module %d missing bumps key:\n%s", i, wire)
		} else if arr, ok := bumps.([]any); !ok || len(arr) != len(reports[i].Bumps) {
			t.Errorf("module %d bumps mismatch: %v", i, bumps)
		}
	}
}
