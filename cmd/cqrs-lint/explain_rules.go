package main

// Rule-ID mode of `cqrs-lint explain` (lives in its own file so explain.go
// stays inside its file-size baseline).

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules"
)

// errUnknownRuleIDs is the static sentinel for unknown explain IDs; the IDs
// ride the wrapped message.
var errUnknownRuleIDs = errors.New("unknown rule ID(s)")

// renderRuleExplanations prints the catalog card for each rule ID. Unknown
// IDs fail loudly instead of silently printing nothing — a fleet audit
// (2026-10-24) found `explain E004` producing no output, which forced the
// consumer to grep rule source comments for semantics.
func renderRuleExplanations(w io.Writer, ids []string) error {
	var unknown []string

	for _, id := range ids {
		rule, ok := rules.LookupRule(id)
		if !ok {
			unknown = append(unknown, id)

			continue
		}

		fmt.Fprint(w, renderRuleCard(rule))
	}

	if len(unknown) > 0 {
		return fmt.Errorf(
			"%w: %s — run 'cqrs-lint rules' for the full rule table",
			errUnknownRuleIDs,
			strings.Join(unknown, ", "),
		)
	}

	return nil
}

// renderRuleCard renders one rule's catalog metadata as a standalone card.
func renderRuleCard(r rules.RuleInfo) string {
	var b strings.Builder

	writeSectionHeader(&b, r.ID+" — "+r.Name)
	fmt.Fprintf(&b, "  Category:    %s\n", r.Category)
	fmt.Fprintf(&b, "  Severity:    %s\n", r.Severity)
	fmt.Fprintf(&b, "  Confidence:  %s\n", r.Confidence)
	fmt.Fprintf(&b, "  Auto-fix:    %t\n", r.AutoFix)
	if r.DocURL != "" {
		fmt.Fprintf(&b, "  Docs:        %s\n", r.DocURL)
	}
	b.WriteString("\n  " + r.Description + "\n")
	b.WriteString("\n  Run 'cqrs-lint rules' for the full rule table; 'cqrs-lint explain' with\n")
	b.WriteString("  no arguments documents the config-format reference.\n\n")

	return b.String()
}
