package signing

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// canonicalFormatVersion identifies the canonical payload format version.
// If the format changes, this constant must be incremented so that
// old and new signatures are distinguishable.
const canonicalFormatVersion = "go-cqrs-lite/signing/v1"

// canonicalPayload builds a deterministic byte representation of an event
// for signing. It excludes the signature itself and non-deterministic fields
// like metadata to prevent circular signing issues.
//
// Schema version is included because it semantically identifies the payload
// structure. Changing the schema version without changing payload content is
// a meaningful event transformation that must be reflected in the signature.
// The payload itself is SHA-256 hashed to keep the canonical representation
// bounded regardless of payload size.
//
// The output is prefixed with a format version tag so that future format
// changes produce different signatures, preventing cross-version collisions.
func canonicalPayload(evt event.Event) []byte {
	if evt == nil {
		return nil
	}

	id := evt.ID().String()
	typ := string(evt.Type())
	aggID := evt.AggregateID().String()
	aggType := string(evt.AggregateType())
	version := evt.Version().Int()
	schemaVer := evt.SchemaVersion().Int()
	occurred := evt.OccurredAt().Format(time.RFC3339Nano)
	payload := evt.Payload()

	var buf []byte

	buf = appendLenPrefixed(buf, canonicalFormatVersion)
	buf = appendLenPrefixed(buf, id)
	buf = appendLenPrefixed(buf, typ)
	buf = appendLenPrefixed(buf, aggID)
	buf = appendLenPrefixed(buf, aggType)
	buf = appendLenPrefixed(buf, strconv.Itoa(version))
	buf = appendLenPrefixed(buf, strconv.Itoa(schemaVer))
	buf = appendLenPrefixed(buf, occurred)

	if len(payload) > 0 {
		h := sha256.Sum256(payload)
		buf = append(buf, h[:]...)
	}

	return buf
}

const lengthPrefixSize = 4

func appendLenPrefixed(buf []byte, s string) []byte {
	b := []byte(s)
	lenBuf := make([]byte, lengthPrefixSize)
	binary.BigEndian.PutUint32(
		lenBuf,
		uint32(len(b)), //nolint:gosec // length fits in uint32 for any reasonable string
	)

	buf = append(buf, lenBuf...)
	buf = append(buf, b...)

	return buf
}
