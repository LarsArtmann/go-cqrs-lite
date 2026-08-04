package loopback

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

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
		return nil, fmt.Errorf("frame size %d exceeds max %d: %w", size, maxOpSize, errFrameTooLarge)
	}

	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read frame payload: %w", err)
	}

	return data, nil
}

// errFrameTooLarge is returned when a received frame exceeds maxOpSize.
var errFrameTooLarge = fmt.Errorf("frame too large")

func sortDurations(d []time.Duration) []time.Duration {
	cp := append([]time.Duration(nil), d...)
	for i := 1; i < len(cp); i++ {
		for j := i; j > 0 && cp[j-1] > cp[j]; j-- {
			cp[j-1], cp[j] = cp[j], cp[j-1]
		}
	}

	return cp
}

func percentileIdx(n int, p float64) int {
	idx := int(float64(n-1) * p)
	if idx >= n {
		idx = n - 1
	}

	if idx < 0 {
		idx = 0
	}

	return idx
}
