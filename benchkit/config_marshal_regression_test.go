package benchkit

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/larsartmann/go-codec"
)

// Guards the pointer-receiver MarshalJSON: marshaling a VALUE Config (the
// common call shape) must still stamp CodecName, not silently skip custom
// marshaling.
func TestConfig_MarshalJSON_ValueStillStampsCodecName(t *testing.T) {
	cfg := Config{
		PayloadSize: 256,
		Codec:       codec.JSONCodec{},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(data), `"CodecName":"json"`) {
		t.Fatalf("expected codecName stamped on value marshal, got: %s", data)
	}
}
