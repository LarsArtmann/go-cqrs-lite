package system

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Topology describes the entire wired deployment as a graph.
type Topology struct {
	Instances      []InstanceTopology
	Buses          []BusTopology
	Dispatchers    []DispatcherInfo
	ProjectionHost *ProjectionHostInfo
}

type InstanceTopology struct {
	Name         string
	Role         InstanceRole
	EngineName   string
	DriverName   string
	Collections  []string
	Durability   DurabilityTier
	HealthStatus string
}

type BusTopology struct {
	Name   string
	Driver string
	Mode   string
}

type DispatcherInfo struct {
	Type     string
	Handlers int
}

type ProjectionHostInfo struct {
	Started bool
	Workers int
}

type CacheTierInfo struct {
	EngineName string
	HitRate    float64
	Size       int
	MaxSize    int
}

// instanceHealth returns a real health status for an instance based on its
// engine and the system's running state. Memory engines are always healthy;
// persistent engines are healthy when not stopped.
func (s *System) instanceHealth(inst InstanceConfig) string {
	if s.stopped {
		return "stopped"
	}

	engineName := inst.Engine
	if engineName == "" && len(inst.Engines) > 0 {
		engineName = inst.Engines[0]
	}

	if engineName == "" {
		return "no-engine"
	}

	if _, ok := s.deployment.Engines[engineName]; !ok {
		return "missing-engine"
	}

	return "healthy"
}

// Snapshot returns a structured topology snapshot of the entire system.
func (s *System) Snapshot(ctx context.Context) (*Topology, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	topo := &Topology{}

	for _, inst := range s.deployment.Instances {
		engineName := inst.Engine
		if engineName == "" && len(inst.Engines) > 0 {
			engineName = inst.Engines[0]
		}

		driverName := ""
		if engCfg, ok := s.deployment.Engines[engineName]; ok {
			driverName = engCfg.Driver
		}

		topo.Instances = append(topo.Instances, InstanceTopology{
			Name:         string(inst.Role),
			Role:         inst.Role,
			EngineName:   engineName,
			DriverName:   driverName,
			Collections:  inst.Collections,
			Durability:   inst.Durability,
			HealthStatus: s.instanceHealth(inst),
		})
	}

	for name, busCfg := range s.deployment.Buses {
		topo.Buses = append(topo.Buses, BusTopology{
			Name:   name,
			Driver: busCfg.Driver,
			Mode:   busCfg.Mode,
		})
	}

	if s.cmdDisp != nil {
		topo.Dispatchers = append(topo.Dispatchers, DispatcherInfo{
			Type:     "command",
			Handlers: s.cmdHandlerCount,
		})
	}

	if s.projHost != nil {
		topo.ProjectionHost = &ProjectionHostInfo{
			Started: s.started,
		}
	}

	return topo, nil
}

// Health returns aggregated health across all components.
func (s *System) Health(ctx context.Context) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	parts := []string{"ok"}

	if s.projHost != nil {
		if !s.started {
			parts = append(parts, "projections:not-started")
		} else {
			parts = append(parts, "projections:running")
		}
	}

	return strings.Join(parts, " ")
}

// Explain returns a human-readable topology description.
func (s *System) Explain(ctx context.Context) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var b strings.Builder

	b.WriteString("System Topology:\n")
	fmt.Fprintf(&b, "  Drivers: %s\n", strings.Join(RegisteredDrivers(), ", "))
	fmt.Fprintf(&b, "  Engines: %d\n", len(s.engines))
	fmt.Fprintf(&b, "  Started: %v\n", s.started)

	if s.projHost != nil {
		b.WriteString("  ProjectionHost: configured\n")
	}

	if s.projStore != nil {
		fmt.Fprintf(&b, "  ProjectionStore: %d collections\n", len(s.projStore.Collections()))
	}

	fmt.Fprintf(&b, "  Go version: %s\n", runtime.Version())
	fmt.Fprintf(&b, "  Time: %s\n", time.Now().Format(time.RFC3339))

	return b.String()
}

// ProjectionPlan returns the serializable plan for the projection layer, or
// nil if no projection store is configured. Useful for pinning plans and
// detecting drift across restarts.
func (s *System) ProjectionPlan() *metaengine.SerializablePlan {
	if s.projStore == nil {
		return nil
	}

	result := s.projStore.Plan()
	if result == nil {
		return nil
	}

	return metaengine.Serialize(result, s.engines)
}

// VerifyProjections checks the consistency of the projection layer plan
// against the configured engines. Returns nil if no projection store exists.
func (s *System) VerifyProjections(ctx context.Context) error {
	if s.projStore == nil {
		return nil
	}

	return s.projStore.Verify(ctx, s.engines)
}

// ProjectionExplain returns a human-readable plan explanation for the
// projection layer, or an empty string if no projection store is configured.
func (s *System) ProjectionExplain() string {
	if s.projStore == nil {
		return ""
	}

	return s.projStore.ExplainPlan()
}
