package query_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func TestQuery_MetadataMerge(t *testing.T) {
	t.Parallel()

	base := query.NewMetadata()
	base.CorrelationID = id.NewCorrelationID()
	query.EnsureCustom(&base)
	base.Custom["tenant"] = "acme"

	overlay := query.NewMetadata()
	overlay.UserID = id.NewUserID()
	query.EnsureCustom(&overlay)
	overlay.Custom["region"] = "us-east-1"

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

	md := query.NewMetadata()
	query.EnsureCustom(&md)
	md.Custom["source"] = "test"

	if md.Custom["source"] != "test" {
		t.Errorf("Custom write/read failed: got %q", md.Custom["source"])
	}
}
