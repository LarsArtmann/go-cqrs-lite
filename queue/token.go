package queue

import (
	"crypto/rand"
	"encoding/hex"
)

// NewClaimToken mints the per-claim holder proof: 16 crypto-random bytes
// hex-encoded. Engines stamp it beside the lease at ClaimDue time and
// every finalize predicate re-checks it, so only the worker that holds
// the CURRENT claim can finalize, renew, or cooperatively cancel — a
// worker whose lease lapsed and whose task was re-claimed loses to the
// new token and sees ErrLeaseNotHeld (ADR-0134). Unguessability is the
// security property: a forged token must be infeasible, hence crypto/rand
// and never a counter.
func NewClaimToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("queue: crypto/rand failed: " + err.Error())
	}

	return hex.EncodeToString(b[:])
}
