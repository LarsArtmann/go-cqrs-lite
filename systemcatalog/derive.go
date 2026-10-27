package systemcatalog

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ErrCatalogValidation mirrors catalog.simple's validation sentinel: a
// derived catalog that fails validation is a programming error in the
// derivation or the overrides, surfaced with the violations attached.
var ErrCatalogValidation error = errorfamily.NewRejection(
	"systemcatalog.validation_failed",
	"systemcatalog: derived catalog failed validation",
)

// FromDomainConfig derives a catalog by booting an EPHEMERAL in-memory
// system from domain (the same recipe systemscenario.Memory uses), letting
// system.New run the Commands/Queries registration closures, reading the
// resulting registrations, and closing the system. Nothing is started and
// nothing persists.
//
// Use it in tests, CI drift gates, and export commands. Boot failures
// (including DomainConfig coeffect-gate errors) are returned honestly.
func FromDomainConfig(
	ctx context.Context,
	domain system.DomainConfig,
	opts ...Option,
) (*catalog.Catalog, error) {
	sys, err := system.New(ctx, domain, memoryDeployment())
	if err != nil {
		return nil, fmt.Errorf("systemcatalog: boot ephemeral system: %w", err)
	}

	derived, deriveErr := FromSystem(sys, domain, opts...)

	closeErr := sys.Close()

	if deriveErr != nil {
		return nil, deriveErr
	}

	if closeErr != nil {
		return nil, fmt.Errorf("systemcatalog: close ephemeral system: %w", closeErr)
	}

	return derived, nil
}

// FromSystem derives a catalog from an ALREADY-BOOTED system plus the
// DomainConfig that built it. Commands and queries come from the system's
// registration records; events come from the config's declared universe.
func FromSystem(
	sys *system.System,
	domain system.DomainConfig,
	opts ...Option,
) (*catalog.Catalog, error) {
	cfg := resolveOptions(opts)
	builder := catalog.NewBuilder(cfg.title, cfg.version)

	if err := DeriveInto(builder, sys, domain, opts...); err != nil {
		return nil, err
	}

	cat := builder.Build()

	if violations := cat.Validate(); len(violations) > 0 {
		return cat, fmt.Errorf("%w: %v", ErrCatalogValidation, violations)
	}

	return cat, nil
}

// DeriveInto registers the derived message inventory as ONE service on an
// existing builder: commands and queries (with reflected payload schemas)
// as received messages, the declared event universe as sent messages with
// Schema versions stamped. It is the curated-merge path — anything the
// consumer adds to the builder (users, flows, data stores, domains) is
// untouched by derivation.
func DeriveInto(
	b *catalog.Builder,
	sys *system.System,
	domain system.DomainConfig,
	opts ...Option,
) error {
	if b == nil {
		return errorfamily.NewRejection(
			"systemcatalog.nil_builder",
			"derive requires a non-nil catalog builder",
		)
	}

	if sys == nil {
		return errorfamily.NewRejection(
			"systemcatalog.nil_system",
			"derive requires a non-nil booted system (see FromDomainConfig for the ephemeral boot)",
		)
	}

	cfg := resolveOptions(opts)

	b.AddService(
		cfg.serviceID,
		cfg.name,
		cfg.version,
		cfg.summary,
		derivedMessages(sys, domain, cfg.overrides)...,
	)

	if len(cfg.owners) > 0 {
		b.ConfigureService(cfg.serviceID, catalog.ServiceOwners(cfg.owners...))
	}

	if cfg.curator != nil {
		cfg.curator(b)
	}

	return nil
}

// derivedMessages collects the full message inventory: registered commands
// (sorted), registered queries (sorted), then the declared event universe
// (sorted) with Schema versions stamped where declared. A message whose id
// has a curated override is emitted as the override instead (curated wins,
// derivation fills the rest).
func derivedMessages(
	sys *system.System,
	domain system.DomainConfig,
	overrides map[string]catalog.MessageConfig,
) []catalog.MessageConfig {
	msgs := make([]catalog.MessageConfig, 0,
		len(domain.Events)+len(domain.Schema)+len(overrides))

	for _, reg := range sys.CommandRegistrations() {
		id := string(reg.Type)

		if override, curated := overrides[id]; curated {
			msgs = append(msgs, override)

			continue
		}

		msgs = append(msgs, catalog.CommandFromSchema(
			catalog.MessageID(id),
			schemaForType(reg.PayloadType),
		))
	}

	for _, reg := range sys.QueryRegistrations() {
		id := string(reg.Type)

		if override, curated := overrides[id]; curated {
			msgs = append(msgs, override)

			continue
		}

		msgs = append(msgs, catalog.QueryFromSchema(
			catalog.MessageID(id),
			schemaForType(reg.PayloadType),
		))
	}

	schemaVersions := map[event.Type]string{}
	for _, declared := range domain.Schema {
		schemaVersions[declared.Type()] = declared.CurrentVersion().String()
	}

	for _, evtType := range eventUniverse(domain) {
		id := string(evtType)

		if override, curated := overrides[id]; curated {
			msgs = append(msgs, override)

			continue
		}

		version, declared := schemaVersions[evtType]

		var opts []catalog.MessageOption
		if declared {
			opts = append(opts, catalog.WithVersion(version))
		}

		msgs = append(msgs, catalog.EventFromSchema(
			catalog.MessageID(id),
			catalog.Sends,
			nil,
			opts...,
		))
	}

	return msgs
}

// eventUniverse returns the sorted union of the explicitly declared event
// types and the types declared by Schema (Schema extends Events, it never
// sits beside it as a second registry).
func eventUniverse(domain system.DomainConfig) []event.Type {
	seen := make(map[event.Type]struct{}, len(domain.Events)+len(domain.Schema))
	universe := make([]event.Type, 0, len(domain.Events)+len(domain.Schema))

	for _, evtType := range domain.Events {
		if _, dup := seen[evtType]; dup {
			continue
		}

		seen[evtType] = struct{}{}
		universe = append(universe, evtType)
	}

	for _, declared := range domain.Schema {
		evtType := declared.Type()
		if _, dup := seen[evtType]; dup {
			continue
		}

		seen[evtType] = struct{}{}
		universe = append(universe, evtType) //art-dupl:accept sorted-dedup-union idiom; shape-twin is cqrs-lint's key collector, not shared logic
	}

	slices.Sort(universe)

	return universe
}

// schemaForType reflects a payload schema, cloning the shared cached schema
// so message options can safely mutate it. Returns nil for unknown types —
// identity without payload shape is valid.
func schemaForType(t reflect.Type) *catalog.Schema {
	if t == nil {
		return nil
	}

	sch := catalog.SchemaFromReflect(t)
	if sch == nil {
		return nil
	}

	return sch.Clone()
}

// memoryDeployment mirrors systemscenario.Memory: two memory engines
// (primary + timers) so DomainConfig.Timers closures resolve out of the box.
func memoryDeployment() system.DeploymentConfig {
	return system.DeploymentConfig{ //nolint:exhaustruct_v5 // minimal ephemeral boot; the rest is operator concern
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"}, //nolint:exhaustruct_v5 // memory needs no DSN
			"timers":  {Driver: "memory"}, //nolint:exhaustruct_v5 // memory needs no DSN
		},
		Instances: []system.InstanceConfig{
			{ //nolint:exhaustruct_v5 // role+engine is the minimal instance
				Role:   system.RoleSourceOfTruth,
				Engine: "primary",
			},
			{ //nolint:exhaustruct_v5 // role+engine is the minimal instance
				Role:   system.RoleProjections,
				Engine: "primary",
			},
		},
	}
}
