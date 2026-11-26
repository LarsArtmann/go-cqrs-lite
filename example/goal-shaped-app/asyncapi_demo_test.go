package main

import (
	"strings"
	"testing"

	catalog "github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/asyncapi"
)

// TestDocs_AsyncAPIExport demos the "docs generate themselves" leg of the
// Goal: the marked commands/events/queries that run the app also export a
// valid AsyncAPI 3.0 document through the cqrs-gen declarations — one
// source of truth, payload schemas included.
func TestDocs_AsyncAPIExport(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("goal-shaped-app", "1.0.0")
	b.AddService("tasks", "Tasks", "1.0.0", "The task domain", CatalogMessages()...)

	doc := asyncapi.NewExporter("goal-shaped-app", "1.0.0").Export(b.Build())

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
