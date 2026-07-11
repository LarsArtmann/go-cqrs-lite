package otel_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"

	"github.com/larsartmann/go-cqrs-lite/otel/v4"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_AttributeConstants(t *testing.T) {
	constants := map[string]string{
		"AttrMessageKind":      otel.AttrMessageKind,
		"AttrCommandType":      otel.AttrCommandType,
		"AttrEventType":        otel.AttrEventType,
		"AttrQueryType":        otel.AttrQueryType,
		"AttrAggregateType":    otel.AttrAggregateType,
		"AttrAggregateID":      otel.AttrAggregateID,
		"AttrAggregateVersion": otel.AttrAggregateVersion,
		"AttrEventCount":       otel.AttrEventCount,
		"AttrProjectionName":   otel.AttrProjectionName,
		"AttrStatus":           otel.AttrStatus,
		"StatusSuccess":        otel.StatusSuccess,
		"StatusError":          otel.StatusError,
		"KindCommand":          otel.KindCommand,
		"KindEvent":            otel.KindEvent,
		"KindQuery":            otel.KindQuery,
	}

	got, err := json.MarshalIndent(constants, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	assertOtelGolden(t, filepath.Join("testdata", "golden", "attribute-constants.json"), got)
}

func TestGolden_CommandAttrs(t *testing.T) {
	attrs := otel.CommandAttrs("CreateUser", fixedID("01HK1540X0841Y0A6BSX1VKR95"))

	got, err := json.MarshalIndent(attrsToMap(attrs), "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	assertOtelGolden(t, filepath.Join("testdata", "golden", "command-attrs.json"), got)
}

func TestGolden_EventAttrs(t *testing.T) {
	attrs := otel.EventAttrs("UserCreated", fixedID("01HK1540X0841Y0A6BSX1VKR95"), "User")

	got, err := json.MarshalIndent(attrsToMap(attrs), "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	assertOtelGolden(t, filepath.Join("testdata", "golden", "event-attrs.json"), got)
}

func TestGolden_QueryAttrs(t *testing.T) {
	attrs := otel.QueryAttrs("GetUser")

	got, err := json.MarshalIndent(attrsToMap(attrs), "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	assertOtelGolden(t, filepath.Join("testdata", "golden", "query-attrs.json"), got)
}

func attrsToMap(attrs []attribute.KeyValue) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[string(a.Key)] = a.Value.AsString()
	}

	return m
}

type fixedID string

func (s fixedID) String() string { return string(s) }

var _ fmt.Stringer = fixedID("")

func assertOtelGolden(t *testing.T, path string, got []byte) {
	t.Helper()

	dir := filepath.Dir(path)

	if *update {
		if mkdirErr := os.MkdirAll(dir, 0o750); mkdirErr != nil {
			t.Fatalf("mkdir: %v", mkdirErr)
		}

		if writeErr := os.WriteFile(path, append(got, '\n'), 0o644); writeErr != nil {
			t.Fatalf("write golden: %v", writeErr)
		}

		return
	}

	want, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read golden %s (run with -update to create): %v", path, readErr)
	}

	gotStr := strings.TrimSpace(string(got))
	wantStr := strings.TrimSpace(string(want))

	if gotStr != wantStr {
		t.Errorf("golden mismatch for %s (run with -update to refresh)", path)
	}
}
