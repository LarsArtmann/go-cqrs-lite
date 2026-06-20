// Package querytest provides panic-on-error test helpers for the query package.
//
// These helpers exist for test code and examples where the input is known to be
// valid and a returned error would only add noise. Never use them in production
// paths — prefer the explicit (value, error) return of [query.New].
package querytest
