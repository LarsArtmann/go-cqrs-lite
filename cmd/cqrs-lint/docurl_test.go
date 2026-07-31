package main

import (
	"testing"

	"github.com/larsartmann/go-finding"
)

func TestEnrichWithDocURLs_AddsMetadata(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Rule: "C001", Severity: finding.SeverityError, Message: "test"},
	}

	result := enrichWithDocURLs(findings)

	if result[0].Metadata == nil {
		t.Fatal("expected Metadata to be non-nil for C001")
	}

	if url, ok := result[0].Metadata["cqrs-lint.doc-url"]; !ok || url == "" {
		t.Fatal("expected cqrs-lint.doc-url metadata to be set for C001")
	}
}

func TestEnrichWithDocURLs_NoURLNoChange(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Rule: "ZZZZ", Severity: finding.SeverityError, Message: "test"},
	}

	result := enrichWithDocURLs(findings)

	if result[0].Metadata != nil {
		t.Fatal("expected Metadata to remain nil for unknown rule")
	}
}
