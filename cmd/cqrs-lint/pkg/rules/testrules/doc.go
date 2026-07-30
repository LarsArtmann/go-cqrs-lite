// Package testrules implements testing-quality rules (T-series).
//
// These rules detect missing or insufficient test coverage for CQRS constructs:
// deciders without scenario tests, projections without error-path tests,
// missing eventtest fakes, missing golden tests, and production store
// imports in test files.
package testrules
