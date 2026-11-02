package architecture_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// e024StubBrandedID mirrors go-branded-id's instantiable surface closely
// enough for the typed loader: named type ID in a package whose path carries
// the "go-branded-id" fragment, with the same method set shape.
const e024StubBrandedID = `package cbid

type ID[B any, V comparable] struct{ value V }

func NewID[B any, V comparable](v V) ID[B, V] { return ID[B, V]{value: v} }

func (id ID[B, V]) Get() V { return id.value }

func (id ID[B, V]) String() string { return "stub" }
`

// e024StubID mirrors the id module: a BrandNamer marker + alias for the
// string-backed StreamID family, and the struct-based ActorID whose String()
// drops the kind.
const e024StubID = `package id

import cbid "test.example/stub/go-branded-id"

type StreamMarker struct{}

func (StreamMarker) Name() string { return "StreamMarker" }

type StreamID = cbid.ID[StreamMarker, string]

type ActorKind int

func (k ActorKind) String() string { return "user" }

type ActorID struct {
	kind ActorKind
	raw  string
}

func (a ActorID) String() string { return a.raw }

func (a ActorID) PrefixedString() string { return a.kind.String() + ":" + a.raw }

func ParseStreamID(s string) (StreamID, error) { return cbid.NewID[StreamMarker](s), nil }
`

// e024StubAttribute mirrors an attribute-package constructor (otel attribute
// shape) for the log-attribute whitelist.
const e024StubAttribute = `package attribute

func String(key, value string) struct{} { return struct{}{} }
`

// --- E024: branded-ID .String() outside display contexts ---

func TestE024_FiresOnStorageKeyAndSprintfAndActorID(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-branded-id/id.go": e024StubBrandedID,
		"stub/go-cqrs-lite/id/id.go": e024StubID,
		"keys.go": `package main

import (
	"fmt"

	domainid "test.example/stub/go-cqrs-lite/id"
)

func storageKey(s domainid.StreamID) string { return s.String() }

func actorKey(a domainid.ActorID) string { return a.String() }

func sprintfKey(s domainid.StreamID) string { return fmt.Sprintf("stream:%s", s.String()) }
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, architecture.NewE024Detector(ctx))
	ruletest.AssertRule(t, findings, "E024", 3)
}

func TestE024_SilentInDisplayPositions(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-branded-id/id.go":  e024StubBrandedID,
		"stub/go-cqrs-lite/id/id.go": e024StubID,
		"stub/attribute/attribute.go": e024StubAttribute,
		"display.go": `package main

import (
	"errors"
	"fmt"
	"log/slog"

	"test.example/stub/attribute"
	domainid "test.example/stub/go-cqrs-lite/id"
)

func logIt(l *slog.Logger, s domainid.StreamID, a domainid.ActorID) {
	l.Info("loaded", "stream", s.String(), "actor", a.String())
	l.LogAttrs(nil, 0, "msg", slog.String("stream", s.String()))
	_ = attribute.String("stream", s.String())
}

func errIt(s domainid.StreamID) error {
	return fmt.Errorf("stream %s gone", s.String())
}

func reparse(s domainid.StreamID) (domainid.StreamID, error) {
	return domainid.ParseStreamID(s.String())
}

var _ = errors.New
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, architecture.NewE024Detector(ctx))
	ruletest.AssertRule(t, findings, "E024", 0)
}

func TestE024_SilentOnNonBrandedStringers(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-branded-id/id.go": e024StubBrandedID,
		"plain.go": `package main

import (
	"strings"

	cbid "test.example/stub/go-branded-id"
)

type Plain string

func (p Plain) String() string { return strings.ToUpper(string(p)) }

func usePlain(p Plain, other string) string {
	return p.String() + strings.ToLower(other)
}
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, architecture.NewE024Detector(ctx))
	ruletest.AssertRule(t, findings, "E024", 0)
}

// A cbid.ID instance whose marker has no Name() renders the bare value in
// String() — flagging it would be a false positive, so the rule stays silent.
func TestE024_SilentOnUnbrandedMarkerInstances(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-branded-id/id.go": e024StubBrandedID,
		"benign.go": `package main

import cbid "test.example/stub/go-branded-id"

type NoNameMarker struct{}

func use() string {
	var v cbid.ID[NoNameMarker, string]
	return v.String()
}
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, architecture.NewE024Detector(ctx))
	ruletest.AssertRule(t, findings, "E024", 0)
}

// Pointer receivers resolve through the same classifier.
func TestE024_FiresThroughPointerReceiver(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-branded-id/id.go":  e024StubBrandedID,
		"stub/go-cqrs-lite/id/id.go": e024StubID,
		"ptr.go": `package main

import domainid "test.example/stub/go-cqrs-lite/id"

func ptrKey(s *domainid.StreamID) string { return s.String() }
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, architecture.NewE024Detector(ctx))
	ruletest.AssertRule(t, findings, "E024", 1)
}

// Test files never carry the finding — display use dominates there and the
// storage-bug class is a production-code problem.
func TestE024_SilentOnTestFiles(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-branded-id/id.go":  e024StubBrandedID,
		"stub/go-cqrs-lite/id/id.go": e024StubID,
		"keys_test.go": `package main

import domainid "test.example/stub/go-cqrs-lite/id"

func storageKey(s domainid.StreamID) string { return s.String() }
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, architecture.NewE024Detector(ctx))
	ruletest.AssertRule(t, findings, "E024", 0)
}

// Without type information the rule must stay silent: a branded receiver is
// indistinguishable from any other Stringer syntactically.
func TestE024_SilentWithoutTypeInfo(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"keys.go": `package main

func storageKey(s StreamID) string { return s.String() }
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE024Detector(ctx))
	ruletest.AssertRule(t, findings, "E024", 0)
}
