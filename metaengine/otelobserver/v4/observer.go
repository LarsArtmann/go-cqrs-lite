// Package otelobserver records metaengine health transitions — engine
// quarantine, reactivation, health probes, and catch-up rebuilds (ADR-0137)
// — as OpenTelemetry counters, so operators can watch failover on a
// dashboard instead of grepping logs or polling Doctor.
//
// The metaengine core stays dependency-free by design; this module is the
// bridge between its typed hooks and an OTel meter:
//
//	store, _ := metaengine.Plan(engines, queries...)
//	provider, _ := cqrsotel.Setup() // or your own MeterProvider
//	obs, _ := otelobserver.Attach(store, provider.Meter("cqrs"))
//
// Attach merges with any hooks already configured on the store, so a
// metrics recorder (metaengine.WithMetrics) and this observer compose.

package otelobserver

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// Observer records metaengine health transitions as OpenTelemetry counters.
// Create with New and wire with Hooks or Attach.
type Observer struct {
	quarantines   cqrsotel.Int64Counter
	reactivations cqrsotel.Int64Counter
	probes        cqrsotel.Int64Counter
	catchups      cqrsotel.Int64Counter
	catchupEvents cqrsotel.Int64Counter
}

// New creates an Observer recording health transitions on the given meter.
// Instruments (all prefixed cqrs. so cqrsotel.NewCQRSViews applies):
//
//	cqrs.metaengine.quarantine.total  {engine}                    — engine quarantined
//	cqrs.metaengine.reactivate.total  {engine, reason}            — quarantine lifted (manual|catchup|probe-fallback)
//	cqrs.metaengine.probe.total       {engine, outcome}           — health probe (ok|fail)
//	cqrs.metaengine.catchup.total     {engine, outcome}           — catch-up rebuild attempt (ok|fail)
//	cqrs.metaengine.catchup.replayed  {engine}                    — events replayed into a rebuilding engine
func New(meter cqrsotel.Meter) (*Observer, error) {
	o := &Observer{}

	instruments := []struct {
		dest *cqrsotel.Int64Counter
		name string
		desc string
	}{
		{
			&o.quarantines, "cqrs.metaengine.quarantine.total",
			"Engines quarantined after consecutive classified failures",
		},
		{
			&o.reactivations, "cqrs.metaengine.reactivate.total",
			"Engine reactivations by reason (manual, catchup, probe-fallback)",
		},
		{
			&o.probes, "cqrs.metaengine.probe.total",
			"Health probes by outcome (ok, fail)",
		},
		{
			&o.catchups, "cqrs.metaengine.catchup.total",
			"Catch-up rebuild attempts by outcome (ok, fail)",
		},
		{
			&o.catchupEvents, "cqrs.metaengine.catchup.replayed",
			"Events replayed into engines during catch-up rebuilds",
		},
	}

	for _, inst := range instruments {
		c, err := meter.Int64Counter(inst.name, cqrsotel.CounterMetricWithDescription(inst.desc))
		if err != nil {
			return nil, err //nolint:wrapcheck // otel SDK error, instrument-scoped
		}

		*inst.dest = c
	}

	return o, nil
}

// Hooks returns the metaengine.Hooks that feed the observer. Wire them with
// metaengine.WithHooks — or use Attach, which merges these into any hooks
// the store already has.
func (o *Observer) Hooks() metaengine.Hooks {
	ctx := context.Background()

	return metaengine.Hooks{
		OnQuarantined: func(engine string, _ int, _ string) {
			o.quarantines.Add(ctx, 1, cqrsotel.CounterAddWithAttributes(
				cqrsotel.AttrString("engine", engine),
			))
		},
		OnReactivated: func(engine string, reason string) {
			o.reactivations.Add(ctx, 1, cqrsotel.CounterAddWithAttributes(
				cqrsotel.AttrString("engine", engine),
				cqrsotel.AttrString("reason", reason),
			))
		},
		OnProbe: func(engine string, err error) {
			outcome := "ok"
			if err != nil {
				outcome = "fail"
			}

			o.probes.Add(ctx, 1, cqrsotel.CounterAddWithAttributes(
				cqrsotel.AttrString("engine", engine),
				cqrsotel.AttrString("outcome", outcome),
			))
		},
		OnCatchUp: func(engine string, replayed int, err error) {
			outcome := "ok"
			if err != nil {
				outcome = "fail"
			}

			o.catchups.Add(ctx, 1, cqrsotel.CounterAddWithAttributes(
				cqrsotel.AttrString("engine", engine),
				cqrsotel.AttrString("outcome", outcome),
			))

			if replayed > 0 {
				o.catchupEvents.Add(ctx, int64(replayed), cqrsotel.CounterAddWithAttributes(
					cqrsotel.AttrString("engine", engine),
				))
			}
		},
	}
}

// Attach creates an Observer for the meter and wires it into the store,
// merging with any hooks already configured there (their callbacks keep
// running). Configure hooks at construction, before concurrent use.
func Attach(store *metaengine.Store, meter cqrsotel.Meter) (*Observer, error) {
	o, err := New(meter)
	if err != nil {
		return nil, err //nolint:wrapcheck // already instrument-scoped
	}

	metaengine.WithHooks(store, store.CurrentHooks().Merge(o.Hooks()))

	return o, nil
}
