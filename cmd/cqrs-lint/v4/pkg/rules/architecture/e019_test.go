package architecture_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

func TestE019_FlagsContractlessOutput(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"catalog.go": `package main

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

var reg *catalog.Registry

var _ = func() {
	reg.AddDataProduct(catalog.DataProduct{
		ID:      "ledger",
		Name:    "Ledger",
		Version: "1.0.0",
		Outputs: []catalog.DataProductOutput{
			{Ref: catalog.Ref{ID: "invoice.issued", Version: "1.0.0"}},
			{
				Ref:      catalog.Ref{ID: "order.placed", Version: "1.0.0"},
				Contract: &catalog.DataContract{Path: "contracts/orders.yaml"},
			},
		},
	})
}
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE019Detector(ctx))
	ruletest.AssertRule(t, findings, "E019", 1)
}

// TestE019_VariablePassedProductIsVisible pins the same-file
// single-indirection resolution (13-32 §f21): a data product assigned to a
// variable and passed by identifier must still be scanned, not silently
// skipped.
func TestE019_VariablePassedProductIsVisible(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"catalog.go": `package main

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

var reg *catalog.Registry

var _ = func() {
	dp := catalog.DataProduct{
		ID:      "varledger",
		Name:    "Var Ledger",
		Version: "1.0.0",
		Outputs: []catalog.DataProductOutput{
			{Ref: catalog.Ref{ID: "invoice.issued", Version: "1.0.0"}},
		},
	}
	reg.AddDataProduct(dp)
}
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE019Detector(ctx))
	ruletest.AssertRule(t, findings, "E019", 1)
}

// TestE019_VariablePassedContractedProductSilent pins the resolution does
// not over-fire: a variable-passed product whose outputs ALL carry contracts
// stays silent.
func TestE019_VariablePassedContractedProductSilent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"catalog.go": `package main

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

var reg *catalog.Registry

var _ = func() {
	dp := catalog.DataProduct{
		ID:      "varledger",
		Name:    "Var Ledger",
		Version: "1.0.0",
		Outputs: []catalog.DataProductOutput{
			{
				Ref:      catalog.Ref{ID: "order.placed", Version: "1.0.0"},
				Contract: &catalog.DataContract{Path: "contracts/orders.yaml"},
			},
		},
	}
	reg.AddDataProduct(dp)
}
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE019Detector(ctx))
	ruletest.AssertRule(t, findings, "E019", 0)
}

func TestE019_FlagsOutputlessProduct(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"catalog.go": `package main

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

var reg *catalog.Registry

var _ = func() {
	reg.AddDataProduct(catalog.DataProduct{
		ID:      "mirror",
		Name:    "Mirror",
		Version: "1.0.0",
		Inputs:  []catalog.Ref{{ID: "invoice.issued", Version: "1.0.0"}},
	})
}
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE019Detector(ctx))
	ruletest.AssertRule(t, findings, "E019", 1)
}

func TestE019_SilentWhenFullyContracted(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"catalog.go": `package main

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

var reg *catalog.Registry

var _ = func() {
	reg.AddDataProduct(catalog.DataProduct{
		ID:      "ledger",
		Name:    "Ledger",
		Version: "1.0.0",
		Outputs: []catalog.DataProductOutput{
			{
				Ref:      catalog.Ref{ID: "order.placed", Version: "1.0.0"},
				Contract: &catalog.DataContract{Path: "contracts/orders.yaml"},
			},
		},
	})
}
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE019Detector(ctx))
	ruletest.AssertRule(t, findings, "E019", 0)
}
