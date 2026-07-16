// Package fix provides the CQRS-specific fix provider for go-finding/pipeline.
package fix

import (
	"bytes"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"
)

// CQRSFixProvider implements pipeline.FixProvider for CQRS-specific fixes.
// It uses the finding's BeforeCode/AfterCode fields and Metadata for
// BeforeCode/AfterCode-based substring matching.
type CQRSFixProvider struct{}

// NewCQRSFixProvider creates a new CQRS fix provider.
func NewCQRSFixProvider() *CQRSFixProvider {
	return &CQRSFixProvider{}
}

// Name returns the provider name.
func (p *CQRSFixProvider) Name() string { return "cqrs-fix" }

// CanHandle returns true if the finding has BeforeCode/AfterCode data.
func (p *CQRSFixProvider) CanHandle(f finding.Finding) bool {
	return f.ToolName == "cqrs-lint" && f.HasCodeChange() && f.BeforeCode != ""
}

// Edits converts a finding into byte-level edits using BeforeCode/AfterCode matching.
func (p *CQRSFixProvider) Edits(content []byte, f finding.Finding) ([]pipeline.FixEdit, error) {
	beforeCode := f.BeforeCode
	afterCode := f.AfterCode

	// Also check Metadata for old/new expressions (used by C006).
	if old, ok := f.Metadata["oldExpr"]; ok {
		beforeCode = old
	}

	if new, ok := f.Metadata["newExpr"]; ok {
		afterCode = new
	}

	if beforeCode == "" {
		return nil, nil
	}

	idx := bytes.Index(content, []byte(beforeCode))
	if idx == -1 {
		return nil, nil
	}

	replacement := []byte(afterCode)
	if afterCode == "" {
		replacement = nil
	}

	return []pipeline.FixEdit{{
		Offset:      idx,
		Length:      len(beforeCode),
		Replacement: replacement,
		Source:      f,
	}}, nil
}
