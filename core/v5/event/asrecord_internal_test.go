package event

import (
	"testing"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

// TestRecordEncoding_Probes pins the one-home codec-stamp mapping (M05·d):
// JSON and CBOR survive as their compact stamps; anything the record layer
// does not know degrades to EncodingUnknown (opaque) — never a guess.
func TestRecordEncoding_Probes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   codec.Encoding
		want record.Encoding
	}{
		{"json", codec.EncodingJSON, record.EncodingJSON},
		{"cbor", codec.EncodingCBOR, record.EncodingCBOR},
		{"unknown codec string", codec.Encoding("msgpack"), record.EncodingUnknown},
		{"empty", codec.Encoding(""), record.EncodingUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := recordEncoding(tt.in); got != tt.want {
				t.Errorf("recordEncoding(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
