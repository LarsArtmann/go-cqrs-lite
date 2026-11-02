package architecture

import (
	"go/types"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

func TestE024DebugCBID(t *testing.T) {
	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"stub/go-branded-id/id.go": `package cbid

type ID[B any, V comparable] struct{ value V }

func NewID[B any, V comparable](v V) ID[B, V] { return ID[B, V]{value: v} }

func (id ID[B, V]) Get() V { return id.value }

func (id ID[B, V]) String() string { return "stub" }
`,
		"stub/go-cqrs-lite/id/id.go": `package id

import cbid "test.example/stub/go-branded-id"

type StreamMarker struct{}

func (StreamMarker) Name() string { return "StreamMarker" }

type StreamID = cbid.ID[StreamMarker, string]
`,
		"keys.go": `package main

import domainid "test.example/stub/go-cqrs-lite/id"

func storageKey(s domainid.StreamID) string { return s.String() }
`,
	})
	defer cleanup()

	for _, gf := range ctx.GoFiles {
		if gf.Pkg == nil || gf.Pkg.TypesInfo == nil {
			t.Logf("%s: no type info", gf.Path)
			continue
		}
		for expr, tv := range gf.Pkg.TypesInfo.Types {
			t.Logf("%s: expr %T %v -> %v", gf.Path, expr, expr, tv.Type)
		}
		for ident, obj := range gf.Pkg.TypesInfo.Uses {
			if tn, ok := obj.(*types.TypeName); ok {
				t.Logf("%s: use %s -> typeName %v (named=%v)", gf.Path, ident.Name, tn.Type(), func() bool {
					_, isNamed := tn.Type().(*types.Named)
					return isNamed
				}())
			}
		}
	}
}
