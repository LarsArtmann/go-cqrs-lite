package turso

// Register the Turso database driver. The blank import ensures the driver is
// available even if no other package in the binary imports storage/turso
// directly.
import _ "turso.tech/database/tursogo"
