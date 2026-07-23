package command_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func labelMiddleware(callOrder *[]string, label string) func(next command.Handler) command.Handler {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			*callOrder = append(*callOrder, label)

			return next(ctx, cmd)
		}
	}
}

var _ = Describe("Command Dispatcher", func() {
	var (
		ctx        context.Context
		dispatcher *command.Dispatcher
		aggID      id.StreamID
	)

	BeforeEach(func() {
		ctx = context.Background()
		dispatcher = command.NewDispatcher()
		aggID = id.NewStreamID()
	})

	Describe("As a developer dispatching commands", func() {
		Context("when I register a handler and dispatch the matching command", func() {
			It("should deliver my command to the handler I registered for this type", func() {
				var received command.Command
				Expect(
					dispatcher.Register(
						"CreateUser",
						func(_ context.Context, cmd command.Command) error {
							received = cmd

							return nil
						},
					),
				).To(Succeed())

				cmd, err := command.New("CreateUser", aggID)
				Expect(err).ToNot(HaveOccurred())

				Expect(dispatcher.Dispatch(ctx, cmd)).To(Succeed())
				Expect(received).ToNot(BeNil())
				Expect(received.Type()).To(Equal(command.Type("CreateUser")))
				Expect(received.StreamID()).To(Equal(aggID))
			})
		})

		Context("when I dispatch a command with no registered handler", func() {
			It(
				"should reject my command and explain that no handler was registered for this type",
				func() {
					cmd, err := command.New("UnknownCommand", aggID)
					Expect(err).ToNot(HaveOccurred())

					err = dispatcher.Dispatch(ctx, cmd)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("handler not found"))
				},
			)
		})

		Context("when I register multiple handlers for different types", func() {
			It("should dispatch each type to its own handler independently", func() {
				var createUserCalled, deleteUserCalled bool

				Expect(
					dispatcher.Register(
						"CreateUser",
						func(_ context.Context, _ command.Command) error {
							createUserCalled = true

							return nil
						},
					),
				).To(Succeed())

				Expect(
					dispatcher.Register(
						"DeleteUser",
						func(_ context.Context, _ command.Command) error {
							deleteUserCalled = true

							return nil
						},
					),
				).To(Succeed())

				createCmd, err := command.New("CreateUser", aggID)
				Expect(err).ToNot(HaveOccurred())
				Expect(dispatcher.Dispatch(ctx, createCmd)).To(Succeed())

				deleteCmd, err := command.New("DeleteUser", aggID)
				Expect(err).ToNot(HaveOccurred())
				Expect(dispatcher.Dispatch(ctx, deleteCmd)).To(Succeed())

				Expect(createUserCalled).To(BeTrue())
				Expect(deleteUserCalled).To(BeTrue())
			})
		})

		Context("when my handler returns an error", func() {
			It("should surface the handler's error so I can decide what to do", func() {
				expectedErr := errors.New("something went wrong")
				Expect(
					dispatcher.Register(
						"FailCommand",
						func(_ context.Context, _ command.Command) error {
							return expectedErr
						},
					),
				).To(Succeed())

				cmd, err := command.New("FailCommand", aggID)
				Expect(err).ToNot(HaveOccurred())

				err = dispatcher.Dispatch(ctx, cmd)
				Expect(err).To(MatchError(expectedErr))
			})
		})
	})

	Describe("As a developer using middleware", func() {
		Context("when I apply multiple middleware to the dispatcher", func() {
			It(
				"should execute outer middleware before inner, so I can layer cross-cutting concerns",
				func() {
					var callOrder []string

					dispatcher.Use(labelMiddleware(&callOrder, "mw1"))
					dispatcher.Use(labelMiddleware(&callOrder, "mw2"))

					Expect(
						dispatcher.Register(
							"TestCommand",
							appendCommandHandler(&callOrder),
						),
					).To(Succeed())

					cmd, err := command.New("TestCommand", aggID)
					Expect(err).ToNot(HaveOccurred())

					Expect(dispatcher.Dispatch(ctx, cmd)).To(Succeed())
					Expect(callOrder).To(Equal([]string{"mw1", "mw2", "handler"}))
				},
			)
		})

		Context("when middleware short-circuits the chain", func() {
			It("should block the handler call entirely, so no business logic runs", func() {
				var handlerCalled bool

				dispatcher.Use(func(_ command.Handler) command.Handler {
					return func(_ context.Context, _ command.Command) error {
						return errors.New("blocked by middleware")
					}
				})

				Expect(
					dispatcher.Register(
						"BlockedCommand",
						func(_ context.Context, _ command.Command) error {
							handlerCalled = true

							return nil
						},
					),
				).To(Succeed())

				cmd, err := command.New("BlockedCommand", aggID)
				Expect(err).ToNot(HaveOccurred())

				err = dispatcher.Dispatch(ctx, cmd)
				Expect(err).To(HaveOccurred())
				Expect(handlerCalled).To(BeFalse())
			})
		})
	})

	Describe("As a developer managing the dispatcher lifecycle", func() {
		Context("when I close the dispatcher", func() {
			It(
				"should reject further dispatch and registration, explaining the dispatcher is closed",
				func() {
					Expect(dispatcher.Close()).To(Succeed())

					cmd, err := command.New("AnyCommand", aggID)
					Expect(err).ToNot(HaveOccurred())

					err = dispatcher.Dispatch(ctx, cmd)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("dispatcher is closed"))

					err = dispatcher.Register(
						"NewCommand",
						func(_ context.Context, _ command.Command) error {
							return nil
						},
					)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("dispatcher is closed"))
				},
			)
		})
	})

	Describe("As a developer creating commands", func() {
		Context("when I create a command with valid inputs", func() {
			It("should populate all fields correctly", func() {
				corrID := id.NewCorrelationID()
				causeID := id.NewCausationID()
				userID := id.NewUserID()
				reqID := id.NewRequestID()

				cmd, err := command.New(
					"CreateOrder", aggID,
					command.WithCorrelationID(corrID),
					command.WithCausationID(causeID),
					command.WithUserID(userID),
					command.WithRequestID(reqID),
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(cmd.Type()).To(Equal(command.Type("CreateOrder")))
				Expect(cmd.StreamID()).To(Equal(aggID))
				Expect(cmd.Metadata().CorrelationID).To(Equal(corrID))
				Expect(cmd.Metadata().CausationID).To(Equal(causeID))
				Expect(cmd.Metadata().UserID).To(Equal(userID))
				Expect(cmd.Metadata().RequestID).To(Equal(reqID))
			})
		})

		Context("when I create a command with an empty type", func() {
			It("should reject my input and explain that the command type is required", func() {
				_, err := command.New("", aggID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("command type is required"))
			})
		})

		Context("when I create a command with a zero aggregate ID", func() {
			It("should reject my input and explain that the aggregate ID is required", func() {
				_, err := command.New("CreateUser", id.StreamID{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("aggregate ID is required"))
			})
		})

		Context("when I create a command with invalid inputs", func() {
			It("should reject with a descriptive error", func() {
				_, err := command.New("", aggID)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("As a developer using typed handlers", func() {
		type createUserCmd struct {
			command.BasicCommand

			Name string
		}

		Context("when I register a typed handler", func() {
			It("should give me the fully typed command struct without manual casting", func() {
				var receivedName string

				err := command.RegisterTyped[*createUserCmd](
					dispatcher, "CreateUser",
					func(_ context.Context, cmd *createUserCmd) error {
						receivedName = cmd.Name

						return nil
					},
				)
				Expect(err).ToNot(HaveOccurred())

				basicCmd, err := command.New("CreateUser", aggID)
				Expect(err).ToNot(HaveOccurred())
				cmd := &createUserCmd{
					BasicCommand: *basicCmd,
					Name:         "Alice",
				}

				Expect(dispatcher.Dispatch(ctx, cmd)).To(Succeed())
				Expect(receivedName).To(Equal("Alice"))
			})
		})
	})
})
