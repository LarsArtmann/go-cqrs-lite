package main

import "errors"

// Command-line faults. Nothing has been measured or written when one of
// these is returned.
var (
	errBackend            = errors.New("unknown backend")
	errBackendUnavailable = errors.New("backend unavailable in this build")
	errDSNRequired        = errors.New("--dsn required")
	errProfile            = errors.New("unknown profile")
	errCodec              = errors.New("unknown codec")
	errFormat             = errors.New("invalid --format")
	errFlagValue          = errors.New("invalid flag value")
	errFlagCombo          = errors.New("conflicting flags")
	errSweepParam         = errors.New("unknown --param")
)

// Runtime faults: the command was valid, but the benchmark, a gate, or the
// report could not be completed.
var (
	errBenchmark   = errors.New("benchmark failed")
	errGate        = errors.New("--strict")
	errBudget      = errors.New("time budget exhausted")
	errInterrupted = errors.New("interrupted")
	errWriteReport = errors.New("write report")
	errRender      = errors.New("render report")
)
