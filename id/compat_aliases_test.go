package id_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestBackwardCompatAliases verifies that every deprecated Aggregate* alias
// resolves to the same type/value as its canonical Stream* replacement. This
// guards the backward-compatibility surface that downstream consumers rely on
// during the ADR-0058 migration window.
func TestBackwardCompatAliases(t *testing.T) {
	t.Parallel()

	// Type aliases are identical by construction — assigning across them compiles.
	var _ id.AggregateID = id.NewStreamID()
	_ = id.AggregateType("User")
	_ = id.AggregateMarker{}

	// Constructors produce identical results.
	got := id.NewAggregateID()
	if got.IsZero() {
		t.Error("NewAggregateID returned zero value")
	}

	derived := id.DeriveAggregateID("ns", "k1", "k2")
	if derived != id.DeriveStreamID("ns", "k1", "k2") {
		t.Error("DeriveAggregateID does not match DeriveStreamID")
	}

	parsed, err := id.ParseAggregateID("lock_user1_user2")
	if err != nil {
		t.Fatalf("ParseAggregateID: %v", err)
	}
	if parsed != mustParseStream(t, "lock_user1_user2") {
		t.Error("ParseAggregateID does not match ParseStreamID")
	}

	strict, err := id.ParseAggregateIDStrict(
		mustParseStream(t, "01HK1540X0841Y0A6BSX1VKR95").String(),
	)
	if err != nil {
		t.Fatalf("ParseAggregateIDStrict: %v", err)
	}
	if strict.IsZero() {
		t.Error("ParseAggregateIDStrict returned zero value")
	}

	ulid := mustParseStream(t, "01HK1540X0841Y0A6BSX1VKR95")
	if id.IsAggregateIDULID(ulid) != id.IsStreamIDULID(ulid) {
		t.Error("IsAggregateIDULID does not match IsStreamIDULID")
	}

	from := id.AggregateIDFrom(stringer("abc"))
	if from != id.StreamIDFrom(stringer("abc")) {
		t.Error("AggregateIDFrom does not match StreamIDFrom")
	}

	// Stream type parsing.
	st, err := id.ParseAggregateType("Order")
	if err != nil {
		t.Fatalf("ParseAggregateType: %v", err)
	}
	if st != id.StreamType("Order") {
		t.Errorf("ParseAggregateType = %q, want Order", st)
	}

	// Ref construction is identical.
	ref := id.NewAggregateRef(id.StreamType("User"), got)
	if ref != id.NewStreamRef(id.StreamType("User"), got) {
		t.Error("NewAggregateRef does not match NewStreamRef")
	}

	// ErrEmptyAggregateType is the same sentinel as ErrEmptyStreamType.
	if id.ErrEmptyAggregateType != id.ErrEmptyStreamType {
		t.Error("ErrEmptyAggregateType is not ErrEmptyStreamType")
	}
}

type stringer string

func (s stringer) String() string { return string(s) }

func mustParseStream(t *testing.T, s string) id.StreamID {
	t.Helper()

	v, err := id.ParseStreamID(s)
	if err != nil {
		t.Fatalf("ParseStreamID(%q): %v", s, err)
	}

	return v
}
