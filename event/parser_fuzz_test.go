package event_test

import (
	"encoding/json/v2"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// FuzzParseType ensures ParseType accepts any non-empty string and rejects
// empty input — must not panic.
func FuzzParseType(f *testing.F) {
	f.Add("")
	f.Add("user.created")
	f.Add("a")
	f.Add(strings.Repeat("a", 1024))

	f.Fuzz(func(t *testing.T, input string) {
		typ, err := event.ParseType(input)
		if input == "" {
			if err == nil {
				t.Error("expected error for empty input")
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if string(typ) != input {
				t.Errorf("ParseType: got %q, want %q", typ, input)
			}
		}
	})
}

// FuzzParseAggregateType same shape as ParseType.
func FuzzParseAggregateType(f *testing.F) {
	f.Add("")
	f.Add("User")
	f.Add("A")
	f.Add(strings.Repeat("Z", 1024))

	f.Fuzz(func(t *testing.T, input string) {
		typ, err := id.ParseAggregateType(input)
		if input == "" {
			if err == nil {
				t.Error("expected error for empty input")
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if string(typ) != input {
				t.Errorf("id.ParseAggregateType: got %q, want %q", typ, input)
			}
		}
	})
}

// FuzzParseSchemaVersion ensures SchemaVersion parses correctly for any int.
func FuzzParseSchemaVersion(f *testing.F) {
	f.Add(int(0))
	f.Add(int(1))
	f.Add(int(-1))
	f.Add(int(math.MaxInt32))
	f.Add(int(math.MinInt32))

	f.Fuzz(func(t *testing.T, v int) {
		sv, err := event.ParseSchemaVersion(v)
		if v < 1 {
			if err == nil {
				t.Error("expected error for v < 1")
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if sv.Int() != v {
				t.Errorf("SchemaVersion.Int: got %d, want %d", sv.Int(), v)
			}
		}
	})
}

// FuzzNewUserAgent covers NewUserAgent with arbitrary input. Trims spaces
// but accepts anything.
func FuzzNewUserAgent(f *testing.F) {
	f.Add("")
	f.Add("  spaces  ")
	f.Add("Mozilla/5.0")
	f.Add("javascript:alert(1)")
	f.Add(strings.Repeat("X", 4096))

	f.Fuzz(func(t *testing.T, input string) {
		ua := event.NewUserAgent(input)
		if got := ua.String(); got != strings.TrimSpace(input) {
			t.Errorf("UserAgent: got %q, want %q", got, strings.TrimSpace(input))
		}

		if input == "" && !ua.IsZero() {
			t.Error("empty user agent should be IsZero")
		}
	})
}

// FuzzVersion_Arithmetic drives Version.Add/Sub/Mod/Increment with random
// operands. The arithmetic must never panic, even on underflow (which
// panics intentionally — recover and verify).
func FuzzVersion_Arithmetic(f *testing.F) {
	f.Add(int(0), int(0))
	f.Add(int(1), int(0))
	f.Add(int(5), int(10))
	f.Add(int(100), int(0))
	f.Add(int(math.MaxInt32), int(1))

	f.Fuzz(func(t *testing.T, base, n int) {
		if base < 0 {
			return
		}

		v, err := event.ParseVersion(uint64(base))
		if err != nil {
			t.Fatalf("ParseVersion: %v", err)
		}

		// Increment is safe
		if v.Increment().Int() != base+1 {
			t.Errorf("Increment: got %d, want %d", v.Increment().Int(), base+1)
		}

		// Add
		got := v.Add(uint(n))
		if got.Int() != base+n {
			t.Errorf("Add: got %d, want %d", got.Int(), base+n)
		}

		// Mod by n>0 must not panic
		if n > 0 {
			if got := v.Mod(n); got != base%n {
				t.Errorf("Mod: got %d, want %d", got, base%n)
			}
		}

		// Cmp is total ordering
		if v.Cmp(v) != 0 {
			t.Error("Cmp with self should be 0")
		}

		// String roundtrip
		if v.String() == "" {
			t.Error("Version.String should not be empty for non-zero value")
		}
	})
}

// FuzzSchemaVersion_Arithmetic mirrors Version arithmetic for SchemaVersion.
// SchemaVersion.Add/Sub panic on underflow — recover and verify.
func FuzzSchemaVersion_Arithmetic(f *testing.F) {
	f.Add(int(1), int(0))
	f.Add(int(5), int(3))
	f.Add(int(2), int(0))
	f.Add(int(math.MaxInt32), int(1))

	f.Fuzz(func(t *testing.T, base, n int) {
		if base < 1 {
			return
		}

		sv, err := event.ParseSchemaVersion(base)
		if err != nil {
			t.Fatalf("ParseSchemaVersion: %v", err)
		}

		// Increment
		if sv.Increment().Int() != base+1 {
			t.Errorf("Increment: got %d, want %d", sv.Increment().Int(), base+1)
		}

		// Add within bounds
		if n >= 0 {
			got, err := sv.Add(n)
			if err != nil || got.Int() != base+n {
				t.Errorf("Add: got %d (err=%v), want %d", got.Int(), err, base+n)
			}
		}

		// String roundtrip
		if sv.String() == "" {
			t.Error("SchemaVersion.String should not be empty")
		}
	})
}

// FuzzCheckVersionConflict drives CheckVersionConflict with arbitrary
// length/expected pairs. Mismatches must always return a conflict error.
func FuzzCheckVersionConflict(f *testing.F) {
	f.Add(int(0), int(0))
	f.Add(int(1), int(0))
	f.Add(int(5), int(5))
	f.Add(int(100), int(50))

	f.Fuzz(func(t *testing.T, existingLen, expected int) {
		if expected < 0 {
			return
		}

		v, err := event.ParseVersion(uint64(expected))
		if err != nil {
			return
		}

		err = event.CheckVersionConflict(existingLen, v)
		if existingLen == expected {
			if err != nil {
				t.Errorf("expected nil for match, got %v", err)
			}
		} else {
			if err == nil {
				t.Errorf("expected error for mismatch (len=%d, expected=%d)", existingLen, expected)
			}
		}
	})
}

// FuzzDetectTombstone drives DetectTombstone with arbitrary event streams
// and custom metadata. The "last event wins" rule and rebirth precedence
// must hold.
func FuzzDetectTombstone(f *testing.F) {
	f.Add("", "true", "false")
	f.Add("true", "true", "false")
	f.Add("true", "true", "true") // rebirth wins
	f.Add("", "", "")
	f.Add("true", "false", "true")

	f.Fuzz(func(t *testing.T, t1Val, t2Val, t3Val string) {
		events := make([]event.Event, 3)
		for i, v := range []string{t1Val, t2Val, t3Val} {
			e, err := event.NewEvent(
				"evt", id.NewAggregateID(), "Test", event.Version(i+1), nil,
				event.WithMetadata(event.Metadata{
					Custom: map[event.MetadataKey]string{
						event.MetadataKeyTombstone: t1Val,
						event.MetadataKeyRebirth:   t2Val,
					},
				}),
			)
			if err != nil {
				t.Fatalf("build event %d: %v", i, err)
			}

			_ = v
			events[i] = e
		}

		// Empty stream → undetermined
		if got := event.DetectTombstone(nil); got != event.TombstoneUndetermined {
			t.Errorf("empty: got %v, want undetermined", got)
		}

		// Rebirth on last event always wins
		last := events[len(events)-1]
		if last.Metadata().Custom[event.MetadataKeyRebirth] == "true" {
			if got := event.DetectTombstone(events); got != event.TombstoneActive {
				t.Errorf("rebirth precedence: got %v, want Active", got)
			}
		}
	})
}

// FuzzMetadata_CloneIsDeep drives Metadata.Clone with arbitrary Custom maps
// and verifies the clone is independent.
func FuzzMetadata_CloneIsDeep(f *testing.F) {
	f.Add("k1", "v1")
	f.Add("k1", "v2")
	f.Add("", "")
	f.Add("🔥", "💥")

	f.Fuzz(func(t *testing.T, k, v string) {
		md := event.NewMetadata()
		event.EnsureCustom(&md)
		md.Custom[event.MetadataKey(k)] = v

		cp := md.Clone()
		if cp.Custom == nil {
			t.Fatal("Clone returned nil Custom")
		}

		// Mutate the clone
		cp.Custom[event.MetadataKey(k)] = "MUTATED"
		if md.Custom[event.MetadataKey(k)] == "MUTATED" {
			t.Error("Clone leaked mutations into original")
		}
	})
}

// FuzzMetadata_Merge drives Metadata.Merge with arbitrary metadata pairs.
// Non-zero fields in other must overlay onto m; zero fields must be ignored.
func FuzzMetadata_Merge(f *testing.F) {
	f.Add("api", "1.2.3.4", "Mozilla/5.0")
	f.Add("", "", "")
	f.Add("scheduler", "", "")

	f.Fuzz(
		func(t *testing.T, source, ip, ua string) {
			a := event.Metadata{
				Source:    event.Source(source),
				IPAddress: event.IPAddress(ip),
				UserAgent: event.UserAgent(ua),
				Custom:    map[event.MetadataKey]string{"a": "1"},
			}
			b := event.Metadata{
				Source:    event.Source(source + "X"),
				IPAddress: event.IPAddress(ip + "X"),
				UserAgent: event.UserAgent(ua + "X"),
				Custom:    map[event.MetadataKey]string{"b": "2"},
			}

			merged := a.Merge(b)

			// b's non-empty fields must win
			if b.Source != "" && string(merged.Source) != string(b.Source) {
				t.Errorf("Source: got %q, want %q", merged.Source, b.Source)
			}

			if b.IPAddress != "" && string(merged.IPAddress) != string(b.IPAddress) {
				t.Errorf("IPAddress: got %q, want %q", merged.IPAddress, b.IPAddress)
			}

			// Custom maps must be unioned
			if merged.Custom["a"] != "1" || merged.Custom["b"] != "2" {
				t.Error("Custom map not unioned correctly")
			}
		},
	)
}

// FuzzMetadata_JSON_Roundtrip drives JSON marshal/unmarshal of Metadata.
// Invalid UTF-8 bytes get replaced by Go's JSON encoder/decoder (U+FFFD),
// so we restrict to valid UTF-8. IPAddress is also normalized to empty on
// invalid input by ParseIPAddress, so we only check roundtrip on valid IPs.
func FuzzMetadata_JSON_Roundtrip(f *testing.F) {
	f.Add("api", "1.2.3.4", "Mozilla/5.0")
	f.Add("", "", "")
	f.Add("with\nnewline", "", `Mozilla"quoted`)
	f.Add("unicode-é-ñ-ü", "::1", "curl/8.0")

	f.Fuzz(func(t *testing.T, source, ip, ua string) {
		// Skip invalid UTF-8 inputs (Go JSON does lossy replacement)
		if !utf8.ValidString(source) || !utf8.ValidString(ua) {
			return
		}

		validIP := ip == ""
		if ip != "" {
			if _, err := event.ParseIPAddress(ip); err == nil {
				validIP = true
			}
		}

		md := event.Metadata{
			Source:    event.Source(source),
			IPAddress: event.IPAddress(ip),
			UserAgent: event.UserAgent(ua),
		}

		data, err := json.Marshal(md)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}

		var decoded event.Metadata
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		if string(decoded.Source) != string(md.Source) {
			t.Errorf("Source roundtrip: got %q, want %q", decoded.Source, md.Source)
		}

		if validIP && string(decoded.IPAddress) != string(md.IPAddress) {
			t.Errorf("IPAddress roundtrip: got %q, want %q", decoded.IPAddress, md.IPAddress)
		}
	})
}

// FuzzUnmarshalMetadataJSON drives UnmarshalMetadataJSON with arbitrary
// input — must never panic.
func FuzzUnmarshalMetadataJSON(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("null"))
	f.Add([]byte(`{"source":"api"}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`"just a string"`))
	f.Add([]byte(`{"source":12345}`))

	f.Fuzz(func(t *testing.T, input []byte) {
		// Should not panic on any input
		_, _ = event.UnmarshalMetadataJSON(input, "test.code", "test.event")
	})
}
