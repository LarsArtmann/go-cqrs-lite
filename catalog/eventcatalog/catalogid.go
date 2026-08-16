package eventcatalog

import (
	"crypto/sha1" //nolint:gosec // UUIDv5 mandates SHA-1; not a security use
	"encoding/hex"
)

// RFC 4122 bit fields for UUID formatting.
const (
	uuidByteLen     = 16
	uuidVersionMask = 0x0f
	uuidVersion5    = 0x50
	uuidVariantMask = 0x3f
	uuidVariantRFC  = 0x80
)

// catalogIDNamespace is the fixed UUID namespace
// ("9c1f9a52-6b3e-4d78-a5f0-3e2b8c7d4a19") used to derive the EventCatalog
// `cId` from the catalog title. The value is arbitrary but MUST stay constant
// so repeated exports of the same catalog produce the same cId.
var catalogIDNamespace = [uuidByteLen]byte{
	0x9c, 0x1f, 0x9a, 0x52, 0x6b, 0x3e, 0x4d, 0x78,
	0xa5, 0xf0, 0x3e, 0x2b, 0x8c, 0x7d, 0x4a, 0x19,
}

// stableCatalogID derives a deterministic RFC 4122 version 5 (SHA-1) UUID
// from the catalog title. EventCatalog requires a `cId` per project; deriving
// it from the title (instead of generating a random one) keeps exports
// idempotent, so regenerating docs never invalidates the catalog identity.
func stableCatalogID(title string) string {
	h := sha1.New()
	h.Write(catalogIDNamespace[:])
	h.Write([]byte(title))

	sum := h.Sum(nil)
	sum[6] = sum[6]&uuidVersionMask | uuidVersion5
	sum[8] = sum[8]&uuidVariantMask | uuidVariantRFC

	return formatUUID(sum[:uuidByteLen])
}

func formatUUID(b []byte) string {
	hexed := hex.EncodeToString(b)

	return hexed[0:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:32]
}
