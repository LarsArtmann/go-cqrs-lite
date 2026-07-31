package main

import (
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func TestApplyDomainBias_FinancialEscalatesSecurity(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Rule: "S002", Severity: finding.SeverityWarning, Message: "signing disabled"},
		{Rule: "C008", Severity: finding.SeverityWarning, Message: "money as float64"},
	}

	result := applyDomainBias(findings, analyzer.DomainFinancial)

	if result[0].Severity != finding.SeverityError {
		t.Errorf(
			"S002 should be escalated to Error for financial domain, got %s",
			result[0].Severity,
		)
	}
	if result[1].Severity != finding.SeverityError {
		t.Errorf(
			"C008 should be escalated to Error for financial domain, got %s",
			result[1].Severity,
		)
	}
}

func TestApplyDomainBias_NonFinancialNoChange(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Rule: "S002", Severity: finding.SeverityWarning, Message: "signing disabled"},
	}

	result := applyDomainBias(findings, analyzer.DomainUnknown)

	if result[0].Severity != finding.SeverityWarning {
		t.Errorf("S002 should stay Warning for unknown domain, got %s", result[0].Severity)
	}
}

func TestApplyDomainBias_AlreadyErrorNoDoubleEscalate(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Rule: "S002", Severity: finding.SeverityError, Message: "signing disabled"},
	}

	result := applyDomainBias(findings, analyzer.DomainFinancial)

	if result[0].Severity != finding.SeverityError {
		t.Errorf("S002 already Error should stay Error, got %s", result[0].Severity)
	}
	if result[0].Message != "signing disabled" {
		t.Errorf(
			"Message should not have escalation tag when already Error, got %s",
			result[0].Message,
		)
	}
}

func TestApplyDomainBias_NonSecurityRulesNotEscalated(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Rule: "C001", Severity: finding.SeverityWarning, Message: "broken command id"},
		{Rule: "B003", Severity: finding.SeverityInfo, Message: "manual event creation"},
	}

	result := applyDomainBias(findings, analyzer.DomainFinancial)

	for _, f := range result {
		if f.Severity == finding.SeverityError {
			t.Errorf("%s should not be escalated for financial domain", f.Rule)
		}
	}
}

func TestDetectDomain_FinancialEventTypes(t *testing.T) {
	t.Parallel()

	ctx := &analyzer.AnalysisContext{
		Registry: &analyzer.CQRSRegistry{
			EventTypesEmitted: map[string]analyzer.EventEmission{
				"payment.processed": {},
			},
		},
	}

	if d := analyzer.DetectFeatures(ctx).Domain; d != analyzer.DomainFinancial {
		t.Errorf("expected DomainFinancial for payment event, got %s", d)
	}
}

func TestDetectDomain_FinancialCommandTypes(t *testing.T) {
	t.Parallel()

	ctx := &analyzer.AnalysisContext{
		Registry: &analyzer.CQRSRegistry{
			CommandTypesRegistered: map[string]bool{
				"TransferFunds": true,
			},
		},
	}

	if d := analyzer.DetectFeatures(ctx).Domain; d != analyzer.DomainFinancial {
		t.Errorf("expected DomainFinancial for transfer command, got %s", d)
	}
}

func TestDetectDomain_NonFinancialUnknown(t *testing.T) {
	t.Parallel()

	ctx := &analyzer.AnalysisContext{
		Registry: &analyzer.CQRSRegistry{
			EventTypesEmitted: map[string]analyzer.EventEmission{
				"user.created": {},
			},
		},
	}

	if d := analyzer.DetectFeatures(ctx).Domain; d != analyzer.DomainUnknown {
		t.Errorf("expected DomainUnknown for user event, got %s", d)
	}
}
