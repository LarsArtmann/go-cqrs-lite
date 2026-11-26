//go:build !cgo

package main

// cgoEnabled reports that this build was compiled without CGo, so the
// CGo-only backends (sqlite-cgo, duckdb) are rejected up front.
const cgoEnabled = false
