package selftest

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// Planted mutation fixture for check-error-taxonomy.sh --self-test.
// NOT part of any build — excluded from go.mod; the scanner greps it as text.
// Each case pins one extraction behavior the gate depends on.

var (
	// Case 1: single-line New — must be captured with its family.
	caseSimple = errorfamily.NewRejection("selftest.simple", "simple case")

	// Case 2: multi-line New with the code on the next line — must be
	// captured (the \(\s* form).
	caseMultiline = errorfamily.NewInfrastructure(
		"selftest.multiline",
		"multiline case",
	)

	// Case 3: Wrap with a leading error argument — must be captured.
	caseWrap = errorfamily.WrapCorruption(caseSimple, "selftest.wrap", "wrap case")

	// Case 4: a quoted string that is NOT a code (previous scanner bug:
	// the greedy cross-quote capture grabbed "limit" and junk like
	// ".reconstruct_event" from sites like this one) — must NOT be
	// captured.
	caseTricky = errorfamily.WrapCorruption(
		caseSimple,
		limitLabel("selftest.prefix"), // "limit" must not leak as a code
		"tricky case",
	)
)

func limitLabel(p string) string { return p + ".suffix" }

// Case 5: a quoted example inside a doc comment — the scanner is
// line-based on constructor shapes, so a commented-out constructor here
// would be captured; keep codes in comments NON-matching instead:
//
//	errorfamily.NewRejection("<never.extract>", "placeholder example")
const exampleCode = "selftest.example_const"
