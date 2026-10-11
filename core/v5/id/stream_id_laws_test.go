package id

import (
	"encoding/json/v2"
	"testing"
)

// TestStreamIDDisplayAndIdentityLaws pins the v4.7.0 branding contract:
// String() is the branded display form, Get()/MarshalText/JSON are the bare
// identity/wire form, and parsing round-trips both forms.
func TestStreamIDDisplayAndIdentityLaws(t *testing.T) {
	//art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
	t.Parallel()

	const bare = "lock_user1_user2"

	parsed, err := ParseStreamID(bare)
	if err != nil {
		t.Fatalf("ParseStreamID(%q) error = %v", bare, err)
	}

	if got := parsed.Get(); got != bare {
		t.Errorf("Get() = %q, want bare identity %q", got, bare)
	}

	if got := parsed.String(); got != "StreamMarker:"+bare {
		t.Errorf("String() = %q, want branded display %q", got, "StreamMarker:"+bare)
	}

	// Round-trip law: parsing the display form yields the same ID.
	fromDisplay, err := ParseStreamID(parsed.String())
	if err != nil {
		t.Fatalf("ParseStreamID(display) error = %v", err)
	}

	if fromDisplay != parsed {
		t.Errorf("ParseStreamID(String()) = %v, want %v", fromDisplay.Get(), parsed.Get())
	}

	// Wire form stays bare.
	text, err := parsed.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}

	if string(text) != bare {
		t.Errorf("MarshalText() = %q, want bare %q", text, bare)
	}

	data, err := json.Marshal(parsed)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if string(data) != `"`+bare+`"` {
		t.Errorf("Marshal() = %s, want bare JSON %q", data, `"`+bare+`"`)
	}
}
