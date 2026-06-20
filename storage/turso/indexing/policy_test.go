package indexing_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v2/indexing"
)

func TestPolicy_Empty(t *testing.T) {
	t.Parallel()

	p := indexing.NewPolicy()
	if p == nil {
		t.Fatal("NewPolicy returned nil")
	}
	if p.ShouldExclude("any") {
		t.Error("expected empty policy to not exclude")
	}
	if p.IsCritical("any") {
		t.Error("expected empty policy to not mark critical")
	}
	if p.ShouldSkipAutoCreate("any") {
		t.Error("expected empty policy to not skip auto-create")
	}
}

func TestPolicy_Nil(t *testing.T) {
	t.Parallel()

	var p *indexing.Policy
	if p.ShouldExclude("any") {
		t.Error("nil policy should not exclude")
	}
	if p.IsCritical("any") {
		t.Error("nil policy should not mark critical")
	}
	if p.ShouldSkipAutoCreate("any") {
		t.Error("nil policy should not skip")
	}
}

func TestPolicy_Exclude(t *testing.T) {
	t.Parallel()

	p := indexing.NewPolicy()
	p.Exclude("audit_log", "trace")

	if !p.ShouldExclude("audit_log") {
		t.Error("expected audit_log to be excluded")
	}
	if !p.ShouldExclude("trace") {
		t.Error("expected trace to be excluded")
	}
	if p.ShouldExclude("events") {
		t.Error("expected events to not be excluded")
	}
}

func TestPolicy_MarkCritical(t *testing.T) {
	t.Parallel()

	p := indexing.NewPolicy()
	p.MarkCritical("events")

	if !p.IsCritical("events") {
		t.Error("expected events to be critical")
	}
	if p.IsCritical("audit_log") {
		t.Error("expected audit_log to not be critical")
	}
}

func TestPolicy_MarkSkipAutoCreate(t *testing.T) {
	t.Parallel()

	p := indexing.NewPolicy()
	p.MarkSkipAutoCreate("events")

	if !p.ShouldSkipAutoCreate("events") {
		t.Error("expected events to skip auto-create")
	}
	if p.ShouldSkipAutoCreate("audit_log") {
		t.Error("expected audit_log to not skip auto-create")
	}
}
