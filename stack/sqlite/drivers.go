package sqlite

// Register the pure-Go SQLite driver. The blank import ensures the driver is
// available even if no other package in the binary imports storage directly.
// modernc.org/sqlite requires no CGo, making the preset portable.
import _ "modernc.org/sqlite"
