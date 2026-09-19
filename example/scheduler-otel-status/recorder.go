package main

import (
	"context"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
)

// newClaimRecorder bridges the store's ClaimMetrics hooks to OTel counters —
// the manual wiring scheduling/sqlstore deliberately leaves to consumers so
// the module itself carries no OTel dependency. This file IS that recipe.
func newClaimRecorder(meter cqrsotel.Meter) (sqlstore.ClaimMetrics, error) {
	batches, err := meter.Int64Counter(
		"cqrs.scheduler.claim.batches",
		cqrsotel.CounterMetricWithDescription(
			"Committed Due polls, empty ones included (poller heartbeat)",
		),
	)
	if err != nil {
		return sqlstore.ClaimMetrics{}, err
	}

	timers, err := meter.Int64Counter("cqrs.scheduler.claim.timers",
		cqrsotel.CounterMetricWithDescription("Timers claimed across all polls"))
	if err != nil {
		return sqlstore.ClaimMetrics{}, err
	}

	renewed, err := meter.Int64Counter("cqrs.scheduler.claim.renewed",
		cqrsotel.CounterMetricWithDescription("Successful RenewLease extensions"))
	if err != nil {
		return sqlstore.ClaimMetrics{}, err
	}

	rejected, err := meter.Int64Counter("cqrs.scheduler.claim.renew_rejected",
		cqrsotel.CounterMetricWithDescription("Renewals rejected with ErrLeaseNotHeld"))
	if err != nil {
		return sqlstore.ClaimMetrics{}, err
	}

	ctx := context.Background()

	return sqlstore.ClaimMetrics{
		Claimed: func(count int) {
			batches.Add(ctx, 1)
			timers.Add(ctx, int64(count))
		},
		Renewed:       func() { renewed.Add(ctx, 1) },
		RenewRejected: func() { rejected.Add(ctx, 1) },
	}, nil
}
