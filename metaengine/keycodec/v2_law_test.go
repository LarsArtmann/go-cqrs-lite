package keycodec_test

// The separator-injection kill (disk-truth plan A8): v1 joins are ambiguous
// when a segment contains Sep, v2 escapes make every join injective, and
// the round-trip survives an adversarial corpus (:, NUL, escape byte,
// 255+ chars, unicode) under rapid.

import (
	"bytes"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// adversarialSegment draws strings that actually contain the bytes the
// escape scheme exists for: NUL (the separator), the escape byte, colons,
// high runes, and long runs — plain rapid.String() never emits control
// bytes, so without this the law would vacuously pass.
func adversarialSegment(rt *rapid.T) string {
	return rapid.OneOf(
		rapid.StringOf(rapid.RuneFrom([]rune{'\x00', '\x01', ':', 'a', 'Z', '0', '→', '世'})),
		rapid.StringOfN(rapid.RuneFrom([]rune{'\x00', '\x01', 'x'}), 130, -1, 300),
		rapid.Just(""), rapid.Just("\x00"), rapid.Just("\x00\x01\x00"),
	).Draw(rt, "segment")
}

func TestEscapeRoundTripTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
	}{
		{"plain", "hello"},
		{"colon", "a:b:c"},
		{"single-nul", "a\x00b"},
		{"leading-nul", "\x00abc"},
		{"trailing-nul", "abc\x00"},
		{"escape-byte", "a\x01b"},
		{"nul-then-escape", "a\x00\x01b"},
		{"escape-then-nul", "a\x01\x00b"},
		{"escape-pair-lookalike", "a\x010b"},
		{"all-specials", "\x00\x01\x00\x01"},
		{"long-255-plus", strings.Repeat("x\x00", 130)},
		{"unicode", "héllo→世界\x00ü"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			escaped := keycodec.Escape(tc.in)

			if strings.ContainsRune(escaped, 0x00) {
				t.Fatalf("Escape(%q) still contains NUL: %q", tc.in, escaped)
			}

			if got := keycodec.Unescape(escaped); got != tc.in {
				t.Fatalf("Unescape(Escape(%q)) = %q", tc.in, got)
			}
		})
	}
}

func TestUnescapeMalformedIsTotal(t *testing.T) {
	t.Parallel()

	if got := keycodec.Unescape("a\x01"); got != "a\x01" {
		t.Fatalf("dangling escape: %q", got)
	}

	if got := keycodec.Unescape("a\x01z"); got != "a\x01z" {
		t.Fatalf("unknown follower: %q", got)
	}
}

// TestV2InjectionKill pins the reason v2 exists: the v1 collision pair
// produces byte-identical keys, the v2 twins do not.
func TestV2InjectionKill(t *testing.T) {
	t.Parallel()

	if !bytes.Equal(keycodec.MapKey("a", "b\x00c"), keycodec.MapKey("a\x00b", "c")) {
		t.Fatal("precondition: v1 pair must collide (the bug being killed)")
	}

	lhs := keycodec.MapKeyV2("a", "b\x00c")
	rhs := keycodec.MapKeyV2("a\x00b", "c")

	if bytes.Equal(lhs, rhs) {
		t.Fatalf("v2 keys collide: %q", lhs)
	}

	if !bytes.Contains(lhs, []byte("m2\x00")) {
		t.Fatalf("v2 key %q is not in the m2 family range", lhs)
	}
}

func TestV1RangeUntouchedByV2(t *testing.T) {
	t.Parallel()

	v1 := keycodec.MapKey("col", "key")
	v2 := keycodec.MapKeyV2("col", "key")

	if bytes.HasPrefix(v2, v1) || bytes.HasPrefix(v1, v2) {
		t.Fatalf("v1 %q and v2 %q ranges overlap", v1, v2)
	}
}

func TestV2PrefixMatchesKeys(t *testing.T) {
	t.Parallel()

	key := keycodec.MapKeyV2("c\x00ol", "k\x00ey")
	prefix := keycodec.CollectionPrefixV2("c\x00ol")

	if !bytes.HasPrefix(key, prefix) {
		t.Fatalf("key %q does not carry its escaped prefix %q", key, prefix)
	}
}

func TestEscapeRoundTripLaw(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		s := adversarialSegment(rt)

		escaped := keycodec.Escape(s)

		if strings.ContainsRune(escaped, 0x00) {
			rt.Fatalf("escaped segment contains NUL: %q", escaped)
		}

		if got := keycodec.Unescape(escaped); got != s {
			rt.Fatalf("round-trip mismatch: %q → %q → %q", s, escaped, got)
		}
	})
}

// TestV2InjectivityLaw is the core property: distinct (collection, key)
// pairs must produce distinct v2 storage keys, for ANY segment content.
func TestV2InjectivityLaw(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		alpha := adversarialSegment(rt)
		beta := adversarialSegment(rt)
		gamma := adversarialSegment(rt)
		delta := adversarialSegment(rt)

		if bytes.Equal(keycodec.MapKeyV2(alpha, beta), keycodec.MapKeyV2(gamma, delta)) {
			if alpha != gamma || beta != delta {
				rt.Fatalf("v2 collision: (%q,%q) vs (%q,%q)", alpha, beta, gamma, delta)
			}
		}
	})
}

// TestV2StreamFamilyInjectivityLaw covers the seq-suffixed journal family:
// stream identity stays separable even when sids carry separators.
func TestV2StreamFamilyInjectivityLaw(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		colA := adversarialSegment(rt)
		sidA := adversarialSegment(rt)
		colB := adversarialSegment(rt)
		sidB := adversarialSegment(rt)

		if bytes.Equal(keycodec.StreamKeyV2(colA, sidA, 7), keycodec.StreamKeyV2(colB, sidB, 7)) {
			if colA != colB || sidA != sidB {
				rt.Fatalf("v2 stream collision: (%q,%q) vs (%q,%q)", colA, sidA, colB, sidB)
			}
		}
	})
}
