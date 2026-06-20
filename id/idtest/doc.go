// Package idtest provides panic-on-error test helpers for the id package's
// branded identifiers. Each MustParse* helper wraps the corresponding
// id.Parse* function, panicking when the input is invalid.
//
// These helpers exist for test code and examples where the input is known to be
// valid and a returned error would only add noise. Never use them in production
// paths — prefer the explicit (value, error) return of [id.Parse*].
package idtest
