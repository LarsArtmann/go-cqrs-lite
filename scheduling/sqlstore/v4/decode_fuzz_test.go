package sqlstore

import (
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// FuzzDecodeDueTimer drives the stored-row decoder with arbitrary bytes and
// IDs. The contract it pins: decodeDueTimer never panics, and EVERY failure
// is Corruption-family — a malformed row is bad stored data, never a caller
// error, so callers can drop the row without misclassifying the batch.
func FuzzDecodeDueTimer(f *testing.F) {
	f.Add(
		"timer-1",
		[]byte(
			`{"v":1,"actor":"user:01HK1540X0841Y0A6BSX1VKR99","payload":{"action":"cancel","amount":3}}`,
		),
	)
	f.Add("timer-1", []byte(`{"v":1,"payload":{"action":"remind","amount":7}}`))
	f.Add("timer-1", []byte(`{"action":"remind","amount":7}`))
	f.Add("timer-1", []byte(`{"v":1}`))
	f.Add("timer-1", []byte(`{"v":1,"payload":{"action":42}}`))
	f.Add("", []byte(`{}`))
	f.Add("timer-1", []byte(``))
	f.Add("timer-1", []byte(`not json at all`))
	f.Add("timer-1", []byte(`{"v":1,"actor":"weird actor","payload":{}}`))

	fireAt := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

	f.Fuzz(func(t *testing.T, rawID string, payload []byte) {
		timer, err := decodeDueTimer[decodeTestPayload](rawID, payload, fireAt)

		if err == nil {
			if timer.FireAt != fireAt {
				t.Fatalf("decoded FireAt = %v, want the stored %v", timer.FireAt, fireAt)
			}

			if timer.ID.String() == "" {
				t.Fatal("decoded timer carries an empty ID despite a successful parse")
			}

			return
		}

		if got := errorfamily.Classify(err); got != errorfamily.Corruption {
			t.Fatalf(
				"decodeDueTimer(%q, %q) error family = %v, want Corruption: %v",
				rawID, payload, got, err,
			)
		}
	})
}
