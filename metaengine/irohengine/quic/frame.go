package quic

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

// frameHeaderSize is the number of bytes used for the length prefix.
// Aliased from irohengine.FrameHeaderSize so all transports share one source of truth.
const frameHeaderSize = irohengine.FrameHeaderSize

// errFrameTooLarge is returned when a received frame exceeds maxOpSize.
// Aliased from irohengine.ErrFrameTooLarge so all transports share one sentinel.
var errFrameTooLarge = irohengine.ErrFrameTooLarge

// frameHeader encodes a payload length as a 4-byte big-endian header.
func frameHeader(size int) []byte {
	h := make([]byte, frameHeaderSize)
	binary.BigEndian.PutUint32(h, uint32(size))
	return h
}

// parseFrameHeader decodes a 4-byte big-endian length prefix.
func parseFrameHeader(header []byte) (uint32, error) {
	if len(header) != frameHeaderSize {
		return 0, fmt.Errorf("frame header must be %d bytes, got %d: %w",
			frameHeaderSize, len(header), errInvalidFrameHeader)
	}
	size := binary.BigEndian.Uint32(header)
	if size > maxOpSize {
		return 0, fmt.Errorf("frame size %d exceeds max %d: %w",
			size, maxOpSize, errFrameTooLarge)
	}
	return size, nil
}

var errInvalidFrameHeader = errors.New("invalid frame header")
