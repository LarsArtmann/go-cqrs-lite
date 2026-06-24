package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestMergeCustomMaps_BothNil(t *testing.T) {
	t.Parallel()

	result := event.MergeCustomMaps[event.MetadataKey](nil, nil)
	if result != nil {
		t.Errorf("MergeCustomMaps(nil, nil) = %v, want nil", result)
	}
}

func TestMergeCustomMaps_NilOtherReturnsBaseUnchanged(t *testing.T) {
	t.Parallel()

	base := map[event.MetadataKey]string{"a": "1"}
	result := event.MergeCustomMaps(base, nil)

	result["a"] = "modified"
	if base["a"] != "modified" {
		t.Error(
			"MergeCustomMaps should return the original base map when other is nil (no allocation)",
		)
	}
}

func TestMergeCustomMaps_EmptyOtherReturnsBaseUnchanged(t *testing.T) {
	t.Parallel()

	base := map[event.MetadataKey]string{"a": "1"}
	result := event.MergeCustomMaps(base, map[event.MetadataKey]string{})

	if len(result) != 1 || result["a"] != "1" {
		t.Errorf("MergeCustomMaps with empty other should preserve base, got %v", result)
	}
}

func TestMergeCustomMaps_OverlaysOther(t *testing.T) {
	t.Parallel()

	base := map[event.MetadataKey]string{"a": "1", "b": "2"}
	other := map[event.MetadataKey]string{"b": "3", "c": "4"}

	result := event.MergeCustomMaps(base, other)

	if result["a"] != "1" {
		t.Errorf("base entry a lost: got %q", result["a"])
	}

	if result["b"] != "3" {
		t.Errorf("overlay should win for b: got %q", result["b"])
	}

	if result["c"] != "4" {
		t.Errorf("new entry c missing: got %q", result["c"])
	}
}

func TestMergeCustomMaps_DoesNotMutateBase(t *testing.T) {
	t.Parallel()

	base := map[event.MetadataKey]string{"a": "1"}
	other := map[event.MetadataKey]string{"b": "2"}

	result := event.MergeCustomMaps(base, other)

	if _, ok := base["b"]; ok {
		t.Error("MergeCustomMaps mutated the base map")
	}

	result["a"] = "modified"
	if base["a"] != "1" {
		t.Error("MergeCustomMaps result aliases the base map")
	}
}

func TestCustomData_Clone_NilCustom(t *testing.T) {
	t.Parallel()

	d := event.CustomData[event.MetadataKey]{}
	cp := d.Clone()

	if cp.Custom != nil {
		t.Errorf("Clone of nil Custom should be nil, got %v", cp.Custom)
	}
}

func TestCustomData_Clone_DeepCopy(t *testing.T) {
	t.Parallel()

	d := event.CustomData[event.MetadataKey]{
		Custom: map[event.MetadataKey]string{"key": "value"},
	}
	cp := d.Clone()

	cp.Custom["key"] = "modified"
	if d.Custom["key"] != "value" {
		t.Error("Clone should produce a deep copy of Custom")
	}
}

func TestCustomData_Merge_BothZero(t *testing.T) {
	t.Parallel()

	result := event.CustomData[event.MetadataKey]{}.Merge(event.CustomData[event.MetadataKey]{})
	if result.Custom != nil {
		t.Errorf("Merge of two zero CustomData should have nil Custom, got %v", result.Custom)
	}
}

func TestCustomData_Merge_OverlaysTracingAndCustom(t *testing.T) {
	t.Parallel()

	cid := id.NewCorrelationID()

	base := event.CustomData[event.MetadataKey]{
		Tracing: event.Tracing{CorrelationID: cid},
		Custom:  map[event.MetadataKey]string{"tenant": "acme"},
	}
	other := event.CustomData[event.MetadataKey]{
		Tracing: event.Tracing{UserID: id.NewUserID()},
		Custom:  map[event.MetadataKey]string{"region": "us-east-1"},
	}

	result := base.Merge(other)

	if result.CorrelationID != cid {
		t.Errorf("base CorrelationID not preserved: got %v", result.CorrelationID)
	}

	if result.UserID.IsZero() {
		t.Error("overlay UserID not merged")
	}

	if result.Custom["tenant"] != "acme" {
		t.Errorf("base Custom lost: tenant = %q", result.Custom["tenant"])
	}

	if result.Custom["region"] != "us-east-1" {
		t.Errorf("overlay Custom not copied: region = %q", result.Custom["region"])
	}
}

func TestCustomData_Merge_DoesNotMutateBase(t *testing.T) {
	t.Parallel()

	base := event.CustomData[event.MetadataKey]{
		Custom: map[event.MetadataKey]string{"a": "1"},
	}
	other := event.CustomData[event.MetadataKey]{
		Custom: map[event.MetadataKey]string{"b": "2"},
	}

	_ = base.Merge(other)

	if _, ok := base.Custom["b"]; ok {
		t.Error("Merge mutated the base Custom map")
	}
}

func TestCustomData_EnsureCustom(t *testing.T) {
	t.Parallel()

	t.Run("initializes nil map", func(t *testing.T) {
		t.Parallel()

		var d event.CustomData[event.MetadataKey]
		d.EnsureCustom()

		if d.Custom == nil {
			t.Error("EnsureCustom should initialize the Custom map")
		}
	})

	t.Run("preserves existing entries", func(t *testing.T) {
		t.Parallel()

		d := event.CustomData[event.MetadataKey]{
			Custom: map[event.MetadataKey]string{"existing": "value"},
		}
		d.EnsureCustom()

		if d.Custom["existing"] != "value" {
			t.Error("EnsureCustom should preserve existing entries")
		}
	})
}
