package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestF030_DeprecatedHTTPTransportImportFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import cqrshttp "github.com/larsartmann/go-cqrs-lite/transport/http/v4"

func _() {
	_ = cqrshttp.NewSSEBroker(nil)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF030Detector(ctx))
	ruletest.AssertRule(t, findings, "F030", 1)
}

func TestF030_DeprecatedGRPCTransportImportFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v4"

func _() {
	_ = cqrsgrpc.NewEventClient
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF030Detector(ctx))
	ruletest.AssertRule(t, findings, "F030", 1)
}

func TestF030_BothDeprecatedImportsFireTwoFindings(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v4"
	cqrshttp "github.com/larsartmann/go-cqrs-lite/transport/http/v4"
)

func _() {
	_, _ = cqrsgrpc.NewEventClient, cqrshttp.NewSSEBroker
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF030Detector(ctx))
	ruletest.AssertRule(t, findings, "F030", 2)
}

func TestF030_NoFindingWithSanctionedPaths(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	cqrswm "github.com/larsartmann/go-cqrs-lite/watermill/v4"
	sse "github.com/larsartmann/go-sse"
)

func _() {
	_, _ = cqrswm.NewEventPublisher, sse.WriteEvent
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF030Detector(ctx))
	ruletest.AssertRule(t, findings, "F030", 0)
}
