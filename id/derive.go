package id

import (
	"crypto/sha256"
)

// hashNamespacedKeys returns the SHA-256 digest of namespace followed by each
// key separated by a NUL byte. Shared by DeriveAggregateID, DeriveCommandID,
// and any future deterministic-ID derivation.
//
// The NUL separator prevents ambiguity when a key can itself contain the
// namespace bytes (e.g. namespace="ab", keys=["cd"] vs namespace="abc",
// keys=["d"] would collide without separation).
func hashNamespacedKeys(namespace string, keys ...string) []byte {
	h := sha256.New()
	_, _ = h.Write([]byte(namespace))

	for _, k := range keys {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(k))
	}

	return h.Sum(nil)
}
