package benchkit

import (
	"testing"
)

func TestGenerator_Deterministic(t *testing.T) {
	t.Parallel()

	g1 := NewGenerator(42, 256)
	g2 := NewGenerator(42, 256)

	p1 := g1.Payload()
	p2 := g2.Payload()

	if len(p1) != len(p2) {
		t.Fatalf("len mismatch: %d vs %d", len(p1), len(p2))
	}

	for i := range p1 {
		if p1[i] != p2[i] {
			t.Fatalf("byte mismatch at %d: %v vs %v", i, p1[i], p2[i])
		}
	}
}

func TestGenerator_DifferentSeeds(t *testing.T) {
	t.Parallel()

	g1 := NewGenerator(1, 256)
	g2 := NewGenerator(2, 256)

	p1 := g1.Payload()
	p2 := g2.Payload()

	same := true

	for i := range p1 {
		if p1[i] != p2[i] {
			same = false
			break
		}
	}

	if same {
		t.Error("different seeds produced identical payloads")
	}
}

func TestGenerator_PayloadSize(t *testing.T) {
	t.Parallel()

	sizes := []int{64, 128, 256, 512, 1024, 4096}

	for _, size := range sizes {
		g := NewGenerator(1, size)
		p := g.Payload()

		if len(p) != size {
			t.Errorf("size=%d: got len=%d, want %d", size, len(p), size)
		}
	}
}

func TestGenerator_DefaultSize(t *testing.T) {
	t.Parallel()

	g := NewGenerator(1, 0) // 0 should default to 256
	p := g.Payload()

	if len(p) != 256 {
		t.Errorf("default size: got len=%d, want 256", len(p))
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
