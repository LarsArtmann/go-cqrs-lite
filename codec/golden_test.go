package codec_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func goldenDir() string {
	return "testdata/golden"
}

type goldenPayload struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
}

func assertCodecGolden(
	t *testing.T,
	goldenPath string,
	got []byte,
	compare func(got, want []byte),
) {
	t.Helper()

	if *updateGolden {
		if err := os.MkdirAll(goldenDir(), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		err := os.WriteFile(goldenPath, append(got, '\n'), 0o644)
		if err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	compare(got, want)
}

func TestGolden_JSONCodec_Encode(t *testing.T) {
	c := codec.JSONCodec{}

	payload := goldenPayload{Email: "alice@example.com", Name: "Alice", Age: 30}

	got, err := c.Encode(payload)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	goldenPath := filepath.Join(goldenDir(), "json_encode.json")

	assertCodecGolden(t, goldenPath, got, func(got, want []byte) {
		if string(got) != string(want)[:len(string(want))-1] {
			t.Errorf(
				"JSON encode mismatch (run with -update to refresh golden files)\n got: %s\nwant: %s",
				got,
				want,
			)
		}
	})
}

func TestGolden_RawCodec_Passthrough(t *testing.T) {
	c := codec.RawCodec{}

	input := []byte(`{"raw":true}`)

	got, err := c.Encode(input)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	goldenPath := filepath.Join(goldenDir(), "raw_passthrough.bin")

	assertCodecGolden(t, goldenPath, got, func(actual, expected []byte) {
		if string(actual) != string(expected)[:len(string(expected))-1] {
			t.Errorf(
				"Raw passthrough mismatch\n got: %s\nwant: %s",
				actual,
				expected,
			)
		}
	})
}
