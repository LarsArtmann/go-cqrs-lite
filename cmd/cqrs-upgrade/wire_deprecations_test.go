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
