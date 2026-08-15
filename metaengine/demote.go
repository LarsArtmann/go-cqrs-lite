package metaengine

import (
	"context"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// DemoteOption configures DemoteEngine.
type DemoteOption func(*demoteConfig)

type demoteConfig struct {
	role  ProjectionRole
	force bool
}

// WithDemoteRole selects the shadow role the engine demotes to: RoleBackup
// (default) or RoleMigration.
func WithDemoteRole(role ProjectionRole) DemoteOption {
	return func(c *demoteConfig) { c.role = role }
}

// WithDemoteForce skips the non-idempotent guard for the re-routed queries'
// catch-up replay. The receiving projections are empty for those queries (they
// were served exclusively by the demoted engine), so the replay is safe; the
// guard exists because emptiness cannot be proven when engines previously
// mirrored those collections. Same contract as WithBackfillForce.
func WithDemoteForce() DemoteOption {
	return func(c *demoteConfig) { c.force = true }
}

// DemoteEngine transitions an Active/DualUse engine to a shadow role (Backup
// by default): the inverse of PromoteEngine (METAENGINE-LAYOUT-ROLES.md §4.4).
//
// Demotion is a drain-then-unroute transition executed atomically with the
// re-plan: under one write lock the role flips, a replicator takes over the
// engine's state, and every query it served is re-assigned to a remaining
// routable engine. From that instant the demoted engine mirrors ALL
// collections asynchronously and never serves reads.
//
// Because the demoted engine stops serving queries, DemoteEngine performs two
// targeted catch-up replays from the attached EventLog (required when the
// store has queries):
//
//   - the demoted engine receives the history of the collections it never
//     served, completing the mirror so a later PromoteEngine is safe;
//   - the queries it DID serve receive their history on their new engines,
//     which never folded those collections before.
//
// The second replay skips non-idempotent folds unless WithDemoteForce is
// passed. Like Backfill, prefer a quiet window: more than
// replicationBufferJobs live events during the mirror catch-up mark the
// engine stale (loud; recover via RemoveEngine + AddEngine + Backfill).
//
// Do NOT call Backfill after DemoteEngine: it replays full history into every
// shadow engine, which would re-apply the served collections the demoted
// engine already holds.
//
// Demotion refuses when it would leave no routable engine, when the engine's
// queries cannot be served by any remaining engine (missing ADT support), or
// when the required EventLog is absent.
func (s *Store) DemoteEngine(ctx context.Context, name string, opts ...DemoteOption) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("metaengine.DemoteEngine: %w", err)
	}

	cfg := demoteConfig{role: RoleBackup}
	for _, opt := range opts {
		opt(&cfg)
	}

	if !cfg.role.IsShadow() {
		return fmt.Errorf(
			"metaengine.DemoteEngine: target role %q is not a shadow role (use %s or %s)",
			cfg.role, RoleBackup, RoleMigration,
		)
	}

	served, missing, err := s.demotePreflight(name)
	if err != nil {
		return err
	}

	var events []EventInput

	if !cfg.force {
		if bad := nonIdempotentQueriesLocked(s, served); len(bad) > 0 {
			return fmt.Errorf(
				"metaengine.DemoteEngine: %d re-routed query(s) have non-idempotent folds (%s) "+
					"— pass WithDemoteForce to replay them onto their new engines",
				len(bad), strings.Join(bad, ", "),
			)
		}
	}

	err = s.replanWithTransition(ctx, triggerEngineDemote, func() error {
		role, known := s.roleByNameLocked(name)
		if !known || !role.routable() {
			return fmt.Errorf("engine %q is no longer demotable (role changed concurrently)", name)
		}

		for _, eng := range s.engines {
			if eng.Profile().Name == name {
				s.replicas[name] = s.newReplicatorLocked(eng)
			}
		}

		s.engineRoles[name] = cfg.role

		// Snapshot the log under the SAME write lock as the flip: because
		// applyWithRecord records + dispatches + replicates under one read
		// lock, this snapshot contains exactly the events dispatched to this
		// engine (for served queries) and exactly those the re-routed replay
		// must cover — no more (double-apply), no fewer (gaps).
		if s.eventLog != nil {
			events = s.eventLog.Events()
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("metaengine.DemoteEngine(%s): %w", name, err)
	}

	s.mu.RLock()
	rep := s.replicas[name]
	s.mu.RUnlock()

	if rep == nil {
		return fmt.Errorf(
			"metaengine.DemoteEngine(%s): replicator vanished during transition "+
				"(engine removed concurrently)", name,
		)
	}

	// The applier starts only after the mirror catch-up so history lands
	// before any buffered live event (chronological mirror state). Every exit
	// path starts it: stopAndDrain in PromoteEngine waits on its done channel.
	started := false

	start := func() {
		if !started {
			started = true
			go rep.run()
		}
	}

	defer start()

	if len(missing) > 0 && len(events) > 0 {
		if err := s.replayToShadow(ctx, rep, events, missing); err != nil {
			rep.failHalt(err)

			return fmt.Errorf(
				"metaengine.DemoteEngine(%s): demoted, but mirror catch-up failed: %w "+
					"— engine is stale; recover via RemoveEngine + AddEngine + Backfill",
				name, err,
			)
		}
	}

	if rep.isStale() {
		return fmt.Errorf(
			"metaengine.DemoteEngine(%s): replication buffer overflowed during catch-up "+
				"— engine is stale; recover via RemoveEngine + AddEngine + Backfill",
			name,
		)
	}

	if len(served) > 0 && len(events) > 0 {
		for _, evt := range events {
			if err := s.applyReplay(ctx, evt.Type, evt.Payload, served); err != nil {
				for qname := range served {
					s.poison.Poison(qname, err)
				}

				return fmt.Errorf(
					"metaengine.DemoteEngine(%s): demoted, but catch-up of the re-routed queries failed: %w "+
						"— the affected queries are poisoned until the store is recreated",
					name,
					err,
				)
			}
		}
	}

	return nil
}

// demotePreflight validates the demotion without mutating anything: the
// engine must exist and be routable, another routable engine must remain, and
// every query the engine serves must be servable by a remaining engine. It
// returns the served and never-served query sets and enforces the EventLog
// requirement.
func (s *Store) demotePreflight(name string) (served, missing map[string]bool, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	role, known := s.roleByNameLocked(name)
	if !known {
		return nil, nil, fmt.Errorf("metaengine.DemoteEngine: engine %q not found", name)
	}

	if !role.routable() {
		return nil, nil, fmt.Errorf(
			"metaengine.DemoteEngine: engine %q has role %s — only %s/%s engines can be demoted",
			name, role, RoleActive, RoleDualUse,
		)
	}

	var others []Engine

	for _, eng := range s.engines {
		ename := eng.Profile().Name
		if ename == name {
			continue
		}

		if r, ok := s.engineRoles[ename]; !ok || r.routable() {
			others = append(others, eng)
		}
	}

	if len(others) == 0 {
		return nil, nil, fmt.Errorf(
			"metaengine.DemoteEngine: %q is the only routable engine — "+
				"demoting it would leave no engine to serve queries", name,
		)
	}

	served = make(map[string]bool)
	missing = make(map[string]bool)

	for _, qname := range sortedQueryNames(s.queries) {
		if s.queries[qname].QueryEngine().Profile().Name == name {
			served[qname] = true
		} else {
			missing[qname] = true
		}
	}

	for _, qname := range sortedQueryNames(s.queries) {
		if !served[qname] {
			continue
		}

		if !anyEngineSupports(others, s.queries[qname].QueryADT()) {
			return nil, nil, fmt.Errorf(
				"metaengine.DemoteEngine: query %q requires an ADT that only %q supports — "+
					"add a supporting engine before demoting", qname, name,
			)
		}
	}

	if len(s.queries) > 0 && s.eventLog == nil {
		return nil, nil, fmt.Errorf(
			"metaengine.DemoteEngine: an EventLog is required to catch up the demoted engine " +
				"and the re-routed queries — attach one via WithEventLog",
		)
	}

	return served, missing, nil
}

// anyEngineSupports reports whether any engine can serve the ADT (natively or
// degraded), matching planQuery's candidate rule.
func anyEngineSupports(engines []Engine, adt ADT) bool {
	for _, eng := range engines {
		if _, ok := eng.Profile().SupportsADT(adt); ok {
			return true
		}
	}

	return false
}

// nonIdempotentQueriesLocked returns the sorted names of queries (within the
// filter) that own at least one non-idempotent fold.
func nonIdempotentQueriesLocked(s *Store, filter map[string]bool) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var bad []string

	for _, qname := range sortedQueryNames(s.queries) {
		if filter != nil && !filter[qname] {
			continue
		}

		for _, fold := range s.queries[qname].QueryFolds() {
			if !isIdempotentFold(fold) {
				bad = append(bad, qname)

				break
			}
		}
	}

	return bad
}

// replayToShadow synchronously replays filtered history into one shadow
// engine, mirroring the replicator's apply semantics (per-fold locks, engine
// transaction when supported). Unlike Backfill it does not touch primary
// engines or the event log.
func (s *Store) replayToShadow(
	ctx context.Context,
	rep *replicator,
	events []EventInput,
	queryFilter map[string]bool,
) error {
	for _, evt := range events {
		job := repJob{
			eventType: evt.Type,
			rec:       record.Record{Type: evt.Type},
			payload:   evt.Payload,
		}

		if err := rep.applyJobFilter(ctx, job, queryFilter); err != nil {
			return fmt.Errorf("replay %s into %s: %w", evt.Type, rep.name, err)
		}
	}

	return nil
}
