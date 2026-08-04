//go:build cgo

package quic

import (
	"fmt"
	"reflect"
	"time"

	"github.com/fxamacker/cbor/v2"
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

// opEncMode encodes time.Time as a dynamic Unix timestamp (integer when no
// sub-second component, float64 otherwise) so LWW timestamp ordering survives
// the wire round-trip. The default cbor.Marshal truncates time to whole seconds,
// collapsing sub-second LWW comparisons into ties.
var opEncMode = func() cbor.EncMode {
	em, _ := cbor.EncOptions{Time: cbor.TimeUnixDynamic}.EncMode()
	return em
}()

func encodeOp(op irohengine.WriteOp) ([]byte, error) {
	data, err := opEncMode.Marshal(op)
	if err != nil {
		return nil, fmt.Errorf("encode writeop: %w", err)
	}

	return data, nil
}

func decodeOp(data []byte) (irohengine.WriteOp, error) {
	var op irohengine.WriteOp
	if err := opDecMode.Unmarshal(data, &op); err != nil {
		return irohengine.WriteOp{}, fmt.Errorf("decode writeop: %w", err)
	}

	return op, nil
}

// opDecMode decodes CBOR maps into map[string]interface{} (matching JSON
// semantics) instead of the default map[interface{}]interface{}.
var opDecMode = func() cbor.DecMode {
	dm, _ := cbor.DecOptions{
		DefaultMapType: reflect.TypeOf(map[string]interface{}{}),
	}.DecMode()
	return dm
}()

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
