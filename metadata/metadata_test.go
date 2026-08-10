package metadata

import (
	"maps"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// testKey is a named string type for testing the generic CustomData[K].
type testKey string

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
			CommonMetadata: record.CommonMetadata{CorrelationID: id.NewCorrelationID()},
			Custom:         map[key]string{"a": "1", "b": "2"},
		}
		cloned := d.Clone()

		if cloned.Custom["a"] != "1" || cloned.Custom["b"] != "2" {
			t.Errorf("cloned Custom missing entries: %v", cloned.Custom)
		}

		cloned.Custom["a"] = "modified"

		if d.Custom["a"] != "1" {
			t.Error("original Custom should not be affected by clone mutation")
		}

		if cloned.CorrelationID != d.CorrelationID {
			t.Error("CommonMetadata should be copied")
		}
	})
}

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

	t.Run("other overlays common metadata and custom", func(t *testing.T) {
		t.Parallel()

		base := CustomData[key]{
			CommonMetadata: record.CommonMetadata{ActorID: id.NewUserActor(id.NewUserID())},
			Custom:         map[key]string{"keep": "base", "override": "base-val"},
		}
		corrOther := id.NewCorrelationID()
		other := CustomData[key]{
			CommonMetadata: record.CommonMetadata{CorrelationID: corrOther},
			Custom:         map[key]string{"override": "other-val", "new": "added"},
		}
		merged := base.Merge(other)

		if merged.CorrelationID != corrOther {
			t.Error("CorrelationID should come from other")
		}

		if merged.ActorID.IsZero() {
			t.Error("ActorID should survive from base")
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

func TestCustomData_WithCustom(t *testing.T) {
	t.Parallel()

	type key = testKey

	t.Run("returns new value with key set", func(t *testing.T) {
		t.Parallel()

		d := CustomData[key]{}
		updated := d.WithCustom("role", "admin")

		if updated.Custom["role"] != "admin" {
			t.Errorf("Custom[role] = %q, want %q", updated.Custom["role"], "admin")
		}
	})

	t.Run("original is not modified", func(t *testing.T) {
		t.Parallel()

		d := CustomData[key]{Custom: map[key]string{"existing": "val"}}
		_ = d.WithCustom("new", "val2")

		if _, ok := d.Custom["new"]; ok {
			t.Error("original should not have the new key")
		}

		if len(d.Custom) != 1 {
			t.Errorf("original Custom should have 1 entry, got %d", len(d.Custom))
		}
	})

	t.Run("nil map is initialized lazily", func(t *testing.T) {
		t.Parallel()

		d := CustomData[key]{}
		updated := d.WithCustom("k", "v")

		if updated.Custom == nil {
			t.Fatal("Custom should be non-nil")
		}
	})

	t.Run("existing entries are preserved", func(t *testing.T) {
		t.Parallel()

		d := CustomData[key]{Custom: map[key]string{"a": "1", "b": "2"}}
		updated := d.WithCustom("c", "3")

		if updated.Custom["a"] != "1" || updated.Custom["b"] != "2" {
			t.Error("existing entries should survive")
		}
	})

	t.Run("overrides existing key", func(t *testing.T) {
		t.Parallel()

		d := CustomData[key]{Custom: map[key]string{"k": "old"}}
		updated := d.WithCustom("k", "new")

		if updated.Custom["k"] != "new" {
			t.Errorf("Custom[k] = %q, want %q", updated.Custom["k"], "new")
		}

		if d.Custom["k"] != "old" {
			t.Error("original should keep old value")
		}
	})
}

func TestMergeCustomMaps(t *testing.T) {
	t.Parallel()

	type key = testKey

	t.Run("empty other returns base unchanged", func(t *testing.T) {
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
