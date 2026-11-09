package id

import (
	"encoding/binary"
	"testing"

	"github.com/oklog/ulid/v2"
)

// TestAdvanceEpoch_NeverLowersMillisecond pins the clock-step-backwards
// contract: once an epoch exists at some millisecond, an older timestamp
// never installs a lower epoch (ULIDs never regress).
func TestAdvanceEpoch_NeverLowersMillisecond(t *testing.T) {
	base := ulid.Now()
	first := advanceEpoch(base)
	if first.ms < base {
		t.Fatalf("epoch ms = %d, want >= %d", first.ms, base)
	}

	if got := advanceEpoch(base - 1000); got.ms < first.ms {
		t.Fatalf(
			"advanceEpoch(backwards) lowered epoch ms to %d, want >= %d",
			got.ms, first.ms,
		)
	}
}

// TestNewULID_LayoutAndOrdering validates the epoch layout from the outside:
// same-millisecond IDs share their 6-byte random prefix and carry strictly
// increasing 4-byte counter suffixes; the full ULIDs are strictly increasing.
// Safe under concurrent ID generation from parallel tests: every ID stamped
// with a given millisecond shares that millisecond's single epoch.
func TestNewULID_LayoutAndOrdering(t *testing.T) {
	const count = 100

	ids := make([]ulid.ULID, count)
	for i := range ids {
		ids[i] = newULID()
	}

	byMs := make(map[uint64][]uint64) // ms → suffix values
	for i, u := range ids {
		if i > 0 && u.Compare(ids[i-1]) <= 0 {
			t.Fatalf("ID %d (%s) not strictly greater than ID %d (%s)", i, u, i-1, ids[i-1])
		}

		ms := binary.BigEndian.Uint64(u[0:8]) >> 16 // first 48 bits = timestamp
		suffix := uint64(binary.BigEndian.Uint32(u[12:16]))
		byMs[ms] = append(byMs[ms], suffix)

		if i > 0 {
			prev := ids[i-1]
			prevMs := binary.BigEndian.Uint64(prev[0:8]) >> 16
			if ms == prevMs && string(u[6:12]) != string(prev[6:12]) {
				t.Fatalf(
					"IDs %d/%s and %d/%s share ms %d but differ in random prefix",
					i-1, prev, i, u, ms,
				)
			}
		}
	}

	for ms, suffixes := range byMs {
		if len(suffixes) < 2 {
			continue
		}

		seen := make(map[uint64]bool, len(suffixes))
		for _, s := range suffixes {
			if seen[s] {
				t.Fatalf("ms %d: duplicate suffix %d within one epoch", ms, s)
			}
			seen[s] = true
		}
	}
}
