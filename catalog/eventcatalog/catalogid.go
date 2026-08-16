package eventcatalog

import (
	"crypto/sha1"
	"encoding/hex"
)

// catalogIDNamespace is the fixed UUID namespace used to derive the
// EventCatalog `cId` from the catalog title. The value is arbitrary but MUST
// stay constant so repeated exports of the same catalog produce the same cId.
const catalogIDNamespace = "9c1f9a52-6b3e-4d78-a5f0-3e2b8c7d4a19"

// stableCatalogID derives a deterministic RFC 4122 version 5 (SHA-1) UUID
// from the catalog title. EventCatalog requires a `cId` per project; deriving
// it from the title (instead of generating a random one) keeps exports
// idempotent, so regenerating docs never invalidates the catalog identity.
func stableCatalogID(title string) string {
	var ns [16]byte

	for i, b := range parseHexUUID(catalogIDNamespace) {
		ns[i] = b
	}

	h := sha1.New() //nolint:gosec // UUIDv5 mandates SHA-1; not a security use
	h.Write(ns[:])
	h.Write([]byte(title))

	sum := h.Sum(nil)
	sum[6] = sum[6]&0x0f | 0x50 // version 5
	sum[8] = sum[8]&0x3f | 0x80 // RFC 4122 variant

	return formatUUID(sum[:16])
}

func parseHexUUID(s string) []byte {
	out := make([]byte, 16)

	for i := range 16 {
		hi := hexVal(s[i*2])
		lo := hexVal(s[i*2+1])
		out[i] = hi<<4 | lo
	}

	return out
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}

func formatUUID(b []byte) string {
	hexed := hex.EncodeToString(b)

	return hexed[0:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:32]
}
