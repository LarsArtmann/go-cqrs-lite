//go:build cgo

package bench

// Register the CGo-based mattn/go-sqlite3 driver alongside the pure-Go
// modernc.org/sqlite driver (imported by stack/sqlite). Both coexist: modernc
// registers as "sqlite", mattn registers as "sqlite3". The benchmark selects
// between them via sqlite.WithDriverName.
//
// This file is excluded from non-CGo builds. The mattn driver must be imported
// separately (blank import) by this module — stack/sqlite itself does not
// depend on it.
import _ "github.com/mattn/go-sqlite3"
