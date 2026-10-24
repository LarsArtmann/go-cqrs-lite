package keycodec

import (
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
	// EncodeJSON would store its %v rendering into the read model.
	t.Run("channel value is a Rejection", func(t *testing.T) {
		t.Parallel()

		_, err := EncodeJSONErr(make(chan int))
		assertEncodeValueRejection(t, err)
	})

	t.Run("function value is a Rejection", func(t *testing.T) {
		t.Parallel()

		_, err := EncodeJSONErr(func() {})
		assertEncodeValueRejection(t, err)
	})

	t.Run("cyclic value is a Rejection", func(t *testing.T) {
		t.Parallel()

		cyclic := map[string]any{}
		cyclic["self"] = cyclic

		_, err := EncodeJSONErr(cyclic)
		assertEncodeValueRejection(t, err)
	})
}

// assertEncodeValueRejection pins family AND code, so a later refactor cannot
// move the failure to a different stage while the family stays Rejection.
func assertEncodeValueRejection(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("error %v classified as %v, want Rejection", err, family)
	}

	if code := errorfamily.Code(err); code != "keycodec.encode_value" {
		t.Fatalf("error code = %q, want keycodec.encode_value (%v)", code, err)
	}
}
