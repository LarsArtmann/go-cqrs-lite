package metadata

import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"maps"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
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

	t.Run("actorID set is non-zero", func(t *testing.T) {
		t.Parallel()
		tr := Tracing{ActorID: id.NewSystemActor("scheduler")}
		if tr.IsZero() {
			t.Error("Tracing with ActorID should be non-zero")
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

	t.Run("actorID overlays in merge", func(t *testing.T) {
		t.Parallel()
		base := Tracing{CorrelationID: corr1}
		actor := id.NewSystemActor("scheduler")
		other := Tracing{ActorID: actor}
		merged := base.Merge(other)

		if !merged.ActorID.Equal(actor) {
			t.Error("ActorID should come from other")
		}

		if merged.CorrelationID != corr1 {
			t.Error("CorrelationID should survive (other has zero)")
		}
	})

	t.Run("zero actorID in other does not clear base", func(t *testing.T) {
		t.Parallel()
		actor := id.NewBotActor("ci-bot")
		base := Tracing{ActorID: actor}
		merged := base.Merge(Tracing{CorrelationID: corr1})

		if !merged.ActorID.Equal(actor) {
			t.Error("ActorID should survive merge with zero other")
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
			ActorID:       id.NewServiceActor("api-gateway"),
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

		if !merged.ActorID.Equal(allNew.ActorID) {
			t.Error("ActorID should be fully overlaid")
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

// TestCustomData_WithCustom covers the functional (non-mutating) custom setter.
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

// TestTracing_JSON covers JSON serialization of Tracing, especially the
// omitempty behavior of ActorID when zero vs set.
func TestTracing_JSON(t *testing.T) {
	t.Parallel()

	t.Run("zero ActorID is omitted from JSON", func(t *testing.T) {
		t.Parallel()
		tr := Tracing{}
		data, err := json.Marshal(tr)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		if strings.Contains(string(data), "actorId") {
			t.Errorf("zero ActorID should be omitted, got %s", data)
		}
	})

	t.Run("set ActorID appears in JSON", func(t *testing.T) {
		t.Parallel()
		tr := Tracing{ActorID: id.NewSystemActor("scheduler")}
		data, err := json.Marshal(tr)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		expected := `"actorId":"system:scheduler"`
		if !strings.Contains(string(data), expected) {
			t.Errorf("expected %s in JSON, got %s", expected, data)
		}
	})

	t.Run("roundtrip preserves ActorID", func(t *testing.T) {
		t.Parallel()
		original := Tracing{ActorID: id.NewBotActor("ci-runner")}
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		var decoded Tracing
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if !decoded.ActorID.Equal(original.ActorID) {
			t.Errorf("roundtrip mismatch: got %v, want %v", decoded.ActorID, original.ActorID)
		}
	})
}

// TestTracing_JSONv1Fallback covers the same serialization under
// encoding/json v1, which supports the omitzero tag since Go 1.24 and
// delegates to ActorID.IsZero. Consumers building without the jsonv2
// experiment still get the same wire shape.
func TestTracing_JSONv1Fallback(t *testing.T) {
	t.Parallel()

	t.Run("zero ActorID is omitted", func(t *testing.T) {
		t.Parallel()
		data, err := jsonv1.Marshal(Tracing{})
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		if strings.Contains(string(data), "actorId") {
			t.Errorf("zero ActorID should be omitted under json/v1, got %s", data)
		}
	})

	t.Run("set ActorID appears and roundtrips", func(t *testing.T) {
		t.Parallel()
		original := Tracing{ActorID: id.NewUserActor(id.NewUserID())}
		data, err := jsonv1.Marshal(original)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		if !strings.Contains(string(data), `"actorId":"user:`) {
			t.Errorf("expected prefixed actorId in json/v1 output, got %s", data)
		}

		var decoded Tracing
		if err := jsonv1.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if !decoded.ActorID.Equal(original.ActorID) {
			t.Errorf("roundtrip mismatch: got %v, want %v",
				decoded.ActorID.PrefixedString(), original.ActorID.PrefixedString())
		}
	})
}
