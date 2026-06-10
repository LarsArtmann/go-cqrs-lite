package indexing

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// AutoIndexer applies index recommendations automatically.
// Call Enable to allow it to create indexes; call Disable to stop.
type AutoIndexer struct {
	advisor *Advisor
	db      *sql.DB
	mu      sync.RWMutex
	enabled bool
}

// NewAutoIndexer creates an auto-indexer for the given database.
func NewAutoIndexer(db *sql.DB) *AutoIndexer {
	return &AutoIndexer{
		advisor: NewAdvisor(db),
		db:      db,
	}
}

// Enable allows the auto-indexer to create indexes.
func (a *AutoIndexer) Enable() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.enabled = true
}

// Disable prevents the auto-indexer from creating indexes.
func (a *AutoIndexer) Disable() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.enabled = false
}

// IsEnabled reports whether auto-indexing is active.
func (a *AutoIndexer) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.enabled
}

// Apply executes the DDL for each recommendation.
// Skips indexes that already exist.
// Returns a rejection error if the auto-indexer is not enabled.
func (a *AutoIndexer) Apply(ctx context.Context, recs []Recommendation) error {
	if !a.IsEnabled() {
		return event.NewRejection("indexing.disabled",
			"auto-indexer is disabled: call Enable() first")
	}

	if err := a.advisor.ExistingIndexes(ctx); err != nil {
		return err
	}

	for _, rec := range recs {
		if a.advisor.HasIndex(rec.Index.Name) {
			continue
		}

		if err := a.createIndex(ctx, rec.Index); err != nil {
			return event.WrapInfrastructure(err, "indexing.create_index",
				fmt.Sprintf("create index %s on %s", rec.Index.Name, rec.Index.Table))
		}
	}

	return nil
}

// ApplyRecommended runs MissingIndexes and applies all recommendations.
func (a *AutoIndexer) ApplyRecommended(ctx context.Context) error {
	if !a.IsEnabled() {
		return event.NewRejection("indexing.disabled",
			"auto-indexer is disabled: call Enable() first")
	}

	recs, err := a.advisor.MissingIndexes(ctx)
	if err != nil {
		return event.WrapInfrastructure(err, "indexing.missing_indexes",
			"find missing indexes")
	}

	return a.Apply(ctx, recs)
}

// ApplyCQRSIndexes creates the predefined RecommendedCQRSIndexes.
// This is the fastest way to get production-ready CQRS indexing.
// Returns a rejection error if the auto-indexer is not enabled.
func (a *AutoIndexer) ApplyCQRSIndexes(ctx context.Context) error {
	if !a.IsEnabled() {
		return event.NewRejection("indexing.disabled",
			"auto-indexer is disabled: call Enable() first")
	}

	if err := a.advisor.ExistingIndexes(ctx); err != nil {
		return err
	}

	for _, idx := range RecommendedCQRSIndexes() {
		if a.advisor.HasIndex(idx.Name) {
			continue
		}

		if err := a.createIndex(ctx, idx); err != nil {
			return event.WrapInfrastructure(err, "indexing.create_cqrs_index",
				fmt.Sprintf("create CQRS index %s", idx.Name))
		}
	}

	return nil
}

// Recommendations returns current recommendations without applying them.
func (a *AutoIndexer) Recommendations(ctx context.Context) ([]Recommendation, error) {
	return a.advisor.MissingIndexes(ctx) //nolint:wrapcheck // transparent delegation
}

func (a *AutoIndexer) createIndex(ctx context.Context, idx Index) error {
	ddl := idx.DDL()

	_, err := a.db.ExecContext(ctx, ddl)
	if err != nil {
		// Ignore "index already exists" errors from race conditions.
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}

		return err //nolint:wrapcheck // caller wraps with context
	}

	return nil
}
