package metadata

import (
	"maps"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// testKey is a named string type for testing the generic CustomData[K].
type testKey string

// TestTracing_IsZero covers the zero-value check for all four tracing fields.
func TestTracing_IsZero(t *testing.T) {
	t.Parallel()

	t.Run("empty is zero", func(t *testing.T) {
		t.Parallel()
		if !(Tracing{}).IsZero() {
			t.Error("zero-value Tracing should be zero")
		}
	})

	t.Run("correlationID set is non-zero", func(t *testing.T) {
		t.Parallel()
		tr := Tracing{CorrelationID: id.NewCorrelationID()}
		if tr.IsZero() {
			t.Error("Tracing with CorrelationID should be non-zero")
		}
	})

	t.Run("causationID set is non-zero", func(t *testing.T) {
		t.Parallel()
		tr := Tracing{CausationID: id.NewCausationID()}
		if tr.IsZero() {
			t.Error("Tracing with CausationID should be non-zero")
		}
	})

	t.Run("userID set is non-zero", func(t *testing.T) {
		t.Parallel()
		tr := Tracing{UserID: id.NewUserID()}
		if tr.IsZero() {
			t.Error("Tracing with UserID should be non-zero")
		}
	})

	t.Run("requestID set is non-zero", func(t *testing.T) {
		t.Parallel()
		tr := Tracing{RequestID: id.NewRequestID()}
		if tr.IsZero() {
			t.Error("Tracing with RequestID should be non-zero")
		}
	})
}

// TestTracing_Merge covers overlay semantics: non-zero fields from other win.
func TestTracing_Merge(t *testing.T) {
	t.Parallel()

	corr1 := id.NewCorrelationID()
	caus1 := id.NewCausationID()
	user1 := id.NewUserID()
	req1 := id.NewRequestID()

	corr2 := id.NewCorrelationID()

	t.Run("other all-zero returns base unchanged", func(t *testing.T) {
		t.Parallel()
		base := Tracing{CorrelationID: corr1, CausationID: caus1}
		merged := base.Merge(Tracing{})

		if merged.CorrelationID != corr1 {
			t.Error("CorrelationID should survive merge with zero other")
		}

		if merged.CausationID != caus1 {
			t.Error("CausationID should survive merge with zero other")
		}
	})

	t.Run("other overlays non-zero fields", func(t *testing.T) {
		t.Parallel()
		base := Tracing{CorrelationID: corr1, UserID: user1}
		other := Tracing{CorrelationID: corr2, RequestID: req1}
		merged := base.Merge(other)

		if merged.CorrelationID != corr2 {
			t.Error("CorrelationID should be overlaid by other")
		}

		if merged.UserID != user1 {
			t.Error("UserID should survive (other has zero)")
		}

		if merged.RequestID != req1 {
			t.Error("RequestID should come from other")
		}
	})

	t.Run("full overlay", func(t *testing.T) {
		t.Parallel()
		base := Tracing{CorrelationID: corr1, CausationID: caus1, UserID: user1, RequestID: req1}
		allNew := Tracing{
			CorrelationID: corr2,
			CausationID:   id.NewCausationID(),
			UserID:        id.NewUserID(),
			RequestID:     id.NewRequestID(),
		}
		merged := base.Merge(allNew)

		if merged.CorrelationID != allNew.CorrelationID {
			t.Error("CorrelationID should be fully overlaid")
		}

		if merged.CausationID != allNew.CausationID {
			t.Error("CausationID should be fully overlaid")
		}

		if merged.UserID != allNew.UserID {
			t.Error("UserID should be fully overlaid")
		}

		if merged.RequestID != allNew.RequestID {
			t.Error("RequestID should be fully overlaid")
		}
	})

	t.Run("merge is non-destructive to base", func(t *testing.T) {
		t.Parallel()
		base := Tracing{CorrelationID: corr1}
		_ = base.Merge(Tracing{CorrelationID: corr2})

		if base.CorrelationID != corr1 {
			t.Error("base should not be mutated by Merge")
		}
	})
}

// TestCustomData_Clone covers nil and populated custom maps.
func TestCustomData_Clone(t *testing.T) {
	t.Parallel()

	type key = testKey

	t.Run("nil custom map", func(t *testing.T) {
		t.Parallel()
		d := CustomData[key]{}
		cloned := d.Clone()

		if cloned.Custom != nil {
			t.Errorf("expected nil Custom after clone, got %v", cloned.Custom)
		}
	})

	t.Run("populated custom map is independent", func(t *testing.T) {
		t.Parallel()
		d := CustomData[key]{
			Tracing: Tracing{CorrelationID: id.NewCorrelationID()},
			Custom:  map[key]string{"a": "1", "b": "2"},
		}
		cloned := d.Clone()

		if cloned.Custom["a"] != "1" || cloned.Custom["b"] != "2" {
			t.Errorf("cloned Custom missing entries: %v", cloned.Custom)
		}

		// Mutate clone, original should be unaffected.
		cloned.Custom["a"] = "modified"

		if d.Custom["a"] != "1" {
			t.Error("original Custom should not be affected by clone mutation")
		}

		if cloned.CorrelationID != d.CorrelationID {
			t.Error("Tracing should be copied")
		}
	})
}

// TestCustomData_Merge covers tracing + custom merge semantics.
func TestCustomData_Merge(t *testing.T) {
	t.Parallel()

	type key = testKey

	t.Run("nil custom maps on both sides", func(t *testing.T) {
		t.Parallel()
		base := CustomData[key]{}
		other := CustomData[key]{}
		merged := base.Merge(other)

		if merged.Custom != nil {
			t.Errorf("expected nil Custom, got %v", merged.Custom)
		}
	})

	t.Run("other overlays tracing and custom", func(t *testing.T) {
		t.Parallel()
		base := CustomData[key]{
			Tracing: Tracing{UserID: id.NewUserID()},
			Custom:  map[key]string{"keep": "base", "override": "base-val"},
		}
		corrOther := id.NewCorrelationID()
		other := CustomData[key]{
			Tracing: Tracing{CorrelationID: corrOther},
			Custom:  map[key]string{"override": "other-val", "new": "added"},
		}
		merged := base.Merge(other)

		if merged.CorrelationID != corrOther {
			t.Error("CorrelationID should come from other")
		}

		if merged.UserID.IsZero() {
			t.Error("UserID should survive from base")
		}

		if merged.Custom["keep"] != "base" {
			t.Error("base-only entry should survive")
		}

		if merged.Custom["override"] != "other-val" {
			t.Error("other should override existing key")
		}

		if merged.Custom["new"] != "added" {
			t.Error("new entry from other should be present")
		}
	})

	t.Run("merge is non-destructive", func(t *testing.T) {
		t.Parallel()
		base := CustomData[key]{Custom: map[key]string{"a": "1"}}
		_ = base.Merge(CustomData[key]{Custom: map[key]string{"a": "2"}})

		if base.Custom["a"] != "1" {
			t.Error("base should not be mutated")
		}
	})
}

// TestCustomData_EnsureCustom covers lazy map initialization.
func TestCustomData_EnsureCustom(t *testing.T) {
	t.Parallel()

	type key = testKey

	t.Run("nil map becomes initialized", func(t *testing.T) {
		t.Parallel()
		d := &CustomData[key]{}
		d.EnsureCustom()

		if d.Custom == nil {
			t.Error("Custom should be non-nil after EnsureCustom")
		}

		if len(d.Custom) != 0 {
			t.Errorf("Custom should be empty, got %d entries", len(d.Custom))
		}
	})

	t.Run("existing map preserved", func(t *testing.T) {
		t.Parallel()
		existing := map[key]string{"x": "y"}
		d := &CustomData[key]{Custom: existing}
		d.EnsureCustom()

		if d.Custom["x"] != "y" {
			t.Error("existing entry should be preserved")
		}
	})
}

// TestMergeCustomMaps covers the standalone merge helper.
func TestMergeCustomMaps(t *testing.T) {
	t.Parallel()

	type key = testKey

	t.Run("empty other returns base unchanged (no allocation)", func(t *testing.T) {
		t.Parallel()
		base := map[key]string{"a": "1"}
		result := MergeCustomMaps(base, nil)

		if !maps.Equal(result, base) {
			t.Errorf("expected base returned unchanged, got %v", result)
		}
	})

	t.Run("empty other with empty base", func(t *testing.T) {
		t.Parallel()
		result := MergeCustomMaps[key](nil, nil)
		// MergeCustomMaps with empty other returns base directly.
		// When other is empty AND base is nil, result is nil.
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("both populated", func(t *testing.T) {
		t.Parallel()
		base := map[key]string{"a": "1", "b": "2"}
		other := map[key]string{"b": "3", "c": "4"}
		result := MergeCustomMaps(base, other)

		if result["a"] != "1" {
			t.Error("base-only key should survive")
		}

		if result["b"] != "3" {
			t.Error("other should override shared key")
		}

		if result["c"] != "4" {
			t.Error("new key from other should be present")
		}
	})

	t.Run("result is independent from inputs", func(t *testing.T) {
		t.Parallel()
		base := map[key]string{"a": "1"}
		other := map[key]string{"b": "2"}
		result := MergeCustomMaps(base, other)

		result["a"] = "modified"
		if base["a"] != "1" {
			t.Error("base should not be affected by result mutation")
		}
	})
}
