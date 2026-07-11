package signing

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

const canonicalFormatVersion = "go-cqrs-lite/signing/v1"

const lengthPrefixSize = 4

func canonicalPayload(evt event.Event) []byte {
	if evt == nil {
		return nil
	}

	id := evt.ID().String()
	typ := string(evt.Type())
	aggID := evt.AggregateID().String()
	aggType := string(evt.AggregateType())
	version := strconv.Itoa(evt.Version().Int())
	schemaVer := strconv.Itoa(evt.SchemaVersion().Int())
	occurred := evt.OccurredAt().Format(time.RFC3339Nano)
	payload := event.PayloadReadOnly(evt)

	totalLen := 8*lengthPrefixSize +
		len(canonicalFormatVersion) + len(id) + len(typ) +
		len(aggID) + len(aggType) + len(version) + len(schemaVer) + len(occurred)

	if len(payload) > 0 {
		totalLen += sha256.Size
	}

	buf := make([]byte, 0, totalLen)

	var lenBuf [lengthPrefixSize]byte

	buf = appendPrefixed(buf, lenBuf[:], canonicalFormatVersion)
	buf = appendPrefixed(buf, lenBuf[:], id)
	buf = appendPrefixed(buf, lenBuf[:], typ)
	buf = appendPrefixed(buf, lenBuf[:], aggID)
	buf = appendPrefixed(buf, lenBuf[:], aggType)
	buf = appendPrefixed(buf, lenBuf[:], version)
	buf = appendPrefixed(buf, lenBuf[:], schemaVer)
	buf = appendPrefixed(buf, lenBuf[:], occurred)

	if len(payload) > 0 {
		h := sha256.Sum256(payload)
		buf = append(buf, h[:]...)
	}

	return buf
}

func appendPrefixed(buf, lenBuf []byte, s string) []byte {
	putUint32(lenBuf, len(s))
	buf = append(buf, lenBuf...)
	buf = append(buf, s...)

	return buf
}

func putUint32(b []byte, n int) {
	binary.BigEndian.PutUint32(
		b,
		uint32(n), //nolint:gosec // bounded by signing context
	)
}
