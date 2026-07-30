package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
)

func TestP009_DetectsLargePayloadWithJSONCodec(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "some/codec"

var defaultCodec = codec.JSONCodec{}

type MessageCreated struct {
	ID        string
	ChannelID string
	AuthorID  string
	Content   string
	Nickname  string
	GuildID   string
	Timestamp string
	Edited    string
	Mention   string
	Pinned    string
	ReplyID   string
}
`,
	})
	findings := runDetector(t, performance.NewP009Detector(ctx))
	assertRule(t, findings, "P009", 1)
}

func TestP009_DetectsByteSliceWithJSONCodec(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "some/codec"

var defaultCodec = codec.JSONCodec{}

type FileUploaded struct {
	Filename string
	Data     []byte
}
`,
	})
	findings := runDetector(t, performance.NewP009Detector(ctx))
	assertRule(t, findings, "P009", 1)
}

func TestP009_NoFindingWhenUsingCBOR(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "some/codec"

var defaultCodec = codec.CBORCodec{}

type MessageCreated struct {
	ID        string
	ChannelID string
	AuthorID  string
	Content   string
	Nickname  string
	GuildID   string
	Timestamp string
	Edited    string
	Mention   string
	Pinned    string
	ReplyID   string
}
`,
	})
	findings := runDetector(t, performance.NewP009Detector(ctx))
	assertRule(t, findings, "P009", 0)
}

func TestP009_NoFindingForSmallPayload(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "some/codec"

var defaultCodec = codec.JSONCodec{}

type UserCreated struct {
	ID   string
	Name string
}
`,
	})
	findings := runDetector(t, performance.NewP009Detector(ctx))
	assertRule(t, findings, "P009", 0)
}

func TestP009_NoFindingForNonEventStruct(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "some/codec"

var defaultCodec = codec.JSONCodec{}

type Configuration struct {
	Field1  string
	Field2  string
	Field3  string
	Field4  string
	Field5  string
	Field6  string
	Field7  string
	Field8  string
	Field9  string
	Field10 string
	Field11 string
}
`,
	})
	findings := runDetector(t, performance.NewP009Detector(ctx))
	assertRule(t, findings, "P009", 0)
}
