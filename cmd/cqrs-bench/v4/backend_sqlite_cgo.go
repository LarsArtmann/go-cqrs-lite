//go:build cgo

package main

// The CGo-based mattn/go-sqlite3 driver is registered alongside the pure-Go
// modernc.org/sqlite driver (imported by stack/sqlite). Both coexist: modernc
// registers as "sqlite", mattn as "sqlite3"; the bench tool selects between
// them via sqlite.WithDriverName.
//
// This file is excluded from non-CGo builds. cgoEnabled lets the backend
// registry reject CGo-only backends with a helpful message instead of a
// generic "unknown driver" error.
import _ "github.com/mattn/go-sqlite3"

// cgoEnabled reports that this build was compiled with CGo.
const cgoEnabled = true
