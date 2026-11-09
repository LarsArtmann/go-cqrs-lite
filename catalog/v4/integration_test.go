package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func TestWalkMessages_VisitsAllMessages(t *testing.T) {
	t.Parallel()

	cat := &catalog.Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Services: []catalog.Service{
			{
				ID:       "svc-a",
				Commands: []catalog.Message{{ID: "cmd-a", Name: "Cmd A"}},
				Events:   []catalog.Message{{ID: "evt-a", Name: "Evt A"}},
				Queries:  []catalog.Message{{ID: "qry-a", Name: "Qry A"}},
			},
			{
				ID:       "svc-b",
				Commands: []catalog.Message{{ID: "cmd-b", Name: "Cmd B"}},
			},
		},
	}

	var ids []string

	catalog.WalkMessages(cat, func(svc catalog.Service, msg catalog.Message) bool {
		ids = append(ids, string(msg.ID))

		return true
	})

	want := []string{"cmd-a", "evt-a", "qry-a", "cmd-b"}
	if len(ids) != len(want) {
		t.Fatalf("expected %d messages, got %d: %v", len(want), len(ids), ids)
	}

	for i, w := range want {
		if ids[i] != w {
			t.Errorf("msg[%d] = %q, want %q", i, ids[i], w)
		}
	}
}

func TestWalkMessages_StopsEarly(t *testing.T) {
	t.Parallel()

	cat := &catalog.Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Services: []catalog.Service{
			{
				ID:       "svc",
				Commands: []catalog.Message{{ID: "cmd"}},
				Events:   []catalog.Message{{ID: "evt"}},
				Queries:  []catalog.Message{{ID: "qry"}},
			},
		},
	}

	var ids []string

	catalog.WalkMessages(cat, func(svc catalog.Service, msg catalog.Message) bool {
		if msg.ID == "evt" {
			return false
		}

		ids = append(ids, string(msg.ID))

		return true
	})

	if len(ids) != 1 || ids[0] != "cmd" {
		t.Errorf("expected [cmd], got %v", ids)
	}
}

func TestWalkMessages_EmptyCatalog(t *testing.T) {
	t.Parallel()

	cat := &catalog.Catalog{Title: "Empty", Version: "1.0.0"}

	called := false
	catalog.WalkMessages(cat, func(svc catalog.Service, msg catalog.Message) bool {
		called = true

		return true
	})

	if called {
		t.Error("expected fn not to be called for empty catalog")
	}
}
