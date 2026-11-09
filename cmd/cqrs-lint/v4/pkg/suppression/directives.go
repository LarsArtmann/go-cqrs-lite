// Comment-directive text parsing: extracting suppressed rule IDs and their
// reasons from //cqrs-lint:ignore / //cqrs-lint:disable comment text.
package suppression

import "strings"

// ParseSuppressions extracts suppressed rule IDs from comment text.
// Works on both single-line and multi-line comment text. ALL directives in
// the text are honored: a line may carry several ("//cqrs-lint:ignore(C007)
// //cqrs-lint:ignore(A008)"), and every one contributes its rule IDs.
func ParseSuppressions(commentText string) map[string]string {
	result := make(map[string]string)

	lines := strings.SplitSeq(commentText, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		// Locate the line's comment: the first "//" outside a string literal
		// or block comment. Everything after it is comment text. This
		// recognizes end-of-line suppressions ("code //cqrs-lint:ignore(A008)")
		// while rejecting two classes of false match that an anywhere-in-line
		// search would catch:
		//
		//   1. Doc/example strings: fmt.Println("//cqrs-lint:ignore(RULE)")
		//      (the // is inside a string literal → skipped).
		//   2. Doc comments that merely mention the syntax:
		//      "// see the //cqrs-lint:ignore docs" (the directive is NOT at the
		//      start of the comment text → rejected by the prefix check).
		cs := commentTextStart(line)
		if cs < 0 {
			continue
		}

		text := strings.TrimSpace(line[cs+2:])
		if !strings.HasPrefix(text, "cqrs-lint:") {
			continue
		}

		parseDirectivesInComment(text, result)
	}

	return result
}

// parseDirectivesInComment scans one comment's text for every
// "cqrs-lint:ignore(...)" / "cqrs-lint:disable(...)" occurrence and records
// each directive's rule IDs into result. Directive keywords mentioned in
// prose (not followed by an optional-space parenthesized ID list) are
// skipped, and scanning continues past them.
func parseDirectivesInComment(text string, result map[string]string) {
	const (
		kwIgnore  = "cqrs-lint:ignore"
		kwDisable = "cqrs-lint:disable"
	)

	search := text
	for {
		idx := min(indexOrMax(search, kwIgnore), indexOrMax(search, kwDisable))
		if idx >= len(search) {
			return // neither keyword occurs
		}

		search = search[idx:]

		kwEnd := len(kwIgnore)
		if strings.HasPrefix(search, kwDisable) {
			kwEnd = len(kwDisable)
		}

		afterKeyword := strings.TrimLeft(search[kwEnd:], " \t")
		if !strings.HasPrefix(afterKeyword, "(") {
			// Prose mention of the keyword, not a directive — keep scanning.
			search = search[kwEnd:]

			continue
		}

		body := afterKeyword[1:]
		before, after, ok := strings.Cut(body, ")")
		if !ok {
			// Malformed: unclosed parenthesis — nothing to extract.
			return
		}

		rawIDs := before
		reason := strings.TrimSpace(after)

		// Support comma-separated rule IDs: ignore(A001,E005).
		for id := range strings.SplitSeq(rawIDs, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				result[id] = reason
			}
		}

		search = after
	}
}

// indexOrMax returns strings.Index(s, sub), or len(s) when absent, so
// min(a, b) selects the earliest real occurrence.
func indexOrMax(s, sub string) int {
	if idx := strings.Index(s, sub); idx >= 0 {
		return idx
	}

	return len(s)
}
