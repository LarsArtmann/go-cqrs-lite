package correctness_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// stubEventPkg is a local stub whose import path contains the
// "go-cqrs-lite/event" qualifier fragment IsQualifierFor matches on, so
// the typed fixture resolves event.New exactly like production code does.
const stubEventPkg = `package event

// New mirrors the real constructor's positional shape: the payload is the
// 5th argument (index 4) and is typed any.
func New(eventType, streamID, streamType string, version int, payload any, opts ...any) (any, error) {
	return nil, nil
}
`

func TestC043_DetectsNamedByteSlicePayload(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-cqrs-lite/event/event.go": stubEventPkg,
		"bridge.go": `package main

import "test.example/stub/go-cqrs-lite/event"

type RawPayload []byte

type Message struct{ Payload RawPayload }

func bridge(msg Message) {
	_, _ = event.New("user.created", "s", "User", 1, msg.Payload)
}
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, correctness.NewC043Detector(ctx))
	ruletest.AssertRule(t, findings, "C043", 1)

	if !strings.Contains(findings[0].Message, "Payload") {
		t.Fatalf("finding should name the type: %s", findings[0].Message)
	}
}

func TestC043_NoFindingOnPlainByteSlice(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-cqrs-lite/event/event.go": stubEventPkg,
		"bridge.go": `package main

import "test.example/stub/go-cqrs-lite/event"

type Message struct{ Payload []byte }

func bridge(msg Message) {
	_, _ = event.New("user.created", "s", "User", 1, []byte(msg.Payload))
	_, _ = event.New("user.created", "s", "User", 1, []byte("{}"))
}
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, correctness.NewC043Detector(ctx))
	ruletest.AssertRule(t, findings, "C043", 0)
}

func TestC043_NoFindingOnStructPayload(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-cqrs-lite/event/event.go": stubEventPkg,
		"bridge.go": `package main

import "test.example/stub/go-cqrs-lite/event"

type UserCreated struct{ ID string }

func emit() {
	_, _ = event.New("user.created", "s", "User", 1, UserCreated{ID: "u1"})
}
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, correctness.NewC043Detector(ctx))
	ruletest.AssertRule(t, findings, "C043", 0)
}

func TestC043_NoFindingOnJsontextValue(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-cqrs-lite/event/event.go": stubEventPkg,
		"bridge.go": `package main

import (
	"encoding/json/jsontext"

	"test.example/stub/go-cqrs-lite/event"
)

func emit(raw jsontext.Value) {
	_, _ = event.New("user.created", "s", "User", 1, raw)
}
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, correctness.NewC043Detector(ctx))
	ruletest.AssertRule(t, findings, "C043", 0)
}

func TestC043_NoFindingOnNamedStringType(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-cqrs-lite/event/event.go": stubEventPkg,
		"bridge.go": `package main

import "test.example/stub/go-cqrs-lite/event"

type StreamRef string

func emit(ref StreamRef) {
	_, _ = event.New("user.created", "s", "User", 1, ref)
}
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, correctness.NewC043Detector(ctx))
	ruletest.AssertRule(t, findings, "C043", 0)
}

// Without type information the rule must stay silent: it cannot tell a
// named []byte from a plain one syntactically.
func TestC043_SilentWithoutTypeInfo(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"bridge.go": `package main

func bridge(msg Message) {
	_, _ = event.New("user.created", "s", "User", 1, msg.Payload)
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC043Detector(ctx))
	ruletest.AssertRule(t, findings, "C043", 0)
}
