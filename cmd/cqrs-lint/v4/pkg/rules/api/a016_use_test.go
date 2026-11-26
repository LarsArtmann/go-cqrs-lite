package api_test

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/api"
)

// routerUseSource is a project with NO command dispatcher: a chi router's
// `.Use` and a logger's `.Use` were enough to make A016 report a dispatcher.
const routerUseSource = `package main

import (
	"github.com/go-chi/chi/v5"
	"golang.org/x/exp/slog"
)

func routes(logger *slog.Logger) {
	r := chi.NewRouter()
	r.Use(requestID)
	logger.Use(traceMiddleware)
	_ = r
}
`

// TestA016_RouterUseIsNotADispatcher: `.Use` on a router/logger must not
// manufacture a command dispatcher.
func TestA016_RouterUseIsNotADispatcher(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{"routes.go": routerUseSource})
	ctx.FeatureProfile = analyzer.FeatureProfile{CommandFlow: analyzer.CommandFlowCommands}

	findings, err := api.NewA016Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("A016 detect: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("router/logger .Use must not trigger A016, got %d: %v", len(findings), findings)
	}
}

// TestA016_RouterUseDoesNotHideRealDispatcher: with the router file seen
// FIRST, the real command dispatcher in another module must still be
// evaluated against its own profile and fire.
func TestA016_RouterUseDoesNotHideRealDispatcher(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/routes.go": routerUseSource,
		"/repo/examples/app/cmd.go": `package app

import "github.com/larsartmann/go-cqrs-lite/command/v4"

func wire() {
	d := command.NewDispatcher()
	d.Use(audit)
}
`,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {CommandFlow: analyzer.CommandFlowReadOnly},
		"/repo/examples/app": {CommandFlow: analyzer.CommandFlowCommands},
	}

	findings, err := api.NewA016Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("A016 detect: %v", err)
	}

	if len(findings) != 1 || !strings.HasSuffix(string(findings[0].Position.File), "examples/app/cmd.go") {
		t.Errorf("expected exactly one finding at the command dispatcher, got %d: %+v", len(findings), findings)
	}
}
