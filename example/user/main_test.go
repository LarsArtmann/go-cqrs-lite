package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
)

func subscribeReadModel(
	journal event.Journal,
	bus *eventtest.FakeBus,
	readModel *ReadModelStore,
) *projection.Runner {
	checkpointStore := memory.NewMemoryCheckpointStore()

	runner, err := projection.NewRunner(journal, bus, checkpointStore)
	if err != nil {
		log.Fatalf("create runner: %v", err)
	}

	if err := runner.Register(readModel); err != nil {
		log.Fatalf("register projection: %v", err)
	}

	go func() {
		if runErr := runner.Run(context.Background()); runErr != nil {
			log.Printf("runner stopped: %v", runErr)
		}
	}()

	time.Sleep(50 * time.Millisecond) // let runner subscribe before events flow

	return runner
}

func TestDecider_CreateUser(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	decide := decideCreateUser(aggID, "alice@example.com", "Alice")

	events, err := decide(UserState{}, event.Version(0))
	if err != nil {
		t.Fatalf("decide create user: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type() != eventUserCreated {
		t.Errorf("expected event type %s, got %s", eventUserCreated, events[0].Type())
	}

	var payload UserCreatedPayload
	if err := json.Unmarshal(events[0].Payload(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if payload.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", payload.Email)
	}

	if payload.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", payload.Name)
	}
}

func TestDecider_CreateUser_EmptyEmail(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	decide := decideCreateUser(aggID, "", "No Email")

	_, err := decide(UserState{}, event.Version(0))
	if err == nil {
		t.Fatal("expected error for empty email")
	}

	var evtErr *event.Error
	if !errors.As(err, &evtErr) {
		t.Fatalf("expected *event.Error, got %T: %v", err, err)
	}

	if evtErr.Family() != event.Rejection {
		t.Errorf("expected Rejection family, got %s", evtErr.Family())
	}
}

func TestDecider_CreateUser_AlreadyExists(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	decide := decideCreateUser(aggID, "dup@example.com", "Dup")

	_, err := decide(UserState{Email: "dup@example.com", Name: "Dup"}, event.Version(1))
	if err == nil {
		t.Fatal("expected error for duplicate user")
	}

	var evtErr *event.Error
	if !errors.As(err, &evtErr) {
		t.Fatalf("expected *event.Error, got %T: %v", err, err)
	}

	if evtErr.Family() != event.Conflict {
		t.Errorf("expected Conflict family, got %s", evtErr.Family())
	}
}

func TestDecider_ChangeName(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	decide := decideChangeName(aggID, "New Name")

	evts, err := decide(UserState{Email: "test@test.com", Name: "Old"}, event.Version(1))
	if err != nil {
		t.Fatalf("decide change name: %v", err)
	}

	if len(evts) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evts))
	}

	var payload UserNameChangedPayload
	if err := json.Unmarshal(evts[0].Payload(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if payload.Name != "New Name" {
		t.Errorf("expected name New Name, got %s", payload.Name)
	}
}

func TestDecider_ChangeName_UserNotFound(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	decide := decideChangeName(aggID, "New")

	_, err := decide(UserState{}, event.Version(0))
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}

	var evtErr *event.Error
	if !errors.As(err, &evtErr) {
		t.Fatalf("expected *event.Error, got %T", err)
	}

	if evtErr.Family() != event.Rejection {
		t.Errorf("expected Rejection, got %s", evtErr.Family())
	}
}

func TestFoldUser(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	createdEvt := newUserCreatedEvent(t, aggID, "a@b.com", "A")

	state, err := foldUser(UserState{}, createdEvt)
	if err != nil {
		t.Fatalf("fold UserCreated: %v", err)
	}

	if state.Email != "a@b.com" {
		t.Errorf("expected email a@b.com, got %s", state.Email)
	}

	if state.Name != "A" {
		t.Errorf("expected name A, got %s", state.Name)
	}

	changedEvt, err := event.NewEvent(
		eventUserNameChanged, aggID, aggregateType, event.Version(2),
		mustMarshal(UserNameChangedPayload{Name: "B"}),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	state, err = foldUser(state, changedEvt)
	if err != nil {
		t.Fatalf("fold UserNameChanged: %v", err)
	}

	if state.Name != "B" {
		t.Errorf("expected name B, got %s", state.Name)
	}

	if state.Email != "a@b.com" {
		t.Errorf("email should be unchanged, got %s", state.Email)
	}
}

func TestReadModel_Projection(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	store := NewReadModelStore()

	createdEvt := newUserCreatedEvent(t, aggID, "x@y.com", "X")

	err := store.Handle(context.Background(), createdEvt)
	if err != nil {
		t.Fatalf("handle UserCreated: %v", err)
	}

	readModelResult, ok := store.Get(aggID)
	if !ok {
		t.Fatal("expected user in read model")
	}

	if readModelResult.Email != "x@y.com" {
		t.Errorf("expected email x@y.com, got %s", readModelResult.Email)
	}

	changedEvt, err := event.NewEvent(
		eventUserNameChanged, aggID, aggregateType, event.Version(2),
		mustMarshal(UserNameChangedPayload{Name: "Y"}),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	err = store.Handle(context.Background(), changedEvt)
	if err != nil {
		t.Fatalf("handle UserNameChanged: %v", err)
	}

	readModelResult, _ = store.Get(aggID)
	if readModelResult.Name != "Y" {
		t.Errorf("expected name Y, got %s", readModelResult.Name)
	}
}

func TestFullCQRS_Lifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := eventtest.NewFakeBus()
	readModel := NewReadModelStore()

	userDecider := decider.Decider[UserState]{
		Initial: UserState{},
		Fold:    foldUser,
	}

	deciderRepo, err := decider.NewRepository(store, bus, userDecider)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	runner := subscribeReadModel(store, bus, readModel)
	defer func() { _ = runner.Close() }()

	userID := id.NewAggregateID()

	err = deciderRepo.Execute(
		ctx, userID, "User",
		decideCreateUser(userID, "test@example.com", "Test User"),
	)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	readModelResult, ok := readModel.Get(userID)
	if !ok {
		t.Fatal("expected user in read model after create")
	}

	if readModelResult.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", readModelResult.Email)
	}

	err = deciderRepo.Execute(
		ctx, userID, "User",
		decideChangeName(userID, "Updated User"),
	)
	if err != nil {
		t.Fatalf("change name: %v", err)
	}

	readModelResult, _ = readModel.Get(userID)
	if readModelResult.Name != "Updated User" {
		t.Errorf("expected name Updated User, got %s", readModelResult.Name)
	}

	users := readModel.List()
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

func TestQueryDispatcher(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	readModel := NewReadModelStore()
	aggID := id.NewAggregateID()

	createdEvt := newUserCreatedEvent(t, aggID, "q@test.com", "Q")

	_ = readModel.Handle(t.Context(), createdEvt)

	qryDisp := query.NewDispatcher()
	registerQueryHandlers(qryDisp, readModel)

	result, err := qryDisp.Dispatch(ctx, &GetUserQuery{aggregateID: aggID})
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	readModelResult, ok := result.(ReadModel)
	if !ok {
		t.Fatalf("expected ReadModel, got %T", result)
	}

	if readModelResult.Email != "q@test.com" {
		t.Errorf("expected email q@test.com, got %s", readModelResult.Email)
	}

	listResult, err := qryDisp.Dispatch(ctx, &ListUsersQuery{})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}

	list, ok := listResult.([]ReadModel)
	if !ok {
		t.Fatalf("expected []ReadModel, got %T", listResult)
	}

	if len(list) != 1 {
		t.Errorf("expected 1 user, got %d", len(list))
	}
}

func TestEventCatalog_Generation(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()

	err := generateEventCatalog(outputDir)
	if err != nil {
		t.Fatalf("generate event catalog: %v", err)
	}

	svcDir := filepath.Join(outputDir, "services", "user-svc")
	indexFile := filepath.Join(svcDir, "index.mdx")

	info, err := os.Stat(indexFile)
	if err != nil {
		t.Fatalf("stat service index: %v", err)
	}

	if info.Size() == 0 {
		t.Error("service index file is empty")
	}

	data, err := os.ReadFile(indexFile)
	if err != nil {
		t.Fatalf("read service index: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "UserCreated") {
		t.Error("expected 'UserCreated' in service index")
	}

	if !strings.Contains(content, "UserNameChanged") {
		t.Error("expected 'UserNameChanged' in service index")
	}

	createdFile := filepath.Join(svcDir, "events", "UserCreated", "index.mdx")
	createdData, err := os.ReadFile(createdFile)
	if err != nil {
		t.Fatalf("read UserCreated event: %v", err)
	}

	if !strings.Contains(string(createdData), "User Created") {
		t.Error("expected 'User Created' in UserCreated event index")
	}
}

func TestErrorClassification(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	decide := decideCreateUser(aggID, "", "No Email")

	_, err := decide(UserState{}, event.Version(0))
	if err == nil {
		t.Fatal("expected error for empty email")
	}

	family := event.Classify(err)
	if family != event.Rejection {
		t.Errorf("expected Rejection, got %s", family)
	}

	if event.IsRetryable(err) {
		t.Error("empty email rejection should not be retryable")
	}
}

func newUserCreatedEvent(
	t *testing.T,
	aggID id.AggregateID,
	email, name string,
) *event.ImmutableEvent {
	t.Helper()
	evt, err := event.NewEvent(
		eventUserCreated, aggID, aggregateType, event.Version(1),
		mustMarshal(UserCreatedPayload{Email: email, Name: name}),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	return evt
}
