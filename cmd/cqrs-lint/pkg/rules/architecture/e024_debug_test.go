package architecture

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// TestE024DebugDisplay dumps which display-position line still flags.
func TestE024DebugDisplay(t *testing.T) {
	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-branded-id/id.go": `package cbid

type ID[B any, V comparable] struct{ value V }

func NewID[B any, V comparable](v V) ID[B, V] { return ID[B, V]{value: v} }

func (id ID[B, V]) String() string { return "stub" }
`,
		"stub/go-cqrs-lite/id/id.go": `package id

import cbid "test.example/stub/go-branded-id"

type StreamMarker struct{}

func (StreamMarker) Name() string { return "StreamMarker" }

type StreamID = cbid.ID[StreamMarker, string]

func ParseStreamID(s string) (StreamID, error) { return cbid.NewID[StreamMarker](s), nil }
`,
		"stub/attribute/attribute.go": `package attribute

func String(key, value string) struct{} { return struct{}{} }
`,
		"display.go": `package main

import (
	"fmt"
	"log/slog"

	"test.example/stub/attribute"
	domainid "test.example/stub/go-cqrs-lite/id"
)

func logIt(l *slog.Logger, s domainid.StreamID) {
	l.Info("loaded", "stream", s.String())
	l.LogAttrs(nil, 0, "msg", slog.String("stream", s.String()))
	_ = attribute.String("stream", s.String())
}

func errIt(s domainid.StreamID) error {
	return fmt.Errorf("stream %s gone", s.String())
}

func reparse(s domainid.StreamID) (domainid.StreamID, error) {
	return domainid.ParseStreamID(s.String())
}
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, NewE024Detector(ctx))
	for _, f := range findings {
		t.Logf("finding at %+v: %s", f.Position, f.Message)
	}
}
