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

	if p1.ID != p2.ID || p1.Name != p2.Name || p1.Value != p2.Value {
		t.Fatalf("payloads differ: %+v vs %+v", p1, p2)
	}
}

func TestGenerator_DifferentSeeds(t *testing.T) {
	t.Parallel()

	g1 := NewGenerator(1, 256)
	g2 := NewGenerator(2, 256)

	p1 := g1.Payload()
	p2 := g2.Payload()

	if p1.ID == p2.ID {
		t.Error("different seeds produced identical payload IDs")
	}
}

func TestGenerator_PayloadFields(t *testing.T) {
	t.Parallel()

	g := NewGenerator(1, 256)
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

	g := NewGenerator(1, 0) // 0 should default to 256
	p := g.Payload()

	if p.Padding == "" {
		t.Error("Padding should be non-empty with default size 256")
	}
}

func TestGenerator_SmallSizeNoCorruption(t *testing.T) {
	t.Parallel()

	g := NewGenerator(1, 32)
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
