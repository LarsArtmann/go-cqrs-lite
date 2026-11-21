package quic

import "time"

// Protocol limits and budgets for the QUIC transport. Collected here so the
// transport file stays focused on the connection lifecycle.

// maxOpSize is the maximum serialized WriteOp size accepted over a QUIC stream.
// 16 MB is generous for CRDT ops while preventing memory exhaustion.
const maxOpSize = 16 * 1024 * 1024

// ackReadLimit bounds the bytes read from a write-ack stream (the ack is
// empty; the budget only drains the stream cleanly).
const ackReadLimit = 1024

// closeTimeout bounds the shutdown wait for in-flight stream sends.
const closeTimeout = 5 * time.Second

// rttWindowSize is how many RTT samples we keep for percentile computation.
const rttWindowSize = 256
