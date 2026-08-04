package architecture_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestE016_DetectsMissingHealthCheck(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import "net/http"

func runServer() {
	srv := &http.Server{Addr: ":8080"}
	_ = srv.ListenAndServe()
}
`,
	})
	ctx.FeatureProfile.ServerLocal = false // simulate production server

	findings := ruletest.RunDetector(t, architecture.NewE016Detector(ctx))
	ruletest.AssertRule(t, findings, "E016", 1)
}

func TestE016_NoFindingWhenHealthCheckPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import (
	"context"
	"net/http"
)

func runServer(bundle Bundle) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_ = bundle.HealthCheck(context.Background())
	})
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE016Detector(ctx))
	ruletest.AssertRule(t, findings, "E016", 0)
}

func TestE016_NoFindingForNonServerProject(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	println("hello")
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE016Detector(ctx))
	ruletest.AssertRule(t, findings, "E016", 0)
}

func TestE016_NoFindingForHealthEndpointRoute(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import "net/http"

func runServer() {
	http.HandleFunc("/healthz", healthHandler)
	srv := &http.Server{Addr: ":8080"}
	_ = srv.ListenAndServe()
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE016Detector(ctx))
	ruletest.AssertRule(t, findings, "E016", 0)
}

func TestE016_NoFindingForLivezEndpoint(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import "net/http"

func runServer() {
	http.HandleFunc("/livez", healthHandler)
	srv := &http.Server{Addr: ":8080"}
	_ = srv.ListenAndServe()
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE016Detector(ctx))
	ruletest.AssertRule(t, findings, "E016", 0)
}

func TestE016_NarrowedScanStringLiteralNotInRouteCall(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import "net/http"

const description = "/healthz is the health endpoint"

func runServer() {
	srv := &http.Server{Addr: ":8080"}
	_ = srv.ListenAndServe()
}
`,
	})
	ctx.FeatureProfile.ServerLocal = false

	findings := ruletest.RunDetector(t, architecture.NewE016Detector(ctx))
	ruletest.AssertRule(t, findings, "E016", 1)
}

func TestE016_NoFindingWithCqrsHtmx(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import (
	"net/http"

	"github.com/larsartmann/cqrs-htmx"
)

func runServer() {
	app := cqrshtmx.New()
	srv := &http.Server{Addr: ":8080"}
	_ = srv.ListenAndServe()
	_ = app
}
`,
	})
	ctx.FeatureProfile.ServerLocal = false

	findings := ruletest.RunDetector(t, architecture.NewE016Detector(ctx))
	ruletest.AssertRule(t, findings, "E016", 0)
}

func TestE017_DetectsMissingGracefulShutdown(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	<-ch
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE017Detector(ctx))
	ruletest.AssertRule(t, findings, "E017", 1)
}

func TestE017_NoFindingWhenGracefulShutdownPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main(bundle Bundle) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	<-ch
	bundle.GracefulClose(context.Background())
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE017Detector(ctx))
	ruletest.AssertRule(t, findings, "E017", 0)
}

func TestE017_NoFindingForNoSignalNotify(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	println("hello")
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE017Detector(ctx))
	ruletest.AssertRule(t, findings, "E017", 0)
}
