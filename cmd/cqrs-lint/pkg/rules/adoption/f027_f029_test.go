package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestF027_OTelImportWithoutSetup(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"net/http"
	"go-cqrs-lite/otel"
)

func main() {
	_ = otel.Version()
	http.ListenAndServe(":8080", nil)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF027Detector(ctx))
	ruletest.AssertRule(t, findings, "F027", 1)
}

func TestF027_OTelWithSetup(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"net/http"
	"go-cqrs-lite/otel"
)

func main() {
	cqrsotel.Setup()
	http.ListenAndServe(":8080", nil)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF027Detector(ctx))
	ruletest.AssertRule(t, findings, "F027", 0)
}

func TestF028_SlogWithoutSetDefault(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"net/http"
	"log/slog"
)

func main() {
	slog.Info("starting")
	http.ListenAndServe(":8080", nil)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF028Detector(ctx))
	ruletest.AssertRule(t, findings, "F028", 1)
}

func TestF028_SlogWithSetDefault(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"net/http"
	"log/slog"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(nil, nil)))
	http.ListenAndServe(":8080", nil)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF028Detector(ctx))
	ruletest.AssertRule(t, findings, "F028", 0)
}

func TestF029_OTelWithoutTracingMiddleware(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"net/http"
	"go-cqrs-lite/otel"
)

func main() {
	cqrsotel.Setup()
	bus.Use(middleware.Logging())
	http.ListenAndServe(":8080", nil)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF029Detector(ctx))
	ruletest.AssertRule(t, findings, "F029", 1)
}

func TestF029_OTelWithTracingMiddleware(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"net/http"
	"go-cqrs-lite/otel"
)

func main() {
	cqrsotel.Setup()
	middleware.NewOTelBundle(tracer, meter)
	http.ListenAndServe(":8080", nil)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF029Detector(ctx))
	ruletest.AssertRule(t, findings, "F029", 0)
}
