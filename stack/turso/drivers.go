package turso

// Register the Turso/LibSQL driver. The blank import ensures the driver is
// available even if no other package in the binary imports storage/turso
// directly.
import _ "turso.tech/database/tursogo"
