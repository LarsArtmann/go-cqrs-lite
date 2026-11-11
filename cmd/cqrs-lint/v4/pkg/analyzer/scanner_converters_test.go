package analyzer

import "testing"

// The converter-indirection false-positive class for id resolution
// (issue #121). Consumers route message ids through a local single-return
// conversion func, so the constant only exists inside the converter's
// argument:
//
//	func messageID(t string) catalog.MessageID { return catalog.MessageID(t) }
//	catalog.Event[P](messageID(string(EventCreated)), catalog.Sends)
//
// Without inlining, the catalog declaration is invisible and E004 flags
// every emission of a complete, test-pinned catalog (timesheets suppressed
// all of E004 to keep the tool usable).

func TestConverterIndirectionResolvesCatalogIDs(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"catalog.go": `package domain

import (
	catalog "github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

const EventCreated event.Type = "timesheet.created"
const EventSubmitted event.Type = "timesheet.submitted"

func messageID(t string) catalog.MessageID {
	return catalog.MessageID(t)
}

func declare() {
	catalog.Event[struct{}](messageID(string(EventCreated)), catalog.Sends)
	catalog.Event[struct{}](messageID(string(EventSubmitted)), catalog.Receives)
}
`,
	})

	for _, want := range []string{"timesheet.created", "timesheet.submitted"} {
		if !ctx.Registry.EventTypesInCatalog[want] {
			t.Errorf("converter-indirect catalog registration missing %q; catalog=%v",
				want, ctx.Registry.EventTypesInCatalog)
		}
	}
}

// The emission side resolves through the same converter: an event.New id
// routed via messageID(...) lands in EventTypesEmitted after const
// resolution (the full pipeline runs inside BuildContextFromSource).
func TestConverterIndirectionResolvesEmittedIDs(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"emit.go": `package domain

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

const EventCreated event.Type = "timesheet.created"

func messageID(t event.Type) event.Type {
	return event.Type(t)
}

func emit() {
	_, _ = event.New(messageID(EventCreated), nil)
}
`,
	})

	emission, ok := ctx.Registry.EventTypesEmitted["timesheet.created"]
	if !ok || emission.File == "" {
		t.Errorf("converter-indirect emission missing; emitted=%v", ctx.Registry.EventTypesEmitted)
	}
}

// A func with more than one statement is NOT a converter — recording it
// would let unrelated same-named calls inline their arguments.
func TestConverterScanRejectsMultiStatementFuncs(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"convert.go": `package domain

func notAConverter(t string) string {
	s := t + "!"
	return s
}

func alsoNot(t string) string {
	return t + "!"
}
`,
	})

	if len(ctx.Registry.converterFuncParams) != 0 {
		t.Errorf("multi-statement / non-param funcs must not register as converters; got %v",
			ctx.Registry.converterFuncParams)
	}
}
