package api_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/api"
)

// A004 must NOT fire on third-party router/handler frameworks whose
// Register/Handle method name collides with CQRS dispatching. Without the
// package denylist, any http.ServeMux / gorilla-mux / chi router with a type
// assertion inside a handler closure triggers the rule.
func TestA004_NoFindingForNonCQRSRouter(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

type Handler func(req any) error

func setup(mux Mux) {
	mux.Handle("/users", func(req any) error {
		r := req.(*Request)
		_ = r
		return nil
	})
}

type Mux interface{ Handle(string, Handler) }
type Request struct{ Path string }
`,
	})
	findings := runDetector(t, api.NewA004Detector(ctx))
	assertRule(t, findings, "A004", 0)
}

// A004 must NOT fire on package-qualified third-party Register APIs whose
// method name collides with CQRS. Huma v2 uses huma.Register[I,O](...) which
// is package-qualified (qualifier "huma" is denylisted). Note: a *variable*
// qualifier like `grpcServer.Register` CANNOT be distinguished from a CQRS
// dispatcher variable (`d.Register`) by name alone — that requires type info.
func TestA004_NoFindingForHumaPackageQualifiedRegister(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

func setup() {
	huma.Register(op, func(in any) error {
		m := in.(*Body)
		_ = m
		return nil
	})
}

type opT struct{}
var op = opT{}
type Body struct{ Name string }
`,
	})
	findings := runDetector(t, api.NewA004Detector(ctx))
	assertRule(t, findings, "A004", 0)
}
