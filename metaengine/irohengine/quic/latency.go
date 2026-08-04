//go:build cgo

package quic

import (
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

// --- LatencyProvider implementation ---

// LatencySnapshot implements irohengine.LatencyProvider, returning real
// latency measured from QUIC ACK timing (conn.Rtt()).
func (t *QuicTransport) LatencySnapshot() irohengine.LatencySnapshot {
	t.rttMu.Lock()
	samples := append([]time.Duration(nil), t.rttSamples...)
	t.rttMu.Unlock()

	if len(samples) == 0 {
		return irohengine.LatencySnapshot{}
	}

	sorted := sortDurations(samples)
	rtt := sorted[len(sorted)/2] // P50
	return irohengine.LatencySnapshot{
		DeliveryP50:    rtt / 2, // one-way approx RTT/2
		DeliveryP99:    sorted[percentileIdx(len(sorted), 0.99)] / 2,
		ConvergenceP99: sorted[percentileIdx(len(sorted), 0.99)],
	}
}

func (t *QuicTransport) recordRTT(d time.Duration) {
	t.rttMu.Lock()
	defer t.rttMu.Unlock()
	t.rttSamples = append(t.rttSamples, d)
	if len(t.rttSamples) > rttWindowSize {
		t.rttSamples = t.rttSamples[len(t.rttSamples)-rttWindowSize:]
	}
}

// --- Codec ---

func encodeOp(op irohengine.WriteOp) ([]byte, error) {
	data, err := json.Marshal(op)
	if err != nil {
		return nil, fmt.Errorf("encode writeop: %w", err)
	}

	return data, nil
}

func decodeOp(data []byte) (irohengine.WriteOp, error) {
	var op irohengine.WriteOp
	if err := json.Unmarshal(data, &op); err != nil {
		return irohengine.WriteOp{}, fmt.Errorf("decode writeop: %w", err)
	}

	return op, nil
}

// --- Helpers ---

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
