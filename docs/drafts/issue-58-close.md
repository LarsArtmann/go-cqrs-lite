Triage verdict: the separable ask is already satisfied and the seam decision is routed to the backlog as an owner question.

1. **CBOR round-trip pin exists** — `TestEventAdapter_CBORPayloadRoundTripSQLite` (system module) pins exactly what you asked: a CBOR payload survives SQLite persist → load with `Encoding` + metadata intact, asserted through `DecodePayloadAuto`. Added since the master rev this issue cited; green at HEAD.

2. **Seam vs procedure → backlog row with owner verdict** — option (1) (a `DeploymentConfig` `event.Store` override) is a second construction path through the ADR-0123 composition root and needs design review, not a drive-by; option (2) (the aggregate-agnostic migration helper) is the honest adopt-system story. Both are captured in TODO_LIST (#58 row) with your verification work quoted as the design input — the envelope analysis (`serializedEvent` opaque payload + codec + `ReconstructEventFromFields`) is exactly the right groundwork.

Keeping this open would just restate the backlog row — closing as tracked there. If the verdict lands on (2), the helper is small and your journal layout is the first consumer test.
