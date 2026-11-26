package main

import (
	"strings"
	"testing"
)

const versionedSources = "package tasks\n\n" +
	"//cqrs:event task.created\ntype TaskCreated struct{}\n\n" +
	"//cqrs:event invoice.received receives v3\ntype InvoiceReceived struct{}\n"

// An event marker's schema version is stamped by the generated constructor and
// reaches the catalog as the message version; without one the constructor
// still stamps version 1, so upcasting can always be added later.
func TestGenerate_EventSchemaVersion(t *testing.T) {
	t.Parallel()

	pkgs := scanFixture(t, map[string]string{"events.go": versionedSources})

	wantVersions := map[string]int{"TaskCreated": 1, "InvoiceReceived": 3}
	for _, m := range pkgs[0].markers {
		if m.schemaVersion != wantVersions[m.structName] {
			t.Errorf("%s: schema version = %d, want %d", m.structName, m.schemaVersion, wantVersions[m.structName])
		}
	}

	code := mustGenerate(t, pkgs, genOptions{target: targetSystem, catalog: true})

	for _, want := range []string{
		"// NewTaskCreatedEvent builds the task.created event for a stream at schema version 1;",
		"options = append(options, event.WithSchemaVersion(1))",
		"// NewInvoiceReceivedEvent builds the invoice.received event for a stream at schema version 3;",
		"options = append(options, event.WithSchemaVersion(3))",
		"catalog.Event[TaskCreated](catalog.MessageID(EventTaskCreated), catalog.Sends, opts...),",
		"catalog.Event[InvoiceReceived](catalog.MessageID(EventInvoiceReceived), catalog.Receives,\n" +
			"\t\t\tappend([]catalog.MessageOption{catalog.WithVersion(\"3.0.0\")}, opts...)...),",
	} {
		if !strings.Contains(code, want) {
			t.Errorf("missing %s in:\n%s", want, code)
		}
	}

	if strings.Count(code, "catalog.WithVersion(") != 1 {
		t.Errorf("only a bumped schema version reaches the catalog, got:\n%s", code)
	}

	// A caller option comes last, so it can still override the stamped version.
	constructor := code[strings.Index(code, "func NewTaskCreatedEvent("):]
	constructor = constructor[:strings.Index(constructor, "\n}\n")]

	optionsAt := strings.Index(constructor, "options = append(options, opts...)")
	stampAt := strings.Index(constructor, "event.WithSchemaVersion(1)")
	newAt := strings.Index(constructor, "event.New(")

	if stampAt >= optionsAt || optionsAt >= newAt {
		t.Errorf("stamp, then caller options, then event.New; got:\n%s", constructor)
	}
}
