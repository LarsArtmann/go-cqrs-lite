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

	if after, ok := f.Metadata["newExpr"]; ok {
		afterCode = after
	}

	if beforeCode == "" {
		return nil, nil
	}

	// Position-based matching: convert the finding's line:column to a byte
	// offset and check if BeforeCode appears there. This handles files with
	// multiple occurrences of the same pattern (e.g., multiple event.NewEvent
	// calls) by fixing the exact one the finding refers to.
	idx := positionBasedIndex(content, f, beforeCode)
	if idx == -1 {
		// Fall back to first-occurrence substring search.
		idx = bytes.Index(content, []byte(beforeCode))
	}

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

// positionBasedIndex converts the finding's line:column to a byte offset and
// checks if beforeCode appears at that offset. Returns -1 if the position is
// invalid or beforeCode does not match at the expected location.
func positionBasedIndex(content []byte, f finding.Finding, beforeCode string) int {
	if f.Position.Line <= 0 || f.Position.Column <= 0 {
		return -1
	}

	line, col := 1, 1

	for i := range content {
		if line == f.Position.Line && col == f.Position.Column {
			if i+len(beforeCode) <= len(content) &&
				string(content[i:i+len(beforeCode)]) == beforeCode {
				return i
			}

			return -1
		}

		if content[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}

	return -1
}
