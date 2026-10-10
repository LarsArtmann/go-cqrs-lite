package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4" // register the "dgraph" driver (operator opt-in)
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // register the "sqlite" driver
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// The composition root. DomainConfig is everything a developer declares;
// DeploymentConfig is everything an operator decides. buildSystem is the ONLY
// seam between them.

// buildSystem wires the composition root from a domain and a deployment.
func buildSystem(ctx context.Context, deployment system.DeploymentConfig) (*system.System, error) {
	return system.New(ctx, graphDomain(), deployment)
}

// graphDomain is the developer half: everything declared, nothing wired.
// Shared by the binary and the systemscenario BDD suite — the SAME config
// the production process boots.
func graphDomain() system.DomainConfig {
	projection, typeDecoder := followGraphProjection()

	return system.DomainConfig{
		Commands:              registerCommands,
		Projections:           projection,
		ProjectionTypeDecoder: typeDecoder,
		// NOTE: no Events universe here. The coeffect gate cannot see through
		// system.RawQuery declarations (they are opaque to it), so declaring
		// Events would only emit unconsumed-event advisories for exactly the
		// types these folds consume. Evolve/Lookup declarations DO feed the
		// gate — declare Events when using those.
	}
}

// registerCommands declares the decider and both commands. The coeffect gate
// (Events universe above) verifies at composition time that every subscribed
// event type is actually produced by this bounded context.
func registerCommands(sys *system.System) {
	err := system.RegisterDecider(sys, streamType, decider.Decider[FollowState]{
		Initial: FollowState{},
		Apply:   applyFollowState,
	})
	if err != nil {
		panic(err)
	}

	registerFollowCommands(sys)
	registerUnfollowCommand(sys)
}

func registerFollowCommands(sys *system.System) {
	err := system.RegisterCommand[FollowCmd, FollowState](sys, cmdFollow,
		func(cmdCtx context.Context, cmd FollowCmd) system.Op[FollowState] {
			return system.Execute(cmdCtx, cmd.StreamID(), streamType,
				follow(cmd.StreamID(), cmd.Followee))
		})
	if err != nil {
		panic(err)
	}
}

func registerUnfollowCommand(sys *system.System) {
	err := system.RegisterCommand[UnfollowCmd, FollowState](sys, cmdUnfollow,
		func(cmdCtx context.Context, cmd UnfollowCmd) system.Op[FollowState] {
			return system.Execute(cmdCtx, cmd.StreamID(), streamType,
				unfollow(cmd.StreamID(), cmd.Followee))
		})
	if err != nil {
		panic(err)
	}
}

// deploymentOptions carries the operator's decisions from main.
type deploymentOptions struct {
	sqliteDSN  string // source-of-truth engine DSN ("" = ephemeral in-memory)
	dgraphAddr string // projections engine gRPC address ("" = sqlite projections)
	configPath string // cqrs.yaml via system.LoadConfig (overrides the above)
}

// buildDeployment decides where data lives — pure operator territory.
//
//	default:              sqlite serves both roles
//	dgraphAddr non-empty: sqlite stays source of truth, Dgraph runs
//	                      projections (native @recurse traversal)
//	configPath non-empty: everything comes from cqrs.yaml / CQRS_* env
func buildDeployment(opts deploymentOptions) (system.DeploymentConfig, error) {
	if opts.configPath != "" {
		deployment, err := system.LoadConfig(opts.configPath)
		if err != nil {
			return system.DeploymentConfig{}, fmt.Errorf("load config: %w", err)
		}

		return deployment, nil
	}

	dsn := opts.sqliteDSN
	if dsn == "" {
		dsn = "file:graph-native?mode=memory&cache=shared"
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "sqlite", DSN: dsn},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}

	if opts.dgraphAddr != "" {
		deployment.Engines["graph"] = system.EngineConfig{
			Driver: "dgraph",
			DSN:    opts.dgraphAddr, // plain gRPC address, e.g. "localhost:9080"
		}
		deployment.Instances[1] = system.InstanceConfig{
			Role: system.RoleProjections, Engine: "graph",
		}
	}

	return deployment, nil
}
