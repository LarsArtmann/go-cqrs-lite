package codec_test

import (
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

type goldenPayload struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
}

// matchGolden wraps go-snaps MatchSnapshot with the module's golden directory.
func matchGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	snaps.WithConfig(
		snaps.Dir(filepath.Join("testdata", "golden")),
		snaps.Filename(name),
	).MatchSnapshot(t, string(got))
}

func TestGolden_JSONCodec_Encode(t *testing.T) {
	c := codec.JSONCodec{}

	payload := goldenPayload{Email: "alice@example.com", Name: "Alice", Age: 30}

	got, err := c.Encode(payload)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	matchGolden(t, "json_encode", got)
}

func TestGolden_CBORCodec_Encode(t *testing.T) {
	c := codec.CBORCodec{}

	payload := goldenPayload{Email: "alice@example.com", Name: "Alice", Age: 30}

	got, err := c.Encode(payload)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// CBOR is binary — hex representation for readable diffable snapshots.
	matchGolden(t, "cbor_encode", []byte(hex.EncodeToString(got)))
}

func TestGolden_RawCodec_Passthrough(t *testing.T) {
	c := codec.RawCodec{}

	input := []byte(`{"raw":true}`)

	got, err := c.Encode(input)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	matchGolden(t, "raw_passthrough", got)
}
