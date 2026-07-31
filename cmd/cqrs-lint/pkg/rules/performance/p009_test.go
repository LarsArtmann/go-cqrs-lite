package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestP009_DetectsLargePayloadWithJSONCodec(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "some/event"

func create() {
	evt, _ := event.New("msg.created", "s", "S", 1, MessageCreated{})
	_ = evt
}

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
		"setup.go": `package main

import "some/codec"

var _ = event.DefaultCodec
var _ = codec.JSONCodec{}

func init() {
	event.DefaultCodec = codec.JSONCodec{}
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP009Detector(ctx))
	ruletest.AssertRule(t, findings, "P009", 1)
}

func TestP009_DetectsByteSliceWithJSONCodec(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "some/event"

func create() {
	evt, _ := event.New("file.uploaded", "s", "S", 1, FileUploaded{})
	_ = evt
}

type FileUploaded struct {
	Filename string
	Data     []byte
}
`,
		"setup.go": `package main

import "some/codec"

func init() {
	event.DefaultCodec = codec.JSONCodec{}
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP009Detector(ctx))
	ruletest.AssertRule(t, findings, "P009", 1)
}

func TestP009_NoFindingWhenUsingCBOR(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "some/event"

func create() {
	evt, _ := event.New("msg.created", "s", "S", 1, MessageCreated{})
	_ = evt
}

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
	findings := ruletest.RunDetector(t, performance.NewP009Detector(ctx))
	ruletest.AssertRule(t, findings, "P009", 0)
}

func TestP009_NoFindingForSmallPayload(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "some/event"

func create() {
	evt, _ := event.New("user.created", "s", "S", 1, UserCreated{})
	_ = evt
}

type UserCreated struct {
	ID   string
	Name string
}
`,
		"setup.go": `package main

import "some/codec"

func init() {
	event.DefaultCodec = codec.JSONCodec{}
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP009Detector(ctx))
	ruletest.AssertRule(t, findings, "P009", 0)
}

func TestP009_NoFindingForNonEventStruct(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "some/event"

func create() {
	evt, _ := event.New("user.created", "s", "S", 1, UserCreated{})
	_ = evt
}

type UserCreated struct{ Name string }

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
		"setup.go": `package main

import "some/codec"

func init() {
	event.DefaultCodec = codec.JSONCodec{}
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP009Detector(ctx))
	ruletest.AssertRule(t, findings, "P009", 0)
}
