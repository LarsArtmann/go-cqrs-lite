package keycodec

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestEncodeJSONErr(t *testing.T) {
	t.Parallel()

	t.Run("representable value round-trips", func(t *testing.T) {
		t.Parallel()

		b, err := EncodeJSONErr(map[string]any{"status": "open", "n": float64(3)})
		if err != nil {
			t.Fatalf("EncodeJSONErr: %v", err)
		}

		if got := string(b); got != `{"n":3,"status":"open"}` {
			t.Fatalf("encoded = %s", got)
		}
	})

	// A channel is the canonical non-representable value: the lenient
	// EncodeJSON would store "0x..." into the read model.
	t.Run("channel value is a Rejection", func(t *testing.T) {
		t.Parallel()

		_, err := EncodeJSONErr(make(chan int))
		if err == nil {
			t.Fatal("channel value must not encode")
		}

		assertEncodeValueRejection(t, err, "chan int")
	})

	t.Run("function value is a Rejection", func(t *testing.T) {
		t.Parallel()

		_, err := EncodeJSONErr(func() {})
		if err == nil {
			t.Fatal("function value must not encode")
		}

		assertEncodeValueRejection(t, err, "func()")
	})

	t.Run("cyclic value is a Rejection", func(t *testing.T) {
		t.Parallel()

		cyclic := map[string]any{}
		cyclic["self"] = cyclic

		_, err := EncodeJSONErr(cyclic)
		if err == nil {
			t.Fatal("cyclic value must not encode")
		}

		assertEncodeValueRejection(t, err, "map[string]interface {}")
	})
}

func assertEncodeValueRejection(t *testing.T, err error, goType string) {
	t.Helper()

	if fam := errorfamily.FamilyOf(err); fam != errorfamily.Rejection {
		t.Errorf("family = %v, want Rejection (err: %v)", fam, err)
	}

	var coded errorfamily.Coded
	if !errors.As(err, &coded) {
		t.Fatalf("error must carry an errorfamily code: %v", err)
	}

	if code := coded.ErrorCode(); code != "keycodec.encode_value" {
		t.Errorf("code = %q, want keycodec.encode_value", code)
	}

	if want := "encode value " + goType; !errorfamily.Is(err, errorfamily.NewRejection("keycodec.encode_value", want)) {
		// Family + code already pinned above; this is a best-effort message shape check.
		t.Logf("message shape: %v (want prefix %q)", err, want)
	}
}
