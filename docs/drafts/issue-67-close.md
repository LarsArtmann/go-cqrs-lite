Fixed on master: a drain error no longer skips Close. system/lifecycle.go:137 now runs Close() unconditionally when draining fails and returns errors.Join(drainErr, closeErr) — the drain-error path never re-enters the Phase 2 select, so the pre-expired-context determinism guard is untouched.

Proof: TestSystem_GracefulClose_DrainError_StillCloses (the old NoClose test inverted — engine must close exactly once despite the drain error), and TestSystem_GracefulClose_DrainTimeout now also asserts the engine closed after the drain context expired. system + systemtest + systemscenario green, -race included.

go-appkit can drop its hand-rolled Drain + always-Close + Join wrapper now — GracefulClose matches that contract, TestEventService_Shutdown_DrainTimeoutStillCloses included.
