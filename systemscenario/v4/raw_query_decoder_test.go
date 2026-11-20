package systemscenario_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// The RawQuery + Count mixing domain: the raw query consumes two events
// the Count projection does not — the exact shape that built a PARTIAL
// auto event decoder before raw decls contributed decoder entries. The
// test drives events through the FULL journal path (command dispatch →
// journal → bus → projection host → adapter decoder → folds), because a
// direct store.Apply bypasses the decoder entirely and cannot observe
// the bug (the host adapter is the only component that errors on a
// missing decoder entry).
type rqPetAdopted struct {
	ID        string
	Name      string
	AdoptedAt string
}

type rqPetRenamed struct {
	ID   string
	Name string
}

type rqPetView struct {
	ID   string
	Name string
}

type rqAdoptCmd struct {
	*command.BasicCommand

	Name string
}

type rqRenameCmd struct {
	*command.BasicCommand

	Name string
}

type rqGetQuery struct {
	*query.BasicQuery

	ID string
}

type rqState struct {
	Exists bool
	Name   string
}

var (
	rqEventAdopted = event.Type("rq.pet.adopted")
	rqEventRenamed = event.Type("rq.pet.renamed")
	rqCmdAdopt     = command.Type("rq.pet.adopt")
	rqCmdRename    = command.Type("rq.pet.rename")
	rqQueryGet     = query.Type("rq.pet.get")
)

func rqApply(s rqState, evt event.Event) (rqState, error) {
	switch evt.Type() {
	case rqEventAdopted:
		p, err := event.DecodePayloadAuto[rqPetAdopted](evt)
		if err != nil {
			return s, err
		}

		s.Exists = true
		s.Name = p.Name
	case rqEventRenamed:
		p, err := event.DecodePayloadAuto[rqPetRenamed](evt)
		if err != nil {
			return s, err
		}

		s.Name = p.Name
	}

	return s, nil
}

func rqDomain() system.DomainConfig {
	return system.DomainConfig{
		Events: []event.Type{rqEventAdopted, rqEventRenamed},
		Projections: []system.ProjectionDeclaration{
			// Declared mechanical folds over wire names; both events are
			// consumed ONLY by this raw declaration.
			system.RawQuery(metaengine.Query[system.ScanInput, rqPetView]("rq_pets",
				metaengine.AutoInsertTyped[rqPetAdopted, rqPetView](string(rqEventAdopted), "ID"),
				metaengine.AutoUpdateTyped[rqPetRenamed, rqPetView](string(rqEventRenamed), "ID"),
			)),
			// A ProjectionSpec consuming a SUBSET of the raw query's events:
			// before the fix, its decoder entry alone made the auto decoder
			// partial, so rq.pet.renamed hit "no decoder registered" in the
			// adapter and crash-looped the projection worker.
			system.Count("rq-pet-counts").
				On(string(rqEventAdopted), rqPetAdopted{}, +1, "adopted").
				Done(),
		},
		Commands: func(sys *system.System) {
			if err := system.RegisterDecider(sys, "RqPet", decider.Decider[rqState]{
				Initial: rqState{},
				Apply:   rqApply,
			}); err != nil {
				panic(err)
			}

			adopt := func(ctx context.Context, cmd rqAdoptCmd) system.Op[rqState] {
				return system.Execute(ctx, cmd.StreamID(), "RqPet",
					func(s rqState, v event.Version) ([]event.Event, error) {
						if s.Exists {
							return nil, errors.New("already exists")
						}

						evt, err := event.New(rqEventAdopted, cmd.StreamID(), "RqPet", v.Increment(),
							rqPetAdopted{ID: cmd.StreamID().Get(), Name: cmd.Name, AdoptedAt: "2026-01-15"})
						if err != nil {
							return nil, err
						}

						return []event.Event{evt}, nil
					})
			}

			rename := func(ctx context.Context, cmd rqRenameCmd) system.Op[rqState] {
				return system.Execute(ctx, cmd.StreamID(), "RqPet",
					func(s rqState, v event.Version) ([]event.Event, error) {
						if !s.Exists {
							return nil, errors.New("not found")
						}

						evt, err := event.New(rqEventRenamed, cmd.StreamID(), "RqPet", v.Increment(),
							rqPetRenamed{ID: cmd.StreamID().Get(), Name: cmd.Name})
						if err != nil {
							return nil, err
						}

						return []event.Event{evt}, nil
					})
			}

			if err := system.RegisterCommand[rqAdoptCmd, rqState](sys, rqCmdAdopt, adopt); err != nil {
				panic(err)
			}

			if err := system.RegisterCommand[rqRenameCmd, rqState](sys, rqCmdRename, rename); err != nil {
				panic(err)
			}
		},
		Queries: func(sys *system.System) {
			pets := metaengine.NewReader[rqPetView](sys.MetaEngine(), "rq_pets")

			if err := system.RegisterQuery[rqGetQuery, rqPetView](sys, string(rqQueryGet),
				func(ctx context.Context, q rqGetQuery) (rqPetView, error) {
					v, found, err := pets.Get(ctx, q.ID)
					if err != nil {
						return rqPetView{}, err
					}

					if !found {
						return rqPetView{}, errors.New("rq pet not found")
					}

					return v, nil
				}); err != nil {
				panic(err)
			}
		},
	}
}

func TestRawQueryMixedWithProjectionSpecDecodesAllEvents(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sc := systemscenario.System(t, ctx, rqDomain(), systemscenario.Memory())

	aggID := id.NewStreamID()

	adoptBasic, err := command.New(rqCmdAdopt, aggID)
	if err != nil {
		t.Fatalf("command.New(adopt): %v", err)
	}

	renameBasic, err := command.New(rqCmdRename, aggID)
	if err != nil {
		t.Fatalf("command.New(rename): %v", err)
	}

	getBasic, err := query.New(rqQueryGet)
	if err != nil {
		t.Fatalf("query.New: %v", err)
	}

	view := func() (any, error) {
		return system.DispatchQuery[rqGetQuery, rqPetView](ctx, sc.System(),
			rqGetQuery{BasicQuery: getBasic, ID: aggID.Get()})
	}

	sc.Given().
		Command(rqAdoptCmd{BasicCommand: adoptBasic, Name: "Rex"}).
		When(rqRenameCmd{BasicCommand: renameBasic, Name: "Rexy"}).
		ThenQuery(view, rqPetView{ID: aggID.Get(), Name: "Rexy"})

	counts, err := system.GetCount(ctx, sc.System(), "rq-pet-counts")
	if err != nil {
		t.Fatalf("count: %v", err)
	}

	if counts["adopted"] != 1 {
		t.Fatalf("counts = %+v, want adopted=1", counts)
	}
}
