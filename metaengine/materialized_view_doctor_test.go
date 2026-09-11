package metaengine

import (
	"context"
	"strings"
	"testing"
)

// fakeMatViewEngine reports a fixed materialized-view set for Doctor tests.
type fakeMatViewEngine struct {
	Engine

	infos []MaterializedViewInfo
}

func (f *fakeMatViewEngine) MaterializedViews() []MaterializedViewInfo {
	return f.infos
}

// TestMaterializedViewsDoctorSection_Content pins the section's line shape:
// collection, physical view name, aggregate shape (COUNT(*) special case,
// fn(column) otherwise, " BY group" suffix for grouped), and live row count.
func TestMaterializedViewsDoctorSection_Content(t *testing.T) {
	t.Parallel()

	store := &Store{engines: []Engine{&fakeMatViewEngine{
		infos: []MaterializedViewInfo{
			{
				Spec: MaterializedViewSpec{Collection: "order_views", Fn: MatViewCount},
				Name: "cqrs_mv_order_views_count",
				Rows: 42,
			},
			{
				Spec: MaterializedViewSpec{
					Collection: "order_views",
					Fn:         MatViewSum,
					Column:     "total",
				},
				Name: "cqrs_mv_order_views_sum",
				Rows: 7,
			},
		},
	}}}

	section := store.MaterializedViewsDoctorSection(context.Background())

	for _, want := range []string{
		"--- Materialized views ---",
		"order_views: cqrs_mv_order_views_count [COUNT(*)] (rows=42)",
		"order_views: cqrs_mv_order_views_sum [SUM(total)] (rows=7)",
	} {
		if !strings.Contains(section, want) {
			t.Errorf("section missing %q:\n%s", want, section)
		}
	}
}

// TestMaterializedViewsDoctorSection_NoneBranch pins the explicit "none"
// line when no engine reports materialized views (an empty section would be
// ambiguous: broken reporter vs nothing declared).
func TestMaterializedViewsDoctorSection_NoneBranch(t *testing.T) {
	t.Parallel()

	store := &Store{engines: []Engine{NewMemoryEngine()}}

	section := store.MaterializedViewsDoctorSection(context.Background())

	if !strings.Contains(section, "  none\n") {
		t.Errorf("expected explicit 'none' line, got:\n%s", section)
	}
}

// TestMaterializedViewsDoctorSection_GroupedWarnPin pins the grouped-view
// upstream-defect warning: tursogo <= v0.8.0-pre.10 maintains grouped
// materialized views incorrectly once a second transaction updates a group
// (verified 2026-09-07, see docs/research/2026-09-07_turso-go-*). This pin
// flips loudly if the caveat text is edited — when upstream fixes the defect,
// update the WARN here AND the serving-side guards in the same change.
func TestMaterializedViewsDoctorSection_GroupedWarnPin(t *testing.T) {
	t.Parallel()

	store := &Store{engines: []Engine{&fakeMatViewEngine{
		infos: []MaterializedViewInfo{
			{Spec: MaterializedViewSpec{
				Collection: "orders", Fn: MatViewSum, Column: "total", GroupBy: "region",
			}, Name: "cqrs_mv_orders_sum", Rows: 3},
		},
	}}}

	section := store.MaterializedViewsDoctorSection(context.Background())

	if !strings.Contains(section, "SUM(total) BY region") {
		t.Errorf("grouped shape line missing:\n%s", section)
	}
	if !strings.Contains(section, "WARN: grouped views return silently wrong aggregates") {
		t.Errorf("grouped-view WARN missing from Doctor section:\n%s", section)
	}
	if !strings.Contains(section, "scalar views are the safe shape") {
		t.Errorf("WARN should steer operators to the scalar shape:\n%s", section)
	}
}

// TestMaterializedViewsDoctorSection_NoGroupNoWarn: scalar specs must NOT
// carry the grouped-view warning (a blanket WARN would train operators to
// ignore it).
func TestMaterializedViewsDoctorSection_NoGroupNoWarn(t *testing.T) {
	t.Parallel()

	store := &Store{engines: []Engine{&fakeMatViewEngine{
		infos: []MaterializedViewInfo{
			{
				Spec: MaterializedViewSpec{Collection: "orders", Fn: MatViewSum, Column: "total"},
				Name: "cqrs_mv_orders",
				Rows: 1,
			},
		},
	}}}

	if section := store.MaterializedViewsDoctorSection(
		context.Background(),
	); strings.Contains(
		section,
		"WARN",
	) {
		t.Errorf("scalar spec must not warn:\n%s", section)
	}
}
