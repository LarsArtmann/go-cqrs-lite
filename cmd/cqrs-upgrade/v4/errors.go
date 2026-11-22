package main

import "errors"

var (
	// errInvalidToFlag fires when --to is not valid semver.
	errInvalidToFlag = errors.New("--to is not a valid semver version")
	// errStrictViolations fires when --strict finds v5-removed API usage.
	errStrictViolations = errors.New("--strict: v5-removed API usage detected")
	// errArgsAfterDir fires when arguments follow the positional module
	// directory. Stdlib flag parsing stops at the first positional, so a
	// flag written after the dir is silently ignored (2026-09-11:
	// `cqrs-upgrade . --strict` ran as a plain report instead of the gate).
	errArgsAfterDir = errors.New(
		"unexpected arguments after the module directory (flags must precede it)")
	// errStrictScanFailed fires when --strict cannot prove cleanliness: the
	// deprecation scan itself failed, so v5-readiness is unproven. A gate
	// that silently passes when its scanner breaks is not a gate.
	errStrictScanFailed = errors.New("--strict: deprecation scan failed")
	// errStrictModuleFailed fires when any module errored in the pipeline
	// (pin collection, go.mod edit, or build verification). An errored module
	// never reached the deprecation scan — unscanned is unproven, so --strict
	// must fail instead of passing on the silence.
	errStrictModuleFailed = errors.New("--strict: module error")
	// errStrictV5Remaining fires when --to-v5 --strict still sees cqrs v4
	// imports with no folded core/v5 twin: the module's migration to the v5
	// core is incomplete, so v5-readiness is unproven.
	errStrictV5Remaining = errors.New(
		"--strict --to-v5: not-yet-folded go-cqrs-lite v4 imports remain")
	// errStrictV5ModuleFailed fires when the --to-v5 pass itself errored on a
	// module (unreadable file, unparseable source). Unscanned is unproven.
	errStrictV5ModuleFailed = errors.New("--strict --to-v5: module error")
)
