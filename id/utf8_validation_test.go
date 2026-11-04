package id

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Regression pins for the 2026-11-03 fuzz finding (FuzzStreamStatus_MarshalOnly):
// a StreamID/StreamType carrying invalid UTF-8 cannot be marshaled by json/v2
// (jsontext: invalid UTF-8), so the parse boundary must reject it instead of
// deferring the failure to a later marshal.

func TestParseStreamID_RejectsInvalidUTF8(t *testing.T) {
	t.Parallel()

	for name, input := range map[string]string{
		"lone continuation byte": "\xdf",
		"truncated 3-byte seq":   "caf\xe4\xb8",
		"byte after valid text":  "user-\xff",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseStreamID(input)
			if err == nil {
				t.Fatalf("ParseStreamID(%q) should reject invalid UTF-8", input)
			}

			if errorfamily.Classify(err) != errorfamily.Rejection {
				t.Errorf(
					"ParseStreamID(%q) family = %v, want Rejection",
					input,
					errorfamily.Classify(err),
				)
			}

			var code string
			if familyErr, ok := err.(interface{ Code() string }); ok {
				code = familyErr.Code()
			}

			if code != "id.parse_stream_not_utf8" {
				t.Errorf("ParseStreamID(%q) code = %q, want id.parse_stream_not_utf8", input, code)
			}
		})
	}
}

func TestParseStreamID_AcceptsMultibyteUTF8(t *testing.T) {
	t.Parallel()

	key := "café-主键" //nolint:gosmopolitan // intentional non-ASCII key under test
	parsed, err := ParseStreamID(key)
	if err != nil {
		t.Fatalf("ParseStreamID(multibyte) error = %v", err)
	}

	if parsed.Get() != key {
		t.Errorf("ParseStreamID(multibyte) = %q, want round-trip", parsed.Get())
	}
}

func TestParseStreamType_RejectsInvalidUTF8(t *testing.T) {
	t.Parallel()

	_, err := ParseStreamType("User\xff")
	if err == nil {
		t.Fatal("ParseStreamType should reject invalid UTF-8")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Errorf("ParseStreamType family = %v, want Rejection", errorfamily.Classify(err))
	}

	coded, ok := errors.AsType[interface {
		error
		Code() string
	}](err)
	if !ok {
		t.Fatalf("ParseStreamType error should carry a code: %v", err)
	}

	if coded.Code() != "id.stream_type_not_utf8" {
		t.Errorf("ParseStreamType code = %q, want id.stream_type_not_utf8", coded.Code())
	}
}

func TestParseStreamType_AcceptsMultibyteUTF8(t *testing.T) {
	t.Parallel()

	parsed, err := ParseStreamType("Besützergüter")
	if err != nil {
		t.Fatalf("ParseStreamType(multibyte) error = %v", err)
	}

	if parsed.String() != "Besützergüter" {
		t.Errorf("ParseStreamType(multibyte) = %q, want round-trip", parsed.String())
	}
}
