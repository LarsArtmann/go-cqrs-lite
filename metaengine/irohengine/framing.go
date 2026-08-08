package irohengine

import "errors"

// FrameHeaderSize is the number of bytes used for the length-prefix header in
// transport framing protocols (quic, loopback). Shared here so all transports
// agree on the wire format — protocol constants only; I/O stays per-transport.
const FrameHeaderSize = 4

// ErrFrameTooLarge is returned when a received frame exceeds the transport's
// max op size. Shared sentinel so callers can check across transports.
var ErrFrameTooLarge = errors.New("frame too large")
