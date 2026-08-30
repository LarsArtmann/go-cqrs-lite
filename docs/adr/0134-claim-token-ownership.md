# ADR-0134: Per-Claimer Claim Tokens for ClaimingTimerStore

- Status: Proposed (stub — design not implemented; scheduled behind the
  RenewLease adoption window)
- Date: 2026-08-30
- Context: `scheduling/sqlstore` D8 claiming extensions
  (RenewLease shipped 2026-08-30, commit `6f5fb66a0`)

## Context

`ClaimingTimerStore.RenewLease(ctx, id, extend)` extends whichever LIVE claim
exists for the timer — claims carry no per-poller identity. This is safe in
the direction that matters for the no-double-fire guarantee (renewal can only
EXTEND a fence, never release it), but it has two consequences:

1. Any instance that can reach the database can renew any live lease. A
   buggy or malicious poller can pin a timer's lease forever (liveness bug,
   not a correctness bug: MarkFired still deletes the row).
2. A handler whose lease expired AND whose timer was re-claimed by another
   instance cannot detect the theft — it happily continues, breaking the
   no-double-fire promise that the lease made to it.

## Decision (proposed shape)

Introduce an opaque, per-claim token:

- `Due` mints a random token per claimed row and returns it with the timer
  (`ClaimedTimer{Timer, Token}` or a `Token() string` accessor).
- `RenewLease(ctx, id, token, extend)` gains the token: the claim UPDATE
  adds `AND lease_token = $token`, so only the current holder can extend.
- `MarkFired(ctx, id, token)` likewise: firing requires proof of holding,
  which turns the fire path into the theft detector — a handler whose lease
  lapsed and whose timer was re-claimed gets ErrLeaseNotHeld on MarkFired.
- The token column rides the same idempotent `ensureLeaseColumn`-style
  migration as `lease_until`; NULL tokens (pre-migration rows) keep
  today's token-less semantics for one release.

## Consequences

- Breaking-ish: `RenewLease` and `MarkFired` signatures gain a parameter.
  v4 ships the token as an ADDITIVE overload pair
  (`RenewLeaseWithToken` / `MarkFiredWithToken`) with the token-less forms
  deprecated toward v5, or defers entirely to v5 — decide at wave time.
- The token must be unguessable (crypto/rand), otherwise the holder proof
  is forgeable.
- Fenced-write literature: this is the standard fencing-token pattern
  (Kleinberg; Kleppmann) applied at the store boundary; document the
  relationship in the migration guide.

## Alternatives considered

- Owner ID string per instance (UUID at startup): simpler, but does not
  detect theft after lease transfer (the "owner" persists in the row), and
  restarts collide. Rejected.
- Lease epochs (monotonic counter per timer): equivalent strength to random
  tokens without the entropy requirement, but leaks sequence info in the
  row and complicates the UPDATE predicate marginally. Viable fallback if
  crypto/rand cost ever matters (it does not at timer-claim rates).
