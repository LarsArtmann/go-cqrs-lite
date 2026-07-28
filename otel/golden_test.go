package otel_test

import (
	"encoding/json/v2"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/larsartmann/go-cqrs-lite/otel/v4"
)

func TestGolden_AttributeConstants(t *testing.T) {
	constants := map[string]string{
		"AttrMessageKind":    otel.AttrMessageKind,
		"AttrCommandType":    otel.AttrCommandType,
		"AttrEventType":      otel.AttrEventType,
		"AttrQueryType":      otel.AttrQueryType,
		"AttrStreamType":     otel.AttrStreamType,
		"AttrStreamID":       otel.AttrStreamID,
		"AttrStreamVersion":  otel.AttrStreamVersion,
		"AttrEventCount":     otel.AttrEventCount,
		"AttrStreamCount":    otel.AttrStreamCount,
		"AttrProjectionName": otel.AttrProjectionName,
		"AttrStatus":         otel.AttrStatus,
		"StatusSuccess":      otel.StatusSuccess,
		"StatusError":        otel.StatusError,
		"KindCommand":        otel.KindCommand,
		"KindEvent":          otel.KindEvent,
		"KindQuery":          otel.KindQuery,
	}

	matchGolden(t, "attribute-constants", marshalSortedMap(constants))
}

func TestGolden_CommandAttrs(t *testing.T) {
	attrs := otel.CommandAttrs("CreateUser", fixedID("01HK1540X0841Y0A6BSX1VKR95"))
	matchGolden(t, "command-attrs", marshalSortedMap(attrsToMap(attrs)))
}

func TestGolden_EventAttrs(t *testing.T) {
	attrs := otel.EventAttrs("UserCreated", fixedID("01HK1540X0841Y0A6BSX1VKR95"), "User")
	matchGolden(t, "event-attrs", marshalSortedMap(attrsToMap(attrs)))
}

func TestGolden_QueryAttrs(t *testing.T) {
	attrs := otel.QueryAttrs("GetUser")
	matchGolden(t, "query-attrs", marshalSortedMap(attrsToMap(attrs)))
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

func marshalSortedMap(m map[string]string) []byte {
	keys := slices.Sorted(maps.Keys(m))
	var b strings.Builder
	b.WriteString("{\n")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",\n")
		}
		kb, _ := json.Marshal(k)
		vb, _ := json.Marshal(m[k])
		b.WriteString("  ")
		b.Write(kb)
		b.WriteString(": ")
		b.Write(vb)
	}
	b.WriteString("\n}")

	return []byte(b.String())
}

// matchGolden wraps go-snaps MatchSnapshot with the module's golden directory.
func matchGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	snaps.WithConfig(
		snaps.Dir(filepath.Join("testdata", "golden")),
		snaps.Filename(name),
	).MatchSnapshot(t, string(got))
}
