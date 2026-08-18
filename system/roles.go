package system

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// ─── Dedicated role wiring (RoleCommands / RoleQueries / RoleSnapshots) ───

// resolveDedicatedRoles collects dedicated role instances. They take
// precedence over the source-of-truth instance for their store. Duplicate
// declarations of one role fail construction: the System holds exactly one
// store per role.
func resolveDedicatedRoles(
	deployment DeploymentConfig,
) (map[InstanceRole]InstanceConfig, error) {
	dedicated := make(map[InstanceRole]InstanceConfig)

	for _, inst := range deployment.Instances {
		switch inst.Role {
		case RoleCommands, RoleQueries, RoleSnapshots:
			if _, dup := dedicated[inst.Role]; dup {
				return nil, fmt.Errorf("%w: %q", ErrDuplicateInstanceRole, inst.Role)
			}

			dedicated[inst.Role] = inst
		}
	}

	return dedicated, nil
}

// wireDedicatedRoles binds the command/query/snapshot stores declared as
// dedicated instances. One engine may serve multiple roles — collections are
// namespaced ("commands", "queries", "snapshots"), so role separation does
// not require engine separation.
func wireDedicatedRoles(
	sys *System,
	deployment DeploymentConfig,
	dedicated map[InstanceRole]InstanceConfig,
	engineCache map[string]metaengine.Engine,
) error {
	for _, role := range []InstanceRole{RoleCommands, RoleQueries, RoleSnapshots} {
		inst, declared := dedicated[role]
		if !declared {
			continue
		}

		eng, engineName, err := resolveInstanceEngine(inst, engineCache)
		if err != nil {
			return err
		}

		serialize := engineNeedsSerialization(deployment, engineName)

		switch role {
		case RoleCommands:
			store, err := buildCommandStore(eng, serialize)
			if err != nil {
				return fmt.Errorf("system: instance %q: %w", inst.Role, err)
			}

			sys.cmdStore = store
		case RoleQueries:
			store, err := buildQueryStore(eng, serialize)
			if err != nil {
				return fmt.Errorf("system: instance %q: %w", inst.Role, err)
			}

			sys.queryStore = store
		case RoleSnapshots:
			store, err := buildSnapshotStore(eng)
			if err != nil {
				return fmt.Errorf("system: instance %q: %w", inst.Role, err)
			}

			sys.snapStore = store
		}
	}

	return nil
}

// wireSourceOfTruth wires the event store plus (when no dedicated instance
// claims them) the command, query, and snapshot stores on the source-of-truth
// instance's engine.
func wireSourceOfTruth(
	sys *System,
	deployment DeploymentConfig,
	inst InstanceConfig,
	engineCache map[string]metaengine.Engine,
	dedicated map[InstanceRole]InstanceConfig,
) error {
	eng, engineName, err := resolveInstanceEngine(inst, engineCache)
	if err != nil {
		return err
	}

	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		return fmt.Errorf("%w: engine %q", ErrNotStreamLogBackend, engineName)
	}

	// Auto-detect serialization: Memory stores pointers directly; all
	// other engines need JSON envelope serialization.
	serialize := engineNeedsSerialization(deployment, engineName)

	var adapterOpts []EventAdapterOption
	if serialize {
		adapterOpts = append(adapterOpts, WithSerialization())
	}

	sys.eventStore = NewEventAdapter(backend, "events", adapterOpts...)

	if _, has := dedicated[RoleSnapshots]; !has {
		if store, err := buildSnapshotStore(eng); err == nil {
			sys.snapStore = store
		}
	}

	if _, has := dedicated[RoleCommands]; !has && inst.Role == RoleSourceOfTruth {
		store, err := buildCommandStore(eng, serialize)
		if err != nil {
			return fmt.Errorf("system: instance %q: %w", inst.Role, err)
		}

		sys.cmdStore = store
	}

	if _, has := dedicated[RoleQueries]; !has && inst.Role == RoleSourceOfTruth {
		store, err := buildQueryStore(eng, serialize)
		if err != nil {
			return fmt.Errorf("system: instance %q: %w", inst.Role, err)
		}

		sys.queryStore = store
	}

	// Wire cache tier if configured.
	if inst.Cache != nil && inst.Cache.Capacity > 0 {
		cached, err := NewCachedEventStore(sys.eventStore, inst.Cache.Capacity)
		if err != nil {
			return fmt.Errorf("system: create cache: %w", err)
		}

		sys.eventStore = cached
	}

	return nil
}

// instanceEngineName resolves the engine an instance references: the single
// Engine field, or the first of a mixed pool.
func instanceEngineName(inst InstanceConfig) string {
	if inst.Engine == "" && len(inst.Engines) > 0 {
		return inst.Engines[0]
	}

	return inst.Engine
}

// resolveInstanceEngine looks up the engine an instance references.
func resolveInstanceEngine(
	inst InstanceConfig, engineCache map[string]metaengine.Engine,
) (metaengine.Engine, string, error) {
	engineName := instanceEngineName(inst)

	eng, ok := engineCache[engineName]
	if !ok {
		return nil, engineName, fmt.Errorf(
			"%w: instance %q references engine %q", ErrUnknownEngine, inst.Role, engineName,
		)
	}

	return eng, engineName, nil
}

// engineNeedsSerialization reports whether the named engine needs JSON
// envelope serialization: Memory stores pointers directly, every other
// driver serializes.
func engineNeedsSerialization(deployment DeploymentConfig, engineName string) bool {
	engCfg, hasCfg := deployment.Engines[engineName]

	return hasCfg && engCfg.Driver != "memory"
}

// buildCommandStore adapts an engine's StreamLogBackend as the command audit
// store on the "commands" collection.
func buildCommandStore(
	eng metaengine.Engine, serialize bool,
) (command.Store, error) {
	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		return nil, fmt.Errorf("%w: engine %q", ErrNotStreamLogBackend, eng.Profile().Name)
	}

	var opts []CommandAdapterOption
	if serialize {
		opts = append(opts, WithCommandSerialization())
	}

	return NewCommandAdapter(backend, "commands", opts...), nil
}

// buildQueryStore adapts an engine's StreamLogBackend as the query audit
// store on the "queries" collection.
func buildQueryStore(
	eng metaengine.Engine, serialize bool,
) (query.QueryStore, error) {
	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		return nil, fmt.Errorf("%w: engine %q", ErrNotStreamLogBackend, eng.Profile().Name)
	}

	var opts []QueryAdapterOption
	if serialize {
		opts = append(opts, WithQuerySerialization())
	}

	return NewQueryAdapter(backend, "queries", opts...), nil
}

// buildSnapshotStore adapts an engine's SnapshotBackend as the snapshot store
// on the "snapshots" collection. Engines without SnapshotBackend are not an
// error here: the source-of-truth path simply leaves the store unset.
func buildSnapshotStore(eng metaengine.Engine) (snapshot.SnapshotStore, error) {
	snapBackend, ok := eng.(metaengine.SnapshotBackend)
	if !ok {
		return nil, fmt.Errorf("%w: engine %q", ErrNotSnapshotBackend, eng.Profile().Name)
	}

	return NewSnapshotAdapter(snapBackend, "snapshots"), nil
}
