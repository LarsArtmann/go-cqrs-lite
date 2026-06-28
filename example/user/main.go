// Package main demonstrates a complete CQRS + Event Sourcing integration
// using go-cqrs-lite with the Decider pattern — pure functions for state
// reconstruction and command decisions, no mutable aggregate required.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/middleware/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	"github.com/larsartmann/go-cqrs-lite/signing/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== go-cqrs-lite: Full CQRS + Event Sourcing Demo ===")
	fmt.Println()

	otelProvider, otelBundle := setupOTel(ctx)
	defer otelProvider.Shutdown(ctx)

	_, bus, readModel, deciderRepo := setupInfrastructure(otelBundle)

	signer := setupSigning(bus)

	var publishedEvents []event.Event

	trackPublishedEvents(bus, &publishedEvents)

	cleanup := registerProjection(bus, readModel)
	defer cleanup()

	cmdDisp, qryDisp := setupDispatchers(deciderRepo, readModel, otelBundle)

	userID := runDemoSteps(ctx, cmdDisp, qryDisp, &publishedEvents)
	runTombstoneRebirthDemo(ctx, cmdDisp, deciderRepo, userID)
	runErrorDemo(ctx, cmdDisp)
	runEventCatalog()

	fmt.Println("=== Demo Complete ===")
	fmt.Printf("  Events published: %d\n", len(publishedEvents))
	fmt.Printf("  Read model users: %d\n", len(readModel.List()))
	fmt.Printf("  User ID: %s\n", userID)
	fmt.Printf("  Signing: HMAC-SHA256 (signer=%T)\n", signer)
}

func setupInfrastructure(otelBundle *middleware.OTelBundle) (
	*memory.MemoryStore,
	event.Bus,
	*ReadModelStore,
	*decider.Repository[UserState],
) {
	store := memory.NewMemoryStore()
	bus := cqrswatermill.NewEventBus()
	readModel := NewReadModelStore()

	if err := bus.Use(otelBundle.Event()...); err != nil {
		log.Fatalf("install event tracing: %v", err)
	}

	if err := bus.UsePublish(otelBundle.Publish()...); err != nil {
		log.Fatalf("install publish tracing: %v", err)
	}

	userDecider := decider.Decider[UserState]{
		Initial: UserState{},
		Apply:   applyUser,
	}

	deciderRepo, err := decider.NewRepository(store, bus, userDecider)
	if err != nil {
		log.Fatalf("create decider repo: %v", err)
	}

	fmt.Println("[infra] Store: MemoryStore | Bus: MemoryBus | Decider Repository: ready")
	fmt.Println("[infra] Read model: bus.SubscribeAll → ReadModelStore.Handle")
	fmt.Println("[infra] OTel: tracing + metrics wired for all message kinds")
	fmt.Println()

	return store, bus, readModel, deciderRepo
}

func setupDispatchers(
	deciderRepo *decider.Repository[UserState],
	readModel *ReadModelStore,
	otelBundle *middleware.OTelBundle,
) (*command.Dispatcher, *query.Dispatcher) {
	cmdDisp := command.NewDispatcher()

	cmdDisp.Use(
		middleware.CommandRecovery(),
		middleware.CommandLogging(newLogger()),
	)
	cmdDisp.Use(otelBundle.Command()...)
	cmdDisp.Use(
		middleware.CommandRetry(middleware.DefaultRetryConfig()),
	)

	registerCommandHandlers(cmdDisp, deciderRepo)

	qryDisp := query.NewDispatcher()
	qryDisp.Use(otelBundle.Query()...)
	registerQueryHandlers(qryDisp, readModel)

	fmt.Println("[infra] Middleware: Recovery → Logging → Tracing+Metrics → Retry")
	fmt.Println()

	return cmdDisp, qryDisp
}

func setupSigning(bus event.Bus) signing.Signer {
	hmacSecret := []byte("demo-hmac-secret-key-exactly-32-b!")

	signer, err := signing.NewHMAC(hmacSecret)
	if err != nil {
		log.Fatalf("create HMAC signer: %v", err)
	}

	err = bus.UsePublish(signing.SignMiddleware(signer))
	if err != nil {
		log.Fatalf("install sign middleware: %v", err)
	}

	err = bus.Use(signing.VerifyMiddleware(signer))
	if err != nil {
		log.Fatalf("install verify middleware: %v", err)
	}

	fmt.Println("[infra] Signing: HMAC-SHA256 (sign on publish, verify on handle)")
	fmt.Println()

	return signer
}

func runDemoSteps(
	ctx context.Context,
	cmdDisp *command.Dispatcher,
	qryDisp *query.Dispatcher,
	publishedEvents *[]event.Event,
) id.AggregateID {
	fmt.Println("--- Step 1: Create User ---")

	userID := id.NewAggregateID()

	err := cmdDisp.Dispatch(ctx, newUserCmd(userID, "alice@example.com", "Alice Smith"))
	if err != nil {
		log.Fatalf("create user: %v", err)
	}

	fmt.Printf("→ Created user %s (alice@example.com)\n\n", userID)

	fmt.Println("--- Step 2: Change Name ---")

	err = cmdDisp.Dispatch(ctx, &ChangeUserNameCmd{
		commandID:   id.NewCommandID(),
		aggregateID: userID,
		name:        "Alice Johnson",
	})
	if err != nil {
		log.Fatalf("change name: %v", err)
	}

	fmt.Printf("→ Changed name to %q, version %d\n\n", "Alice Johnson", len(*publishedEvents))

	fmt.Println("--- Step 3: Query User ---")

	rm, err := query.DispatchTyped[ReadModel](ctx, qryDisp, &GetUserQuery{aggregateID: userID})
	if err != nil {
		log.Fatalf("get user: %v", err)
	}

	fmt.Printf("→ User{Email: %q, Name: %q}\n\n", rm.Email, rm.Name)

	return userID
}

func runTombstoneRebirthDemo(
	ctx context.Context,
	cmdDisp *command.Dispatcher,
	deciderRepo *decider.Repository[UserState],
	userID id.AggregateID,
) {
	fmt.Println("--- Step 4: Tombstone + Rebirth ---")

	err := cmdDisp.Dispatch(ctx, &DeleteUserCmd{
		commandID:   id.NewCommandID(),
		aggregateID: userID,
		reason:      "GDPR request",
	})
	if err != nil {
		log.Fatalf("delete user: %v", err)
	}

	fmt.Printf("→ Deleted user %s (tombstoned)\n", userID)

	state, _, err := deciderRepo.Load(ctx, userID, aggregateType)
	if err != nil {
		log.Fatalf("load state after delete: %v", err)
	}

	fmt.Printf("→ State after delete: Deleted=%v, Reason=%q\n", state.Deleted, state.DeleteReason)

	err = cmdDisp.Dispatch(ctx, &RebirthUserCmd{
		commandID:   id.NewCommandID(),
		aggregateID: userID,
		email:       "alice.v2@example.com",
		name:        "Alice Reborn",
	})
	if err != nil {
		log.Fatalf("rebirth user: %v", err)
	}

	fmt.Printf("→ Reborn user %s as alice.v2@example.com\n", userID)

	state, _, err = deciderRepo.Load(ctx, userID, aggregateType)
	if err != nil {
		log.Fatalf("load state after rebirth: %v", err)
	}

	fmt.Printf(
		"→ State after rebirth: Email=%q, Name=%q, Deleted=%v\n\n",
		state.Email,
		state.Name,
		state.Deleted,
	)
}

func runErrorDemo(ctx context.Context, cmdDisp *command.Dispatcher) {
	fmt.Println("--- Step 5: Validation Error (empty email) ---")

	err := cmdDisp.Dispatch(ctx, newUserCmd(id.NewAggregateID(), "", "No Email"))
	if err != nil {
		evtErr, ok := errors.AsType[*event.Error](err)
		if ok {
			fmt.Printf("→ Rejected [%s]: %s\n\n", evtErr.Family(), evtErr.Message())
		} else {
			fmt.Printf("→ Error: %v\n\n", err)
		}
	}

	fmt.Println("--- Step 6: Error Classification ---")

	family := event.Classify(err)
	fmt.Printf("→ classify(create-no-email) = %s\n", family)
	fmt.Printf("→ isRetryable(create-no-email) = %v\n\n", event.IsRetryable(err))
}

func runEventCatalog() {
	fmt.Println("--- Step 7: EventCatalog ---")

	outputDir := filepath.Join(os.TempDir(), "eventcatalog-output")

	if err := generateEventCatalog(outputDir); err != nil {
		log.Fatalf("generate event catalog: %v", err)
	}

	fmt.Printf("→ EventCatalog written to %s\n\n", outputDir)
}

func setupOTel(ctx context.Context) (*cqrsotel.Provider, *middleware.OTelBundle) {
	provider, err := cqrsotel.Setup(
		cqrsotel.WithService("user-demo", "1.0.0", "local"),
		cqrsotel.WithStdoutExporter(os.Stdout),
	)
	if err != nil {
		log.Fatalf("setup otel: %v", err)
	}

	bundle, err := middleware.NewOTelBundle(
		cqrsotel.NewTracer("user-demo"),
		cqrsotel.NewMeter("user-demo"),
	)
	if err != nil {
		log.Fatalf("create otel bundle: %v", err)
	}

	fmt.Println("[infra] OTel: TracerProvider + MeterProvider + stdout span exporter configured")

	_ = ctx

	return provider, bundle
}
