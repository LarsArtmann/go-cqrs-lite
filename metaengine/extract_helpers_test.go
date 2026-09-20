package metaengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// plannedTaskView mirrors the 2026-09-18 Ledger CRM shape that exposed the
// camelCase-pushdown bug: filterable fields registered by their GO names
// while the stored documents key them by snake_case json tags.
type plannedTaskView struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	DueAt    string `json:"due_at"`
	URLKey   string `json:"url_key"`
	Title    string `json:"title"`
}

func TestExtractFieldsGoNameColumnsOverSnakeTags(t *testing.T) {
	t.Parallel()

	view := plannedTaskView{
		ID:       "t1",
		ParentID: "p1",
		DueAt:    "2026-09-20",
		URLKey:   "k1",
		Title:    "ship it",
	}

	goNameColumns := []metaengine.PlannedColumn{
		{Name: "ID", Type: "TEXT"},
		{Name: "ParentID", Type: "TEXT"},
		{Name: "DueAt", Type: "TEXT"},
		{Name: "URLKey", Type: "TEXT"},
		{Name: "Title", Type: "TEXT"},
	}

	got := metaengine.ExtractFields(view, goNameColumns)
	for _, col := range goNameColumns {
		if got[col.Name] == nil {
			t.Errorf(
				"column %s extracted as nil — camelCase filterable fields over snake_case json tags silently NULL again (the 2026-09-18 CRM pushdown bug)",
				col.Name,
			)
		}
	}

	if got["ParentID"] != "p1" || got["DueAt"] != "2026-09-20" || got["URLKey"] != "k1" {
		t.Errorf("wrong values extracted: %+v", got)
	}
}

func TestExtractFieldsTagColumnsStillMatch(t *testing.T) {
	t.Parallel()

	view := plannedTaskView{ParentID: "p1"}

	tagColumns := []metaengine.PlannedColumn{
		{Name: "parent_id", Type: "TEXT"},
		{Name: "due_at", Type: "TEXT"},
	}

	got := metaengine.ExtractFields(view, tagColumns)
	if got["parent_id"] != "p1" {
		t.Errorf("json-tag-named column must keep matching the tag: %+v", got)
	}
	if got["due_at"] != "" {
		t.Errorf("zero-value fields extract as typed zero values, not nil: %+v", got)
	}
}

func TestExtractFieldsMapPathSnakeFallback(t *testing.T) {
	t.Parallel()

	doc := map[string]any{ // the decoded document: json-tag keys
		"parent_id": "p2",
		"due_at":    "2026-09-21",
	}

	goNameColumns := []metaengine.PlannedColumn{
		{Name: "ParentID", Type: "TEXT"},
		{Name: "DueAt", Type: "TEXT"},
		{Name: "Missing", Type: "TEXT"},
	}

	got := metaengine.ExtractFields(doc, goNameColumns)
	if got["ParentID"] != "p2" {
		t.Errorf("map path must fall back to snake_case(column): %+v", got)
	}
	if got["DueAt"] != "2026-09-21" {
		t.Errorf("map path snake fallback for DueAt: %+v", got)
	}
	if got["Missing"] != nil {
		t.Errorf("absent fields must stay nil: %+v", got)
	}
}
