// Package idtest provides test helpers for the id package's branded identifiers.
// Each Parse* helper wraps the corresponding id.Parse* function, calling
// tb.Fatalf when the input is invalid — no panics.
//
// These helpers exist for test code and examples where the input is known to be
// valid and a returned error would only add noise. Never use them in production
// paths — prefer the explicit (value, error) return of [id.Parse*].
package idtest
