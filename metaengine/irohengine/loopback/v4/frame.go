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
