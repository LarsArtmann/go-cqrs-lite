package indexing

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// AutoIndexerOption configures an AutoIndexer.
type AutoIndexerOption func(*AutoIndexer)

// WithAutoAnalyze runs ANALYZE after creating indexes.
func WithAutoAnalyze() AutoIndexerOption {
	return func(a *AutoIndexer) { a.autoAnalyze = true }
}

// WithDryRun prevents any DDL execution. DDL statements are collected
// and returned by ApplyCQRSIndexes (via a callback) instead of being
// executed. Useful for testing and review.
func WithDryRun() AutoIndexerOption {
	return func(a *AutoIndexer) { a.dryRun = true }
}

// LastDDL returns the DDL statements that would have been executed
// by the most recent Apply* call. Only populated when WithDryRun is set.
func (a *AutoIndexer) LastDDL() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := make([]string, len(a.lastDDL))
	copy(out, a.lastDDL)

	return out
}

// AutoIndexer applies index recommendations automatically.
// Call Enable to allow it to create indexes; call Disable to stop.
type AutoIndexer struct {
	advisor     *Advisor
	db          *sql.DB
	mu          sync.RWMutex
	enabled     bool
	autoAnalyze bool
	dryRun      bool
	lastDDL     []string
	hooksConfig hooks
}

// NewAutoIndexer creates an auto-indexer for the given database.
func NewAutoIndexer(db *sql.DB, opts ...AutoIndexerOption) *AutoIndexer {
	a := &AutoIndexer{
		advisor: NewAdvisor(db),
		db:      db,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Enable allows the auto-indexer to create indexes.
func (a *AutoIndexer) Enable() {
	a.setEnabled(true)
}

// Disable prevents the auto-indexer from creating indexes.
func (a *AutoIndexer) Disable() {
	a.setEnabled(false)
}

// setEnabled toggles the auto-indexer's active flag under the mutex.
func (a *AutoIndexer) setEnabled(v bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.enabled = v
}

// IsEnabled reports whether auto-indexing is active.
func (a *AutoIndexer) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.enabled
}

// rejectIfDisabled returns a rejection error when the auto-indexer is not
// enabled, recording it on the span. Returns nil when enabled.
func (a *AutoIndexer) rejectIfDisabled(span trace.Span) error {
	if a.IsEnabled() {
		return nil
	}

	err := errorfamily.NewRejection("indexing.disabled",
		"auto-indexer is disabled: call Enable() first")
	cqrsotel.RecordError(span, err)

	return err
}

// Apply executes the DDL for each recommendation.
// Skips indexes that already exist.
// Returns a rejection error if the auto-indexer is not enabled.
func (a *AutoIndexer) Apply(ctx context.Context, recs []Recommendation) error {
	ctx, span := startIndexingSpan(
		ctx,
		SpanAutoIndexerApply,
		trace.WithAttributes(attribute.Int(AttrRecommendationCount, len(recs))),
	)
	defer endSpan(span, nil)

	if err := a.rejectIfDisabled(span); err != nil {
		return err
	}

	if err := a.advisor.ExistingIndexes(ctx); err != nil {
		cqrsotel.RecordError(span, err)

		return err
	}

	created := 0

	for _, rec := range recs {
		if a.advisor.HasIndex(rec.Index.Name) {
			continue
		}

		if err := a.createIndex(ctx, rec.Index); err != nil {
			wrappedErr := errorfamily.WrapInfrastructure(err, "indexing.create_index",
				fmt.Sprintf("create index %s on %s", rec.Index.Name, rec.Index.Table))
			cqrsotel.RecordError(span, wrappedErr)

			return wrappedErr
		}

		created++
	}

	span.SetAttributes(attribute.Int("indexing.index.created", created))

	if err := a.maybeAnalyze(ctx); err != nil {
		cqrsotel.RecordError(span, err)
	}

	return nil
}

func (a *AutoIndexer) maybeAnalyze(ctx context.Context) error {
	if !a.autoAnalyze {
		return nil
	}

	return Analyze(ctx, a.db) //nolint:wrapcheck // transparent delegation
}

// ApplyRecommended runs MissingIndexes and applies all recommendations.
func (a *AutoIndexer) ApplyRecommended(ctx context.Context) error {
	ctx, span := startIndexingSpan(ctx, "indexing.auto_indexer.apply_recommended")
	defer endSpan(span, nil)

	if err := a.rejectIfDisabled(span); err != nil {
		return err
	}

	recs, err := a.advisor.MissingIndexes(ctx)
	if err != nil {
		wrappedErr := errorfamily.WrapInfrastructure(err, "indexing.missing_indexes",
			"find missing indexes")
		cqrsotel.RecordError(span, wrappedErr)

		return wrappedErr
	}

	span.SetAttributes(attribute.Int(AttrRecommendationCount, len(recs)))

	return a.Apply(ctx, recs)
}

// ApplyCQRSIndexes creates the predefined RecommendedCQRSIndexes.
// This is the fastest way to get production-ready CQRS indexing.
// Returns a rejection error if the auto-indexer is not enabled.
func (a *AutoIndexer) ApplyCQRSIndexes(ctx context.Context) error {
	ctx, span := startIndexingSpan(ctx, SpanAutoIndexerApplyCQRS)
	defer endSpan(span, nil)

	if err := a.rejectIfDisabled(span); err != nil {
		return err
	}

	if err := a.advisor.ExistingIndexes(ctx); err != nil {
		cqrsotel.RecordError(span, err)

		return err
	}

	created := 0

	for _, idx := range RecommendedCQRSIndexes() {
		if a.advisor.HasIndex(idx.Name) {
			continue
		}

		if err := a.createIndex(ctx, idx); err != nil {
			wrappedErr := errorfamily.WrapInfrastructure(err, "indexing.create_cqrs_index",
				fmt.Sprintf("create CQRS index %s", idx.Name))
			cqrsotel.RecordError(span, wrappedErr)

			return wrappedErr
		}

		recordIndexAttributes(span, idx)
		created++
	}

	span.SetAttributes(attribute.Int("indexing.index.created", created))

	if err := a.maybeAnalyze(ctx); err != nil {
		cqrsotel.RecordError(span, err)
	}

	return nil
}

// Recommendations returns current recommendations without applying them.
func (a *AutoIndexer) Recommendations(ctx context.Context) ([]Recommendation, error) {
	return a.advisor.MissingIndexes(ctx) //nolint:wrapcheck // transparent delegation
}

// Close releases any resources held by the AutoIndexer.
// The underlying *sql.DB is owned by the caller and not closed.
func (a *AutoIndexer) Close() error {
	a.setEnabled(false)

	return nil
}

// Drop removes the given indexes from the database.
// Skips indexes that don't exist. Returns a rejection error if the
// auto-indexer is not enabled.
func (a *AutoIndexer) Drop(ctx context.Context, indexes ...Index) error {
	ctx, span := startIndexingSpan(
		ctx,
		SpanAutoIndexerDrop,
		trace.WithAttributes(attribute.Int("indexing.index.count", len(indexes))),
	)
	defer endSpan(span, nil)

	if err := a.rejectIfDisabled(span); err != nil {
		return err
	}

	dropped := 0

	for _, idx := range indexes {
		if err := a.hooksConfig.fireBeforeDrop(ctx, idx, a); err != nil {
			return err
		}

		ddl := idx.DropDDL()

		_, err := a.db.ExecContext(ctx, ddl)
		if err != nil {
			wrappedErr := errorfamily.WrapInfrastructure(err, "indexing.drop_index",
				fmt.Sprintf("drop index %s", idx.Name))
			cqrsotel.RecordError(span, wrappedErr)

			return wrappedErr
		}

		a.hooksConfig.fireAfterDrop(ctx, idx, a)
		dropped++
	}

	span.SetAttributes(attribute.Int("indexing.index.dropped", dropped))

	return nil
}

// RecommendAndApply combines MissingIndexes and ApplyRecommended in one call.
// Convenience for the most common auto-indexing workflow.
func (a *AutoIndexer) RecommendAndApply(ctx context.Context) error {
	return a.ApplyRecommended(ctx) //nolint:wrapcheck // transparent delegation
}

func (a *AutoIndexer) createIndex(ctx context.Context, idx Index) error {
	if err := a.hooksConfig.fireBeforeCreate(ctx, idx, a); err != nil {
		return err
	}

	ddl := idx.DDL()

	a.mu.Lock()
	a.lastDDL = append(a.lastDDL, ddl)
	dryRun := a.dryRun
	a.mu.Unlock()

	if dryRun {
		a.hooksConfig.fireAfterCreate(ctx, idx, a)

		return nil
	}

	_, err := a.db.ExecContext(ctx, ddl)
	if err != nil {
		if isIndexAlreadyExists(err) {
			return nil
		}

		return err //nolint:wrapcheck // caller wraps with context
	}

	a.hooksConfig.fireAfterCreate(ctx, idx, a)

	return nil
}

// isIndexAlreadyExists reports whether err is the "index already exists"
// message the SQLite-compatible driver returns when CREATE INDEX hits a duplicate name. It is
// an idempotency guard so concurrent AutoIndexer runs don't fail when another
// run created the same index first. The driver exposes no typed code for this
// case, so the message contract is matched explicitly and lowercased to stay
// robust against capitalization differences across driver versions.
func isIndexAlreadyExists(err error) bool {
	return errContainsAny(err, "already exists")
}
