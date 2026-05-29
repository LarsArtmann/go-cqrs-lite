package main

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/decider"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/query"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/middleware"
	"github.com/larsartmann/go-cqrs-lite/signing"
)

func TestFullStack_WithSigning(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	readModel := NewReadModelStore()

	hmacSecret := []byte("test-hmac-secret-key-exactly-32-b!")
	signer, err := signing.NewHMAC(hmacSecret)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	if err := bus.UsePublish(signing.SignMiddleware(signer)); err != nil {
		t.Fatalf("install sign middleware: %v", err)
	}

	if err := bus.Use(signing.VerifyMiddleware(signer)); err != nil {
		t.Fatalf("install verify middleware: %v", err)
	}

	userDecider := decider.Decider[UserState]{
		Initial: UserState{},
		Fold:    foldUser,
	}

	deciderRepo, err := decider.NewRepository(store, bus, userDecider)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	subscribeReadModel(bus, readModel)

	cmdDisp := command.NewDispatcher()
	cmdDisp.Use(
		middleware.CommandRecovery(),
		middleware.CommandRetry(middleware.DefaultRetryConfig()),
	)

	registerCommandHandlers(cmdDisp, deciderRepo)

	qryDisp := query.NewDispatcher()
	registerQueryHandlers(qryDisp, readModel)

	userID := id.NewAggregateID()

	err = cmdDisp.Dispatch(ctx, newUserCmd(userID, "signed@test.com", "Signed User"))
	if err != nil {
		t.Fatalf("create user with signing: %v", err)
	}

	rm, ok := readModel.Get(userID)
	if !ok {
		t.Fatal("expected user in read model after signed create")
	}

	if rm.Email != "signed@test.com" {
		t.Errorf("expected email signed@test.com, got %s", rm.Email)
	}

	result, err := query.DispatchTyped[ReadModel](ctx, qryDisp, &GetUserQuery{aggregateID: userID})
	if err != nil {
		t.Fatalf("query user: %v", err)
	}

	if result.Email != "signed@test.com" {
		t.Errorf("query result: expected email signed@test.com, got %s", result.Email)
	}

	users := readModel.List()
	if len(users) != 1 {
		t.Errorf("expected 1 user in list, got %d", len(users))
	}
}

func TestFullStack_DuplicateUserRejection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()

	userDecider := decider.Decider[UserState]{
		Initial: UserState{},
		Fold:    foldUser,
	}

	deciderRepo, err := decider.NewRepository(store, bus, userDecider)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	cmdDisp := command.NewDispatcher()
	registerCommandHandlers(cmdDisp, deciderRepo)

	userID := id.NewAggregateID()

	err = cmdDisp.Dispatch(ctx, newUserCmd(userID, "dup@test.com", "First"))
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	err = cmdDisp.Dispatch(ctx, newUserCmd(userID, "dup2@test.com", "Second"))
	if err == nil {
		t.Fatal("expected error for duplicate user creation")
	}

	var evtErr *event.Error
	if !errors.As(err, &evtErr) {
		t.Fatalf("expected *event.Error, got %T", err)
	}

	if evtErr.Family() != event.Conflict {
		t.Errorf("expected Conflict, got %s", evtErr.Family())
	}
}
