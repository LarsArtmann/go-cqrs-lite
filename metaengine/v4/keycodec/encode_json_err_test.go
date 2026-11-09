package keycodec

import (
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestEncodeJSONErr(t *testing.T) {
	t.Parallel()

	t.Run("representable value round-trips", func(t *testing.T) {
		t.Parallel()

		// Struct fields emit in declaration order — deterministic. (A map
		// literal would be flaky: encoding/json/v2 does not sort map keys,
		// gotchas-language-footguns.)
		type repRow struct {
			Status string  `json:"status"`
			N      float64 `json:"n"`
		}

		b, err := EncodeJSONErr(repRow{Status: "open", N: 3})
		if err != nil {
			t.Fatalf("EncodeJSONErr: %v", err)
		}

		if got := string(b); got != `{"status":"open","n":3}` {
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
