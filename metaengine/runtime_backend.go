package metaengine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ProjectionRole defines the purpose of a projection on a specific engine
// (ADR-0124 §7). Roles determine sync strategy when multiple engines serve
// the same query.
type ProjectionRole string

const (
	// RoleActive: the primary projection serving live queries. Synced via the
	// fold pipeline (strong consistency — events applied to all Active
	// projections in one transaction).
	RoleActive ProjectionRole = "Active"

	// RoleDualUse: two engines serving different query shapes simultaneously.
	// Synced via the fold pipeline (strong consistency, same as Active).
	RoleDualUse ProjectionRole = "DualUse"

	// RoleMigration: a new engine being populated. Switch over to Active when
	// caught up. Synced via async replication (eventual consistency).
	RoleMigration ProjectionRole = "Migration"

	// RoleBackup: a redundant copy for disaster recovery. Synced via async
	// replication (eventual consistency, failure-isolated).
	RoleBackup ProjectionRole = "Backup"
)

// AddedEngine tracks an engine added at runtime (ADR-0124 §7).
type AddedEngine struct {
	Engine     Engine
	Role       ProjectionRole
	Backfilled bool
}

// AddEngine registers a new engine at runtime and triggers a re-plan so the
// planner can route queries to it if it offers a better cost profile (ADR-0124).
//
// The engine is added with RoleActive by default. After re-planning, queries
// that are cheaper on the new engine are re-routed to it.
//
// If an EventLog is attached (via WithEventLog), call Backfill(ctx) afterward
// to replay events into the new engine's projections.
func (s *Store) AddEngine(ctx context.Context, engine Engine) error {
	if engine == nil {
		return errors.New("metaengine.Store.AddEngine: engine is nil")
	}

	name := engine.Profile().Name
	if name == "" {
		return errors.New("metaengine.Store.AddEngine: engine has empty Name in profile")
	}

	s.mu.Lock()
	for _, e := range s.engines {
		if e.Profile().Name == name {
			s.mu.Unlock()
			return fmt.Errorf("metaengine.Store.AddEngine: engine %q already registered", name)
		}
	}

	s.engines = append(s.engines, engine)
	s.mu.Unlock()

	return s.Replan(ctx)
}

// RemoveEngine deregisters an engine at runtime and triggers a re-plan.
// Queries routed to the removed engine are re-routed to the next-best engine.
// The engine is NOT closed — the caller owns its lifecycle.
func (s *Store) RemoveEngine(ctx context.Context, name string) error {
	s.mu.Lock()

	idx := -1
	for i, e := range s.engines {
		if e.Profile().Name == name {
			idx = i
			break
		}
	}

	if idx == -1 {
		s.mu.Unlock()
		return fmt.Errorf("metaengine.Store.RemoveEngine: engine %q not found", name)
	}

	s.engines = append(s.engines[:idx], s.engines[idx+1:]...)
	s.mu.Unlock()

	return s.Replan(ctx)
}

// Backfill replays all events from the attached EventLog into all projections.
// This is used after AddEngine to populate new projections on the added engine.
//
// Backfill detects non-idempotent fold types (Counter, Graph, Log, Multimap,
// Vector, Search, Spatial, and Map-update) and REFUSES to replay by default —
// replaying these would silently double-count or duplicate data. To override
// (e.g. when the target projections are known-empty), pass WithBackfillForce().
//
// If no EventLog is attached, Backfill returns nil without error.
func (s *Store) Backfill(ctx context.Context, opts ...BackfillOption) error {
	if s.eventLog == nil {
		return nil
	}

	events := s.eventLog.Events()
	if len(events) == 0 {
		return nil
	}

	cfg := backfillConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	return s.replayEvents(ctx, events, nil, cfg.force)
}

// replayEvents replays a slice of events through the fold pipeline WITHOUT
// recording them to the EventLog (they are already there). When queryFilter is
// non-nil, only folds for the named queries are applied. When force is false,
// the method refuses if any affected query has non-idempotent folds.
func (s *Store) replayEvents(
	ctx context.Context,
	events []EventInput,
	queryFilter map[string]bool,
	force bool,
) error {
	if !force {
		var nonIdem []string

		s.mu.RLock()
		for _, name := range sortedQueryNames(s.queries) {
			if queryFilter != nil && !queryFilter[name] {
				continue
			}

			for _, fold := range s.queries[name].QueryFolds() {
				if !isIdempotentFold(fold) {
					nonIdem = append(nonIdem,
						fmt.Sprintf("%s/%s(%s)", name, fold.EventType(), fold.Kind()))
				}
			}
		}
		s.mu.RUnlock()

		if len(nonIdem) > 0 {
			return fmt.Errorf(
				"metaengine: %d non-idempotent fold(s): %s — "+
					"replaying would double-count or duplicate data; "+
					"use force/WithBackfillForce() on empty projections",
				len(nonIdem),
				strings.Join(nonIdem, ", "),
			)
		}
	}

	for _, evt := range events {
		if err := s.applyReplay(ctx, evt.Type, evt.Payload, queryFilter); err != nil {
			return fmt.Errorf("metaengine: replay %s: %w", evt.Type, err)
		}
	}

	return nil
}

// foldTask pairs a query with one of its folds for batch dispatch.
type foldTask struct {
	q    queryMeta
	fold Fold
}

// dispatchFolds collects matching folds for eventType, groups them by engine,
// and applies them atomically per engine. When queryFilter is non-nil, only
// folds for named queries are dispatched. The caller must NOT hold s.mu.
func (s *Store) dispatchFolds(
	ctx context.Context,
	eventType string,
	rec record.Record,
	payload any,
	queryFilter map[string]bool,
) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	byEngine := make(map[Engine][]foldTask)

	for _, name := range sortedQueryNames(s.queries) {
		if queryFilter != nil && !queryFilter[name] {
			continue
		}

		q := s.queries[name]
		foldIdx, ok := q.QueryFoldByEvent()[eventType]
		if !ok {
			continue
		}

		fold := q.QueryFolds()[foldIdx]
		eng := q.QueryEngine()
		byEngine[eng] = append(byEngine[eng], foldTask{q: q, fold: fold})
	}

	for _, eng := range s.engines {
		tasks, ok := byEngine[eng]
		if !ok {
			continue
		}

		applyAll := func(ctx context.Context) error {
			for _, t := range tasks {
				s.foldMu.Lock()

				if ra, ok := t.fold.(RecordAwareFold); ok {
					ra.SetCurrentRecord(rec)
				}

				applyErr := s.applyFold(ctx, t.q, t.fold, payload)

				s.foldMu.Unlock()

				if applyErr != nil {
					return fmt.Errorf(
						"query %q fold for %s: %w",
						t.q.QueryName(),
						eventType,
						applyErr,
					)
				}
			}

			return nil
		}

		if tx, ok := eng.(Transactional); ok {
			if err := tx.RunInTx(ctx, applyAll); err != nil {
				return fmt.Errorf("batch apply on %s: %w", eng.Profile().Name, err)
			}
		} else {
			if err := applyAll(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// applyReplay dispatches an event through matching folds WITHOUT recording to
// the EventLog. When queryFilter is non-nil, only folds for the named queries
// are applied.
func (s *Store) applyReplay(
	ctx context.Context,
	eventType string,
	payload any,
	queryFilter map[string]bool,
) error {
	return s.dispatchFolds(ctx, eventType, record.Record{Type: eventType}, payload, queryFilter)
}

// BackfillOption configures Backfill behavior.
type BackfillOption func(*backfillConfig)

type backfillConfig struct {
	force bool
}

// WithBackfillForce skips the idempotency check. Use ONLY when the target
// projections are known-empty (e.g. a freshly added engine). Replaying events
// into non-empty projections with non-idempotent folds will corrupt data.
func WithBackfillForce() BackfillOption {
	return func(c *backfillConfig) { c.force = true }
}

// isIdempotentFold reports whether replaying this fold produces the same result
// as the first application. Insert/Remove/Set folds are idempotent (overwrite or
// no-op). Update/Count/Edge/Multi/Append/Vector/Search/Spatial folds are NOT
// idempotent (they accumulate or depend on previous state).
func isIdempotentFold(f Fold) bool {
	switch f.Kind() {
	case FoldInsert, FoldRemove, FoldSet, FoldSkip:
		return true
	default:
		return false
	}
}

// EngineNames returns the names of all registered engines.
func (s *Store) EngineNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.engines))
	for _, e := range s.engines {
		names = append(names, e.Profile().Name)
	}

	return names
}
