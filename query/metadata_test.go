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

func TestQuery_WithActor(t *testing.T) {
	t.Parallel()

	actor := id.NewSystemActor("scheduler")
	q, err := query.New("GetUser", query.WithActor(actor))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if !q.Metadata().ActorID.Equal(actor) {
		t.Errorf("ActorID = %s, want %s",
			q.Metadata().ActorID.PrefixedString(), actor.PrefixedString())
	}
}

func TestQuery_AllMetadata(t *testing.T) {
	t.Parallel()

	cid := id.NewCorrelationID()
	caid := id.NewCausationID()
	uid := id.NewUserID()
	rid := id.NewRequestID()
	actor := id.NewBotActor("ci-runner")

	q, err := query.New("GetUser",
		query.WithCorrelationID(cid),
		query.WithCausationID(caid),
		query.WithUserID(uid),
		query.WithRequestID(rid),
		query.WithActor(actor),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	m := q.Metadata()
	if m.CorrelationID != cid {
		t.Errorf("CorrelationID = %v, want %v", m.CorrelationID, cid)
	}

	if m.CausationID != caid {
		t.Errorf("CausationID = %v, want %v", m.CausationID, caid)
	}

	if m.UserID != uid {
		t.Errorf("UserID = %v, want %v", m.UserID, uid)
	}

	if m.RequestID != rid {
		t.Errorf("RequestID = %v, want %v", m.RequestID, rid)
	}

	if !m.ActorID.Equal(actor) {
		t.Errorf("ActorID = %s, want %s", m.ActorID.PrefixedString(), actor.PrefixedString())
	}
}

func TestQuery_WithCustomMetadata(t *testing.T) {
	t.Parallel()

	t.Run("single custom entry", func(t *testing.T) {
		t.Parallel()

		q, err := query.New("test.query",
			query.WithCustomMetadata("tenant", "acme"),
		)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		md := q.Metadata()
		if md.Custom["tenant"] != "acme" {
			t.Errorf("Custom[tenant] = %q, want %q", md.Custom["tenant"], "acme")
		}
	})

	t.Run("multiple calls accumulate", func(t *testing.T) {
		t.Parallel()

		q, err := query.New("test.query",
			query.WithCustomMetadata("tenant", "acme"),
			query.WithCustomMetadata("region", "us-east-1"),
		)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		md := q.Metadata()
		if md.Custom["tenant"] != "acme" {
			t.Errorf("Custom[tenant] = %q, want %q", md.Custom["tenant"], "acme")
		}

		if md.Custom["region"] != "us-east-1" {
			t.Errorf("Custom[region] = %q, want %q", md.Custom["region"], "us-east-1")
		}
	})
}
