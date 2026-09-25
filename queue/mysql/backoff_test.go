package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

// deadlockBackoff contract (queue M4 tail (a), 2026-09-25): exponential
// growth from claimRetryBaseDelay capped at claimRetryMaxDelay, full jitter
// over [0, cap], never negative, and the cap engages by the attempt whose
// un-jittered delay would exceed it.
func TestDeadlockBackoff_BoundsAndShape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		attempt   int
		upperBound time.Duration
	}{
		{0, claimRetryBaseDelay},
		{1, 2 * claimRetryBaseDelay},
		{2, 4 * claimRetryBaseDelay},
		{3, 8 * claimRetryBaseDelay}, // 200ms > cap already at attempt 3
		{7, claimRetryMaxDelay},
		{63, claimRetryMaxDelay},
	}

	for _, tc := range cases {
		for range 64 {
			got := deadlockBackoff(tc.attempt)
			if got < 0 {
				t.Fatalf("attempt %d: negative backoff %v", tc.attempt, got)
			}

			want := min(claimRetryBaseDelay<<tc.attempt, claimRetryMaxDelay)
			if got > want {
				t.Fatalf("attempt %d: backoff %v exceeds un-jittered cap %v", tc.attempt, got, want)
			}
		}
	}
}

// TestDeadlockBackoff_JitterActuallySpreads pins that the jitter is real:
// over many samples of one attempt the distribution must touch both halves
// of [0, cap] — a broken (constant) backoff would sit in one half.
func TestDeadlockBackoff_JitterActuallySpreads(t *testing.T) {
	t.Parallel()

	const samples = 512

	cap2 := min(claimRetryBaseDelay<<2, claimRetryMaxDelay) // attempt 2's cap
	low, high := 0, 0

	for range samples {
		if deadlockBackoff(2) <= cap2/2 {
			low++
		} else {
			high++
		}
	}

	if low < samples/10 || high < samples/10 {
		t.Fatalf("jitter collapsed: low=%d high=%d of %d — backoff is effectively constant", low, high, samples)
	}
}

// TestDeadlockBackoff_ExponentialGrowthMonotoneInExpectation pins the
// exponential shape: the mean over samples must grow with the attempt.
func TestDeadlockBackoff_ExponentialGrowth(t *testing.T) {
	t.Parallel()

	mean := func(attempt int) float64 {
		var sum time.Duration
		const n = 1024

		for range n {
			sum += deadlockBackoff(attempt)
		}

		return float64(sum) / float64(n)
	}

	m0, m1 := mean(0), mean(1)
	if m1 <= m0*1.2 { // 2x cap in theory; anything <=1.2x means no growth
		t.Fatalf("backoff not exponential: mean(1)=%v not > 1.2*mean(0)=%v", m1, m0*1.2)
	}
}

// TestSleep_CancelledContextAborts pins the retry sleep is ctx-aborting:
// a cancelled context returns an error wrapping ctx.Err() instead of
// sleeping out the full window.
func TestSleep_CancelledContextAborts(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := sleep(ctx, claimRetryMaxDelay)

	if err == nil {
		t.Fatal("cancelled ctx must error, not sleep")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error must wrap context.Canceled, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > claimRetryBaseDelay {
		t.Fatalf("cancelled sleep must return immediately, took %v", elapsed)
	}
}

// TestIsDeadlock pins the InnoDB signal numbers and the non-signal pass-through.
func TestIsDeadlock(t *testing.T) {
	t.Parallel()

	for _, num := range []uint16{1213, 1205} {
		if !isDeadlock(&mysqldriver.MySQLError{Number: num}) {
			t.Errorf("error %d must classify as deadlock", num)
		}
	}

	for _, err := range []error{nil, errors.New("boom"), &mysqldriver.MySQLError{Number: 1064}} {
		if isDeadlock(err) {
			t.Errorf("err %v must NOT classify as deadlock", err)
		}
	}
}
