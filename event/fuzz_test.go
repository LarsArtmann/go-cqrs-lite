package event_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

func FuzzParseSource(f *testing.F) {
	f.Add("api")
	f.Add("")
	f.Add("scheduler")
	f.Add("  spaces  ")
	f.Add("source-with-unicode-\u00e9")

	f.Fuzz(func(t *testing.T, input string) {
		src, err := event.ParseSource(input)

		trimmed := trimSpaces(input)
		if trimmed == "" {
			if err == nil {
				t.Error("expected error for empty source")
			}

			return
		}

		if err != nil {
			t.Errorf("unexpected error for %q: %v", input, err)
		}

		if src.String() != trimmed {
			t.Errorf("roundtrip mismatch: got %q, want %q", src.String(), trimmed)
		}
	})
}

func FuzzParseIPAddress(f *testing.F) {
	f.Add("192.168.1.1")
	f.Add("")
	f.Add("::1")
	f.Add("not-an-ip")
	f.Add("999.999.999.999")
	f.Add("2001:0db8:85a3:0000:0000:8a2e:0370:7334")

	f.Fuzz(func(_ *testing.T, input string) {
		_, _ = event.ParseIPAddress(input)
	})
}

func FuzzParseVersion(f *testing.F) {
	f.Add(0)
	f.Add(1)
	f.Add(-1)
	f.Add(1000000)

	f.Fuzz(func(t *testing.T, v int) {
		ver, err := event.ParseVersion(v)

		if v < 0 {
			if err == nil {
				t.Error("expected error for negative version")
			}

			return
		}

		if err != nil {
			t.Errorf("unexpected error for %d: %v", v, err)
		}

		if ver.Int() != v {
			t.Errorf("roundtrip mismatch: got %d, want %d", ver.Int(), v)
		}
	})
}

func trimSpaces(s string) string {
	return strings.TrimSpace(s)
}
