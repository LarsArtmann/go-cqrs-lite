package record

import (
	"errors"
	"testing"
)

// Lockstep: event.Type, command.Type, and query.Type are aliases of
// record.Type (ADR-0111 contract 21a). The cross-type comparison pin lives in
// each consuming module's tests (it cannot live here — record cannot import
// its consumers); this side pins the type's own behavior.
func TestTypeBasics(t *testing.T) {
	t.Parallel()

	var vt Type = "user.created"
	if vt.String() != "user.created" {
		t.Errorf("Type.String() = %q, want %q", vt.String(), "user.created")
	}

	if zero := Type("").IsZero(); !zero {
		t.Error("empty Type must report IsZero")
	}
}

func TestParseInstanceID(t *testing.T) {
	t.Parallel()

	got, err := ParseInstanceID("01JXCMD0000000000000000000")
	if err != nil {
		t.Fatalf("ParseInstanceID: %v", err)
	}

	if got.String() != "01JXCMD0000000000000000000" {
		t.Errorf("String() = %q", got.String())
	}

	if got.IsZero() {
		t.Error("parsed InstanceID must not be zero")
	}

	if _, err := ParseInstanceID(""); !errors.Is(err, ErrEmptyInstanceID) {
		t.Errorf("empty parse err = %v, want ErrEmptyInstanceID", err)
	}
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	v, err := ParseVersion(3)
	if err != nil {
		t.Fatalf("ParseVersion(3): %v", err)
	}

	if v != Version(3) || v.IsZero() || v.UInt64() != 3 || v.String() != "3" {
		t.Errorf("Version round-trip broken: %+v", v)
	}

	if zero := Version(0); !zero.IsZero() {
		t.Error("Version(0) must report IsZero (unversioned)")
	}

	if _, err := ParseVersion(-1); !errors.Is(err, ErrNegativeVersion) {
		t.Errorf("negative parse err = %v, want ErrNegativeVersion", err)
	}
}

func TestParseCorrelationID(t *testing.T) {
	t.Parallel()

	got, err := ParseCorrelationID("01JXCORR00000000000000000")
	if err != nil {
		t.Fatalf("ParseCorrelationID: %v", err)
	}

	if got.String() != "01JXCORR00000000000000000" || got.IsZero() {
		t.Errorf("CorrelationID round-trip broken: %+v", got)
	}

	if _, err := ParseCorrelationID(""); !errors.Is(err, ErrEmptyCorrelationID) {
		t.Errorf("empty parse err = %v, want ErrEmptyCorrelationID", err)
	}
}

// Zero-value semantics: the Record's retyped fields express "absent" as their
// zero brands — no epoch-as-presence, no empty-string-as-identity downstream.
func TestRecordZeroValueSemantics(t *testing.T) {
	t.Parallel()

	var rec Record

	if !rec.ID.IsZero() || !rec.Type.IsZero() || !rec.Version.IsZero() {
		t.Errorf("zero Record fields must report IsZero: %+v", rec)
	}

	if !rec.MetaData.CorrelationID.IsZero() {
		t.Errorf("zero CommonMetadata.CorrelationID must report IsZero")
	}
}
