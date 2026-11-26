package loopback

import (
	"fmt"
	"net"
	"sync"
)

// peerConn is one TCP connection to a peer.
//
// Publish is called concurrently (the engine takes no lock around it) and
// net.Conn allows only one writer at a time: two interleaved frames leave the
// peer reading a bogus length, after which every later op on that connection
// is dropped. wmu plus a single-buffer Write keeps each frame whole.
type peerConn struct {
	conn net.Conn
	wmu  sync.Mutex
}

// writeFrame sends [4-byte big-endian length][payload] as one Write.
func (p *peerConn) writeFrame(data []byte) error {
	frame := make([]byte, frameHeaderSize+len(data))
	frame[0] = byte(len(data) >> 24)
	frame[1] = byte(len(data) >> 16)
	frame[2] = byte(len(data) >> 8)
	frame[3] = byte(len(data))
	copy(frame[frameHeaderSize:], data)

	p.wmu.Lock()
	defer p.wmu.Unlock()

	if _, err := p.conn.Write(frame); err != nil {
		return fmt.Errorf("write frame: %w", err)
	}

	return nil
}
