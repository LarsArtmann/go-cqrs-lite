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

	// Case 6: Newf (format-verb constructor) with a qualified family — must
	// be captured. The pre-2026-10-16 pattern was blind to Newf/Wrapf, so a
	// doc regenerated from erraudit's pool (which sees them) would
	// false-positive the rg-fallback gate on every one of these rows.
	caseNewf = errorfamily.Newf(errorfamily.Rejection, "selftest.newf", "newf case %s", "arg")

	// Case 7: Wrapf with a leading error argument and a qualified family,
	// spanning lines exactly like event/store_middleware_streaming.go —
	// must be captured.
	caseWrapf = errorfamily.Wrapf(caseSimple, errorfamily.Transient,
		"selftest.wrapf",
		"wrapf case %s", "arg")

	// Case 8: dynamically concatenated code — the code group requires a
	// full literal dotted string, so the bare "selftest." prefix must NOT
	// be captured as junk (erraudit's pool skips non-literal codes the same
	// way; neither pool ever claims this site).
	caseDynamic = errorfamily.Newf(errorfamily.Rejection, "selftest."+part, "dynamic case")

	// Case 9: Wrapf whose first argument is a nested CALL with comma
	// arguments — the exact storage/pebble corrupt-event shape
	// (a.corruptEventErr(string(iter.Key()), err)). The f-form's first-arg
	// group may cross commas and parens but never a quote — must be
	// captured, and the quoted string inside causeLabel must not leak.
	caseCallArg = errorfamily.Wrapf(
		causeLabel("selftest.tricky", part),
		errorfamily.Corruption, "selftest.callarg",
		"call-arg case",
	)
)

func limitLabel(p string) string { return p + ".suffix" }

func causeLabel(p, q string) string { return p + q }

var part = "dynamic"

// Case 5: a quoted example inside a doc comment — the scanner is
// line-based on constructor shapes, so a commented-out constructor here
// would be captured; keep codes in comments NON-matching instead:
//
//	errorfamily.NewRejection("<never.extract>", "placeholder example")
const exampleCode = "selftest.example_const"
