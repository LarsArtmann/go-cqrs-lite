package query_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func TestQuery_MetadataMerge(t *testing.T) {
	t.Parallel()

	base := query.Metadata{
		Custom: map[query.MetadataKey]string{"tenant": "acme"},
	}
	base.CorrelationID = id.NewCorrelationID()

	overlay := query.Metadata{
		Custom: map[query.MetadataKey]string{"region": "us-east-1"},
	}
	overlay.UserID = id.NewUserID()

	merged := base.Merge(overlay)

	if merged.CorrelationID != base.CorrelationID {
		t.Errorf("CorrelationID not preserved: got %v, want %v",
			merged.CorrelationID, base.CorrelationID)
	}

	if merged.UserID != overlay.UserID {
		t.Errorf("UserID not overlaid: got %v, want %v", merged.UserID, overlay.UserID)
	}

	if merged.Custom["tenant"] != "acme" {
		t.Errorf("base Custom lost: tenant = %q", merged.Custom["tenant"])
	}

	if merged.Custom["region"] != "us-east-1" {
		t.Errorf("overlay Custom not copied: region = %q", merged.Custom["region"])
	}

	if _, ok := base.Custom["region"]; ok {
		t.Error("merge mutated the base Custom map")
	}
}

func TestQuery_MetadataKeyIsLocal(t *testing.T) {
	t.Parallel()

	md := query.Metadata{
		Custom: map[query.MetadataKey]string{"source": "test"},
	}

	if md.Custom["source"] != "test" {
		t.Errorf("Custom write/read failed: got %q", md.Custom["source"])
	}
}
