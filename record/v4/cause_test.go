package record_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestCauseKind_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		kind record.CauseKind
		want string
	}{
		{record.CauseNone, "none"},
		{record.CauseCommand, "command"},
		{record.CauseTimer, "timer"},
		{record.CauseEvent, "event"},
		{record.CauseUnknown, "unknown"},
		{record.CauseKind(99), "none"},
	}

	for _, tc := range cases {
		if got := tc.kind.String(); got != tc.want {
			t.Errorf("CauseKind(%d).String() = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestCause_ZeroValueIsNone(t *testing.T) {
	t.Parallel()

	var c record.Cause
	if !c.IsZero() {
		t.Error("zero Cause must IsZero()")
	}

	if c.Kind != record.CauseNone {
		t.Errorf("zero Cause Kind = %d, want CauseNone", c.Kind)
	}

	if c.ID != "" {
		t.Errorf("zero Cause ID = %q, want empty", c.ID)
	}

	if c.String() != "" {
		t.Errorf("zero Cause String() = %q, want empty", c.String())
	}
}

func TestCause_StringWireForm(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		cause record.Cause
		want  string
	}{
		{"command", record.Cause{Kind: record.CauseCommand, ID: "01JCMD"}, "command:01JCMD"},
		{"timer", record.Cause{Kind: record.CauseTimer, ID: "timer-7"}, "timer:timer-7"},
		{"event", record.Cause{Kind: record.CauseEvent, ID: "01JEVT"}, "event:01JEVT"},
		{"unknown", record.Cause{Kind: record.CauseUnknown, ID: "01JANY"}, "unknown:01JANY"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.cause.IsZero() {
				t.Fatalf("%s: populated Cause must not IsZero()", tc.name)
			}

			if got := tc.cause.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCommonMetadata_CauseField(t *testing.T) {
	t.Parallel()

	// The Cause field is addressable on CommonMetadata and independent of the
	// deprecated CausationID (both are populated by the AsRecord bridges
	// during the deprecation window).
	md := record.CommonMetadata{
		CausationID: "legacy-id",
		Cause:       record.Cause{Kind: record.CauseCommand, ID: "cmd-1"},
	}

	if md.Cause.Kind != record.CauseCommand || md.Cause.ID != "cmd-1" {
		t.Errorf("Cause = %+v, want {Command cmd-1}", md.Cause)
	}

	if md.CausationID != "legacy-id" {
		t.Errorf("CausationID = %q, want %q (independent storage)", md.CausationID, "legacy-id")
	}
}
