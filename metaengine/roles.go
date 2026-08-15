package metaengine

import (
	"context"
	"fmt"
)

// Valid reports whether r is one of the four defined projection roles.
func (r ProjectionRole) Valid() bool {
	switch r {
	case RoleActive, RoleDualUse, RoleMigration, RoleBackup:
		return true
	default:
		return false
	}
}

// IsShadow reports whether engines with this role mirror all collections via
// async replication instead of serving reads (Migration, Backup).
func (r ProjectionRole) IsShadow() bool {
	return r == RoleMigration || r == RoleBackup
}

// routable reports whether engines with this role may serve query reads
// (Active, DualUse).
func (r ProjectionRole) routable() bool {
	return r == RoleActive || r == RoleDualUse
}

// AddEngineOption configures AddEngine.
type AddEngineOption func(*addEngineConfig)

type addEngineConfig struct {
	role ProjectionRole
}

// WithEngineRole assigns the added engine a projection role (ADR-0124 §7,
// METAENGINE-LAYOUT-ROLES.md). Defaults to RoleActive.
//
// RoleActive and RoleDualUse engines are routable and synced synchronously
// through the fold pipeline. RoleMigration and RoleBackup engines are shadow
// engines: they mirror ALL collections via async replication (eventual
// consistency, failure-isolated) and never serve reads until promoted via
// PromoteEngine.
func WithEngineRole(role ProjectionRole) AddEngineOption {
	return func(c *addEngineConfig) { c.role = role }
}

// EngineRole returns the current projection role of a registered engine.
// The second return is false when no engine with that name is registered.
func (s *Store) EngineRole(name string) (ProjectionRole, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.roleByNameLocked(name)
}

// roleByNameLocked resolves a role by engine name. Missing entries mean the
// engine was present at Plan() time and defaults to RoleActive. The caller
// must hold the store lock (read or write).
func (s *Store) roleByNameLocked(name string) (ProjectionRole, bool) {
	for _, eng := range s.engines {
		if eng.Profile().Name == name {
			if role, ok := s.engineRoles[name]; ok {
				return role, true
			}

			return RoleActive, true
		}
	}

	return "", false
}

// routableLocked returns the engines allowed to serve reads. The caller must
// hold the store lock. Shadow engines (Backup/Migration) are excluded — a
// lagging mirror must never be observed by a query.
func (s *Store) routableLocked() []Engine {
	out := make([]Engine, 0, len(s.engines))

	for _, eng := range s.engines {
		name := eng.Profile().Name

		if role, ok := s.engineRoles[name]; !ok || role.routable() {
			out = append(out, eng)
		}
	}

	return out
}

// ReplicationStatus describes a shadow engine's replication health.
type ReplicationStatus struct {
	// Role is the engine's current projection role.
	Role ProjectionRole

	// Queued is the number of accepted jobs waiting to be applied. An
	// approximate, point-in-time value.
	Queued int

	// Applied is the total number of replication jobs applied to the engine
	// (including synchronous backfill replay).
	Applied int64

	// Stale is true when replication halted: permanent failure, buffer
	// overflow, or shutdown. A stale mirror is behind and must be recovered
	// (remove, fix, re-add, backfill) before promotion.
	Stale bool

	// LastError describes why replication halted, if it did.
	LastError string
}

// ReplicationStatus returns the replication health of a shadow engine. The
// second return is false when the engine is not a shadow engine.
func (s *Store) ReplicationStatus(name string) (ReplicationStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rep, ok := s.replicas[name]
	if !ok {
		return ReplicationStatus{}, false
	}

	return ReplicationStatus{
		Role:     s.roleOfNameLocked(name),
		Queued:   rep.queued(),
		Applied:  rep.appliedCount(),
		Stale:    rep.isStale(),
		LastError: rep.lastError(),
	}, true
}

func (s *Store) roleOfNameLocked(name string) ProjectionRole {
	if role, ok := s.engineRoles[name]; ok {
		return role
	}

	return RoleActive
}

// PromoteEngine transitions a shadow engine (Backup or Migration) to RoleActive
// (ADR-0124 §7 cutover/promote, METAENGINE-LAYOUT-ROLES.md §4).
//
// Promotion drains the replication backlog first — while briefly holding the
// store write lock, which blocks concurrent applies — so the promoted engine is
// fully caught up at the instant it becomes routable. After the role flip, the
// store re-plans with trigger "engine-promoted" so queries can be routed to
// the promoted engine.
//
// Promoting a stale engine fails: recover it first (remove, fix, re-add,
// backfill). The context bounds the drain wait.
func (s *Store) PromoteEngine(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("metaengine.PromoteEngine: %w", err)
	}

	s.mu.Lock()

	role, known := s.roleByNameLocked(name)
	if !known {
		s.mu.Unlock()
		return fmt.Errorf("metaengine.PromoteEngine: engine %q not found", name)
	}

	if !role.IsShadow() {
		s.mu.Unlock()
		return fmt.Errorf(
			"metaengine.PromoteEngine: engine %q has role %s — only %s/%s engines can be promoted",
			name, role, RoleMigration, RoleBackup,
		)
	}

	rep, ok := s.replicas[name]
	if ok {
		if err := rep.stopAndDrain(ctx); err != nil {
			s.mu.Unlock()
			return fmt.Errorf("metaengine.PromoteEngine(%s): %w", name, err)
		}
	}

	delete(s.replicas, name)
	s.engineRoles[name] = RoleActive
	s.mu.Unlock()

	return s.replanWithTrigger(ctx, triggerEnginePromote)
}
