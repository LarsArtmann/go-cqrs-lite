package metaengine

import (
	"context"
	"fmt"
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
	Engine   Engine
	Role     ProjectionRole
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
		return fmt.Errorf("metaengine.Store.AddEngine: engine is nil")
	}

	name := engine.Profile().Name
	if name == "" {
		return fmt.Errorf("metaengine.Store.AddEngine: engine has empty Name in profile")
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
// WARNING: Backfill replays ALL events, which means projections that already
// have data will receive duplicate events. For insert/remove folds (Map ADT),
// this is safe (idempotent — same key overwrites). For counter/set folds, it
// is NOT safe (double-counting). Use Backfill only on fresh stores or clear
// projections first.
//
// If no EventLog is attached, Backfill returns nil without error.
func (s *Store) Backfill(ctx context.Context) error {
	if s.eventLog == nil {
		return nil
	}

	events := s.eventLog.Events()
	if len(events) == 0 {
		return nil
	}

	for _, evt := range events {
		if err := s.Apply(ctx, evt.Type, evt.Payload); err != nil {
			return fmt.Errorf("metaengine.Store.Backfill: replay %s: %w", evt.Type, err)
		}
	}

	return nil
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
