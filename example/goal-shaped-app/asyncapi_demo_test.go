package main

import (
	"strings"
	"testing"

	catalog "github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/asyncapi"
)

// TestDocs_AsyncAPIExport demos the "docs generate themselves" leg of the
// Goal: the same declared commands/events/queries that run the app also
// export a valid AsyncAPI 3.0 document — no second source of truth to keep
// in sync.
func TestDocs_AsyncAPIExport(t *testing.T) {
	t.Parallel()

	r := catalog.NewRegistry("goal-shaped-app", "1.0.0")
	r.AddCommand("tasks", catalog.Message{Name: "task.create", Summary: "Create a task"})
	r.AddCommand("tasks", catalog.Message{Name: "task.complete", Summary: "Complete a task"})
	r.AddCommand("tasks", catalog.Message{Name: "task.delete", Summary: "Delete a task"})

	r.AddEvent("tasks", catalog.Message{Name: "task.created", Summary: "A task was created"})
	r.AddEvent("tasks", catalog.Message{Name: "task.updated", Summary: "A task was updated"})
	r.AddEvent("tasks", catalog.Message{Name: "task.deleted", Summary: "A task was deleted (ADR-0114 tombstone)"})

	r.AddQuery("tasks", catalog.Message{Name: "task.get", Summary: "Fetch one task view"})
	r.AddQuery("tasks", catalog.Message{Name: "task.open", Summary: "List open task views"})

	doc := asyncapi.NewExporter("goal-shaped-app", "1.0.0").Export(r.Build())

	json, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	out := string(json)

	for _, want := range []string{
		"task.created", "task.updated", "task.deleted",
		"task.create", "task.get", "asyncapi",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("AsyncAPI document missing %q", want)
		}
	}

	yaml, err := doc.MarshalYAML()
	if err != nil || len(yaml) == 0 {
		t.Fatalf("MarshalYAML: %v (len=%d)", err, len(yaml))
	}
}
