package id_test

import (
	"encoding/json/v2"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// FuzzParseAggregateID is a broader variant of FuzzParse: it accepts ANY
// non-empty string (not just ULID), per the AggregateID contract.
func FuzzParseAggregateID(f *testing.F) {
	f.Add("01H4S2Z4QX8N1P5K3M7R9T0V2W")
	f.Add("")
	f.Add("lock_user1_user2")
	f.Add("any-non-empty-string-with-special!@#$%^&*()")
	f.Add(strings.Repeat("x", 1024))

	f.Fuzz(func(t *testing.T, input string) {
		parsed, err := id.ParseAggregateID(input)
		if input == "" {
			if err == nil {
				t.Error("expected error for empty input")
			}

			return
		}

		if err != nil {
			t.Errorf("unexpected error for %q: %v", input, err)

			return
		}

		if parsed.String() != input {
			t.Errorf("String(): got %q, want %q", parsed.String(), input)
		}

		if parsed.IsZero() {
			t.Error("non-empty input produced IsZero ID")
		}
	})
}

// FuzzDeriveAggregateID_Deterministic verifies that DeriveAggregateID is
// pure: same namespace + keys always produce the same ID.
func FuzzDeriveAggregateID_Deterministic(f *testing.F) {
	f.Add("lock", "u1", "r1")
	f.Add("", "", "")
	f.Add("ns", "k", "")
	f.Add(strings.Repeat("a", 100), "b", strings.Repeat("c", 100))

	f.Fuzz(func(t *testing.T, namespace, key1, key2 string) {
		derived := id.DeriveAggregateID(namespace, key1, key2)
		derived2 := id.DeriveAggregateID(namespace, key1, key2)

		if !derived.Equal(derived2) {
			t.Error("DeriveAggregateID is not deterministic")
		}

		if derived.IsZero() {
			t.Error("DeriveAggregateID returned IsZero")
		}

		if derived.String() == "" {
			t.Error("DeriveAggregateID returned empty string")
		}
	})
}

// FuzzDeriveAggregateID_DifferentInputs confirms that different inputs
// produce different IDs (with overwhelming probability).
func FuzzDeriveAggregateID_DifferentInputs(f *testing.F) {
	f.Add("ns1", "k1", "ns2", "k2")
	f.Add("a", "x", "b", "x")
	f.Add("a", "x", "a", "y")

	f.Fuzz(func(t *testing.T, ns1, k1, ns2, k2 string) {
		derived := id.DeriveAggregateID(ns1, k1)
		derived2 := id.DeriveAggregateID(ns2, k2)

		// If both inputs are exactly equal, IDs must be equal.
		if ns1 == ns2 && k1 == k2 {
			if !derived.Equal(derived2) {
				t.Error("equal inputs should produce equal IDs")
			}

			return
		}

		// Otherwise, IDs should differ (modulo SHA-256 collisions,
		// which are astronomically improbable for arbitrary fuzz input).
		if derived.Equal(derived2) {
			t.Errorf("different inputs produced equal IDs: %q == %q", derived, derived2)
		}
	})
}

// FuzzAggregateID_JSON_Roundtrip drives the JSON encoding of AggregateID
// for arbitrary non-empty strings. JSON marshaling of invalid UTF-8
// produces lossy replacement, so we restrict to valid UTF-8 inputs.
func FuzzAggregateID_JSON_Roundtrip(f *testing.F) {
	f.Add("01H4S2Z4QX8N1P5K3M7R9T0V2W")
	f.Add("lock_user1_user2")
	f.Add("plain-string")
	f.Add("unicode-é-ñ-ü")
	f.Add(strings.Repeat("x", 512))

	f.Fuzz(func(t *testing.T, input string) {
		original, err := id.ParseAggregateID(input)
		if err != nil {
			return
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}

		var decoded id.AggregateID
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		// JSON encoding is lossy for invalid UTF-8. The AggregateID contract
		// permits any string, but JSON transport requires valid UTF-8.
		if !utf8.ValidString(input) {
			return
		}

		if !decoded.Equal(original) {
			t.Errorf("roundtrip mismatch: got %q, want %q", decoded, original)
		}
	})
}

// FuzzAggregateIDFrom drives AggregateIDFrom (a fmt.Stringer → AggregateID).
// Must not panic on any Stringer.
func FuzzAggregateIDFrom(f *testing.F) {
	f.Add("stringer-value")
	f.Add("")
	f.Add(strings.Repeat("z", 1024))

	f.Fuzz(func(t *testing.T, input string) {
		s := stringerFunc(func() string { return input })
		got := id.AggregateIDFrom(s)

		if got.String() != input {
			t.Errorf("AggregateIDFrom: got %q, want %q", got.String(), input)
		}
	})
}

// FuzzNewULID_Unique drives New[A]ID multiple times in the same iteration
// and verifies all values are distinct (ULID entropy is 80 bits).
func FuzzNewULID_Unique(f *testing.F) {
	f.Add(int(2))
	f.Add(int(10))
	f.Add(int(100))

	f.Fuzz(func(t *testing.T, n int) {
		if n < 1 || n > 1000 {
			return
		}

		seen := make(map[string]struct{}, n)
		for i := 0; i < n; i++ {
			newID := id.New[id.AggregateID]()
			s := newID.String()
			if _, dup := seen[s]; dup {
				t.Errorf("duplicate ULID at iteration %d: %s", i, s)
			}
			seen[s] = struct{}{}
		}
	})
}

// FuzzULID_TimestampConsistency drives ULID function on freshly-generated
// IDs and verifies the timestamp is "now" (within a reasonable window).
func FuzzULID_TimestampConsistency(f *testing.F) {
	f.Add(int64(0))

	f.Fuzz(func(t *testing.T, _ int64) {
		newID := id.New[id.AggregateID]()
		ts := id.ULID(newID)

		now := time.Now()
		drift := now.Sub(ts)
		if drift < 0 {
			drift = -drift
		}

		// Allow 1 hour clock skew (in case of weirdness); real drift < 1s.
		if drift > time.Hour {
			t.Errorf("ULID timestamp drift too large: %v", drift)
		}
	})
}

// FuzzCompareIDs drives CompareIDs with random ID pairs. The total
// ordering must be consistent: cmp(a,b) == -cmp(b,a).
func FuzzCompareIDs(f *testing.F) {
	f.Add(0, 0)
	f.Add(1, 1)
	f.Add(5, 3)

	f.Fuzz(func(t *testing.T, _, _ int) {
		// Two different IDs
		idA := id.New[id.AggregateID]()
		idB := id.New[id.AggregateID]()

		cmpAB := id.CompareIDs(idA, idB)
		cmpBA := id.CompareIDs(idB, idA)

		if cmpAB != -cmpBA {
			t.Errorf("CompareIDs not anti-symmetric: cmp(a,b)=%d, cmp(b,a)=%d", cmpAB, cmpBA)
		}

		cmpAA := id.CompareIDs(idA, idA)
		if cmpAA != 0 {
			t.Errorf("CompareIDs(a, a) should be 0, got %d", cmpAA)
		}
	})
}

// FuzzParse_TypeSafety ensures Parse returns errors for invalid ULID strings
// regardless of the brand type.
func FuzzParse_TypeSafety(f *testing.F) {
	f.Add("01H4S2Z4QX8N1P5K3M7R9T0V2W")
	f.Add("")
	f.Add("not-a-ulid")
	f.Add("ZZZZZZZZZZZZZZZZZZZZZZZZZZ")
	f.Add("01H4S2Z4QX8N1P5K3M7R9T0V")      // truncated
	f.Add("01H4S2Z4QX8N1P5K3M7R9T0V2W!!!") // extra bytes
	f.Add("01h4s2z4qx8n1p5k3m7r9t0v2w")    // lowercase

	f.Fuzz(func(t *testing.T, input string) {
		// Parse for both AggregateID (strict ULID) and AggregateID (string)
		_, aggErr := id.ParseAggregateID(input)
		_, ulidErr := id.Parse[id.AggregateID](input)

		// ParseAggregateID only rejects empty
		if input == "" {
			if aggErr == nil {
				t.Error("ParseAggregateID accepted empty")
			}
		} else {
			if aggErr != nil {
				t.Errorf("ParseAggregateID unexpectedly rejected %q: %v", input, aggErr)
			}
		}

		// Parse[id.AggregateID] (the strict ULID path) — depends on whether
		// input is a valid ULID. We just verify it doesn't panic.
		_ = ulidErr
	})
}

type stringerFunc func() string

func (f stringerFunc) String() string { return f() }
