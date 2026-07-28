// Package lintutil holds helpers shared across the cqrs-lint rule
// subpackages (correctness, consistency, boilerplate, etc.).
package lintutil

import (
	"github.com/larsartmann/go-finding"
)

// AppendBuild appends a built [finding.Finding] to findings unless err is
// non-nil. Building a finding only fails when the builder was mis-configured
// at construction time (a programming bug in the rule itself), so the error
// is silently dropped — the rule's own unit tests catch builder bugs, and
// surfacing them in the lint output would pollute results for users.
//
// Collapses the repeated
//
//	f, err := finding.NewBuilder(...).Build()
//	if err != nil {
//	    return
//	}
//	*findings = append(*findings, f)
//
// boilerplate found in every rule's report* helper.
func AppendBuild(findings *[]finding.Finding, f finding.Finding, err error) {
	if err != nil {
		return
	}

	*findings = append(*findings, f)
}
