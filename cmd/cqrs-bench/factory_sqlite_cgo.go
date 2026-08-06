//go:build cgo

package main

// Register the CGo-based mattn/go-sqlite3 driver alongside the pure-Go
// modernc.org/sqlite driver (imported by stack/sqlite). Both coexist: modernc
// registers as "sqlite", mattn registers as "sqlite3". The bench tool selects
// between them via sqlite.WithDriverName.
//
// This file is excluded from non-CGo builds. sqliteCgoAvailable lets the
// factory give a helpful error instead of a generic "unknown driver" message.
import _ "github.com/mattn/go-sqlite3"

const sqliteCgoAvailable = true
