package analyzer_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// capturePayloadType must see every payload FORM real emit sites use. The
// conversion case (T(x)) came from nsfw-classifier's history domain: its
// media.capture op passed MediaCaptured(cmd.View) — a conversion — and the
// registry's blindness made that package the lone E003 flag while identical
// siblings routed through emit helpers.
func TestCapturePayloadType_ConversionPayload(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

type MediaCaptured struct {
	Hash string
}

func capture(v View) {
	event.New("media.captured", sid, st, version, MediaCaptured(v))
}
`,
	})

	if !ctx.Registry.EventPayloadTypes["MediaCaptured"] {
		t.Error("conversion payload T(x) not recorded in EventPayloadTypes")
	}
}

func TestCapturePayloadType_PointerCompositeAndLiteralStillSeen(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

type RoomCreated struct {
	ID string
}

type DomainObserved struct {
	Domain string
}

func emit() {
	event.New("room.created", sid, st, version, &RoomCreated{ID: "r"})
	event.New("domain.observed", sid, st, version, DomainObserved{Domain: "x"})
}
`,
	})

	if !ctx.Registry.EventPayloadTypes["RoomCreated"] {
		t.Error("&T{...} payload not recorded in EventPayloadTypes")
	}

	if !ctx.Registry.EventPayloadTypes["DomainObserved"] {
		t.Error("T{...} literal payload not recorded in EventPayloadTypes")
	}
}

func TestCapturePayloadType_MultiArgCallIsNotConversion(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

type NotAPayload struct{}

func f(a, b string) string { return a }

func emit() {
	event.New("x.y", sid, st, version, f("a", "b"))
}
`,
	})

	if ctx.Registry.EventPayloadTypes["NotAPayload"] {
		t.Error("multi-arg call wrongly recorded as a conversion payload")
	}
}
