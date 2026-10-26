package systemcatalog

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ─── Fixture domain: one command, one query, two events (one versioned) ───

func mustStreamID(s string) id.StreamID {
	sid, err := id.ParseStreamID(s)
	if err != nil {
		panic(err)
	}

	return sid
}

var (
	fixtureStream = mustStreamID("task-1")
	fixtureCmdID  = id.NewCommandID()
)

type createTaskCmd struct {
	Title string `json:"title"`
}

func (c createTaskCmd) Type() command.Type    { return "task.create" }
func (c createTaskCmd) StreamID() id.StreamID { return fixtureStream }
func (c createTaskCmd) ID() id.CommandID       { return fixtureCmdID }

type listTasksQuery struct {
	Owner string `json:"owner"`
}

const (
	evtCreated   = event.Type("task.created")
	evtCompleted = event.Type("task.completed")
)

func fixtureDomain() system.DomainConfig {
	return system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", decider.Decider[string]{
				Initial: "",
				Apply: func(state string, evt event.Event) (string, error) {
					return state, nil
				},
			})

				system.RegisterCommand[createTaskCmd, string](sys, "task.create",
					func(ctx context.Context, cmd createTaskCmd) system.Op[string] {
					return system.Execute(ctx, fixtureStream, "Task",
						func(_ string, ver event.Version) ([]event.Event, error) {
							evt, err := event.New(evtCreated, fixtureStream, "Task", ver+1,
								struct{ Title string }{Title: cmd.Title})
							if err != nil {
								return nil, err
							}

							return []event.Event{evt}, nil
						})
				})
		},
		Queries: func(sys *system.System) {
			system.RegisterQuery[listTasksQuery, []string](sys, "task.list",
				func(ctx context.Context, q listTasksQuery) ([]string, error) {
					return nil, nil
				})
		},
		Events: []event.Type{evtCreated, evtCompleted},
		Schema: []schema.EventSchema{
			schema.Event(evtCreated, 2),
		},
	}
}

// ─── FromDomainConfig ───

func TestFromDomainConfig(t *testing.T) {
	cat, err := FromDomainConfig(context.Background(), fixtureDomain(),
		WithTitle("Task Service"),
		WithServiceSummary("one command, one query, two events"),
	)
	if err != nil {
		t.Fatalf("FromDomainConfig: %v", err)
	}

	if cat.Title != "Task Service" {
		t.Errorf("title = %q, want %q", cat.Title, "Task Service")
	}

	if len(cat.Services) != 1 {
		t.Fatalf("services = %d, want 1", len(cat.Services))
	}

	svc := cat.Services[0]
	if svc.ID != "task-service" {
		t.Errorf("service id = %q, want %q", svc.ID, "task-service")
	}

	if len(svc.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(svc.Commands))
	}

	cmd := svc.Commands[0]
	if string(cmd.ID) != "task.create" {
		t.Errorf("command id = %q, want task.create", cmd.ID)
	}

	if cmd.Name != "Task Create" {
		t.Errorf("command name = %q, want %q", cmd.Name, "Task Create")
	}

	if cmd.Schema == nil {
		t.Fatal("command schema missing: payload type was registered, schema must be reflected")
	}

	titleProp, ok := cmd.Schema.Properties["title"]
	if !ok || titleProp.Type != "string" {
		t.Errorf("command schema properties[title] = %+v (ok=%v), want string", titleProp, ok)
	}

	if len(svc.Queries) != 1 {
		t.Fatalf("queries = %d, want 1", len(svc.Queries))
	}

	qry := svc.Queries[0]
	if string(qry.ID) != "task.list" || qry.Schema == nil {
		t.Errorf("query = %q (schema=%v), want task.list with schema", qry.ID, qry.Schema)
	}

	if len(svc.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(svc.Events))
	}

	if got, want := string(svc.Events[0].ID), "task.completed"; got != want {
		t.Errorf("events must be sorted: first = %q, want %q", got, want)
	}

	if got, want := string(svc.Events[1].ID), "task.created"; got != want {
		t.Errorf("second event = %q, want %q", got, want)
	}

	if got, want := string(svc.Events[1].Version), "2"; got != want {
		t.Errorf("declared schema version not stamped: = %q, want %q", got, want)
	}

	if got, want := string(svc.Events[0].Version), "1.0.0"; got != want {
		t.Errorf("undeclared event version = %q, want default %q", got, want)
	}

	if violations := cat.Validate(); len(violations) > 0 {
		t.Errorf("derived catalog invalid: %v", violations)
	}
}

func TestFromDomainConfig_ProducersConsumers(t *testing.T) {
	cat, err := FromDomainConfig(context.Background(), fixtureDomain(), WithTitle("Task Service"))
	if err != nil {
		t.Fatalf("FromDomainConfig: %v", err)
	}

	enriched := cat.DeriveProducersConsumers()
	svc := enriched.Services[0]

	if got := svc.Events[1].Producers; len(got) != 1 || got[0] != "task-service" {
		t.Errorf("task.created producers = %v, want [task-service]", got)
	}

	if got := svc.Commands[0].Consumers; len(got) != 1 || got[0] != "task-service" {
		t.Errorf("task.create consumers = %v, want [task-service]", got)
	}
}

// ─── Curated merge survival ───

func TestFromDomainConfig_CuratorSurvives(t *testing.T) {
	cat, err := FromDomainConfig(context.Background(), fixtureDomain(),
		WithTitle("Task Service"),
		WithServiceOwners("lars"),
		WithCurator(func(b *catalog.Builder) {
			b.AddUser(catalog.User{ID: "lars", Name: "Lars"})
			b.AddDataStore(catalog.DataStore{ID: "tasks-db", Name: "Tasks DB"})
			b.AddFlow(catalog.Flow{Name: "Create Task", Steps: []catalog.FlowStep{{ID: "s1"}}})
		}),
	)
	if err != nil {
		t.Fatalf("FromDomainConfig: %v", err)
	}

	if len(cat.Users) != 1 || cat.Users[0].ID != "lars" {
		t.Errorf("curated user lost: %+v", cat.Users)
	}

	if len(cat.DataStores) != 1 || cat.DataStores[0].ID != "tasks-db" {
		t.Errorf("curated data store lost: %+v", cat.DataStores)
	}

	if len(cat.Flows) != 1 || cat.Flows[0].Name != "Create Task" {
		t.Errorf("curated flow lost: %+v", cat.Flows)
	}

	if len(cat.Services[0].Owners) != 1 || cat.Services[0].Owners[0] != "lars" {
		t.Errorf("service owners = %v, want [lars]", cat.Services[0].Owners)
	}
}

// ─── FromSystem / DeriveInto errors ───

func TestDeriveInto_NilArguments(t *testing.T) {
	if err := DeriveInto(catalog.NewBuilder("t", "1.0.0"), nil, system.DomainConfig{}); err == nil {
		t.Error("nil system must be a rejection, got nil")
	}

	if err := DeriveInto(nil, &system.System{}, system.DomainConfig{}); err == nil {
		t.Error("nil builder must be a rejection, got nil")
	}
}

func TestFromSystem_MatchesEphemeralBoot(t *testing.T) {
	domain := fixtureDomain()

	sys, err := system.New(context.Background(), domain, memoryDeployment())
	if err != nil {
		t.Fatalf("boot: %v", err)
	}

	defer func() { _ = sys.Close() }()

	fromSystem, err := FromSystem(sys, domain, WithTitle("Task Service"))
	if err != nil {
		t.Fatalf("FromSystem: %v", err)
	}

	ephemeral, err := FromDomainConfig(context.Background(), domain, WithTitle("Task Service"))
	if err != nil {
		t.Fatalf("FromDomainConfig: %v", err)
	}

	if len(fromSystem.Services[0].Commands) != len(ephemeral.Services[0].Commands) ||
		len(fromSystem.Services[0].Events) != len(ephemeral.Services[0].Events) {
		t.Errorf("FromSystem and FromDomainConfig disagree: %+v vs %+v",
			fromSystem.Services[0], ephemeral.Services[0])
	}
}

// ─── eventUniverse ───

func TestEventUniverse_UnionDedupSorted(t *testing.T) {
	domain := system.DomainConfig{
		Events: []event.Type{"b.first", "a.second"},
		Schema: []schema.EventSchema{
			schema.Event("a.second", 3),
			schema.Event("c.third", 1),
		},
	}

	got := eventUniverse(domain)
	want := []event.Type{"a.second", "b.first", "c.third"}

	if len(got) != len(want) {
		t.Fatalf("universe = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("universe = %v, want %v", got, want)
		}
	}
}
