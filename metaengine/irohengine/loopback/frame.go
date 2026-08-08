package loopback

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

// frameHeaderSize is aliased from irohengine.FrameHeaderSize so all transports
// share one source of truth for the wire-format constant.
const frameHeaderSize = irohengine.FrameHeaderSize

// errFrameTooLarge is aliased from irohengine.ErrFrameTooLarge.
var errFrameTooLarge = irohengine.ErrFrameTooLarge

// writeFrame writes a length-prefixed message to w.
// Format: [4-byte big-endian length][payload bytes].
func writeFrame(w io.Writer, data []byte) error {
	header := make([]byte, frameHeaderSize)
	binary.BigEndian.PutUint32(header, uint32(len(data)))
	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("write frame header: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write frame payload: %w", err)
	}

	return nil
}

// readFrame reads a length-prefixed message from r.
func readFrame(r io.Reader) ([]byte, error) {
	header := make([]byte, frameHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("read frame header: %w", err)
	}

	size := binary.BigEndian.Uint32(header)
	if size > maxOpSize {
		return nil, fmt.Errorf(
			"frame size %d exceeds max %d: %w",
			size,
			maxOpSize,
			errFrameTooLarge,
		)
	}

	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read frame payload: %w", err)
	}

	return data, nil
}
