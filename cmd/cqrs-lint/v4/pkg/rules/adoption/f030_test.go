package adoption_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
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

// TestF030_DeletedParentIdempotencyImportFires pins the ADR-0128 shim
// deletion: the parent idempotency module's source is gone (2026-08-15),
// only its tags resolve. Seven fleet modules still import it.
func TestF030_DeletedParentIdempotencyImportFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/idempotency/v4"

func _() {
	_ = idempotency.NewMemoryStore
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF030Detector(ctx))
	ruletest.AssertRule(t, findings, "F030", 1)

	if len(findings) == 0 {
		return
	}

	suggestion := findings[0].Suggestion
	if !strings.Contains(suggestion, "go-idempotency") {
		t.Errorf("suggestion should point at go-idempotency, got %q", suggestion)
	}
}

// TestF030_DeletedKvstoreImportFiresOnce pins longest-fragment matching: the
// kvstore path also contains the deleted parent fragment, but a consumer must
// get exactly one finding — for the module it actually imports.
func TestF030_DeletedKvstoreImportFiresOnce(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4"

func _() {
	_ = kvstore.New
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF030Detector(ctx))
	ruletest.AssertRule(t, findings, "F030", 1)

	if len(findings) == 0 {
		return
	}

	if got := findings[0].Message; !strings.Contains(got, "kvstore") {
		t.Errorf("message should name the kvstore module, got %q", got)
	}
}

// TestF030_DeprecatedSqlstoreIsNotReportedAsDeleted is the ordering guard:
// idempotency/sqlstore is DEPRECATED but present (V007's territory). The
// deleted parent fragment must never swallow it into a false "deleted module"
// warning.
func TestF030_DeprecatedSqlstoreIsNotReportedAsDeleted(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"

func _() {
	_ = sqlstore.NewSQLiteStore
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF030Detector(ctx))
	ruletest.AssertRule(t, findings, "F030", 0)
}

func TestF030_DeletedShimImportsFire(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/retry"
	"github.com/larsartmann/go-cqrs-lite/flightrecorder"
)

func _() {
	_, _, _ = codec.JSONCodec{}, retry.Do, flightrecorder.New
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF030Detector(ctx))
	ruletest.AssertRule(t, findings, "F030", 3)
}

// TestF030_TestFileImportFires pins the test-file posture: a _test.go import
// of a deleted module pins the same frozen tag in go.mod/go.sum and blocks
// upgrades exactly as hard as a production import, so it fires too. The
// sanctioned-path import in the sibling test file must stay silent, proving
// the widening is per-import, not per-file-blanket.
func TestF030_TestFileImportFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import cqrswm "github.com/larsartmann/go-cqrs-lite/watermill/v4"

func _() {
	_ = cqrswm.NewEventBus
}
`,
		"service_test.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
)

func TestX(t *testing.T) {
	_, _ = idempotency.NewMemoryStore, sqlstore.NewSQLiteStore
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF030Detector(ctx))
	ruletest.AssertRule(t, findings, "F030", 1)

	if len(findings) == 0 {
		return
	}

	if got := findings[0].Message; !strings.Contains(got, "idempotency/v4") {
		t.Errorf("message should name the deleted shim import, got %q", got)
	}
}
