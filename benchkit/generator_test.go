package benchkit

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

func TestGenerator_Deterministic(t *testing.T) {
	t.Parallel()

	g1 := NewGenerator(42, 256, nil)
	g2 := NewGenerator(42, 256, nil)

	p1 := g1.Payload()
	p2 := g2.Payload()

	if p1.ID != p2.ID || p1.Name != p2.Name || p1.Value != p2.Value {
		t.Fatalf("payloads differ: %+v vs %+v", p1, p2)
	}
}

func TestGenerator_DifferentSeeds(t *testing.T) {
	t.Parallel()

	g1 := NewGenerator(1, 256, nil)
	g2 := NewGenerator(2, 256, nil)

	p1 := g1.Payload()
	p2 := g2.Payload()

	if p1.ID == p2.ID {
		t.Error("different seeds produced identical payload IDs")
	}
}

func TestGenerator_PayloadFields(t *testing.T) {
	t.Parallel()

	g := NewGenerator(1, 256, nil)
	p := g.Payload()

	if p.ID == "" {
		t.Error("ID is empty")
	}

	if p.Name == "" {
		t.Error("Name is empty")
	}

	if p.Value < 10 || p.Value > 1000 {
		t.Errorf("Value = %f, want 10-1000", p.Value)
	}

	if p.Items < 1 || p.Items > 20 {
		t.Errorf("Items = %d, want 1-20", p.Items)
	}

	if len(p.Tags) == 0 {
		t.Error("Tags is empty")
	}

	if p.Metadata["source"] == "" {
		t.Error("Metadata.source is empty")
	}

	if p.Padding == "" {
		t.Error("Padding is empty for size=256 (should have padding)")
	}
}

func TestGenerator_DefaultSize(t *testing.T) {
	t.Parallel()

	g := NewGenerator(1, 0, nil) // 0 should default to 256
	p := g.Payload()

	if p.Padding == "" {
		t.Error("Padding should be non-empty with default size 256")
	}
}

func TestGenerator_SmallSizeNoCorruption(t *testing.T) {
	t.Parallel()

	g := NewGenerator(1, 32, nil)
	p := g.Payload()

	// At small sizes, padding should be empty but the payload should still
	// be a valid struct with all fields populated
	if p.ID == "" || p.Name == "" {
		t.Error("payload fields empty at small size")
	}

	if p.Padding != "" {
		t.Error("Padding should be empty when target size < base payload size")
	}
}

func TestGenerator_PayloadSizeAccuracy(t *testing.T) {
	t.Parallel()

	codecs := []struct {
		name      string
		codec     codec.Codec
		tolerance int
	}{
		{"json", codec.JSONCodec{}, 2},
		{"cbor", codec.CBORCodec{}, 5},
	}

	sizes := []int{256, 512, 1024, 4096}

	for _, tc := range codecs {
		for _, target := range sizes {
			t.Run(fmt.Sprintf("%s/size=%d", tc.name, target), func(t *testing.T) {
				t.Parallel()

				g := NewGenerator(1, target, tc.codec)
				p := g.Payload()

				data, err := tc.codec.Encode(p)
				if err != nil {
					t.Fatalf("codec.Encode failed: %v", err)
				}

				actual := len(data)
				diff := actual - target
				if diff < 0 {
					diff = -diff
				}

				// JSON is exact (linear overhead). CBOR can be off by a few bytes
				// due to string-header boundary crossings (23/255/65535).
				if diff > tc.tolerance {
					t.Errorf(
						"%s size=%d: actual encoded size %d differs from target by %d bytes (max %d)",
						tc.name,
						target,
						actual,
						diff,
						tc.tolerance,
					)
				}
			})
		}
	}
}

func TestProfile_TotalEvents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		profile  Profile
		expected int
	}{
		{"dev", ProfileDev, 500},
		{"small", ProfileSmall, 10_000},
		{"medium", ProfileMedium, 500_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.profile.TotalEvents(); got != tt.expected {
				t.Errorf("TotalEvents() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestProfileByName(t *testing.T) {
	t.Parallel()

	names := []string{"dev", "small", "medium", "large", "stress", "write-heavy", "read-heavy"}

	for _, name := range names {
		p, ok := ProfileByName(name)
		if !ok {
			t.Errorf("ProfileByName(%q) returned ok=false", name)
		}

		if p.Name != name {
			t.Errorf("ProfileByName(%q).Name = %q", name, p.Name)
		}
	}

	_, ok := ProfileByName("nonexistent")
	if ok {
		t.Error("ProfileByName(nonexistent) should return ok=false")
	}
}

func TestMixedGenerator_ProducesAllSizes(t *testing.T) {
	t.Parallel()

	targets := []int{64, 256, 1024}
	g := NewMixedGenerator(7, targets, codec.JSONCodec{})

	seen := make(map[int]bool)

	// Draw enough samples that every size is very likely to appear.
	for range 1000 {
		p := g.Payload()
		data, err := codec.JSONCodec{}.Encode(p)
		if err != nil {
			t.Fatalf("encode failed: %v", err)
		}

		// Round to the nearest configured target so small padding drift
		// (a few bytes) doesn't fragment the bucket.
		bucket := nearest(len(data), targets)
		seen[bucket] = true
	}

	for _, want := range targets {
		if !seen[want] {
			t.Errorf("size %d never appeared in 1000 mixed draws", want)
		}
	}
}

func TestMixedGenerator_MeanSize(t *testing.T) {
	t.Parallel()

	g := NewMixedGenerator(1, []int{64, 256, 4096}, nil)
	if got := g.MeanSize(); got != (64+256+4096)/3 {
		t.Errorf("MeanSize() = %d, want %d", got, (64+256+4096)/3)
	}
}

func TestMixedGenerator_Deterministic(t *testing.T) {
	t.Parallel()

	targets := []int{64, 256, 1024}
	g1 := NewMixedGenerator(99, targets, nil)
	g2 := NewMixedGenerator(99, targets, nil)

	for range 50 {
		a := g1.Payload()
		b := g2.Payload()
		if a.ID != b.ID || a.Padding != b.Padding {
			t.Fatalf("mixed payloads differ at same seed: %+v vs %+v", a, b)
		}
	}
}

func TestMixedGenerator_SingleSizeMatchesNewGenerator(t *testing.T) {
	t.Parallel()

	uni := NewGenerator(5, 512, nil)
	mixed := NewMixedGenerator(5, []int{512}, nil)

	for range 20 {
		a := uni.Payload()
		b := mixed.Payload()
		if a.ID != b.ID || a.Padding != b.Padding {
			t.Fatalf("single-size mixed differs from uniform generator")
		}
	}
}

func TestMixedGenerator_DefaultsOnEmpty(t *testing.T) {
	t.Parallel()

	g := NewMixedGenerator(1, nil, nil)
	if got := g.MeanSize(); got != 256 {
		t.Errorf("empty sizes should default to 256, got %d", got)
	}
}

func nearest(actual int, targets []int) int {
	best := targets[0]
	bestDist := abs(actual - best)

	for _, t := range targets[1:] {
		d := abs(actual - t)
		if d < bestDist {
			best = t
			bestDist = d
		}
	}

	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}

	return n
}
