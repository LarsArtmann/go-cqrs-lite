## Problem

`Store.EnforceCap` / `EnforceCapGrouped` can deadlock permanently against a concurrent fold batch on any Transactional engine with a bounded connection pool (observed on sqliteengine, webphone's projection host: order-dependent 60–600 s hangs, ~1-in-2 runs under a contacts+transcripts battery).

Lock-order inversion:

- Fold path (`dispatchFoldsCoreLocked`, `metaengine/runtime_backend.go:380`): opens the engine transaction first (`runInTxWithRetry` → `RunInTx` acquires `txMu` + pooled conn + BEGIN) and takes each query's fold lock INSIDE it → **conn → foldLock**.
- Trim path (pre-fix `metaengine/cap.go:169`): takes the query's fold lock first and holds it across `capScanPage`/`capDelete`, which each need a pooled conn → **foldLock → conn**.

With the pool exhausted (sqlite single-conn setups pin this shape; `newEngine` in `sqliteengine/dueclaim_test.go:26` sets `MaxOpenConns(1)`), the fold holds a connection wanting the fold lock while the trim holds the fold lock wanting a connection. Neither side uses ctx-aware mutexes, so the hang is permanent.

The replicator already had the right order (`replicator.go:188` — `RunInTx` outside, fold lock inside); the trim was the only inverter.

## Fix

`metaengine/cap.go` @ de39a43a9: new `withCapTrimTx` helper runs the trim inside the engine's `RunInTx` (connection acquired first) with the fold lock taken inside — matching the fold path's order for every Transactional engine. Non-transactional engines keep the plain fold lock (holders never wait on a pool). `EnforceCapGrouped` fixed identically. Bonus: the whole trim is now atomic per engine (one commit) instead of per-row deletes.

## Evidence

Regression test `metaengine/sqliteengine/cap_deadlock_test.go` (same commit): one-connection pool, 400 concurrent `Apply`s racing 60 `EnforceCap`s, 60 s watchdog.

- pre-fix (e0404f6e7): FAIL — 90 s package timeout, both goroutines parked in `sync.WaitGroup.Wait`
- post-fix (de39a43a9): `ok ... 0.066s`, `-count=3` green

Full cap suites stay green: `go test ./metaengine/ -run TestWithCap` and the sqliteengine grouped-cap contracts.

💘 Generated with Crush
